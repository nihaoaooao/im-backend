package service

import (
	"testing"
	"time"
)

// TestRecallSchedulerCreation 测试创建定时检查器
func TestRecallSchedulerCreation(t *testing.T) {
	scheduler := NewRecallScheduler(nil, 10*time.Second)

	if scheduler == nil {
		t.Error("Expected scheduler to be created")
	}

	if scheduler.interval != 10*time.Second {
		t.Errorf("Expected interval to be 10 seconds, got %v", scheduler.interval)
	}

	if scheduler.isRunning {
		t.Error("Expected scheduler to not be running initially")
	}
}

// TestRecallSchedulerStartStop 测试启动和停止
func TestRecallSchedulerStartStop(t *testing.T) {
	scheduler := NewRecallScheduler(nil, 100*time.Millisecond)

	// 测试启动
	scheduler.Start()
	if !scheduler.IsRunning() {
		t.Error("Expected scheduler to be running after Start()")
	}

	// 测试停止
	scheduler.Stop()
	if scheduler.IsRunning() {
		t.Error("Expected scheduler to not be running after Stop()")
	}
}

// TestRecallSchedulerSetInterval 测试设置间隔
func TestRecallSchedulerSetInterval(t *testing.T) {
	scheduler := NewRecallScheduler(nil, 10*time.Second)

	// 测试设置新间隔
	newInterval := 30 * time.Second
	scheduler.SetInterval(newInterval)

	if scheduler.GetInterval() != newInterval {
		t.Errorf("Expected interval to be %v, got %v", newInterval, scheduler.GetInterval())
	}
}

// TestRecallSchedulerConcurrentStart 重复启动测试
func TestRecallSchedulerConcurrentStart(t *testing.T) {
	scheduler := NewRecallScheduler(nil, 100*time.Millisecond)

	// 启动第一次
	scheduler.Start()
	if !scheduler.IsRunning() {
		t.Error("Expected scheduler to be running")
	}

	// 尝试再次启动（应该忽略）
	scheduler.Start()

	// 停止
	scheduler.Stop()
}

// TestRecallSchedulerMultipleStop 重复停止测试
func TestRecallSchedulerMultipleStop(t *testing.T) {
	scheduler := NewRecallScheduler(nil, 100*time.Millisecond)

	// 先启动
	scheduler.Start()

	// 停止第一次
	scheduler.Stop()

	// 再次停止（应该安全）
	scheduler.Stop()
}
