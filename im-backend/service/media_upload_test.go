package service

import (
	"context"
	"mime/multipart"
	"testing"

	"im-backend/models"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func setupMediaTestDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}

	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		DSN:                  "sqlmock_db",
		PreferSimpleProtocol: true,
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open gorm: %v", err)
	}

	return gormDB, mock
}

// TestValidateFileType 测试文件类型验证
func TestValidateFileType(t *testing.T) {
	db, _ := setupMediaTestDB(t)

	// 注意：这里需要真实的 COS client 测试
	// 由于 COS client 初始化需要配置，这里只测试服务创建
	service := &MediaUploadService{
		db: db,
	}

	// 验证结构
	assert.NotNil(t, service)
}

// TestMediaModel 测试 Media 模型
func TestMediaModel(t *testing.T) {
	// 测试模型定义
	media := models.Media{
		ID:           1,
		UserID:       100,
		Type:         models.MediaTypeImage,
		OriginalURL:  "https://example.com/media/123.jpg",
		ThumbnailURL: "https://example.com/media/123_thumb.jpg",
		FileSize:     1024000,
		Width:        1920,
		Height:       1080,
		Format:       "jpg",
		StorageKey:   "media/image/123.jpg",
	}

	assert.Equal(t, "media", media.TableName())
	assert.Equal(t, models.MediaTypeImage, media.Type)
	assert.Equal(t, int64(1024000), media.FileSize)
}

// TestMediaType 测试媒体类型常量
func TestMediaType(t *testing.T) {
	assert.Equal(t, models.MediaType("image"), models.MediaTypeImage)
	assert.Equal(t, models.MediaType("voice"), models.MediaTypeVoice)
	assert.Equal(t, models.MediaType("video"), models.MediaTypeVideo)
}

// TestUserMediaQuota 测试用户配额模型
func TestUserMediaQuota(t *testing.T) {
	quota := models.UserMediaQuota{
		ID:         1,
		UserID:     100,
		TotalSize:  10240000,
		FileCount:  10,
	}

	assert.Equal(t, "user_media_quota", quota.TableName())
	assert.Equal(t, int64(100), quota.UserID)
	assert.Equal(t, int64(10), quota.FileCount)
}

// TestUploadRequest 测试上传请求结构
func TestUploadRequest(t *testing.T) {
	req := &UploadRequest{
		Type:   models.MediaTypeImage,
		UserID: 100,
	}

	// 注意：File 是 multipart.FileHeader 类型，需要单独测试
	assert.Equal(t, models.MediaTypeImage, req.Type)
	assert.Equal(t, int64(100), req.UserID)
}

// TestUploadResponse 测试上传响应结构
func TestUploadResponse(t *testing.T) {
	resp := &UploadResponse{
		URL:          "https://example.com/media/123.jpg",
		ThumbnailURL: "https://example.com/media/123_thumb.jpg",
		Type:         "image",
		Size:         1024000,
		Width:        1920,
		Height:       1080,
		Format:       "jpg",
	}

	assert.Equal(t, "https://example.com/media/123.jpg", resp.URL)
	assert.Equal(t, "https://example.com/media/123_thumb.jpg", resp.ThumbnailURL)
	assert.Equal(t, int64(1024000), resp.Size)
	assert.Equal(t, 1920, resp.Width)
	assert.Equal(t, 1080, resp.Height)
}

// TestContextCancellation 测试上下文取消
func TestContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// 验证上下文已取消
	select {
	case <-ctx.Done():
		assert.Equal(t, context.Canceled, ctx.Err())
	default:
		t.Fatal("context should be cancelled")
	}
}
