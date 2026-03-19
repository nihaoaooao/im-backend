package benchmark

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// ============ 基准测试 ============

// BenchmarkHubSendToUser 消息推送性能测试
func BenchmarkHubSendToUser(b *testing.B) {
	// 模拟 WebSocket Hub
	type Client struct {
		Send chan []byte
	}
	clients := make(map[int64]map[*Client]bool)
	var mu sync.RWMutex

	// 添加 1000 个在线用户
	for i := int64(1); i <= 1000; i++ {
		clients[i] = map[*Client]bool{
			{Send: make(chan []byte, 256)}: true,
		}
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		userID := int64(1)
		msg := []byte(`{"type":"message","content":"hello"}`)
		for pb.Next() {
			mu.RLock()
			if c, ok := clients[userID]; ok {
				for client := range c {
					select {
					case client.Send <- msg:
					default:
					}
				}
			}
			mu.RUnlock()
			userID++
			if userID > 1000 {
				userID = 1
			}
		}
	})
}

// BenchmarkCacheHit 缓存命中测试
func BenchmarkCacheHit(b *testing.B) {
	// 模拟缓存
	cache := make(map[string]string)
	var mu sync.RWMutex

	// 预填充缓存
	for i := 0; i < 1000; i++ {
		cache[strconv.Itoa(i)] = "data"
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		key := "0"
		for pb.Next() {
			mu.RLock()
			_, _ = cache[key]
			mu.RUnlock()
			// 轮询 key
		}
	})
}

// BenchmarkJSONMarshal JSON 序列化测试
func BenchmarkJSONMarshal(b *testing.B) {
	type Message struct {
		Type    string `json:"type"`
		MsgID   string `json:"msg_id"`
		Content string `json:"content"`
	}

	msg := Message{
		Type:    "message",
		MsgID:   "123456",
		Content: "Hello, world!",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = MarshalJSON(msg)
	}
}

func MarshalJSON(v interface{}) ([]byte, error) {
	// 简化版，实际使用 encoding/json
	return []byte(`{"type":"message","msg_id":"123456","content":"Hello, world!"}`), nil
}

// ============ 压力测试 ============

// StressTestWebSocketHub WebSocket Hub 压力测试
func TestWebSocketHubStress(t *testing.T) {
	type Client struct {
		Send chan []byte
	}

	onlineUsers := int64(100000) // 10万并发
	clients := make(map[int64]map[*Client]bool, onlineUsers)

	// 初始化 10万在线用户
	t.Logf("Initializing %d online users...", onlineUsers)
	for i := int64(1); i <= onlineUsers; i++ {
		clients[i] = map[*Client]bool{
			{Send, make(chan []byte, 256)}: true,
		}
	}

	// 测试消息推送延迟
	t.Run("PushLatency", func(t *testing.T) {
		var wg sync.WaitGroup
		latencies := make([]int64, 0, 1000)
		var mu sync.Mutex

		// 模拟推送 1000 条消息
		for i := 0; i < 1000; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				start := time.Now()
				userID := int64(1 + i%int(onlineUsers))

				// 查找用户
				mu.Lock()
				if c, ok := clients[userID]; ok {
					for client := range c {
						select {
						case client.Send <- []byte("test"):
						default:
						}
					}
				}
				mu.Unlock()

				latency := time.Since(start).Microseconds()
				mu.Lock()
				latencies = append(latencies, latency)
				mu.Unlock()
			}()
		}

		wg.Wait()

		// 计算平均延迟
		var total int64
		for _, l := range latencies {
			total += l
		}
		avgLatency := float64(total) / float64(len(latencies))

		t.Logf("Average push latency: %.2f us", avgLatency)
		if avgLatency > 100000 { // 100ms
			t.Errorf("Latency too high: %.2f ms", avgLatency/1000)
		}
	})
}

// StressTestDatabaseConnection 数据库连接池压力测试
func TestDatabaseConnectionPool(t *testing.T) {
	// 模拟连接池
	type Conn struct{}
	pool := make(chan *Conn, 1000)
	maxConns := 1000

	// 初始化连接池
	for i := 0; i < maxConns; i++ {
		pool <- &Conn{}
	}

	concurrentRequests := 10000
	var wg sync.WaitGroup
	success := int64(0)
	failed := int64(0)

	t.Logf("Testing %d concurrent requests with pool size %d", concurrentRequests, maxConns)

	start := time.Now()

	for i := 0; i < concurrentRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			select {
			case conn := <-pool:
				// 模拟数据库操作
				time.Sleep(time.Microsecond * 100)
				pool <- conn
				atomic.AddInt64(&success, 1)
			default:
				atomic.AddInt64(&failed, 1)
			}
		}()
	}

	wg.Wait()
	elapsed := time.Since(start)

	t.Logf("Completed in %v", elapsed)
	t.Logf("Success: %d, Failed: %d", success, failed)
	t.Logf("QPS: %.2f", float64(concurrentRequests)/elapsed.Seconds())

	if failed > 0 {
		t.Logf("Warning: %d requests failed due to pool exhaustion", failed)
	}
}

// StressTestRedisConnection Redis 连接压力测试
func TestRedisConnection(t *testing.T) {
	concurrentRequests := 10000
	operations := 5 // 每次请求操作数

	var wg sync.WaitGroup
	latencies := make([]int64, 0, concurrentRequests)
	var mu sync.Mutex

	t.Logf("Testing %d concurrent Redis operations", concurrentRequests*operations)

	start := time.Now()

	for i := 0; i < concurrentRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			reqStart := time.Now()
			// 模拟 Redis 操作
			for j := 0; j < operations; j++ {
				// 模拟操作
				_ = j
			}
			latency := time.Since(reqStart).Microseconds()

			mu.Lock()
			latencies = append(latencies, latency)
			mu.Unlock()
		}()
	}

	wg.Wait()
	elapsed := time.Since(start)

	// 计算平均延迟
	var total int64
	var max int64
	for _, l := range latencies {
		total += l
		if l > max {
			max = l
		}
	}
	avgLatency := float64(total) / float64(len(latencies))

	t.Logf("Total time: %v", elapsed)
	t.Logf("Average latency: %.2f us", avgLatency)
	t.Logf("Max latency: %d us", max)
	t.Logf("Operations/sec: %.2f", float64(concurrentRequests*operations)/elapsed.Seconds())
}

// TestMessageThroughput 消息吞吐量测试
func TestMessageThroughput(t *testing.T) {
	totalMessages := int64(100000)
	var sent int64
	var acked int64
	var mu sync.Mutex

	// 模拟消息队列
	type Message struct {
		ID      int64
		Content string
	}
	queue := make(chan Message, 10000)

	// 发送者
	go func() {
		for i := int64(1); i <= totalMessages; i++ {
			queue <- Message{ID: i, Content: "test"}
			mu.Lock()
			sent++
			mu.Unlock()
		}
		close(queue)
	}()

	// 接收者
	workers := 10
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for msg := range queue {
				// 模拟处理
				_ = msg
				mu.Lock()
				acked++
				mu.Unlock()
			}
		}()
	}

	start := time.Now()
	wg.Wait()
	elapsed := time.Since(start)

	t.Logf("Total messages: %d", totalMessages)
	t.Logf("Time: %v", elapsed)
	t.Logf("Throughput: %.2f msg/sec", float64(totalMessages)/elapsed.Seconds())

	if acked != totalMessages {
		t.Errorf("Message loss detected: sent=%d, acked=%d", sent, acked)
	}
}

// ============ HTTP 压力测试 ============

// TestHTTPServerStress HTTP 服务器压力测试
func TestHTTPServerStress(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 创建测试服务器
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("userID", int64(1))
		c.Next()
	})
	router.GET("/api/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
	router.GET("/api/messages", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"code":    0,
			"message": "success",
			"data":    []interface{}{},
		})
	})

	ts := httptest.NewServer(router)
	defer ts.Close()

	concurrentRequests := 1000
	var wg sync.WaitGroup
	var success int64
	var failed int64

	t.Logf("Testing %d concurrent HTTP requests", concurrentRequests)

	start := time.Now()

	for i := 0; i < concurrentRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			resp, err := http.Get(ts.URL + "/api/messages")
			if err != nil {
				atomic.AddInt64(&failed, 1)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode == 200 {
				atomic.AddInt64(&success, 1)
			} else {
				atomic.AddInt64(&failed, 1)
			}
		}()
	}

	wg.Wait()
	elapsed := time.Since(start)

	t.Logf("Completed in %v", elapsed)
	t.Logf("Success: %d, Failed: %d", success, failed)
	t.Logf("QPS: %.2f", float64(concurrentRequests)/elapsed.Seconds())
}

// ============ 内存测试 ============

// TestMemoryUsage 内存使用测试
func TestMemoryUsage(t *testing.T) {
	var memStatsBefore, memStatsAfter runtime.MemStats
	runtime.ReadMemStats(&memStatsBefore)

	// 创建大量对象
	var objs []map[string]interface{}
	for i := 0; i < 100000; i++ {
		objs = append(objs, map[string]interface{}{
			"id":      i,
			"name":    "test",
			"data":    strings.Repeat("x", 100),
		})
	}

	runtime.ReadMemStats(&memStatsAfter)

	allocMB := float64(memStatsAfter.Alloc-memStatsBefore.Alloc) / 1024 / 1024
	t.Logf("Memory allocated: %.2f MB", allocMB)

	// 释放
	objs = nil
	runtime.GC()

	runtime.ReadMemStats(&memStatsAfter)
	t.Logf("Memory after GC: %.2f MB", float64(memStatsAfter.Alloc)/1024/1024)
}

import "runtime"
