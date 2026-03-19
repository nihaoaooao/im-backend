package config

import (
	"log"
	"os"
	"time"
)

var (
	// Log 日志
	Log *log.Logger

	// ServerConfig 服务器配置
	ServerConfig = &Server{}
)

// Server 服务器配置
type Server struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
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
