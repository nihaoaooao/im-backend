package service

import (
	"context"
	"sync"
	"time"

	"im-backend/config"
)

// RecallScheduler 消息撤回定时检查器
// 用于定期检查并标记超过撤回时限的消息
type RecallScheduler struct {
	messageService *MessageService
	interval       time.Duration
	stopCh         chan struct{}
	wg             sync.WaitGroup
	isRunning      bool
}

// NewRecallScheduler 创建撤回定时检查器
func NewRecallScheduler(messageService *MessageService, interval time.Duration) *RecallScheduler {
	return &RecallScheduler{
		messageService: messageService,
		interval:       interval,
		stopCh:         make(chan struct{}),
		isRunning:      false,
	}
}

// Start 启动定时检查器
func (r *RecallScheduler) Start() {
	if r.isRunning {
		config.Log.Printf("Warning: RecallScheduler is already running")
		return
	}

	r.isRunning = true
	r.wg.Add(1)

	go func() {
		defer r.wg.Done()
		r.run()
	}()

	config.Log.Printf("Info: RecallScheduler started, interval: %v", r.interval)
}

// Stop 停止定时检查器
func (r *RecallScheduler) Stop() {
	if !r.isRunning {
		return
	}

	close(r.stopCh)
	r.wg.Wait()
	r.isRunning = false

	config.Log.Printf("Info: RecallScheduler stopped")
}

// run 执行定时检查
func (r *RecallScheduler) run() {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-r.stopCh:
			return
		case <-ticker.C:
			r.checkAndRevoke()
		}
	}
}

// checkAndRevoke 检查并标记不可撤回的消息
func (r *RecallScheduler) checkAndRevoke() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 每次处理100条
	affected, err := r.messageService.RevokeMessageByTimePeriod(100)
	if err != nil {
		config.Log.Printf("Error: Failed to revoke expired messages, error: %v", err)
		return
	}

	if affected > 0 {
		config.Log.Printf("Info: Marked messages as non-revocable, count: %d", affected)
	}

	// 定期输出可撤回消息的统计信息
	recallableMessages, err := r.messageService.GetRecallableMessages(1000)
	if err != nil {
		config.Log.Printf("Error: Failed to get recallable messages, error: %v", err)
		return
	}

	config.Log.Printf("Debug: Recallable messages: %d", len(recallableMessages))

	_ = ctx
}

// IsRunning 检查是否正在运行
func (r *RecallScheduler) IsRunning() bool {
	return r.isRunning
}

// GetInterval 获取检查间隔
func (r *RecallScheduler) GetInterval() time.Duration {
	return r.interval
}

// SetInterval 设置检查间隔
func (r *RecallScheduler) SetInterval(interval time.Duration) {
	r.interval = interval
}
