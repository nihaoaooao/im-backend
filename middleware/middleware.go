package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
)

// RedisClient 可选的 Redis 客户端（用于 Token 黑名单）
var RedisClient *redis.Client

// SetRedisClient 设置 Redis 客户端
func SetRedisClient(client *redis.Client) {
	RedisClient = client
}

// Auth JWT 认证中间件
func Auth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code": 401,
				"msg":  "缺少 Authorization 头",
			})
			c.Abort()
			return
		}

		// 解析 Bearer token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code": 401,
				"msg":  "无效的 Authorization 格式",
			})
			c.Abort()
			return
		}

		tokenString := parts[1]

		// 检查 token 是否在黑名单（如果 Redis 已初始化）
		if RedisClient != nil {
			blacklistKey := "token:blacklist:" + tokenString
			_, err := RedisClient.Get(context.Background(), blacklistKey).Result()
			if err == nil {
				c.JSON(http.StatusUnauthorized, gin.H{
					"code": 401,
					"msg":  "Token 已失效",
				})
				c.Abort()
				return
			}
		}

		// 解析 JWT token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// [M004] JWT算法验证：只允许HMAC算法
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code": 401,
				"msg":  "无效的 Token",
			})
			c.Abort()
			return
		}

		// 从 token 中获取用户信息
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code": 401,
				"msg":  "无效的 Token 声明",
			})
			c.Abort()
			return
		}

		// 将 user_id 存入上下文
		userID := int64(claims["user_id"].(float64))
		c.Set("user_id", userID)
		c.Set("username", claims["username"])
		c.Next()
	}
}

// CORS 跨域中间件 - [P2] 修复：限制允许的域名
func Cors() gin.HandlerFunc {
	// 允许的域名列表（生产环境应该配置化）
	allowedOrigins := map[string]bool{
		"http://localhost":     true,
		"http://localhost:8080": true,
		"http://localhost:3000": true,
		"http://127.0.0.1":     true,
		"http://127.0.0.1:8080": true,
		"http://127.0.0.1:3000": true,
	}

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// 检查Origin是否在允许列表中
		if origin != "" && allowedOrigins[origin] {
			c.Header("Access-Control-Allow-Origin", origin)
		} else if origin == "" {
			// 同源请求，不设置Allow-Origin
		} else {
			// 不允许的Origin，拒绝请求
			c.JSON(http.StatusForbidden, gin.H{
				"code": 403,
				"msg":  "不允许的跨域请求",
			})
			c.Abort()
			return
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Header("Access-Control-Max-Age", "86400")
		c.Header("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// Recovery 恢复中间件
func Recovery() gin.HandlerFunc {
	return gin.Recovery()
}

// Logger 日志中间件
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		if status >= 400 {
			// 只记录错误日志
			// 这里可以集成日志框架
			_ = latency
			_ = path
		}
	}
}

// RateLimit 限流中间件（简单实现）
func RateLimit() gin.HandlerFunc {
	// 这里可以实现更复杂的限流逻辑
	// 暂时使用简单的示例
	return func(c *gin.Context) {
		// TODO: 实现限流
		c.Next()
	}
}
