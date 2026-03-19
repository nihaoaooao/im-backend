package monitoring

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// PrometheusMiddleware Prometheus 中间件
func PrometheusMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == "/metrics" || c.Request.URL.Path == "/health" {
			c.Next()
			return
		}

		start := time.Now()
		path := c.FullPath()
		if path == "" {
			path = "unknown"
		}

		// 记录请求中
		HTTPRequestsInFlight.Inc()
		defer HTTPRequestsInFlight.Dec()

		c.Next()

		duration := time.Since(start).Seconds()
		status := c.Writer.Status()
		method := c.Request.Method

		// 记录指标
		HTTPRequestsTotal.WithLabelValues(method, path, statusToString(status)).Inc()
		HTTPRequestDuration.WithLabelValues(method, path).Observe(duration)
	}
}

// PrometheusHandler 返回 Prometheus 指标
func PrometheusHandler() gin.HandlerFunc {
	handler := promhttp.Handler()
	return func(c *gin.Context) {
		handler.ServeHTTP(c.Writer, c.Request)
	}
}

// HealthHandler 健康检查
func HealthHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"timestamp": time.Now().Unix(),
		})
	}
}

// ReadyHandler 就绪检查
func ReadyHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 检查数据库连接
		// 检查 Redis 连接
		// 检查其他依赖
		c.JSON(http.StatusOK, gin.H{
			"status": "ready",
			"timestamp": time.Now().Unix(),
		})
	}
}

// MetricsHandler 返回 JSON 格式的指标
func MetricsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		registry := prometheus.DefaultRegistry
		
		metrics, err := registry.Gather()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}

		result := make([]map[string]interface{}, 0, len(metrics))
		for _, m := range metrics {
			metric := map[string]interface{}{
				"name": m.GetName(),
				"help": m.GetHelp(),
				"type": m.GetType().String(),
			}
			
			// 添加指标数据
			var metricData []map[string]interface{}
			for _, metricFamily := range metrics {
				for _, m := range metricFamily.GetMetric() {
					var labels map[string]string
					if m.Label != nil {
						labels = make(map[string]string)
						for _, l := range m.Label {
							labels[l.GetName()] = l.GetValue()
						}
					}
					
					var value float64
					switch t := m.GetMetric().(type) {
					case *prometheus.Metric:
						// 处理不同类型的指标
					}
					
					metricData = append(metricData, map[string]interface{}{
						"labels": labels,
						"value": value,
					})
				}
			}
			metric["metrics"] = metricData
			result = append(result, metric)
		}

		c.JSON(http.StatusOK, gin.H{
			"metrics": result,
		})
	}
}

// CustomMetrics 自定义指标收集
type CustomMetrics struct {
	RequestCount    *prometheus.CounterVec
	RequestDuration *prometheus.HistogramVec
	ResponseSize    *prometheus.SummaryVec
}

// NewCustomMetrics 创建自定义指标
func NewCustomMetrics() *CustomMetrics {
	return &CustomMetrics{
		RequestCount: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "app_requests_total",
				Help: "Total number of requests",
			},
			[]string{"method", "endpoint", "status"},
		),
		RequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "app_request_duration_seconds",
				Help:    "Request duration in seconds",
				Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
			},
			[]string{"method", "endpoint"},
		),
		ResponseSize: prometheus.NewSummaryVec(
			prometheus.SummaryOpts{
				Name: "app_response_size_bytes",
				Help: "Response size in bytes",
			},
			[]string{"method", "endpoint"},
		),
	}
}

// Register 注册指标
func (m *CustomMetrics) Register() {
	prometheus.MustRegister(m.RequestCount)
	prometheus.MustRegister(m.RequestDuration)
	prometheus.MustRegister(m.ResponseSize)
}

// RecordRequest 记录请求
func (m *CustomMetrics) RecordRequest(method, endpoint, status string, duration time.Duration, size int) {
	m.RequestCount.WithLabelValues(method, endpoint, status).Inc()
	m.RequestDuration.WithLabelValues(method, endpoint).Observe(duration.Seconds())
	m.ResponseSize.WithLabelValues(method, endpoint).Observe(float64(size))
}

// ============ 业务指标中间件 ============

// BusinessMetricsMiddleware 业务指标中间件
func BusinessMetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// 根据路由记录业务指标
		path := c.FullPath()
		
		switch {
		case path == "/api/v1/messages/send":
			MessagesSent.WithLabelValues("direct", "text").Inc()
		case path == "/api/v1/media/upload":
			mediaType := c.PostForm("type")
			if mediaType == "" {
				mediaType = "unknown"
			}
			MediaUploadsTotal.WithLabelValues(mediaType, strconv.Itoa(c.Writer.Status())).Inc()
		case path == "/api/v1/messages/read":
			ReadReceipts.Inc()
		}
	}
}
