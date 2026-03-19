package ws

import (
	"encoding/json"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// OptimizedHub 优化后的 WebSocket Hub
type OptimizedHub struct {
	// 注册的用户
	Clients map[int64]map[*Client]bool

	// 注册请求
	Register chan *Client

	// 注销请求
	Unregister chan *Client

	// 广播消息
	Broadcast chan []byte

	// 批量广播
	BatchBroadcast chan [][]byte

	// 互斥锁
	Mutex sync.RWMutex

	// 统计信息
	stats HubStats
}

// HubStats Hub 统计信息
type HubStats struct {
	TotalClients   int64
	TotalMessages  int64
	TotalBroadcast int64
	StartTime      time.Time
}

// NewOptimizedHub 创建优化后的 Hub
func NewOptimizedHub() *OptimizedHub {
	h := &OptimizedHub{
		Clients:       make(map[int64]map[*Client]bool),
		Register:      make(chan *Client, 1024),
		Unregister:    make(chan *Client, 1024),
		Broadcast:     make(chan []byte, 4096),
		BatchBroadcast: make(chan [][]byte, 256),
		stats: HubStats{
			StartTime: time.Now(),
		},
	}
	go h.run()
	go h.batchProcess()
	return h
}

// run 运行 Hub 主循环
func (h *OptimizedHub) run() {
	for {
		select {
		case client := <-h.Register:
			h.Mutex.Lock()
			if h.Clients[client.UserID] == nil {
				h.Clients[client.UserID] = make(map[*Client]bool)
			}
			h.Clients[client.UserID][client] = true
			atomic.AddInt64(&h.stats.TotalClients, 1)
			h.Mutex.Unlock()

		case client := <-h.Unregister:
			h.Mutex.Lock()
			if clients, ok := h.Clients[client.UserID]; ok {
				if _, ok := clients[client]; ok {
					delete(clients, client)
					close(client.Send)
					if len(clients) == 0 {
						delete(h.Clients, client.UserID)
					}
				}
			}
			h.Mutex.Unlock()

		case message := <-h.Broadcast:
			h.broadcastToAll(message)
			atomic.AddInt64(&h.stats.TotalBroadcast, 1)
		}
	}
}

// batchProcess 批量处理消息
func (h *OptimizedHub) batchProcess() {
	ticker := time.NewTicker(10 * time.Millisecond) // 10ms 批量处理
	defer ticker.Stop()

	var batch [][]byte
	for {
		select {
		case messages := <-h.BatchBroadcast:
			batch = append(batch, messages...)
		case <-ticker.C:
			if len(batch) > 0 {
				h.flushBatch(batch)
				batch = nil
			}
		}
	}
}

// flushBatch 刷新批量消息
func (h *OptimizedHub) flushBatch(messages [][]byte) {
	h.Mutex.RLock()
	defer h.Mutex.RUnlock()

	// 使用并发发送
	var wg sync.WaitGroup
	for _, clients := range h.Clients {
		for client := range clients {
			wg.Add(1)
			go func(c *Client, msgs [][]byte) {
				defer wg.Done()
				for _, msg := range msgs {
					select {
					case c.Send <- msg:
					default:
						// 队列满，关闭连接
						close(c.Send)
					}
				}
			}(client, messages)
		}
	}
	wg.Wait()
	atomic.AddInt64(&h.stats.TotalMessages, int64(len(messages)*len(h.Clients)))
}

// broadcastToAll 广播到所有客户端
func (h *OptimizedHub) broadcastToAll(message []byte) {
	h.Mutex.RLock()
	defer h.Mutex.RUnlock()

	for _, clients := range h.Clients {
		for client := range clients {
			select {
			case client.Send <- message:
			default:
				// 队列满，关闭连接
				close(client.Send)
			}
		}
	}
}

// SendToUser 发送消息给指定用户
func (h *OptimizedHub) SendToUser(userID int64, data interface{}) {
	h.Mutex.RLock()
	clients, ok := h.Clients[userID]
	h.Mutex.RUnlock()

	if !ok || len(clients) == 0 {
		return
	}

	msgBytes, err := json.Marshal(data)
	if err != nil {
		return
	}

	// 使用并发发送
	var wg sync.WaitGroup
	for client := range clients {
		wg.Add(1)
		go func(c *Client) {
			defer wg.Done()
			select {
			case c.Send <- msgBytes:
			default:
			}
		}(client)
	}
	wg.Wait()
}

// SendToUsers 批量发送消息给多个用户
func (h *OptimizedHub) SendToUsers(userIDs []int64, data interface{}) {
	msgBytes, err := json.Marshal(data)
	if err != nil {
		return
	}

	h.Mutex.RLock()
	defer h.Mutex.RUnlock()

	var wg sync.WaitGroup
	for _, userID := range userIDs {
		if clients, ok := h.Clients[userID]; ok {
			for client := range clients {
				wg.Add(1)
				go func(c *Client) {
					defer wg.Done()
					select {
					case c.Send <- msgBytes:
					default:
					}
				}(client)
			}
		}
	}
	wg.Wait()
}

// BroadcastToConversation 发送到会话
func (h *OptimizedHub) BroadcastToConversation(conversationID int64, excludeUserID int64, data interface{}) {
	msgBytes, err := json.Marshal(data)
	if err != nil {
		return
	}

	h.Mutex.RLock()
	defer h.Mutex.RUnlock()

	var wg sync.WaitGroup
	for userID, clients := range h.Clients {
		if userID == excludeUserID {
			continue
		}
		for client := range clients {
			wg.Add(1)
			go func(c *Client) {
				defer wg.Done()
				select {
				case c.Send <- msgBytes:
				default:
				}
			}(client)
		}
	}
	wg.Wait()
}

// IsUserOnline 检查用户是否在线
func (h *OptimizedHub) IsUserOnline(userID int64) bool {
	h.Mutex.RLock()
	defer h.Mutex.RUnlock()

	clients, ok := h.Clients[userID]
	return ok && len(clients) > 0
}

// GetOnlineUserCount 获取在线用户数
func (h *OptimizedHub) GetOnlineUserCount() int {
	h.Mutex.RLock()
	defer h.Mutex.RUnlock()

	count := 0
	for _, clients := range h.Clients {
		count += len(clients)
	}
	return count
}

// GetUserConnections 获取用户的所有连接
func (h *OptimizedHub) GetUserConnections(userID int64) []*Client {
	h.Mutex.RLock()
	defer h.Mutex.RUnlock()

	clients, ok := h.Clients[userID]
	if !ok {
		return nil
	}

	result := make([]*Client, 0, len(clients))
	for client := range clients {
		result = append(result, client)
	}
	return result
}

// GetStats 获取统计信息
func (h *OptimizedHub) GetStats() HubStats {
	return HubStats{
		TotalClients:   atomic.LoadInt64(&h.stats.TotalClients),
		TotalMessages:  atomic.LoadInt64(&h.stats.TotalMessages),
		TotalBroadcast: atomic.LoadInt64(&h.stats.TotalBroadcast),
		StartTime:      h.stats.StartTime,
	}
}

// GetMemoryUsage 获取内存使用情况
func (h *OptimizedHub) GetMemoryUsage() runtime.MemStats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m
}

// BatchSend 批量发送（用于消息聚合）
func (h *OptimizedHub) BatchSend(messages [][]byte) {
	h.BatchBroadcast <- messages
}

// OptimizedClient 优化后的客户端
type OptimizedClient struct {
	Hub       *OptimizedHub
	UserID    int64
	Send      chan []byte
	Connected time.Time
}

// NewOptimizedClient 创建优化后的客户端
func NewOptimizedClient(hub *OptimizedHub, userID int64) *OptimizedClient {
	return &OptimizedClient{
		Hub:       hub,
		UserID:    userID,
		Send:      make(chan []byte, 256), // 增加缓冲区
		Connected: time.Now(),
	}
}
