package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"im-backend/config"
	"im-backend/models"

	"github.com/redis/go-redis/v9"
	"github.com/tencentyun/cos-go-sdk-v5"
	"gorm.io/gorm"
)

var (
	ErrInvalidFileType     = errors.New("不支持的文件类型")
	ErrFileTooLarge        = errors.New("文件大小超过限制")
	ErrUploadFailed         = errors.New("文件上传失败")
	ErrMediaNotFound        = errors.New("媒体文件不存在")
	ErrNotMediaOwner        = errors.New("无权操作此文件")
	ErrInvalidMediaType     = errors.New("无效的媒体类型")
	ErrInvalidFilename      = errors.New("无效的文件名")
)

// P2-1: 允许的文件扩展名白名单
var allowedExtensions = map[models.MediaType][]string{
	models.MediaTypeImage: {".jpg", ".jpeg", ".png", ".gif", ".webp"},
	models.MediaTypeVoice: {".mp3", ".m4a", ".wav", ".webm", ".aac"},
	models.MediaTypeVideo: {".mp4", ".mov", ".avi", ".webm", ".mkv"},
}

// sanitizeFilename 清理文件名，防止路径遍历攻击 (P2-1)
func sanitizeFilename(filename string) (string, error) {
	if filename == "" {
		return "", ErrInvalidFilename
	}

	// 1. 移除路径，只保留文件名
	filename = filepath.Base(filename)

	// 2. 检查路径遍历字符
	if strings.Contains(filename, "..") {
		return "", ErrInvalidFilename
	}

	// 3. 移除特殊字符（只允许字母、数字、下划线、连字符、点）
	reg := regexp.MustCompile(`[^a-zA-Z0-9_\-\.]`)
	filename = reg.ReplaceAllString(filename, "")

	// 4. 移除前后空格和控制字符
	filename = strings.TrimSpace(filename)
	filename = strings.Trim(filename, "\x00-\x1f")

	// 5. 确保文件名不为空
	if filename == "" || filename == "." || filename == ".." {
		return "", ErrInvalidFilename
	}

	// 6. 检查扩展名是否在白名单中
	ext := strings.ToLower(filepath.Ext(filename))
	isAllowed := false
	for _, allowedExt := range allowedExtensions {
		for _, e := range allowedExt {
			if ext == e {
				isAllowed = true
				break
			}
		}
		if isAllowed {
			break
		}
	}

	if !isAllowed {
		return "", ErrInvalidFilename
	}

	return filename, nil
}

// MediaUploadService 媒体上传服务
type MediaUploadService struct {
	db          *gorm.DB
	cosClient   *cos.Client
	redisClient *redis.Client
}

// NewMediaUploadService 创建媒体上传服务
func NewMediaUploadService(db *gorm.DB, redisClient *redis.Client) (*MediaUploadService, error) {
	// 初始化 COS 客户端 - 使用存根实现，生产环境需要配置真实的 COS
	var client *cos.Client
	conf := config.COSConfig
	if conf.SecretID != "" && conf.SecretKey != "" {
		// 使用 cos.NewClient 方式创建客户端
		client = &cos.Client{}
	}

	return &MediaUploadService{
		db:          db,
		cosClient:   client,
		redisClient: redisClient,
	}, nil
}

// UploadRequest 上传请求
type UploadRequest struct {
	File   *multipart.FileHeader
	Type   models.MediaType
	UserID int64
}

// UploadResponse 上传响应
type UploadResponse struct {
	URL          string  `json:"url"`
	ThumbnailURL string  `json:"thumbnailUrl,omitempty"`
	Type         string  `json:"type"`
	Size         int64   `json:"size"`
	Width        int     `json:"width,omitempty"`
	Height       int     `json:"height,omitempty"`
	Duration     float64 `json:"duration,omitempty"`
	Format       string  `json:"format,omitempty"`
}

// Upload 上传文件
func (s *MediaUploadService) Upload(ctx context.Context, req *UploadRequest) (*UploadResponse, error) {
	// 1. 验证文件类型
	if err := s.validateFileType(req.File, req.Type); err != nil {
		return nil, err
	}

	// 2. 验证文件大小
	if err := s.validateFileSize(req.File, req.Type); err != nil {
		return nil, err
	}

	// 3. 打开文件
	file, err := req.File.Open()
	if err != nil {
		return nil, fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	// 4. 清理文件名并生成唯一文件名 (P2-1)
	sanitizedFilename, err := sanitizeFilename(req.File.Filename)
	if err != nil {
		return nil, err
	}
	ext := strings.ToLower(filepath.Ext(sanitizedFilename))
	uuid := generateUUID()
	key := fmt.Sprintf("media/%s/%s%s", req.Type, uuid, ext)

	// 5. 根据类型处理文件（传递 req.File 以便获取文件信息）
	var resp *UploadResponse
	switch req.Type {
	case models.MediaTypeImage:
		resp, err = s.processImage(ctx, req.File, key, ext)
	case models.MediaTypeVoice:
		resp, err = s.processVoice(ctx, req.File, key, ext)
	case models.MediaTypeVideo:
		resp, err = s.processVideo(ctx, req.File, key, ext)
	default:
		return nil, ErrInvalidMediaType
	}

	if err != nil {
		return nil, err
	}

	// 6. 上传到 COS
	if err := s.uploadToCOS(ctx, file, key); err != nil {
		return nil, err
	}

	// 7. 生成访问 URL
	cdnDomain := config.COSConfig.CDNDomain
	if cdnDomain == "" {
		cdnDomain = fmt.Sprintf("https://%s.cos.%s.myqcloud.com", config.COSConfig.Bucket, config.COSConfig.Region)
	}
	resp.URL = fmt.Sprintf("%s/%s", cdnDomain, key)

	// 8. 保存到数据库
	media := &models.Media{
		UserID:       req.UserID,
		Type:         req.Type,
		OriginalURL:  resp.URL,
		ThumbnailURL: resp.ThumbnailURL,
		FileSize:     resp.Size,
		Width:        resp.Width,
		Height:       resp.Height,
		Duration:     resp.Duration,
		Format:       resp.Format,
		StorageKey:   key,
	}

	if err := s.db.Create(media).Error; err != nil {
		return nil, fmt.Errorf("保存媒体记录失败: %w", err)
	}

	// 9. 更新用户配额
	s.updateUserQuota(req.UserID, resp.Size)

	return resp, nil
}

// validateFileType 验证文件类型 (P0-3: 服务端 MIME 检测 + 扩展名白名单 + 内容验证)
func (s *MediaUploadService) validateFileType(file *multipart.FileHeader, mediaType models.MediaType) error {
	// 1. 检查扩展名是否在白名单中 (P2-1)
	ext := strings.ToLower(filepath.Ext(file.Filename))
	allowedExts, ok := allowedExtensions[mediaType]
	if !ok {
		return ErrInvalidMediaType
	}

	isExtAllowed := false
	for _, allowedExt := range allowedExts {
		if ext == allowedExt {
			isExtAllowed = true
			break
		}
	}
	if !isExtAllowed {
		return ErrInvalidFileType
	}

	// 2. 读取文件头进行服务端 MIME 类型检测 (P0-3)
	f, err := file.Open()
	if err != nil {
		return ErrInvalidFileType
	}
	defer f.Close()

	// 读取前 512 字节用于检测
	header := make([]byte, 512)
	n, err := f.Read(header)
	if err != nil && err != io.EOF {
		return ErrInvalidFileType
	}
	header = header[:n]

	// 检测 MIME 类型 (服务端检测，不使用客户端提供的值)
	mimeType := http.DetectContentType(header)

	// 3. 验证 MIME 类型
	var allowedMIMEs []string
	switch mediaType {
	case models.MediaTypeImage:
		allowedMIMEs = []string{"image/jpeg", "image/png", "image/gif", "image/webp"}
	case models.MediaTypeVoice:
		allowedMIMEs = []string{"audio/mpeg", "audio/mp4", "audio/x-m4a", "audio/wav", "audio/webm"}
	case models.MediaTypeVideo:
		allowedMIMEs = []string{"video/mp4", "video/quicktime", "video/x-msvideo", "video/webm"}
	}

	isMIMEAllowed := false
	for _, allowedMIME := range allowedMIMEs {
		if mimeType == allowedMIME {
			isMIMEAllowed = true
			break
		}
	}
	if !isMIMEAllowed {
		return ErrInvalidFileType
	}

	// 4. 对于图片，验证是否为真实图片 (P0-3: 图片内容验证)
	if mediaType == models.MediaTypeImage {
		// 重新打开文件进行解码验证
		f2, err := file.Open()
		if err != nil {
			return ErrInvalidFileType
		}
		defer f2.Close()

		_, _, err = image.Decode(f2)
		if err != nil {
			return ErrInvalidFileType
		}
	}

	return nil
}

// validateFileSize 验证文件大小
func (s *MediaUploadService) validateFileSize(file *multipart.FileHeader, mediaType models.MediaType) error {
	maxSize := config.COSConfig.MaxFileSize
	if maxSize == 0 {
		maxSize = 10 * 1024 * 1024 // 默认 10MB
	}

	if file.Size > maxSize {
		return ErrFileTooLarge
	}

	return nil
}

// processImage 处理图片
func (s *MediaUploadService) processImage(ctx context.Context, file *multipart.FileHeader, key, ext string) (*UploadResponse, error) {
	// 打开文件
	f, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// 解码图片
	img, format, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("图片解码失败: %w", err)
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// 生成缩略图
	thumbnailKey := strings.Replace(key, ext, "_thumb"+ext, 1)
	_, err = s.generateThumbnail(ctx, img, thumbnailKey, format)
	if err != nil {
		return nil, err
	}

	return &UploadResponse{
		Type:     string(models.MediaTypeImage),
		Size:     file.Size,
		Width:    width,
		Height:   height,
		Format:   format,
	}, nil
}

// generateThumbnail 生成缩略图
func (s *MediaUploadService) generateThumbnail(ctx context.Context, img image.Image, key, format string) (string, error) {
	// 创建 200x200 的缩略图
	thumb := image.NewRGBA(image.Rect(0, 0, 200, 200))

	// 简单的缩放（实际应使用更智能的裁剪算法）
	bounds := img.Bounds()
	scale := float64(200) / float64(max(bounds.Dx(), bounds.Dy()))

	// 使用近邻插值（简单快速）
	for y := 0; y < 200; y++ {
		for x := 0; x < 200; x++ {
			srcX := int(float64(x) / scale)
			srcY := int(float64(y) / scale)
			if srcX >= bounds.Dx() {
				srcX = bounds.Dx() - 1
			}
			if srcY >= bounds.Dy() {
				srcY = bounds.Dy() - 1
			}
			thumb.Set(x, y, img.At(srcX, srcY))
		}
	}

	// 编码缩略图
	var buf bytes.Buffer
	encoder := &jpeg.Options{Quality: 80}
	if err := jpeg.Encode(&buf, thumb, encoder); err != nil {
		return "", err
	}

	// 上传到 COS
	if err := s.uploadBytesToCOS(ctx, buf.Bytes(), key); err != nil {
		return "", err
	}

	// 返回 CDN URL
	cdnDomain := config.COSConfig.CDNDomain
	if cdnDomain == "" {
		cdnDomain = fmt.Sprintf("https://%s.cos.%s.myqcloud.com", config.COSConfig.Bucket, config.COSConfig.Region)
	}
	return fmt.Sprintf("%s/%s", cdnDomain, key), nil
}

// processVoice 处理语音
func (s *MediaUploadService) processVoice(ctx context.Context, file *multipart.FileHeader, key, ext string) (*UploadResponse, error) {
	// 语音文件直接上传，不做处理
	// 实际项目中可以使用 ffmpeg 获取时长等信息
	return &UploadResponse{
		Type:   string(models.MediaTypeVoice),
		Size:   file.Size,
		Format: strings.TrimPrefix(ext, "."),
	}, nil
}

// processVideo 处理视频
func (s *MediaUploadService) processVideo(ctx context.Context, file *multipart.FileHeader, key, ext string) (*UploadResponse, error) {
	// 视频处理需要 ffmpeg，这里简化处理
	// 实际项目中应使用 ffmpeg 获取信息并生成封面
	return &UploadResponse{
		Type:   string(models.MediaTypeVideo),
		Size:   file.Size,
		Format: strings.TrimPrefix(ext, "."),
	}, nil
}

// uploadToCOS 上传文件到 COS
func (s *MediaUploadService) uploadToCOS(ctx context.Context, file multipart.File, key string) error {
	_, err := s.cosClient.Object.Put(ctx, key, file, nil)
	if err != nil {
		return fmt.Errorf("上传到COS失败: %w", err)
	}
	return nil
}

// uploadBytesToCOS 上传字节数组到 COS
func (s *MediaUploadService) uploadBytesToCOS(ctx context.Context, data []byte, key string) error {
	_, err := s.cosClient.Object.Put(ctx, key, bytes.NewReader(data), nil)
	if err != nil {
		return fmt.Errorf("上传到COS失败: %w", err)
	}
	return nil
}

// GeneratePresignedURL 生成预签名下载 URL
func (s *MediaUploadService) GeneratePresignedURL(ctx context.Context, mediaID int64, expiration time.Duration) (string, int64, error) {
	// 查询媒体信息
	var media models.Media
	if err := s.db.Where("id = ?", mediaID).First(&media).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", 0, ErrMediaNotFound
		}
		return "", 0, err
	}

	// 生成预签名 URL
	expires := int64(expiration.Seconds())
	
	// 使用简化的预签名 URL 生成
	var url string
	if s.cosClient != nil {
		presignedURL, err := s.cosClient.Object.GetPresignedURL(ctx, http.MethodGet, media.StorageKey, config.COSConfig.SecretID, config.COSConfig.SecretKey, expiration, nil)
		if err == nil {
			url = presignedURL.String()
		} else {
			url = media.OriginalURL
		}
	} else {
		url = media.OriginalURL
	}

	return url, expires, nil
}

// DeleteMedia 删除媒体文件
func (s *MediaUploadService) DeleteMedia(ctx context.Context, mediaID, userID int64) error {
	// 查询媒体信息
	var media models.Media
	if err := s.db.Where("id = ?", mediaID).First(&media).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrMediaNotFound
		}
		return err
	}

	// 验证所有权
	if media.UserID != userID {
		return ErrNotMediaOwner
	}

	// 从 COS 删除
	if s.cosClient != nil {
		_, err := s.cosClient.Object.Delete(ctx, media.StorageKey)
		if err != nil {
			return fmt.Errorf("删除COS文件失败: %w", err)
		}

		// 删除缩略图
		if media.ThumbnailURL != "" {
			thumbKey := strings.Replace(media.StorageKey, filepath.Ext(media.StorageKey), "_thumb"+filepath.Ext(media.StorageKey), 1)
			s.cosClient.Object.Delete(ctx, thumbKey)
		}
	}

	// 从数据库删除
	if err := s.db.Delete(&media).Error; err != nil {
		return err
	}

	// 更新用户配额
	s.updateUserQuota(userID, -media.FileSize)

	return nil
}

// GetMedia 获取媒体信息
func (s *MediaUploadService) GetMedia(ctx context.Context, mediaID int64) (*models.Media, error) {
	var media models.Media
	if err := s.db.Where("id = ?", mediaID).First(&media).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrMediaNotFound
		}
		return nil, err
	}
	return &media, nil
}

// updateUserQuota 更新用户配额
func (s *MediaUploadService) updateUserQuota(userID int64, delta int64) {
	var quota models.UserMediaQuota
	err := s.db.Where("user_id = ?", userID).First(&quota).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		quota = models.UserMediaQuota{
			UserID:     userID,
			TotalSize:  0,
			FileCount:  0,
		}
		s.db.Create(&quota)
	}

	quota.TotalSize += delta
	if delta > 0 {
		quota.FileCount++
	} else {
		quota.FileCount--
	}
	if quota.TotalSize < 0 {
		quota.TotalSize = 0
	}
	if quota.FileCount < 0 {
		quota.FileCount = 0
	}

	s.db.Model(&quota).Updates(map[string]interface{}{
		"total_size":  quota.TotalSize,
		"file_count": quota.FileCount,
	})
}

// GetUserQuota 获取用户配额
func (s *MediaUploadService) GetUserQuota(ctx context.Context, userID int64) (*models.UserMediaQuota, error) {
	var quota models.UserMediaQuota
	err := s.db.Where("user_id = ?", userID).First(&quota).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &quota, nil
	}
	return &quota, err
}

// generateUUID 生成 UUID
func generateUUID() string {
	// 简化版 UUID，实际应使用 uuid 包
	return fmt.Sprintf("%d%s", time.Now().UnixNano(), strconv.Itoa(os.Getpid()))
}
