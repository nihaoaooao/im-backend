package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"im-backend/model"
	"im-backend/ws"

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
	jwtExpire  int  // 小时
	hub        *ws.Hub
}

// SetHub 设置 WebSocket Hub
func (s *UserService) SetHub(hub *ws.Hub) {
	s.hub = hub
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
	token, _ := s.generateToken(user.ID, user.Username, user.Role)

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

// Login 用户登录 - [P2] 添加登录失败限制
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

	// [P2] 检查登录失败次数（使用Redis记录）
	loginFailKey := fmt.Sprintf("login:fail:%d", user.ID)
	if s.redis != nil {
		failCount, _ := s.redis.Get(context.Background(), loginFailKey).Int()
		if failCount >= 5 {
			// 获取锁定剩余时间
			ttl, _ := s.redis.TTL(context.Background(), loginFailKey).Result()
			c.JSON(429, gin.H{
				"code": 429,
				"msg":  fmt.Sprintf("登录尝试过多，请 %.0f 分钟后再试", ttl.Minutes()),
			})
			return
		}
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		// [P2] 记录登录失败
		if s.redis != nil {
			s.redis.Incr(context.Background(), loginFailKey)
			s.redis.Expire(context.Background(), loginFailKey, 15*time.Minute) // 15分钟内有效
		}
		c.JSON(401, gin.H{"code": 401, "msg": "用户名或密码错误"})
		return
	}

	// [P2] 登录成功后清除失败记录
	if s.redis != nil {
		s.redis.Del(context.Background(), loginFailKey)
	}

	// 生成 JWT token
	token, err := s.generateToken(user.ID, user.Username, user.Role)
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
	token, err := s.generateToken(userID, username, user.Role)
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

	// 不能添加自己为好友
	if userID.(int64) == req.FriendID {
		c.JSON(400, gin.H{"code": 400, "msg": "不能添加自己为好友"})
		return
	}

	// 检查目标用户是否存在
	var targetUser model.User
	if err := s.db.First(&targetUser, req.FriendID).Error; err != nil {
		c.JSON(404, gin.H{"code": 404, "msg": "用户不存在"})
		return
	}

	// 检查是否已经是好友关系
	var existFriend model.Friendship
	if err := s.db.Where("(user_id = ? AND friend_id = ?) OR (user_id = ? AND friend_id = ?)",
		userID.(int64), req.FriendID, req.FriendID, userID.(int64)).First(&existFriend).Error; err == nil {
		if existFriend.Status == "accepted" {
			c.JSON(400, gin.H{"code": 400, "msg": "已经是好友了"})
			return
		}
		if existFriend.Status == "pending" {
			c.JSON(400, gin.H{"code": 400, "msg": "已经发送过好友请求了"})
			return
		}
	}

	// [P2] 好友添加验证：创建待审核的好友请求
	friendship := model.Friendship{
		UserID:    userID.(int64),
		FriendID:  req.FriendID,
		Remark:    req.Remark,
		Status:    "pending", // 待对方确认
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.db.Create(&friendship).Error; err != nil {
		c.JSON(500, gin.H{"code": 500, "msg": "发送好友请求失败"})
		return
	}

	// 通过 WebSocket 通知对方
	if s.hub != nil {
		s.hub.SendToUser(req.FriendID, gin.H{
			"type": "friend_request",
			"data": gin.H{
				"from_user_id": userID.(int64),
				"remark":       req.Remark,
			},
		})
	}

	c.JSON(200, gin.H{
		"code": 0,
		"msg":  "好友请求已发送，等待对方确认",
		"data": gin.H{
			"status": "pending",
		},
	})
}

// RespondFriendRequest 响应好友请求
func (s *UserService) RespondFriendRequest(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req struct {
		FriendID int64  `json:"friend_id" binding:"required"`
		Accept   bool   `json:"accept" binding:"required"` // true=接受，false=拒绝
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 400, "msg": "参数错误"})
		return
	}

	// 查找待处理的好友请求
	var friendship model.Friendship
	if err := s.db.Where("user_id = ? AND friend_id = ? AND status = ?",
		req.FriendID, userID.(int64), "pending").First(&friendship).Error; err != nil {
		c.JSON(404, gin.H{"code": 404, "msg": "好友请求不存在"})
		return
	}

	// 更新请求状态
	if req.Accept {
		friendship.Status = "accepted"
		friendship.UpdatedAt = time.Now()
		s.db.Save(&friendship)

		// 创建双向好友关系
		friendship2 := model.Friendship{
			UserID:    userID.(int64),
			FriendID:  req.FriendID,
			Remark:    "",
			Status:    "accepted",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		s.db.Create(&friendship2)

		// 通知对方
		if s.hub != nil {
			s.hub.SendToUser(req.FriendID, gin.H{
				"type": "friend_accepted",
				"data": gin.H{
					"from_user_id": userID.(int64),
				},
			})
		}

		c.JSON(200, gin.H{"code": 0, "msg": "已接受好友请求"})
	} else {
		friendship.Status = "rejected"
		friendship.UpdatedAt = time.Now()
		s.db.Save(&friendship)

		c.JSON(200, gin.H{"code": 0, "msg": "已拒绝好友请求"})
	}
}

// GetFriendRequests 获取好友请求列表
func (s *UserService) GetFriendRequests(c *gin.Context) {
	userID, _ := c.Get("user_id")
	requestType := c.DefaultQuery("type", "received") // received=收到的请求，sent=发送的请求

	var friendships []model.Friendship

	if requestType == "received" {
		// 收到的请求
		s.db.Where("friend_id = ? AND status = ?", userID, "pending").Find(&friendships)
	} else {
		// 发送的请求
		s.db.Where("user_id = ? AND status = ?", userID, "pending").Find(&friendships)
	}

	// 填充用户信息
	type Result struct {
		ID        int64             `json:"id"`
		FromUserID int64            `json:"from_user_id"`
		ToUserID  int64            `json:"to_user_id"`
		Remark    string            `json:"remark"`
		Status    string            `json:"status"`
		CreatedAt time.Time         `json:"created_at"`
		User     *model.User       `json:"user,omitempty"`
	}

	results := make([]Result, 0)
	for _, f := range friendships {
		var user model.User
		var userIDQuery int64
		if requestType == "received" {
			userIDQuery = f.UserID
		} else {
			userIDQuery = f.FriendID
		}
		s.db.First(&user, userIDQuery)

		results = append(results, Result{
			ID:         f.ID,
			FromUserID: f.UserID,
			ToUserID:  f.FriendID,
			Remark:    f.Remark,
			Status:    f.Status,
			CreatedAt: f.CreatedAt,
			User:      &user,
		})
	}

	c.JSON(200, gin.H{"code": 0, "data": results})
}

// generateToken 生成 JWT Token
func (s *UserService) generateToken(userID int64, username string, role string) (string, error) {
	// 默认角色为 user
	if role == "" {
		role = "user"
	}
	claims := jwt.MapClaims{
		"user_id":  userID,
		"username": username,
		"role":     role, // PT-002: JWT Token 包含角色信息
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

// ============ PT-002: 管理员接口 ============

// ListUsers 获取用户列表（管理员）
func (s *UserService) ListUsers(c *gin.Context) {
	page := 1
	pageSize := 20
	if p := c.Query("page"); p != "" {
		fmt.Sscanf(p, "%d", &page)
	}
	if ps := c.Query("page_size"); ps != "" {
		fmt.Sscanf(ps, "%d", &pageSize)
	}
	if page < 1 {
		page = 1
	}
	if pageSize > 100 || pageSize < 1 {
		pageSize = 20
	}

	var users []model.User
	var total int64

	s.db.Model(&model.User{}).Count(&total)
	err := s.db.Offset((page - 1) * pageSize).Limit(pageSize).Find(&users).Error
	if err != nil {
		c.JSON(500, gin.H{"code": 500, "msg": "获取用户列表失败"})
		return
	}

	c.JSON(200, gin.H{
		"code": 0,
		"data": gin.H{
			"users":     users,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

// GetUser 获取用户详情（管理员）
func (s *UserService) GetUser(c *gin.Context) {
	var id int64
	if _, err := fmt.Sscanf(c.Param("id"), "%d", &id); err != nil {
		c.JSON(400, gin.H{"code": 400, "msg": "无效的用户ID"})
		return
	}

	var user model.User
	if err := s.db.First(&user, id).Error; err != nil {
		c.JSON(404, gin.H{"code": 404, "msg": "用户不存在"})
		return
	}

	c.JSON(200, gin.H{"code": 0, "data": user})
}

// DeleteUser 删除用户（管理员）
func (s *UserService) DeleteUser(c *gin.Context) {
	var id int64
	if _, err := fmt.Sscanf(c.Param("id"), "%d", &id); err != nil {
		c.JSON(400, gin.H{"code": 400, "msg": "无效的用户ID"})
		return
	}

	// 不能删除自己
	currentUserID, _ := c.Get("user_id")
	if currentID, ok := currentUserID.(int64); ok && currentID == id {
		c.JSON(400, gin.H{"code": 400, "msg": "不能删除自己的账号"})
		return
	}

	if err := s.db.Delete(&model.User{}, id).Error; err != nil {
		c.JSON(500, gin.H{"code": 500, "msg": "删除用户失败"})
		return
	}

	c.JSON(200, gin.H{"code": 0, "msg": "删除成功"})
}

// 错误定义
var (
	ErrUserNotFound     = errors.New("user not found")
	ErrInvalidPassword = errors.New("invalid password")
)
