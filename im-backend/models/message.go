package models

import (
	"time"
)

// Message 消息模型
type Message struct {
	ID              int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	MsgID           string    `gorm:"type:varchar(64);uniqueIndex;not null" json:"msg_id"`
	ConversationID  int64     `gorm:"index;not null" json:"conversation_id"`
	SenderID        int64     `gorm:"index;not null" json:"sender_id"`
	ReceiverID      int64     `gorm:"index;not null" json:"receiver_id"`
	ReceiverType    string    `gorm:"type:varchar(20);not null" json:"receiver_type"` // user, group
	Content         string    `gorm:"type:text" json:"content"`
	ContentType     string    `gorm:"type:varchar(20);default:'text'" json:"content_type"` // text, image, audio, video, file
	Extra           string    `gorm:"type:jsonb" json:"extra"`
	IsRecalled      bool      `gorm:"default:false" json:"is_recalled"`
	Revoked         bool      `gorm:"default:false" json:"revoked"`
	RevokedAt       *time.Time `gorm:"index" json:"revoked_at,omitempty"`
	CanRevoke       bool      `gorm:"default:true" json:"can_revoke"`
	CreatedAt       time.Time `gorm:"index" json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// TableName 指定表名
func (Message) TableName() string {
	return "messages"
}

// MessageRecall 消息撤回记录
type MessageRecall struct {
	ID             int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	MessageID      int64     `gorm:"index;not null" json:"message_id"`
	MsgID          string    `gorm:"type:varchar(64);not null" json:"msg_id"`
	ConversationID int64     `gorm:"index;not null" json:"conversation_id"`
	SenderID       int64     `gorm:"index;not null" json:"sender_id"`
	RevokerID      int64     `gorm:"index;not null" json:"revoker_id"`
	RecallTime     time.Time `gorm:"index" json:"recall_time"`
	Reason         string    `gorm:"type:varchar(255)" json:"reason"`
}

// TableName 指定表名
func (MessageRecall) TableName() string {
	return "message_recalls"
}

// MessageRead 消息已读记录
type MessageRead struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	MessageID int64     `gorm:"index;not null" json:"message_id"`
	UserID    int64     `gorm:"index;not null" json:"user_id"`
	ReadAt    time.Time `gorm:"index" json:"read_at"`
	CreatedAt time.Time `json:"created_at"`
}

// TableName 指定表名
func (MessageRead) TableName() string {
	return "message_reads"
}

// ConversationUnreadCount 会话未读数
type ConversationUnreadCount struct {
	ID              int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	ConversationID  int64     `gorm:"index;not null" json:"conversation_id"`
	UserID          int64     `gorm:"index;not null" json:"user_id"`
	UnreadCount     int64     `gorm:"default:0" json:"unread_count"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// TableName 指定表名
func (ConversationUnreadCount) TableName() string {
	return "conversation_unread_counts"
}
