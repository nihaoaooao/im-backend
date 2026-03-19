package api

import (
	"net/http"
	"strconv"

	"im-backend/service"
	"im-backend/utils"

	"github.com/gin-gonic/gin"
)

// MessageRecallHandler 消息撤回API处理程序
type MessageRecallHandler struct {
	messageService *service.MessageService
}

// NewMessageRecallHandler 创建消息撤回处理器
func NewMessageRecallHandler(messageService *service.MessageService) *MessageRecallHandler {
	return &MessageRecallHandler{
		messageService: messageService,
	}
}

// RecallRequest 撤回请求
type RecallRequest struct {
	MsgID string `json:"msg_id" binding:"required"`
}

// RecallResponse 撤回响应
type RecallResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	MsgID   string `json:"msg_id,omitempty"`
}

// RecallMessage 撤回消息
// @Summary 撤回消息
// @Description 撤回指定的消息，只能撤回自己发送的消息，且需在2分钟内
// @Tags 消息
// @Accept json
// @Produce json
// @Param msg_id body string true "消息ID"
// @Success 200 {object} RecallResponse
// @Router /api/v1/messages/recall [post]
func (h *MessageRecallHandler) RecallMessage(c *gin.Context) {
	// 从上下文获取用户ID（通过JWT认证中间件设置）
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, RecallResponse{
			Code:    401,
			Message: "未登录或登录已过期",
		})
		return
	}

	var req RecallRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, RecallResponse{
			Code:    400,
			Message: "请求参数错误",
		})
		return
	}

	// 调用服务层撤回消息
	err := h.messageService.RecallMessage(userID.(int64), req.MsgID)
	if err != nil {
		code, message := mapErrorToCodeMessage(err)
		c.JSON(code, RecallResponse{
			Code:    code,
			Message: message,
			MsgID:   req.MsgID,
		})
		return
	}

	c.JSON(http.StatusOK, RecallResponse{
		Code:    200,
		Message: "消息撤回成功",
		MsgID:   req.MsgID,
	})
}

// GetRecallableMessages 获取可撤回的消息列表
func (h *MessageRecallHandler) GetRecallableMessages(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未登录或登录已过期",
		})
		return
	}

	limitStr := c.DefaultQuery("limit", "50")
	limit, err := strconv.ParseInt(limitStr, 10, 64)
	if err != nil || limit <= 0 {
		limit = 50
	}

	messages, err := h.messageService.GetRecallableMessages(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取消息列表失败",
		})
		return
	}

	// 过滤出当前用户发送的消息
	var recallableMessages []gin.H
	for _, msg := range messages {
		if msg.SenderID == userID.(int64) && !msg.Revoked {
			recallableMessages = append(recallableMessages, gin.H{
				"msg_id":          msg.MsgID,
				"conversation_id": msg.ConversationID,
				"content":         utils.MaskSensitiveData(msg.Content),
				"created_at":      msg.CreatedAt,
				"can_revoke":      msg.CanRevoke,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "success",
		"data":    recallableMessages,
	})
}

// mapErrorToCodeMessage 将错误映射为HTTP状态码和消息
func mapErrorToCodeMessage(err error) (int, string) {
	switch err {
	case service.ErrMessageNotFound:
		return 404, "消息不存在"
	case service.ErrNotMessageSender:
		return 403, "只能撤回自己发送的消息"
	case service.ErrMessageAlreadyRevoked:
		return 400, "消息已撤回"
	case service.ErrCannotRevoke:
		return 400, "消息无法撤回（已超过撤回时限）"
	case service.ErrRecallTimeExpired:
		return 400, "消息撤回时限已过（2分钟）"
	default:
		return 500, "服务器内部错误"
	}
}
