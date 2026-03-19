package models

import (
	"time"

	"gorm.io/gorm"
)

// User 用户模型
type User struct {
	ID           uint           `gorm:"primarykey" json:"id"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
	Username     string         `gorm:"uniqueIndex;size:50;not null" json:"username"`
	PasswordHash string         `gorm:"size:255;not null" json:"-"`
	Nickname     string         `gorm:"size:100" json:"nickname"`
	Email        string         `gorm:"size:100;uniqueIndex" json:"email"`
	Phone        string         `gorm:"size:20" json:"phone"`
	Avatar       string         `gorm:"size:255" json:"avatar"`
	Role         string         `gorm:"size:20;default:'user'" json:"role"` // admin, user
	Status       string         `gorm:"size:20;default:'active'" json:"status"` // active, inactive, banned
	LastLoginAt  *time.Time     `json:"last_login_at,omitempty"`
}

// TableName 表名
func (User) TableName() string {
	return "users"
}
