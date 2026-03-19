package api

import (
	"net/http"
	"strconv"

	"im-backend/service"

	"github.com/gin-gonic/gin"
)

// ReadReceiptHandler 已读回执API处理程序
type ReadReceiptHandler struct {
	readService *service.ReadReceiptService
}

// NewReadReceiptHandler 创建已读回执处理器
func NewReadReceiptHandler(readService *service.ReadReceiptService) *ReadReceiptHandler {
	return &ReadReceiptHandler{
		readService: readService,
	}
}

// MarkReadRequest 标记已读请求
type MarkReadRequest struct {
	MessageIDs []int64 `json:"messageIds" binding:"required"`
}

// MarkReadResponse 标记已读响应
type MarkReadResponse struct {
	Code     int    `json:"code"`
	Message  string `json:"message"`
	ReadCount int64  `json:"readCount"`
}

// MarkMessagesAsRead 标记消息已读
// @Summary 标记消息已读
// @Description 标记指定消息为已读状态，同时更新未读计数
// @Tags 消息
// @Accept json
// @Produce json
// @Param messageIds body []int64 true "消息ID数组"
// @Success 200 {object} MarkReadResponse
// @Router /api/v1/messages/read [post]
func (h *ReadReceiptHandler) MarkMessagesAsRead(c *gin.Context) {
	// 从上下文获取用户ID（通过JWT认证中间件设置）
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未登录或登录已过期",
		})
		return
	}

	var req MarkReadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请求参数错误",
		})
		return
	}

	// 调用服务层标记已读
	readCount, err := h.readService.MarkMessagesAsRead(c.Request.Context(), userID.(int64), req.MessageIDs)
	if err != nil {
		code, message := mapReadErrorToCodeMessage(err)
		c.JSON(code, gin.H{
			"code":    code,
			"message": message,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":       0,
		"message":    "success",
		"readCount":  readCount,
	})
}

// GetReadStatusRequest 获取已读状态请求
type GetReadStatusRequest struct {
	MessageID int64 `uri:"id" binding:"required"`
}

// GetReadStatusResponse 获取已读状态响应
type GetReadStatusResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    *ReadStatusData `json:"data,omitempty"`
}

// ReadStatusData 已读状态数据
type ReadStatusData struct {
	MessageID  int64           `json:"messageId"`
	ReadBy     []ReadUserInfo  `json:"readBy"`
	UnreadBy   []ReadUserInfo  `json:"unreadBy"`
	ReadCount  int64           `json:"readCount"`
	UnreadCount int64          `json:"unreadCount"`
}

// ReadUserInfo 已读用户信息
type ReadUserInfo struct {
	UserID    int64  `json:"userId"`
	Username  string `json:"username"`
	ReadAt    string `json:"readAt,omitempty"`
}

// GetMessageReadStatus 获取消息已读状态
// @Summary 获取消息已读状态
// @Description 查询指定消息的已读用户列表和未读用户列表
// @Tags 消息
// @Accept json
// @Produce json
// @Param id path int true "消息ID"
// @Param members query string false "群成员ID列表，逗号分隔"
// @Success 200 {object} GetReadStatusResponse
// @Router /api/v1/messages/:id/read-status [get]
func (h *ReadReceiptHandler) GetMessageReadStatus(c *gin.Context) {
	// 从上下文获取用户ID
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未登录或登录已过期",
		})
		return
	}

	// 解析消息ID
	messageIDStr := c.Param("id")
	messageID, err := strconv.ParseInt(messageIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的消息ID",
		})
		return
	}

	// 解析群成员ID列表
	membersStr := c.Query("members")
	var members []int64
	if membersStr != "" {
		members, err = parseMemberIDs(membersStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": "无效的成员ID列表",
			})
			return
		}
	}

	// 调用服务层获取已读状态
	readStatus, err := h.readService.GetMessageReadStatus(c.Request.Context(), messageID, members)
	if err != nil {
		code, message := mapReadErrorToCodeMessage(err)
		c.JSON(code, gin.H{
			"code":    code,
			"message": message,
		})
		return
	}

	// 转换数据格式
	readBy := make([]ReadUserInfo, len(readStatus.ReadBy))
	for i, u := range readStatus.ReadBy {
		readBy[i] = ReadUserInfo{
			UserID:   u.UserID,
			Username: u.Username,
			ReadAt:   u.ReadAt,
		}
	}

	unreadBy := make([]ReadUserInfo, len(readStatus.UnreadBy))
	for i, u := range readStatus.UnreadBy {
		unreadBy[i] = ReadUserInfo{
			UserID:   u.UserID,
			Username: u.Username,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"message": "success",
		"data": ReadStatusData{
			MessageID:    readStatus.MessageID,
			ReadBy:       readBy,
			UnreadBy:     unreadBy,
			ReadCount:    readStatus.ReadCount,
			UnreadCount:  readStatus.UnreadCount,
		},
	})
}

// GetConversationUnreadCount 获取会话未读数
// @Summary 获取会话未读数
// @Description 获取指定会话的未读消息数量
// @Tags 会话
// @Accept json
// @Produce json
// @Param conversationId path int true "会话ID"
// @Success 200 {object} gin.H
// @Router /api/v1/conversations/:conversationId/unread-count [get]
func (h *ReadReceiptHandler) GetConversationUnreadCount(c *gin.Context) {
	// 从上下文获取用户ID
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未登录或登录已过期",
		})
		return
	}

	// 解析会话ID
	conversationIDStr := c.Param("conversationId")
	conversationID, err := strconv.ParseInt(conversationIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的会话ID",
		})
		return
	}

	// 调用服务层获取未读数
	count, err := h.readService.GetConversationUnreadCount(c.Request.Context(), conversationID, userID.(int64))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取未读数失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"conversationId": conversationID,
			"unreadCount":     count,
		},
	})
}

// MarkConversationAsRead 标记整个会话为已读
// @Summary 标记会话已读
// @Description 将指定会话的所有未读消息标记为已读
// @Tags 会话
// @Accept json
// @Produce json
// @Param conversationId path int true "会话ID"
// @Param messageIds body []int64 false "可选，指定要标记已读的消息ID数组"
// @Success 200 {object} gin.H
// @Router /api/v1/conversations/:conversationId/read [post]
func (h *ReadReceiptHandler) MarkConversationAsRead(c *gin.Context) {
	// 从上下文获取用户ID
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未登录或登录已过期",
		})
		return
	}

	// 解析会话ID
	conversationIDStr := c.Param("conversationId")
	conversationID, err := strconv.ParseInt(conversationIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的会话ID",
		})
		return
	}

	// 解析可选的消息ID数组
	var messageIDs []int64
	c.ShouldBindJSON(&messageIDs)

	// 调用服务层
	readCount, err := h.readService.MarkConversationAsRead(c.Request.Context(), conversationID, userID.(int64), messageIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "标记已读失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":       0,
		"message":    "success",
		"readCount":  readCount,
	})
}

// parseMemberIDs 解析逗号分隔的成员ID
func parseMemberIDs(s string) ([]int64, error) {
	if s == "" {
		return nil, nil
	}

	var result []int64
	ids := splitAndTrim(s, ",")
	for _, idStr := range ids {
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			return nil, err
		}
		result = append(result, id)
	}
	return result, nil
}

// splitAndTrim 分割并trim
func splitAndTrim(s, sep string) []string {
	parts := make([]string, 0)
	for _, p := range split(s, sep) {
		if trimmed := trim(p); trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return parts
}

func split(s, sep string) []string {
	result := make([]string, 0)
	start := 0
	for i := 0; i <= len(s)-len(sep); i++ {
		if s[i:i+len(sep)] == sep {
			result = append(result, s[start:i])
			start = i + len(sep)
		}
	}
	result = append(result, s[start:])
	return result
}

func trim(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}

// mapReadErrorToCodeMessage 将错误映射为HTTP状态码和消息
func mapReadErrorToCodeMessage(err error) (int, string) {
	switch err {
	case service.ErrMessageNotFound:
		return 404, "消息不存在"
	case service.ErrAlreadyRead:
		return 400, "消息已读"
	case service.ErrInvalidMessageIDs:
		return 400, "无效的消息ID列表"
	default:
		return 500, "服务器内部错误"
	}
}
