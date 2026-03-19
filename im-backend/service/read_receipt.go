package service

import (
	"context"
	"errors"
	"fmt"
	"im-backend/config"
	"im-backend/models"
	"im-backend/ws"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

var (
	ErrAlreadyRead       = errors.New("消息已读")
	ErrInvalidMessageIDs = errors.New("无效的消息ID列表")
)

// ReadReceiptService 已读回执服务
type ReadReceiptService struct {
	db          *gorm.DB
	hub         *ws.Hub
	redisClient *redis.Client
}

// NewReadReceiptService 创建已读回执服务
func NewReadReceiptService(db *gorm.DB, hub *ws.Hub, redisClient *redis.Client) *ReadReceiptService {
	return &ReadReceiptService{
		db:          db,
		hub:         hub,
		redisClient: redisClient,
	}
}

// MarkMessagesAsRead 标记消息为已读
// userID: 已读用户ID
// messageIDs: 消息ID数组
// 返回: 已读消息数量, 错误信息
func (s *ReadReceiptService) MarkMessagesAsRead(ctx context.Context, userID int64, messageIDs []int64) (int64, error) {
	if len(messageIDs) == 0 {
		return 0, ErrInvalidMessageIDs
	}

	// 去重
	uniqueIDs := make(map[int64]bool)
	for _, id := range messageIDs {
		uniqueIDs[id] = true
	}

	var readCount int64
	now := time.Now()

	// 批量查询消息
	var messages []models.Message
	if err := s.db.Where("id IN ?", messageIDs).Find(&messages).Error; err != nil {
		return 0, err
	}

	// 获取有效的消息ID（需要是接收者或群成员）
	validMessageIDs := make(map[int64]models.Message)
	for _, msg := range messages {
		// 单聊：接收者可以标记已读
		// 群聊：群成员可以标记已读
		if msg.ReceiverID == userID || msg.ReceiverType == "group" {
			validMessageIDs[msg.ID] = msg
		}
	}

	// 批量插入已读记录
	for msgID, msg := range validMessageIDs {
		// 检查是否已读
		var existing models.MessageRead
		err := s.db.Where("message_id = ? AND user_id = ?", msgID, userID).First(&existing).Error
		if err == nil {
			// 已存在，跳过
			continue
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			continue
		}

		// 创建已读记录
		readRecord := models.MessageRead{
			MessageID: msgID,
			UserID:    userID,
			ReadAt:    now,
		}
		if err := s.db.Create(&readRecord).Error; err != nil {
			config.Log.Printf("Failed to create read record, message_id: %d, error: %v", msgID, err)
			continue
		}

		readCount++

		// 更新 Redis 缓存
		s.cacheReadUser(ctx, msgID, userID)

		// 更新会话未读数
		s.updateUnreadCount(ctx, msg.ConversationID, userID, -1)

		// 广播已读通知给消息发送者
		s.broadcastReadReceipt(msg, userID)
	}

	return readCount, nil
}

// cacheReadUser 将已读用户添加到 Redis 缓存
func (s *ReadReceiptService) cacheReadUser(ctx context.Context, messageID int64, userID int64) {
	if s.redisClient == nil {
		return
	}

	key := fmt.Sprintf("message:%d:read_users", messageID)
	s.redisClient.SAdd(ctx, key, userID)
	// 设置过期时间 7 天
	s.redisClient.Expire(ctx, key, 7*24*time.Hour)
}

// updateUnreadCount 更新会话未读计数
func (s *ReadReceiptService) updateUnreadCount(ctx context.Context, conversationID int64, userID int64, delta int64) {
	if s.redisClient == nil {
		return
	}

	// 使用 Redis 原子操作
	key := fmt.Sprintf("conversation:%d:unread:%d", conversationID, userID)
	s.redisClient.IncrBy(ctx, key, delta)
	// 设置过期时间 30 天
	s.redisClient.Expire(ctx, key, 30*24*time.Hour)

	// 同时更新数据库
	var count models.ConversationUnreadCount
	err := s.db.Where("conversation_id = ? AND user_id = ?", conversationID, userID).First(&count).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		count = models.ConversationUnreadCount{
			ConversationID: conversationID,
			UserID:         userID,
			UnreadCount:    0,
		}
		s.db.Create(&count)
	} else if err == nil {
		count.UnreadCount += delta
		if count.UnreadCount < 0 {
			count.UnreadCount = 0
		}
		s.db.Model(&count).Update("unread_count", count.UnreadCount)
	}
}

// broadcastReadReceipt 广播已读回执给消息发送者
func (s *ReadReceiptService) broadcastReadReceipt(message models.Message, readUserID int64) {
	if s.hub == nil {
		return
	}

	// 获取已读用户列表
	readUsers := s.GetReadUserList(context.Background(), message.ID)

	// 构建通知
	notice := ws.ReadReceiptNotice{
		Type:            "read_receipt",
		MessageID:       message.ID,
		ConversationID:  message.ConversationID,
		ReadBy:          readUsers,
		ReadCount:       int64(len(readUsers)),
		TotalCount:      1, // 单聊固定为1
	}

	// 发送给消息发送者
	s.hub.SendToUser(message.SenderID, notice)
}

// GetReadUserList 获取消息已读用户列表
func (s *ReadReceiptService) GetReadUserList(ctx context.Context, messageID int64) []int64 {
	// 先尝试从 Redis 获取
	if s.redisClient != nil {
		key := fmt.Sprintf("message:%d:read_users", messageID)
		users, err := s.redisClient.SMembers(ctx, key).Result()
		if err == nil && len(users) > 0 {
			result := make([]int64, 0, len(users))
			for _, u := range users {
				var uid int64
				fmt.Sscanf(u, "%d", &uid)
				result = append(result, uid)
			}
			return result
		}
	}

	// 从数据库获取
	var reads []models.MessageRead
	if err := s.db.Where("message_id = ?", messageID).Find(&reads).Error; err != nil {
		return nil
	}

	result := make([]int64, len(reads))
	for i, r := range reads {
		result[i] = r.UserID
	}

	return result
}

// GetUnreadUserList 获取消息未读用户列表
func (s *ReadReceiptService) GetUnreadUserList(ctx context.Context, messageID int64, allUserIDs []int64) []int64 {
	readUsers := s.GetReadUserList(ctx, messageID)
	readMap := make(map[int64]bool)
	for _, uid := range readUsers {
		readMap[uid] = true
	}

	var unread []int64
	for _, uid := range allUserIDs {
		if !readMap[uid] {
			unread = append(unread, uid)
		}
	}

	return unread
}

// GetMessageReadStatus 获取消息已读状态
func (s *ReadReceiptService) GetMessageReadStatus(ctx context.Context, messageID int64, conversationMembers []int64) (*ws.MessageReadStatus, error) {
	var message models.Message
	if err := s.db.Where("id = ?", messageID).First(&message).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrMessageNotFound
		}
		return nil, err
	}

	// 获取已读用户详情
	reads, err := s.getReadUsersWithInfo(ctx, messageID)
	if err != nil {
		return nil, err
	}

	// 计算未读用户
	readMap := make(map[int64]bool)
	for _, r := range reads {
		readMap[r.UserID] = true
	}

	var unread []ws.UserInfo
	for _, uid := range conversationMembers {
		if !readMap[uid] {
			// 需要查询用户信息
			user := s.getUserInfo(uid)
			if user.UserID > 0 {
				unread = append(unread, user)
			}
		}
	}

	return &ws.MessageReadStatus{
		MessageID:       messageID,
		ReadBy:          reads,
		ReadCount:       int64(len(reads)),
		UnreadBy:        unread,
		UnreadCount:     int64(len(unread)),
	}, nil
}

// getReadUsersWithInfo 获取已读用户详细信息
func (s *ReadReceiptService) getReadUsersWithInfo(ctx context.Context, messageID int64) ([]ws.UserInfo, error) {
	var reads []models.MessageRead
	if err := s.db.Where("message_id = ?", messageID).Order("read_at ASC").Find(&reads).Error; err != nil {
		return nil, err
	}

	result := make([]ws.UserInfo, len(reads))
	for i, r := range reads {
		userInfo := s.getUserInfo(r.UserID)
		userInfo.ReadAt = r.ReadAt.Format(time.RFC3339)
		result[i] = userInfo
	}

	return result, nil
}

// getUserInfo 获取用户信息（需要根据实际用户服务实现）
func (s *ReadReceiptService) getUserInfo(userID int64) ws.UserInfo {
	// TODO: 从用户服务获取真实用户信息
	// 这里返回简化信息，实际需要查询用户表
	return ws.UserInfo{
		UserID: userID,
	}
}

// GetConversationUnreadCount 获取会话未读数
func (s *ReadReceiptService) GetConversationUnreadCount(ctx context.Context, conversationID int64, userID int64) (int64, error) {
	// 先从 Redis 获取
	if s.redisClient != nil {
		key := fmt.Sprintf("conversation:%d:unread:%d", conversationID, userID)
		count, err := s.redisClient.Get(ctx, key).Int64()
		if err == nil {
			return count, nil
		}
	}

	// 从数据库获取
	var count models.ConversationUnreadCount
	err := s.db.Where("conversation_id = ? AND user_id = ?", conversationID, userID).First(&count).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, nil
	}
	return count.UnreadCount, err
}

// SetConversationUnreadCount 设置会话未读数
func (s *ReadReceiptService) SetConversationUnreadCount(ctx context.Context, conversationID int64, userID int64, count int64) error {
	// 更新 Redis
	if s.redisClient != nil {
		key := fmt.Sprintf("conversation:%d:unread:%d", conversationID, userID)
		s.redisClient.Set(ctx, key, count, 30*24*time.Hour)
	}

	// 更新数据库
	var existing models.ConversationUnreadCount
	err := s.db.Where("conversation_id = ? AND user_id = ?", conversationID, userID).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		record := models.ConversationUnreadCount{
			ConversationID: conversationID,
			UserID:         userID,
			UnreadCount:    count,
		}
		return s.db.Create(&record).Error
	}

	return s.db.Model(&existing).Update("unread_count", count).Error
}

// MarkConversationAsRead 标记整个会话为已读
func (s *ReadReceiptService) MarkConversationAsRead(ctx context.Context, conversationID int64, userID int64, messageIDs []int64) (int64, error) {
	if len(messageIDs) == 0 {
		// 如果没有指定消息ID，获取会话中所有未读消息
		var messages []models.Message
		err := s.db.Where("conversation_id = ? AND receiver_id = ? AND receiver_type = ?", conversationID, userID, "user").
			Find(&messages).Error
		if err != nil {
			return 0, err
		}
		messageIDs = make([]int64, len(messages))
		for i, m := range messages {
			messageIDs[i] = m.ID
		}
	}

	return s.MarkMessagesAsRead(ctx, userID, messageIDs)
}

// ResetConversationUnreadCount 重置会话未读数
func (s *ReadReceiptService) ResetConversationUnreadCount(ctx context.Context, conversationID int64, userID int64) error {
	return s.SetConversationUnreadCount(ctx, conversationID, userID, 0)
}
