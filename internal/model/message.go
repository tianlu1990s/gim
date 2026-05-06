package model

import "time"

// Message 消息表。ClientMsgID 有唯一索引，用于去重（网络重试不会重复写入）。
// Seq 与会话 ID 组成联合索引，历史消息按 Seq 范围查询。
type Message struct {
	ID             uint64    `gorm:"primaryKey;autoIncrement" json:"-"`
	ConversationID string    `gorm:"index:idx_conv_seq;type:varchar(128);not null" json:"conversationId"`
	Seq            int64     `gorm:"index:idx_conv_seq;type:bigint;not null" json:"seq"` // 会话内递增序号
	SenderID       string    `gorm:"type:varchar(64);not null" json:"senderId"`
	MsgType        int       `gorm:"type:int;not null" json:"msgType"` // 1=文本 2=图片 3=文件 4=系统消息
	Content        string    `gorm:"type:text;not null" json:"content"`
	ClientMsgID    string    `gorm:"uniqueIndex;type:varchar(64);not null" json:"clientMsgId"` // 客户端生成，全局唯一，去重用
	ServerMsgID    string    `gorm:"index;type:varchar(64);not null" json:"serverMsgId"`       // 服务端生成
	IsRevoked      bool      `gorm:"type:tinyint(1);not null;default:0" json:"isRevoked"`     // 是否已撤回
	CreatedAt      time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"createdAt"`
}

func (Message) TableName() string { return "messages" }
