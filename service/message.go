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
		MemberIDs []int64 `json:"memberIds" binding:"required,min=1"`
		TargetID  int64  `json:"targetId"` // 用于私聊，目标用户ID
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 400, "msg": "参数错误: " + err.Error()})
		return
	}

	// 确定私聊目标用户ID
	targetID := req.TargetID
	if targetID == 0 && len(req.MemberIDs) > 0 {
		targetID = req.MemberIDs[0]
	}

	// 私聊会话
	if req.Type == "private" && targetID > 0 {
		// 不能与自己聊天
		if targetID == userID.(int64) {
			c.JSON(400, gin.H{"code": 400, "msg": "不能与自己聊天"})
			return
		}

		// 检查目标用户是否存在
		var targetUser model.User
		if err := s.db.First(&targetUser, targetID).Error; err != nil {
			c.JSON(404, gin.H{"code": 404, "msg": "用户不存在"})
			return
		}

		// 查找已存在的私聊会话
		// 找出两个用户共同参与的私聊会话
		var existConvs []model.Conversation
		s.db.Table("conversations").
			Where("type = ?", "private").
			Where("id IN (?)",
				s.db.Model(&model.ConversationMember{}).Where("user_id = ?", userID),
			).
			Where("id IN (?)",
				s.db.Model(&model.ConversationMember{}).Where("user_id = ?", targetID),
			).
			Find(&existConvs)

		// 如果存在，返回第一个
		if len(existConvs) > 0 {
			// 获取对方信息
			var member model.ConversationMember
			s.db.Where("conversation_id = ? AND user_id = ?", existConvs[0].ID, targetID).First(&member)

			c.JSON(200, gin.H{
				"code": 0,
				"data": gin.H{
					"conversationId": existConvs[0].ID,
					"targetId":       targetID,
					"targetName":     targetUser.Nickname,
					"targetAvatar":   targetUser.Avatar,
				},
			})
			return
		}

		// 创建新会话
		conv := model.Conversation{
			Type:        "private",
			Name:        targetUser.Nickname, // 私聊名称默认为对方昵称
			CreatorID:   userID.(int64),
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		if err := s.db.Create(&conv).Error; err != nil {
			c.JSON(500, gin.H{"code": 500, "msg": "创建会话失败"})
			return
		}

		// 添加成员（当前用户）
		if err := s.db.Create(&model.ConversationMember{
			ConversationID: conv.ID,
			UserID:         userID.(int64),
			Role:           "member",
			JoinedAt:       time.Now(),
		}).Error; err != nil {
			c.JSON(500, gin.H{"code": 500, "msg": "添加成员失败"})
			return
		}

		// 添加成员（目标用户）
		if err := s.db.Create(&model.ConversationMember{
			ConversationID: conv.ID,
			UserID:         targetID,
			Role:           "member",
			JoinedAt:       time.Now(),
		}).Error; err != nil {
			c.JSON(500, gin.H{"code": 500, "msg": "添加成员失败"})
			return
		}

		c.JSON(200, gin.H{
			"code": 0,
			"data": gin.H{
				"conversationId": conv.ID,
				"targetId":       targetID,
				"targetName":     targetUser.Nickname,
				"targetAvatar":   targetUser.Avatar,
			},
		})
		return
	}

	// 群聊会话（群组服务处理）
	c.JSON(400, gin.H{"code": 400, "msg": "请使用群组接口创建群聊"})
}

// SendMessage 发送消息 - [P2] 添加消息内容长度限制
func (s *MessageService) SendMessage(c *gin.Context) {
	userID, _ := c.Get("user_id")
	username, _ := c.Get("username")

	var req struct {
		ConversationID int64  `json:"conversationId" binding:"required"`
		Content        string `json:"content" binding:"required"`
		ContentType    string `json:"contentType"`
		Extra          string `json:"extra"` // 扩展信息 JSON 字符串
	}

	// [P2] 消息内容长度限制
	const MaxMessageLength = 5000 // 最大5000字符
	if len(req.Content) > MaxMessageLength {
		c.JSON(400, gin.H{"code": 400, "msg": fmt.Sprintf("消息内容过长，最大支持 %d 字符", MaxMessageLength)})
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 400, "msg": "参数错误: " + err.Error()})
		return
	}

	if req.ContentType == "" {
		req.ContentType = "text"
	}

	// 检查会话是否存在
	var conv model.Conversation
	if err := s.db.First(&conv, req.ConversationID).Error; err != nil {
		c.JSON(404, gin.H{"code": 404, "msg": "会话不存在"})
		return
	}

	// 检查用户是否是会话成员
	var member model.ConversationMember
	if err := s.db.Where("conversation_id = ? AND user_id = ?", req.ConversationID, userID).First(&member).Error; err != nil {
		c.JSON(403, gin.H{"code": 403, "msg": "无权限在此会话发言"})
		return
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
		Extra:          req.Extra,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := s.db.Create(&msg).Error; err != nil {
		c.JSON(500, gin.H{"code": 500, "msg": "发送失败: " + err.Error()})
		return
	}

	// 更新会话最后消息
	s.db.Model(&model.Conversation{}).Where("id = ?", req.ConversationID).Updates(map[string]interface{}{
		"last_msg_id":       msg.ID,
		"last_msg_content":  req.Content,
		"last_msg_time":     msg.CreatedAt,
	})

	// 获取会话所有成员信息
	var members []model.ConversationMember
	s.db.Where("conversation_id = ?", req.ConversationID).Find(&members)

	// 构建发送者信息
	senderName := username.(string)
	if member.Nickname != "" {
		senderName = member.Nickname
	}

	// 推送消息给在线成员
	onlineUsers := make(map[int64]bool)
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
			// 构建推送消息
			pushData := gin.H{
				"msgId":           msg.MsgID,
				"messageId":       msg.ID,
				"conversationId":  req.ConversationID,
				"senderId":        userID.(int64),
				"senderName":      senderName,
				"content":         req.Content,
				"contentType":     req.ContentType,
				"extra":           req.Extra,
				"timestamp":        msg.CreatedAt.UnixMilli(),
				"conversationType": conv.Type,
			}

			s.hub.SendToUser(m.UserID, ws.Message{
				Type: "message",
				Data: pushData,
			})
			onlineUsers[m.UserID] = true
		}
	}

	c.JSON(200, gin.H{
		"code": 0,
		"data": gin.H{
			"messageId":       msg.ID,
			"msgId":           msg.MsgID,
			"timestamp":       msg.CreatedAt.UnixMilli(),
			"conversationId":  req.ConversationID,
			"senderId":        userID.(int64),
			"senderName":      senderName,
			"content":         req.Content,
			"contentType":    req.ContentType,
		},
	})
}

// GetHistory 获取历史消息 - [P2] 添加分页上限
func (s *MessageService) GetHistory(c *gin.Context) {
	userID, _ := c.Get("user_id")

	conversationID, _ := strconv.ParseInt(c.Query("conversationId"), 10, 64)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	// [P2] 分页上限限制
	const MaxLimit = 100 // 最大每次获取100条
	if limit > MaxLimit {
		limit = MaxLimit
	}
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

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
		c.JSON(400, gin.H{"code": 400, "msg": "参数错误: " + err.Error()})
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

	// 检查是否已经撤回
	if msg.IsRecalled {
		c.JSON(400, gin.H{"code": 400, "msg": "消息已撤回"})
		return
	}

	// 检查是否在 2 分钟内
	if time.Since(msg.CreatedAt) > 2*time.Minute {
		c.JSON(400, gin.H{"code": 400, "msg": "只能撤回 2 分钟内的消息"})
		return
	}

	// 标记为已撤回
	if err := s.db.Model(&msg).Update("is_recalled", true).Error; err != nil {
		c.JSON(500, gin.H{"code": 500, "msg": "撤回失败"})
		return
	}

	// 获取会话成员
	var members []model.ConversationMember
	s.db.Where("conversation_id = ?", msg.ConversationID).Find(&members)

	// 广播撤回通知给所有成员（包括发送者）
	if s.hub != nil {
		recallNotify := gin.H{
			"msgId":          msg.MsgID,
			"messageId":      msg.ID,
			"conversationId": msg.ConversationID,
			"revokerId":      userID.(int64),
			"revokeTime":     time.Now().UnixMilli(),
		}

		for _, m := range members {
			s.hub.SendToUser(m.UserID, ws.Message{
				Type: "recall",
				Data: recallNotify,
			})
		}
	}

	c.JSON(200, gin.H{
		"code": 0,
		"msg": "已撤回",
		"data": gin.H{
			"messageId": msg.ID,
			"msgId":     msg.MsgID,
		},
	})
}

// MarkAsRead 标记已读
func (s *MessageService) MarkAsRead(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req struct {
		ConversationID int64  `json:"conversationId" binding:"required"`
		MessageID      int64  `json:"messageId"`        // 单条消息ID
		MessageIDs     []int64 `json:"messageIds"`      // 多条消息ID
		Timestamp      int64  `json:"timestamp"`       // 时间戳，表示该时间之前的消息都标记为已读
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 400, "msg": "参数错误: " + err.Error()})
		return
	}

	// 检查会话是否存在
	var conv model.Conversation
	if err := s.db.First(&conv, req.ConversationID).Error; err != nil {
		c.JSON(404, gin.H{"code": 404, "msg": "会话不存在"})
		return
	}

	// 检查用户是否是会话成员
	var member model.ConversationMember
	if err := s.db.Where("conversation_id = ? AND user_id = ?", req.ConversationID, userID).First(&member).Error; err != nil {
		c.JSON(403, gin.H{"code": 403, "msg": "无权限"})
		return
	}

	// 需要标记已读的消息ID列表
	var messageIDs []int64

	if len(req.MessageIDs) > 0 {
		// 指定消息ID列表
		messageIDs = req.MessageIDs
	} else if req.MessageID > 0 {
		// 单条消息
		messageIDs = []int64{req.MessageID}
	} else if req.Timestamp > 0 {
		// 按时间戳标记该时间之前的所有未读消息
		var messages []model.Message
		s.db.Where("conversation_id = ? AND sender_id != ? AND created_at <= ?",
			req.ConversationID, userID, time.UnixMilli(req.Timestamp),
		).Find(&messages)
		for _, m := range messages {
			messageIDs = append(messageIDs, m.ID)
		}
	} else {
		// 默认标记最新一条消息
		var lastMsg model.Message
		if err := s.db.Where("conversation_id = ? AND sender_id != ?", req.ConversationID, userID).
			Order("created_at DESC").First(&lastMsg).Error; err == nil {
			messageIDs = []int64{lastMsg.ID}
		}
	}

	if len(messageIDs) == 0 {
		c.JSON(200, gin.H{"code": 0, "msg": "没有需要标记的消息"})
		return
	}

	// 获取这些消息的发送者，用于推送已读回执
	readReceipts := make([]gin.H, 0)
	now := time.Now()

	for _, msgID := range messageIDs {
		var msg model.Message
		if err := s.db.First(&msg, msgID).Error; err != nil {
			continue
		}

		// 检查是否已读（避免重复标记）
		var existingRead model.MessageRead
		if err := s.db.Where("message_id = ? AND user_id = ?", msgID, userID).First(&existingRead).Error; err == nil {
			continue // 已存在已读记录
		}

		// 写入已读记录
		read := model.MessageRead{
			MessageID:      msgID,
			UserID:          userID.(int64),
			ConversationID:  req.ConversationID,
			ReadAt:          now,
		}
		s.db.Create(&read)

		// 记录已读回执信息，用于通知发送者
		readReceipts = append(readReceipts, gin.H{
			"messageId":    msgID,
			"msgId":       msg.MsgID,
			"readerId":    userID.(int64),
			"readTime":    now.UnixMilli(),
		})
	}

	// 重置未读计数
	s.db.Model(&model.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ?", req.ConversationID, userID).
		Update("unread_count", 0)

	// 通知消息发送者（已读回执）
	if s.hub != nil && len(readReceipts) > 0 {
		for _, receipt := range readReceipts {
			// 查找消息发送者
			msgID := receipt["messageId"].(int64)
			var msg model.Message
			if err := s.db.First(&msg, msgID).Error; err != nil {
				continue
			}

			// 不需要通知自己
			if msg.SenderID == userID.(int64) {
				continue
			}

			// 推送已读回执给发送者
			s.hub.SendToUser(msg.SenderID, ws.Message{
				Type: "read_receipt",
				Data: receipt,
			})
		}
	}

	c.JSON(200, gin.H{
		"code": 0,
		"msg": "已读",
		"data": gin.H{
			"conversationId": req.ConversationID,
			"readCount":     len(readReceipts),
		},
	})
}

// GetConversationDetail 获取会话详情
func (s *MessageService) GetConversationDetail(c *gin.Context) {
	userID, _ := c.Get("user_id")

	conversationID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(400, gin.H{"code": 400, "msg": "参数错误"})
		return
	}

	// 查询会话
	var conv model.Conversation
	if err := s.db.First(&conv, conversationID).Error; err != nil {
		c.JSON(404, gin.H{"code": 404, "msg": "会话不存在"})
		return
	}

	// 检查用户是否是会话成员
	var member model.ConversationMember
	if err := s.db.Where("conversation_id = ? AND user_id = ?", conversationID, userID).First(&member).Error; err != nil {
		c.JSON(403, gin.H{"code": 403, "msg": "无权限访问"})
		return
	}

	// 获取所有成员
	var members []model.ConversationMember
	s.db.Where("conversation_id = ?", conversationID).Find(&members)

	// 构建成员信息
	memberInfos := make([]gin.H, 0)
	for _, m := range members {
		var user model.User
		s.db.First(&user, m.UserID)

		memberInfos = append(memberInfos, gin.H{
			"userId":   m.UserID,
			"nickname": m.Nickname,
			"role":     m.Role,
			"avatar":   user.Avatar,
		})
	}

	// 获取最后一条消息
	var lastMsg model.Message
	lastMsgFound := false
	if conv.LastMsgID > 0 {
		if err := s.db.First(&lastMsg, conv.LastMsgID).Error; err == nil {
			lastMsgFound = true
		}
	}

	// 构建响应
	data := gin.H{
		"id":             conv.ID,
		"type":           conv.Type,
		"name":           conv.Name,
		"creatorId":     conv.CreatorID,
		"unreadCount":   member.UnreadCount,
		"memberCount":   len(members),
		"members":       memberInfos,
		"createdAt":     conv.CreatedAt,
		"updatedAt":     conv.UpdatedAt,
	}

	if lastMsgFound {
		data["lastMessage"] = gin.H{
			"msgId":        lastMsg.MsgID,
			"messageId":   lastMsg.ID,
			"senderId":    lastMsg.SenderID,
			"content":     lastMsg.Content,
			"contentType": lastMsg.ContentType,
			"timestamp":   lastMsg.CreatedAt.UnixMilli(),
		}
	}

	c.JSON(200, gin.H{"code": 0, "data": data})
}

// GetUnreadCount 获取未读消息数
func (s *MessageService) GetUnreadCount(c *gin.Context) {
	userID, _ := c.Get("user_id")

	// 获取所有会话的未读总数
	var totalUnread int64
	s.db.Model(&model.ConversationMember{}).
		Where("user_id = ?", userID).
		Select("COALESCE(SUM(unread_count), 0)").
		Scan(&totalUnread)

	// 获取每个会话的未读数
	var members []model.ConversationMember
	s.db.Where("user_id = ? AND unread_count > 0", userID).Find(&members)

	conversationUnreads := make([]gin.H, 0)
	for _, m := range members {
		conversationUnreads = append(conversationUnreads, gin.H{
			"conversationId": m.ConversationID,
			"unreadCount":    m.UnreadCount,
		})
	}

	c.JSON(200, gin.H{
		"code": 0,
		"data": gin.H{
			"totalUnread":      totalUnread,
			"conversations":   conversationUnreads,
		},
	})
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
