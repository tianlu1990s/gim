package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/tianlu1990s/gim/internal/model"
	"github.com/tianlu1990s/gim/internal/service"
	"github.com/tianlu1990s/gim/pkg/errcode"
	"github.com/tianlu1990s/gim/pkg/resp"
)

// AuthHandler 认证相关 HTTP Handler。
// 负责接收 JSON 请求 → 参数绑定 → 调用 Service → 返回统一格式响应。
type AuthHandler struct {
	svc service.AuthService
}

func NewAuthHandler(svc service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

// Register 用户注册。
// POST /api/v1/auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	var req model.RegisterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, errcode.ErrInvalidParam.WithDetail(err.Error()))
		return
	}
	user, err := h.svc.Register(c.Request.Context(), &req)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.Success(c, user.ToVO())
}

// Login 用户登录。
// POST /api/v1/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req model.LoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, errcode.ErrInvalidParam.WithDetail(err.Error()))
		return
	}
	token, err := h.svc.Login(c.Request.Context(), &req)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.Success(c, token)
}

// Refresh 刷新 accessToken。
// POST /api/v1/auth/refresh
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req model.RefreshReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, errcode.ErrInvalidParam.WithDetail(err.Error()))
		return
	}
	token, err := h.svc.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.Success(c, token)
}

// Logout 用户登出。
// POST /api/v1/auth/logout（需鉴权）
func (h *AuthHandler) Logout(c *gin.Context) {
	userID := c.GetString("userID")
	platform := c.GetString("platform")
	accessToken := extractBearerToken(c)
	if err := h.svc.Logout(c.Request.Context(), userID, platform, accessToken); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.Success(c, nil)
}
