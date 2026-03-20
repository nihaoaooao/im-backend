package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// MessageQueue 消息队列缓存
type MessageQueue struct {
	client        *RedisClient
	ctx           context.Context
	streamKey     string
	offlineKey    string
	dlqKey        string
	delayKey      string
	consumerGroup string
}

// NewMessageQueue 创建消息队列
func NewMessageQueue(client *RedisClient) *MessageQueue {
	return &MessageQueue{
		client:        client,
		ctx:           context.Background(),
		streamKey:     "message:queue",
		offlineKey:    "message:offline:%d",
		dlqKey:        "message:queue:dlq",
		delayKey:      "message:delay:queue",
		consumerGroup: "im-consumer-group",
	}
}

// Init 初始化消息队列
func (m *MessageQueue) Init() error {
	err := m.client.client.XGroupCreateMkStream(m.ctx, m.streamKey, m.consumerGroup, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return fmt.Errorf("failed to create consumer group: %w", err)
	}
	fmt.Printf("[MessageQueue] Initialized: stream=%s, group=%s\n", m.streamKey, m.consumerGroup)
	return nil
}

// Message 消息结构
type Message struct {
	ID             string                 `json:"id"`
	MsgID          string                 `json:"msg_id"`
	ConversationID int64                  `json:"conversation_id"`
	SenderID       int64                  `json:"sender_id"`
	ReceiverID     int64                  `json:"receiver_id"`
	ReceiverType   string                 `json:"receiver_type"`
	Content        string                 `json:"content"`
	ContentType    string                 `json:"content_type"`
	Timestamp      int64                  `json:"timestamp"`
	Extra          map[string]interface{} `json:"extra"`
}

// Enqueue 消息入队
func (m *MessageQueue) Enqueue(msg *Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	_, err = m.client.XAdd(m.streamKey, map[string]interface{}{
		"type":    "message",
		"data":    string(data),
		"created": time.Now().Unix(),
	})
	return err
}

// EnqueueToOffline 离线消息入队
func (m *MessageQueue) EnqueueToOffline(userID int64, msg *Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	key := fmt.Sprintf(m.offlineKey, userID)
	return m.client.LPush(key, string(data))
}

// DequeueOffline 离线消息出队
func (m *MessageQueue) DequeueOffline(userID int64) (*Message, error) {
	key := fmt.Sprintf(m.offlineKey, userID)
	data, err := m.client.RPop(key)
	if err != nil {
		return nil, err
	}
	var msg Message
	if err := json.Unmarshal([]byte(data), &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

// GetOfflineMessages 获取所有离线消息
func (m *MessageQueue) GetOfflineMessages(userID int64) ([]*Message, error) {
	key := fmt.Sprintf(m.offlineKey, userID)
	data, err := m.client.LRange(key, 0, -1)
	if err != nil {
		return nil, err
	}
	var messages []*Message
	for _, d := range data {
		var msg Message
		if err := json.Unmarshal([]byte(d), &msg); err != nil {
			continue
		}
		messages = append(messages, &msg)
	}
	return messages, nil
}

// ClearOfflineMessages 清除离线消息
func (m *MessageQueue) ClearOfflineMessages(userID int64) error {
	key := fmt.Sprintf(m.offlineKey, userID)
	return m.client.Del(key)
}

// OfflineMessageCount 离线消息数量
func (m *MessageQueue) OfflineMessageCount(userID int64) (int64, error) {
	key := fmt.Sprintf(m.offlineKey, userID)
	return m.client.LLen(key)
}

// DequeueBatch 批量消费消息
func (m *MessageQueue) DequeueBatch(consumer string, count int, block time.Duration) ([]*Message, error) {
	stream := []string{m.streamKey, ">"}
	msgs, err := m.client.XReadGroup(m.consumerGroup, consumer, stream, int64(count), block)
	if err != nil {
		return nil, err
	}
	var messages []*Message
	for _, msg := range msgs {
		data, ok := msg.Values["data"].(string)
		if !ok {
			continue
		}
		var m Message
		if err := json.Unmarshal([]byte(data), &m); err != nil {
			continue
		}
		m.ID = msg.ID
		messages = append(messages, &m)
	}
	return messages, nil
}

// Ack 确认消息
func (m *MessageQueue) Ack(messageID string) error {
	_, err := m.client.XAck(m.streamKey, m.consumerGroup, messageID)
	return err
}

// SendToDLQ 发送到死信队列
func (m *MessageQueue) SendToDLQ(msg *Message, reason string) error {
	data, _ := json.Marshal(msg)
	_, err := m.client.XAdd(m.dlqKey, map[string]interface{}{
		"original_id": msg.ID,
		"reason":      reason,
		"data":        string(data),
		"failed_at":   time.Now().Unix(),
	})
	return err
}

// AddToDelayQueue 添加到延迟队列
func (m *MessageQueue) AddToDelayQueue(msgID string, delay time.Duration) error {
	score := time.Now().Add(delay).Unix()
	return m.client.ZAdd(m.delayKey, redis.Z{
		Score:  float64(score),
		Member: msgID,
	})
}

// GetReadyDelayMessages 获取可执行的延迟任务
func (m *MessageQueue) GetReadyDelayMessages() ([]string, error) {
	now := time.Now().Unix()
	return m.client.ZRangeByScore(m.delayKey, "0", fmt.Sprintf("%d", now))
}

// RemoveDelayMessage 移除延迟任务
func (m *MessageQueue) RemoveDelayMessage(msgID string) error {
	return m.client.ZRem(m.delayKey, msgID)
}

// GetQueueLength 获取队列长度
func (m *MessageQueue) GetQueueLength() (int64, error) {
	return m.client.XLen(m.streamKey)
}

// UnreadCache 未读消息计数缓存
type UnreadCache struct {
	client *RedisClient
	ctx    context.Context
	key    string
}

// NewUnreadCache 创建未读计数缓存
func NewUnreadCache(client *RedisClient) *UnreadCache {
	return &UnreadCache{
		client: client,
		ctx:    context.Background(),
		key:    "user:%d:unread",
	}
}

// Incr 未读数+1
func (u *UnreadCache) Incr(userID int64, conversationID int64) error {
	key := fmt.Sprintf(u.key, userID)
	_, err := u.client.HIncrBy(key, fmt.Sprintf("%d", conversationID), 1)
	return err
}

// Decr 未读数-1
func (u *UnreadCache) Decr(userID int64, conversationID int64) error {
	key := fmt.Sprintf(u.key, userID)
	current, _ := u.client.HGet(key, fmt.Sprintf("%d", conversationID))
	if current == "0" || current == "" {
		return nil
	}
	_, err := u.client.HIncrBy(key, fmt.Sprintf("%d", conversationID), -1)
	return err
}

// Set 设置未读数
func (u *UnreadCache) Set(userID int64, conversationID int64, count int64) error {
	key := fmt.Sprintf(u.key, userID)
	return u.client.HSet(key, fmt.Sprintf("%d", conversationID), count)
}

// Get 获取未读数
func (u *UnreadCache) Get(userID int64, conversationID int64) (int64, error) {
	key := fmt.Sprintf(u.key, userID)
	value, err := u.client.HGet(key, fmt.Sprintf("%d", conversationID))
	if err != nil {
		return 0, nil
	}
	var count int64
	fmt.Sscanf(value, "%d", &count)
	return count, nil
}

// GetAll 获取用户所有未读数
func (u *UnreadCache) GetAll(userID int64) (map[int64]int64, error) {
	key := fmt.Sprintf(u.key, userID)
	data, err := u.client.HGetAll(key)
	if err != nil {
		return nil, err
	}
	result := make(map[int64]int64)
	for k, v := range data {
		var convID, count int64
		fmt.Sscanf(k, "%d", &convID)
		fmt.Sscanf(v, "%d", &count)
		result[convID] = count
	}
	return result, nil
}

// Clear 清除会话未读数
func (u *UnreadCache) Clear(userID int64, conversationID int64) error {
	key := fmt.Sprintf(u.key, userID)
	return u.client.HDel(key, fmt.Sprintf("%d", conversationID))
}

// ClearAll 清除用户所有未读数
func (u *UnreadCache) ClearAll(userID int64) error {
	key := fmt.Sprintf(u.key, userID)
	return u.client.Del(key)
}
