package api

import (
	"im-backend/model"
	"time"
)

// ============ 用户 DTO ============

// UserDTO 用户基本信息（公开）
type UserDTO struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Nickname string `json:"nickname,omitempty"`
	Avatar   string `json:"avatar,omitempty"`
	Gender   string `json:"gender,omitempty"`
	Status   string `json:"status,omitempty"`
}

// UserDetailDTO 用户详细信息（个人）
type UserDetailDTO struct {
	ID         int64     `json:"id"`
	Username   string    `json:"username"`
	Nickname   string    `json:"nickname,omitempty"`
	Avatar     string    `json:"avatar,omitempty"`
	Gender     string    `json:"gender,omitempty"`
	Email      string    `json:"email,omitempty"`
	Phone      string    `json:"phone,omitempty"`
	Status     string    `json:"status,omitempty"`
	LastLoginAt time.Time `json:"last_login_at,omitempty"`
}

// ToUserDTO 转换为用户DTO
func ToUserDTO(user *model.User) *UserDTO {
	if user == nil {
		return nil
	}
	return &UserDTO{
		ID:       user.ID,
		Username: user.Username,
		Nickname: user.Nickname,
		Avatar:   user.Avatar,
		Gender:   user.Gender,
		Status:   user.Status,
	}
}

// ToUserDetailDTO 转换为用户详细信息DTO
func ToUserDetailDTO(user *model.User) *UserDetailDTO {
	if user == nil {
		return nil
	}
	return &UserDetailDTO{
		ID:         user.ID,
		Username:   user.Username,
		Nickname:   user.Nickname,
		Avatar:     user.Avatar,
		Gender:     user.Gender,
		Email:      user.Email,
		Phone:      user.Phone,
		Status:     user.Status,
		LastLoginAt: user.LastLoginAt,
	}
}

// ============ 好友 DTO ============

// FriendDTO 好友信息
type FriendDTO struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	FriendID  int64     `json:"friend_id"`
	Remark    string    `json:"remark,omitempty"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`

	// 关联的用户信息
	Friend *UserDTO `json:"friend,omitempty"`
}

// FriendRequestDTO 好友请求
type FriendRequestDTO struct {
	ID          int64     `json:"id"`
	FromUserID  int64     `json:"from_user_id"`
	ToUserID    int64     `json:"to_user_id"`
	Status      string    `json:"status"` // pending, accepted, rejected
	Message     string    `json:"message,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// 关联的用户信息
	FromUser *UserDTO `json:"from_user,omitempty"`
	ToUser   *UserDTO `json:"to_user,omitempty"`
}

// ToFriendDTO 转换为好友DTO
func ToFriendDTO(f *model.Friendship, friend *model.User) *FriendDTO {
	if f == nil {
		return nil
	}
	dto := &FriendDTO{
		ID:        f.ID,
		UserID:    f.UserID,
		FriendID:  f.FriendID,
		Remark:    f.Remark,
		Status:    f.Status,
		CreatedAt: f.CreatedAt,
	}
	if friend != nil {
		dto.Friend = ToUserDTO(friend)
	}
	return dto
}

// ============ 会话 DTO ============

// ConversationDTO 会话信息
type ConversationDTO struct {
	ID             int64     `json:"id"`
	Type           string    `json:"type"`
	Name           string    `json:"name,omitempty"`
	LastMsgContent string    `json:"last_msg_content,omitempty"`
	LastMsgTime    time.Time `json:"last_msg_time,omitempty"`
	UnreadCount    int64     `json:"unread_count"`
	MemberCount    int64     `json:"member_count"`

	// 私聊时显示对方信息
	Peer *UserDTO `json:"peer,omitempty"`
}

// ToConversationDTO 转换为会话DTO
func ToConversationDTO(conv *model.Conversation, member *model.ConversationMember, peer *model.User) *ConversationDTO {
	if conv == nil {
		return nil
	}
	dto := &ConversationDTO{
		ID:             conv.ID,
		Type:           conv.Type,
		Name:           conv.Name,
		LastMsgContent: conv.LastMsgContent,
		LastMsgTime:    conv.LastMsgTime,
	}
	if member != nil {
		dto.UnreadCount = member.UnreadCount
	}
	if peer != nil {
		dto.Peer = ToUserDTO(peer)
	}
	return dto
}

// ============ 消息 DTO ============

// MessageDTO 消息信息
type MessageDTO struct {
	ID             int64     `json:"id"`
	MsgID          string    `json:"msg_id"`
	ConversationID int64    `json:"conversation_id"`
	SenderID       int64    `json:"sender_id"`
	SenderName     string    `json:"sender_name,omitempty"`
	Content        string    `json:"content"`
	ContentType    string    `json:"content_type"`
	IsRecalled     bool      `json:"is_recalled"`
	CreatedAt      time.Time `json:"created_at"`
}

// ToMessageDTO 转换为消息DTO
func ToMessageDTO(msg *model.Message) *MessageDTO {
	if msg == nil {
		return nil
	}
	return &MessageDTO{
		ID:             msg.ID,
		MsgID:          msg.MsgID,
		ConversationID: msg.ConversationID,
		SenderID:       msg.SenderID,
		Content:        msg.Content,
		ContentType:    msg.ContentType,
		IsRecalled:     msg.IsRecalled,
		CreatedAt:      msg.CreatedAt,
	}
}

// ============ 群组 DTO ============

// GroupDTO 群组信息
type GroupDTO struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Avatar    string    `json:"avatar,omitempty"`
	OwnerID   int64    `json:"owner_id"`
	MemberCount int64   `json:"member_count"`
	CreatedAt time.Time `json:"created_at"`
}

// GroupMemberDTO 群成员信息
type GroupMemberDTO struct {
	UserID   int64     `json:"user_id"`
	Nickname string    `json:"nickname,omitempty"`
	Role     string    `json:"role"`
	JoinedAt time.Time `json:"joined_at"`

	User *UserDTO `json:"user,omitempty"`
}

// ============ 登录响应 DTO ============

// LoginResponse 登录响应
type LoginResponse struct {
	UserID       int64  `json:"user_id"`
	Username     string `json:"username"`
	Nickname     string `json:"nickname,omitempty"`
	Avatar       string `json:"avatar,omitempty"`
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	ExpireIn     int    `json:"expire_in"`
}
