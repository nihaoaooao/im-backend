package queue

import (
	"testing"
	"time"

	"im-backend/model"
)

// ============ 测试常量 ============

func TestConstants(t *testing.T) {
	if StreamKey != "message:queue" {
		t.Errorf("Expected StreamKey 'message:queue', got '%s'", StreamKey)
	}

	if DLQKey != "message:queue:dlq" {
		t.Errorf("Expected DLQKey 'message:queue:dlq', got '%s'", DLQKey)
	}

	if DelayQueueKey != "message:delay:queue" {
		t.Errorf("Expected DelayQueueKey 'message:delay:queue', got '%s'", DelayQueueKey)
	}

	if ConsumerGroup != "im-consumer-group" {
		t.Errorf("Expected ConsumerGroup 'im-consumer-group', got '%s'", ConsumerGroup)
	}
}

func TestRetryConfig(t *testing.T) {
	if MaxRetries != 3 {
		t.Errorf("Expected MaxRetries 3, got %d", MaxRetries)
	}

	if RetryDelay != 1*time.Second {
		t.Errorf("Expected RetryDelay 1s, got %v", RetryDelay)
	}

	if MaxRetryDelay != 30*time.Second {
		t.Errorf("Expected MaxRetryDelay 30s, got %v", MaxRetryDelay)
	}
}

func TestConcurrencyConfig(t *testing.T) {
	if DefaultConcurrency != 100 {
		t.Errorf("Expected DefaultConcurrency 100, got %d", DefaultConcurrency)
	}

	if BatchSize != 100 {
		t.Errorf("Expected BatchSize 100, got %d", BatchSize)
	}
}

func TestTimeoutConfig(t *testing.T) {
	if BlockTimeout != 5*time.Second {
		t.Errorf("Expected BlockTimeout 5s, got %v", BlockTimeout)
	}

	if ConsumerTimeout != 30*time.Second {
		t.Errorf("Expected ConsumerTimeout 30s, got %v", ConsumerTimeout)
	}
}

// ============ 测试 SimpleHandler ============

func TestSimpleHandler(t *testing.T) {
	handled := false

	handler := NewSimpleHandler(func(msg *model.Message) error {
		handled = true
		return nil
	})

	if handler == nil {
		t.Fatal("NewSimpleHandler returned nil")
	}

	msg := &model.Message{MsgID: "test-123"}
	if err := handler.HandleMessage(msg); err != nil {
		t.Errorf("SimpleHandler error: %v", err)
	}

	if !handled {
		t.Error("SimpleHandler was not called")
	}
}

func TestSimpleHandlerError(t *testing.T) {
	handler := NewSimpleHandler(func(msg *model.Message) error {
		return nil
	})

	if handler == nil {
		t.Error("NewSimpleHandler returned nil")
	}
}
