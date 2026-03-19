package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"im-backend/config"
	"im-backend/middleware"
	"im-backend/model"
	"im-backend/queue"
	"im-backend/repository"
	"im-backend/service"
	"im-backend/ws"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func main() {
	// 加载配置
	cfg := config.Load()

	// 初始化数据库连接
	db, err := repository.InitDB(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// 初始化 Redis
	redisClient, err := repository.InitRedis(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	// 初始化服务层
	userService := service.NewUserService(db, redisClient, cfg.JWTSecret, cfg.JWTExpire)
	msgService := service.NewMessageService(db, redisClient)
	groupService := service.NewGroupService(db, redisClient)

	// 初始化 WebSocket Hub
	hub := ws.NewHub(redisClient)
	go hub.Run()

	// 设置消息服务的 Hub
	msgService.SetHub(hub)

	// 设置用户服务的 Hub
	userService.SetHub(hub)

	// ============ 初始化消息队列服务 ============
	// 创建消息处理器
	handler := queue.NewSimpleHandler(func(msg *model.Message) error {
		log.Printf("[QueueHandler] Processing message: %s", msg.MsgID)

		// 消息通过会话ID获取接收者，这里简化处理
		// 实际应该从会话成员中获取接收者
		// 暂时只记录日志
		log.Printf("[QueueHandler] Message %s ready for delivery", msg.MsgID)

		return nil
	})

	// 创建消费者服务
	consumerService := queue.NewConsumerService(redisClient, handler, 100)

	// 启动消费者（多个消费者实例）
	consumerNames := []string{
		"consumer-1",
		"consumer-2",
		"consumer-3",
	}
	if err := consumerService.Start(consumerNames); err != nil {
		log.Printf("[Queue] Failed to start consumer service: %v", err)
		// 不致命，继续运行
	}

	// 启动独立的 WebSocket 服务器（端口 8081）
	go func() {
		http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
			ws.ServeWs(hub, w, r, cfg.JWTSecret)
		})

		// WebSocket 统计接口
		http.HandleFunc("/ws/stats", func(w http.ResponseWriter, r *http.Request) {
			stats := hub.GetStats()
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"code":0,"data":%v}`, stats)
		})

		log.Printf("[WebSocket Server] Starting on :%d", cfg.WSPort)
		if err := http.ListenAndServe(fmt.Sprintf(":%d", cfg.WSPort), nil); err != nil {
			log.Fatalf("[WebSocket Server] Failed to start: %v", err)
		}
	}()

	// 设置 Gin 路由
	r := gin.Default()

	// 添加中间件
	r.Use(middleware.Cors())
	r.Use(middleware.Recovery())
	middleware.SetRedisClient(redisClient)

	// API 路由
	api := r.Group("/api")
	{
		// 认证路由
		auth := api.Group("/auth")
		{
			auth.POST("/register", userService.Register)
			auth.POST("/login", userService.Login)
			auth.POST("/refresh", userService.RefreshToken)
			auth.POST("/logout", middleware.Auth(cfg.JWTSecret), userService.Logout)
		}

		// 用户路由（需要认证）
		user := api.Group("/user")
		user.Use(middleware.Auth(cfg.JWTSecret))
		{
			user.GET("/profile", userService.GetProfile)
			user.PUT("/profile", userService.UpdateProfile)
			user.GET("/friends", userService.GetFriends)
			user.POST("/friend", userService.AddFriend)
			user.POST("/friend/respond", userService.RespondFriendRequest)
			user.GET("/friend/requests", userService.GetFriendRequests)
		}

		// 会话路由
		conv := api.Group("/conversations")
		conv.Use(middleware.Auth(cfg.JWTSecret))
		{
			conv.GET("", msgService.GetConversations)
			conv.POST("", msgService.CreateConversation)
			conv.GET("/:id", msgService.GetConversationDetail)
			conv.GET("/unread", msgService.GetUnreadCount)
		}

		// 消息路由
		msg := api.Group("/messages")
		msg.Use(middleware.Auth(cfg.JWTSecret))
		{
			msg.POST("/send", msgService.SendMessage)
			msg.GET("/history", msgService.GetHistory)
			msg.POST("/revoke", msgService.RevokeMessage)
			msg.POST("/read", msgService.MarkAsRead)
		}

		// 群组路由
		group := api.Group("/group")
		group.Use(middleware.Auth(cfg.JWTSecret))
		{
			group.POST("/create", groupService.CreateGroup)
			group.GET("/:id", groupService.GetGroupInfo)
			group.POST("/:id/member", groupService.AddMember)
			group.DELETE("/:id/member/:uid", groupService.RemoveMember)
			group.POST("/:id/mute", groupService.MuteMember)
		}

		// 文件上传
		media := api.Group("/media")
		media.Use(middleware.Auth(cfg.JWTSecret))
		{
			media.POST("/upload", msgService.UploadMedia)
		}

		// WebSocket 消息推送 API
		wsAPI := api.Group("/ws")
		wsAPI.Use(middleware.Auth(cfg.JWTSecret))
		{
			wsAPI.POST("/push", func(c *gin.Context) {
				ws.HandleWebSocketMessage(hub, c)
			})
			wsAPI.POST("/online-status", func(c *gin.Context) {
				ws.HandleGetOnlineStatus(hub, c)
			})
			wsAPI.GET("/connections/:user_id", func(c *gin.Context) {
				ws.HandleGetUserConnections(hub, c)
			})
		}

		// 消息队列管理 API
		queueAPI := api.Group("/queue")
		queueAPI.Use(middleware.Auth(cfg.JWTSecret))
		{
			queueAPI.GET("/stats", func(c *gin.Context) {
				stats := consumerService.GetStats()
				c.JSON(200, gin.H{"code": 0, "data": stats})
			})
			queueAPI.GET("/length", func(c *gin.Context) {
				ctx := c.Request.Context()
				q := queue.NewMessageQueue(redisClient)
				length, err := q.GetQueueLength(ctx)
				if err != nil {
					c.JSON(500, gin.H{"code": 500, "msg": err.Error()})
					return
				}
				c.JSON(200, gin.H{"code": 0, "data": gin.H{"length": length}})
			})
			queueAPI.POST("/clear", func(c *gin.Context) {
				ctx := c.Request.Context()
				q := queue.NewMessageQueue(redisClient)
				if err := q.ClearQueue(ctx); err != nil {
					c.JSON(500, gin.H{"code": 500, "msg": err.Error()})
					return
				}
				c.JSON(200, gin.H{"code": 0, "msg": "Queue cleared"})
			})
		}
	}

	// WebSocket 路由（通过 Gin）
	r.GET("/ws", ws.HandleWebSocketHTTP(hub, cfg.JWTSecret))

	// Swagger 文档路由
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"queue":  "running",
		})
	})

	// WebSocket 统计（独立端口）
	r.GET("/api/ws/stats", func(c *gin.Context) {
		ws.HandleGetStats(hub, c)
	})

	// 启动 HTTP 服务器
	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("[HTTP Server] Starting on %s", addr)
	log.Printf("[WebSocket] ws://localhost:%d/ws?token=<JWT>", cfg.WSPort)
	log.Printf("[Queue] Consumer service running with %d consumers", len(consumerNames))

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Println("[Server] Shutting down...")
		consumerService.Stop()
		log.Println("[Server] Queue stopped")
	}()

	if err := r.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
