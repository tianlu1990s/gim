package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/tianlu1990s/gim/internal/model"
)

type ConversationRepo interface {
	// CreateIfNotExistTx 事务中创建会话（如不存在），用于好友同意时自动创建双方会话
	CreateIfNotExistTx(ctx context.Context, tx *gorm.DB, ownerID, convID string, convType int, targetID string) error
	List(ctx context.Context, ownerID string, page, pageSize int) ([]*model.ConversationVO, int64, error)
	UpdatePin(ctx context.Context, ownerID, convID string, isPinned bool) error
	Delete(ctx context.Context, ownerID, convID string) error // 软删除：仅删除该用户的会话视图，不删除消息
	UpdateMaxSeq(ctx context.Context, convID string, seq int64) error
	GetByID(ctx context.Context, ownerID, convID string) (*model.Conversation, error)
	ListByOwner(ctx context.Context, ownerID string) ([]*model.Conversation, error)
}

type conversationRepo struct {
	db *gorm.DB
}

func newConversationRepo(db *gorm.DB) ConversationRepo {
	return &conversationRepo{db: db}
}

func (r *conversationRepo) CreateIfNotExistTx(ctx context.Context, tx *gorm.DB, ownerID, convID string, convType int, targetID string) error {
	// 先检查是否存在，避免重复创建（INSERT IF NOT EXISTS 的 Go 实现）
	var count int64
	tx.WithContext(ctx).Model(&model.Conversation{}).
		Where("owner_id = ? AND conversation_id = ?", ownerID, convID).Count(&count)
	if count > 0 {
		return nil
	}
	return tx.WithContext(ctx).Create(&model.Conversation{
		OwnerID:        ownerID,
		ConversationID: convID,
		ConvType:       convType,
		TargetID:       targetID,
	}).Error
}

// List 查询会话列表，LEFT JOIN users 获取对方的昵称和头像。
func (r *conversationRepo) List(ctx context.Context, ownerID string, page, pageSize int) ([]*model.ConversationVO, int64, error) {
	var total int64
	var convos []*model.ConversationVO
	r.db.WithContext(ctx).Model(&model.Conversation{}).Where("owner_id = ?", ownerID).Count(&total)
	err := r.db.WithContext(ctx).
		Table("conversations c").
		Select("c.conversation_id, c.conv_type, c.target_id, c.max_seq, c.is_pinned, c.updated_at, u.nickname AS target_name, u.avatar_url AS target_avatar").
		Joins("LEFT JOIN users u ON u.user_id = c.target_id").
		Where("c.owner_id = ?", ownerID).
			Order("c.is_pinned DESC, c.updated_at DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Scan(&convos).Error
	return convos, total, err
}

func (r *conversationRepo) UpdatePin(ctx context.Context, ownerID, convID string, isPinned bool) error {
	return r.db.WithContext(ctx).Model(&model.Conversation{}).
		Where("owner_id = ? AND conversation_id = ?", ownerID, convID).
		Update("is_pinned", isPinned).Error
}

// Delete 软删除：只删除该用户的会话记录，不影响对方和消息数据。
func (r *conversationRepo) Delete(ctx context.Context, ownerID, convID string) error {
	return r.db.WithContext(ctx).Where("owner_id = ? AND conversation_id = ?", ownerID, convID).Delete(&model.Conversation{}).Error
}

// UpdateMaxSeq 消息发送后更新会话的 maxSeq，用于未读数计算。
func (r *conversationRepo) UpdateMaxSeq(ctx context.Context, convID string, seq int64) error {
	return r.db.WithContext(ctx).Model(&model.Conversation{}).
		Where("conversation_id = ?", convID).
		Update("max_seq", seq).Error
}

func (r *conversationRepo) GetByID(ctx context.Context, ownerID, convID string) (*model.Conversation, error) {
	var conv model.Conversation
	err := r.db.WithContext(ctx).Where("owner_id = ? AND conversation_id = ?", ownerID, convID).First(&conv).Error
	return &conv, err
}

func (r *conversationRepo) ListByOwner(ctx context.Context, ownerID string) ([]*model.Conversation, error) {
	var convs []*model.Conversation
	err := r.db.WithContext(ctx).Where("owner_id = ?", ownerID).Find(&convs).Error
	return convs, err
}
