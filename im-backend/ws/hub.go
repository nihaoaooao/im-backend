package ws

import (
	"encoding/json"
	"sync"
	"time"
)

// Hub WebSocket Hub
type Hub struct {
	// 注册的用户
	Clients map[int64]map[*Client]bool

	// 注册请求
	Register chan *Client

	// 注销请求
	Unregister chan *Client

	// 广播消息
	Broadcast chan []byte

	// 互斥锁
	Mutex sync.RWMutex
}

// Client WebSocket 客户端
type Client struct {
	Hub  *Hub
	UserID int64
	Send  chan []byte
}

// Message 通用消息结构
type Message struct {
	Type    string          `json:"type"`
	Content json.RawMessage `json:"content"`
}

// ChatMessage 聊天消息
type ChatMessage struct {
	Type           string `json:"type"`
	MsgID          string `json:"msg_id"`
	ConversationID int64  `json:"conversation_id"`
	SenderID       int64  `json:"sender_id"`
	Content        string `json:"content"`
	ContentType    string `json:"content_type"`
	Timestamp      int64  `json:"timestamp"`
}

// MessageRecallNotice 消息撤回通知
type MessageRecallNotice struct {
	Type             string `json:"type"` // recall
	MsgID            string `json:"msg_id"`
	ConversationID   int64  `json:"conversation_id"`
	RevokerID        int64  `json:"revoker_id"`
	OriginalSenderID int64  `json:"original_sender_id"`
	RevokedAt        int64  `json:"revoked_at"`
}

// ReadReceiptNotice 已读回执通知
type ReadReceiptNotice struct {
	Type            string   `json:"type"` // read_receipt
	MessageID       int64    `json:"messageId"`
	ConversationID  int64    `json:"conversationId"`
	ReadBy          []int64  `json:"readBy"`
	ReadCount       int64    `json:"readCount"`
	TotalCount      int64    `json:"totalCount"`
}

// UserInfo 用户信息
type UserInfo struct {
	UserID   int64  `json:"userId"`
	Username string `json:"username,omitempty"`
	ReadAt   string `json:"readAt,omitempty"`
}

// MessageReadStatus 消息已读状态
type MessageReadStatus struct {
	MessageID       int64     `json:"messageId"`
	ReadBy          []UserInfo `json:"readBy"`
	ReadCount       int64     `json:"readCount"`
	UnreadBy        []UserInfo `json:"unreadBy"`
	UnreadCount     int64     `json:"unreadCount"`
}

// NewHub 创建新的Hub
func NewHub() *Hub {
	return &Hub{
		Clients:    make(map[int64]map[*Client]bool),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Broadcast:  make(chan []byte, 256),
	}
}

// Run 运行Hub
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.Mutex.Lock()
			if h.Clients[client.UserID] == nil {
				h.Clients[client.UserID] = make(map[*Client]bool)
			}
			h.Clients[client.UserID][client] = true
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
			h.Mutex.RLock()
			for _, clients := range h.Clients {
				for client := range clients {
					select {
					case client.Send <- message:
					default:
						close(client.Send)
						delete(clients, client)
					}
				}
			}
			h.Mutex.RUnlock()
		}
	}
}

// SendToUser 发送消息给指定用户
func (h *Hub) SendToUser(userID int64, data interface{}) {
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

	for client := range clients {
		select {
		case client.Send <- msgBytes:
		default:
			close(client.Send)
			delete(clients, client)
		}
	}
}

// IsUserOnline 检查用户是否在线
func (h *Hub) IsUserOnline(userID int64) bool {
	h.Mutex.RLock()
	defer h.Mutex.RUnlock()

	clients, ok := h.Clients[userID]
	return ok && len(clients) > 0
}

// GetOnlineUserCount 获取在线用户数
func (h *Hub) GetOnlineUserCount() int {
	h.Mutex.RLock()
	defer h.Mutex.RUnlock()

	count := 0
	for _, clients := range h.Clients {
		count += len(clients)
	}
	return count
}

// InitHub 初始化全局Hub
var GlobalHub = NewHub()

// 启动Hub
func init() {
	go GlobalHub.Run()
}
