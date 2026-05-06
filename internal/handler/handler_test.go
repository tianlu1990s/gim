package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/tianlu1990s/gim/internal/model"
	"github.com/tianlu1990s/gim/internal/service"
	"github.com/tianlu1990s/gim/pkg/errcode"
)

// ============================================================
// Mock: AuthService
// ============================================================

type mockAuthService struct {
	RegisterFunc func(ctx context.Context, req *model.RegisterReq) (*model.User, error)
	LoginFunc    func(ctx context.Context, req *model.LoginReq) (*model.TokenPair, error)
	RefreshFunc  func(ctx context.Context, refreshToken string) (*model.TokenPair, error)
	LogoutFunc   func(ctx context.Context, userID, platform, accessToken string) error
}

func (m *mockAuthService) Register(ctx context.Context, req *model.RegisterReq) (*model.User, error) {
	return m.RegisterFunc(ctx, req)
}
func (m *mockAuthService) Login(ctx context.Context, req *model.LoginReq) (*model.TokenPair, error) {
	return m.LoginFunc(ctx, req)
}
func (m *mockAuthService) Refresh(ctx context.Context, refreshToken string) (*model.TokenPair, error) {
	return m.RefreshFunc(ctx, refreshToken)
}
func (m *mockAuthService) Logout(ctx context.Context, userID, platform, accessToken string) error {
	return m.LogoutFunc(ctx, userID, platform, accessToken)
}

// ============================================================
// Mock: UserService
// ============================================================

type mockUserService struct {
	GetProfileFunc      func(ctx context.Context, userID string) (*model.UserVO, error)
	UpdateProfileFunc   func(ctx context.Context, userID string, req *model.UpdateProfileReq) (*model.UserVO, error)
	GetOtherProfileFunc func(ctx context.Context, currentUserID, targetUserID string) (*model.OtherUserVO, error)
	SearchFunc          func(ctx context.Context, userID string, req *model.SearchReq) (*model.PageResult[*model.SearchUserVO], error)
}

func (m *mockUserService) GetProfile(ctx context.Context, userID string) (*model.UserVO, error) {
	return m.GetProfileFunc(ctx, userID)
}
func (m *mockUserService) UpdateProfile(ctx context.Context, userID string, req *model.UpdateProfileReq) (*model.UserVO, error) {
	return m.UpdateProfileFunc(ctx, userID, req)
}
func (m *mockUserService) GetOtherProfile(ctx context.Context, currentUserID, targetUserID string) (*model.OtherUserVO, error) {
	return m.GetOtherProfileFunc(ctx, currentUserID, targetUserID)
}
func (m *mockUserService) Search(ctx context.Context, userID string, req *model.SearchReq) (*model.PageResult[*model.SearchUserVO], error) {
	return m.SearchFunc(ctx, userID, req)
}

// ============================================================
// Mock: FriendService
// ============================================================

type mockFriendService struct {
	SendRequestFunc    func(ctx context.Context, userID string, req *model.SendFriendRequestReq) (int64, error)
	ListRequestsFunc   func(ctx context.Context, userID string, page, pageSize int) ([]*model.FriendRequestVO, int64, error)
	AcceptRequestFunc  func(ctx context.Context, userID string, requestID int64) error
	RejectRequestFunc  func(ctx context.Context, userID string, requestID int64) error
	DeleteFunc         func(ctx context.Context, ownerID, friendID string) error
	ListFunc           func(ctx context.Context, ownerID string, page, pageSize int) ([]*model.FriendVO, int64, error)
	SetRemarkFunc      func(ctx context.Context, ownerID, friendID, remark string) error
}

func (m *mockFriendService) SendRequest(ctx context.Context, userID string, req *model.SendFriendRequestReq) (int64, error) {
	return m.SendRequestFunc(ctx, userID, req)
}
func (m *mockFriendService) ListRequests(ctx context.Context, userID string, page, pageSize int) ([]*model.FriendRequestVO, int64, error) {
	return m.ListRequestsFunc(ctx, userID, page, pageSize)
}
func (m *mockFriendService) AcceptRequest(ctx context.Context, userID string, requestID int64) error {
	return m.AcceptRequestFunc(ctx, userID, requestID)
}
func (m *mockFriendService) RejectRequest(ctx context.Context, userID string, requestID int64) error {
	return m.RejectRequestFunc(ctx, userID, requestID)
}
func (m *mockFriendService) Delete(ctx context.Context, ownerID, friendID string) error {
	return m.DeleteFunc(ctx, ownerID, friendID)
}
func (m *mockFriendService) List(ctx context.Context, ownerID string, page, pageSize int) ([]*model.FriendVO, int64, error) {
	return m.ListFunc(ctx, ownerID, page, pageSize)
}
func (m *mockFriendService) SetRemark(ctx context.Context, ownerID, friendID, remark string) error {
	return m.SetRemarkFunc(ctx, ownerID, friendID, remark)
}

// ============================================================
// Mock: MessageService
// ============================================================

type mockMessageService struct {
	SendMessageFunc func(ctx context.Context, senderID string, req *model.SendMsgReq) (*model.SendMsgResp, error)
	HistoryFunc     func(ctx context.Context, userID string, req *model.HistoryReq) (*service.HistoryResp, error)
	MarkReadFunc    func(ctx context.Context, userID string, req *model.MarkReadReq) error
	RevokeFunc      func(ctx context.Context, userID string, req *model.RevokeMsgReq) error
}

func (m *mockMessageService) SendMessage(ctx context.Context, senderID string, req *model.SendMsgReq) (*model.SendMsgResp, error) {
	return m.SendMessageFunc(ctx, senderID, req)
}
func (m *mockMessageService) History(ctx context.Context, userID string, req *model.HistoryReq) (*service.HistoryResp, error) {
	return m.HistoryFunc(ctx, userID, req)
}
func (m *mockMessageService) MarkRead(ctx context.Context, userID string, req *model.MarkReadReq) error {
	return m.MarkReadFunc(ctx, userID, req)
}
func (m *mockMessageService) Revoke(ctx context.Context, userID string, req *model.RevokeMsgReq) error {
	return m.RevokeFunc(ctx, userID, req)
}

// ============================================================
// Mock: ConversationService
// ============================================================

type mockConversationService struct {
	ListFunc   func(ctx context.Context, userID string, page, pageSize int) (*model.PageResult[*model.ConversationVO], error)
	PinFunc    func(ctx context.Context, userID, convID string, isPinned bool) error
	DeleteFunc func(ctx context.Context, userID, convID string) error
}

func (m *mockConversationService) List(ctx context.Context, userID string, page, pageSize int) (*model.PageResult[*model.ConversationVO], error) {
	return m.ListFunc(ctx, userID, page, pageSize)
}
func (m *mockConversationService) Pin(ctx context.Context, userID, convID string, isPinned bool) error {
	return m.PinFunc(ctx, userID, convID, isPinned)
}
func (m *mockConversationService) Delete(ctx context.Context, userID, convID string) error {
	return m.DeleteFunc(ctx, userID, convID)
}

// ============================================================
// Test Helpers
// ============================================================

func setupJSONContext(method, target string, body []byte) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, target, bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	return c, w
}

func setupQueryContext(target string) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, target, nil)
	return c, w
}

func parseResp(w *httptest.ResponseRecorder) map[string]any {
	var result map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		panic("failed to parse response JSON: " + err.Error())
	}
	return result
}

// newUser creates a User model for test use.
func newUser() *model.User {
	return &model.User{
		UserID:    "testuser",
		Nickname:  "Test User",
		AvatarURL: "https://example.com/avatar.jpg",
		Phone:     "13800138000",
		Email:     "test@example.com",
		Status:    1,
		CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}
}

// newUserVO creates a UserVO from a User.
func newUserVO(u *model.User) *model.UserVO {
	return u.ToVO()
}

// ============================================================
// AuthHandler Tests
// ============================================================

func TestAuthHandler_Register(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("happy path", func(t *testing.T) {
		mockSvc := &mockAuthService{
			RegisterFunc: func(_ context.Context, req *model.RegisterReq) (*model.User, error) {
				if req.UserID != "newuser" || req.Password != "password123" {
					t.Errorf("unexpected req: %+v", req)
				}
				u := newUser()
				u.UserID = req.UserID
				u.Nickname = req.Nickname
				return u, nil
			},
		}
		h := &AuthHandler{svc: mockSvc}
		body, _ := json.Marshal(map[string]any{
			"userId":   "newuser",
			"password": "password123",
			"nickname": "New User",
		})
		c, w := setupJSONContext(http.MethodPost, "/api/v1/auth/register", body)
		h.Register(c)

		resp := parseResp(w)
		if code := resp["code"].(float64); code != 0 {
			t.Errorf("expected code 0, got %v", code)
		}
		data, ok := resp["data"].(map[string]any)
		if !ok {
			t.Fatal("expected data object")
		}
		if data["userId"] != "newuser" {
			t.Errorf("expected userId 'newuser', got %v", data["userId"])
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		mockSvc := &mockAuthService{}
		h := &AuthHandler{svc: mockSvc}
		c, w := setupJSONContext(http.MethodPost, "/api/v1/auth/register", []byte(`{invalid`))
		h.Register(c)

		resp := parseResp(w)
		if code := resp["code"].(float64); code != 40001 {
			t.Errorf("expected code 40001, got %v", code)
		}
	})

	t.Run("service error", func(t *testing.T) {
		mockSvc := &mockAuthService{
			RegisterFunc: func(_ context.Context, req *model.RegisterReq) (*model.User, error) {
				return nil, errcode.ErrUserAlreadyExists
			},
		}
		h := &AuthHandler{svc: mockSvc}
		body, _ := json.Marshal(map[string]any{
			"userId":   "existing",
			"password": "password123",
		})
		c, w := setupJSONContext(http.MethodPost, "/api/v1/auth/register", body)
		h.Register(c)

		resp := parseResp(w)
		if code := resp["code"].(float64); code != 40010 {
			t.Errorf("expected code 40010, got %v", code)
		}
		if msg := resp["msg"].(string); msg != "用户已存在" {
			t.Errorf("expected msg '用户已存在', got %v", msg)
		}
	})
}

func TestAuthHandler_Login(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("happy path", func(t *testing.T) {
		mockSvc := &mockAuthService{
			LoginFunc: func(_ context.Context, req *model.LoginReq) (*model.TokenPair, error) {
				if req.UserID != "testuser" || req.Password != "pass" || req.Platform != "web" {
					t.Errorf("unexpected req: %+v", req)
				}
				return &model.TokenPair{
					AccessToken:  "access_token_abc",
					RefreshToken: "refresh_token_xyz",
					UserID:       "testuser",
				}, nil
			},
		}
		h := &AuthHandler{svc: mockSvc}
		body, _ := json.Marshal(map[string]any{
			"userId":   "testuser",
			"password": "pass",
			"platform": "web",
		})
		c, w := setupJSONContext(http.MethodPost, "/api/v1/auth/login", body)
		h.Login(c)

		resp := parseResp(w)
		if code := resp["code"].(float64); code != 0 {
			t.Errorf("expected code 0, got %v", code)
		}
		data := resp["data"].(map[string]any)
		if data["accessToken"] != "access_token_abc" {
			t.Errorf("unexpected accessToken: %v", data["accessToken"])
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		mockSvc := &mockAuthService{}
		h := &AuthHandler{svc: mockSvc}
		c, w := setupJSONContext(http.MethodPost, "/api/v1/auth/login", []byte(`{bad`))
		h.Login(c)

		resp := parseResp(w)
		if code := resp["code"].(float64); code != 40001 {
			t.Errorf("expected code 40001, got %v", code)
		}
	})

	t.Run("service error", func(t *testing.T) {
		mockSvc := &mockAuthService{
			LoginFunc: func(_ context.Context, req *model.LoginReq) (*model.TokenPair, error) {
				return nil, errcode.ErrUserOrPasswordWrong
			},
		}
		h := &AuthHandler{svc: mockSvc}
		body, _ := json.Marshal(map[string]any{
			"userId":   "nobody",
			"password": "wrong",
			"platform": "web",
		})
		c, w := setupJSONContext(http.MethodPost, "/api/v1/auth/login", body)
		h.Login(c)

		resp := parseResp(w)
		if code := resp["code"].(float64); code != 40110 {
			t.Errorf("expected code 40110, got %v", code)
		}
	})
}

func TestAuthHandler_Refresh(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("happy path", func(t *testing.T) {
		mockSvc := &mockAuthService{
			RefreshFunc: func(_ context.Context, refreshToken string) (*model.TokenPair, error) {
				if refreshToken != "rt_valid" {
					t.Errorf("unexpected refreshToken: %s", refreshToken)
				}
				return &model.TokenPair{
					AccessToken:  "new_access_token",
					RefreshToken: "rt_valid",
					UserID:       "testuser",
				}, nil
			},
		}
		h := &AuthHandler{svc: mockSvc}
		body, _ := json.Marshal(map[string]any{
			"refreshToken": "rt_valid",
			"platform":     "web",
		})
		c, w := setupJSONContext(http.MethodPost, "/api/v1/auth/refresh", body)
		h.Refresh(c)

		resp := parseResp(w)
		if code := resp["code"].(float64); code != 0 {
			t.Errorf("expected code 0, got %v", code)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		mockSvc := &mockAuthService{}
		h := &AuthHandler{svc: mockSvc}
		c, w := setupJSONContext(http.MethodPost, "/api/v1/auth/refresh", []byte(`{bad`))
		h.Refresh(c)

		resp := parseResp(w)
		if code := resp["code"].(float64); code != 40001 {
			t.Errorf("expected code 40001, got %v", code)
		}
	})

	t.Run("service error", func(t *testing.T) {
		mockSvc := &mockAuthService{
			RefreshFunc: func(_ context.Context, refreshToken string) (*model.TokenPair, error) {
				return nil, errcode.ErrUnauthorized.WithDetail("refreshToken 无效或已过期")
			},
		}
		h := &AuthHandler{svc: mockSvc}
		body, _ := json.Marshal(map[string]any{
			"refreshToken": "rt_expired",
			"platform":     "web",
		})
		c, w := setupJSONContext(http.MethodPost, "/api/v1/auth/refresh", body)
		h.Refresh(c)

		resp := parseResp(w)
		if code := resp["code"].(float64); code != 40101 {
			t.Errorf("expected code 40101, got %v", code)
		}
	})
}

func TestAuthHandler_Logout(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("missing userID in context", func(t *testing.T) {
		called := false
		mockSvc := &mockAuthService{
			LogoutFunc: func(_ context.Context, userID, platform, accessToken string) error {
				called = true
				if userID != "" {
					t.Errorf("expected empty userID, got %q", userID)
				}
				return nil
			},
		}
		h := &AuthHandler{svc: mockSvc}
		// Do NOT set "userID" — simulate missing auth context
		c, w := setupJSONContext(http.MethodPost, "/api/v1/auth/logout", nil)
		h.Logout(c)

		resp := parseResp(w)
		if code := resp["code"].(float64); code != 0 {
			t.Errorf("expected code 0, got %v", code)
		}
		if !called {
			t.Error("expected Logout to be called")
		}
	})

	t.Run("success", func(t *testing.T) {
		called := false
		mockSvc := &mockAuthService{
			LogoutFunc: func(_ context.Context, userID, platform, accessToken string) error {
				called = true
				if userID != "testuser" || platform != "web" {
					t.Errorf("unexpected params: userID=%q platform=%q", userID, platform)
				}
				return nil
			},
		}
		h := &AuthHandler{svc: mockSvc}
		c, w := setupJSONContext(http.MethodPost, "/api/v1/auth/logout", nil)
		c.Set("userID", "testuser")
		c.Set("platform", "web")
		c.Request.Header.Set("Authorization", "Bearer some_access_token")
		h.Logout(c)

		resp := parseResp(w)
		if code := resp["code"].(float64); code != 0 {
			t.Errorf("expected code 0, got %v", code)
		}
		if !called {
			t.Error("expected Logout to be called")
		}
	})
}

// ============================================================
// UserHandler Tests
// ============================================================

func TestUserHandler_GetProfile(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("success", func(t *testing.T) {
		mockSvc := &mockUserService{
			GetProfileFunc: func(_ context.Context, userID string) (*model.UserVO, error) {
				if userID != "testuser" {
					t.Errorf("unexpected userID: %s", userID)
				}
				return newUserVO(newUser()), nil
			},
		}
		h := &UserHandler{svc: mockSvc}
		c, w := setupJSONContext(http.MethodGet, "/api/v1/user/profile", nil)
		c.Set("userID", "testuser")
		h.GetProfile(c)

		resp := parseResp(w)
		if code := resp["code"].(float64); code != 0 {
			t.Errorf("expected code 0, got %v", code)
		}
		data := resp["data"].(map[string]any)
		if data["userId"] != "testuser" {
			t.Errorf("expected userId 'testuser', got %v", data["userId"])
		}
	})

	t.Run("service not found", func(t *testing.T) {
		mockSvc := &mockUserService{
			GetProfileFunc: func(_ context.Context, userID string) (*model.UserVO, error) {
				return nil, errcode.ErrUserNotFound
			},
		}
		h := &UserHandler{svc: mockSvc}
		c, w := setupJSONContext(http.MethodGet, "/api/v1/user/profile", nil)
		c.Set("userID", "nonexistent")
		h.GetProfile(c)

		resp := parseResp(w)
		if code := resp["code"].(float64); code != 40410 {
			t.Errorf("expected code 40410, got %v", code)
		}
		if msg := resp["msg"].(string); msg != "用户不存在" {
			t.Errorf("expected msg '用户不存在', got %v", msg)
		}
	})
}

func TestUserHandler_UpdateProfile(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("success", func(t *testing.T) {
		mockSvc := &mockUserService{
			UpdateProfileFunc: func(_ context.Context, userID string, req *model.UpdateProfileReq) (*model.UserVO, error) {
				if userID != "testuser" || req.Nickname != "NewNick" {
					t.Errorf("unexpected params: userID=%s req=%+v", userID, req)
				}
				u := newUser()
				u.Nickname = "NewNick"
				return u.ToVO(), nil
			},
		}
		h := &UserHandler{svc: mockSvc}
		body, _ := json.Marshal(map[string]any{
			"nickname": "NewNick",
		})
		c, w := setupJSONContext(http.MethodPut, "/api/v1/user/profile", body)
		c.Set("userID", "testuser")
		h.UpdateProfile(c)

		resp := parseResp(w)
		if code := resp["code"].(float64); code != 0 {
			t.Errorf("expected code 0, got %v", code)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		mockSvc := &mockUserService{}
		h := &UserHandler{svc: mockSvc}
		c, w := setupJSONContext(http.MethodPut, "/api/v1/user/profile", []byte(`{bad`))
		c.Set("userID", "testuser")
		h.UpdateProfile(c)

		resp := parseResp(w)
		if code := resp["code"].(float64); code != 40001 {
			t.Errorf("expected code 40001, got %v", code)
		}
	})
}

func TestUserHandler_GetOtherProfile(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("success", func(t *testing.T) {
		mockSvc := &mockUserService{
			GetOtherProfileFunc: func(_ context.Context, currentUserID, targetUserID string) (*model.OtherUserVO, error) {
				if currentUserID != "testuser" || targetUserID != "otheruser" {
					t.Errorf("unexpected params: current=%s target=%s", currentUserID, targetUserID)
				}
				return &model.OtherUserVO{
					UserID:    "otheruser",
					Nickname:  "Other User",
					IsFriend:  true,
					Remark:    "my friend",
				}, nil
			},
		}
		h := &UserHandler{svc: mockSvc}
		c, w := setupJSONContext(http.MethodGet, "/api/v1/user/profile/otheruser", nil)
		c.Set("userID", "testuser")
		c.Params = []gin.Param{{Key: "userId", Value: "otheruser"}}
		h.GetOtherProfile(c)

		resp := parseResp(w)
		if code := resp["code"].(float64); code != 0 {
			t.Errorf("expected code 0, got %v", code)
		}
		data := resp["data"].(map[string]any)
		if data["userId"] != "otheruser" {
			t.Errorf("expected userId 'otheruser', got %v", data["userId"])
		}
		if data["isFriend"] != true {
			t.Errorf("expected isFriend true, got %v", data["isFriend"])
		}
		if data["remark"] != "my friend" {
			t.Errorf("expected remark 'my friend', got %v", data["remark"])
		}
	})

	t.Run("missing userId param", func(t *testing.T) {
		mockSvc := &mockUserService{}
		h := &UserHandler{svc: mockSvc}
		c, w := setupJSONContext(http.MethodGet, "/api/v1/user/profile/", nil)
		c.Set("userID", "testuser")
		// Do NOT set Params — c.Param("userId") returns ""
		h.GetOtherProfile(c)

		resp := parseResp(w)
		if code := resp["code"].(float64); code != 40001 {
			t.Errorf("expected code 40001, got %v", code)
		}
		if msg := resp["msg"].(string); msg != "参数错误" {
			t.Errorf("expected msg '参数错误', got %v", msg)
		}
		if detail := resp["detail"].(string); detail != "userId 不能为空" {
			t.Errorf("expected detail 'userId 不能为空', got %v", detail)
		}
	})
}

func TestUserHandler_Search(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("success", func(t *testing.T) {
		mockSvc := &mockUserService{
			SearchFunc: func(_ context.Context, userID string, req *model.SearchReq) (*model.PageResult[*model.SearchUserVO], error) {
				if userID != "testuser" || req.Keyword != "alice" {
					t.Errorf("unexpected params: userID=%s req=%+v", userID, req)
				}
				return &model.PageResult[*model.SearchUserVO]{
					List: []*model.SearchUserVO{
						{UserID: "alice", Nickname: "Alice"},
					},
					Total:    1,
					Page:     1,
					PageSize: 20,
				}, nil
			},
		}
		h := &UserHandler{svc: mockSvc}
		body, _ := json.Marshal(map[string]any{
			"keyword": "alice",
		})
		c, w := setupJSONContext(http.MethodPost, "/api/v1/user/search", body)
		c.Set("userID", "testuser")
		h.Search(c)

		resp := parseResp(w)
		if code := resp["code"].(float64); code != 0 {
			t.Errorf("expected code 0, got %v", code)
		}
		data := resp["data"].(map[string]any)
		total := data["total"].(float64)
		if total != 1 {
			t.Errorf("expected total 1, got %v", total)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		mockSvc := &mockUserService{}
		h := &UserHandler{svc: mockSvc}
		c, w := setupJSONContext(http.MethodPost, "/api/v1/user/search", []byte(`{bad`))
		c.Set("userID", "testuser")
		h.Search(c)

		resp := parseResp(w)
		if code := resp["code"].(float64); code != 40001 {
			t.Errorf("expected code 40001, got %v", code)
		}
	})
}

// ============================================================
// FriendHandler Tests
// ============================================================

func TestFriendHandler_SendRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("success", func(t *testing.T) {
		mockSvc := &mockFriendService{
			SendRequestFunc: func(_ context.Context, userID string, req *model.SendFriendRequestReq) (int64, error) {
				if userID != "testuser" || req.ToUserID != "friend1" {
					t.Errorf("unexpected params: userID=%s req=%+v", userID, req)
				}
				return 42, nil
			},
		}
		h := &FriendHandler{svc: mockSvc}
		body, _ := json.Marshal(map[string]any{
			"toUserId": "friend1",
			"message":  "hello!",
		})
		c, w := setupJSONContext(http.MethodPost, "/api/v1/friend/request", body)
		c.Set("userID", "testuser")
		h.SendRequest(c)

		resp := parseResp(w)
		if code := resp["code"].(float64); code != 0 {
			t.Errorf("expected code 0, got %v", code)
		}
		data := resp["data"].(map[string]any)
		if data["id"].(float64) != 42 {
			t.Errorf("expected id 42, got %v", data["id"])
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		mockSvc := &mockFriendService{}
		h := &FriendHandler{svc: mockSvc}
		c, w := setupJSONContext(http.MethodPost, "/api/v1/friend/request", []byte(`{bad`))
		c.Set("userID", "testuser")
		h.SendRequest(c)

		resp := parseResp(w)
		if code := resp["code"].(float64); code != 40001 {
			t.Errorf("expected code 40001, got %v", code)
		}
	})

	t.Run("self-friend error", func(t *testing.T) {
		mockSvc := &mockFriendService{
			SendRequestFunc: func(_ context.Context, userID string, req *model.SendFriendRequestReq) (int64, error) {
				if userID == req.ToUserID {
					return 0, errcode.ErrCannotFriendSelf
				}
				return 1, nil
			},
		}
		h := &FriendHandler{svc: mockSvc}
		// Send friend request to self
		body, _ := json.Marshal(map[string]any{
			"toUserId": "testuser",
		})
		c, w := setupJSONContext(http.MethodPost, "/api/v1/friend/request", body)
		c.Set("userID", "testuser")
		h.SendRequest(c)

		resp := parseResp(w)
		if code := resp["code"].(float64); code != 40023 {
			t.Errorf("expected code 40023, got %v", code)
		}
		if msg := resp["msg"].(string); msg != "不能添加自己为好友" {
			t.Errorf("expected msg '不能添加自己为好友', got %v", msg)
		}
	})
}

func TestFriendHandler_ListRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("success with default pagination", func(t *testing.T) {
		mockSvc := &mockFriendService{
			ListRequestsFunc: func(_ context.Context, userID string, page, pageSize int) ([]*model.FriendRequestVO, int64, error) {
				if userID != "testuser" || page != 1 || pageSize != 20 {
					t.Errorf("unexpected params: userID=%s page=%d pageSize=%d", userID, page, pageSize)
				}
				return []*model.FriendRequestVO{
					{
						ID:         1,
						FromUserID: "alice",
						Nickname:   "Alice",
						Message:    "add me",
						Status:     0,
					},
				}, 1, nil
			},
		}
		h := &FriendHandler{svc: mockSvc}
		// No query params — rely on defaults
		c, w := setupQueryContext("/api/v1/friend/request/incoming")
		c.Set("userID", "testuser")
		h.ListRequests(c)

		resp := parseResp(w)
		if code := resp["code"].(float64); code != 0 {
			t.Errorf("expected code 0, got %v", code)
		}
		data := resp["data"].(map[string]any)
		if data["total"].(float64) != 1 {
			t.Errorf("expected total 1, got %v", data["total"])
		}
		if data["page"].(float64) != 1 {
			t.Errorf("expected page 1, got %v", data["page"])
		}
		if data["pageSize"].(float64) != 20 {
			t.Errorf("expected pageSize 20, got %v", data["pageSize"])
		}
	})

	t.Run("success with custom pagination", func(t *testing.T) {
		mockSvc := &mockFriendService{
			ListRequestsFunc: func(_ context.Context, userID string, page, pageSize int) ([]*model.FriendRequestVO, int64, error) {
				if page != 2 || pageSize != 10 {
					t.Errorf("unexpected pagination: page=%d pageSize=%d", page, pageSize)
				}
				return nil, 0, nil
			},
		}
		h := &FriendHandler{svc: mockSvc}
		c, w := setupQueryContext("/api/v1/friend/request/incoming?page=2&pageSize=10")
		c.Set("userID", "testuser")
		h.ListRequests(c)

		resp := parseResp(w)
		if code := resp["code"].(float64); code != 0 {
			t.Errorf("expected code 0, got %v", code)
		}
	})
}

func TestFriendHandler_AcceptRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("invalid id param", func(t *testing.T) {
		mockSvc := &mockFriendService{}
		h := &FriendHandler{svc: mockSvc}
		c, w := setupJSONContext(http.MethodPost, "/api/v1/friend/request/abc/accept", nil)
		c.Set("userID", "testuser")
		c.Params = []gin.Param{{Key: "id", Value: "abc"}}
		h.AcceptRequest(c)

		resp := parseResp(w)
		if code := resp["code"].(float64); code != 40001 {
			t.Errorf("expected code 40001, got %v", code)
		}
		if detail := resp["detail"].(string); detail != "无效的申请ID" {
			t.Errorf("expected detail '无效的申请ID', got %v", detail)
		}
	})

	t.Run("success", func(t *testing.T) {
		called := false
		mockSvc := &mockFriendService{
			AcceptRequestFunc: func(_ context.Context, userID string, requestID int64) error {
				called = true
				if userID != "testuser" || requestID != 99 {
					t.Errorf("unexpected params: userID=%s requestID=%d", userID, requestID)
				}
				return nil
			},
		}
		h := &FriendHandler{svc: mockSvc}
		c, w := setupJSONContext(http.MethodPost, "/api/v1/friend/request/99/accept", nil)
		c.Set("userID", "testuser")
		c.Params = []gin.Param{{Key: "id", Value: "99"}}
		h.AcceptRequest(c)

		resp := parseResp(w)
		if code := resp["code"].(float64); code != 0 {
			t.Errorf("expected code 0, got %v", code)
		}
		if !called {
			t.Error("expected AcceptRequest to be called")
		}
	})
}

func TestFriendHandler_RejectRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("invalid id param", func(t *testing.T) {
		mockSvc := &mockFriendService{}
		h := &FriendHandler{svc: mockSvc}
		c, w := setupJSONContext(http.MethodPost, "/api/v1/friend/request/xyz/reject", nil)
		c.Set("userID", "testuser")
		c.Params = []gin.Param{{Key: "id", Value: "xyz"}}
		h.RejectRequest(c)

		resp := parseResp(w)
		if code := resp["code"].(float64); code != 40001 {
			t.Errorf("expected code 40001, got %v", code)
		}
	})

	t.Run("success", func(t *testing.T) {
		called := false
		mockSvc := &mockFriendService{
			RejectRequestFunc: func(_ context.Context, userID string, requestID int64) error {
				called = true
				if userID != "testuser" || requestID != 55 {
					t.Errorf("unexpected params: userID=%s requestID=%d", userID, requestID)
				}
				return nil
			},
		}
		h := &FriendHandler{svc: mockSvc}
		c, w := setupJSONContext(http.MethodPost, "/api/v1/friend/request/55/reject", nil)
		c.Set("userID", "testuser")
		c.Params = []gin.Param{{Key: "id", Value: "55"}}
		h.RejectRequest(c)

		resp := parseResp(w)
		if code := resp["code"].(float64); code != 0 {
			t.Errorf("expected code 0, got %v", code)
		}
		if !called {
			t.Error("expected RejectRequest to be called")
		}
	})
}

func TestFriendHandler_Delete(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("missing userId param", func(t *testing.T) {
		mockSvc := &mockFriendService{}
		h := &FriendHandler{svc: mockSvc}
		c, w := setupJSONContext(http.MethodDelete, "/api/v1/friend/", nil)
		c.Set("userID", "testuser")
		// No params — c.Param("userId") returns ""
		h.Delete(c)

		resp := parseResp(w)
		if code := resp["code"].(float64); code != 40001 {
			t.Errorf("expected code 40001, got %v", code)
		}
		if detail := resp["detail"].(string); detail != "userId 不能为空" {
			t.Errorf("expected detail 'userId 不能为空', got %v", detail)
		}
	})

	t.Run("success", func(t *testing.T) {
		called := false
		mockSvc := &mockFriendService{
			DeleteFunc: func(_ context.Context, ownerID, friendID string) error {
				called = true
				if ownerID != "testuser" || friendID != "friend2" {
					t.Errorf("unexpected params: owner=%s friend=%s", ownerID, friendID)
				}
				return nil
			},
		}
		h := &FriendHandler{svc: mockSvc}
		c, w := setupJSONContext(http.MethodDelete, "/api/v1/friend/friend2", nil)
		c.Set("userID", "testuser")
		c.Params = []gin.Param{{Key: "userId", Value: "friend2"}}
		h.Delete(c)

		resp := parseResp(w)
		if code := resp["code"].(float64); code != 0 {
			t.Errorf("expected code 0, got %v", code)
		}
		if !called {
			t.Error("expected Delete to be called")
		}
	})
}

func TestFriendHandler_List(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("success", func(t *testing.T) {
		mockSvc := &mockFriendService{
			ListFunc: func(_ context.Context, ownerID string, page, pageSize int) ([]*model.FriendVO, int64, error) {
				if ownerID != "testuser" || page != 1 || pageSize != 20 {
					t.Errorf("unexpected params: owner=%s page=%d pageSize=%d", ownerID, page, pageSize)
				}
				return []*model.FriendVO{
					{FriendID: "alice", Nickname: "Alice", Remark: "AA"},
					{FriendID: "bob", Nickname: "Bob", Remark: "BB"},
				}, 2, nil
			},
		}
		h := &FriendHandler{svc: mockSvc}
		c, w := setupQueryContext("/api/v1/friend/list")
		c.Set("userID", "testuser")
		h.List(c)

		resp := parseResp(w)
		if code := resp["code"].(float64); code != 0 {
			t.Errorf("expected code 0, got %v", code)
		}
		data := resp["data"].(map[string]any)
		if data["total"].(float64) != 2 {
			t.Errorf("expected total 2, got %v", data["total"])
		}
		list := data["list"].([]any)
		if len(list) != 2 {
			t.Errorf("expected 2 friends, got %d", len(list))
		}
	})
}

func TestFriendHandler_SetRemark(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("success", func(t *testing.T) {
		called := false
		mockSvc := &mockFriendService{
			SetRemarkFunc: func(_ context.Context, ownerID, friendID, remark string) error {
				called = true
				if ownerID != "testuser" || friendID != "friend1" || remark != "BestFriend" {
					t.Errorf("unexpected params: owner=%s friend=%s remark=%s", ownerID, friendID, remark)
				}
				return nil
			},
		}
		h := &FriendHandler{svc: mockSvc}
		body, _ := json.Marshal(map[string]any{
			"remark": "BestFriend",
		})
		c, w := setupJSONContext(http.MethodPut, "/api/v1/friend/friend1/remark", body)
		c.Set("userID", "testuser")
		c.Params = []gin.Param{{Key: "userId", Value: "friend1"}}
		h.SetRemark(c)

		resp := parseResp(w)
		if code := resp["code"].(float64); code != 0 {
			t.Errorf("expected code 0, got %v", code)
		}
		if !called {
			t.Error("expected SetRemark to be called")
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		mockSvc := &mockFriendService{}
		h := &FriendHandler{svc: mockSvc}
		c, w := setupJSONContext(http.MethodPut, "/api/v1/friend/friend1/remark", []byte(`{bad`))
		c.Set("userID", "testuser")
		c.Params = []gin.Param{{Key: "userId", Value: "friend1"}}
		h.SetRemark(c)

		resp := parseResp(w)
		if code := resp["code"].(float64); code != 40001 {
			t.Errorf("expected code 40001, got %v", code)
		}
	})
}

// ============================================================
// MessageHandler Tests
// ============================================================

func TestMessageHandler_Send(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("success", func(t *testing.T) {
		mockSvc := &mockMessageService{
			SendMessageFunc: func(_ context.Context, senderID string, req *model.SendMsgReq) (*model.SendMsgResp, error) {
				if senderID != "testuser" || req.ConversationID != "conv1" || req.Content != "hello" {
					t.Errorf("unexpected params: sender=%s req=%+v", senderID, req)
				}
				return &model.SendMsgResp{
					Seq:         1,
					ServerMsgID: "snowflake_123",
					SendTime:    1700000000000,
				}, nil
			},
		}
		h := &MessageHandler{svc: mockSvc}
		body, _ := json.Marshal(map[string]any{
			"conversationId": "conv1",
			"clientMsgId":    "cmid_001",
			"contentType":    1,
			"content":        "hello",
		})
		c, w := setupJSONContext(http.MethodPost, "/api/v1/msg/send", body)
		c.Set("userID", "testuser")
		h.Send(c)

		resp := parseResp(w)
		if code := resp["code"].(float64); code != 0 {
			t.Errorf("expected code 0, got %v", code)
		}
		data := resp["data"].(map[string]any)
		if data["seq"].(float64) != 1 {
			t.Errorf("expected seq 1, got %v", data["seq"])
		}
		if data["serverMsgId"] != "snowflake_123" {
			t.Errorf("unexpected serverMsgId: %v", data["serverMsgId"])
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		mockSvc := &mockMessageService{}
		h := &MessageHandler{svc: mockSvc}
		c, w := setupJSONContext(http.MethodPost, "/api/v1/msg/send", []byte(`{bad`))
		c.Set("userID", "testuser")
		h.Send(c)

		resp := parseResp(w)
		if code := resp["code"].(float64); code != 40001 {
			t.Errorf("expected code 40001, got %v", code)
		}
	})
}

func TestMessageHandler_History(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("success with ShouldBindQuery", func(t *testing.T) {
		mockSvc := &mockMessageService{
			HistoryFunc: func(_ context.Context, userID string, req *model.HistoryReq) (*service.HistoryResp, error) {
				if userID != "testuser" || req.ConversationID != "conv1" || req.Count != 10 {
					t.Errorf("unexpected params: userID=%s req=%+v", userID, req)
				}
				return &service.HistoryResp{
					List:    []*model.Message{},
					HasMore: false,
					MinSeq:  0,
					MaxSeq:  5,
				}, nil
			},
		}
		h := &MessageHandler{svc: mockSvc}
		c, w := setupQueryContext("/api/v1/msg/history?conversationId=conv1&count=10")
		c.Set("userID", "testuser")
		h.History(c)

		resp := parseResp(w)
		if code := resp["code"].(float64); code != 0 {
			t.Errorf("expected code 0, got %v", code)
		}
		data := resp["data"].(map[string]any)
		if data["maxSeq"].(float64) != 5 {
			t.Errorf("expected maxSeq 5, got %v", data["maxSeq"])
		}
	})

	t.Run("missing required query param", func(t *testing.T) {
		mockSvc := &mockMessageService{}
		h := &MessageHandler{svc: mockSvc}
		// Missing "conversationId" and "count"
		c, w := setupQueryContext("/api/v1/msg/history")
		c.Set("userID", "testuser")
		h.History(c)

		resp := parseResp(w)
		if code := resp["code"].(float64); code != 40001 {
			t.Errorf("expected code 40001, got %v", code)
		}
	})
}

func TestMessageHandler_MarkRead(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("success", func(t *testing.T) {
		called := false
		mockSvc := &mockMessageService{
			MarkReadFunc: func(_ context.Context, userID string, req *model.MarkReadReq) error {
				called = true
				if userID != "testuser" || req.ConversationID != "conv1" || req.ReadSeq != 10 {
					t.Errorf("unexpected params: userID=%s req=%+v", userID, req)
				}
				return nil
			},
		}
		h := &MessageHandler{svc: mockSvc}
		body, _ := json.Marshal(map[string]any{
			"conversationId": "conv1",
			"readSeq":        10,
		})
		c, w := setupJSONContext(http.MethodPost, "/api/v1/msg/read", body)
		c.Set("userID", "testuser")
		h.MarkRead(c)

		resp := parseResp(w)
		if code := resp["code"].(float64); code != 0 {
			t.Errorf("expected code 0, got %v", code)
		}
		if !called {
			t.Error("expected MarkRead to be called")
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		mockSvc := &mockMessageService{}
		h := &MessageHandler{svc: mockSvc}
		c, w := setupJSONContext(http.MethodPost, "/api/v1/msg/read", []byte(`{bad`))
		c.Set("userID", "testuser")
		h.MarkRead(c)

		resp := parseResp(w)
		if code := resp["code"].(float64); code != 40001 {
			t.Errorf("expected code 40001, got %v", code)
		}
	})
}

func TestMessageHandler_Revoke(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("success", func(t *testing.T) {
		called := false
		mockSvc := &mockMessageService{
			RevokeFunc: func(_ context.Context, userID string, req *model.RevokeMsgReq) error {
				called = true
				if userID != "testuser" || req.ConversationID != "conv1" || req.ClientMsgID != "cmid_001" {
					t.Errorf("unexpected params: userID=%s req=%+v", userID, req)
				}
				return nil
			},
		}
		h := &MessageHandler{svc: mockSvc}
		body, _ := json.Marshal(map[string]any{
			"conversationId": "conv1",
			"clientMsgId":    "cmid_001",
		})
		c, w := setupJSONContext(http.MethodPost, "/api/v1/msg/revoke", body)
		c.Set("userID", "testuser")
		h.Revoke(c)

		resp := parseResp(w)
		if code := resp["code"].(float64); code != 0 {
			t.Errorf("expected code 0, got %v", code)
		}
		if !called {
			t.Error("expected Revoke to be called")
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		mockSvc := &mockMessageService{}
		h := &MessageHandler{svc: mockSvc}
		c, w := setupJSONContext(http.MethodPost, "/api/v1/msg/revoke", []byte(`{bad`))
		c.Set("userID", "testuser")
		h.Revoke(c)

		resp := parseResp(w)
		if code := resp["code"].(float64); code != 40001 {
			t.Errorf("expected code 40001, got %v", code)
		}
	})
}

// ============================================================
// ConversationHandler Tests
// ============================================================

func TestConversationHandler_List(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("success", func(t *testing.T) {
		mockSvc := &mockConversationService{
			ListFunc: func(_ context.Context, userID string, page, pageSize int) (*model.PageResult[*model.ConversationVO], error) {
				if userID != "testuser" || page != 1 || pageSize != 20 {
					t.Errorf("unexpected params: userID=%s page=%d pageSize=%d", userID, page, pageSize)
				}
				return &model.PageResult[*model.ConversationVO]{
					List: []*model.ConversationVO{
						{ConversationID: "conv1", ConvType: 1, TargetID: "alice", UnreadCount: 3},
					},
					Total:    1,
					Page:     1,
					PageSize: 20,
				}, nil
			},
		}
		h := &ConversationHandler{svc: mockSvc}
		c, w := setupQueryContext("/api/v1/conversation/list")
		c.Set("userID", "testuser")
		h.List(c)

		resp := parseResp(w)
		if code := resp["code"].(float64); code != 0 {
			t.Errorf("expected code 0, got %v", code)
		}
		data := resp["data"].(map[string]any)
		list := data["list"].([]any)
		if len(list) != 1 {
			t.Errorf("expected 1 conversation, got %d", len(list))
		}
		if data["total"].(float64) != 1 {
			t.Errorf("expected total 1, got %v", data["total"])
		}
	})
}

func TestConversationHandler_Pin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("success", func(t *testing.T) {
		called := false
		mockSvc := &mockConversationService{
			PinFunc: func(_ context.Context, userID, convID string, isPinned bool) error {
				called = true
				if userID != "testuser" || convID != "conv1" || isPinned != true {
					t.Errorf("unexpected params: userID=%s convID=%s isPinned=%v", userID, convID, isPinned)
				}
				return nil
			},
		}
		h := &ConversationHandler{svc: mockSvc}
		body, _ := json.Marshal(map[string]any{
			"isPinned": true,
		})
		c, w := setupJSONContext(http.MethodPut, "/api/v1/conversation/conv1/pin", body)
		c.Set("userID", "testuser")
		c.Params = []gin.Param{{Key: "id", Value: "conv1"}}
		h.Pin(c)

		resp := parseResp(w)
		if code := resp["code"].(float64); code != 0 {
			t.Errorf("expected code 0, got %v", code)
		}
		if !called {
			t.Error("expected Pin to be called")
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		mockSvc := &mockConversationService{}
		h := &ConversationHandler{svc: mockSvc}
		c, w := setupJSONContext(http.MethodPut, "/api/v1/conversation/conv1/pin", []byte(`{bad`))
		c.Set("userID", "testuser")
		c.Params = []gin.Param{{Key: "id", Value: "conv1"}}
		h.Pin(c)

		resp := parseResp(w)
		if code := resp["code"].(float64); code != 40001 {
			t.Errorf("expected code 40001, got %v", code)
		}
	})
}

func TestConversationHandler_Delete(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("success", func(t *testing.T) {
		called := false
		mockSvc := &mockConversationService{
			DeleteFunc: func(_ context.Context, userID, convID string) error {
				called = true
				if userID != "testuser" || convID != "conv1" {
					t.Errorf("unexpected params: userID=%s convID=%s", userID, convID)
				}
				return nil
			},
		}
		h := &ConversationHandler{svc: mockSvc}
		c, w := setupJSONContext(http.MethodDelete, "/api/v1/conversation/conv1", nil)
		c.Set("userID", "testuser")
		c.Params = []gin.Param{{Key: "id", Value: "conv1"}}
		h.Delete(c)

		resp := parseResp(w)
		if code := resp["code"].(float64); code != 0 {
			t.Errorf("expected code 0, got %v", code)
		}
		if !called {
			t.Error("expected Delete to be called")
		}
	})

	t.Run("empty convID param", func(t *testing.T) {
		mockSvc := &mockConversationService{}
		h := &ConversationHandler{svc: mockSvc}
		c, w := setupJSONContext(http.MethodDelete, "/api/v1/conversation/", nil)
		c.Set("userID", "testuser")
		// No params — c.Param("id") returns ""
		h.Delete(c)

		resp := parseResp(w)
		if code := resp["code"].(float64); code != 40001 {
			t.Errorf("expected code 40001, got %v", code)
		}
		if detail := resp["detail"].(string); detail != "会话ID不能为空" {
			t.Errorf("expected detail '会话ID不能为空', got %v", detail)
		}
	})
}

// ============================================================
// extractBearerToken Tests
// ============================================================

func TestExtractBearerToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("valid Bearer token", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		c.Request.Header.Set("Authorization", "Bearer my_access_token_123")
		token := extractBearerToken(c)
		if token != "my_access_token_123" {
			t.Errorf("expected 'my_access_token_123', got %q", token)
		}
	})

	t.Run("non-Bearer header returns empty", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		c.Request.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
		token := extractBearerToken(c)
		if token != "" {
			t.Errorf("expected empty token for non-Bearer header, got %q", token)
		}
	})

	t.Run("empty header returns empty", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		// No Authorization header
		token := extractBearerToken(c)
		if token != "" {
			t.Errorf("expected empty token for empty header, got %q", token)
		}
	})

	t.Run("header too short returns empty", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		c.Request.Header.Set("Authorization", "Bear") // Less than 7 chars for "Bearer "
		token := extractBearerToken(c)
		if token != "" {
			t.Errorf("expected empty token for short header, got %q", token)
		}
	})

	t.Run("case-insensitive Bearer prefix", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		c.Request.Header.Set("Authorization", "bearer case_insensitive_token")
		token := extractBearerToken(c)
		if token != "case_insensitive_token" {
			t.Errorf("expected 'case_insensitive_token', got %q", token)
		}
	})
}
