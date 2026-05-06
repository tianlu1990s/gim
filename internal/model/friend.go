package model

import "time"

// Friend 好友关系表。双向存储：Alice 加 Bob → 两条记录（owner=Alice+friend=Bob，owner=Bob+friend=Alice）。
// 双向存储的好处：每个人查自己的好友列表只需 WHERE owner_id = 自己，简单高效。
type Friend struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"-"`
	OwnerID   string    `gorm:"index:idx_owner;type:varchar(64);not null" json:"ownerId"`
	FriendID  string    `gorm:"index:idx_owner;type:varchar(64);not null" json:"friendId"`
	Remark    string    `gorm:"type:varchar(64);not null;default:''" json:"remark"` // 备注名
	CreatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"createdAt"`
}

func (Friend) TableName() string { return "friends" }

// FriendRequest 好友申请表。Status: 0=待处理 1=已同意 2=已拒绝。
type FriendRequest struct {
	ID         uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	FromUserID string    `gorm:"index:idx_to;type:varchar(64);not null" json:"fromUserId"`
	ToUserID   string    `gorm:"index:idx_to;type:varchar(64);not null" json:"toUserId"`
	Message    string    `gorm:"type:varchar(256);not null;default:''" json:"message"` // 验证消息
	Status     int8      `gorm:"type:tinyint;not null;default:0" json:"status"`        // 0=待处理 1=已同意 2=已拒绝
	CreatedAt  time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"createdAt"`
	UpdatedAt  time.Time `gorm:"not null;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" json:"updatedAt"`
}

func (FriendRequest) TableName() string { return "friend_requests" }
