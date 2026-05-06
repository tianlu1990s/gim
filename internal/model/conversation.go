package model

import "time"

// Conversation 会话表。每个用户-会话是一行（同一单聊会话，Alice 和 Bob 各有自己的一条记录）。
// ConvType: 1=单聊 2=群聊（Phase 2）。
type Conversation struct {
	ID             uint64    `gorm:"primaryKey;autoIncrement" json:"-"`
	OwnerID        string    `gorm:"index:idx_owner_conv;type:varchar(64);not null" json:"ownerId"`
	ConversationID string    `gorm:"index:idx_owner_conv;type:varchar(128);not null" json:"conversationId"`
	ConvType       int       `gorm:"type:int;not null" json:"convType"` // 1=单聊 2=群聊
	TargetID       string    `gorm:"type:varchar(64);not null;default:''" json:"targetId"` // 对方的 userId 或群 ID
	MaxSeq         int64     `gorm:"type:bigint;not null;default:0" json:"maxSeq"`         // 会话最大消息 Seq
	IsPinned       bool      `gorm:"type:tinyint(1);not null;default:0" json:"isPinned"`   // 是否置顶
	CreatedAt      time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"createdAt"`
	UpdatedAt      time.Time `gorm:"not null;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" json:"updatedAt"`
}

func (Conversation) TableName() string { return "conversations" }
