package monitoring

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// ============ HTTP 指标 ============

	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"method", "path"},
	)

	HTTPRequestsInFlight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "http_requests_in_flight",
			Help: "Number of HTTP requests currently being processed",
		},
	)

	// ============ 业务指标 ============

	// 消息指标
	MessagesSent = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "im_messages_sent_total",
			Help: "Total number of messages sent",
		},
		[]string{"conversation_type", "content_type"},
	)

	MessagesDelivered = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "im_messages_delivered_total",
			Help: "Total number of messages delivered",
		},
		[]string{"conversation_type"},
	)

	MessagesStored = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "im_messages_stored_total",
			Help: "Total number of messages stored",
		},
	)

	// 用户指标
	ActiveUsers = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "im_active_users",
			Help: "Number of active users",
		},
	)

	UserLoginsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "im_user_logins_total",
			Help: "Total number of user logins",
		},
	)

	UserLogoutsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "im_user_logouts_total",
			Help: "Total number of user logouts",
		},
	)

	// 会话指标
	ConversationsCreated = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "im_conversations_created_total",
			Help: "Total number of conversations created",
		},
	)

	ActiveConversations = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "im_active_conversations",
			Help: "Number of active conversations",
		},
	)

	// ============ WebSocket 指标 ============

	WSConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "im_websocket_connections",
			Help: "Number of WebSocket connections",
		},
	)

	WSMessagesSent = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "im_websocket_messages_sent_total",
			Help: "Total number of WebSocket messages sent",
		},
	)

	WSMessagesReceived = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "im_websocket_messages_received_total",
			Help: "Total number of WebSocket messages received",
		},
	)

	WSMessageDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "im_websocket_message_duration_seconds",
			Help:    "WebSocket message processing duration",
			Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
		},
	)

	// ============ 数据库指标 ============

	DBQueryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "im_db_query_duration_seconds",
			Help:    "Database query duration in seconds",
			Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5},
		},
		[]string{"query_type"},
	)

	DBConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "im_db_connections",
			Help: "Number of database connections",
		},
	)

	// ============ Redis 指标 ============

	RedisCommandsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "im_redis_commands_total",
			Help: "Total number of Redis commands",
		},
		[]string{"command", "status"},
	)

	RedisCommandDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "im_redis_command_duration_seconds",
			Help:    "Redis command duration in seconds",
			Buckets: []float64{.0001, .0005, .001, .005, .01, .025, .05, .1},
		},
		[]string{"command"},
	)

	// ============ 文件上传指标 ============

	MediaUploadsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "im_media_uploads_total",
			Help: "Total number of media uploads",
		},
		[]string{"media_type", "status"},
	)

	MediaUploadDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "im_media_upload_duration_seconds",
			Help:    "Media upload duration in seconds",
			Buckets: []float64{.1, .5, 1, 2.5, 5, 10, 25, 60},
		},
		[]string{"media_type"},
	)

	MediaFileSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "im_media_file_size_bytes",
			Help:    "Media file size in bytes",
			Buckets: []float64{1024, 10240, 102400, 1048576, 10485760, 104857600},
		},
		[]string{"media_type"},
	)

	// ============ 错误指标 ============

	ErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "im_errors_total",
			Help: "Total number of errors",
		},
		[]string{"type", "code"},
	)

	// ============ 限流指标 ============

	RateLimitHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "im_rate_limit_hits_total",
			Help: "Total number of rate limit hits",
		},
		[]string{"endpoint"},
	)

	// ============ 消息撤回指标 ============

	MessageRecalls = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "im_message_recalls_total",
			Help: "Total number of message recalls",
		},
	)

	// ============ 已读回执指标 ============

	ReadReceipts = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "im_read_receipts_total",
			Help: "Total number of read receipts",
		},
	)
)

// ============ 指标工具函数 ============

// RecordDBQuery 记录数据库查询
func RecordDBQuery(queryType string, duration float64) {
	DBQueryDuration.WithLabelValues(queryType).Observe(duration)
}

// RecordRedisCommand 记录 Redis 命令
func RecordRedisCommand(command, status string, duration float64) {
	RedisCommandsTotal.WithLabelValues(command, status).Observe(duration)
	RedisCommandDuration.WithLabelValues(command).Observe(duration)
}

// RecordMediaUpload 记录媒体上传
func RecordMediaUpload(mediaType, status string, duration float64, fileSize int64) {
	MediaUploadsTotal.WithLabelValues(mediaType, status).Inc()
	MediaUploadDuration.WithLabelValues(mediaType).Observe(duration)
	MediaFileSize.WithLabelValues(mediaType).Observe(float64(fileSize))
}

// RecordError 记录错误
func RecordError(errType, code string) {
	ErrorsTotal.WithLabelValues(errType, code).Inc()
}

// RecordHTTPMetrics 记录 HTTP 指标
func RecordHTTP(method, path string, status int, duration float64) {
	HTTPRequestsTotal.WithLabelValues(method, path, statusToString(status)).Inc()
	HTTPRequestDuration.WithLabelValues(method, path).Observe(duration)
}

func statusToString(status int) string {
	switch {
	case status >= 500:
		return "5xx"
	case status >= 400:
		return "4xx"
	case status >= 300:
		return "3xx"
	case status >= 200:
		return "2xx"
	default:
		return "other"
	}
}
