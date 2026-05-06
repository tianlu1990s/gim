package service

import (
	"context"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/tianlu1990s/gim/internal/model"
	"github.com/tianlu1990s/gim/internal/repository"
	"github.com/tianlu1990s/gim/internal/ws"
	"github.com/tianlu1990s/gim/pkg/convid"
	"github.com/tianlu1990s/gim/pkg/errcode"
)

// FriendService 好友关系管理业务接口。
type FriendService interface {
	SendRequest(ctx context.Context, userID string, req *model.SendFriendRequestReq) (int64, error)
	ListRequests(ctx context.Context, userID string, page, pageSize int) ([]*model.FriendRequestVO, int64, error)
	AcceptRequest(ctx context.Context, userID string, requestID int64) error
	RejectRequest(ctx context.Context, userID string, requestID int64) error
	Delete(ctx context.Context, ownerID, friendID string) error
	List(ctx context.Context, ownerID string, page, pageSize int) ([]*model.FriendVO, int64, error)
	SetRemark(ctx context.Context, ownerID, friendID, remark string) error
}

type friendService struct {
	friendRepo    repository.FriendRepo
	friendReqRepo repository.FriendRequestRepo
	convRepo      repository.ConversationRepo
	userRepo      repository.UserRepo
	repos         *repository.Repositories
	hub           *ws.Hub
	rdb           *redis.Client
}

func newFriendService(repos *repository.Repositories, hub *ws.Hub, rdb *redis.Client) FriendService {
	return &friendService{
		friendRepo:    repos.Friend,
		friendReqRepo: repos.FriendReq,
		convRepo:      repos.Conversation,
		userRepo:      repos.User,
		repos:         repos,
		hub:           hub,
		rdb:           rdb,
	}
}

// SendRequest 发送好友申请。校验不能添加自己、目标存在、不是已有好友、没有待处理申请。
func (s *friendService) SendRequest(ctx context.Context, userID string, req *model.SendFriendRequestReq) (int64, error) {
	if userID == req.ToUserID {
		return 0, errcode.ErrCannotFriendSelf
	}
	// 检查目标用户是否存在且未被禁用
	targetUser, err := s.userRepo.GetByID(ctx, req.ToUserID)
	if err != nil || targetUser == nil {
		return 0, errcode.ErrUserNotFound
	}
	if targetUser.Status != 1 {
		return 0, errcode.ErrUserDisabled
	}
	// 检查是否已是好友
	isFriend, _ := s.friendRepo.IsFriend(ctx, userID, req.ToUserID)
	if isFriend {
		return 0, errcode.ErrAlreadyFriend
	}
	// 检查是否已有待处理申请（双向都要检查，防止双方互相发申请）
	hasPending, _ := s.friendReqRepo.HasPendingRequest(ctx, userID, req.ToUserID)
	if hasPending {
		return 0, errcode.ErrFriendRequestExists
	}
	hasPendingReverse, _ := s.friendReqRepo.HasPendingRequest(ctx, req.ToUserID, userID)
	if hasPendingReverse {
		return 0, errcode.ErrFriendRequestExists
	}

	id, err := s.friendReqRepo.Create(ctx, userID, req.ToUserID, req.Message)
	if err != nil {
		return 0, errcode.ErrInternal.WithDetail(err.Error())
	}

	// WS 推送通知目标用户（新的好友申请）
	s.hub.PushToUser(req.ToUserID, &ws.WSMessage{
		Type: 105,
		Data: map[string]any{"type": "friend_request", "userId": userID},
	})

	return id, nil
}

// ListRequests 查询收到的好友申请列表，按创建时间倒序。
func (s *friendService) ListRequests(ctx context.Context, userID string, page, pageSize int) ([]*model.FriendRequestVO, int64, error) {
	return s.friendReqRepo.ListIncoming(ctx, userID, page, pageSize)
}

// AcceptRequest 同意好友申请。事务中完成：更新申请状态 + 双向好友关系 + 双方会话创建。
// 三个操作必须在同一事务中——任一失败则全部回滚，保证数据一致性。
func (s *friendService) AcceptRequest(ctx context.Context, userID string, requestID int64) error {
	freq, err := s.friendReqRepo.GetByID(ctx, requestID)
	if err != nil {
		return errcode.ErrResourceNotFound
	}
	// 安全校验：只有申请的接收者才能同意
	if freq.ToUserID != userID {
		return errcode.ErrForbidden
	}
	if freq.Status != 0 {
		return errcode.ErrAlreadyProcessed
	}

	err = s.repos.Transaction(ctx, func(tx *gorm.DB) error {
		// 1. 更新申请状态为已同意
		if err := s.friendReqRepo.UpdateStatusTx(ctx, tx, requestID, 1); err != nil {
			return err
		}
		// 2. 双向写入好友关系（Alice→Bob 和 Bob→Alice 各一条记录）
		if err := s.friendRepo.CreateTx(ctx, tx, freq.FromUserID, freq.ToUserID, ""); err != nil {
			return err
		}
		if err := s.friendRepo.CreateTx(ctx, tx, freq.ToUserID, freq.FromUserID, ""); err != nil {
			return err
		}
		// 3. 为双方各创建一条会话记录（同一单聊 session，各自管理自己的置顶/删除/未读）
		convID := convid.GenSingleConvID(freq.FromUserID, freq.ToUserID)
		if err := s.convRepo.CreateIfNotExistTx(ctx, tx, freq.FromUserID, convID, 1, freq.ToUserID); err != nil {
			return err
		}
		if err := s.convRepo.CreateIfNotExistTx(ctx, tx, freq.ToUserID, convID, 1, freq.FromUserID); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return errcode.ErrInternal.WithDetail(err.Error())
	}

	// WS 通知申请方（好友已同意）
	s.hub.PushToUser(freq.FromUserID, &ws.WSMessage{
		Type: 107,
		Data: map[string]any{"type": "friend_accepted", "userId": userID},
	})

	return nil
}

// RejectRequest 拒绝好友申请。
func (s *friendService) RejectRequest(ctx context.Context, userID string, requestID int64) error {
	freq, err := s.friendReqRepo.GetByID(ctx, requestID)
	if err != nil {
		return errcode.ErrResourceNotFound
	}
	if freq.ToUserID != userID {
		return errcode.ErrForbidden
	}
	if freq.Status != 0 {
		return errcode.ErrAlreadyProcessed
	}
	if err := s.friendReqRepo.UpdateStatus(ctx, requestID, 2); err != nil {
		return errcode.ErrInternal.WithDetail(err.Error())
	}
	return nil
}

// Delete 删除好友。仅删除当前用户的好友记录（单向），对方的记录保留。
func (s *friendService) Delete(ctx context.Context, ownerID, friendID string) error {
	isFriend, _ := s.friendRepo.IsFriend(ctx, ownerID, friendID)
	if !isFriend {
		return errcode.ErrNotFriend
	}
	return s.friendRepo.Delete(ctx, ownerID, friendID)
}

// List 获取好友列表。
func (s *friendService) List(ctx context.Context, ownerID string, page, pageSize int) ([]*model.FriendVO, int64, error) {
	return s.friendRepo.List(ctx, ownerID, page, pageSize)
}

// SetRemark 设置好友备注。
func (s *friendService) SetRemark(ctx context.Context, ownerID, friendID, remark string) error {
	isFriend, _ := s.friendRepo.IsFriend(ctx, ownerID, friendID)
	if !isFriend {
		return errcode.ErrNotFriend
	}
	return s.friendRepo.SetRemark(ctx, ownerID, friendID, remark)
}
