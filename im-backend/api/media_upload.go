package api

import (
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"im-backend/models"
	"im-backend/service"

	"github.com/gin-gonic/gin"
)

// MediaUploadHandler 媒体上传API处理程序
type MediaUploadHandler struct {
	mediaService *service.MediaUploadService
}

// NewMediaUploadHandler 创建媒体上传处理器
func NewMediaUploadHandler(mediaService *service.MediaUploadService) *MediaUploadHandler {
	return &MediaUploadHandler{
		mediaService: mediaService,
	}
}

// UploadResponse 上传响应
type UploadResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    *UploadData `json:"data,omitempty"`
}

// UploadData 上传数据
type UploadData struct {
	URL          string  `json:"url"`
	ThumbnailURL string  `json:"thumbnailUrl,omitempty"`
	Type         string  `json:"type"`
	Size         int64   `json:"size"`
	Width        int     `json:"width,omitempty"`
	Height       int     `json:"height,omitempty"`
	Duration     float64 `json:"duration,omitempty"`
	Format       string  `json:"format,omitempty"`
}

// UploadMedia 上传媒体文件
// @Summary 上传媒体文件
// @Description 上传图片、语音、视频文件到对象存储
// @Tags 媒体
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "文件"
// @Param type formData string true "文件类型 (image|voice|video)"
// @Success 200 {object} UploadResponse
// @Router /api/v1/media/upload [post]
func (h *MediaUploadHandler) UploadMedia(c *gin.Context) {
	// 从上下文获取用户ID
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未登录或登录已过期",
		})
		return
	}

	// 获取文件
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, UploadResponse{
			Code:    400,
			Message: "请选择要上传的文件",
		})
		return
	}

	// 获取文件类型
	fileType := c.PostForm("type")
	mediaType := models.MediaType(fileType)
	if mediaType != models.MediaTypeImage && mediaType != models.MediaTypeVoice && mediaType != models.MediaTypeVideo {
		c.JSON(http.StatusBadRequest, UploadResponse{
			Code:    400,
			Message: "无效的文件类型，请使用 image、voice 或 video",
		})
		return
	}

	// 验证文件扩展名
	ext := filepath.Ext(file.Filename)
	if !isAllowedExtension(ext, mediaType) {
		c.JSON(http.StatusBadRequest, UploadResponse{
			Code:    400,
			Message: "不支持的文件格式",
		})
		return
	}

	// 调用服务层上传
	req := &service.UploadRequest{
		File:   file,
		Type:   mediaType,
		UserID: userID.(int64),
	}

	resp, err := h.mediaService.Upload(c.Request.Context(), req)
	if err != nil {
		code, message := mapMediaErrorToCodeMessage(err)
		c.JSON(http.StatusBadRequest, UploadResponse{
			Code:    code,
			Message: message,
		})
		return
	}

	c.JSON(http.StatusOK, UploadResponse{
		Code:    0,
		Message: "success",
		Data: &UploadData{
			URL:          resp.URL,
			ThumbnailURL: resp.ThumbnailURL,
			Type:         resp.Type,
			Size:         resp.Size,
			Width:        resp.Width,
			Height:       resp.Height,
			Duration:     resp.Duration,
			Format:       resp.Format,
		},
	})
}

// GetMediaURLRequest 获取媒体URL请求
type GetMediaURLRequest struct {
	MediaID     int64     `uri:"id" binding:"required"`
	Expiration  time.Duration `form:"expiration,default=1h"`
}

// GetMediaURLResponse 获取媒体URL响应
type GetMediaURLResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    *GetMediaURLData `json:"data,omitempty"`
}

// GetMediaURLData 获取媒体URL数据
type GetMediaURLData struct {
	URL      string `json:"url"`
	ExpireAt int64  `json:"expireAt"`
}

// GetMediaURL 获取媒体下载链接
// @Summary 获取媒体下载链接
// @Description 获取媒体文件的临时下载链接
// @Tags 媒体
// @Accept json
// @Produce json
// @Param id path int true "媒体ID"
// @Param expiration query string false "过期时间，默认1h"
// @Success 200 {object} GetMediaURLResponse
// @Router /api/v1/media/:id/url [get]
func (h *MediaUploadHandler) GetMediaURL(c *gin.Context) {
	// 从上下文获取用户ID
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未登录或登录已过期",
		})
		return
	}

	// 解析媒体ID
	mediaIDStr := c.Param("id")
	mediaID, err := strconv.ParseInt(mediaIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的媒体ID",
		})
		return
	}

	// 解析过期时间
	expirationStr := c.DefaultQuery("expiration", "1h")
	expiration, err := time.ParseDuration(expirationStr)
	if err != nil || expiration <= 0 {
		expiration = time.Hour
	}

	// 调用服务层
	url, expireAt, err := h.mediaService.GeneratePresignedURL(c.Request.Context(), mediaID, expiration)
	if err != nil {
		code, message := mapMediaErrorToCodeMessage(err)
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    code,
			"message": message,
		})
		return
	}

	_ = userID // TODO: 验证用户是否有权限访问

	c.JSON(http.StatusOK, GetMediaURLResponse{
		Code:    0,
		Message: "success",
		Data: &GetMediaURLData{
			URL:      url,
			ExpireAt: expireAt,
		},
	})
}

// DeleteMediaRequest 删除媒体请求
type DeleteMediaRequest struct {
	MediaID int64 `uri:"id" binding:"required"`
}

// DeleteMediaResponse 删除媒体响应
type DeleteMediaResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// DeleteMedia 删除媒体文件
// @Summary 删除媒体文件
// @Description 删除指定的媒体文件
// @Tags 媒体
// @Accept json
// @Produce json
// @Param id path int true "媒体ID"
// @Success 200 {object} DeleteMediaResponse
// @Router /api/v1/media/:id [delete]
func (h *MediaUploadHandler) DeleteMedia(c *gin.Context) {
	// 从上下文获取用户ID
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未登录或登录已过期",
		})
		return
	}

	// 解析媒体ID
	mediaIDStr := c.Param("id")
	mediaID, err := strconv.ParseInt(mediaIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的媒体ID",
		})
		return
	}

	// 调用服务层删除
	err = h.mediaService.DeleteMedia(c.Request.Context(), mediaID, userID.(int64))
	if err != nil {
		code, message := mapMediaErrorToCodeMessage(err)
		c.JSON(http.StatusBadRequest, DeleteMediaResponse{
			Code:    code,
			Message: message,
		})
		return
	}

	c.JSON(http.StatusOK, DeleteMediaResponse{
		Code:    0,
		Message: "success",
	})
}

// GetMediaInfo 获取媒体信息
// @Summary 获取媒体信息
// @Description 获取媒体文件的详细信息
// @Tags 媒体
// @Accept json
// @Produce json
// @Param id path int true "媒体ID"
// @Success 200 {object} gin.H
// @Router /api/v1/media/:id [get]
func (h *MediaUploadHandler) GetMediaInfo(c *gin.Context) {
	// 解析媒体ID
	mediaIDStr := c.Param("id")
	mediaID, err := strconv.ParseInt(mediaIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的媒体ID",
		})
		return
	}

	// 调用服务层获取
	media, err := h.mediaService.GetMedia(c.Request.Context(), mediaID)
	if err != nil {
		code, message := mapMediaErrorToCodeMessage(err)
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    code,
			"message": message,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"message": "success",
		"data": media,
	})
}

// GetUserQuota 获取用户配额
// @Summary 获取用户配额
// @Description 获取当前用户的媒体存储配额使用情况
// @Tags 用户
// @Accept json
// @Produce json
// @Success 200 {object} gin.H
// @Router /api/v1/user/quota [get]
func (h *MediaUploadHandler) GetUserQuota(c *gin.Context) {
	// 从上下文获取用户ID
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未登录或登录已过期",
		})
		return
	}

	quota, err := h.mediaService.GetUserQuota(c.Request.Context(), userID.(int64))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取配额失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"message": "success",
		"data": gin.H{
			"total_size": quota.TotalSize,
			"file_count": quota.FileCount,
		},
	})
}

// isAllowedExtension 检查文件扩展名是否允许
func isAllowedExtension(ext string, mediaType models.MediaType) bool {
	ext = filepath.Ext(ext)
	ext = ext[1:] // 去除点

	allowedExts := map[models.MediaType][]string{
		models.MediaTypeImage: {"jpg", "jpeg", "png", "gif", "webp", "bmp"},
		models.MediaTypeVoice: {"mp3", "aac", "m4a", "wav", "ogg", "flac"},
		models.MediaTypeVideo: {"mp4", "mov", "avi", "mkv", "webm", "flv"},
	}

	exts, ok := allowedExts[mediaType]
	if !ok {
		return false
	}

	for _, e := range exts {
		if e == ext {
			return true
		}
	}
	return false
}

// mapMediaErrorToCodeMessage 将错误映射为HTTP状态码和消息
func mapMediaErrorToCodeMessage(err error) (int, string) {
	switch err {
	case service.ErrInvalidFileType:
		return 40001, "不支持的文件类型"
	case service.ErrFileTooLarge:
		return 40002, "文件大小超过限制（最大10MB）"
	case service.ErrUploadFailed:
		return 50001, "文件上传失败"
	case service.ErrMediaNotFound:
		return 40401, "媒体文件不存在"
	case service.ErrNotMediaOwner:
		return 40301, "无权操作此文件"
	case service.ErrInvalidMediaType:
		return 40003, "无效的媒体类型"
	default:
		return 50000, "服务器内部错误"
	}
}
