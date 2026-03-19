package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"im-backend/config"
	"im-backend/model"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB 全局数据库连接
var DB *gorm.DB
var SQLDB *sql.DB

// InitDB 初始化数据库连接
func InitDB(cfg *config.Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName,
	)

	// 先用原生 sql.DB 创建连接池
	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// 配置连接池参数
	sqlDB.SetMaxOpenConns(cfg.DBMaxOpen)    // 最大打开连接数
	sqlDB.SetMaxIdleConns(cfg.DBMaxIdle)    // 最大空闲连接数
	sqlDB.SetConnMaxLifetime(time.Hour)      // 连接生命周期
	sqlDB.SetConnMaxIdleTime(10 * time.Minute) // 空闲连接超时

	// 测试连接
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// 使用 GORM
	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN:                  dsn,
		PreferSimpleProtocol: true,
	}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// 设置 GORM 的连接池
	sqlDB, err = db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database pool: %w", err)
	}
	sqlDB.SetMaxOpenConns(cfg.DBMaxOpen)
	sqlDB.SetMaxIdleConns(cfg.DBMaxIdle)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// 自动迁移表结构
	err = db.AutoMigrate(
		&model.User{},
		&model.Conversation{},
		&model.Message{},
		&model.ConversationMember{},
		&model.MessageRead{},
		&model.Friendship{},
		&model.UserToken{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	DB = db
	SQLDB = sqlDB

	fmt.Printf("[DB] Connected to PostgreSQL: %s:%d/%s\n", cfg.DBHost, cfg.DBPort, cfg.DBName)
	fmt.Printf("[DB] Connection pool: max_open=%d, max_idle=%d\n", cfg.DBMaxOpen, cfg.DBMaxIdle)

	return db, nil
}

// Redis 全局 Redis 客户端
var Redis *redis.Client

// InitRedis 初始化 Redis 连接
func InitRedis(cfg *config.Config) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", cfg.RedisHost, cfg.RedisPort),
		Password:     cfg.RedisPassword,
		DB:           cfg.RedisDB,
		PoolSize:     100,       // 连接池大小
		MinIdleConns: 10,       // 最小空闲连接
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolTimeout:  10 * time.Second,
	})

	ctx := context.Background()
	_, err := client.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	Redis = client

	fmt.Printf("[Redis] Connected to Redis: %s:%d\n", cfg.RedisHost, cfg.RedisPort)
	fmt.Printf("[Redis] Connection pool: size=%d, min_idle=%d\n", 100, 10)

	return client, nil
}

// GetDB 获取数据库连接
func GetDB() *gorm.DB {
	return DB
}

// GetRedis 获取 Redis 客户端
func GetRedis() *redis.Client {
	return Redis
}

// Close 关闭所有连接
func Close() {
	if SQLDB != nil {
		SQLDB.Close()
	}
	if Redis != nil {
		Redis.Close()
	}
}
