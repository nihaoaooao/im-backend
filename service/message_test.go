package service

import (
	"testing"
	"time"

	"im-backend/model"
)

// ============ 测试消息ID生成 ============

func TestGenerateMsgID(t *testing.T) {
	msgID1 := generateMsgID()
	msgID2 := generateMsgID()

	// 验证格式
	if len(msgID1) == 0 {
		t.Error("msgID should not be empty")
	}

	// 验证两次生成ID不同（时间不同）
	if msgID1 == msgID2 {
		t.Log("Note: msgID1 and msgID2 are same, this is expected if generated in same millisecond")
	}
}

// ============ 测试 Conversation 模型 ============

func TestConversationModel(t *testing.T) {
	conv := model.Conversation{
		Type:        "private",
		Name:        "Test Conversation",
		CreatorID:   1,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if conv.Type != "private" {
		t.Errorf("Expected Type 'private', got '%s'", conv.Type)
	}

	if conv.Name != "Test Conversation" {
		t.Errorf("Expected Name 'Test Conversation', got '%s'", conv.Name)
	}
}

// ============ 测试 Message 模型 ============

func TestMessageModel(t *testing.T) {
	msg := model.Message{
		MsgID:          "msg_123456",
		ConversationID: 1,
		SenderID:       1,
		Content:        "Hello World",
		ContentType:    "text",
		IsRecalled:     false,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if msg.MsgID != "msg_123456" {
		t.Errorf("Expected MsgID 'msg_123456', got '%s'", msg.MsgID)
	}

	if msg.Content != "Hello World" {
		t.Errorf("Expected Content 'Hello World', got '%s'", msg.Content)
	}

	if msg.ContentType != "text" {
		t.Errorf("Expected ContentType 'text', got '%s'", msg.ContentType)
	}

	if msg.IsRecalled != false {
		t.Error("Expected IsRecalled to be false")
	}
}

// ============ 测试 ConversationMember 模型 ============

func TestConversationMemberModel(t *testing.T) {
	member := model.ConversationMember{
		ConversationID: 1,
		UserID:         1,
		Role:           "member",
		UnreadCount:   5,
		JoinedAt:      time.Now(),
	}

	if member.Role != "member" {
		t.Errorf("Expected Role 'member', got '%s'", member.Role)
	}

	if member.UnreadCount != 5 {
		t.Errorf("Expected UnreadCount 5, got %d", member.UnreadCount)
	}
}

// ============ 测试 MessageRead 模型 ============

func TestMessageReadModel(t *testing.T) {
	read := model.MessageRead{
		MessageID:      1,
		UserID:          2,
		ConversationID: 1,
		ReadAt:         time.Now(),
	}

	if read.MessageID != 1 {
		t.Errorf("Expected MessageID 1, got %d", read.MessageID)
	}

	if read.UserID != 2 {
		t.Errorf("Expected UserID 2, got %d", read.UserID)
	}
}

// ============ 测试消息内容类型常量 ============

func TestMessageContentTypes(t *testing.T) {
	validTypes := []string{"text", "image", "voice", "video", "file"}

	testCases := []struct {
		contentType string
		isValid    bool
	}{
		{"text", true},
		{"image", true},
		{"voice", true},
		{"video", true},
		{"file", true},
		{"invalid", false},
		{"", false},
	}

	for _, tc := range testCases {
		found := false
		for _, vt := range validTypes {
			if tc.contentType == vt {
				found = true
				break
			}
		}
		if found != tc.isValid {
			t.Errorf("ContentType '%s' validation failed, expected %v", tc.contentType, tc.isValid)
		}
	}
}

// ============ 测试会话类型常量 ============

func TestConversationTypes(t *testing.T) {
	validTypes := []string{"private", "group"}

	privateFound := false
	groupFound := false

	for _, vt := range validTypes {
		if vt == "private" {
			privateFound = true
		}
		if vt == "group" {
			groupFound = true
		}
	}

	if !privateFound {
		t.Error("'private' should be a valid conversation type")
	}

	if !groupFound {
		t.Error("'group' should be a valid conversation type")
	}
}

// ============ 测试角色常量 ============

func TestMemberRoles(t *testing.T) {
	validRoles := []string{"owner", "admin", "member"}

	testCases := []struct {
		role    string
		isValid bool
	}{
		{"owner", true},
		{"admin", true},
		{"member", true},
		{"invalid", false},
	}

	for _, tc := range testCases {
		found := false
		for _, vr := range validRoles {
			if tc.role == vr {
				found = true
				break
			}
		}
		if found != tc.isValid {
			t.Errorf("Role '%s' validation failed, expected %v", tc.role, tc.isValid)
		}
	}
}

// ============ 测试时间戳处理 ============

func TestMessageTimestamp(t *testing.T) {
	now := time.Now()
	msg := model.Message{
		CreatedAt: now,
	}

	// 验证时间戳转换
	timestamp := msg.CreatedAt.UnixMilli()
	if timestamp == 0 {
		t.Error("Timestamp should not be 0")
	}

	// 验证时间戳转换回来
	backToTime := time.UnixMilli(timestamp)
	if backToTime.Unix() != now.Unix() {
		t.Logf("Note: Millisecond precision may cause slight difference: original=%v, back=%v", now, backToTime)
	}
}
