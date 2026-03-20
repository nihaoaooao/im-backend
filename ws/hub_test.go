package ws

import (
	"testing"
	"time"
)

// ============ 测试分片算法 ============

func TestFNV32a(t *testing.T) {
	tests := []struct {
		input    string
		expected uint32
	}{
		{"user:1", 0}, // 具体值不重要，重要的是一致性
		{"user:2", 0},
		{"user:100", 0},
	}

	// 测试一致性：相同输入产生相同输出
	result1 := fnv32a("test")
	result2 := fnv32a("test")
	if result1 != result2 {
		t.Errorf("FNV32a not consistent: %d != %d", result1, result2)
	}

	// 测试不同输入产生不同输出
	result3 := fnv32a("user:1")
	result4 := fnv32a("user:2")
	if result3 == result4 {
		t.Logf("Warning: Different inputs produced same hash")
	}

	_ = tests // 避免未使用警告
}

func TestGetShard(t *testing.T) {
	// 测试分片在有效范围内
	for i := int64(1); i <= 1000; i++ {
		shard := getShard(i)
		if shard < 0 || shard >= ShardCount {
			t.Errorf("Shard out of range: %d", shard)
		}
	}

	// 测试一致性
	if getShard(100) != getShard(100) {
		t.Error("getShard not consistent")
	}
}

// ============ 测试 Hub 创建 ============

func TestNewHub(t *testing.T) {
	hub := NewHub(nil)

	if hub == nil {
		t.Fatal("NewHub returned nil")
	}

	if len(hub.shards) != ShardCount {
		t.Errorf("Expected %d shards, got %d", ShardCount, len(hub.shards))
	}

	if hub.register == nil {
		t.Error("register channel is nil")
	}

	if hub.unregister == nil {
		t.Error("unregister channel is nil")
	}

	if hub.broadcast == nil {
		t.Error("broadcast channel is nil")
	}
}

// ============ 测试连接管理 ============

func TestClientCreation(t *testing.T) {
	hub := NewHub(nil)

	client := &Client{
		id:         "test_conn_1",
		userID:     123,
		send:       make(chan []byte, 10),
		hub:        hub,
		createdAt:  time.Now(),
		lastActive: time.Now(),
	}

	if client.id != "test_conn_1" {
		t.Errorf("Expected id 'test_conn_1', got '%s'", client.id)
	}

	if client.userID != 123 {
		t.Errorf("Expected userID 123, got %d", client.userID)
	}

	if client.send == nil {
		t.Error("send channel is nil")
	}
}

// ============ 测试连接数限制 ============

func TestMaxConnectionsPerUser(t *testing.T) {
	if MaxConnectionsPerUser != 5 {
		t.Errorf("Expected MaxConnectionsPerUser to be 5, got %d", MaxConnectionsPerUser)
	}
}

// ============ 测试心跳配置 ============

func TestHeartbeatConfig(t *testing.T) {
	if HeartbeatInterval != 30 {
		t.Errorf("Expected HeartbeatInterval to be 30, got %d", HeartbeatInterval)
	}

	if PingTimeout != 90 {
		t.Errorf("Expected PingTimeout to be 90, got %d", PingTimeout)
	}
}

// ============ 测试消息结构 ============

func TestMessageStructure(t *testing.T) {
	msg := Message{
		Type: "test",
		Data: map[string]interface{}{
			"content": "hello",
		},
	}

	if msg.Type != "test" {
		t.Errorf("Expected Type 'test', got '%s'", msg.Type)
	}

	if msg.Data == nil {
		t.Error("Data is nil")
	}
}

// ============ 测试 BroadcastMessage ============

func TestBroadcastMessage(t *testing.T) {
	msg := BroadcastMessage{
		Message:    "test message",
		TargetID:   123,
		TargetType: "user",
	}

	if msg.TargetID != 123 {
		t.Errorf("Expected TargetID 123, got %d", msg.TargetID)
	}

	if msg.TargetType != "user" {
		t.Errorf("Expected TargetType 'user', got '%s'", msg.TargetType)
	}
}

// ============ 测试常量定义 ============

func TestConstants(t *testing.T) {
	if ShardCount != 64 {
		t.Errorf("Expected ShardCount to be 64, got %d", ShardCount)
	}

	if SendChanSize != 256 {
		t.Errorf("Expected SendChanSize to be 256, got %d", SendChanSize)
	}

	if WriteWait != 10*time.Second {
		t.Errorf("Expected WriteWait to be 10s, got %v", WriteWait)
	}

	if ReadDeadline != 60*time.Second {
		t.Errorf("Expected ReadDeadline to be 60s, got %v", ReadDeadline)
	}
}
