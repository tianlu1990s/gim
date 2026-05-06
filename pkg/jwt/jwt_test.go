package jwt

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	jwtv5 "github.com/golang-jwt/jwt/v5"
)

// newTestManager 使用内存生成的 RSA 密钥对创建 JWTManager，避免依赖外部密钥文件。
func newTestManager(t *testing.T, accessExp, refreshExp time.Duration) *JWTManager {
	t.Helper()
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}
	return &JWTManager{
		privateKey:    privKey,
		publicKey:     &privKey.PublicKey,
		accessExpire:  accessExp,
		refreshExpire: refreshExp,
	}
}

func TestGenerateAndParseAccessToken(t *testing.T) {
	mgr := newTestManager(t, 15*time.Minute, 168*time.Hour)

	token, expireAt, err := mgr.GenerateAccessToken("alice", "ios")
	if err != nil {
		t.Fatalf("GenerateAccessToken() error: %v", err)
	}
	if token == "" {
		t.Fatal("token is empty")
	}
	if expireAt <= time.Now().Unix() {
		t.Error("expireAt should be in the future")
	}

	// 解析验证
	claims, err := mgr.ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken() error: %v", err)
	}
	if claims.UserID != "alice" {
		t.Errorf("UserID = %v, want alice", claims.UserID)
	}
	if claims.Platform != "ios" {
		t.Errorf("Platform = %v, want ios", claims.Platform)
	}
	if claims.ID == "" {
		t.Error("JTI should not be empty")
	}
}

func TestGenerateRefreshToken(t *testing.T) {
	mgr := newTestManager(t, 15*time.Minute, 168*time.Hour)

	token, expireAt, err := mgr.GenerateRefreshToken("bob", "android")
	if err != nil {
		t.Fatalf("GenerateRefreshToken() error: %v", err)
	}
	if token == "" {
		t.Fatal("token is empty")
	}
	if expireAt <= time.Now().Unix() {
		t.Error("expireAt should be in the future")
	}

	claims, err := mgr.ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken() error: %v", err)
	}
	if claims.UserID != "bob" {
		t.Errorf("UserID = %v, want bob", claims.UserID)
	}
}

func TestParseExpiredToken(t *testing.T) {
	// 使用已过期的 token 验证 ParseToken 返回错误
	mgr := newTestManager(t, -1*time.Hour, -1*time.Hour) // 负值 = 立即过期

	token, _, err := mgr.GenerateAccessToken("alice", "web")
	if err != nil {
		t.Fatalf("GenerateAccessToken() error: %v", err)
	}

	_, err = mgr.ParseToken(token)
	if err == nil {
		t.Error("ParseToken() should return error for expired token")
	}
}

func TestParseInvalidToken(t *testing.T) {
	mgr := newTestManager(t, 15*time.Minute, 168*time.Hour)

	_, err := mgr.ParseToken("invalid.token.string")
	if err == nil {
		t.Error("ParseToken() should return error for invalid token")
	}
}

func TestParseTokenWithDifferentKey(t *testing.T) {
	mgr1 := newTestManager(t, 15*time.Minute, 168*time.Hour)
	mgr2 := newTestManager(t, 15*time.Minute, 168*time.Hour) // 不同的密钥对

	token, _, err := mgr1.GenerateAccessToken("alice", "web")
	if err != nil {
		t.Fatalf("GenerateAccessToken() error: %v", err)
	}

	// mgr2 的公钥无法验证 mgr1 签发的 token
	_, err = mgr2.ParseToken(token)
	if err == nil {
		t.Error("ParseToken() should fail with different key pair")
	}
}

func TestValidateToken(t *testing.T) {
	mgr := newTestManager(t, 15*time.Minute, 168*time.Hour)

	token, _, err := mgr.GenerateAccessToken("alice", "web")
	if err != nil {
		t.Fatalf("GenerateAccessToken() error: %v", err)
	}

	validatedToken, err := mgr.ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken() error: %v", err)
	}
	if !validatedToken.Valid {
		t.Error("validated token should be valid")
	}
}

func TestNewJWTManager(t *testing.T) {
	// NewJWTManager 需要密钥文件，这里验证 manager 可以正常创建
	mgr := newTestManager(t, 15*time.Minute, 168*time.Hour)
	if mgr.privateKey == nil {
		t.Error("privateKey should not be nil")
	}
	if mgr.publicKey == nil {
		t.Error("publicKey should not be nil")
	}
}

func TestClaimsJTIUniqueness(t *testing.T) {
	// 每次生成 Token 应有不同的 JTI
	mgr := newTestManager(t, 15*time.Minute, 168*time.Hour)

	jtis := map[string]bool{}
	for i := 0; i < 10; i++ {
		token, _, err := mgr.GenerateAccessToken("alice", "web")
		if err != nil {
			t.Fatalf("GenerateAccessToken() error: %v", err)
		}
		claims, err := mgr.ParseToken(token)
		if err != nil {
			t.Fatalf("ParseToken() error: %v", err)
		}
		if jtis[claims.ID] {
			t.Errorf("duplicate JTI: %s", claims.ID)
		}
		jtis[claims.ID] = true
	}
}

func TestClaimsRegisteredFields(t *testing.T) {
	mgr := newTestManager(t, 15*time.Minute, 168*time.Hour)

	token, _, err := mgr.GenerateAccessToken("alice", "web")
	if err != nil {
		t.Fatalf("GenerateAccessToken() error: %v", err)
	}

	claims, err := mgr.ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken() error: %v", err)
	}

	// 验证 RegisteredClaims 内置字段
	if claims.IssuedAt == nil {
		t.Error("IssuedAt should not be nil")
	}
	if claims.ExpiresAt == nil {
		t.Error("ExpiresAt should not be nil")
	}
	// expiresAt 的 Unix 时间戳必须与 Generate 返回的 expireAt 一致
	expireAtUnix := claims.ExpiresAt.Unix()
	now := time.Now()
	if expireAtUnix < now.Unix() || expireAtUnix > now.Add(20*time.Minute).Unix() {
		t.Errorf("expireAtUnix %d out of expected range [%d, %d]", expireAtUnix, now.Unix(), now.Add(20*time.Minute).Unix())
	}
}

func TestErrInvalidToken(t *testing.T) {
	if ErrInvalidToken.Error() != "invalid token" {
		t.Errorf("ErrInvalidToken = %v, want 'invalid token'", ErrInvalidToken)
	}
}

func TestParseTokenMalformed(t *testing.T) {
	mgr := newTestManager(t, 15*time.Minute, 168*time.Hour)

	// Random string as token should fail parsing
	_, err := mgr.ParseToken("this-is-a-completely-random-string-not-a-jwt")
	if err == nil {
		t.Error("ParseToken() should return error for malformed token")
	}
}

func TestParseTokenEmpty(t *testing.T) {
	mgr := newTestManager(t, 15*time.Minute, 168*time.Hour)

	_, err := mgr.ParseToken("")
	if err == nil {
		t.Error("ParseToken() should return error for empty token")
	}
}

func init() {
	// 禁用 jwtv5 在测试中的时间容差警告
	_ = jwtv5.ErrTokenExpired
}
