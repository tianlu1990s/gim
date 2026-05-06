package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/tianlu1990s/gim/internal/model"
)

type FriendRepo interface {
	Create(ctx context.Context, ownerID, friendID, remark string) error
	CreateTx(ctx context.Context, tx *gorm.DB, ownerID, friendID, remark string) error // 事务版本，用于同意申请时双向写入
	Delete(ctx context.Context, ownerID, friendID string) error
	IsFriend(ctx context.Context, ownerID, friendID string) (bool, error)
	GetFriend(ctx context.Context, ownerID, friendID string) (*model.Friend, error)
	List(ctx context.Context, ownerID string, page, pageSize int) ([]*model.FriendVO, int64, error)
	SetRemark(ctx context.Context, ownerID, friendID, remark string) error
}

type FriendRequestRepo interface {
	Create(ctx context.Context, fromID, toID, message string) (int64, error)
	GetByID(ctx context.Context, id int64) (*model.FriendRequest, error)
	ListIncoming(ctx context.Context, toID string, page, pageSize int) ([]*model.FriendRequestVO, int64, error)
	UpdateStatus(ctx context.Context, id int64, status int) error
	UpdateStatusTx(ctx context.Context, tx *gorm.DB, id int64, status int) error // 事务版本
	HasPendingRequest(ctx context.Context, fromID, toID string) (bool, error)    // 检查是否已有待处理的申请
}

type friendRepo struct {
	db *gorm.DB
}

type friendRequestRepo struct {
	db *gorm.DB
}

func newFriendRepo(db *gorm.DB) FriendRepo {
	return &friendRepo{db: db}
}

func newFriendRequestRepo(db *gorm.DB) FriendRequestRepo {
	return &friendRequestRepo{db: db}
}

func (r *friendRepo) Create(ctx context.Context, ownerID, friendID, remark string) error {
	return r.db.WithContext(ctx).Create(&model.Friend{
		OwnerID:  ownerID,
		FriendID: friendID,
		Remark:   remark,
	}).Error
}

func (r *friendRepo) CreateTx(ctx context.Context, tx *gorm.DB, ownerID, friendID, remark string) error {
	return tx.WithContext(ctx).Create(&model.Friend{
		OwnerID:  ownerID,
		FriendID: friendID,
		Remark:   remark,
	}).Error
}

func (r *friendRepo) Delete(ctx context.Context, ownerID, friendID string) error {
	// 单向删除：仅删除 owner→friend 记录，对方关系保留
	return r.db.WithContext(ctx).Where("owner_id = ? AND friend_id = ?", ownerID, friendID).Delete(&model.Friend{}).Error
}

func (r *friendRepo) IsFriend(ctx context.Context, ownerID, friendID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Friend{}).
		Where("owner_id = ? AND friend_id = ?", ownerID, friendID).Count(&count).Error
	return count > 0, err
}

func (r *friendRepo) GetFriend(ctx context.Context, ownerID, friendID string) (*model.Friend, error) {
	var f model.Friend
	err := r.db.WithContext(ctx).Where("owner_id = ? AND friend_id = ?", ownerID, friendID).First(&f).Error
	return &f, err
}

// List 查询好友列表，LEFT JOIN users 表获取最新昵称和头像。
func (r *friendRepo) List(ctx context.Context, ownerID string, page, pageSize int) ([]*model.FriendVO, int64, error) {
	var total int64
	var friends []*model.FriendVO
	r.db.WithContext(ctx).Model(&model.Friend{}).Where("owner_id = ?", ownerID).Count(&total)
	err := r.db.WithContext(ctx).
		Table("friends").
		Select("friends.friend_id, friends.remark, friends.created_at, users.nickname, users.avatar_url").
		Joins("LEFT JOIN users ON users.user_id = friends.friend_id").
		Where("friends.owner_id = ?", ownerID).
		Order("friends.created_at DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Scan(&friends).Error
	return friends, total, err
}

func (r *friendRepo) SetRemark(ctx context.Context, ownerID, friendID, remark string) error {
	return r.db.WithContext(ctx).Model(&model.Friend{}).
		Where("owner_id = ? AND friend_id = ?", ownerID, friendID).
		Update("remark", remark).Error
}

func (r *friendRequestRepo) Create(ctx context.Context, fromID, toID, message string) (int64, error) {
	req := &model.FriendRequest{
		FromUserID: fromID,
		ToUserID:   toID,
		Message:    message,
	}
	err := r.db.WithContext(ctx).Create(req).Error
	return int64(req.ID), err
}

func (r *friendRequestRepo) GetByID(ctx context.Context, id int64) (*model.FriendRequest, error) {
	var req model.FriendRequest
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&req).Error
	return &req, err
}

// ListIncoming 查询收到的好友申请，JOIN users 获取申请人昵称和头像。
func (r *friendRequestRepo) ListIncoming(ctx context.Context, toID string, page, pageSize int) ([]*model.FriendRequestVO, int64, error) {
	var total int64
	var reqs []*model.FriendRequestVO
	r.db.WithContext(ctx).Model(&model.FriendRequest{}).Where("to_user_id = ?", toID).Count(&total)
	err := r.db.WithContext(ctx).
		Table("friend_requests").
		Select("friend_requests.id, friend_requests.from_user_id, friend_requests.message, friend_requests.status, friend_requests.created_at, users.nickname, users.avatar_url").
		Joins("LEFT JOIN users ON users.user_id = friend_requests.from_user_id").
		Where("friend_requests.to_user_id = ?", toID).
		Order("friend_requests.created_at DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Scan(&reqs).Error
	return reqs, total, err
}

func (r *friendRequestRepo) UpdateStatus(ctx context.Context, id int64, status int) error {
	return r.db.WithContext(ctx).Model(&model.FriendRequest{}).Where("id = ?", id).Update("status", status).Error
}

func (r *friendRequestRepo) UpdateStatusTx(ctx context.Context, tx *gorm.DB, id int64, status int) error {
	return tx.WithContext(ctx).Model(&model.FriendRequest{}).Where("id = ?", id).Update("status", status).Error
}

// HasPendingRequest 检查是否存在待处理的申请，防止重复发送。
func (r *friendRequestRepo) HasPendingRequest(ctx context.Context, fromID, toID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.FriendRequest{}).
		Where("from_user_id = ? AND to_user_id = ? AND status = 0", fromID, toID).Count(&count).Error
	return count > 0, err
}
