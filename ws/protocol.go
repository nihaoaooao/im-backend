package ws

// ============ WebSocket 消息协议 ============
//
// 通信方式: JSON over WebSocket
// 编码: UTF-8
// 心跳: 每 30 秒发送一次 ping
//
// 客户端 -> 服务器消息格式:
// {
//     "type": "message",
//     "data": { ... },
//     "timestamp": 1640000000000
// }
//
// 服务器 -> 客户端消息格式:
// {
//     "type": "message",
//     "data": { ... },
//     "timestamp": 1640000000000
// }
//

// ============ 客户端发送消息类型 ============

// ClientMessage 客户端发送的消息
type ClientMessage struct {
	Type      string      `json:"type" example:"message"` // 消息类型
	Data      interface{} `json:"data"`                   // 消息数据
	Timestamp int64       `json:"timestamp" example:"1640000000000"` // 客户端时间戳
}

// ============ 客户端消息类型定义 ============

const (
	// ClientMsgTypeMessage 发送消息
	ClientMsgTypeMessage = "message"
	// ClientMsgTypePing 心跳
	ClientMsgTypePing = "ping"
	// ClientMsgTypeRead 已读回执
	ClientMsgTypeRead = "read"
	// ClientMsgTypeTyping 正在输入
	ClientMsgTypeTyping = "typing"
	// ClientMsgTypeRecall 撤回消息
	ClientMsgTypeRecall = "recall"
	// ClientMsgTypeJoinGroup 加入群组
	ClientMsgTypeJoinGroup = "join_group"
	// ClientMsgTypeLeaveGroup 离开群组
	ClientMsgTypeLeaveGroup = "leave_group"
)

// ClientSendMessageData 客户端发送消息数据
type ClientSendMessageData struct {
	ConversationID int64  `json:"conversationId" example:"1"` // 会话ID
	Content        string `json:"content" example:"Hello"`   // 消息内容
	ContentType    string `json:"contentType" example:"text"` // 内容类型: text, image, voice, video, file
	ClientMsgID    string `json:"clientMsgId,omitempty"`      // 客户端消息ID(用于去重)
	Extra          string `json:"extra,omitempty"`            // 扩展字段
}

// ClientReadData 已读回执数据
type ClientReadData struct {
	ConversationID int64   `json:"conversationId" example:"1"` // 会话ID
	MessageIDs     []int64 `json:"messageIds" example:"[1001,1002]"` // 已读消息ID列表
}

// ClientTypingData 正在输入数据
type ClientTypingData struct {
	ConversationID int64  `json:"conversationId" example:"1"` // 会话ID
	ContentType    string `json:"contentType" example:"text"` // 输入内容类型
}

// ClientRecallData 撤回消息数据
type ClientRecallData struct {
	MessageID int64 `json:"messageId" example:"1001"` // 消息ID
}

// ClientJoinGroupData 加入群组数据
type ClientJoinGroupData struct {
	GroupID int64 `json:"groupId" example:"1"` // 群组ID
}

// ClientLeaveGroupData 离开群组数据
type ClientLeaveGroupData struct {
	GroupID int64 `json:"groupId" example:"1"` // 群组ID
}

// ============ 服务器推送消息类型 ============

const (
	// ServerMsgTypeMessage 新消息
	ServerMsgTypeMessage = "message"
	// ServerMsgTypeAck 消息ACK
	ServerMsgTypeAck = "ack"
	// ServerMsgTypePong 心跳响应
	ServerMsgTypePong = "pong"
	// ServerMsgTypeRecall 消息撤回通知
	ServerMsgTypeRecall = "recall"
	// ServerMsgTypeRead 已读通知
	ServerMsgTypeRead = "read"
	// ServerMsgTypeTyping 输入状态通知
	ServerMsgTypeTyping = "typing"
	// ServerMsgTypeMemberJoin 成员加入通知
	ServerMsgTypeMemberJoin = "member_join"
	// ServerMsgTypeMemberLeave 成员离开通知
	ServerMsgTypeMemberLeave = "member_leave"
	// ServerMsgTypeGroupMuted 群成员被禁言通知
	ServerMsgTypeGroupMuted = "group_muted"
	// ServerMsgTypeError 错误通知
	ServerMsgTypeError = "error"
	// ServerMsgTypeOnlineStatus 在线状态通知
	ServerMsgTypeOnlineStatus = "online_status"
	// ServerMsgTypeSync 同步消息
	ServerMsgTypeSync = "sync"
)

// ServerMessage 服务器推送的消息
type ServerMessage struct {
	Type      string      `json:"type" example:"message"`                 // 消息类型
	Data      interface{} `json:"data"`                                   // 消息数据
	Timestamp int64       `json:"timestamp" example:"1640000000000"`     // 服务器时间戳
	MessageID string      `json:"messageId,omitempty"`                   // 服务器消息ID
}

// ServerAckData 消息ACK数据
type ServerAckData struct {
	ClientMsgID string `json:"clientMsgId"`              // 客户端消息ID
	ServerMsgID string `json:"serverMsgId" example:"msg_1640000000_123"` // 服务器消息ID
	Status     string `json:"status" example:"ok"`      // 状态: ok, error
	Error      string `json:"error,omitempty"`          // 错误信息
}

// ServerMessageData 服务器推送的消息数据
type ServerMessageData struct {
	ID             int64  `json:"id" example:"1001"`                       // 消息ID
	MsgID          string `json:"msgId" example:"msg_1640000000_123"`     // 全局唯一消息ID
	ConversationID int64  `json:"conversationId" example:"1"`              // 会话ID
	SenderID       int64  `json:"senderId" example:"1001"`               // 发送者ID
	SenderName     string `json:"senderName" example:"John Doe"`          // 发送者名称
	SenderAvatar   string `json:"senderAvatar" example:"https://..."`     // 发送者头像
	Content        string `json:"content" example:"Hello"`                // 消息内容
	ContentType    string `json:"contentType" example:"text"`             // 内容类型
	Extra          string `json:"extra,omitempty"`                       // 扩展字段
	CreatedAt      int64  `json:"createdAt" example:"1640000000000"`     // 创建时间
}

// ServerRecallData 撤回通知数据
type ServerRecallData struct {
	MessageID   int64 `json:"messageId" example:"1001"`      // 被撤回的消息ID
	RecallBy    int64 `json:"recallBy" example:"1001"`       // 撤回者ID
	RecallTime  int64 `json:"recallTime" example:"1640000000000"` // 撤回时间
}

// ServerReadData 已读通知数据
type ServerReadData struct {
	ConversationID int64   `json:"conversationId" example:"1"`   // 会话ID
	MessageIDs      []int64 `json:"messageIds" example:"[1001,1002]"` // 已读消息ID列表
	ReadBy          int64   `json:"readBy" example:"1002"`        // 已读者ID
	ReadTime        int64   `json:"readTime" example:"1640000000000"` // 已读时间
}

// ServerTypingData 输入状态通知数据
type ServerTypingData struct {
	ConversationID int64  `json:"conversationId" example:"1"` // 会话ID
	UserID          int64  `json:"userId" example:"1001"`      // 用户ID
	Username        string `json:"username" example:"john_doe"` // 用户名
	ContentType    string `json:"contentType" example:"text"` // 输入内容类型
}

// ServerMemberJoinData 成员加入通知数据
type ServerMemberJoinData struct {
	GroupID   int64  `json:"groupId" example:"1"`        // 群组ID
	UserID    int64  `json:"userId" example:"1002"`     // 用户ID
	Username  string `json:"username" example:"jane_doe"` // 用户名
	Nickname  string `json:"nickname" example:"Jane"`   // 昵称
	JoinedAt  int64  `json:"joinedAt" example:"1640000000000"` // 加入时间
	InvitedBy int64  `json:"invitedBy" example:"1001"`  // 邀请者
}

// ServerMemberLeaveData 成员离开通知数据
type ServerMemberLeaveData struct {
	GroupID   int64  `json:"groupId" example:"1"`       // 群组ID
	UserID    int64  `json:"userId" example:"1002"`      // 用户ID
	Username  string `json:"username" example:"jane_doe"` // 用户名
	LeftAt    int64  `json:"leftAt" example:"1640000000000"` // 离开时间
}

// ServerGroupMutedData 群成员被禁言通知数据
type ServerGroupMutedData struct {
	GroupID     int64  `json:"groupId" example:"1"`       // 群组ID
	UserID      int64  `json:"userId" example:"1002"`      // 被禁言用户ID
	MutedBy     int64  `json:"mutedBy" example:"1001"`    // 禁言者
	MuteMinutes int    `json:"muteMinutes" example:"10"`  // 禁言分钟数
	Reason      string `json:"reason" example:"Spamming"` // 禁言原因
	StartTime   int64  `json:"startTime" example:"1640000000000"` // 开始时间
	EndTime     int64  `json:"endTime" example:"1640000060000"`   // 结束时间
}

// ServerErrorData 错误通知数据
type ServerErrorData struct {
	Code    int    `json:"code" example:"400"`    // 错误码
	Message string `json:"message" example:"error message"` // 错误信息
}

// ServerOnlineStatusData 在线状态通知数据
type ServerOnlineStatusData struct {
	UserID     int64  `json:"userId" example:"1001"`      // 用户ID
	IsOnline   bool   `json:"isOnline" example:"true"`      // 是否在线
	DeviceType string `json:"deviceType" example:"web"`     // 设备类型: web, android, ios
	LastSeen   int64  `json:"lastSeen" example:"1640000000000"` // 最后在线时间
}

// ServerSyncData 同步消息数据
type ServerSyncData struct {
	ConversationID int64       `json:"conversationId" example:"1"` // 会话ID
	Messages       []MessageInfo `json:"messages"`              // 消息列表
	SyncType       string     `json:"syncType" example:"offline"` // 同步类型: offline, history
}

// MessageInfo 历史消息结构(用于同步)
type MessageInfo struct {
	ID             int64  `json:"id" example:"1001"`
	MsgID          string `json:"msgId" example:"msg_1640000000_123"`
	ConversationID int64  `json:"conversationId" example:"1"`
	SenderID       int64  `json:"senderId" example:"1001"`
	Content        string `json:"content" example:"Hello"`
	ContentType    string `json:"contentType" example:"text"`
	Extra          string `json:"extra,omitempty"`
	IsRecalled     bool   `json:"isRecalled" example:"false"`
	CreatedAt      int64  `json:"createdAt" example:"1640000000000"`
}

// ============ WebSocket 连接流程 ============
//
// 1. 客户端通过 HTTP GET 升级到 WebSocket
//    GET /ws?token=<JWT_TOKEN>
//    
// 2. 服务器验证 Token，成功后建立连接
//
// 3. 客户端定期发送 ping 心跳(建议 30 秒)
//    {"type": "ping", "timestamp": 1640000000000}
//
// 4. 服务器响应 pong
//    {"type": "pong", "timestamp": 1640000000000}
//
// 5. 发送消息流程:
//    - 客户端: {"type": "message", "data": {...}, "timestamp": 1640000000000}
//    - 服务器: {"type": "ack", "data": {"clientMsgId": "...", "serverMsgId": "..."}, "timestamp": 1640000000000}
//    - 服务器推送: {"type": "message", "data": {...}, "timestamp": 1640000000000}
//
// 6. 断开连接时，服务器清理连接资源
//

// ============ 错误码定义 ============

const (
	// WS 错误码
	WSCodeSuccess           = 0
	WSCodeInvalidToken      = 4001 // 无效Token
	WSCodeTokenExpired      = 4002 // Token过期
	WSCodeConversationNotFound = 2001 // 会话不存在
	WSCodeNoPermission      = 2002 // 无权限
	WSCodeMessageTooOld     = 3002 // 消息太旧无法撤回
	WSCodeGroupNotFound     = 4001 // 群组不存在
	WSCodeAlreadyInGroup    = 4003 // 已在群中
	WSCodeNotInGroup        = 4004 // 不在群中
	WSCodeMuted             = 4005 // 被禁言
	WSCodeServerError       = 5000 // 服务器错误
)
