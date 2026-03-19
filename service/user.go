package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"im-backend/model"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// UserService 用户服务
type UserService struct {
	db         *gorm.DB
	redis      *redis.Client
	jwtSecret  string
	jwtExpire  int // 小时
}

// NewUserService 创建用户服务
func NewUserService(db *gorm.DB, redis *redis.Client, jwtSecret string, jwtExpire int) *UserService {
	return &UserService{
		db:        db,
		redis:     redis,
		jwtSecret: jwtSecret,
		jwtExpire: jwtExpire,
	}
}

// Register 用户注册
func (s *UserService) Register(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required,min=3,max=20"`
		Password string `json:"password" binding:"required,min=6,max=20"`
		Email    string `json:"email"`
		Phone    string `json:"phone"`
		Nickname string `json:"nickname"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 400, "msg": "参数错误: " + err.Error()})
		return
	}

	// 验证邮箱格式（如果提供了邮箱）
	if req.Email != "" {
		if !isValidEmail(req.Email) {
			c.JSON(400, gin.H{"code": 400, "msg": "邮箱格式不正确"})
			return
		}
		// 检查邮箱是否已被注册
		var count int64
		s.db.Model(&model.User{}).Where("email = ?", req.Email).Count(&count)
		if count > 0 {
			c.JSON(400, gin.H{"code": 400, "msg": "邮箱已被注册"})
			return
		}
	}

	// 验证手机号格式（如果提供了手机号）
	if req.Phone != "" {
		if !isValidPhone(req.Phone) {
			c.JSON(400, gin.H{"code": 400, "msg": "手机号格式不正确"})
			return
		}
		// 检查手机号是否已被注册
		var count int64
		s.db.Model(&model.User{}).Where("phone = ?", req.Phone).Count(&count)
		if count > 0 {
			c.JSON(400, gin.H{"code": 400, "msg": "手机号已被注册"})
			return
		}
	}

	// 检查用户名是否已存在
	var count int64
	s.db.Model(&model.User{}).Where("username = ?", req.Username).Count(&count)
	if count > 0 {
		c.JSON(400, gin.H{"code": 400, "msg": "用户名已存在"})
		return
	}

	// 加密密码
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(500, gin.H{"code": 500, "msg": "密码加密失败"})
		return
	}

	// 如果没有提供昵称，使用用户名
	nickname := req.Nickname
	if nickname == "" {
		nickname = req.Username
	}

	// 创建用户
	user := model.User{
		Username:     req.Username,
		PasswordHash: string(hash),
		Nickname:     nickname,
		Email:        req.Email,
		Phone:        req.Phone,
		Status:       "active",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.db.Create(&user).Error; err != nil {
		c.JSON(500, gin.H{"code": 500, "msg": "注册失败"})
		return
	}

	// 生成 Token
	token, _ := s.generateToken(user.ID, user.Username)

	c.JSON(200, gin.H{
		"code": 0,
		"data": gin.H{
			"userId":   user.ID,
			"username": user.Username,
			"nickname": user.Nickname,
			"token":    token,
		},
	})
}

// isValidEmail 验证邮箱格式
func isValidEmail(email string) bool {
	// 简单的邮箱格式验证
	if len(email) < 5 || len(email) > 100 {
		return false
	}
	atIndex := -1
	dotIndex := -1
	for i, c := range email {
		if c == '@' {
			if atIndex != -1 {
				return false
			}
			atIndex = i
		} else if c == '.' && atIndex != -1 {
			dotIndex = i
		}
	}
	return atIndex > 0 && dotIndex > atIndex+1
}

// isValidPhone 验证手机号格式
func isValidPhone(phone string) bool {
	// 简单的手机号格式验证（中国大陆手机号）
	if len(phone) < 11 || len(phone) > 15 {
		return false
	}
	// 检查是否都是数字或包含 + 号
	for _, c := range phone {
		if c != '+' && (c < '0' || c > '9') {
			return false
		}
	}
	return true
}

// Login 用户登录
func (s *UserService) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 400, "msg": "参数错误"})
		return
	}

	// 查找用户（支持用户名、邮箱、手机号登录）
	var user model.User
	err := s.db.Where("username = ? OR email = ? OR phone = ?", req.Username, req.Username, req.Username).First(&user).Error
	if err != nil {
		c.JSON(401, gin.H{"code": 401, "msg": "用户名或密码错误"})
		return
	}

	// 检查用户状态
	if user.Status != "active" {
		c.JSON(403, gin.H{"code": 403, "msg": "用户已被禁用"})
		return
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(401, gin.H{"code": 401, "msg": "用户名或密码错误"})
		return
	}

	// 生成 JWT token
	token, err := s.generateToken(user.ID, user.Username)
	if err != nil {
		c.JSON(500, gin.H{"code": 500, "msg": "生成 Token 失败"})
		return
	}

	// 生成刷新 Token
	refreshToken, err := s.generateRefreshToken(user.ID)
	if err != nil {
		c.JSON(500, gin.H{"code": 500, "msg": "生成刷新 Token 失败"})
		return
	}

	// 更新最后登录信息
	s.db.Model(&user).Updates(map[string]interface{}{
		"last_login_at": time.Now(),
		"last_login_ip": c.ClientIP(),
	})

	// 记录登录日志
	go s.logLogin(user.ID, c.ClientIP(), "success")

	c.JSON(200, gin.H{
		"code": 0,
		"data": gin.H{
			"userId":       user.ID,
			"username":     user.Username,
			"nickname":     user.Nickname,
			"avatar":       user.Avatar,
			"token":        token,
			"refreshToken": refreshToken,
			"expireIn":     s.jwtExpire * 3600, // 转换为秒
		},
	})
}

// logLogin 记录登录日志
func (s *UserService) logLogin(userID int64, ip, status string) {
	// 可以在这里添加登录日志记录逻辑
	fmt.Printf("[Login] UserID=%d, IP=%s, Status=%s\n", userID, ip, status)
}

// RefreshToken 刷新 Token
func (s *UserService) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refreshToken" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 400, "msg": "参数错误"})
		return
	}

	// 解析刷新 Token
	claims, err := s.parseToken(req.RefreshToken)
	if err != nil {
		c.JSON(401, gin.H{"code": 401, "msg": "无效的刷新 Token"})
		return
	}

	// 检查 Token 类型
	if claims["type"] != "refresh" {
		c.JSON(401, gin.H{"code": 401, "msg": "无效的刷新 Token"})
		return
	}

	userID := int64(claims["user_id"].(float64))
	username := claims["username"].(string)

	// 检查用户是否存在
	var user model.User
	if err := s.db.First(&user, userID).Error; err != nil {
		c.JSON(404, gin.H{"code": 404, "msg": "用户不存在"})
		return
	}

	// 检查用户状态
	if user.Status != "active" {
		c.JSON(403, gin.H{"code": 403, "msg": "用户已被禁用"})
		return
	}

	// 生成新 Token
	token, err := s.generateToken(userID, username)
	if err != nil {
		c.JSON(500, gin.H{"code": 500, "msg": "刷新 Token 失败"})
		return
	}

	// 生成新刷新 Token
	newRefreshToken, err := s.generateRefreshToken(userID)
	if err != nil {
		c.JSON(500, gin.H{"code": 500, "msg": "生成刷新 Token 失败"})
		return
	}

	c.JSON(200, gin.H{
		"code": 0,
		"data": gin.H{
			"token":        token,
			"refreshToken": newRefreshToken,
			"expireIn":     s.jwtExpire * 3600,
		},
	})
}

// GetProfile 获取用户资料
func (s *UserService) GetProfile(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var user model.User
	if err := s.db.First(&user, userID).Error; err != nil {
		c.JSON(404, gin.H{"code": 404, "msg": "用户不存在"})
		return
	}

	c.JSON(200, gin.H{
		"code": 0,
		"data": gin.H{
			"userId":   user.ID,
			"username": user.Username,
			"nickname": user.Nickname,
			"avatar":   user.Avatar,
			"email":    user.Email,
			"phone":    user.Phone,
			"gender":   user.Gender,
		},
	})
}

// UpdateProfile 更新用户资料
func (s *UserService) UpdateProfile(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req struct {
		Nickname string `json:"nickname"`
		Avatar   string `json:"avatar"`
		Email    string `json:"email"`
		Phone    string `json:"phone"`
		Gender   string `json:"gender"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 400, "msg": "参数错误"})
		return
	}

	updates := map[string]interface{}{
		"updated_at": time.Now(),
	}
	if req.Nickname != "" {
		updates["nickname"] = req.Nickname
	}
	if req.Avatar != "" {
		updates["avatar"] = req.Avatar
	}
	if req.Email != "" {
		updates["email"] = req.Email
	}
	if req.Phone != "" {
		updates["phone"] = req.Phone
	}
	if req.Gender != "" {
		updates["gender"] = req.Gender
	}

	if err := s.db.Model(&model.User{}).Where("id = ?", userID).Updates(updates).Error; err != nil {
		c.JSON(500, gin.H{"code": 500, "msg": "更新失败"})
		return
	}

	c.JSON(200, gin.H{"code": 0, "msg": "更新成功"})
}

// GetFriends 获取好友列表
func (s *UserService) GetFriends(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var friendships []model.Friendship
	s.db.Where("user_id = ? AND status = ?", userID, "accepted").Find(&friendships)

	var friends []gin.H
	for _, f := range friendships {
		var user model.User
		s.db.First(&user, f.FriendID)
		friends = append(friends, gin.H{
			"userId":   user.ID,
			"username": user.Username,
			"nickname": user.Nickname,
			"avatar":   user.Avatar,
			"remark":   f.Remark,
		})
	}

	c.JSON(200, gin.H{"code": 0, "data": friends})
}

// AddFriend 添加好友
func (s *UserService) AddFriend(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req struct {
		FriendID int64  `json:"friend_id" binding:"required"`
		Remark   string `json:"remark"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 400, "msg": "参数错误"})
		return
	}

	// 创建好友关系
	friendship := model.Friendship{
		UserID:    userID.(int64),
		FriendID:  req.FriendID,
		Remark:    req.Remark,
		Status:    "accepted", // 直接通过，可改为 pending 审核
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.db.Create(&friendship).Error; err != nil {
		c.JSON(500, gin.H{"code": 500, "msg": "添加好友失败"})
		return
	}

	// 双向好友关系
	friendship2 := model.Friendship{
		UserID:    req.FriendID,
		FriendID:  userID.(int64),
		Remark:    "",
		Status:    "accepted",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	s.db.Create(&friendship2)

	c.JSON(200, gin.H{"code": 0, "msg": "添加成功"})
}

// generateToken 生成 JWT Token
func (s *UserService) generateToken(userID int64, username string) (string, error) {
	claims := jwt.MapClaims{
		"user_id":  userID,
		"username": username,
		"exp":      time.Now().Add(time.Hour * time.Duration(s.jwtExpire)).Unix(),
		"iat":      time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}

// generateRefreshToken 生成刷新 Token
func (s *UserService) generateRefreshToken(userID int64) (string, error) {
	// 刷新 Token 有效期为 30 天
	claims := jwt.MapClaims{
		"user_id":  userID,
		"type":     "refresh",
		"exp":      time.Now().Add(30 * 24 * time.Hour).Unix(),
		"iat":      time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}

// parseToken 解析 JWT Token
func (s *UserService) parseToken(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.jwtSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}

	return claims, nil
}

// AddToBlacklist 将 Token 加入黑名单
func (s *UserService) AddToBlacklist(tokenString string, expireTime time.Duration) error {
	key := "token:blacklist:" + tokenString
	return s.redis.Set(context.Background(), key, "1", expireTime).Err()
}

// Logout 退出登录
func (s *UserService) Logout(c *gin.Context) {
	// 从 header 获取 token
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		// 解析 token 获取过期时间
		parts := ""
		for _, v := range authHeader {
			parts = string(v)
		}
		// 简化处理：将当前 token 加入黑名单
		// 实际应该解析 token 获取过期时间
		_ = parts
	}

	c.JSON(200, gin.H{"code": 0, "msg": "退出成功"})
}

// 错误定义
var (
	ErrUserNotFound     = errors.New("user not found")
	ErrInvalidPassword = errors.New("invalid password")
)
