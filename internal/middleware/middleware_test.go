package middleware

import (
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"github.com/tianlu1990s/gim/pkg/jwt"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// newTestJWTManager 创建用于测试的 JWTManager（内存生成 RSA 密钥对）。
func newTestJWTManager(t *testing.T) *jwt.JWTManager {
	t.Helper()
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}
	return jwt.NewJWTManagerFromKeys(privKey, &privKey.PublicKey, 15*time.Minute, 168*time.Hour)
}

func TestJWTAuthMissingHeader(t *testing.T) {
	jwtMgr := newTestJWTManager(t)

	r := gin.New()
	r.Use(JWTAuth(jwtMgr, nil))
	r.GET("/test", func(c *gin.Context) { c.Status(200) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("should return 200 OK (body.code carries error), got %d", w.Code)
	}
}

func TestJWTAuthInvalidToken(t *testing.T) {
	jwtMgr := newTestJWTManager(t)

	r := gin.New()
	r.Use(JWTAuth(jwtMgr, nil))
	r.GET("/test", func(c *gin.Context) { c.Status(200) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.here")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("should return 200, got %d", w.Code)
	}
}

func TestJWTAuthNoBearerPrefix(t *testing.T) {
	jwtMgr := newTestJWTManager(t)

	r := gin.New()
	r.Use(JWTAuth(jwtMgr, nil))
	r.GET("/test", func(c *gin.Context) { c.Status(200) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "token123")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("should return 200, got %d", w.Code)
	}
}

func TestJWTAuthValidToken(t *testing.T) {
	jwtMgr := newTestJWTManager(t)

	// 生成有效 token
	token, _, err := jwtMgr.GenerateAccessToken("testuser", "web")
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	r := gin.New()
	r.Use(JWTAuth(jwtMgr, nil)) // rdb=nil 跳过黑名单检查
	r.GET("/test", func(c *gin.Context) {
		userID, _ := c.Get("userID")
		platform, _ := c.Get("platform")
		c.JSON(200, gin.H{"userId": userID, "platform": platform})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	if !contains(w.Body.String(), "testuser") {
		t.Errorf("response should contain testuser: %s", w.Body.String())
	}
}

func TestCORS(t *testing.T) {
	r := gin.New()
	r.Use(CORS())
	r.GET("/test", func(c *gin.Context) { c.Status(200) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("OPTIONS", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("OPTIONS status = %d, want %d", w.Code, http.StatusNoContent)
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("missing CORS header")
	}
}

func TestCORSMaxAgeHeader(t *testing.T) {
	r := gin.New()
	r.Use(CORS())
	r.GET("/test", func(c *gin.Context) { c.Status(200) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("OPTIONS", "/test", nil)
	r.ServeHTTP(w, req)

	if maxAge := w.Header().Get("Access-Control-Max-Age"); maxAge != "86400" {
		t.Errorf("Max-Age = %s, want 86400", maxAge)
	}
}

func TestRecovery(t *testing.T) {
	r := gin.New()
	r.Use(Recovery())
	r.GET("/panic", func(c *gin.Context) {
		panic("test panic")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/panic", nil)
	r.ServeHTTP(w, req)

	// Recovery 应返回 500 而非让请求崩溃
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestRecoveryNoPanic(t *testing.T) {
	r := gin.New()
	r.Use(Recovery())
	r.GET("/ok", func(c *gin.Context) { c.Status(200) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/ok", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestRequestLogger(t *testing.T) {
	r := gin.New()
	r.Use(RequestLogger())
	r.GET("/test", func(c *gin.Context) { c.Status(200) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test?key=value", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestRequestLoggerErrorStatus(t *testing.T) {
	r := gin.New()
	r.Use(RequestLogger())
	r.GET("/error", func(c *gin.Context) { c.Status(http.StatusInternalServerError) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/error", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", w.Code)
	}
}

func TestRequestLoggerNotFound(t *testing.T) {
	r := gin.New()
	r.Use(RequestLogger())
	r.GET("/notfound", func(c *gin.Context) { c.Status(http.StatusNotFound) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/notfound", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

// --- RateLimit tests ---

func TestRateLimitCreate(t *testing.T) {
	handler := RateLimit(nil, 10, time.Minute)
	if handler == nil {
		t.Error("RateLimit returned nil handler")
	}
}

func TestRateLimitPassThrough(t *testing.T) {
	// When Redis is unreachable, the middleware should pass through (degrade gracefully).
	rdb := redis.NewClient(&redis.Options{
		Addr:         "127.0.0.1:1",
		DialTimeout:  10 * time.Millisecond,
		MaxRetries:   -1,
		PoolSize:     1,
		MinIdleConns: 0,
	})
	defer rdb.Close()

	r := gin.New()
	r.Use(RateLimit(rdb, 10, time.Minute))
	r.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200 (pass through when Redis is down)", w.Code)
	}
}

// --- JWT Auth enhancements ---

func TestJWTAuthValidTokenInjectsContext(t *testing.T) {
	jwtMgr := newTestJWTManager(t)

	token, _, err := jwtMgr.GenerateAccessToken("testuser2", "android")
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	r := gin.New()
	r.Use(JWTAuth(jwtMgr, nil))
	r.GET("/test", func(c *gin.Context) {
		userID, exists := c.Get("userID")
		if !exists {
			t.Error("userID not injected into context")
		}
		if userID != "testuser2" {
			t.Errorf("userID = %v, want testuser2", userID)
		}
		platform, exists := c.Get("platform")
		if !exists {
			t.Error("platform not injected into context")
		}
		if platform != "android" {
			t.Errorf("platform = %v, want android", platform)
		}
		c.Status(200)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

// --- helpers ---

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
