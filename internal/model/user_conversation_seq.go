package model

import "time"

// UserConversationSeq 记录每个用户在每个会话中的已读位置（ReadSeq）。
// 未读数 = 会话 MaxSeq - 用户 ReadSeq。联合唯一索引保证每个用户-会话只有一条记录。
type UserConversationSeq struct {
	ID             uint64    `gorm:"primaryKey;autoIncrement" json:"-"`
	UserID         string    `gorm:"uniqueIndex:idx_user_conv;type:varchar(64);not null" json:"userId"`
	ConversationID string    `gorm:"uniqueIndex:idx_user_conv;type:varchar(128);not null" json:"conversationId"`
	ReadSeq        int64     `gorm:"type:bigint;not null;default:0" json:"readSeq"` // 已读到的最大 Seq
	UpdatedAt      time.Time `gorm:"not null;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" json:"updatedAt"`
}

func (UserConversationSeq) TableName() string { return "user_conversation_seqs" }
