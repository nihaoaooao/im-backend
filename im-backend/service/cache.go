package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// CacheService 缓存服务
type CacheService struct {
	client *redis.Client
}

// NewCacheService 创建缓存服务
func NewCacheService(client *redis.Client) *CacheService {
	return &CacheService{
		client: client,
	}
}

// CacheConfig 缓存配置
type CacheConfig struct {
	DefaultTTL time.Duration // 默认过期时间
	HotDataTTL time.Duration // 热点数据过期时间
}

// 默认配置
var DefaultCacheConfig = &CacheConfig{
	DefaultTTL: 30 * time.Minute,
	HotDataTTL: 5 * time.Minute,
}

// ============ 会话缓存 ============

// ConversationCache 会话缓存
type ConversationCache struct {
	ID           int64   `json:"id"`
	Name         string  `json:"name"`
	Avatar       string  `json:"avatar"`
	LastMsg      string  `json:"last_msg"`
	LastMsgTime  int64   `json:"last_msg_time"`
	UnreadCount  int64   `json:"unread_count"`
}

// GetConversations 获取会话列表缓存
func (s *CacheService) GetConversations(ctx context.Context, userID int64) ([]ConversationCache, error) {
	key := fmt.Sprintf("user:%d:conversations", userID)
	data, err := s.client.Get(ctx, key).Bytes()
	if err != nil {
		return nil, err
	}

	var conversations []ConversationCache
	if err := json.Unmarshal(data, &conversations); err != nil {
		return nil, err
	}
	return conversations, nil
}

// SetConversations 设置会话列表缓存
func (s *CacheService) SetConversations(ctx context.Context, userID int64, conversations []ConversationCache) error {
	key := fmt.Sprintf("user:%d:conversations", userID)
	data, err := json.Marshal(conversations)
	if err != nil {
		return err
	}
	return s.client.Set(ctx, key, data, DefaultCacheConfig.DefaultTTL).Err()
}

// InvalidateConversations 失效会话缓存
func (s *CacheService) InvalidateConversations(ctx context.Context, userID int64) error {
	key := fmt.Sprintf("user:%d:conversations", userID)
	return s.client.Del(ctx, key).Err()
}

// ============ 用户信息缓存 ============

// UserCache 用户信息缓存
type UserCache struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
	Status   int    `json:"status"`
}

// GetUser 获取用户缓存
func (s *CacheService) GetUser(ctx context.Context, userID int64) (*UserCache, error) {
	key := fmt.Sprintf("user:%d:info", userID)
	data, err := s.client.Get(ctx, key).Bytes()
	if err != nil {
		return nil, err
	}

	var user UserCache
	if err := json.Unmarshal(data, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

// SetUser 设置用户缓存
func (s *CacheService) SetUser(ctx context.Context, userID int64, user *UserCache) error {
	key := fmt.Sprintf("user:%d:info", userID)
	data, err := json.Marshal(user)
	if err != nil {
		return err
	}
	return s.client.Set(ctx, key, data, DefaultCacheConfig.HotDataTTL).Err()
}

// InvalidateUser 失效用户缓存
func (s *CacheService) InvalidateUser(ctx context.Context, userID int64) error {
	key := fmt.Sprintf("user:%d:info", userID)
	return s.client.Del(ctx, key).Err()
}

// ============ 消息缓存 ============

// MessageCache 消息缓存
type MessageCache struct {
	ID             int64  `json:"id"`
	MsgID          string `json:"msg_id"`
	ConversationID int64 `json:"conversation_id"`
	SenderID       int64  `json:"sender_id"`
	Content        string `json:"content"`
	CreatedAt      int64  `json:"created_at"`
}

// GetMessages 获取消息列表缓存
func (s *CacheService) GetMessages(ctx context.Context, conversationID int64, offset, limit int64) ([]MessageCache, error) {
	key := fmt.Sprintf("conversation:%d:messages:%d:%d", conversationID, offset, limit)
	data, err := s.client.Get(ctx, key).Bytes()
	if err != nil {
		return nil, err
	}

	var messages []MessageCache
	if err := json.Unmarshal(data, &messages); err != nil {
		return nil, err
	}
	return messages, nil
}

// SetMessages 设置消息列表缓存
func (s *CacheService) SetMessages(ctx context.Context, conversationID int64, offset, limit int64, messages []MessageCache) error {
	key := fmt.Sprintf("conversation:%d:messages:%d:%d", conversationID, offset, limit)
	data, err := json.Marshal(messages)
	if err != nil {
		return err
	}
	// 消息缓存时间较短，保证及时性
	return s.client.Set(ctx, key, data, 5*time.Minute).Err()
}

// InvalidateMessages 失效消息缓存
func (s *CacheService) InvalidateMessages(ctx context.Context, conversationID int64) error {
	pattern := fmt.Sprintf("conversation:%d:messages:*", conversationID)
	return s.deletePattern(ctx, pattern)
}

// ============ 在线状态缓存 ============

// SetUserOnline 设置用户在线状态
func (s *CacheService) SetUserOnline(ctx context.Context, userID int64) error {
	key := fmt.Sprintf("online:%d", userID)
	return s.client.Set(ctx, key, "1", 5*time.Minute).Err()
}

// SetUserOffline 设置用户离线状态
func (s *CacheService) SetUserOffline(ctx context.Context, userID int64) error {
	key := fmt.Sprintf("online:%d", userID)
	return s.client.Del(ctx, key).Err()
}

// IsUserOnline 检查用户是否在线
func (s *CacheService) IsUserOnline(ctx context.Context, userID int64) bool {
	key := fmt.Sprintf("online:%d", userID)
	result, err := s.client.Exists(ctx, key).Result()
	return err == nil && result > 0
}

// GetOnlineUsers 获取在线用户列表
func (s *CacheService) GetOnlineUsers(ctx context.Context) ([]int64, error) {
	pattern := "online:*"
	keys, err := s.client.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, err
	}

	var userIDs []int64
	for _, key := range keys {
		var userID int64
		fmt.Sscanf(key, "online:%d", &userID)
		userIDs = append(userIDs, userID)
	}
	return userIDs, nil
}

// ============ 限流缓存 ============

// IncrRateLimit 限流计数器
func (s *CacheService) IncrRateLimit(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	result, err := s.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}

	// 首次设置过期时间
	if result == 1 {
		s.client.Expire(ctx, key, ttl)
	}
	return result, nil
}

// GetRateLimit 获取限流计数
func (s *CacheService) GetRateLimit(ctx context.Context, key string) (int64, error) {
	return s.client.Get(ctx, key).Int64()
}

// ResetRateLimit 重置限流
func (s *CacheService) ResetRateLimit(ctx context.Context, key string) error {
	return s.client.Del(ctx, key).Err()
}

// ============ 批量操作 ============

// MGet 批量获取
func (s *CacheService) MGet(ctx context.Context, keys []string) ([]string, error) {
	return s.client.MGet(ctx, keys...).Slice()
}

// MSet 批量设置
func (s *CacheService) MSet(ctx context.Context, values map[string]interface{}) error {
	args := make([]interface{}, 0, len(values)*2)
	for k, v := range values {
		args = append(args, k, v)
	}
	return s.client.MSet(ctx, args...).Err()
}

// ============ 工具方法 ============

// deletePattern 删除匹配模式的所有key
func (s *CacheService) deletePattern(ctx context.Context, pattern string) error {
	iter := s.client.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		if err := s.client.Del(ctx, iter.Val()).Err(); err != nil {
			return err
		}
	}
	return iter.Err()
}

// Pipeline 创建管道
func (s *CacheService) Pipeline() redis.Pipeliner {
	return s.client.Pipeline()
}

// ============ 连接池配置 ============

// ConfigRedisConnectionPool 配置 Redis 连接池
func ConfigRedisConnectionPool(client *redis.Client) {
	poolConfig := &redis.PoolConfig{
		MaxIdle:         100,         // 最大空闲连接数
		MaxActive:       1000,        // 最大活跃连接数
		IdleTimeout:     5 * time.Minute, // 空闲超时
		MaxConnLifetime: 30 * time.Minute, // 连接最大生命周期
		Wait:            true,        // 等待可用连接
	}
	client.PoolConfig(poolConfig)
}

// ============ 性能监控 ============

// GetStats 获取 Redis 状态
func (s *CacheService) GetStats(ctx context.Context) (map[string]interface{}, error) {
	info, err := s.client.Info(ctx, "stats").Result()
	if err != nil {
		return nil, err
	}

	stats := make(map[string]interface{})
	// 解析 info 输出
	for _, line := range splitLines(info) {
		if len(line) > 0 && !startsWith(line, "#") {
			parts := splitColon(line)
			if len(parts) == 2 {
				stats[parts[0]] = parts[1]
			}
		}
	}
	return stats, nil
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	lines = append(lines, s[start:])
	return lines
}

func splitColon(s string) []string {
	for i := 0; i < len(s); i++ {
		if s[i] == ':' {
			return []string{s[:i], s[i+1:]}
		}
	}
	return nil
}

func startsWith(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}
