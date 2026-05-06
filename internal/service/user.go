package service

import (
	"context"

	"github.com/tianlu1990s/gim/internal/model"
	"github.com/tianlu1990s/gim/internal/repository"
	"github.com/tianlu1990s/gim/pkg/errcode"
)

// UserService 用户资料管理业务接口。
type UserService interface {
	GetProfile(ctx context.Context, userID string) (*model.UserVO, error)
	UpdateProfile(ctx context.Context, userID string, req *model.UpdateProfileReq) (*model.UserVO, error)
	GetOtherProfile(ctx context.Context, currentUserID, targetUserID string) (*model.OtherUserVO, error)
	Search(ctx context.Context, userID string, req *model.SearchReq) (*model.PageResult[*model.SearchUserVO], error)
}

type userService struct {
	userRepo   repository.UserRepo
	friendRepo repository.FriendRepo
}

func newUserService(repos *repository.Repositories) UserService {
	return &userService{
		userRepo:   repos.User,
		friendRepo: repos.Friend,
	}
}

// GetProfile 获取当前用户资料。
func (s *userService) GetProfile(ctx context.Context, userID string) (*model.UserVO, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil || user == nil {
		return nil, errcode.ErrUserNotFound
	}
	return user.ToVO(), nil
}

// UpdateProfile 更新用户资料。nickname/avatarUrl/phone/email 均可更新，
// 手机号和邮箱需检查唯一性（排除自己）。
func (s *userService) UpdateProfile(ctx context.Context, userID string, req *model.UpdateProfileReq) (*model.UserVO, error) {
	updates := make(map[string]any)
	if req.Nickname != "" {
		updates["nickname"] = req.Nickname
	}
	if req.AvatarURL != "" {
		updates["avatar_url"] = req.AvatarURL
	}
	if req.Phone != "" {
		exists, err := s.userRepo.ExistsByPhone(ctx, req.Phone, userID)
		if err != nil {
			return nil, errcode.ErrInternal.WithDetail(err.Error())
		}
		if exists {
			return nil, errcode.ErrInvalidParam.WithDetail("手机号已被其他用户使用")
		}
		updates["phone"] = req.Phone
	}
	if req.Email != "" {
		exists, err := s.userRepo.ExistsByEmail(ctx, req.Email, userID)
		if err != nil {
			return nil, errcode.ErrInternal.WithDetail(err.Error())
		}
		if exists {
			return nil, errcode.ErrInvalidParam.WithDetail("邮箱已被其他用户使用")
		}
		updates["email"] = req.Email
	}
	if len(updates) == 0 {
		return s.GetProfile(ctx, userID)
	}
	if err := s.userRepo.Update(ctx, userID, updates); err != nil {
		return nil, errcode.ErrInternal.WithDetail(err.Error())
	}
	return s.GetProfile(ctx, userID)
}

// GetOtherProfile 查看他人资料。额外返回好友关系和备注（仅好友可见）。
// 非好友可以看到基本信息（userId、nickname、avatarUrl），但看到不到备注。
func (s *userService) GetOtherProfile(ctx context.Context, currentUserID, targetUserID string) (*model.OtherUserVO, error) {
	user, err := s.userRepo.GetByID(ctx, targetUserID)
	if err != nil || user == nil {
		return nil, errcode.ErrUserNotFound
	}
	vo := &model.OtherUserVO{
		UserID:    user.UserID,
		Nickname:  user.Nickname,
		AvatarURL: user.AvatarURL,
	}
	// 检查好友关系，若为好友则附带备注
	isFriend, _ := s.friendRepo.IsFriend(ctx, currentUserID, targetUserID)
	if isFriend {
		friend, err := s.friendRepo.GetFriend(ctx, currentUserID, targetUserID)
		if err == nil {
			vo.IsFriend = true
			vo.Remark = friend.Remark
		}
	}
	return vo, nil
}

// Search 搜索用户。纯数字→手机号精确匹配，否则→昵称模糊搜索。
func (s *userService) Search(ctx context.Context, userID string, req *model.SearchReq) (*model.PageResult[*model.SearchUserVO], error) {
	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	users, total, err := s.userRepo.Search(ctx, req.Keyword, page, pageSize)
	if err != nil {
		return nil, errcode.ErrInternal.WithDetail(err.Error())
	}
	list := make([]*model.SearchUserVO, 0, len(users))
	for _, u := range users {
		list = append(list, &model.SearchUserVO{
			UserID:    u.UserID,
			Nickname:  u.Nickname,
			AvatarURL: u.AvatarURL,
		})
	}
	return &model.PageResult[*model.SearchUserVO]{
		List:     list,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}
