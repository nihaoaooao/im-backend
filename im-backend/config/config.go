package config

import (
	"fmt"
	"log"
	"os"
	"time"
)

var (
	// Log 日志
	Log *log.Logger

	// ServerConfig 服务器配置
	ServerConfig = &Server{}

	// COSConfig 对象存储配置
	COSConfig = &COS{}
)

// Server 服务器配置
type Server struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// COS 腾讯云对象存储配置
type COS struct {
	Region      string
	SecretID    string
	SecretKey   string
	Bucket      string
	CDNDomain   string
	MaxFileSize int64 // 最大文件大小（字节），默认10MB
}

// MediaConfig 媒体配置
type MediaConfig struct {
	ImageMaxSize   int64 // 图片最大大小（字节）
	VoiceMaxSize   int64 // 语音最大大小（字节）
	VideoMaxSize   int64 // 视频最大大小（字节）
	ThumbnailSize  int   // 缩略图大小
	ImageQuality   int   // 图片压缩质量
}

// InitConfig 初始化配置
func InitConfig() {
	Log = log.New(os.Stdout, "[IM-Backend] ", log.LstdFlags|log.Lshortfile)

	port := getEnv("SERVER_PORT", "8080")
	readTimeout := getEnvDuration("SERVER_READ_TIMEOUT", 60*time.Second)
	writeTimeout := getEnvDuration("SERVER_WRITE_TIMEOUT", 60*time.Second)

	ServerConfig = &Server{
		Port:         port,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
	}

	// 初始化 COS 配置
	COSConfig = &COS{
		Region:      getEnv("COS_REGION", "ap-guangzhou"),
		SecretID:    getEnv("COS_SECRET_ID", ""),
		SecretKey:   getEnv("COS_SECRET_KEY", ""),
		Bucket:      getEnv("COS_BUCKET", "im-bucket"),
		CDNDomain:   getEnv("COS_CDN_DOMAIN", ""),
		MaxFileSize: getEnvInt64("COS_MAX_FILE_SIZE", 10*1024*1024), // 默认10MB
	}
}

// getEnvInt64 获取整数环境变量
func getEnvInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		var result int64
		if _, err := fmt.Sscanf(value, "%d", &result); err == nil {
			return result
		}
	}
	return defaultValue
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
