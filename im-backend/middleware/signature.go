package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// SignatureConfig 签名配置
type SignatureConfig struct {
	SecretKey    string
	ExpiryTime   time.Duration
	AllowedHosts []string // 允许的域名白名单
}

// DefaultSignatureConfig 默认签名配置
var DefaultSignatureConfig = &SignatureConfig{
	SecretKey:    os.Getenv("URL_SIGN_SECRET_KEY"),
	ExpiryTime:   1 * time.Hour,
	AllowedHosts: []string{"localhost", "example.com"},
}

// VerifySignatureMiddleware 验证URL签名中间件 (P2-3)
func VerifySignatureMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 如果没有配置密钥，则跳过验证（开发环境）
		if DefaultSignatureConfig.SecretKey == "" {
			c.Next()
			return
		}

		// 获取签名参数
		signature := c.Query("sig")
		expireStr := c.Query("expire")

		if signature == "" || expireStr == "" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    40301,
				"message": "缺少签名参数",
			})
			return
		}

		// 验证过期时间
		expireTime, err := strconv.ParseInt(expireStr, 10, 64)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    40302,
				"message": "无效的过期时间",
			})
			return
		}

		if time.Now().Unix() > expireTime {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    40303,
				"message": "链接已过期",
			})
			return
		}

		// 验证签名
		path := c.Request.URL.Path
		expectedSig := generateSignature(path, expireStr)

		if !hmac.Equal([]byte(signature), []byte(expectedSig)) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    40304,
				"message": "签名验证失败",
			})
			return
		}

		c.Next()
	}
}

// generateSignature 生成 HMAC-SHA256 签名 (P2-3)
func generateSignature(path, expireStr string) string {
	message := fmt.Sprintf("%s:%s", path, expireStr)
	mac := hmac.New(sha256.New, []byte(DefaultSignatureConfig.SecretKey))
	mac.Write([]byte(message))
	return hex.EncodeToString(mac.Sum(nil))
}

// HotlinkProtectionMiddleware 防盗链中间件 (P2-4)
func HotlinkProtectionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 如果没有配置允许的域名，则跳过验证（开发环境）
		if len(DefaultSignatureConfig.AllowedHosts) == 0 {
			c.Next()
			return
		}

		referer := c.Request.Referer()
		host := c.Request.Host

		// 检查是否为直接访问（没有 Referer）
		// 对于图片等静态资源，通常需要 Referer
		if referer == "" {
			// 允许来自允许域名的请求
			allowed := false
			for _, allowedHost := range DefaultSignatureConfig.AllowedHosts {
				if strings.Contains(host, allowedHost) {
					allowed = true
					break
				}
			}
			if !allowed {
				// 对于直接访问，检查是否有有效的签名
				signature := c.Query("sig")
				if signature == "" {
					c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
						"code":    40305,
						"message": "防盗链验证失败",
					})
					return
				}
			}
		} else {
			// 检查 Referer 是否在白名单中
			refererAllowed := false
			for _, allowedHost := range DefaultSignatureConfig.AllowedHosts {
				if strings.Contains(referer, allowedHost) {
					refererAllowed = true
					break
				}
			}

			if !refererAllowed {
				// 检查是否有有效签名
				signature := c.Query("sig")
				if signature == "" {
					c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
						"code":    40305,
						"message": "防盗链验证失败",
					})
					return
				}
			}
		}

		c.Next()
	}
}

// GenerateSignedURL 生成带签名的URL (P2-3)
func GenerateSignedURL(path string, expiry time.Duration) string {
	if DefaultSignatureConfig.SecretKey == "" {
		return path
	}

	expireTime := time.Now().Add(expiry).Unix()
	expireStr := fmt.Sprintf("%d", expireTime)
	signature := generateSignature(path, expireStr)

	return fmt.Sprintf("%s?expire=%d&sig=%s", path, expireTime, signature)
}
