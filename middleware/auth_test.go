package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// TestAuthMiddleware 测试 JWT 认证中间件
func TestAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	jwtSecret := "test-secret-key"

	// 生成有效的 token
	validToken := generateTestToken(t, jwtSecret, int64(123), "testuser")

	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
		expectedUserID int64
	}{
		{
			name:           "No Authorization header",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Invalid format",
			authHeader:     "InvalidToken",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Wrong scheme",
			authHeader:     "Basic sometoken",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Valid token",
			authHeader:     "Bearer " + validToken,
			expectedStatus: http.StatusOK,
			expectedUserID: 123,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, r := gin.CreateTestContext(w)

			// 添加认证中间件
			r.Use(Auth(jwtSecret))
			r.GET("/test", func(c *gin.Context) {
				userID, _ := c.Get("user_id")
				username, _ := c.Get("username")
				c.JSON(http.StatusOK, gin.H{
					"user_id":  userID,
					"username": username,
				})
			})

			// 设置请求
			c.Request, _ = http.NewRequest("GET", "/test", nil)
			if tt.authHeader != "" {
				c.Request.Header.Set("Authorization", tt.authHeader)
			}

			// 执行
			r.ServeHTTP(w, c.Request)

			// 验证
			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedStatus == http.StatusOK {
				// 可以进一步验证返回的用户信息
				t.Logf("Response: %s", w.Body.String())
			}
		})
	}
}

// TestExpiredToken 测试过期 token
func TestExpiredToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	jwtSecret := "test-secret-key"

	// 生成过期的 token
	claims := jwt.MapClaims{
		"user_id":  123,
		"username": "testuser",
		"exp":      time.Now().Add(-1 * time.Hour).Unix(), // 已过期
		"iat":      time.Now().Add(-2 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	expiredToken, _ := token.SignedString([]byte(jwtSecret))

	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.Use(Auth(jwtSecret))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	c.Request, _ = http.NewRequest("GET", "/test", nil)
	c.Request.Header.Set("Authorization", "Bearer "+expiredToken)

	r.ServeHTTP(w, c.Request)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

// TestTokenBlacklist 测试 Token 黑名单
func TestTokenBlacklist(t *testing.T) {
	// 注意：这个测试需要真实的 Redis 连接
	// 这里只测试逻辑，不实际执行
	t.Skip("Skipping blacklist test - requires Redis connection")
}

// generateTestToken 生成测试用 Token
func generateTestToken(t *testing.T, secret string, userID int64, username string) string {
	claims := jwt.MapClaims{
		"user_id":  userID,
		"username": username,
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
		"iat":      time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}
	return tokenString
}

// BenchmarkAuthMiddleware 认证中间件性能测试
func BenchmarkAuthMiddleware(b *testing.B) {
	gin.SetMode(gin.TestMode)

	jwtSecret := "test-secret-key"
	validToken := generateTestTokenForBenchmark(jwtSecret)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		c, r := gin.CreateTestContext(w)

		r.Use(Auth(jwtSecret))
		r.GET("/test", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		c.Request, _ = http.NewRequest("GET", "/test", nil)
		c.Request.Header.Set("Authorization", "Bearer "+validToken)

		r.ServeHTTP(w, c.Request)
	}
}

// generateTestTokenForBenchmark 生成性能测试用 Token
func generateTestTokenForBenchmark(secret string) string {
	claims := jwt.MapClaims{
		"user_id":  123,
		"username": "testuser",
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
		"iat":      time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(secret))
	return tokenString
}
