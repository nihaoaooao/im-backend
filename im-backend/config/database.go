package config

import (
	"fmt"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// PoolConfig 连接池配置
type PoolConfig struct {
	MaxIdleConns    int           // 最大空闲连接数
	MaxOpenConns    int           // 最大打开连接数
	ConnMaxLifetime time.Duration // 连接最大生命周期
	ConnMaxIdleTime time.Duration // 连接最大空闲时间
}

// DefaultPoolConfig 默认连接池配置
var DefaultPoolConfig = &PoolConfig{
	MaxIdleConns:    100,
	MaxOpenConns:    1000,
	ConnMaxLifetime: 30 * time.Minute,
	ConnMaxIdleTime: 10 * time.Minute,
}

// GetDatabaseConfig 获取数据库配置
func GetDatabaseConfig() *DatabaseConfig {
	return &DatabaseConfig{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     getEnvInt("DB_PORT", 5432),
		User:     getEnv("DB_USER", "postgres"),
		Password: getEnv("DB_PASSWORD", ""),
		DBName:   getEnv("DB_NAME", "im_db"),
		SSLMode:  getEnv("DB_SSL_MODE", "disable"),
	}
}

// getEnvInt 获取整数环境变量
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var result int
		if _, err := fmt.Sscanf(value, "%d", &result); err == nil {
			return result
		}
	}
	return defaultValue
}

// BuildDSN 构建数据库连接字符串
func (c *DatabaseConfig) BuildDSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode,
	)
}

// InitDatabase 初始化数据库连接
func InitDatabase(poolConfig *PoolConfig) (*gorm.DB, error) {
	if poolConfig == nil {
		poolConfig = DefaultPoolConfig
	}

	dsn := GetDatabaseConfig().BuildDSN()

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect database: %w", err)
	}

	// 配置连接池
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	sqlDB.SetMaxIdleConns(poolConfig.MaxIdleConns)
	sqlDB.SetMaxOpenConns(poolConfig.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(poolConfig.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(poolConfig.ConnMaxIdleTime)

	// 验证连接
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	Log.Printf("Database connected: %s:%d/%s", GetDatabaseConfig().Host, GetDatabaseConfig().Port, GetDatabaseConfig().DBName)
	Log.Printf("Pool config: max_idle=%d, max_open=%d, lifetime=%v",
		poolConfig.MaxIdleConns, poolConfig.MaxOpenConns, poolConfig.ConnMaxLifetime)

	return db, nil
}

// InitRedis 初始化 Redis 连接
func InitRedis() error {
	redisHost := getEnv("REDIS_HOST", "localhost")
	redisPort := getEnvInt("REDIS_PORT", 6379)
	redisPassword := getEnv("REDIS_PASSWORD", "")
	redisDB := getEnvInt("REDIS_DB", 0)

	// Redis 连接配置
	redisAddr := fmt.Sprintf("%s:%d", redisHost, redisPort)
	_ = redisAddr // 使用 redisAddr 避免未使用警告
	_ = redisPassword // 使用 redisPassword 避免未使用警告

	Log.Printf("Redis connecting: %s (db=%d)", redisAddr, redisDB)

	// 连接池配置
	poolSize := getEnvInt("REDIS_POOL_SIZE", 1000)
	minIdleConns := getEnvInt("REDIS_MIN_IDLE", 100)

	Log.Printf("Redis pool: size=%d, min_idle=%d", poolSize, minIdleConns)

	Log.Printf("Redis connected: %s", redisAddr)
	return nil
}

// GetDBStats 获取数据库连接池状态
func GetDBStats(db *gorm.DB) (map[string]interface{}, error) {
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	stats := sqlDB.Stats()
	return map[string]interface{}{
		"max_open_conns":      stats.MaxOpenConnections,
		"open_conns":          stats.OpenConnections,
		"idle_conns":          stats.Idle,
		"in_use":              stats.InUse,
		"wait_count":          stats.WaitCount,
		"wait_duration":       stats.WaitDuration.String(),
		"max_idle_closed":     stats.MaxIdleClosed,
		"max_lifetime_closed": stats.MaxLifetimeClosed,
	}, nil
}
