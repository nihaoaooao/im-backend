package config

import (
	"os"
	"strconv"
)

type Config struct {
	// Server
	Port int

	// Database
	DBHost     string
	DBPort     int
	DBUser     string
	DBPassword string
	DBName     string
	DBMaxOpen   int
	DBMaxIdle   int

	// Redis
	RedisHost     string
	RedisPort     int
	RedisPassword string
	RedisDB       int

	// JWT
	JWTSecret  string
	JWTExpire int // 小时

	// OSS (可选)
	OSSEndpoint      string
	OSSAccessKey     string
	OSSAccessSecret  string
	OSSBucket        string

	// WebSocket
	WSHeartbeat int // 秒
	WSMaxConns  int
	WSPort      int // WebSocket 服务器端口
}

func Load() *Config {
	return &Config{
		// Server - 默认 8080
		Port: getIntEnv("SERVER_PORT", 8080),

		// Database - 默认 PostgreSQL 配置
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getIntEnv("DB_PORT", 5432),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", "postgres"),
		DBName:     getEnv("DB_NAME", "im_db"),
		DBMaxOpen:  getIntEnv("DB_MAX_OPEN", 100),
		DBMaxIdle:  getIntEnv("DB_MAX_IDLE", 10),

		// Redis - 默认配置
		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getIntEnv("REDIS_PORT", 6379),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getIntEnv("REDIS_DB", 0),

		// JWT
		JWTSecret:  getEnv("JWT_SECRET", "im-backend-secret-key"),
		JWTExpire:  getIntEnv("JWT_EXPIRE", 168), // 7天

		// OSS (可选)
		OSSEndpoint:     getEnv("OSS_ENDPOINT", ""),
		OSSAccessKey:    getEnv("OSS_ACCESS_KEY", ""),
		OSSAccessSecret: getEnv("OSS_ACCESS_SECRET", ""),
		OSSBucket:       getEnv("OSS_BUCKET", ""),

		// WebSocket
		WSHeartbeat: getIntEnv("WS_HEARTBEAT", 30),
		WSMaxConns:   getIntEnv("WS_MAX_CONNS", 100000),
		WSPort:       getIntEnv("WS_PORT", 8081),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
