package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	ErrInvalidToken   = errors.New("无效的token")
	ErrTokenExpired   = errors.New("token已过期")
	ErrSignature      = errors.New("签名验证失败")
	ErrInvalidAlgorithm = errors.New("无效的算法")
)

// AllowedAlgorithm 强制允许的算法
const AllowedAlgorithm = "HS256"

// JWTConfig JWT配置
type JWTConfig struct {
	SecretKey     string        // 签名密钥
	ExpiryTime    time.Duration // token过期时间
	Issuer        string        // token签发者
	AllowedAlgs   []string      // 支持的算法
}

// DefaultJWTConfig 默认JWT配置
var DefaultJWTConfig = &JWTConfig{
	SecretKey:     os.Getenv("JWT_SECRET_KEY"),
	ExpiryTime:    24 * time.Hour, // 默认24小时
	Issuer:        "im-backend",
	AllowedAlgs:   []string{"HS256"},
}

// Claims JWT声明
type Claims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	Issuer   string `json:"iss"`
	IssuedAt int64  `json:"iat"`
	ExpireAt int64  `json:"exp"`
}

// JWTAuthMiddleware JWT认证中间件
func JWTAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    40101,
				"message": "缺少Authorization header",
			})
			return
		}

		// 检查Bearer格式
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    40102,
				"message": "Authorization header格式错误，应为 'Bearer <token>'",
			})
			return
		}

		token := parts[1]

		// 解析和验证token
		claims, err := ValidateToken(token)
		if err != nil {
			status := http.StatusUnauthorized
			message := "token验证失败"

			switch err {
			case ErrTokenExpired:
				message = "token已过期，请重新登录"
			case ErrInvalidToken:
				message = "无效的token"
			case ErrSignature:
				message = "token签名验证失败"
			}

			c.AbortWithStatusJSON(status, gin.H{
				"code":    40103,
				"message": message,
			})
			return
		}

		// 将用户信息存入context
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)

		c.Next()
	}
}

// GenerateToken 生成JWT token
func GenerateToken(userID int64, username string) (string, error) {
	if DefaultJWTConfig.SecretKey == "" {
		return "", errors.New("JWT密钥未配置")
	}

	now := time.Now().Unix()
	expiry := now + int64(DefaultJWTConfig.ExpiryTime.Seconds())

	claims := Claims{
		UserID:   userID,
		Username: username,
		Issuer:   DefaultJWTConfig.Issuer,
		IssuedAt: now,
		ExpireAt: expiry,
	}

	// 创建token
	header := map[string]string{
		"alg": "HS256",
		"typ": "JWT",
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", err
	}

	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	// Base64URL编码
	headerEncoded := base64.URLEncoding.EncodeToString([]byte(headerJSON))
	claimsEncoded := base64.URLEncoding.EncodeToString([]byte(claimsJSON))

	// 生成签名
	signature := generateHMACSignature(headerEncoded + "." + claimsEncoded)

	// 组合token
	token := headerEncoded + "." + claimsEncoded + "." + signature

	return token, nil
}

// ValidateToken 验证JWT token
func ValidateToken(token string) (*Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, ErrInvalidToken
	}

	headerEncoded := parts[0]
	claimsEncoded := parts[1]
	signature := parts[2]

	// ========== PT-001: 算法验证（必须首先执行）==========
	// 解码 header，验证算法
	headerJSON, err := base64.URLEncoding.DecodeString(headerEncoded)
	if err != nil {
		return nil, ErrInvalidToken
	}

	var header struct {
		Alg string `json:"alg"`
		Typ string `json:"typ"`
	}
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return nil, ErrInvalidToken
	}

	// 拒绝 "none" 算法（这是最常见的安全攻击）
	if strings.ToLower(header.Alg) == "none" {
		return nil, ErrInvalidAlgorithm
	}

	// 强制要求 HS256 算法
	if header.Alg != AllowedAlgorithm {
		return nil, ErrInvalidAlgorithm
	}
	// ========== 算法验证结束 ==========

	// 验证签名
	expectedSignature := generateHMACSignature(headerEncoded + "." + claimsEncoded)
	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return nil, ErrSignature
	}

	// 解析claims
	claimsJSON, err := base64.URLEncoding.DecodeString(claimsEncoded)
	if err != nil {
		return nil, ErrInvalidToken
	}

	var claims Claims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return nil, ErrInvalidToken
	}

	// 检查过期时间
	if time.Now().Unix() > claims.ExpireAt {
		return nil, ErrTokenExpired
	}

	// 验证issuer
	if claims.Issuer != DefaultJWTConfig.Issuer {
		return nil, ErrInvalidToken
	}

	return &claims, nil
}

// generateHMACSignature 生成HMAC-SHA256签名
func generateHMACSignature(message string) string {
	mac := hmac.New(sha256.New, []byte(DefaultJWTConfig.SecretKey))
	mac.Write([]byte(message))
	return base64.URLEncoding.EncodeToString(mac.Sum(nil))
}

// GetUserID 从context获取用户ID
func GetUserID(c *gin.Context) int64 {
	userID, exists := c.Get("user_id")
	if !exists {
		return 0
	}
	return userID.(int64)
}

// GetUsername 从context获取用户名
func GetUsername(c *gin.Context) string {
	username, exists := c.Get("username")
	if !exists {
		return ""
	}
	return username.(string)
}

// RefreshToken 刷新token
func RefreshToken(token string) (string, error) {
	claims, err := ValidateToken(token)
	if err != nil {
		return "", err
	}

	// 生成新token
	return GenerateToken(claims.UserID, claims.Username)
}

// InitJWTConfig 初始化JWT配置（启动时调用）
func InitJWTConfig() error {
	secretKey := os.Getenv("JWT_SECRET_KEY")
	if secretKey == "" {
		return errors.New("JWT_SECRET_KEY环境变量未设置")
	}

	DefaultJWTConfig.SecretKey = secretKey

	// 验证密钥长度（建议至少32字符）
	if len(secretKey) < 32 {
		fmt.Println("警告: JWT密钥长度小于32字符，建议使用更长的密钥")
	}

	return nil
}
