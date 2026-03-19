package main

import (
	"context"
	"im-backend/api"
	"im-backend/config"
	"im-backend/middleware"
	"im-backend/service"
	"im-backend/ws"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	// 初始化配置
	config.InitConfig()

	// 创建Hub
	hub := ws.NewHub()
	go hub.Run()

	// 创建消息服务
	messageService := service.NewMessageService(nil, hub, nil)

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
		// 消息撤回路由（需要认证）
		messagesGroup := apiGroup.Group("/messages")
		messagesGroup.Use(authMiddleware())
		{
			messagesGroup.POST("/recall", recallHandler.RecallMessage)
			messagesGroup.GET("/recallable", recallHandler.GetRecallableMessages)
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
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		// 这里应该配置允许的域名列表
		allowedOrigins := []string{"http://localhost:3000", "http://localhost:8080"}
		
		for _, allowed := range allowedOrigins {
			if origin == allowed {
				c.Header("Access-Control-Allow-Origin", origin)
				break
			}
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")
		c.Header("Access-Control-Allow-Credentials", "true")

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
