package service

import (
	"context"

	"github.com/tianlu1990s/gim/internal/model"
	"github.com/tianlu1990s/gim/internal/repository"
	"github.com/tianlu1990s/gim/pkg/errcode"
)

// ConversationService 会话管理业务接口。
type ConversationService interface {
	List(ctx context.Context, userID string, page, pageSize int) (*model.PageResult[*model.ConversationVO], error)
	Pin(ctx context.Context, userID, convID string, isPinned bool) error
	Delete(ctx context.Context, userID, convID string) error
}

type conversationService struct {
	convRepo repository.ConversationRepo
	msgRepo  repository.MessageRepo
}

func newConversationService(repos *repository.Repositories) ConversationService {
	return &conversationService{
		convRepo: repos.Conversation,
		msgRepo:  repos.Message,
	}
}

// List 获取会话列表。每个会话填充未读数（maxSeq - readSeq）和最后消息预览。
// 排序规则：置顶优先，其次按更新时间倒序。
func (s *conversationService) List(ctx context.Context, userID string, page, pageSize int) (*model.PageResult[*model.ConversationVO], error) {
	convs, total, err := s.convRepo.List(ctx, userID, page, pageSize)
	if err != nil {
		return nil, errcode.ErrInternal.WithDetail(err.Error())
	}

	for _, conv := range convs {
		// 计算未读数 = maxSeq - readSeq
		readSeq, _ := s.msgRepo.GetUserReadSeq(ctx, userID, conv.ConversationID)
		conv.UnreadCount = conv.MaxSeq - readSeq
		if conv.UnreadCount < 0 {
			conv.UnreadCount = 0
		}
		conv.ReadSeq = readSeq

		// 获取最后一条消息预览
		lastMsg, err := s.msgRepo.GetLastMsg(ctx, conv.ConversationID)
		if err == nil && lastMsg != nil {
			conv.LastMsg = lastMsg
			conv.LastMsgContent = lastMsg.Content
			conv.LastMsgTime = lastMsg.CreatedAt.Format("2006-01-02 15:04:05")
		}
	}

	return &model.PageResult[*model.ConversationVO]{
		List:     convs,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// Pin 置顶/取消置顶会话。
func (s *conversationService) Pin(ctx context.Context, userID, convID string, isPinned bool) error {
	return s.convRepo.UpdatePin(ctx, userID, convID, isPinned)
}

// Delete 删除会话。仅删除当前用户的会话视图，消息数据保留，对方会话不受影响。
func (s *conversationService) Delete(ctx context.Context, userID, convID string) error {
	// 检查会话是否存在
	_, err := s.convRepo.GetByID(ctx, userID, convID)
	if err != nil {
		return errcode.ErrConversationNotFound
	}
	return s.convRepo.Delete(ctx, userID, convID)
}
