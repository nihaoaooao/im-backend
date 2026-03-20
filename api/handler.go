package api

import (
	"net/http"
	"time"

	"im-backend/model"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// Handler API 处理器
type Handler struct {
	db    *gorm.DB
	redis *redis.Client
}

// NewHandler 创建新的处理器
func NewHandler(db *gorm.DB, redis *redis.Client) *Handler {
	return &Handler{db: db, redis: redis}
}

// ============ 认证相关 API ============

// Register 用户注册
// @Summary 用户注册
// @Description 创建新用户账号
// @Tags 认证
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "注册请求"
// @Success 200 {object} Response{data=RegisterResponse}
// @Router /api/auth/register [post]
func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{Code: ErrCodeInvalidParam, Msg: "参数错误: " + err.Error()})
		return
	}

	// 检查用户名是否已存在
	var count int64
	h.db.Model(&model.User{}).Where("username = ?", req.Username).Count(&count)
	if count > 0 {
		c.JSON(http.StatusBadRequest, Response{Code: ErrCodeUserAlreadyExist, Msg: "用户名已存在"})
		return
	}

	// TODO: 加密密码并创建用户

	c.JSON(http.StatusOK, Response{
		Code: ErrCodeSuccess,
		Msg:  "success",
		Data: RegisterResponse{
			UserID:   0,
			Username: req.Username,
		},
	})
}

// Login 用户登录
// @Summary 用户登录
// @Description 用户登录获取 Token
// @Tags 认证
// @Accept json
// @Produce json
// @Param request body LoginRequest true "登录请求"
// @Success 200 {object} Response{data=LoginResponse}
// @Router /api/auth/login [post]
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{Code: ErrCodeInvalidParam, Msg: "参数错误"})
		return
	}

	// TODO: 验证用户并生成 Token

	c.JSON(http.StatusOK, Response{
		Code: ErrCodeSuccess,
		Msg:  "success",
		Data: LoginResponse{
			UserID:   0,
			Username: req.Username,
			Token:    "mock-token",
		},
	})
}

// RefreshToken 刷新 Token
// @Summary 刷新 Token
// @Description 刷新 JWT Token
// @Tags 认证
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} Response{data=RefreshTokenResponse}
// @Router /api/auth/refresh [post]
func (h *Handler) RefreshToken(c *gin.Context) {
	c.JSON(http.StatusOK, Response{
		Code: ErrCodeSuccess,
		Msg:  "success",
		Data: RefreshTokenResponse{
			Token: "new-mock-token",
		},
	})
}

// ============ 用户相关 API ============

// GetProfile 获取用户资料
// @Summary 获取用户资料
// @Description 获取当前登录用户资料
// @Tags 用户
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} Response{data=GetProfileResponse}
// @Router /api/user/profile [get]
func (h *Handler) GetProfile(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var user model.User
	if err := h.db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, Response{Code: ErrCodeUserNotFound, Msg: "用户不存在"})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code: ErrCodeSuccess,
		Msg:  "success",
		Data: GetProfileResponse{
			UserID:   user.ID,
			Username: user.Username,
			Nickname: user.Nickname,
			Avatar:   user.Avatar,
			Email:    user.Email,
			Phone:    user.Phone,
			Gender:   user.Gender,
		},
	})
}

// UpdateProfile 更新用户资料
// @Summary 更新用户资料
// @Description 更新当前登录用户资料
// @Tags 用户
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body UpdateProfileRequest true "更新资料请求"
// @Success 200 {object} Response
// @Router /api/user/profile [put]
func (h *Handler) UpdateProfile(c *gin.Context) {
	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{Code: ErrCodeInvalidParam, Msg: "参数错误"})
		return
	}

	// TODO: 更新用户资料

	c.JSON(http.StatusOK, Response{Code: ErrCodeSuccess, Msg: "更新成功"})
}

// GetFriends 获取好友列表
// @Summary 获取好友列表
// @Description 获取当前用户的好友列表
// @Tags 用户
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} Response{data=[]FriendInfo}
// @Router /api/user/friends [get]
func (h *Handler) GetFriends(c *gin.Context) {
	c.JSON(http.StatusOK, Response{
		Code: ErrCodeSuccess,
		Msg:  "success",
		Data: []FriendInfo{},
	})
}

// AddFriend 添加好友
// @Summary 添加好友
// @Description 添加指定用户为好友
// @Tags 用户
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body AddFriendRequest true "添加好友请求"
// @Success 200 {object} Response
// @Router /api/user/friend [post]
func (h *Handler) AddFriend(c *gin.Context) {
	var req AddFriendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{Code: ErrCodeInvalidParam, Msg: "参数错误"})
		return
	}

	c.JSON(http.StatusOK, Response{Code: ErrCodeSuccess, Msg: "添加成功"})
}

// GetUserByID 根据ID获取用户
// @Summary 根据ID获取用户
// @Description 根据用户ID获取用户信息
// @Tags 用户
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param userId path int true "用户ID"
// @Success 200 {object} Response{data=GetUserByIDResponse}
// @Router /api/user/{userId} [get]
func (h *Handler) GetUserByID(c *gin.Context) {
	userID := c.Param("userId")
	// TODO: 查询用户

	c.JSON(http.StatusOK, Response{
		Code: ErrCodeSuccess,
		Msg:  "success",
		Data: GetUserByIDResponse{},
	})
}

// SearchUser 搜索用户
// @Summary 搜索用户
// @Description 根据关键词搜索用户
// @Tags 用户
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param keyword query string true "搜索关键词"
// @Success 200 {object} Response{data=[]GetUserByIDResponse}
// @Router /api/user/search [get]
func (h *Handler) SearchUser(c *gin.Context) {
	keyword := c.Query("keyword")
	// TODO: 搜索用户

	c.JSON(http.StatusOK, Response{
		Code: ErrCodeSuccess,
		Msg:  "success",
		Data: []GetUserByIDResponse{},
	})
}

// ============ 会话相关 API ============

// GetConversations 获取会话列表
// @Summary 获取会话列表
// @Description 获取当前用户的所有会话列表
// @Tags 会话
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码"
// @Param pageSize query int false "每页数量"
// @Success 200 {object} Response{data=[]ConversationInfo}
// @Router /api/conversations [get]
func (h *Handler) GetConversations(c *gin.Context) {
	c.JSON(http.StatusOK, Response{
		Code: ErrCodeSuccess,
		Msg:  "success",
		Data: []ConversationInfo{},
	})
}

// CreateConversation 创建会话
// @Summary 创建会话
// @Description 创建私聊或群聊会话
// @Tags 会话
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateConversationRequest true "创建会话请求"
// @Success 200 {object} Response{data=CreateConversationResponse}
// @Router /api/conversations [post]
func (h *Handler) CreateConversation(c *gin.Context) {
	var req CreateConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{Code: ErrCodeInvalidParam, Msg: "参数错误"})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code: ErrCodeSuccess,
		Msg:  "success",
		Data: CreateConversationResponse{
			ConversationID: 0,
		},
	})
}

// DeleteConversation 删除会话
// @Summary 删除会话
// @Description 删除指定的会话
// @Tags 会话
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param conversationId path int true "会话ID"
// @Success 200 {object} Response
// @Router /api/conversations/{conversationId} [delete]
func (h *Handler) DeleteConversation(c *gin.Context) {
	conversationID := c.Param("conversationId")
	// TODO: 删除会话

	c.JSON(http.StatusOK, Response{Code: ErrCodeSuccess, Msg: "删除成功"})
}

// ============ 消息相关 API ============

// SendMessage 发送消息
// @Summary 发送消息
// @Description 向指定会话发送消息
// @Tags 消息
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body SendMessageRequest true "发送消息请求"
// @Success 200 {object} Response{data=SendMessageResponse}
// @Router /api/messages/send [post]
func (h *Handler) SendMessage(c *gin.Context) {
	var req SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{Code: ErrCodeInvalidParam, Msg: "参数错误"})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code: ErrCodeSuccess,
		Msg:  "success",
		Data: SendMessageResponse{
			MessageID: 0,
			MsgID:     "msg_" + string(rune(time.Now().UnixMilli())),
			Timestamp: time.Now().UnixMilli(),
		},
	})
}

// GetHistory 获取历史消息
// @Summary 获取历史消息
// @Description 获取指定会话的历史消息
// @Tags 消息
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param conversationId query int true "会话ID"
// @Param limit query int false "数量限制"
// @Param offset query int false "偏移量"
// @Param startTime query int false "开始时间戳"
// @Param endTime query int false "结束时间戳"
// @Success 200 {object} Response{data=GetHistoryResponse}
// @Router /api/messages/history [get]
func (h *Handler) GetHistory(c *gin.Context) {
	c.JSON(http.StatusOK, Response{
		Code: ErrCodeSuccess,
		Msg:  "success",
		Data: GetHistoryResponse{
			Messages: []MessageInfo{},
			PageInfo: PageInfo{},
		},
	})
}

// RevokeMessage 撤回消息
// @Summary 撤回消息
// @Description 撤回指定消息(仅可撤回2分钟内的消息)
// @Tags 消息
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body RevokeMessageRequest true "撤回消息请求"
// @Success 200 {object} Response
// @Router /api/messages/revoke [post]
func (h *Handler) RevokeMessage(c *gin.Context) {
	var req RevokeMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{Code: ErrCodeInvalidParam, Msg: "参数错误"})
		return
	}

	c.JSON(http.StatusOK, Response{Code: ErrCodeSuccess, Msg: "已撤回"})
}

// MarkAsRead 标记已读
// @Summary 标记已读
// @Description 标记消息为已读
// @Tags 消息
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body MarkAsReadRequest true "标记已读请求"
// @Success 200 {object} Response
// @Router /api/messages/read [post]
func (h *Handler) MarkAsRead(c *gin.Context) {
	var req MarkAsReadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{Code: ErrCodeInvalidParam, Msg: "参数错误"})
		return
	}

	c.JSON(http.StatusOK, Response{Code: ErrCodeSuccess, Msg: "已读"})
}

// UploadMedia 上传媒体文件
// @Summary 上传媒体文件
// @Description 上传图片、音频、视频等媒体文件
// @Tags 文件
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param file formData file true "文件"
// @Param fileType query string false "文件类型"
// @Success 200 {object} Response{data=UploadMediaResponse}
// @Router /api/media/upload [post]
func (h *Handler) UploadMedia(c *gin.Context) {
	c.JSON(http.StatusOK, Response{
		Code: ErrCodeSuccess,
		Msg:  "success",
		Data: UploadMediaResponse{
			URL:      "https://cdn.example.com/files/mock.jpg",
			FileSize: 0,
			FileType: "image/jpeg",
		},
	})
}

// ============ 群组相关 API ============

// CreateGroup 创建群组
// @Summary 创建群组
// @Description 创建一个新的群组
// @Tags 群组
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateGroupRequest true "创建群组请求"
// @Success 200 {object} Response{data=CreateGroupResponse}
// @Router /api/group/create [post]
func (h *Handler) CreateGroup(c *gin.Context) {
	var req CreateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{Code: ErrCodeInvalidParam, Msg: "参数错误"})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code: ErrCodeSuccess,
		Msg:  "success",
		Data: CreateGroupResponse{
			GroupID: 0,
		},
	})
}

// GetGroupInfo 获取群组信息
// @Summary 获取群组信息
// @Description 获取指定群组的信息
// @Tags 群组
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "群组ID"
// @Success 200 {object} Response{data=GroupInfo}
// @Router /api/group/{id} [get]
func (h *Handler) GetGroupInfo(c *gin.Context) {
	groupID := c.Param("id")
	// TODO: 获取群组信息

	c.JSON(http.StatusOK, Response{
		Code: ErrCodeSuccess,
		Msg:  "success",
		Data: GroupInfo{},
	})
}

// AddMember 添加群成员
// @Summary 添加群成员
// @Description 向群组添加成员
// @Tags 群组
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "群组ID"
// @Param request body AddGroupMemberRequest true "添加成员请求"
// @Success 200 {object} Response
// @Router /api/group/{id}/member [post]
func (h *Handler) AddMember(c *gin.Context) {
	var req AddGroupMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{Code: ErrCodeInvalidParam, Msg: "参数错误"})
		return
	}

	c.JSON(http.StatusOK, Response{Code: ErrCodeSuccess, Msg: "添加成功"})
}

// RemoveMember 移除群成员
// @Summary 移除群成员
// @Description 从群组移除指定成员
// @Tags 群组
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "群组ID"
// @Param uid path int true "用户ID"
// @Success 200 {object} Response
// @Router /api/group/{id}/member/{uid} [delete]
func (h *Handler) RemoveMember(c *gin.Context) {
	c.JSON(http.StatusOK, Response{Code: ErrCodeSuccess, Msg: "移除成功"})
}

// MuteMember 禁言群成员
// @Summary 禁言群成员
// @Description 禁言群组成员
// @Tags 群组
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "群组ID"
// @Param request body MuteGroupMemberRequest true "禁言请求"
// @Success 200 {object} Response
// @Router /api/group/{id}/mute [post]
func (h *Handler) MuteMember(c *gin.Context) {
	var req MuteGroupMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{Code: ErrCodeInvalidParam, Msg: "参数错误"})
		return
	}

	c.JSON(http.StatusOK, Response{Code: ErrCodeSuccess, Msg: "禁言成功"})
}

// GetGroupMembers 获取群成员列表
// @Summary 获取群成员列表
// @Description 获取指定群组的成员列表
// @Tags 群组
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "群组ID"
// @Success 200 {object} Response{data=[]GroupMemberInfo}
// @Router /api/group/{id}/members [get]
func (h *Handler) GetGroupMembers(c *gin.Context) {
	c.JSON(http.StatusOK, Response{
		Code: ErrCodeSuccess,
		Msg:  "success",
		Data: []GroupMemberInfo{},
	})
}

// QuitGroup 退出群组
// @Summary 退出群组
// @Description 当前用户退出群组
// @Tags 群组
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "群组ID"
// @Success 200 {object} Response
// @Router /api/group/{id}/quit [post]
func (h *Handler) QuitGroup(c *gin.Context) {
	c.JSON(http.StatusOK, Response{Code: ErrCodeSuccess, Msg: "已退出群组"})
}

// DismissGroup 解散群组
// @Summary 解散群组
// @Description 解散指定群组(仅群主可操作)
// @Tags 群组
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "群组ID"
// @Success 200 {object} Response
// @Router /api/group/{id}/dismiss [post]
func (h *Handler) DismissGroup(c *gin.Context) {
	c.JSON(http.StatusOK, Response{Code: ErrCodeSuccess, Msg: "群组已解散"})
}
