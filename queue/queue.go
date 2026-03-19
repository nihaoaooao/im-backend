package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"im-backend/model"
	"im-backend/repository"

	"github.com/redis/go-redis/v9"
)

// MessageQueue 消息队列
type MessageQueue struct {
	redis         *redis.Client
	consumerGroup string
	streamKey     string
	dlqKey        string // 死信队列
}

// NewMessageQueue 创建消息队列
func NewMessageQueue(redisClient *redis.Client) *MessageQueue {
	return &MessageQueue{
		redis:         redisClient,
		consumerGroup: "im-consumer-group",
		streamKey:     "message:queue",
		dlqKey:        "message:queue:dlq",
	}
}

// Init 初始化队列
func (q *MessageQueue) Init() error {
	ctx := context.Background()

	// 创建消费者组
	err := q.redis.XGroupCreateMkStream(ctx, q.streamKey, q.consumerGroup, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return fmt.Errorf("failed to create consumer group: %w", err)
	}

	log.Printf("Message queue initialized: stream=%s, group=%s", q.streamKey, q.consumerGroup)
	return nil
}

// Enqueue 入队
func (q *MessageQueue) Enqueue(ctx context.Context, msg *model.Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	return q.redis.XAdd(ctx, &redis.XAddArgs{
		Stream: q.streamKey,
		Values: map[string]interface{}{
			"type":    "message",
			"data":    string(data),
			"created": time.Now().Unix(),
		},
	}).Err()
}

// Dequeue 出队（批量）
func (q *MessageQueue) Dequeue(ctx context.Context, consumer string, count int) ([]redis.XMessage, error) {
	result, err := q.redis.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    q.consumerGroup,
		Consumer: consumer,
		Streams:  []string{q.streamKey, ">"},
		Count:    int64(count),
		Block:    5 * time.Second,
	}).Result()

	if err != nil {
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

// Consumer 消费者
type Consumer struct {
	queue       *MessageQueue
	name        string
	handler     MessageHandler
	concurrency int
	channel     chan []byte // 5万缓冲通道
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
}

// MessageHandler 消息处理器接口
type MessageHandler interface {
	HandleMessage(msg *model.Message) error
}

// NewConsumer 创建消费者
func NewConsumer(queue *MessageQueue, name string, handler MessageHandler, concurrency int) *Consumer {
	ctx, cancel := context.WithCancel(context.Background())

	return &Consumer{
		queue:       queue,
		name:        name,
		handler:     handler,
		concurrency: concurrency,
		channel:     make(chan []byte, 50000), // 5万缓冲通道
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Run 启动消费者（100+ 并发）
func (c *Consumer) Run() {
	// 启动多个 goroutine
	for i := 0; i < c.concurrency; i++ {
		c.wg.Add(1)
		go c.worker(i)
	}

	log.Printf("Consumer %s started with %d workers", c.name, c.concurrency)
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
				log.Printf("Failed to unmarshal message: %v", err)
				continue
			}

			if err := c.handler.HandleMessage(&msg); err != nil {
				log.Printf("Failed to handle message: %v", err)
			}
		}
	}
}

// Consume 消费消息
func (c *Consumer) Consume() {
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			messages, err := c.queue.Dequeue(c.ctx, c.name, 100) // 批量获取 100 条
			if err != nil {
				if err != redis.Nil {
					log.Printf("Dequeue error: %v", err)
				}
				time.Sleep(100 * time.Millisecond)
				continue
			}

			for _, msg := range messages {
				data, ok := msg.Values["data"].(string)
				if !ok {
					continue
				}

				// 发送到处理通道
				select {
				case c.channel <- []byte(data):
					// 确认处理完成
					c.queue.Ack(c.ctx, msg.ID)
				default:
					// 通道满，记录日志
					log.Printf("Channel full, message queued: %s", msg.ID)
				}
			}
		}
	}
}

// Stop 停止消费者
func (c *Consumer) Stop() {
	c.cancel()
	c.wg.Wait()
	log.Printf("Consumer %s stopped", c.name)
}

// DelayQueue 延迟队列（用于消息撤回）
type DelayQueue struct {
	redis   *redis.Client
	zsetKey string
}

// NewDelayQueue 创建延迟队列
func NewDelayQueue(redisClient *redis.Client) *DelayQueue {
	return &DelayQueue{
		redis:   redisClient,
		zsetKey: "message:delay:queue",
	}
}

// Add 添加延迟任务
func (d *DelayQueue) Add(ctx context.Context, messageID string, delay time.Duration) error {
	score := time.Now().Add(delay).Unix()
	return d.redis.ZAdd(ctx, d.zsetKey, redis.Z{
		Score:  float64(score),
		Member: messageID,
	}).Err()
}

// GetReady 获取可执行的任务
func (d *DelayQueue) GetReady(ctx context.Context) ([]string, error) {
	now := time.Now().Unix()
	result, err := d.redis.ZRangeByScore(ctx, d.zsetKey, &redis.ZRangeBy{
		Min: "0",
		Max: fmt.Sprintf("%d", now),
	}).Result()

	if err != nil {
		return nil, err
	}

	// 删除已取出的任务
	if len(result) > 0 {
		d.redis.ZRem(ctx, d.zsetKey, result)
	}

	return result, nil
}
