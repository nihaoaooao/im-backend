package service

import (
	"testing"
	"time"

	"im-backend/models"
)

// MockMessageService 模拟消息服务用于测试
type MockMessageService struct {
	messages map[string]*models.Message
	recalls  []models.MessageRecall
}

// NewMockMessageService 创建模拟消息服务
func NewMockMessageService() *MockMessageService {
	return &MockMessageService{
		messages: make(map[string]*models.Message),
		recalls:  make([]models.MessageRecall, 0),
	}
}

// GetMessage 获取消息
func (m *MockMessageService) GetMessage(msgID string) (*models.Message, bool) {
	msg, ok := m.messages[msgID]
	return msg, ok
}

// CreateMessage 创建消息
func (m *MockMessageService) CreateMessage(msg *models.Message) {
	m.messages[msg.MsgID] = msg
}

// RevokeMessage 撤回消息
func (m *MockMessageService) RevokeMessage(msgID string) error {
	msg, ok := m.messages[msgID]
	if !ok {
		return ErrMessageNotFound
	}
	if msg.Revoked {
		return ErrMessageAlreadyRevoked
	}
	msg.Revoked = true
	now := time.Now()
	msg.RevokedAt = &now

	// 记录撤回
	m.recalls = append(m.recalls, models.MessageRecall{
		MessageID:      msg.ID,
		MsgID:          msg.MsgID,
		ConversationID: msg.ConversationID,
		SenderID:       msg.SenderID,
		RevokerID:      msg.SenderID,
		RecallTime:     now,
	})

	return nil
}

// TestRecallMessageBasic 基础撤回测试
func TestRecallMessageBasic(t *testing.T) {
	mock := NewMockMessageService()

	// 创建测试消息
	testMsg := &models.Message{
		ID:              1,
		MsgID:           "msg_001",
		ConversationID: 100,
		SenderID:        1,
		ReceiverID:      2,
		ReceiverType:    "user",
		Content:         "Hello",
		CanRevoke:       true,
		Revoked:         false,
		CreatedAt:       time.Now(),
	}
	mock.CreateMessage(testMsg)

	// 测试撤回自己的消息
	err := mock.RevokeMessage("msg_001")
	if err != nil {
		t.Errorf("Expected recall to succeed, got error: %v", err)
	}

	// 验证消息已撤回
	msg, _ := mock.GetMessage("msg_001")
	if !msg.Revoked {
		t.Error("Expected message to be revoked")
	}
	if msg.RevokedAt == nil {
		t.Error("Expected revoked_at to be set")
	}

	// 验证撤回记录已创建
	if len(mock.recalls) != 1 {
		t.Errorf("Expected 1 recall record, got %d", len(mock.recalls))
	}
}

// TestRecallNonExistentMessage 撤回不存在的消息
func TestRecallNonExistentMessage(t *testing.T) {
	mock := NewMockMessageService()

	err := mock.RevokeMessage("non_existent")
	if err != ErrMessageNotFound {
		t.Errorf("Expected ErrMessageNotFound, got: %v", err)
	}
}

// TestRecallAlreadyRevokedMessage 撤回已撤回的消息
func TestRecallAlreadyRevokedMessage(t *testing.T) {
	mock := NewMockMessageService()

	// 创建已撤回的消息
	testMsg := &models.Message{
		ID:        2,
		MsgID:     "msg_002",
		SenderID:  1,
		Revoked:   true,
		CreatedAt: time.Now(),
	}
	mock.CreateMessage(testMsg)

	err := mock.RevokeMessage("msg_002")
	if err != ErrMessageAlreadyRevoked {
		t.Errorf("Expected ErrMessageAlreadyRevoked, got: %v", err)
	}
}

// TestIsWithinRecallTime 测试撤回时间判断
func TestIsWithinRecallTime(t *testing.T) {
	service := &MessageService{
		recallLimit: 2 * time.Minute,
	}

	// 测试1分钟前创建的消息（可撤回）
	oneMinuteAgo := time.Now().Add(-1 * time.Minute)
	if !service.IsWithinRecallTime(oneMinuteAgo) {
		t.Error("Expected message created 1 minute ago to be within recall time")
	}

	// 测试3分钟前创建的消息（不可撤回）
	threeMinutesAgo := time.Now().Add(-3 * time.Minute)
	if service.IsWithinRecallTime(threeMinutesAgo) {
		t.Error("Expected message created 3 minutes ago to be beyond recall time")
	}

	// 测试刚刚创建的消息（可撤回）
	justNow := time.Now()
	if !service.IsWithinRecallTime(justNow) {
		t.Error("Expected message created just now to be within recall time")
	}
}

// TestRecallLimit 测试设置撤回时限
func TestRecallLimit(t *testing.T) {
	service := &MessageService{}

	// 测试默认时限
	if service.GetRecallLimit() != 2*time.Minute {
		t.Errorf("Expected default recall limit to be 2 minutes, got %v", service.GetRecallLimit())
	}

	// 测试设置时限
	newLimit := 5 * time.Minute
	service.SetRecallLimit(newLimit)
	if service.GetRecallLimit() != newLimit {
		t.Errorf("Expected recall limit to be %v, got %v", newLimit, service.GetRecallLimit())
	}
}

// TestMessageServiceCreation 测试消息服务创建
func TestMessageServiceCreation(t *testing.T) {
	// 测试创建消息服务（不实际连接数据库）
	service := NewMessageService(nil, nil, nil)

	if service == nil {
		t.Error("Expected service to be created")
	}

	if service.recallLimit != 2*time.Minute {
		t.Errorf("Expected recall limit to be 2 minutes, got %v", service.recallLimit)
	}
}
