package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/tianlu1990s/gim/internal/model"
	"github.com/tianlu1990s/gim/pkg/errcode"
)

type UserRepo interface {
	Create(ctx context.Context, user *model.User) error
	GetByID(ctx context.Context, userID string) (*model.User, error)
	ExistsByID(ctx context.Context, userID string) (bool, error)
	Update(ctx context.Context, userID string, updates map[string]any) error
	Search(ctx context.Context, keyword string, page, pageSize int) ([]*model.User, int64, error)
	ExistsByPhone(ctx context.Context, phone, excludeUserID string) (bool, error) // excludeUserID 用于编辑时排除自己
	ExistsByEmail(ctx context.Context, email, excludeUserID string) (bool, error)
}

type userRepo struct {
	db *gorm.DB
}

func newUserRepo(db *gorm.DB) UserRepo {
	return &userRepo{db: db}
}

func (r *userRepo) Create(ctx context.Context, user *model.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *userRepo) GetByID(ctx context.Context, userID string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errcode.ErrUserNotFound
	}
	return &user, err
}

func (r *userRepo) ExistsByID(ctx context.Context, userID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.User{}).Where("user_id = ?", userID).Count(&count).Error
	return count > 0, err
}

func (r *userRepo) Update(ctx context.Context, userID string, updates map[string]any) error {
	return r.db.WithContext(ctx).Model(&model.User{}).Where("user_id = ?", userID).Updates(updates).Error
}

// Search 根据关键词搜索用户。纯数字→手机号精确匹配，否则→昵称模糊匹配。
func (r *userRepo) Search(ctx context.Context, keyword string, page, pageSize int) ([]*model.User, int64, error) {
	var users []*model.User
	var total int64
	query := r.db.WithContext(ctx).Model(&model.User{}).Where("status = 1")
	if isDigit(keyword) {
		query = query.Where("phone = ?", keyword)
	} else {
		query = query.Where("nickname LIKE ?", "%"+keyword+"%")
	}
	query.Count(&total)
	err := query.Offset((page - 1) * pageSize).Limit(pageSize).Find(&users).Error
	return users, total, err
}

func (r *userRepo) ExistsByPhone(ctx context.Context, phone, excludeUserID string) (bool, error) {
	var count int64
	q := r.db.WithContext(ctx).Model(&model.User{}).Where("phone = ?", phone)
	if excludeUserID != "" {
		q = q.Where("user_id != ?", excludeUserID) // 编辑时跳过自己
	}
	err := q.Count(&count).Error
	return count > 0, err
}

func (r *userRepo) ExistsByEmail(ctx context.Context, email, excludeUserID string) (bool, error) {
	var count int64
	q := r.db.WithContext(ctx).Model(&model.User{}).Where("email = ?", email)
	if excludeUserID != "" {
		q = q.Where("user_id != ?", excludeUserID)
	}
	err := q.Count(&count).Error
	return count > 0, err
}

func isDigit(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return len(s) > 0
}
