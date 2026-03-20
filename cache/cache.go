package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"im-backend/config"

	"github.com/redis/go-redis/v9"
)

// RedisClient Redis 客户端封装
type RedisClient struct {
	client *redis.Client
	ctx    context.Context
}

// NewRedisClient 创建 Redis 客户端
func NewRedisClient(cfg *config.Config) (*RedisClient, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", cfg.RedisHost, cfg.RedisPort),
		Password:     cfg.RedisPassword,
		DB:           cfg.RedisDB,
		PoolSize:     100,
		MinIdleConns: 10,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolTimeout:  10 * time.Second,
	})

	ctx := context.Background()
	_, err := client.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisClient{
		client: client,
		ctx:    ctx,
	}, nil
}

// GetClient 获取原始客户端
func (r *RedisClient) GetClient() *redis.Client {
	return r.client
}

// Close 关闭连接
func (r *RedisClient) Close() error {
	return r.client.Close()
}

// ============ 基础操作 ============

// Set 设置字符串
func (r *RedisClient) Set(key string, value interface{}, expiration time.Duration) error {
	return r.client.Set(r.ctx, key, value, expiration).Err()
}

// Get 获取字符串
func (r *RedisClient) Get(key string) (string, error) {
	return r.client.Get(r.ctx, key).Result()
}

// SetJSON 设置 JSON
func (r *RedisClient) SetJSON(key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return r.client.Set(r.ctx, key, data, expiration).Err()
}

// GetJSON 获取 JSON
func (r *RedisClient) GetJSON(key string, dest interface{}) error {
	data, err := r.client.Get(r.ctx, key).Bytes()
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dest)
}

// Del 删除键
func (r *RedisClient) Del(keys ...string) error {
	return r.client.Del(r.ctx, keys...).Err()
}

// Exists 检查键是否存在
func (r *RedisClient) Exists(key string) (bool, error) {
	n, err := r.client.Exists(r.ctx, key).Result()
	return n > 0, err
}

// Expire 设置过期时间
func (r *RedisClient) Expire(key string, expiration time.Duration) error {
	return r.client.Expire(r.ctx, key, expiration).Err()
}

// ============ Hash 操作 ============

// HSet 设置 Hash
func (r *RedisClient) HSet(key string, field string, value interface{}) error {
	return r.client.HSet(r.ctx, key, field, value).Err()
}

// HGet 获取 Hash
func (r *RedisClient) HGet(key string, field string) (string, error) {
	return r.client.HGet(r.ctx, key, field).Result()
}

// HGetAll 获取所有 Hash
func (r *RedisClient) HGetAll(key string) (map[string]string, error) {
	return r.client.HGetAll(r.ctx, key).Result()
}

// HDel 删除 Hash 字段
func (r *RedisClient) HDel(key string, fields ...string) error {
	return r.client.HDel(r.ctx, key, fields...).Err()
}

// HIncrBy 增加 Hash 值
func (r *RedisClient) HIncrBy(key string, field string, increment int64) (int64, error) {
	return r.client.HIncrBy(r.ctx, key, field, increment).Result()
}

// HLen 获取 Hash 长度
func (r *RedisClient) HLen(key string) (int64, error) {
	return r.client.HLen(r.ctx, key).Result()
}

// ============ Set 操作 ============

// SAdd 添加到集合
func (r *RedisClient) SAdd(key string, members ...interface{}) error {
	return r.client.SAdd(r.ctx, key, members...).Err()
}

// SRem 从集合移除
func (r *RedisClient) SRem(key string, members ...interface{}) error {
	return r.client.SRem(r.ctx, key, members...).Err()
}

// SMembers 获取集合所有成员
func (r *RedisClient) SMembers(key string) ([]string, error) {
	return r.client.SMembers(r.ctx, key).Result()
}

// SIsMember 检查是否是成员
func (r *RedisClient) SIsMember(key string, member interface{}) (bool, error) {
	return r.client.SIsMember(r.ctx, key, member).Result()
}

// SCard 获取集合大小
func (r *RedisClient) SCard(key string) (int64, error) {
	return r.client.SCard(r.ctx, key).Result()
}

// ============ List 操作 ============

// LPush 入队
func (r *RedisClient) LPush(key string, values ...interface{}) error {
	return r.client.LPush(r.ctx, key, values...).Err()
}

// RPop 出队
func (r *RedisClient) RPop(key string) (string, error) {
	return r.client.RPop(r.ctx, key).Result()
}

// LRange 获取列表范围
func (r *RedisClient) LRange(key string, start int64, stop int64) ([]string, error) {
	return r.client.LRange(r.ctx, key, start, stop).Result()
}

// LLen 获取列表长度
func (r *RedisClient) LLen(key string) (int64, error) {
	return r.client.LLen(r.ctx, key).Result()
}

// ============ Stream 操作 ============

// XAdd 添加到流
func (r *RedisClient) XAdd(stream string, values map[string]interface{}) (string, error) {
	return r.client.XAdd(r.ctx, &redis.XAddArgs{
		Stream: stream,
		Values: values,
	}).Result()
}

// XReadGroup 消费组读取
func (r *RedisClient) XReadGroup(group string, consumer string, streams []string, count int64, block time.Duration) ([]redis.XMessage, error) {
	result, err := r.client.XReadGroup(r.ctx, &redis.XReadGroupArgs{
		Group:    group,
		Consumer: consumer,
		Streams:  streams,
		Count:    count,
		Block:    block,
	}).Result()
	if err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, nil
	}
	return result[0].Messages, nil
}

// XAck 确认消息
func (r *RedisClient) XAck(stream string, group string, messageIDs ...string) (int64, error) {
	return r.client.XAck(r.ctx, stream, group, messageIDs...).Result()
}

// XDel 删除消息
func (r *RedisClient) XDel(stream string, messageIDs ...string) (int64, error) {
	return r.client.XDel(r.ctx, stream, messageIDs...).Result()
}

// XLen 获取流长度
func (r *RedisClient) XLen(stream string) (int64, error) {
	return r.client.XLen(r.ctx, stream).Result()
}

// ============ ZSet 操作 ============

// ZAdd 添加到有序集合
func (r *RedisClient) ZAdd(key string, members ...redis.Z) error {
	return r.client.ZAdd(r.ctx, key, members...).Err()
}

// ZRangeByScore 按分数范围获取
func (r *RedisClient) ZRangeByScore(key string, min string, max string) ([]string, error) {
	return r.client.ZRangeByScore(r.ctx, key, &redis.ZRangeBy{
		Min: min,
		Max: max,
	}).Result()
}

// ZRem 删除成员
func (r *RedisClient) ZRem(key string, members ...interface{}) error {
	return r.client.ZRem(r.ctx, key, members...).Err()
}

// ============ 管道操作 ============

// Pipeline 管道
func (r *RedisClient) Pipeline() redis.Pipeliner {
	return r.client.Pipeline()
}

// ============ 分布式锁 ============

// Lock 尝试获取分布式锁
func (r *RedisClient) Lock(key string, value interface{}, expiration time.Duration) (bool, error) {
	return r.client.SetNX(r.ctx, key, value, expiration).Result()
}

// Unlock 释放分布式锁
func (r *RedisClient) Unlock(key string) error {
	return r.client.Del(r.ctx, key).Err()
}
