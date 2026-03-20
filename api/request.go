package api

// ============ 通用响应 ============

// Response 通用响应结构
type Response struct {
	Code int         `json:"code" example:"0"`     // 状态码: 0-成功, 非0-失败
	Msg  string      `json:"msg" example:"success"` // 消息
	Data interface{} `json:"data,omitempty"`       // 数据
}

// PageInfo 分页信息
type PageInfo struct {
	Page     int `json:"page" example:"1"`      // 当前页码
	PageSize int `json:"pageSize" example:"20"` // 每页数量
	Total    int `json:"total" example:"100"`   // 总数
}

// ============ 认证相关 ============

// RegisterRequest 用户注册请求
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=20" example:"john_doe"`   // 用户名
	Password string `json:"password" binding:"required,min=6" example:"password123"`      // 密码
	Email    string `json:"email" example:"john@example.com"`                               // 邮箱
	Nickname string `json:"nickname" example:"John Doe"`                                    // 昵称
	Phone    string `json:"phone" example:"+86-13800138000"`                              // 手机号
}

// RegisterResponse 用户注册响应
type RegisterResponse struct {
	UserID   int64  `json:"userId" example:"1001"`    // 用户ID
	Username string `json:"username" example:"john_doe"` // 用户名
}

// LoginRequest 用户登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required" example:"john_doe"` // 用户名
	Password string `json:"password" binding:"required" example:"password123"` // 密码
}

// LoginResponse 用户登录响应
type LoginResponse struct {
	UserID   int64  `json:"userId" example:"1001"`    // 用户ID
	Username string `json:"username" example:"john_doe"` // 用户名
	Nickname string `json:"nickname" example:"John Doe"` // 昵称
	Avatar   string `json:"avatar" example:"https://example.com/avatar.jpg"` // 头像
	Token    string `json:"token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."` // JWT Token
}

// RefreshTokenRequest 刷新Token请求
type RefreshTokenRequest struct {
	Token string `json:"token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."` // 当前Token
}

// RefreshTokenResponse 刷新Token响应
type RefreshTokenResponse struct {
	Token string `json:"token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."` // 新Token
}

// ============ 用户相关 ============

// GetProfileResponse 获取用户资料响应
type GetProfileResponse struct {
	UserID   int64  `json:"userId" example:"1001"`                   // 用户ID
	Username string `json:"username" example:"john_doe"`             // 用户名
	Nickname string `json:"nickname" example:"John Doe"`              // 昵称
	Avatar   string `json:"avatar" example:"https://example.com/avatar.jpg"` // 头像
	Email    string `json:"email" example:"john@example.com"`          // 邮箱
	Phone    string `json:"phone" example:"+86-13800138000"`           // 手机号
	Gender   string `json:"gender" example:"male"`                    // 性别: male, female, unknown
}

// UpdateProfileRequest 更新用户资料请求
type UpdateProfileRequest struct {
	Nickname string `json:"nickname" example:"John Doe"`                      // 昵称
	Avatar   string `json:"avatar" example:"https://example.com/avatar.jpg"` // 头像URL
	Email    string `json:"email" example:"john@example.com"`                 // 邮箱
	Phone    string `json:"phone" example:"+86-13800138000"`                  // 手机号
	Gender   string `json:"gender" example:"male"`                             // 性别
}

// FriendInfo 好友信息
type FriendInfo struct {
	UserID   int64  `json:"userId" example:"1001"`                   // 好友ID
	Username string `json:"username" example:"john_doe"`              // 好友用户名
	Nickname string `json:"nickname" example:"John Doe"`              // 好友昵称
	Avatar   string `json:"avatar" example:"https://example.com/avatar.jpg"` // 好友头像
	Remark   string `json:"remark" example:"My Friend"`               // 好友备注
}

// AddFriendRequest 添加好友请求
type AddFriendRequest struct {
	FriendID int64  `json:"friendId" binding:"required" example:"1002"` // 好友用户ID
	Remark   string `json:"remark" example:"My Friend"`                 // 备注
}

// GetUserByIDRequest 根据ID获取用户请求
type GetUserByIDRequest struct {
	UserID int64 `json:"userId" example:"1001"` // 用户ID
}

// GetUserByIDResponse 根据ID获取用户响应
type GetUserByIDResponse struct {
	UserID   int64  `json:"userId" example:"1001"`                   // 用户ID
	Username string `json:"username" example:"john_doe"`              // 用户名
	Nickname string `json:"nickname" example:"John Doe"`              // 昵称
	Avatar   string `json:"avatar" example:"https://example.com/avatar.jpg"` // 头像
	Gender   string `json:"gender" example:"male"`                    // 性别
}

// SearchUserRequest 搜索用户请求
type SearchUserRequest struct {
	Keyword string `json:"keyword" example:"john"` // 搜索关键词
}

// ============ 会话相关 ============

// ConversationInfo 会话信息
type ConversationInfo struct {
	ID              int64     `json:"id" example:"1"`                             // 会话ID
	Type            string    `json:"type" example:"private"`                     // 类型: private, group
	Name            string    `json:"name" example:"Chat with John"`              // 会话名称
	LastMsgContent string    `json:"lastMsgContent" example:"Hello"`             // 最后一条消息内容
	LastMsgTime     int64     `json:"lastMsgTime" example:"1640000000000"`        // 最后消息时间戳
	UnreadCount     int64     `json:"unreadCount" example:"5"`                   // 未读数
	Members         []int64   `json:"members,omitempty"`                         // 成员ID列表
	Avatar          string    `json:"avatar,omitempty"`                           // 群头像/私聊对方头像
}

// CreateConversationRequest 创建会话请求
type CreateConversationRequest struct {
	Type      string `json:"type" binding:"required" example:"private"`       // 会话类型: private, group
	Name      string `json:"name" example:"My Group"`                         // 会话名称(群聊必填)
	MemberIDs []int64 `json:"memberIds" example:"[1001,1002,1003]"`           // 成员ID列表
}

// CreateConversationResponse 创建会话响应
type CreateConversationResponse struct {
	ConversationID int64 `json:"conversationId" example:"1"` // 会话ID
}

// GetConversationsRequest 获取会话列表请求
type GetConversationsRequest struct {
	Page     int `json:"page" example:"1"`      // 页码
	PageSize int `json:"pageSize" example:"20"` // 每页数量
}

// ============ 消息相关 ============

// SendMessageRequest 发送消息请求
type SendMessageRequest struct {
	ConversationID int64  `json:"conversationId" binding:"required" example:"1"`     // 会话ID
	Content        string `json:"content" binding:"required" example:"Hello"`         // 消息内容
	ContentType    string `json:"contentType" example:"text"`                         // 内容类型: text, image, voice, video, file
	Extra          string `json:"extra,omitempty"`                                     // 扩展字段(JSON)
}

// SendMessageResponse 发送消息响应
type SendMessageResponse struct {
	MessageID int64  `json:"messageId" example:"1001"`           // 消息ID
	MsgID     string `json:"msgId" example:"msg_1640000000_123"`  // 全局唯一消息ID
	Timestamp int64  `json:"timestamp" example:"1640000000000"`  // 发送时间戳
}

// MessageInfo 消息信息
type MessageInfo struct {
	ID             int64  `json:"id" example:"1001"`                       // 消息ID
	MsgID          string `json:"msgId" example:"msg_1640000000_123"`      // 全局唯一消息ID
	ConversationID int64  `json:"conversationId" example:"1"`              // 会话ID
	SenderID       int64  `json:"senderId" example:"1001"`                // 发送者ID
	Content        string `json:"content" example:"Hello"`                // 消息内容
	ContentType    string `json:"contentType" example:"text"`              // 内容类型
	Extra          string `json:"extra,omitempty"`                        // 扩展字段
	IsRecalled     bool   `json:"isRecalled" example:"false"`              // 是否已撤回
	CreatedAt      int64  `json:"createdAt" example:"1640000000000"`       // 创建时间
}

// GetHistoryRequest 获取历史消息请求
type GetHistoryRequest struct {
	ConversationID int64 `json:"conversationId" binding:"required" example:"1"` // 会话ID
	Limit         int    `json:"limit" example:"50"`                              // 数量限制
	Offset        int    `json:"offset" example:"0"`                              // 偏移量
	StartTime     int64  `json:"startTime,omitempty"`                             // 开始时间戳
	EndTime       int64  `json:"endTime,omitempty"`                              // 结束时间戳
}

// GetHistoryResponse 获取历史消息响应
type GetHistoryResponse struct {
	Messages []MessageInfo `json:"messages"` // 消息列表
	PageInfo PageInfo      `json:"pageInfo"` // 分页信息
}

// RevokeMessageRequest 撤回消息请求
type RevokeMessageRequest struct {
	MessageID int64 `json:"messageId" binding:"required" example:"1001"` // 消息ID
}

// MarkAsReadRequest 标记已读请求
type MarkAsReadRequest struct {
	MessageIDs []int64 `json:"messageIds" example:"[1001,1002,1003]"` // 消息ID列表
}

// ============ 群组相关 ============

// CreateGroupRequest 创建群组请求
type CreateGroupRequest struct {
	Name        string  `json:"name" binding:"required" example:"My Group"`       // 群名称
	Avatar      string  `json:"avatar" example:"https://example.com/group.jpg"`  // 群头像
	Description string  `json:"description" example:"This is a test group"`    // 群描述
	MemberIDs   []int64 `json:"memberIds" example:"[1001,1002,1003]"`           // 初始成员ID
}

// CreateGroupResponse 创建群组响应
type CreateGroupResponse struct {
	GroupID int64 `json:"groupId" example:"1"` // 群组ID
}

// GroupInfo 群组信息
type GroupInfo struct {
	ID          int64    `json:"id" example:"1"`                         // 群ID
	Name        string   `json:"name" example:"My Group"`                // 群名称
	Avatar      string   `json:"avatar" example:"https://example.com/group.jpg"` // 群头像
	Description string   `json:"description" example:"Group description"` // 群描述
	OwnerID     int64    `json:"ownerId" example:"1001"`                 // 群主ID
	MemberCount int      `json:"memberCount" example:"10"`               // 成员数量
	CreatedAt   int64    `json:"createdAt" example:"1640000000000"`      // 创建时间
}

// GetGroupInfoRequest 获取群组信息请求
type GetGroupInfoRequest struct {
	GroupID int64 `json:"groupId" example:"1"` // 群组ID
}

// AddGroupMemberRequest 添加群成员请求
type AddGroupMemberRequest struct {
	UserID int64  `json:"userId" binding:"required" example:"1002"` // 用户ID
	Remark string `json:"remark" example:"New member"`              // 备注
}

// RemoveGroupMemberRequest 移除群成员请求
type RemoveGroupMemberRequest struct {
	UserID int64 `json:"userId" binding:"required" example:"1002"` // 用户ID
}

// MuteGroupMemberRequest 禁言群成员请求
type MuteGroupMemberRequest struct {
	UserID    int64 `json:"userId" binding:"required" example:"1002"` // 用户ID
	MuteMinutes int `json:"muteMinutes" example:"10"`                 // 禁言分钟数
}

// GroupMemberInfo 群成员信息
type GroupMemberInfo struct {
	UserID   int64  `json:"userId" example:"1001"`     // 用户ID
	Username string `json:"username" example:"john_doe"` // 用户名
	Nickname string `json:"nickname" example:"John Doe"` // 群昵称
	Avatar   string `json:"avatar" example:"https://example.com/avatar.jpg"` // 头像
	Role     string `json:"role" example:"member"`     // 角色: owner, admin, member
	JoinedAt int64  `json:"joinedAt" example:"1640000000000"` // 加入时间
}

// ============ 文件上传 ============

// UploadMediaRequest 上传媒体文件请求
// 通过 multipart/form-data 上传
type UploadMediaRequest struct {
	File     []byte `json:"-" form:"file"`                   // 文件内容
	FileName string `json:"fileName" example:"image.jpg"`    // 文件名
	FileType string `json:"fileType" example:"image/jpeg"`   // 文件类型
}

// UploadMediaResponse 上传媒体文件响应
type UploadMediaResponse struct {
	URL      string `json:"url" example:"https://cdn.example.com/files/xxx.jpg"` // 文件URL
	FileSize int64  `json:"fileSize" example:"102400"`                            // 文件大小
	FileType string `json:"fileType" example:"image/jpeg"`                        // 文件类型
}

// ============ 错误码定义 ============

const (
	// 通用错误码
	ErrCodeSuccess       = 0
	ErrCodeInvalidParam  = 400
	ErrCodeUnauthorized = 401
	ErrCodeForbidden    = 403
	ErrCodeNotFound     = 404
	ErrCodeServerError  = 500

	// 业务错误码
	ErrCodeUserAlreadyExist    = 1001
	ErrCodeUserNotFound        = 1002
	ErrCodeInvalidPassword     = 1003
	ErrCodeConversationNotFound = 2001
	ErrCodeNoPermission        = 2002
	ErrCodeMessageNotFound     = 3001
	ErrCodeMessageTooOld        = 3002
	ErrCodeGroupNotFound       = 4001
	ErrCodeGroupFull           = 4002
	ErrCodeAlreadyInGroup      = 4003
	ErrCodeNotInGroup         = 4004
)
