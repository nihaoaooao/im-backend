package cache

import (
	"context"
	"fmt"
	"time"
)

// OnlineCache 用户在线状态缓存
type OnlineCache struct {
	client   *RedisClient
	ctx      context.Context
	keyUser  string
	keyOnline string
}

// NewOnlineCache 创建在线状态缓存
func NewOnlineCache(client *RedisClient) *OnlineCache {
	return &OnlineCache{
		client:    client,
		ctx:       context.Background(),
		keyUser:  "user:%d:connections",
		keyOnline: "online:users",
	}
}

// SetConnection 设置用户连接
func (o *OnlineCache) SetConnection(userID int64, connectionID string) error {
	key := fmt.Sprintf(o.keyUser, userID)
	return o.client.SAdd(key, connectionID)
}

// RemoveConnection 移除用户连接
func (o *OnlineCache) RemoveConnection(userID int64, connectionID string) error {
	key := fmt.Sprintf(o.keyUser, userID)
	err := o.client.SRem(key, connectionID)
	if err != nil {
		return err
	}
	count, _ := o.client.SCard(key)
	if count == 0 {
		return o.client.SRem(o.keyOnline, userID)
	}
	return nil
}

// GetConnections 获取用户所有连接
func (o *OnlineCache) GetConnections(userID int64) ([]string, error) {
	key := fmt.Sprintf(o.keyUser, userID)
	return o.client.SMembers(key)
}

// IsOnline 检查用户是否在线
func (o *OnlineCache) IsOnline(userID int64) (bool, error) {
	members, _ := o.client.SMembers(fmt.Sprintf(o.keyUser, userID))
	return len(members) > 0, nil
}

// GetOnlineUsers 获取所有在线用户
func (o *OnlineCache) GetOnlineUsers() ([]int64, error) {
	members, err := o.client.SMembers(o.keyOnline)
	if err != nil {
		return nil, err
	}
	var userIDs []int64
	for _, m := range members {
		var uid int64
		fmt.Sscanf(m, "%d", &uid)
		userIDs = append(userIDs, uid)
	}
	return userIDs, nil
}

// GetOnlineCount 获取在线用户数
func (o *OnlineCache) GetOnlineCount() (int64, error) {
	return o.client.SCard(o.keyOnline)
}

// SetUserOnline 设置用户在线状态
func (o *OnlineCache) SetUserOnline(userID int64) error {
	return o.client.SAdd(o.keyOnline, userID)
}

// SetUserOffline 设置用户离线状态
func (o *OnlineCache) SetUserOffline(userID int64) error {
	return o.client.SRem(o.keyOnline, userID)
}

// ConnectionCount 获取用户连接数
func (o *OnlineCache) ConnectionCount(userID int64) (int64, error) {
	key := fmt.Sprintf(o.keyUser, userID)
	return o.client.SCard(key)
}

// MaxConnectionsPerUser 单用户最大连接数
const MaxConnectionsPerUser = 5

// CanConnect 检查用户是否可以建立新连接
func (o *OnlineCache) CanConnect(userID int64) (bool, error) {
	count, err := o.ConnectionCount(userID)
	if err != nil {
		return false, err
	}
	return count < MaxConnectionsPerUser, nil
}

// CleanupUserConnections 清理用户所有连接
func (o *OnlineCache) CleanupUserConnections(userID int64) error {
	key := fmt.Sprintf(o.keyUser, userID)
	members, _ := o.client.SMembers(key)
	for _, connID := range members {
		o.client.SRem(key, connID)
	}
	return o.client.SRem(o.keyOnline, userID)
}

// Heartbeat 用户心跳
func (o *OnlineCache) Heartbeat(userID int64, connectionID string) error {
	key := fmt.Sprintf(o.keyUser, userID)
	exists, _ := o.client.SIsMember(key, connectionID)
	if !exists {
		return nil
	}
	return o.client.Expire(key, 90*time.Second)
}
