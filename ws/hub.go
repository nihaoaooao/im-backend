package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

// ============ 常量定义 ============
const (
	// 心跳间隔（秒）
	HeartbeatInterval = 30
	// 超时时间（秒）
	PingTimeout = 90
	// 单用户最大连接数
	MaxConnectionsPerUser = 5
	// 分片锁数量
	ShardCount = 64
	// 连接超时
	WriteWait = 10 * time.Second
	// 读超时
	ReadDeadline = 60 * time.Second
	// 消息队列大小
	SendChanSize = 256
)

// ============ Hub 结构体 ============

// Hub 消息中心
type Hub struct {
	// 64 分片锁 - 每个分片存储用户ID到客户端映射
	shards     [ShardCount]map[int64]map[*Client]bool
	shardLocks [ShardCount]sync.RWMutex

	// 注册通道
	register chan *Client
	// 注销通道
	unregister chan *Client
	// 广播消息通道
	broadcast chan *BroadcastMessage

	// Redis 客户端
	redis *redis.Client

	// 配置
	heartbeat int // 秒
	maxConns  int
}

// BroadcastMessage 广播消息结构
type BroadcastMessage struct {
	Message    interface{}
	TargetID   int64
	TargetType string // "user" 或 "group"
}

// ============ Client 结构体 ============

// Client WebSocket 客户端
type Client struct {
	id         string
	userID     int64
	conn       *websocket.Conn
	send       chan []byte
	hub        *Hub
	createdAt  time.Time
	lastActive time.Time // 最后活跃时间
}

// Message WebSocket 消息
type Message struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// ============ WebSocket 升级器 ============

// upgrader WebSocket 升级器
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // 生产环境应检查 Origin
	},
}

// ============ 初始化方法 ============

// NewHub 创建新的 Hub
func NewHub(redisClient *redis.Client) *Hub {
	hub := &Hub{
		register:    make(chan *Client, 10000),
		unregister:  make(chan *Client, 10000),
		broadcast:   make(chan *BroadcastMessage, 50000),
		redis:       redisClient,
		heartbeat:   HeartbeatInterval,
		maxConns:    MaxConnectionsPerUser,
	}

	// 初始化 64 个分片
	for i := 0; i < ShardCount; i++ {
		hub.shards[i] = make(map[int64]map[*Client]bool)
	}

	return hub
}

// ============ FNV-1a 分片算法 ============

// fnv32a FNV-1a 32bit 哈希算法
func fnv32a(key string) uint32 {
	hash := uint32(2166136261)
	for i := 0; i < len(key); i++ {
		hash ^= uint32(key[i])
		hash *= 16777619
	}
	return hash
}

// getShard 获取分片索引
func getShard(userID int64) int {
	key := fmt.Sprintf("user:%d", userID)
	return int(fnv32a(key) % ShardCount)
}

// ============ Hub 运行主循环 ============

// Run 运行 Hub
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.addClient(client)

		case client := <-h.unregister:
			h.removeClient(client)

		case msg := <-h.broadcast:
			if msg.TargetType == "user" {
				h.SendToUser(msg.TargetID, msg.Message)
			} else if msg.TargetType == "group" {
				h.SendToGroup(msg.TargetID, msg.Message)
			}
		}
	}
}

// ============ 客户端管理 ============

// addClient 添加客户端
func (h *Hub) addClient(client *Client) {
	shard := getShard(client.userID)

	h.shardLocks[shard].Lock()
	defer h.shardLocks[shard].Unlock()

	// 检查连接数限制
	if h.shards[shard][client.userID] != nil {
		currentConns := len(h.shards[shard][client.userID])
		if currentConns >= h.maxConns {
			// 超过最大连接数，断开最早的连接
			h.disconnectOldest(client.userID, shard)
		}
	}

	// 创建用户连接映射
	if h.shards[shard][client.userID] == nil {
		h.shards[shard][client.userID] = make(map[*Client]bool)
	}

	// 添加客户端
	h.shardLocks[shard].Unlock()

	// 更新 Redis（需要在锁外执行以避免死锁）
	h.addClientToRedis(client)

	h.shardLocks[shard].Lock()
	h.shards[shard][client.userID][client] = true
	h.shardLocks[shard].Unlock()

	log.Printf("[WebSocket] User %d connected, total connections: %d",
		client.userID, h.GetUserConnectionCount(client.userID))
}

// removeClient 移除客户端
func (h *Hub) removeClient(client *Client) {
	shard := getShard(client.userID)

	h.shardLocks[shard].Lock()
	defer h.shardLocks[shard].Unlock()

	if clients, ok := h.shards[shard][client.userID]; ok {
		if _, ok := clients[client]; ok {
			delete(clients, client)
			close(client.send)
			if len(clients) == 0 {
				delete(h.shards[shard], client.userID)
			}
		}
	}

	// 从 Redis 移除
	h.removeClientFromRedis(client)

	log.Printf("[WebSocket] User %d disconnected, remaining connections: %d",
		client.userID, h.GetUserConnectionCount(client.userID))
}

// disconnectOldest 断开最早的连接
func (h *Hub) disconnectOldest(userID int64, shard int) {
	if clients, ok := h.shards[shard][userID]; ok && len(clients) > 0 {
		var oldest *Client
		var oldestTime time.Time
		for client := range clients {
			if oldest == nil || client.createdAt.Before(oldestTime) {
				oldest = client
				oldestTime = client.createdAt
			}
		}
		if oldest != nil {
			delete(clients, oldest)
			close(oldest.send)
			oldest.conn.Close()
			h.removeClientFromRedis(oldest)
			log.Printf("[WebSocket] Disconnected oldest connection for user %d", userID)
		}
	}
}

// addClientToRedis 添加客户端到 Redis
func (h *Hub) addClientToRedis(client *Client) {
	if h.redis == nil {
		return
	}

	ctx := context.Background()
	// 存储连接信息
	connInfo := map[string]interface{}{
		"id":         client.id,
		"user_id":    client.userID,
		"created_at": client.createdAt.Unix(),
		"last_active": client.lastActive.Unix(),
	}
	connJSON, _ := json.Marshal(connInfo)

	// 使用 Hash 存储，key: user:{uid}:connections, field: conn_id
	h.redis.HSet(ctx, fmt.Sprintf("user:%d:connections", client.userID), client.id, connJSON)
	// 设置过期时间（24小时无活动自动清理）
	h.redis.Expire(ctx, fmt.Sprintf("user:%d:connections", client.userID), 24*time.Hour)

	// 记录在线状态
	h.redis.SAdd(ctx, "online_users", client.userID)
}

// removeClientFromRedis 从 Redis 移除客户端
func (h *Hub) removeClientFromRedis(client *Client) {
	if h.redis == nil {
		return
	}

	ctx := context.Background()
	// 移除连接
	h.redis.HDel(ctx, fmt.Sprintf("user:%d:connections", client.userID), client.id)

	// 检查是否还有其他连接
	count, _ := h.redis.HLen(ctx, fmt.Sprintf("user:%d:connections", client.userID)).Result()
	if count == 0 {
		// 移除在线状态
		h.redis.SRem(ctx, "online_users", client.userID)
	}
}

// UpdateClientActivity 更新客户端活跃时间
func (h *Hub) UpdateClientActivity(client *Client) {
	client.lastActive = time.Now()

	if h.redis != nil {
		ctx := context.Background()
		connInfo := map[string]interface{}{
			"id":         client.id,
			"user_id":    client.userID,
			"created_at": client.createdAt.Unix(),
			"last_active": client.lastActive.Unix(),
		}
		connJSON, _ := json.Marshal(connInfo)
		h.redis.HSet(ctx, fmt.Sprintf("user:%d:connections", client.userID), client.id, connJSON)
	}
}

// ============ 消息发送 ============

// SendToUser 发送消息给指定用户的所有连接
func (h *Hub) SendToUser(userID int64, message interface{}) {
	shard := getShard(userID)

	h.shardLocks[shard].RLock()
	clients := h.shards[shard][userID]
	h.shardLocks[shard].RUnlock()

	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("[WebSocket] Marshal message failed: %v", err)
		return
	}

	for client := range clients {
		select {
		case client.send <- data:
		default:
			// 通道满，关闭连接
			close(client.send)
			h.shardLocks[shard].Lock()
			delete(clients, client)
			h.shardLocks[shard].Unlock()
			client.conn.Close()
		}
	}
}

// SendToGroup 发送消息给群组所有成员
func (h *Hub) SendToGroup(groupID int64, message interface{}) {
	// TODO: 从数据库获取群成员列表
	// 这里需要注入 GroupService 来获取群成员
	log.Printf("[WebSocket] Broadcasting to group %d", groupID)
}

// Broadcast 广播消息
func (h *Hub) Broadcast(msg *BroadcastMessage) {
	select {
	case h.broadcast <- msg:
	default:
		log.Printf("[WebSocket] Broadcast channel full")
	}
}

// ============ 连接查询 ============

// GetUserConnectionCount 获取用户连接数
func (h *Hub) GetUserConnectionCount(userID int64) int {
	shard := getShard(userID)
	h.shardLocks[shard].RLock()
	defer h.shardLocks[shard].RUnlock()

	if clients, ok := h.shards[shard][userID]; ok {
		return len(clients)
	}
	return 0
}

// IsUserOnline 检查用户是否在线
func (h *Hub) IsUserOnline(userID int64) bool {
	return h.GetUserConnectionCount(userID) > 0
}

// GetOnlineUsers 获取所有在线用户
func (h *Hub) GetOnlineUsers() []int64 {
	var onlineUsers []int64

	h.redis.SMembers(context.Background(), "online_users").ScanSlice(&onlineUsers)
	return onlineUsers
}

// ============ WebSocket 处理器 ============

// HandleWebSocket 处理 WebSocket 连接
func HandleWebSocket(hub *Hub, c *gin.Context, jwtSecret string) {
	// 获取 token
	token := c.Query("token")
	if token == "" {
		token = c.PostForm("token")
	}

	if token == "" {
		c.JSON(400, gin.H{"code": 400, "msg": "缺少 token"})
		return
	}

	// 解析 JWT
	claims, err := parseToken(token, jwtSecret)
	if err != nil {
		c.JSON(401, gin.H{"code": 401, "msg": "无效的 token"})
		return
	}

	userID := int64(claims["user_id"].(float64))

	// 检查连接数限制
	currentCount := hub.GetUserConnectionCount(userID)
	if currentCount >= MaxConnectionsPerUser {
		c.JSON(429, gin.H{"code": 429, "msg": "已达到最大连接数"})
		return
	}

	// 升级为 WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("[WebSocket] Upgrade error: %v", err)
		return
	}

	// 生成客户端 ID
	clientID := fmt.Sprintf("conn_%d_%d_%s", userID, time.Now().UnixNano(), strconv.Itoa(currentCount+1))

	// 创建客户端
	client := &Client{
		id:         clientID,
		userID:     userID,
		conn:       conn,
		send:       make(chan []byte, SendChanSize),
		hub:        hub,
		createdAt:  time.Now(),
		lastActive: time.Now(),
	}

	// 注册客户端
	hub.register <- client

	// 启动读协程
	go client.writePump()

	// 启动写协程
	go client.readPump()
}

// ============ 客户端读写循环 ============

// readPump 读取客户端消息
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(512 * 1024) // 512KB
	// 设置读超时（90秒无消息断开）
	c.conn.SetReadDeadline(time.Now().Add(PingTimeout * time.Second))
	c.conn.SetPongHandler(func(string) error {
		// 收到 Pong，更新活跃时间
		c.hub.UpdateClientActivity(c)
		c.conn.SetReadDeadline(time.Now().Add(PingTimeout * time.Second))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[WebSocket] Read error: %v", err)
			}
			break
		}

		// 更新活跃时间
		c.hub.UpdateClientActivity(c)

		// 解析消息
		var msg Message
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("[WebSocket] Unmarshal error: %v", err)
			continue
		}

		// 处理消息
		c.handleMessage(msg)
	}
}

// writePump 发送消息给客户端
func (c *Client) writePump() {
	ticker := time.NewTicker(HeartbeatInterval * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(WriteWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			// 发送 Ping
			c.conn.SetWriteDeadline(time.Now().Add(WriteWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// ============ 消息处理 ============

// handleMessage 处理客户端消息
func (c *Client) handleMessage(msg Message) {
	switch msg.Type {
	case "message":
		// 消息发送确认
		data, _ := json.Marshal(Message{
			Type: "ack",
			Data: map[string]interface{}{
				"status": "ok",
				"time":   time.Now().Unix(),
			},
		})
		c.send <- data

	case "ping":
		// 心跳响应
		data, _ := json.Marshal(Message{
			Type: "pong",
			Data: map[string]interface{}{
				"time": time.Now().Unix(),
			},
		})
		c.send <- data

	case "read":
		// 已读回执处理
		// TODO: 转发给消息服务

	case "typing":
		// 正在输入状态
		// TODO: 通知对方

	case "recall":
		// 消息撤回
		// TODO: 转发给消息服务

	default:
		log.Printf("[WebSocket] Unknown message type: %s", msg.Type)
	}
}

// ============ JWT 解析 ============

// parseToken 解析 JWT Token
func parseToken(tokenString, secret string) (map[string]interface{}, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})

	if err != nil || !token.Valid {
		return nil, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid claims")
	}

	return claims, nil
}

// ============ HTTP API ============

// ServeWs 处理 WebSocket 连接（HTTP Server 模式）
func ServeWs(hub *Hub, w http.ResponseWriter, r *http.Request, jwtSecret string) {
	// 从 URL 参数获取 token
	query := r.URL.Query()
	token := query.Get("token")
	if token == "" {
		// 从 Header 获取
		token = r.Header.Get("Sec-WebSocket-Protocol")
		if token == "" {
			http.Error(w, "缺少 token", http.StatusUnauthorized)
			return
		}
	}

	// 解析 JWT
	claims, err := parseToken(token, jwtSecret)
	if err != nil {
		http.Error(w, "无效的 token", http.StatusUnauthorized)
		return
	}

	userID := int64(claims["user_id"].(float64))

	// 检查连接数限制
	currentCount := hub.GetUserConnectionCount(userID)
	if currentCount >= MaxConnectionsPerUser {
		http.Error(w, "已达到最大连接数", http.StatusTooManyRequests)
		return
	}

	// 升级为 WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WebSocket] Upgrade error: %v", err)
		return
	}

	// 生成客户端 ID
	clientID := fmt.Sprintf("conn_%d_%d_%s", userID, time.Now().UnixNano(), strconv.Itoa(currentCount+1))

	// 创建客户端
	client := &Client{
		id:         clientID,
		userID:     userID,
		conn:       conn,
		send:       make(chan []byte, SendChanSize),
		hub:        hub,
		createdAt:  time.Now(),
		lastActive: time.Now(),
	}

	// 注册客户端
	hub.register <- client

	// 启动读写协程
	go client.writePump()
	go client.readPump()
}

// ServeWebSocketWithAuth 带认证的 WebSocket 处理函数（可被 HTTP 服务器调用）
func ServeWebSocketWithAuth(hub *Hub, w http.ResponseWriter, r *http.Request, jwtSecret string, userID int64) {
	// 检查连接数限制
	currentCount := hub.GetUserConnectionCount(userID)
	if currentCount >= MaxConnectionsPerUser {
		http.Error(w, "已达到最大连接数", http.StatusTooManyRequests)
		return
	}

	// 升级为 WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WebSocket] Upgrade error: %v", err)
		return
	}

	// 生成客户端 ID
	clientID := fmt.Sprintf("conn_%d_%d_%s", userID, time.Now().UnixNano(), strconv.Itoa(currentCount+1))

	// 创建客户端
	client := &Client{
		id:         clientID,
		userID:     userID,
		conn:       conn,
		send:       make(chan []byte, SendChanSize),
		hub:        hub,
		createdAt:  time.Now(),
		lastActive: time.Now(),
	}

	// 注册客户端
	hub.register <- client

	// 启动读写协程
	go client.writePump()
	go client.readPump()
}

// HandleWebSocketHTTP 处理 WebSocket 连接（通过 HTTP 路由）
func HandleWebSocketHTTP(hub *Hub, jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取 token
		token := c.Query("token")
		if token == "" {
			token = c.PostForm("token")
		}

		if token == "" {
			c.JSON(400, gin.H{"code": 400, "msg": "缺少 token"})
			return
		}

		// 解析 JWT
		claims, err := parseToken(token, jwtSecret)
		if err != nil {
			c.JSON(401, gin.H{"code": 401, "msg": "无效的 token"})
			return
		}

		userID := int64(claims["user_id"].(float64))

		// 检查连接数限制
		currentCount := hub.GetUserConnectionCount(userID)
		if currentCount >= MaxConnectionsPerUser {
			c.JSON(429, gin.H{"code": 429, "msg": "已达到最大连接数"})
			return
		}

		// 升级为 WebSocket
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Printf("[WebSocket] Upgrade error: %v", err)
			return
		}

		// 生成客户端 ID
		clientID := fmt.Sprintf("conn_%d_%d_%s", userID, time.Now().UnixNano(), strconv.Itoa(currentCount+1))

		// 创建客户端
		client := &Client{
			id:         clientID,
			userID:     userID,
			conn:       conn,
			send:       make(chan []byte, SendChanSize),
			hub:        hub,
			createdAt:  time.Now(),
			lastActive: time.Now(),
		}

		// 注册客户端
		hub.register <- client

		// 启动读写协程
		go client.writePump()
		go client.readPump()
	}
}

// ============ 辅助函数 ============

// GetUserConnections 获取用户所有连接信息
func (h *Hub) GetUserConnections(userID int64) []map[string]interface{} {
	if h.redis == nil {
		return nil
	}

	ctx := context.Background()
	result, err := h.redis.HGetAll(ctx, fmt.Sprintf("user:%d:connections", userID)).Result()
	if err != nil {
		return nil
	}

	var connections []map[string]interface{}
	for id, data := range result {
		var connInfo map[string]interface{}
		if err := json.Unmarshal([]byte(data), &connInfo); err == nil {
			connInfo["id"] = id
			connections = append(connections, connInfo)
		}
	}
	return connections
}

// SendPushMessage 发送推送消息
func (h *Hub) SendPushMessage(targetID int64, msgType string, content interface{}) {
	msg := Message{
		Type: msgType,
		Data: content,
	}
	h.SendToUser(targetID, msg)
}

// BroadcastToGroup 群发消息到群组
func (h *Hub) BroadcastToGroup(groupID int64, msgType string, content interface{}) {
	msg := Message{
		Type: msgType,
		Data: content,
	}
	h.Broadcast(&BroadcastMessage{
		Message:    msg,
		TargetID:   groupID,
		TargetType: "group",
	})
}

// GetStats 获取 WebSocket 统计信息
func (h *Hub) GetStats() map[string]interface{} {
	totalConnections := 0
	onlineUsers := 0

	for i := 0; i < ShardCount; i++ {
		h.shardLocks[i].RLock()
		for userID, clients := range h.shards[i] {
			totalConnections += len(clients)
			if len(clients) > 0 {
				onlineUsers++
			}
			_ = userID // 避免未使用警告
		}
		h.shardLocks[i].RUnlock()
	}

	return map[string]interface{}{
		"total_connections": totalConnections,
		"online_users":      onlineUsers,
		"shard_count":       ShardCount,
	}
}

// SendMessageToUserByID 通过用户ID发送消息
func SendMessageToUserByID(hub *Hub, userID int64, message interface{}) {
	hub.SendToUser(userID, message)
}

// SendMessageToUsers 批量发送消息给多个用户
func SendMessageToUsers(hub *Hub, userIDs []int64, message interface{}) {
	for _, userID := range userIDs {
		hub.SendToUser(userID, message)
	}
}

// HandleWebSocketMessage 处理 WebSocket 消息请求
func HandleWebSocketMessage(hub *Hub, c *gin.Context) {
	var req struct {
		TargetID   int64       `json:"target_id" binding:"required"`
		TargetType string      `json:"target_type" binding:"required"` // user 或 group
		MsgType    string      `json:"msg_type" binding:"required"`
		Data       interface{} `json:"data"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 400, "msg": "参数错误: " + err.Error()})
		return
	}

	msg := Message{
		Type: req.MsgType,
		Data: req.Data,
	}

	if req.TargetType == "user" {
		hub.SendToUser(req.TargetID, msg)
	} else if req.TargetType == "group" {
		hub.BroadcastToGroup(req.TargetID, req.MsgType, req.Data)
	}

	c.JSON(200, gin.H{"code": 0, "msg": "消息已发送"})
}

// HandleGetOnlineStatus 获取在线状态
func HandleGetOnlineStatus(hub *Hub, c *gin.Context) {
	var req struct {
		UserIDs []int64 `json:"user_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 400, "msg": "参数错误"})
		return
	}

	status := make(map[int64]bool)
	for _, userID := range req.UserIDs {
		status[userID] = hub.IsUserOnline(userID)
	}

	c.JSON(200, gin.H{"code": 0, "data": status})
}

// HandleGetUserConnections 获取用户连接详情
func HandleGetUserConnections(hub *Hub, c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil {
		c.JSON(400, gin.H{"code": 400, "msg": "无效的用户ID"})
		return
	}

	connections := hub.GetUserConnections(userID)
	c.JSON(200, gin.H{"code": 0, "data": connections})
}

// HandleGetStats 获取 WebSocket 统计信息
func HandleGetStats(hub *Hub, c *gin.Context) {
	stats := hub.GetStats()
	c.JSON(200, gin.H{"code": 0, "data": stats})
}
