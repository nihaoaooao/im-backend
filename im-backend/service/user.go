package service

import (
	"context"
	"errors"
	"im-backend/models"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrUserNotFound     = errors.New("用户不存在")
	ErrInvalidPassword = errors.New("密码错误")
	ErrUserDisabled    = errors.New("用户已被禁用")
)

// UserService 用户服务
type UserService struct {
	DB *gorm.DB
}

// NewUserService 创建用户服务
func NewUserService(db *gorm.DB) *UserService {
	return &UserService{
		DB: db,
	}
}

// GetUserByID 根据ID获取用户
func (s *UserService) GetUserByID(ctx context.Context, userID int64) (*models.User, error) {
	var user models.User
	if err := s.DB.WithContext(ctx).First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

// GetUserByUsername 根据用户名获取用户
func (s *UserService) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	var user models.User
	if err := s.DB.WithContext(ctx).Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

// VerifyPassword 验证密码
func (s *UserService) VerifyPassword(ctx context.Context, username, password string) (*models.User, error) {
	user, err := s.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, err
	}

	// 检查用户状态
	if user.Status != "active" {
		return nil, ErrUserDisabled
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidPassword
	}

	return user, nil
}

// CreateUser 创建用户
func (s *UserService) CreateUser(ctx context.Context, user *models.User) error {
	// 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.PasswordHash), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.PasswordHash = string(hashedPassword)

	return s.DB.WithContext(ctx).Create(user).Error
}

// UpdateLastLogin 更新最后登录时间
func (s *UserService) UpdateLastLogin(ctx context.Context, userID int64) error {
	now := time.Now()
	return s.DB.Model(&models.User{}).Where("id = ?", userID).Update("last_login_at", now).Error
}
