package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"im-backend/model"

	"github.com/redis/go-redis/v9"
)

// ============ 常量定义 ============

const (
	// Stream keys
	StreamKey           = "message:queue"
	DLQKey             = "message:queue:dlq"
	DelayQueueKey      = "message:delay:queue"
	PendingQueueKey    = "message:pending"

	// Consumer group
	ConsumerGroup = "im-consumer-group"

	// 重试配置
	MaxRetries      = 3
	RetryDelay      = 1 * time.Second
	MaxRetryDelay   = 30 * time.Second

	// 并发配置
	DefaultConcurrency = 100
	BatchSize         = 100

	// 超时配置
	BlockTimeout    = 5 * time.Second
	ConsumerTimeout = 30 * time.Second
)

// ============ 消息队列结构 ============

// MessageQueue Redis 消息队列
type MessageQueue struct {
	redis         *redis.Client
	consumerGroup string
	streamKey     string
	dlqKey        string
	delayKey      string
	pendingKey    string

	// 配置
	maxRetries    int
	retryDelay    time.Duration
	maxRetryDelay time.Duration
}

// NewMessageQueue 创建消息队列
func NewMessageQueue(redisClient *redis.Client) *MessageQueue {
	return &MessageQueue{
		redis:         redisClient,
		consumerGroup: ConsumerGroup,
		streamKey:     StreamKey,
		dlqKey:        DLQKey,
		delayKey:      DelayQueueKey,
		pendingKey:    PendingQueueKey,
		maxRetries:    MaxRetries,
		retryDelay:    RetryDelay,
		maxRetryDelay: MaxRetryDelay,
	}
}

// Init 初始化队列
func (q *MessageQueue) Init() error {
	ctx := context.Background()

	// 创建主 stream（如果不存在）
	exists, err := q.redis.Exists(ctx, q.streamKey).Result()
	if err != nil {
		return fmt.Errorf("failed to check stream: %w", err)
	}
	if exists == 0 {
		// 创建一个空 stream
		q.redis.XAdd(ctx, &redis.XAddArgs{
			Stream: q.streamKey,
			Values: map[string]interface{}{"init": "true"},
		})
		q.redis.XDel(ctx, q.streamKey, "0-0")
	}

	// 创建消费者组
	err = q.redis.XGroupCreateMkStream(ctx, q.streamKey, q.consumerGroup, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return fmt.Errorf("failed to create consumer group: %w", err)
	}

	// 初始化死信队列
	q.redis.XAdd(ctx, &redis.XAddArgs{
		Stream: q.dlqKey,
		Values: map[string]interface{}{"init": "true"},
	})
	q.redis.XDel(ctx, q.dlqKey, "0-0")

	log.Printf("[MessageQueue] Initialized: stream=%s, group=%s", q.streamKey, q.consumerGroup)
	return nil
}

// ============ 消息生产者 ============

// Enqueue 消息入队
func (q *MessageQueue) Enqueue(ctx context.Context, msg *model.Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	return q.redis.XAdd(ctx, &redis.XAddArgs{
		Stream: q.streamKey,
		Values: map[string]interface{}{
			"type":        "message",
			"data":        string(data),
			"created":     time.Now().Unix(),
			"retry_count": 0,
			"msg_id":     msg.MsgID,
			"sender_id":  msg.SenderID,
		},
	}).Err()
}

// EnqueueWithDelay 延迟消息入队
func (q *MessageQueue) EnqueueWithDelay(ctx context.Context, msg *model.Message, delay time.Duration) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// 存储消息数据
	msgKey := fmt.Sprintf("delay:msg:%s", msg.MsgID)
	q.redis.Set(ctx, msgKey, string(data), 24*time.Hour)

	// 添加到延迟队列
	score := time.Now().Add(delay).Unix()
	return q.redis.ZAdd(ctx, q.delayKey, redis.Z{
		Score:  float64(score),
		Member: msg.MsgID,
	}).Err()
}

// EnqueueBatch 批量入队
func (q *MessageQueue) EnqueueBatch(ctx context.Context, msgs []*model.Message) error {
	if len(msgs) == 0 {
		return nil
	}

	cmds := make([]*redis.StringCmd, 0, len(msgs))
	pipe := q.redis.Pipeline()

	for _, msg := range msgs {
		data, err := json.Marshal(msg)
		if err != nil {
			continue
		}
		cmd := pipe.XAdd(ctx, &redis.XAddArgs{
			Stream: q.streamKey,
			Values: map[string]interface{}{
				"type":        "message",
				"data":        string(data),
				"created":     time.Now().Unix(),
				"retry_count": 0,
				"msg_id":     msg.MsgID,
				"sender_id":  msg.SenderID,
			},
		})
		cmds = append(cmds, cmd)
	}

	_, err := pipe.Exec(ctx)
	return err
}

// ============ 消息消费者 ============

// Consumer 消费者
type Consumer struct {
	queue       *MessageQueue
	name        string
	handler     MessageHandler
	concurrency int
	channel     chan []byte
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
	running     bool
	mu          sync.RWMutex

	// 统计
	processedCount int64
	errorCount    int64
}

// MessageHandler 消息处理器接口
type MessageHandler interface {
	HandleMessage(msg *model.Message) error
}

// HandlerFunc 函数式处理器
type HandlerFunc func(msg *model.Message) error

func (f HandlerFunc) HandleMessage(msg *model.Message) error {
	return f(msg)
}

// NewConsumer 创建消费者
func NewConsumer(queue *MessageQueue, name string, handler MessageHandler, concurrency int) *Consumer {
	ctx, cancel := context.WithCancel(context.Background())

	if concurrency <= 0 {
		concurrency = DefaultConcurrency
	}

	return &Consumer{
		queue:       queue,
		name:        name,
		handler:     handler,
		concurrency: concurrency,
		channel:     make(chan []byte, concurrency*10),
		ctx:         ctx,
		cancel:      cancel,
		running:     false,
	}
}

// Run 启动消费者
func (c *Consumer) Run() {
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return
	}
	c.running = true
	c.mu.Unlock()

	// 启动工作协程
	for i := 0; i < c.concurrency; i++ {
		c.wg.Add(1)
		go c.worker(i)
	}

	// 启动消费协程
	go c.consume()

	log.Printf("[Consumer] %s started with %d workers", c.name, c.concurrency)
}

// worker 工作协程
func (c *Consumer) worker(id int) {
	defer c.wg.Done()

	for {
		select {
		case <-c.ctx.Done():
			return

		case data := <-c.channel:
			var msg model.Message
			if err := json.Unmarshal(data, &msg); err != nil {
				log.Printf("[Consumer] Failed to unmarshal message: %v", err)
				continue
			}

			if err := c.handler.HandleMessage(&msg); err != nil {
				log.Printf("[Consumer] Failed to handle message: %v", err)
				// 这里不重试，由 Consume 方法处理
			}
		}
	}
}

// consume 消费消息
func (c *Consumer) consume() {
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			messages, err := c.queue.Dequeue(c.ctx, c.name, BatchSize)
			if err != nil {
				if err != redis.Nil {
					log.Printf("[Consumer] Dequeue error: %v", err)
				}
				time.Sleep(100 * time.Millisecond)
				continue
			}

			for _, msg := range messages {
				c.processMessage(msg)
			}
		}
	}
}

// processMessage 处理单条消息
func (c *Consumer) processMessage(msg redis.XMessage) {
	data, ok := msg.Values["data"].(string)
	if !ok {
		log.Printf("[Consumer] Invalid message format")
		return
	}

	// 获取重试计数
	retryCount := 0
	if rc, ok := msg.Values["retry_count"].(string); ok {
		fmt.Sscanf(rc, "%d", &retryCount)
	}

	// 发送到处理通道
	select {
	case c.channel <- []byte(data):
		// 确认处理完成
		c.queue.Ack(c.ctx, msg.ID)
		c.processedCount++
	default:
		// 通道满，放回pending
		log.Printf("[Consumer] Channel full, requeueing message: %s", msg.ID)
		c.queue.Requeue(msg)
	}
}

// Stop 停止消费者
func (c *Consumer) Stop() {
	c.mu.Lock()
	if !c.running {
		c.mu.Unlock()
		return
	}
	c.running = false
	c.mu.Unlock()

	c.cancel()
	c.wg.Wait()
	close(c.channel)

	log.Printf("[Consumer] %s stopped, processed: %d, errors: %d",
		c.name, c.processedCount, c.errorCount)
}

// GetStats 获取消费者统计
func (c *Consumer) GetStats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return map[string]interface{}{
		"name":             c.name,
		"running":          c.running,
		"concurrency":      c.concurrency,
		"processed_count":   c.processedCount,
		"error_count":      c.errorCount,
		"channel_len":      len(c.channel),
		"channel_capacity": cap(c.channel),
	}
}

// ============ 出队和确认 ============

// Dequeue 出队（批量）
func (q *MessageQueue) Dequeue(ctx context.Context, consumer string, count int) ([]redis.XMessage, error) {
	result, err := q.redis.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    q.consumerGroup,
		Consumer: consumer,
		Streams:  []string{q.streamKey, ">"},
		Count:    int64(count),
		Block:    BlockTimeout,
	}).Result()

	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}

	if len(result) == 0 {
		return nil, nil
	}

	return result[0].Messages, nil
}

// DequeuePending 获取pending消息（失败重试）
func (q *MessageQueue) DequeuePending(ctx context.Context, consumer string, count int) ([]redis.XMessage, error) {
	// 获取pending消息
	result, err := q.redis.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    q.consumerGroup,
		Consumer: consumer,
		Streams:  []string{q.streamKey, "0"},
		Count:    int64(count),
		Block:    2 * time.Second,
	}).Result()

	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}

	if len(result) == 0 {
		return nil, nil
	}

	return result[0].Messages, nil
}

// Ack 确认消息处理完成
func (q *MessageQueue) Ack(ctx context.Context, messageID string) error {
	return q.redis.XAck(ctx, q.streamKey, q.consumerGroup, messageID).Err()
}

// Requeue 重新入队（用于处理失败的消息）
func (q *MessageQueue) Requeue(msg redis.XMessage) error {
	ctx := context.Background()

	// 增加重试计数
	retryCount := 0
	if rc, ok := msg.Values["retry_count"].(string); ok {
		fmt.Sscanf(rc, "%d", &retryCount)
	}
	retryCount++

	// 如果超过最大重试次数，发送到死信队列
	if retryCount > q.maxRetries {
		return q.SendToDLQ(ctx, msg, fmt.Sprintf("max retries exceeded: %d", retryCount))
	}

	// 更新消息并重新入队
	msg.Values["retry_count"] = fmt.Sprintf("%d", retryCount)
	msg.Values["retried_at"] = time.Now().Unix()

	// 使用 XACK 确认原消息
	q.redis.XAck(ctx, q.streamKey, q.consumerGroup, msg.ID)

	// 重新入队
	return q.redis.XAdd(ctx, &redis.XAddArgs{
		Stream: q.streamKey,
		Values: msg.Values,
	}).Err()
}

// ============ 死信队列 ============

// SendToDLQ 发送到死信队列
func (q *MessageQueue) SendToDLQ(ctx context.Context, msg redis.XMessage, reason string) error {
	data, _ := json.Marshal(msg.Values)

	return q.redis.XAdd(ctx, &redis.XAddArgs{
		Stream: q.dlqKey,
		Values: map[string]interface{}{
			"original_id": msg.ID,
			"reason":      reason,
			"data":        string(data),
			"failed_at":   time.Now().Unix(),
		},
	}).Err()
}

// GetDLQMessages 获取死信队列消息
func (q *MessageQueue) GetDLQMessages(ctx context.Context, count int64) ([]redis.XMessage, error) {
	return q.redis.XRange(ctx, q.dlqKey, "-", "+").Result()
}

// ============ 延迟队列 ============

// ProcessDelayQueue 处理延迟队列
func (q *MessageQueue) ProcessDelayQueue(ctx context.Context) error {
	now := time.Now().Unix()

	// 获取可执行的任务
	result, err := q.redis.ZRangeByScore(ctx, q.delayKey, &redis.ZRangeBy{
		Min: "0",
		Max: fmt.Sprintf("%d", now),
	}).Result()
	if err != nil {
		return err
	}

	if len(result) == 0 {
		return nil
	}

	// 处理每个延迟消息
	for _, msgID := range result {
		// 获取消息数据
		msgKey := fmt.Sprintf("delay:msg:%s", msgID)
		data, err := q.redis.Get(ctx, msgKey).Result()
		if err != nil {
			log.Printf("[DelayQueue] Failed to get message %s: %v", msgID, err)
			continue
		}

		// 反序列化消息
		var msg model.Message
		if err := json.Unmarshal([]byte(data), &msg); err != nil {
			log.Printf("[DelayQueue] Failed to unmarshal message %s: %v", msgID, err)
			continue
		}

		// 重新入队
		if err := q.Enqueue(ctx, &msg); err != nil {
			log.Printf("[DelayQueue] Failed to requeue message %s: %v", msgID, err)
			continue
		}

		// 从延迟队列移除
		q.redis.ZRem(ctx, q.delayKey, msgID)
		q.redis.Del(ctx, msgKey)

		log.Printf("[DelayQueue] Processed delayed message: %s", msgID)
	}

	return nil
}

// AddToDelayQueue 添加到延迟队列
func (q *MessageQueue) AddToDelayQueue(ctx context.Context, msgID string, delay time.Duration) error {
	score := time.Now().Add(delay).Unix()
	return q.redis.ZAdd(ctx, q.delayKey, redis.Z{
		Score:  float64(score),
		Member: msgID,
	}).Err()
}

// ============ 队列管理 ============

// GetQueueLength 获取队列长度
func (q *MessageQueue) GetQueueLength(ctx context.Context) (int64, error) {
	return q.redis.XLen(ctx, q.streamKey).Result()
}

// GetPendingCount 获取pending消息数
func (q *MessageQueue) GetPendingCount(ctx context.Context) (int64, error) {
	pending, err := q.redis.XPending(ctx, q.streamKey, q.consumerGroup).Result()
	if err != nil {
		return 0, err
	}
	return pending.Count, nil
}

// GetConsumerInfo 获取消费者信息
func (q *MessageQueue) GetConsumerInfo(ctx context.Context) ([]map[string]interface{}, error) {
	info, err := q.redis.XInfoConsumers(ctx, q.streamKey, q.consumerGroup).Result()
	if err != nil {
		return nil, err
	}
	var result []map[string]interface{}
	for _, c := range info {
		result = append(result, map[string]interface{}{
			"name":    c.Name,
			"pending": c.Pending,
			"idle":    c.Idle,
		})
	}
	return result, nil
}

// ClearQueue 清空队列
func (q *MessageQueue) ClearQueue(ctx context.Context) error {
	// 删除 stream
	err := q.redis.Del(ctx, q.streamKey).Err()
	if err != nil {
		return err
	}

	// 重新创建 stream 和消费者组
	return q.Init()
}

// ============ 独立消费者服务 ============

// ConsumerService 消费者服务
type ConsumerService struct {
	queue      *MessageQueue
	consumers  map[string]*Consumer
	handler    MessageHandler
	concurrency int
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
	running    bool
	mu         sync.RWMutex
}

// NewConsumerService 创建消费者服务
func NewConsumerService(redisClient *redis.Client, handler MessageHandler, concurrency int) *ConsumerService {
	ctx, cancel := context.WithCancel(context.Background())

	return &ConsumerService{
		queue:       NewMessageQueue(redisClient),
		consumers:   make(map[string]*Consumer),
		handler:     handler,
		concurrency: concurrency,
		ctx:         ctx,
		cancel:      cancel,
		running:     false,
	}
}

// Start 启动消费者服务
func (s *ConsumerService) Start(consumerNames []string) error {
	// 初始化队列
	if err := s.queue.Init(); err != nil {
		return fmt.Errorf("failed to init queue: %w", err)
	}

	s.mu.Lock()
	s.running = true
	s.mu.Unlock()

	// 为每个消费者创建实例
	for _, name := range consumerNames {
		consumer := NewConsumer(s.queue, name, s.handler, s.concurrency)
		consumer.Run()
		s.consumers[name] = consumer
	}

	// 启动延迟队列处理器
	go s.processDelayQueue()

	log.Printf("[ConsumerService] Started with %d consumers", len(consumerNames))
	return nil
}

// processDelayQueue 处理延迟队列
func (s *ConsumerService) processDelayQueue() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			if err := s.queue.ProcessDelayQueue(s.ctx); err != nil {
				log.Printf("[ConsumerService] Error processing delay queue: %v", err)
			}
		}
	}
}

// Stop 停止消费者服务
func (s *ConsumerService) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	s.mu.Unlock()

	s.cancel()

	for _, consumer := range s.consumers {
		consumer.Stop()
	}

	log.Printf("[ConsumerService] Stopped")
}

// GetStats 获取统计信息
func (s *ConsumerService) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := make(map[string]interface{})
	for name, consumer := range s.consumers {
		stats[name] = consumer.GetStats()
	}

	// 添加队列统计
	ctx := context.Background()
	queueLen, _ := s.queue.GetQueueLength(ctx)
	pendingCount, _ := s.queue.GetPendingCount(ctx)

	stats["queue"] = map[string]interface{}{
		"length":        queueLen,
		"pending_count": pendingCount,
	}

	return stats
}

// ============ 便捷函数 ============

// SimpleHandler 简单消息处理器
type SimpleHandler struct {
	handleFunc func(msg *model.Message) error
}

func (h *SimpleHandler) HandleMessage(msg *model.Message) error {
	return h.handleFunc(msg)
}

// NewSimpleHandler 创建简单处理器
func NewSimpleHandler(f func(msg *model.Message) error) MessageHandler {
	return &SimpleHandler{handleFunc: f}
}
