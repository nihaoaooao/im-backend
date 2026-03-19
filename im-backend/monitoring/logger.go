package monitoring

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

// LogLevel 日志级别
type LogLevel int

const (
	DebugLevel LogLevel = iota
	InfoLevel
	WarnLevel
	ErrorLevel
	FatalLevel
)

func (l LogLevel) String() string {
	switch l {
	case DebugLevel:
		return "DEBUG"
	case InfoLevel:
		return "INFO"
	case WarnLevel:
		return "WARN"
	case ErrorLevel:
		return "ERROR"
	case FatalLevel:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// LogEntry 日志条目
type LogEntry struct {
	Timestamp string                 `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Caller    string                 `json:"caller,omitempty"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
	Stack     string                 `json:"stack,omitempty"`
}

// Logger 结构化日志记录器
type Logger struct {
	mu         sync.Mutex
	output     io.Writer
	level      LogLevel
	fields     map[string]interface{}
	caller     bool
	stackTrace bool
}

// NewLogger 创建日志记录器
func NewLogger(output io.Writer, level LogLevel) *Logger {
	return &Logger{
		output: output,
		level:  level,
		fields: make(map[string]interface{}),
		caller: true,
	}
}

// DefaultLogger 默认日志记录器
var DefaultLogger = NewLogger(os.Stdout, InfoLevel)

// WithFields 添加全局字段
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()

	newLogger := &Logger{
		output: l.output,
		level:  l.level,
		fields: make(map[string]interface{}),
		caller: l.caller,
	}

	// 合并字段
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}
	for k, v := range fields {
		newLogger.fields[k] = v
	}

	return newLogger
}

// WithCaller 是否显示调用者信息
func (l *Logger) WithCaller(caller bool) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.caller = caller
	return l
}

// WithStackTrace 是否记录堆栈跟踪
func (l *Logger) WithStackTrace(stackTrace bool) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.stackTrace = stackTrace
	return l
}

// log 内部日志记录方法
func (l *Logger) log(level LogLevel, message string, fields map[string]interface{}) {
	if level < l.level {
		return
	}

	entry := LogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Level:     level.String(),
		Message:   message,
		Fields:    fields,
	}

	// 添加调用者信息
	if l.caller {
		_, file, line, ok := runtime.Caller(2)
		if ok {
			entry.Caller = fmt.Sprintf("%s:%d", filepath.Base(file), line)
		}
	}

	// 添加堆栈跟踪
	if l.stackTrace && level >= ErrorLevel {
		buf := make([]byte, 4096)
		n := runtime.Stack(buf, false)
		entry.Stack = string(buf[:n])
	}

	// 序列化并输出
	data, err := json.Marshal(entry)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to marshal log entry: %v\n", err)
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	fmt.Fprintln(l.output, string(data))
}

// Debug 调试日志
func (l *Logger) Debug(message string, fields ...interface{}) {
	l.log(DebugLevel, message, pairsToMap(fields...))
}

// Info 信息日志
func (l *Logger) Info(message string, fields ...interface{}) {
	l.log(InfoLevel, message, pairsToMap(fields...))
}

// Warn 警告日志
func (l *Logger) Warn(message string, fields ...interface{}) {
	l.log(WarnLevel, message, pairsToMap(fields...))
}

// Error 错误日志
func (l *Logger) Error(message string, fields ...interface{}) {
	l.log(ErrorLevel, message, pairsToMap(fields...))
}

// Fatal 致命错误日志
func (l *Logger) Fatal(message string, fields ...interface{}) {
	l.log(FatalLevel, message, pairsToMap(fields...))
	os.Exit(1)
}

// ============ 全局日志方法 ============

// Debug 调试日志
func Debug(message string, fields ...interface{}) {
	DefaultLogger.Debug(message, fields...)
}

// Info 信息日志
func Info(message string, fields ...interface{}) {
	DefaultLogger.Info(message, fields...)
}

// Warn 警告日志
func Warn(message string, fields ...interface{}) {
	DefaultLogger.Warn(message, fields...)
}

// Error 错误日志
func Error(message string, fields ...interface{}) {
	DefaultLogger.Error(message, fields...)
}

// Fatal 致命错误日志
func Fatal(message string, fields ...interface{}) {
	DefaultLogger.Fatal(message, fields...)
}

// ============ 工具函数 ============

// pairsToMap 将键值对转换为 map
func pairsToMap(pairs ...interface{}) map[string]interface{} {
	fields := make(map[string]interface{})
	for i := 0; i < len(pairs)-1; i += 2 {
		key, ok := pairs[i].(string)
		if !ok {
			continue
		}
		fields[key] = pairs[i+1]
	}
	return fields
}

// ============ 日志中间件 ============

// LogMiddleware 日志中间件
func LogMiddleware() func(c interface{}) {
	return func(c interface{}) {
		// 记录请求日志
		Info("request",
			"method", getMethod(c),
			"path", getPath(c),
			"client_ip", getClientIP(c),
		)
	}
}

// ============ 日志导出器接口 ============

// LogExporter 日志导出器接口
type LogExporter interface {
	Export(entry LogEntry) error
	Close() error
}

// FileExporter 文件导出器
type FileExporter struct {
	file *os.File
}

func NewFileExporter(path string) (*FileExporter, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	return &FileExporter{file: file}, nil
}

func (e *FileExporter) Export(entry LogEntry) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	_, err = e.file.Write(append(data, '\n'))
	return err
}

func (e *FileExporter) Close() error {
	return e.file.Close()
}

// ============ 应用日志 ============

// AppLogger 应用日志
type AppLogger struct {
	Service  string `json:"service"`
	Version  string `json:"version"`
	Env      string `json:"env"`
}

// NewAppLogger 创建应用日志
func NewAppLogger(service, version, env string) *AppLogger {
	return &AppLogger{
		Service: service,
		Version: version,
		Env:     env,
	}
}

// LogStartup 记录启动信息
func (l *AppLogger) LogStartup() {
	Info("Application started",
		"service", l.Service,
		"version", l.Version,
		"environment", l.Env,
		"timestamp", time.Now().UTC().Format(time.RFC3339),
	)
}

// LogShutdown 记录关闭信息
func (l *AppLogger) LogShutdown() {
	Info("Application shutting down",
		"service", l.Service,
		"timestamp", time.Now().UTC().Format(time.RFC3339),
	)
}

// ============ 请求日志 ============

// RequestLogger 请求日志
func RequestLogger(method, path, ip, userAgent string, status int, duration time.Duration) {
	Info("HTTP Request",
		"method", method,
		"path", path,
		"client_ip", ip,
		"user_agent", userAgent,
		"status", status,
		"duration_ms", duration.Milliseconds(),
	)
}

// WebSocketLogger WebSocket 日志
func WebSocketLogger(action, userID string, extras map[string]interface{}) {
	fields := map[string]interface{}{
		"action": action,
		"user_id": userID,
	}
	for k, v := range extras {
		fields[k] = v
	}
	Info("WebSocket", fields...)
}

// BusinessLogger 业务日志
func BusinessLogger(action string, extras map[string]interface{}) {
	Info("Business", extras...)
}

// ============ 模拟接口实现 ============

func getMethod(c interface{}) string { return "GET" }
func getPath(c interface{}) string   { return "/" }
func getClientIP(c interface{}) string { return "127.0.0.1" }
