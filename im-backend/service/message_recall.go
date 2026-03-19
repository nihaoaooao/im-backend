package service

import (
	"errors"
	"im-backend/config"
	"im-backend/models"
	"im-backend/ws"
	"time"

	"gorm.io/gorm"
)

var (
	ErrMessageNotFound       = errors.New("消息不存在")
	ErrNotMessageSender      = errors.New("只能撤回自己发送的消息")
	ErrMessageAlreadyRevoked = errors.New("消息已撤回")
	ErrCannotRevoke          = errors.New("消息无法撤回（已超过撤回时限）")
	ErrRecallTimeExpired     = errors.New("消息撤回时限已过")
)

// MessageService 消息服务
type MessageService struct {
	db            *gorm.DB
	hub           *ws.Hub
	redisClient   interface{}
	recallLimit   time.Duration // 撤回时限，默认2分钟
}

// NewMessageService 创建消息服务
func NewMessageService(db *gorm.DB, hub *ws.Hub, redisClient interface{}) *MessageService {
	return &MessageService{
		db:          db,
		hub:         hub,
		redisClient: redisClient,
		recallLimit: 2 * time.Minute, // 默认2分钟撤回时限
	}
}

// RecallMessage 撤回消息
// senderID: 撤回者ID
// msgID: 消息ID
// 返回: 撤回是否成功，错误信息
func (s *MessageService) RecallMessage(senderID int64, msgID string) error {
	// 1. 查询消息
	var message models.Message
	result := s.db.Where("msg_id = ?", msgID).First(&message)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return ErrMessageNotFound
		}
		return result.Error
	}

	// 2. 检查是否是发送者本人
	if message.SenderID != senderID {
		return ErrNotMessageSender
	}

	// 3. 检查消息是否已撤回
	if message.Revoked {
		return ErrMessageAlreadyRevoked
	}

	// 4. 检查是否在撤回时限内
	if !message.CanRevoke {
		return ErrCannotRevoke
	}

	// 检查时间是否超过撤回时限
	if time.Since(message.CreatedAt) > s.recallLimit {
		// 标记为不可撤回
		s.db.Model(&message).Update("can_revoke", false)
		return ErrRecallTimeExpired
	}

	// 5. 执行撤回
	now := time.Now()
	updateData := map[string]interface{}{
		"revoked":      true,
		"revoked_at":   now,
		"can_revoke":   false,
		"is_recalled":  true,
	}

	if err := s.db.Model(&message).Updates(updateData).Error; err != nil {
		return err
	}

	// 6. 记录撤回日志
	recall := models.MessageRecall{
		MessageID:      message.ID,
		MsgID:          message.MsgID,
		ConversationID: message.ConversationID,
		SenderID:       message.SenderID,
		RevokerID:      senderID,
		RecallTime:     now,
	}
	if err := s.db.Create(&recall).Error; err != nil {
		// 记录失败不影响撤回主流程
		config.Log.Printf("Failed to create recall record, error: %v", err)
	}

	// 7. 广播撤回消息给相关用户
	s.broadcastRecall(message, senderID)

	return nil
}

// broadcastRecall 广播撤回消息
func (s *MessageService) broadcastRecall(message models.Message, revokerID int64) {
	recallNotice := ws.MessageRecallNotice{
		Type:            "recall",
		MsgID:           message.MsgID,
		ConversationID:  message.ConversationID,
		RevokerID:       revokerID,
		OriginalSenderID: message.SenderID,
		RevokedAt:       time.Now().Unix(),
	}

	// 发送给发送者
	if s.hub != nil {
		s.hub.SendToUser(message.SenderID, recallNotice)
		// 发送给接收者
		s.hub.SendToUser(message.ReceiverID, recallNotice)
	}
}

// RevokeMessageByTimePeriod 根据时间段批量标记消息为不可撤回
// 用于定时任务，将超过撤回时限的消息标记为不可撤回
func (s *MessageService) RevokeMessageByTimePeriod(limit int64) (int64, error) {
	cutoffTime := time.Now().Add(-s.recallLimit)

	result := s.db.Model(&models.Message{}).
		Where("can_revoke = ? AND created_at < ? AND revoked = ?", true, cutoffTime, false).
		Limit(int(limit)).
		Updates(map[string]interface{}{
			"can_revoke": false,
		})

	if result.Error != nil {
		return 0, result.Error
	}

	return result.RowsAffected, nil
}

// GetRecallableMessages 获取可撤回的消息列表
// 用于定时任务检查
func (s *MessageService) GetRecallableMessages(limit int64) ([]models.Message, error) {
	var messages []models.Message
	cutoffTime := time.Now().Add(-s.recallLimit)

	err := s.db.Where("can_revoke = ? AND created_at < ? AND revoked = ?", true, cutoffTime, false).
		Limit(int(limit)).
		Find(&messages).Error

	return messages, err
}

// IsWithinRecallTime 检查消息是否在撤回时限内
func (s *MessageService) IsWithinRecallTime(createdAt time.Time) bool {
	return time.Since(createdAt) <= s.recallLimit
}

// SetRecallLimit 设置撤回时限
func (s *MessageService) SetRecallLimit(limit time.Duration) {
	s.recallLimit = limit
}

// GetRecallLimit 获取撤回时限
func (s *MessageService) GetRecallLimit() time.Duration {
	return s.recallLimit
}
