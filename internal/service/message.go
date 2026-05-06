package service

import (
	"context"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/tianlu1990s/gim/internal/model"
	"github.com/tianlu1990s/gim/internal/repository"
	"github.com/tianlu1990s/gim/internal/ws"
	"github.com/tianlu1990s/gim/pkg/convutil"
	"github.com/tianlu1990s/gim/pkg/errcode"
	"github.com/tianlu1990s/gim/pkg/rediskey"
	"github.com/tianlu1990s/gim/pkg/snowflake"
)

// HistoryResp 消息历史查询响应。
type HistoryResp struct {
	List    []*model.Message `json:"list"`
	HasMore bool             `json:"hasMore"`
	MinSeq  int64            `json:"minSeq"`
	MaxSeq  int64            `json:"maxSeq"`
}

// MessageService 消息收发业务接口。
// 消息模块是 IM 的核心——发送去重、Seq 分配、持久化、推送四个环节缺一不可。
type MessageService interface {
	SendMessage(ctx context.Context, senderID string, req *model.SendMsgReq) (*model.SendMsgResp, error)
	History(ctx context.Context, userID string, req *model.HistoryReq) (*HistoryResp, error)
	MarkRead(ctx context.Context, userID string, req *model.MarkReadReq) error
	Revoke(ctx context.Context, userID string, req *model.RevokeMsgReq) error
}

type messageService struct {
	msgRepo    repository.MessageRepo
	convRepo   repository.ConversationRepo
	friendRepo repository.FriendRepo
	hub        *ws.Hub
	rdb        *redis.Client
}

func newMessageService(repos *repository.Repositories, hub *ws.Hub, rdb *redis.Client) MessageService {
	return &messageService{
		msgRepo:    repos.Message,
		convRepo:   repos.Conversation,
		friendRepo: repos.Friend,
		hub:        hub,
		rdb:        rdb,
	}
}

// SendMessage 发送消息——IM 最核心流程。
// 执行顺序：去重（SETNX）→ 好友校验 → Redis INCR 分配 Seq → 持久化 MySQL → 更新会话 maxSeq → WS 推送。
// 这个顺序是设计决策：先去重防止重复写入，先持久化再推送防止接收方看到"幽灵消息"。
func (s *messageService) SendMessage(ctx context.Context, senderID string, req *model.SendMsgReq) (*model.SendMsgResp, error) {
	convID := req.ConversationID

	// 1. 去重：SETNX 原子操作，防止网络重试重复写入同一条消息
	dedupKey := rediskey.DedupMsgKey(req.ClientMsgID)
	ok, err := s.rdb.SetNX(ctx, dedupKey, "1", 5*time.Minute).Result()
	if err != nil {
		return nil, errcode.ErrInternal.WithDetail("去重检查失败")
	}
	if !ok {
		// 消息已存在，返回已有消息的 seq（幂等性保证）
		existing, err := s.msgRepo.GetByClientMsgID(ctx, req.ClientMsgID)
		if err == nil && existing != nil {
			return &model.SendMsgResp{Seq: existing.Seq, ServerMsgID: existing.ServerMsgID, SendTime: existing.CreatedAt.UnixMilli()}, nil
		}
	}

	// 2. 单聊好友校验：必须先成为好友才能发消息
	if strings.HasPrefix(convID, "single_") {
		targetID := convutil.ExtractTargetID(convID, senderID)
		isFriend, _ := s.friendRepo.IsFriend(ctx, senderID, targetID)
		if !isFriend {
			return nil, errcode.ErrNotFriend
		}
	}

	// 3. Redis INCR 分配会话内递增 Seq，选择 Redis 而非 MySQL AUTO_INCREMENT：
	//    MySQL 自增在高并发下受表级锁限制，Redis INCR 单机可达 10w+ QPS
	seq, err := s.msgRepo.IncrSeq(ctx, convID)
	if err != nil {
		return nil, errcode.ErrInternal.WithDetail("分配 Seq 失败")
	}

	// 4. 生成服务端消息 ID（Snowflake 全局唯一）
	serverMsgID := snowflake.Generate().String()

	// 5. 持久化消息到 MySQL
	msg := &model.Message{
		ConversationID: convID,
		Seq:            seq,
		SenderID:       senderID,
		MsgType:        req.ContentType,
		Content:        req.Content,
		ClientMsgID:    req.ClientMsgID,
		ServerMsgID:    serverMsgID,
	}
	if err := s.msgRepo.Create(ctx, msg); err != nil {
		return nil, errcode.ErrInternal.WithDetail(err.Error())
	}

	// 6. 更新会话 maxSeq（未读数 = maxSeq - readSeq）
	s.convRepo.UpdateMaxSeq(ctx, convID, seq)

	// 7. WS 推送消息通知给接收方
	now := time.Now().UnixMilli()
	pushMsg := &ws.WSMessage{
		Type: 101,
		Data: map[string]any{
			"conversationId": convID,
			"seq":            seq,
			"senderId":       senderID,
			"contentType":    req.ContentType,
			"content":        req.Content,
			"serverMsgId":    serverMsgID,
			"clientMsgId":    req.ClientMsgID,
			"sendTime":       now,
		},
	}
	targetIDs := convutil.GetConversationMembers(convID, senderID)
	for _, targetID := range targetIDs {
		s.hub.PushToUser(targetID, pushMsg)
	}

	return &model.SendMsgResp{
		Seq:         seq,
		ServerMsgID: serverMsgID,
		SendTime:    now,
	}, nil
}

// History 查询历史消息。从 maxSeq 往前拉取，返回倒序列表（最新在前）。
// 前端上滑加载更多时传入当前列表最小的 seq 作为 startSeq。
func (s *messageService) History(ctx context.Context, userID string, req *model.HistoryReq) (*HistoryResp, error) {
	convID := req.ConversationID

	maxSeq, _ := s.msgRepo.GetMaxSeq(ctx, convID)
	if maxSeq == 0 {
		return &HistoryResp{List: []*model.Message{}, HasMore: false, MaxSeq: 0, MinSeq: 0}, nil
	}

	// startSeq=0 表示从最新开始拉取
	startSeq := req.StartSeq
	if startSeq == 0 {
		startSeq = maxSeq
	}

	// 计算查询范围：往前推 count 条，确保能拉到足够的消息
	rangeStart := startSeq - int64(req.Count)
	if rangeStart < 0 {
		rangeStart = 0
	}

	msgs, err := s.msgRepo.GetBySeqRange(ctx, convID, rangeStart, startSeq, req.Count+1)
	if err != nil {
		return nil, errcode.ErrInternal.WithDetail(err.Error())
	}

	// hasMore：如果返回了超过 count 条，说明还有更早的消息
	hasMore := len(msgs) > req.Count
	if hasMore {
		msgs = msgs[:req.Count]
	}

	// 倒序排列（最新在前），符合前端展示习惯
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}

	minSeq, _ := s.msgRepo.GetMinSeq(ctx, convID)

	return &HistoryResp{
		List:    msgs,
		HasMore: hasMore,
		MinSeq:  minSeq,
		MaxSeq:  maxSeq,
	}, nil
}

// MarkRead 标记已读。更新 Redis 中的已读位置（热数据），同时写 MySQL 持久化。
// 更新后推送已读回执给对端，让对方知道消息已被读取。
func (s *messageService) MarkRead(ctx context.Context, userID string, req *model.MarkReadReq) error {
	convID := req.ConversationID

	// 校验 readSeq 不能超过 maxSeq
	maxSeq, _ := s.msgRepo.GetMaxSeq(ctx, convID)
	if req.ReadSeq > maxSeq {
		return errcode.ErrInvalidParam.WithDetail("readSeq 超过 maxSeq")
	}

	// 更新 Redis（快速读取）和 MySQL（持久化）
	if err := s.msgRepo.SetUserReadSeq(ctx, userID, convID, req.ReadSeq); err != nil {
		return errcode.ErrInternal.WithDetail(err.Error())
	}
	if err := s.msgRepo.UpdateUserReadSeqDB(ctx, userID, convID, req.ReadSeq); err != nil {
		return errcode.ErrInternal.WithDetail(err.Error())
	}

	// WS 推送已读回执给对端
	targetIDs := convutil.GetConversationMembers(convID, userID)
	for _, targetID := range targetIDs {
		s.hub.PushToUser(targetID, &ws.WSMessage{
			Type: 102,
			Data: map[string]any{
				"conversationId": convID,
				"readUserId":     userID,
				"readSeq":        req.ReadSeq,
			},
		})
	}
	return nil
}

// Revoke 撤回消息。仅允许撤回自己发送的消息，标记 is_revoked=true（软删除）。
// 撤回后通过 WS 推送通知给所有会话成员。
func (s *messageService) Revoke(ctx context.Context, userID string, req *model.RevokeMsgReq) error {
	convID := req.ConversationID

	// 查询消息，验证发送者是本人
	msg, err := s.msgRepo.GetByClientMsgID(ctx, req.ClientMsgID)
	if err != nil {
		return errcode.ErrMsgNotFound
	}
	if msg.SenderID != userID {
		return errcode.ErrForbidden.WithDetail("只能撤回自己发送的消息")
	}
	if msg.IsRevoked {
		return nil // 已撤回，幂等
	}

	if err := s.msgRepo.Revoke(ctx, convID, req.ClientMsgID); err != nil {
		return errcode.ErrInternal.WithDetail(err.Error())
	}

	// WS 推送撤回通知给所有会话成员
	targetIDs := convutil.GetConversationMembers(convID, userID)
	for _, targetID := range targetIDs {
		s.hub.PushToUser(targetID, &ws.WSMessage{
			Type: 103,
			Data: map[string]any{
				"conversationId": convID,
				"seq":            msg.Seq,
				"clientMsgId":    req.ClientMsgID,
			},
		})
	}

	return nil
}
