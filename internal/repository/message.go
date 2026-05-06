package repository

import (
	"context"
	"strconv"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/tianlu1990s/gim/internal/model"
	"github.com/tianlu1990s/gim/pkg/rediskey"
)

type MessageRepo interface {
	Create(ctx context.Context, msg *model.Message) error
	GetBySeqRange(ctx context.Context, conversationID string, startSeq, endSeq int64, limit int) ([]*model.Message, error)
	GetByClientMsgID(ctx context.Context, clientMsgID string) (*model.Message, error) // 去重查询
	Revoke(ctx context.Context, conversationID, clientMsgID string) error
	GetLastMsg(ctx context.Context, conversationID string) (*model.Message, error) // 会话列表的最后消息预览
	GetUserReadSeq(ctx context.Context, userID, conversationID string) (int64, error) // Redis 读取
	SetUserReadSeq(ctx context.Context, userID, conversationID string, seq int64) error // Redis 写入
	UpdateUserReadSeqDB(ctx context.Context, userID, conversationID string, seq int64) error // DB 持久化
	GetMaxSeq(ctx context.Context, conversationID string) (int64, error) // Redis 读取
	GetMinSeq(ctx context.Context, conversationID string) (int64, error)
	IncrSeq(ctx context.Context, conversationID string) (int64, error) // Redis INCR，原子递增
}

type messageRepo struct {
	db  *gorm.DB
	rdb *redis.Client
}

func newMessageRepo(db *gorm.DB, rdb *redis.Client) MessageRepo {
	return &messageRepo{db: db, rdb: rdb}
}

func (r *messageRepo) Create(ctx context.Context, msg *model.Message) error {
	return r.db.WithContext(ctx).Create(msg).Error
}

// GetBySeqRange 按 Seq 范围查询历史消息，默认排除已撤回的消息。
// startSeq=0 表示不设下限，endSeq=0 表示不设上限。
func (r *messageRepo) GetBySeqRange(ctx context.Context, conversationID string, startSeq, endSeq int64, limit int) ([]*model.Message, error) {
	var msgs []*model.Message
	query := r.db.WithContext(ctx).Where("conversation_id = ? AND is_revoked = 0", conversationID)
	if startSeq > 0 {
		query = query.Where("seq >= ?", startSeq)
	}
	if endSeq > 0 {
		query = query.Where("seq <= ?", endSeq)
	}
	err := query.Order("seq ASC").Limit(limit).Find(&msgs).Error
	return msgs, err
}

// GetByClientMsgID 根据客户端消息 ID 查询，用于去重：同一 clientMsgID 不会写入两次。
func (r *messageRepo) GetByClientMsgID(ctx context.Context, clientMsgID string) (*model.Message, error) {
	var msg model.Message
	err := r.db.WithContext(ctx).Where("client_msg_id = ?", clientMsgID).First(&msg).Error
	return &msg, err
}

func (r *messageRepo) Revoke(ctx context.Context, conversationID, clientMsgID string) error {
	return r.db.WithContext(ctx).Model(&model.Message{}).
		Where("conversation_id = ? AND client_msg_id = ?", conversationID, clientMsgID).
		Update("is_revoked", true).Error
}

// GetLastMsg 获取会话最后一条未撤回消息，用于会话列表预览。
func (r *messageRepo) GetLastMsg(ctx context.Context, conversationID string) (*model.Message, error) {
	var msg model.Message
	err := r.db.WithContext(ctx).
		Where("conversation_id = ? AND is_revoked = 0", conversationID).
		Order("seq DESC").First(&msg).Error
	return &msg, err
}

// GetUserReadSeq 从 Redis 读取用户已读 Seq。Redis 不存在时返回 0。
// 已读位置优先存 Redis（热数据），定期同步到 DB（冷数据）。
func (r *messageRepo) GetUserReadSeq(ctx context.Context, userID, conversationID string) (int64, error) {
	val, err := r.rdb.Get(ctx, rediskey.ReadSeqKey(userID, conversationID)).Result()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(val, 10, 64)
}

func (r *messageRepo) SetUserReadSeq(ctx context.Context, userID, conversationID string, seq int64) error {
	return r.rdb.Set(ctx, rediskey.ReadSeqKey(userID, conversationID), seq, 0).Err()
}

func (r *messageRepo) UpdateUserReadSeqDB(ctx context.Context, userID, conversationID string, seq int64) error {
	// UPSERT：首次标记已读时 INSERT 新行，后续 UPDATE
	ucs := model.UserConversationSeq{
		UserID:         userID,
		ConversationID: conversationID,
		ReadSeq:        seq,
	}
	return r.db.WithContext(ctx).
		Where("user_id = ? AND conversation_id = ?", userID, conversationID).
		Assign(model.UserConversationSeq{ReadSeq: seq}).
		FirstOrCreate(&ucs).Error
}

// GetMaxSeq 从 Redis 获取会话当前最大 Seq。
func (r *messageRepo) GetMaxSeq(ctx context.Context, conversationID string) (int64, error) {
	val, err := r.rdb.Get(ctx, rediskey.SeqConvKey(conversationID)).Result()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(val, 10, 64)
}

func (r *messageRepo) GetMinSeq(ctx context.Context, conversationID string) (int64, error) {
	var minSeq int64
	err := r.db.WithContext(ctx).Model(&model.Message{}).
		Where("conversation_id = ?", conversationID).
		Select("MIN(seq)").Scan(&minSeq).Error
	return minSeq, err
}

// IncrSeq 使用 Redis INCR 原子递增 Seq。选择 Redis 而非 MySQL 自增：
// MySQL 表级/行级锁在高并发下成为瓶颈，Redis INCR 单机可达 10w+ QPS。
func (r *messageRepo) IncrSeq(ctx context.Context, conversationID string) (int64, error) {
	return r.rdb.Incr(ctx, rediskey.SeqConvKey(conversationID)).Result()
}
