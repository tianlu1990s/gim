package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/tianlu1990s/gim/internal/model"
	"github.com/tianlu1990s/gim/internal/service"
	"github.com/tianlu1990s/gim/pkg/errcode"
	"github.com/tianlu1990s/gim/pkg/resp"
)

// MessageHandler 消息相关 HTTP Handler。
type MessageHandler struct {
	svc service.MessageService
}

func NewMessageHandler(svc service.MessageService) *MessageHandler {
	return &MessageHandler{svc: svc}
}

// History 查询历史消息。
// GET /api/v1/msg/history
func (h *MessageHandler) History(c *gin.Context) {
	userID := c.GetString("userID")
	var req model.HistoryReq
	if err := c.ShouldBindQuery(&req); err != nil {
		resp.Fail(c, errcode.ErrInvalidParam.WithDetail(err.Error()))
		return
	}
	result, err := h.svc.History(c.Request.Context(), userID, &req)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.Success(c, result)
}

// MarkRead 标记已读。
// POST /api/v1/msg/read
func (h *MessageHandler) MarkRead(c *gin.Context) {
	userID := c.GetString("userID")
	var req model.MarkReadReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, errcode.ErrInvalidParam.WithDetail(err.Error()))
		return
	}
	if err := h.svc.MarkRead(c.Request.Context(), userID, &req); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.Success(c, nil)
}

// Send 发送消息。
// POST /api/v1/msg/send
func (h *MessageHandler) Send(c *gin.Context) {
	userID := c.GetString("userID")
	var req model.SendMsgReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, errcode.ErrInvalidParam.WithDetail(err.Error()))
		return
	}
	result, err := h.svc.SendMessage(c.Request.Context(), userID, &req)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.Success(c, result)
}

// Revoke 撤回消息。
// POST /api/v1/msg/revoke
func (h *MessageHandler) Revoke(c *gin.Context) {
	userID := c.GetString("userID")
	var req model.RevokeMsgReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, errcode.ErrInvalidParam.WithDetail(err.Error()))
		return
	}
	if err := h.svc.Revoke(c.Request.Context(), userID, &req); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.Success(c, nil)
}
