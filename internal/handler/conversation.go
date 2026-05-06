package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/tianlu1990s/gim/internal/model"
	"github.com/tianlu1990s/gim/internal/service"
	"github.com/tianlu1990s/gim/pkg/errcode"
	"github.com/tianlu1990s/gim/pkg/resp"
)

// ConversationHandler 会话管理 HTTP Handler。
type ConversationHandler struct {
	svc service.ConversationService
}

func NewConversationHandler(svc service.ConversationService) *ConversationHandler {
	return &ConversationHandler{svc: svc}
}

// List 获取会话列表。
// GET /api/v1/conversation/list
func (h *ConversationHandler) List(c *gin.Context) {
	userID := c.GetString("userID")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	result, err := h.svc.List(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.Success(c, result)
}

// Pin 置顶/取消置顶会话。
// PUT /api/v1/conversation/:id/pin
func (h *ConversationHandler) Pin(c *gin.Context) {
	userID := c.GetString("userID")
	convID := c.Param("id")
	if convID == "" {
		resp.Fail(c, errcode.ErrInvalidParam.WithDetail("会话ID不能为空"))
		return
	}
	var req model.PinConversationReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, errcode.ErrInvalidParam.WithDetail(err.Error()))
		return
	}
	if err := h.svc.Pin(c.Request.Context(), userID, convID, req.IsPinned); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.Success(c, nil)
}

// Delete 删除会话。
// DELETE /api/v1/conversation/:id
func (h *ConversationHandler) Delete(c *gin.Context) {
	userID := c.GetString("userID")
	convID := c.Param("id")
	if convID == "" {
		resp.Fail(c, errcode.ErrInvalidParam.WithDetail("会话ID不能为空"))
		return
	}
	if err := h.svc.Delete(c.Request.Context(), userID, convID); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.Success(c, nil)
}
