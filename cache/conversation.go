package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// ConversationCache 会话信息缓存
type ConversationCache struct {
	client     *RedisClient
	ctx        context.Context
	keyInfo    string
	keyMembers string
	keyUserConv string
}

// ConversationInfo 会话信息
type ConversationInfo struct {
	ID              int64  `json:"id"`
	Type            string `json:"type"` // private, group
	Name            string `json:"name"`
	Avatar          string `json:"avatar"`
	CreatorID       int64  `json:"creator_id"`
	LastMsgID       int64  `json:"last_msg_id"`
	LastMsgContent  string `json:"last_msg_content"`
	LastMsgTime     int64  `json:"last_msg_time"`
	MemberCount    int64  `json:"member_count"`
	CreatedAt      int64  `json:"created_at"`
	UpdatedAt      int64  `json:"updated_at"`
}

// NewConversationCache 创建会话缓存
func NewConversationCache(client *RedisClient) *ConversationCache {
	return &ConversationCache{
		client:      client,
		ctx:         context.Background(),
		keyInfo:    "conversation:%d:info",
		keyMembers: "conversation:%d:members",
		keyUserConv: "user:%d:conversations",
	}
}

// SetInfo 缓存会话信息
func (c *ConversationCache) SetInfo(info *ConversationInfo) error {
	key := fmt.Sprintf(c.keyInfo, info.ID)
	data, err := json.Marshal(info)
	if err != nil {
		return err
	}
	return c.client.Set(key, data, time.Hour)
}

// GetInfo 获取会话信息
func (c *ConversationCache) GetInfo(conversationID int64) (*ConversationInfo, error) {
	key := fmt.Sprintf(c.keyInfo, conversationID)
	data, err := c.client.Get(key)
	if err != nil {
		return nil, err
	}

	var info ConversationInfo
	if err := json.Unmarshal([]byte(data), &info); err != nil {
		return nil, err
	}

	return &info, nil
}

// DelInfo 删除会话缓存
func (c *ConversationCache) DelInfo(conversationID int64) error {
	key := fmt.Sprintf(c.keyInfo, conversationID)
	return c.client.Del(key)
}

// SetMembers 缓存群成员
func (c *ConversationCache) SetMembers(conversationID int64, memberIDs []int64) error {
	key := fmt.Sprintf(c.keyMembers, conversationID)
	c.client.Del(key)

	if len(memberIDs) == 0 {
		return nil
	}

	members := make([]interface{}, len(memberIDs))
	for i, id := range memberIDs {
		members[i] = id
	}

	return c.client.SAdd(key, members...)
}

// GetMembers 获取群成员
func (c *ConversationCache) GetMembers(conversationID int64) ([]int64, error) {
	key := fmt.Sprintf(c.keyMembers, conversationID)
	members, err := c.client.SMembers(key)
	if err != nil {
		return nil, err
	}

	var memberIDs []int64
	for _, m := range members {
		var id int64
		fmt.Sscanf(m, "%d", &id)
		memberIDs = append(memberIDs, id)
	}

	return memberIDs, nil
}

// AddMember 添加群成员缓存
func (c *ConversationCache) AddMember(conversationID int64, memberID int64) error {
	key := fmt.Sprintf(c.keyMembers, conversationID)
	return c.client.SAdd(key, memberID)
}

// RemoveMember 移除群成员缓存
func (c *ConversationCache) RemoveMember(conversationID int64, memberID int64) error {
	key := fmt.Sprintf(c.keyMembers, conversationID)
	return c.client.SRem(key, memberID)
}

// IsMember 检查是否是成员
func (c *ConversationCache) IsMember(conversationID int64, memberID int64) (bool, error) {
	key := fmt.Sprintf(c.keyMembers, conversationID)
	return c.client.SIsMember(key, memberID)
}

// GetMemberCount 获取成员数量
func (c *ConversationCache) GetMemberCount(conversationID int64) (int64, error) {
	key := fmt.Sprintf(c.keyMembers, conversationID)
	return c.client.SCard(key)
}

// ClearMembers 清除成员缓存
func (c *ConversationCache) ClearMembers(conversationID int64) error {
	key := fmt.Sprintf(c.keyMembers, conversationID)
	return c.client.Del(key)
}

// SetUserConversations 缓存用户会话列表
func (c *ConversationCache) SetUserConversations(userID int64, conversationIDs []int64) error {
	key := fmt.Sprintf(c.keyUserConv, userID)
	c.client.Del(key)

	if len(conversationIDs) == 0 {
		return nil
	}

	convs := make([]interface{}, len(conversationIDs))
	for i, id := range conversationIDs {
		convs[i] = id
	}

	return c.client.SAdd(key, convs...)
}

// GetUserConversations 获取用户会话列表
func (c *ConversationCache) GetUserConversations(userID int64) ([]int64, error) {
	key := fmt.Sprintf(c.keyUserConv, userID)
	members, err := c.client.SMembers(key)
	if err != nil {
		return nil, err
	}

	var convIDs []int64
	for _, m := range members {
		var id int64
		fmt.Sscanf(m, "%d", &id)
		convIDs = append(convIDs, id)
	}

	return convIDs, nil
}

// AddUserConversation 添加用户会话
func (c *ConversationCache) AddUserConversation(userID int64, conversationID int64) error {
	key := fmt.Sprintf(c.keyUserConv, userID)
	return c.client.SAdd(key, conversationID)
}

// RemoveUserConversation 移除用户会话
func (c *ConversationCache) RemoveUserConversation(userID int64, conversationID int64) error {
	key := fmt.Sprintf(c.keyUserConv, userID)
	return c.client.SRem(key, conversationID)
}

// ClearUserConversations 清除用户会话缓存
func (c *ConversationCache) ClearUserConversations(userID int64) error {
	key := fmt.Sprintf(c.keyUserConv, userID)
	return c.client.Del(key)
}

// UserCache 用户信息缓存
type UserCache struct {
	client *RedisClient
	ctx    context.Context
	key    string
}

// UserProfile 用户资料
type UserProfile struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Gender   string `json:"gender"`
	Status   string `json:"status"`
}

// NewUserCache 创建用户缓存
func NewUserCache(client *RedisClient) *UserCache {
	return &UserCache{
		client: client,
		ctx:    context.Background(),
		key:    "user:%d:profile",
	}
}

// SetProfile 缓存用户资料
func (u *UserCache) SetProfile(profile *UserProfile) error {
	key := fmt.Sprintf(u.key, profile.ID)
	data, err := json.Marshal(profile)
	if err != nil {
		return err
	}
	return u.client.Set(key, data, 30*time.Minute)
}

// GetProfile 获取用户资料
func (u *UserCache) GetProfile(userID int64) (*UserProfile, error) {
	key := fmt.Sprintf(u.key, userID)
	data, err := u.client.Get(key)
	if err != nil {
		return nil, err
	}

	var profile UserProfile
	if err := json.Unmarshal([]byte(data), &profile); err != nil {
		return nil, err
	}

	return &profile, nil
}

// DelProfile 删除用户缓存
func (u *UserCache) DelProfile(userID int64) error {
	key := fmt.Sprintf(u.key, userID)
	return u.client.Del(key)
}

// TokenCache Token 缓存
type TokenCache struct {
	client       *RedisClient
	ctx          context.Context
	blacklistKey string
}

// NewTokenCache 创建 Token 缓存
func NewTokenCache(client *RedisClient) *TokenCache {
	return &TokenCache{
		client:       client,
		ctx:          context.Background(),
		blacklistKey: "token:blacklist:%s",
	}
}

// AddToBlacklist 加入黑名单
func (t *TokenCache) AddToBlacklist(token string, expiry time.Duration) error {
	key := fmt.Sprintf(t.blacklistKey, token)
	return t.client.Set(key, "1", expiry)
}

// IsBlacklisted 检查是否在黑名单
func (t *TokenCache) IsBlacklisted(token string) (bool, error) {
	key := fmt.Sprintf(t.blacklistKey, token)
	return t.client.Exists(key)
}

// RemoveFromBlacklist 从黑名单移除
func (t *TokenCache) RemoveFromBlacklist(token string) error {
	key := fmt.Sprintf(t.blacklistKey, token)
	return t.client.Del(key)
}
