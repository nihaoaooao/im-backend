package middleware

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"runtime"
	"sync/atomic"
)

// ============ 性能监控指标 ============

// Metrics 性能指标
type Metrics struct {
	TotalRequests   int64
	SuccessRequests int64
	FailedRequests  int64
	TotalLatency    int64 // 纳秒
	MaxLatency      int64
	StartTime       time.Time
}

var globalMetrics Metrics

// ResetMetrics 重置指标
func ResetMetrics() {
	globalMetrics = Metrics{StartTime: time.Now()}
}

// GetMetrics 获取当前指标
func GetMetrics() Metrics {
	return Metrics{
		TotalRequests:   atomic.LoadInt64(&globalMetrics.TotalRequests),
		SuccessRequests: atomic.LoadInt64(&globalMetrics.SuccessRequests),
		FailedRequests:  atomic.LoadInt64(&globalMetrics.FailedRequests),
		TotalLatency:    atomic.LoadInt64(&globalMetrics.TotalLatency),
		MaxLatency:      atomic.LoadInt64(&globalMetrics.MaxLatency),
		StartTime:       globalMetrics.StartTime,
	}
}

// PerformanceMonitor 性能监控中间件
func PerformanceMonitor() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// 处理请求
		c.Next()

		// 计算延迟
		latency := time.Since(start).Nanoseconds()
		atomic.AddInt64(&globalMetrics.TotalRequests, 1)
		atomic.AddInt64(&globalMetrics.TotalLatency, latency)

		// 更新最大延迟
		for {
			current := atomic.LoadInt64(&globalMetrics.MaxLatency)
			if latency <= current {
				break
			}
			if atomic.CompareAndSwapInt64(&globalMetrics.MaxLatency, current, latency) {
				break
			}
		}

		// 统计成功/失败
		if c.Writer.Status() >= 400 {
			atomic.AddInt64(&globalMetrics.FailedRequests, 1)
		} else {
			atomic.AddInt64(&globalMetrics.SuccessRequests, 1)
		}

		// 记录慢请求
		if latency > 100*1000*1000 { // 100ms
			fmt.Printf("[SLOW] %s %s latency=%v status=%d\n",
				c.Request.Method, c.Request.URL.Path, time.Duration(latency), c.Writer.Status())
		}
	}
}

// MetricsHandler 返回性能指标
func MetricsHandler(c *gin.Context) {
	m := GetMetrics()
	uptime := time.Since(m.StartTime)

	var avgLatency int64
	if m.TotalRequests > 0 {
		avgLatency = m.TotalLatency / m.TotalRequests
	}

	// 获取内存使用
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	c.JSON(200, gin.H{
		"uptime":          uptime.String(),
		"total_requests":  m.TotalRequests,
		"success_count":   m.SuccessRequests,
		"failed_count":    m.FailedRequests,
		"avg_latency_ms":  float64(avgLatency) / 1e6,
		"max_latency_ms":  float64(m.MaxLatency) / 1e6,
		"qps":             float64(m.TotalRequests) / uptime.Seconds(),
		"memory": gin.H{
			"alloc_mb":      float64(memStats.Alloc) / 1024 / 1024,
			"total_alloc_mb": float64(memStats.TotalAlloc) / 1024 / 1024,
			"sys_mb":        float64(memStats.Sys) / 1024 / 1024,
			"num_gc":        memStats.NumGC,
		},
	})
}

// ============ 限流中间件 ============

// RateLimiter 限流器
type RateLimiter struct {
	requests map[string][]int64
	limit    int
	window   time.Duration
	mu       sync.RWMutex
}

// NewRateLimiter 创建限流器
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		requests: make(map[string][]int64),
		limit:    limit,
		window:   window,
	}
	// 定期清理过期记录
	go rl.cleanup()
	return rl
}

// cleanup 清理过期记录
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.window)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now().UnixNano()
		for key, times := range rl.requests {
			var valid []int64
			for _, t := range times {
				if now-t < rl.window.Nanoseconds() {
					valid = append(valid, t)
				}
			}
			if len(valid) == 0 {
				delete(rl.requests, key)
			} else {
				rl.requests[key] = valid
			}
		}
		rl.mu.Unlock()
	}
}

// Allow 检查是否允许请求
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now().UnixNano()
	times := rl.requests[key]

	// 清理过期的
	var valid []int64
	for _, t := range times {
		if now-t < rl.window.Nanoseconds() {
			valid = append(valid, t)
		}
	}

	if len(valid) >= rl.limit {
		rl.requests[key] = valid
		return false
	}

	rl.requests[key] = append(valid, now)
	return true
}

// GetRemaining 获取剩余请求数
func (rl *RateLimiter) GetRemaining(key string) int {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	now := time.Now().UnixNano()
	times := rl.requests[key]

	var count int
	for _, t := range times {
		if now-t < rl.window.Nanoseconds() {
			count++
		}
	}

	return rl.limit - count
}

// RateLimitMiddleware 限流中间件
func RateLimitMiddleware(limit int, window time.Duration) gin.HandlerFunc {
	limiter := NewRateLimiter(limit, window)

	return func(c *gin.Context) {
		// 使用 IP 或用户 ID 作为 key
		key := c.ClientIP()
		if userID, exists := c.Get("userID"); exists {
			key = fmt.Sprintf("user:%d", userID)
		}

		if !limiter.Allow(key) {
			c.JSON(429, gin.H{
				"code":    429,
				"message": "请求过于频繁，请稍后再试",
				"retry_after": window.Seconds(),
			})
			c.Abort()
			return
		}

		remaining := limiter.GetRemaining(key)
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Header("X-RateLimit-Limit", strconv.Itoa(limit))

		c.Next()
	}
}

// ============ CORS 中间件优化 ============

// CORSMiddleware CORS 中间件
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		if origin != "" {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Max-Age", "86400")
		}

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// ============ 请求ID中间件 ============

// RequestIDMiddleware 请求ID中间件
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

func generateRequestID() string {
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), randInt())
}

var (
	randSeed int64
)

func randInt() int64 {
	return atomic.AddInt64(&randSeed, 1)
}
