package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/tianlu1990s/gim/internal/model"
	"github.com/tianlu1990s/gim/internal/service"
	"github.com/tianlu1990s/gim/pkg/errcode"
	"github.com/tianlu1990s/gim/pkg/resp"
)

// UserHandler 用户资料相关 HTTP Handler。
type UserHandler struct {
	svc service.UserService
}

func NewUserHandler(svc service.UserService) *UserHandler {
	return &UserHandler{svc: svc}
}

// GetProfile 获取当前用户资料。
// GET /api/v1/user/profile
func (h *UserHandler) GetProfile(c *gin.Context) {
	userID := c.GetString("userID")
	user, err := h.svc.GetProfile(c.Request.Context(), userID)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.Success(c, user)
}

// UpdateProfile 更新用户资料。
// PUT /api/v1/user/profile
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID := c.GetString("userID")
	var req model.UpdateProfileReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, errcode.ErrInvalidParam.WithDetail(err.Error()))
		return
	}
	user, err := h.svc.UpdateProfile(c.Request.Context(), userID, &req)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.Success(c, user)
}

// GetOtherProfile 查看他人资料。
// GET /api/v1/user/profile/:userId
func (h *UserHandler) GetOtherProfile(c *gin.Context) {
	currentUserID := c.GetString("userID")
	targetUserID := c.Param("userId")
	if targetUserID == "" {
		resp.Fail(c, errcode.ErrInvalidParam.WithDetail("userId 不能为空"))
		return
	}
	user, err := h.svc.GetOtherProfile(c.Request.Context(), currentUserID, targetUserID)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.Success(c, user)
}

// Search 搜索用户。
// POST /api/v1/user/search
func (h *UserHandler) Search(c *gin.Context) {
	var req model.SearchReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, errcode.ErrInvalidParam.WithDetail(err.Error()))
		return
	}
	result, err := h.svc.Search(c.Request.Context(), c.GetString("userID"), &req)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.Success(c, result)
}
