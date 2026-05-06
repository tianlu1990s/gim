package model

import "time"

// User 映射到 MySQL users 表。
// 密码字段 json:"-" 防止序列化时泄露；ID 用 uint64 自增主键，对外暴露 UserID（字符串）。
type User struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"-"`
	UserID    string    `gorm:"uniqueIndex;type:varchar(64);not null" json:"userId"`
	Nickname  string    `gorm:"type:varchar(64);not null;default:''" json:"nickname"`
	AvatarURL string    `gorm:"type:varchar(512);not null;default:''" json:"avatarUrl"`
	Password  string    `gorm:"type:varchar(128);not null" json:"-"` // bcrypt 哈希，永不返回给前端
	Phone     string    `gorm:"index;type:varchar(20);not null;default:''" json:"phone"`
	Email     string    `gorm:"index;type:varchar(128);not null;default:''" json:"email"`
	Status    int8      `gorm:"type:tinyint;not null;default:1" json:"status"` // 1=正常 2=禁用
	CreatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"createdAt"`
	UpdatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" json:"updatedAt"`
}

func (User) TableName() string { return "users" }
