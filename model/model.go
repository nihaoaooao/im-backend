package model

import "time"

// User 用户模型
type User struct {
	ID           int64     `json:"id" gorm:"primaryKey"`
	Username     string    `json:"username" gorm:"uniqueIndex;size:50;not null"`
	PasswordHash string    `json:"-" gorm:"size:255;not null"`
	Email        string    `json:"email" gorm:"size:100"`
	Phone        string    `json:"phone" gorm:"size:20"`
	Nickname     string    `json:"nickname" gorm:"size:100"`
	Avatar       string    `json:"avatar" gorm:"size:255"`
	Gender       string    `json:"gender" gorm:"size:10;default:unknown"`
	Status       string    `json:"status" gorm:"size:20;default:active"`
	LastLoginAt  time.Time `json:"last_login_at"`
	LastLoginIP  string    `json:"last_login_ip" gorm:"size:50"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Conversation 会话模型
type Conversation struct {
	ID             int64     `json:"id" gorm:"primaryKey"`
	Type           string    `json:"type" gorm:"size:20;not null"` // private, group
	Name           string    `json:"name" gorm:"size:100"`
	CreatorID     int64     `json:"creator_id"`
	LastMsgID     int64     `json:"last_msg_id"`
	LastMsgContent string   `json:"last_msg_content"`
	LastMsgTime    time.Time `json:"last_msg_time"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`

	// 关联
	Members []ConversationMember `json:"members,omitempty" gorm:"foreignKey:ConversationID"`
}

// Message 消息模型
type Message struct {
	ID             int64     `json:"id" gorm:"primaryKey"`
	MsgID          string    `json:"msg_id" gorm:"uniqueIndex;size:64;not null"`
	ConversationID int64     `json:"conversation_id" gorm:"index;not null"`
	SenderID       int64     `json:"sender_id" gorm:"index;not null"`
	Content        string    `json:"content" gorm:"type:text"`
	ContentType    string    `json:"content_type" gorm:"size:20;default:text"` // text, image, voice, video, file
	Extra          string    `json:"extra" gorm:"type:jsonb"`
	IsRecalled     bool      `json:"is_recalled" gorm:"default:false"`
	CreatedAt      time.Time `json:"created_at" gorm:"index"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// ConversationMember 会话成员模型
type ConversationMember struct {
	ID             int64     `json:"id" gorm:"primaryKey"`
	ConversationID int64     `json:"conversation_id" gorm:"index;not null"`
	UserID         int64     `json:"user_id" gorm:"index;not null"`
	Role           string    `json:"role" gorm:"size:20;default:member"` // owner, admin, member
	Nickname       string    `json:"nickname" gorm:"size:100"`
	UnreadCount    int64     `json:"unread_count" gorm:"default:0"`
	JoinedAt       time.Time `json:"joined_at"`
}

// MessageRead 已读回执模型
type MessageRead struct {
	ID             int64     `json:"id" gorm:"primaryKey"`
	MessageID      int64     `json:"message_id" gorm:"index;not null"`
	UserID         int64     `json:"user_id" gorm:"index;not null"`
	ConversationID int64     `json:"conversation_id" gorm:"index;not null"`
	ReadAt         time.Time `json:"read_at"`
}

// Friendship 好友关系模型
type Friendship struct {
	ID        int64     `json:"id" gorm:"primaryKey"`
	UserID    int64     `json:"user_id" gorm:"index;not null"`
	FriendID  int64     `json:"friend_id" gorm:"index;not null"`
	Remark    string    `json:"remark" gorm:"size:100"`
	Status    string    `json:"status" gorm:"size:20;default:pending"` // pending, accepted, blocked
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// UserToken 用户 Token 模型
type UserToken struct {
	ID         int64     `json:"id" gorm:"primaryKey"`
	UserID     int64     `json:"user_id" gorm:"index;not null"`
	Token      string    `json:"token" gorm:"uniqueIndex;size:255;not null"`
	DeviceType string    `json:"device_type" gorm:"size:50"`
	DeviceID   string    `json:"device_id" gorm:"size:100"`
	IP         string    `json:"ip" gorm:"size:50"`
	ExpiresAt  time.Time `json:"expires_at"`
	CreatedAt  time.Time `json:"created_at"`
}
