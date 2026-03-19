package api

import (
	"net/http"
	"strconv"

	"im-backend/middleware"
	"im-backend/models"
	"im-backend/service"

	"github.com/gin-gonic/gin"
)

// AuthHandler 认证处理器
type AuthHandler struct {
	userService *service.UserService
}

// NewAuthHandler 创建认证处理器
func NewAuthHandler(userService *service.UserService) *AuthHandler {
	return &AuthHandler{
		userService: userService,
	}
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	AccessToken string      `json:"access_token"`
	TokenType   string      `json:"token_type"`
	ExpiresIn   int64       `json:"expires_in"`
	User        UserInfo    `json:"user"`
}

// UserInfo 用户信息
type UserInfo struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Nickname string `json:"nickname"`
	Role     string `json:"role"`
}

// Login 登录
// @Summary 用户登录
// @Tags 认证
// @Accept json
// @Produce json
// @Param request body LoginRequest true "登录请求"
// @Success 200 {object} LoginResponse
// @Router /api/v1/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请求参数错误",
		})
		return
	}

	// 验证用户
	user, err := h.userService.VerifyPassword(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		switch err {
		case service.ErrUserNotFound:
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    40101,
				"message": "用户不存在",
			})
		case service.ErrInvalidPassword:
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    40102,
				"message": "密码错误",
			})
		case service.ErrUserDisabled:
			c.JSON(http.StatusForbidden, gin.H{
				"code":    40301,
				"message": "用户已被禁用",
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    50001,
				"message": "服务器错误",
			})
		}
		return
	}

	// 为每个用户生成唯一的 JWT Token
	token, err := middleware.GenerateToken(int64(user.ID), user.Username, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    50002,
			"message": "生成Token失败",
		})
		return
	}

	// 更新最后登录时间
	h.userService.UpdateLastLogin(c.Request.Context(), int64(user.ID))

	// 返回响应
	c.JSON(http.StatusOK, LoginResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   86400, // 24小时
		User: UserInfo{
			ID:       int64(user.ID),
			Username: user.Username,
			Nickname: user.Nickname,
			Role:     user.Role,
		},
	})
}

// RegisterRequest 注册请求
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Password string `json:"password" binding:"required,min=6,max=100"`
	Nickname string `json:"nickname"`
	Email    string `json:"email" binding:"required,email"`
}

// Register 注册
// @Summary 用户注册
// @Tags 认证
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "注册请求"
// @Success 201 {object} UserInfo
// @Router /api/v1/auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	// 检查用户名是否已存在
	_, err := h.userService.GetUserByUsername(c.Request.Context(), req.Username)
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{
			"code":    40901,
			"message": "用户名已存在",
		})
		return
	}

	// 创建用户
	user := &models.User{
		Username: req.Username,
		PasswordHash: req.Password,
		Nickname: req.Nickname,
		Email:    req.Email,
		Role:    "user",
		Status:  "active",
	}

	if err := h.userService.CreateUser(c.Request.Context(), user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    50001,
			"message": "创建用户失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, UserInfo{
		ID:       int64(user.ID),
		Username: user.Username,
		Nickname: user.Nickname,
		Role:     user.Role,
	})
}

// Logout 登出
// @Summary 用户登出
// @Tags 认证
// @Security BearerAuth
// @Success 200 {object} gin.H
// @Router /api/v1/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	// JWT 无状态，登出只需要通知客户端删除 Token
	// 如果需要实现 Token 注销，可以将 Token 加入黑名单
	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "登出成功",
	})
}

// RefreshToken 刷新 Token
// @Summary 刷新 Token
// @Tags 认证
// @Security BearerAuth
// @Success 200 {object} gin.H
// @Router /api/v1/auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	// 从上下文获取当前用户信息
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未授权",
		})
		return
	}

	username, _ := c.Get("username")
	role, _ := c.Get("role")

	// 生成新 Token
	token, err := middleware.GenerateToken(userID.(int64), username.(string), role.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    50001,
			"message": "刷新Token失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":        200,
		"access_token": token,
		"token_type":   "Bearer",
		"expires_in":   86400,
	})
}

// GetCurrentUser 获取当前用户信息
// @Summary 获取当前用户信息
// @Tags 认证
// @Security BearerAuth
// @Success 200 {object} UserInfo
// @Router /api/v1/auth/me [get]
func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未授权",
		})
		return
	}

	user, err := h.userService.GetUserByID(c.Request.Context(), userID.(int64))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    40401,
			"message": "用户不存在",
		})
		return
	}

	c.JSON(http.StatusOK, UserInfo{
		ID:       int64(user.ID),
		Username: user.Username,
		Nickname: user.Nickname,
		Role:     user.Role,
	})
}

// GetUserIDFromContext 从上下文获取用户ID
func GetUserIDFromContext(c *gin.Context) int64 {
	userID, exists := c.Get("user_id")
	if !exists {
		return 0
	}
	switch v := userID.(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case uint:
		return int64(v)
	case string:
		id, _ := strconv.ParseInt(v, 10, 64)
		return id
	default:
		return 0
	}
}
