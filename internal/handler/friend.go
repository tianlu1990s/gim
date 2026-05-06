package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/tianlu1990s/gim/internal/model"
	"github.com/tianlu1990s/gim/internal/service"
	"github.com/tianlu1990s/gim/pkg/errcode"
	"github.com/tianlu1990s/gim/pkg/resp"
)

// FriendHandler 好友关系相关 HTTP Handler。
type FriendHandler struct {
	svc service.FriendService
}

func NewFriendHandler(svc service.FriendService) *FriendHandler {
	return &FriendHandler{svc: svc}
}

// SendRequest 发送好友申请。
// POST /api/v1/friend/request
func (h *FriendHandler) SendRequest(c *gin.Context) {
	userID := c.GetString("userID")
	var req model.SendFriendRequestReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, errcode.ErrInvalidParam.WithDetail(err.Error()))
		return
	}
	id, err := h.svc.SendRequest(c.Request.Context(), userID, &req)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.Success(c, gin.H{"id": id})
}

// ListRequests 查询收到的好友申请列表。
// GET /api/v1/friend/request/incoming
func (h *FriendHandler) ListRequests(c *gin.Context) {
	userID := c.GetString("userID")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	list, total, err := h.svc.ListRequests(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.Success(c, gin.H{"list": list, "total": total, "page": page, "pageSize": pageSize})
}

// AcceptRequest 同意好友申请。
// POST /api/v1/friend/request/:id/accept
func (h *FriendHandler) AcceptRequest(c *gin.Context) {
	userID := c.GetString("userID")
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		resp.Fail(c, errcode.ErrInvalidParam.WithDetail("无效的申请ID"))
		return
	}
	if err := h.svc.AcceptRequest(c.Request.Context(), userID, id); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.Success(c, nil)
}

// RejectRequest 拒绝好友申请。
// POST /api/v1/friend/request/:id/reject
func (h *FriendHandler) RejectRequest(c *gin.Context) {
	userID := c.GetString("userID")
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		resp.Fail(c, errcode.ErrInvalidParam.WithDetail("无效的申请ID"))
		return
	}
	if err := h.svc.RejectRequest(c.Request.Context(), userID, id); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.Success(c, nil)
}

// Delete 删除好友。
// DELETE /api/v1/friend/:userId
func (h *FriendHandler) Delete(c *gin.Context) {
	userID := c.GetString("userID")
	friendID := c.Param("userId")
	if friendID == "" {
		resp.Fail(c, errcode.ErrInvalidParam.WithDetail("userId 不能为空"))
		return
	}
	if err := h.svc.Delete(c.Request.Context(), userID, friendID); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.Success(c, nil)
}

// List 获取好友列表。
// GET /api/v1/friend/list
func (h *FriendHandler) List(c *gin.Context) {
	userID := c.GetString("userID")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	list, total, err := h.svc.List(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.Success(c, gin.H{"list": list, "total": total, "page": page, "pageSize": pageSize})
}

// SetRemark 设置好友备注。
// PUT /api/v1/friend/:userId/remark
func (h *FriendHandler) SetRemark(c *gin.Context) {
	userID := c.GetString("userID")
	friendID := c.Param("userId")
	if friendID == "" {
		resp.Fail(c, errcode.ErrInvalidParam.WithDetail("userId 不能为空"))
		return
	}
	var req model.SetRemarkReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, errcode.ErrInvalidParam.WithDetail(err.Error()))
		return
	}
	if err := h.svc.SetRemark(c.Request.Context(), userID, friendID, req.Remark); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.Success(c, nil)
}
