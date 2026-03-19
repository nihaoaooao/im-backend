package main

import (
	"context"
	"im-backend/api"
	"im-backend/config"
	"im-backend/middleware"
	"im-backend/models"
	"im-backend/service"
	"im-backend/ws"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	// 初始化配置
	config.InitConfig()

	// 初始化数据库
	db, err := config.InitDatabase(nil)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// 自动迁移数据库表
	if err := db.AutoMigrate(
		&models.User{},
		&models.Message{},
		&models.MessageRecall{},
		&models.MessageRead{},
		&models.ConversationUnreadCount{},
		&models.UserMediaQuota{},
	); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// 创建Hub
	hub := ws.NewHub()
	go hub.Run()

	// 创建用户服务
	userService := service.NewUserService(db)

	// 创建消息服务
	messageService := service.NewMessageService(nil, hub, nil)

	// 创建认证处理器
	authHandler := api.NewAuthHandler(userService)

	// 创建撤回处理器
	recallHandler := api.NewMessageRecallHandler(messageService)

	// 创建撤回定时检查器（每分钟检查一次）
	scheduler := service.NewRecallScheduler(messageService, 1*time.Minute)
	scheduler.Start()
	defer scheduler.Stop()

	// 设置Gin模式
	gin.SetMode(gin.ReleaseMode)
	if os.Getenv("GIN_MODE") == "debug" {
		gin.SetMode(gin.DebugMode)
	}

	// 创建路由器
	router := gin.New()

	// 设置最大文件上传大小: 100MB (P0-2: 防止内存耗尽攻击)
	router.MaxMultipartMemory = config.ServerConfig.MaxMultipartMemory

	router.Use(gin.Logger())

	// CORS中间件
	router.Use(corsMiddleware())

	// API路由
	apiGroup := router.Group("/api/v1")
	{
		// 认证路由（不需要认证，但有速率限制）
		// PT-005: 登录每分钟 5 次，注册每分钟 3 次
		authGroup := apiGroup.Group("/auth")
		{
			authGroup.POST("/login", middleware.RateLimitMiddleware(5, time.Minute), authHandler.Login)
			authGroup.POST("/register", middleware.RateLimitMiddleware(3, time.Minute), authHandler.Register)
		}

		// 需要认证的路由
		protectedGroup := apiGroup.Group("")
		protectedGroup.Use(authMiddleware())
		{
			// 认证相关
			protectedGroup.POST("/auth/logout", authHandler.Logout)
			protectedGroup.POST("/auth/refresh", authHandler.RefreshToken)
			protectedGroup.GET("/auth/me", authHandler.GetCurrentUser)

			// 消息撤回路由
			protectedGroup.POST("/messages/recall", recallHandler.RecallMessage)
			protectedGroup.GET("/messages/recallable", recallHandler.GetRecallableMessages)
		}

		// WebSocket路由
		apiGroup.GET("/ws", handleWebSocket(hub))
	}

	// 健康检查
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "ok",
			"timestamp": time.Now().Unix(),
		})
	})

	// 创建服务器
	srv := &http.Server{
		Addr:         ":" + config.ServerConfig.Port,
		Handler:      router,
		ReadTimeout:  config.ServerConfig.ReadTimeout,
		WriteTimeout: config.ServerConfig.WriteTimeout,
	}

	// 启动服务器
	go func() {
		config.Log.Printf("Starting server on port %s", config.ServerConfig.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	config.Log.Println("Shutting down server...")

	// 优雅关闭
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	config.Log.Println("Server exited")
}

func corsMiddleware() gin.HandlerFunc {
	// PT-004: 从环境变量获取允许的域名列表
	allowedOriginsEnv := os.Getenv("CORS_ALLOWED_ORIGINS")
	var allowedOrigins []string
	if allowedOriginsEnv != "" {
		allowedOrigins = strings.Split(allowedOriginsEnv, ",")
	} else {
		// 默认允许的域名（开发环境）
		allowedOrigins = []string{
			"http://localhost:3000",
			"http://localhost:8080",
			"http://localhost:5173",
		}
	}

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// 检查 Origin 是否在白名单中
		isAllowed := false
		for _, allowedOrigin := range allowedOrigins {
			if origin == allowedOrigin {
				isAllowed = true
				break
			}
		}

		// 如果不在白名单中，拒绝请求
		if !isAllowed && origin != "" {
			c.Header("Access-Control-Allow-Origin", "")
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    40301,
				"message": "CORS: Origin not allowed",
			})
			return
		}

		// 只对白名单内的域名返回允许的 Origin
		if origin != "" {
			c.Header("Access-Control-Allow-Origin", origin)
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Requested-With")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Max-Age", "86400") // 24小时

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func authMiddleware() gin.HandlerFunc {
	// 使用JWT认证中间件
	return middleware.JWTAuthMiddleware()
}

func handleWebSocket(hub *ws.Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 这里应该实现WebSocket升级
		// 简化版本
		c.JSON(http.StatusOK, gin.H{"message": "WebSocket endpoint"})
	}
}
