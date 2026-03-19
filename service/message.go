package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"im-backend/model"
	"im-backend/ws"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type MessageService struct {
	db    *gorm.DB
	redis *redis.Client
	hub   *ws.Hub
}

func NewMessageService(db *gorm.DB, redis *redis.Client) *MessageService {
	return &MessageService{db: db, redis: redis}
}

// SetHub 设置 WebSocket Hub
func (s *MessageService) SetHub(hub *ws.Hub) {
	s.hub = hub
}

// GetConversations 获取会话列表
func (s *MessageService) GetConversations(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var members []model.ConversationMember
	s.db.Where("user_id = ?", userID).Find(&members)

	var conversations []gin.H
	for _, m := range members {
		var conv model.Conversation
		if err := s.db.First(&conv, m.ConversationID).Error; err != nil {
			continue
		}

		conversations = append(conversations, gin.H{
			"id":                conv.ID,
			"type":              conv.Type,
			"name":              conv.Name,
			"lastMsgContent":   conv.LastMsgContent,
			"lastMsgTime":       conv.LastMsgTime,
			"unreadCount":       m.UnreadCount,
		})
	}

	c.JSON(200, gin.H{"code": 0, "data": conversations})
}

// CreateConversation 创建会话
func (s *MessageService) CreateConversation(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req struct {
		Type      string `json:"type" binding:"required"` // private, group
		Name      string `json:"name"`
		MemberIDs []int64 `json:"memberIds"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 400, "msg": "参数错误"})
		return
	}

	// 私聊会话
	if req.Type == "private" && len(req.MemberIDs) > 0 {
		// 检查是否已存在私聊会话
		var existConv model.ConversationMember
		result := s.db.Where("conversation_id IN (?) AND user_id = ?",
			s.db.Model(&model.ConversationMember{}).Where("user_id = ?", req.MemberIDs[0]),
			userID,
		).First(&existConv)

		if result.Error == nil {
			c.JSON(200, gin.H{"code": 0, "data": gin.H{"conversationId": existConv.ConversationID}})
			return
		}

		// 创建新会话
		conv := model.Conversation{
			Type:      "private",
			CreatorID: userID.(int64),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		s.db.Create(&conv)

		// 添加成员
		s.db.Create(&model.ConversationMember{
			ConversationID: conv.ID,
			UserID:         userID.(int64),
			Role:           "member",
			JoinedAt:       time.Now(),
		})
		s.db.Create(&model.ConversationMember{
			ConversationID: conv.ID,
			UserID:         req.MemberIDs[0],
			Role:           "member",
			JoinedAt:       time.Now(),
		})

		c.JSON(200, gin.H{"code": 0, "data": gin.H{"conversationId": conv.ID}})
		return
	}

	// 群聊会话（群组服务处理）
	c.JSON(400, gin.H{"code": 400, "msg": "请使用群组接口创建群聊"})
}

// SendMessage 发送消息
func (s *MessageService) SendMessage(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req struct {
		ConversationID int64  `json:"conversationId" binding:"required"`
		Content        string `json:"content" binding:"required"`
		ContentType    string `json:"contentType"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 400, "msg": "参数错误"})
		return
	}

	if req.ContentType == "" {
		req.ContentType = "text"
	}

	// 生成消息 ID
	msgID := generateMsgID()

	// 创建消息
	msg := model.Message{
		MsgID:          msgID,
		ConversationID: req.ConversationID,
		SenderID:       userID.(int64),
		Content:        req.Content,
		ContentType:    req.ContentType,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := s.db.Create(&msg).Error; err != nil {
		c.JSON(500, gin.H{"code": 500, "msg": "发送失败"})
		return
	}

	// 更新会话最后消息
	s.db.Model(&model.Conversation{}).Where("id = ?", req.ConversationID).Updates(map[string]interface{}{
		"last_msg_id":      msg.ID,
		"last_msg_content": req.Content,
		"last_msg_time":    msg.CreatedAt,
	})

	// 获取会话成员
	var members []model.ConversationMember
	s.db.Where("conversation_id = ?", req.ConversationID).Find(&members)

	// 推送消息给在线成员
	for _, m := range members {
		if m.UserID == userID.(int64) {
			continue
		}

		// 增加未读计数
		s.db.Model(&model.ConversationMember{}).
			Where("conversation_id = ? AND user_id = ?", req.ConversationID, m.UserID).
			Update("unread_count", gorm.Expr("unread_count + ?", 1))

		// 通过 WebSocket 推送
		if s.hub != nil {
			s.hub.SendToUser(m.UserID, ws.Message{
				Type: "message",
				Data: msg,
			})
		}
	}

	c.JSON(200, gin.H{
		"code": 0,
		"data": gin.H{
			"messageId": msg.ID,
			"msgId":     msg.MsgID,
			"timestamp": msg.CreatedAt.UnixMilli(),
		},
	})
}

// GetHistory 获取历史消息
func (s *MessageService) GetHistory(c *gin.Context) {
	userID, _ := c.Get("user_id")

	conversationID, _ := strconv.ParseInt(c.Query("conversationId"), 10, 64)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	// 检查用户是否有权限访问该会话
	var member model.ConversationMember
	if err := s.db.Where("conversation_id = ? AND user_id = ?", conversationID, userID).First(&member).Error; err != nil {
		c.JSON(403, gin.H{"code": 403, "msg": "无权限访问"})
		return
	}

	var messages []model.Message
	s.db.Where("conversation_id = ?", conversationID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&messages)

	// 反转顺序（按时间正序）
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	c.JSON(200, gin.H{"code": 0, "data": messages})
}

// RevokeMessage 撤回消息
func (s *MessageService) RevokeMessage(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req struct {
		MessageID int64 `json:"messageId" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 400, "msg": "参数错误"})
		return
	}

	// 查询消息
	var msg model.Message
	if err := s.db.First(&msg, req.MessageID).Error; err != nil {
		c.JSON(404, gin.H{"code": 404, "msg": "消息不存在"})
		return
	}

	// 检查是否是发送者
	if msg.SenderID != userID.(int64) {
		c.JSON(403, gin.H{"code": 403, "msg": "只能撤回自己发送的消息"})
		return
	}

	// 检查是否在 2 分钟内
	if time.Since(msg.CreatedAt) > 2*time.Minute {
		c.JSON(400, gin.H{"code": 400, "msg": "只能撤回 2 分钟内的消息"})
		return
	}

	// 标记为已撤回
	s.db.Model(&msg).Update("is_recalled", true)

	// 广播撤回通知
	if s.hub != nil {
		var members []model.ConversationMember
		s.db.Where("conversation_id = ?", msg.ConversationID).Find(&members)

		for _, m := range members {
			s.hub.SendToUser(m.UserID, ws.Message{
				Type: "recall",
				Data: gin.H{"messageId": req.MessageID},
			})
		}
	}

	c.JSON(200, gin.H{"code": 0, "msg": "已撤回"})
}

// MarkAsRead 标记已读
func (s *MessageService) MarkAsRead(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req struct {
		MessageIDs []int64 `json:"messageIds" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 400, "msg": "参数错误"})
		return
	}

	// 标记已读
	for _, msgID := range req.MessageIDs {
		var msg model.Message
		if err := s.db.First(&msg, msgID).Error; err != nil {
			continue
		}

		// 写入已读记录
		read := model.MessageRead{
			MessageID:      msgID,
			UserID:         userID.(int64),
			ConversationID: msg.ConversationID,
			ReadAt:         time.Now(),
		}
		s.db.Create(&read)

		// 减少未读计数
		s.db.Model(&model.ConversationMember{}).
			Where("conversation_id = ? AND user_id = ?", msg.ConversationID, userID).
			Update("unread_count", gorm.Expr("GREATEST(unread_count - 1, 0)"))
	}

	c.JSON(200, gin.H{"code": 0, "msg": "已读"})
}

// UploadMedia 上传媒体文件（示例）
func (s *MessageService) UploadMedia(c *gin.Context) {
	// TODO: 实现文件上传到 OSS
	c.JSON(200, gin.H{
		"code": 0,
		"data": gin.H{
			"url": "https://example.com/upload/test.jpg",
		},
	})
}

// generateMsgID 生成消息唯一ID
func generateMsgID() string {
	return fmt.Sprintf("msg_%d_%d", time.Now().UnixMilli(), time.Now().Nanosecond()%1000)
}

// 消息入队
func (s *MessageService) EnqueueMessage(ctx context.Context, msg interface{}) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return s.redis.XAdd(ctx, &redis.XAddArgs{
		Stream: "message:queue",
		Values: map[string]interface{}{
			"data": string(data),
		},
	}).Err()
}

// 消息出队
func (s *MessageService) DequeueMessage(ctx context.Context) ([]redis.XMessage, error) {
	result, err := s.redis.XRead(ctx, &redis.XReadArgs{
		Streams: []string{"message:queue", "0"},
		Count:   1,
		Block:   5 * 1000 * 1000, // 5秒
	}).Result()
	if err != nil {
		return nil, err
	}
	return result[0].Messages, nil
}
