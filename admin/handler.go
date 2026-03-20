package admin

import (
	"html/template"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"im-backend/model"
	"im-backend/service"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// [SECURITY] SQL注入防护：过滤搜索关键词中的特殊字符
func sanitizeSearchKeyword(keyword string) string {
	// 移除可能导致SQL注入的特殊字符
	// 允许：中文、字母、数字、下划线、空格、@、.
	re := regexp.MustCompile(`[;'\"\\%--\x00-\x1F]`)
	sanitized := re.ReplaceAllString(keyword, "")
	// 限制最大长度
	if len(sanitized) > 100 {
		sanitized = sanitized[:100]
	}
	return strings.TrimSpace(sanitized)
}

// [SECURITY] XSS防护：HTML转义函数
func escapeHTML(s string) string {
	var builder strings.Builder
	for _, r := range s {
		switch r {
		case '<':
			builder.WriteString("&lt;")
		case '>':
			builder.WriteString("&gt;")
		case '"':
			builder.WriteString("&quot;")
		case '\'':
			builder.WriteString("&#39;")
		case '&':
			builder.WriteString("&amp;")
		default:
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

// AdminHandler 管理后台处理器
type AdminHandler struct {
	db    *gorm.DB
	redis *redis.Client
}

// NewAdminHandler 创建管理后台处理器
func NewAdminHandler(db *gorm.DB, redis *redis.Client) *AdminHandler {
	return &AdminHandler{
		db:    db,
		redis: redis,
	}
}

// PageData 页面通用数据
type PageData struct {
	ActivePage string
	Stats      DashboardStats
	Logs       []model.AdminLog
}

// DashboardStats 控制台统计数据
type DashboardStats struct {
	TotalUsers      int64   `json:"total_users"`
	TotalGroups     int64   `json:"total_groups"`
	TotalMessages   int64   `json:"total_messages"`
	OnlineUsers     int64   `json:"online_users"`
	Dates           []string `json:"dates"`
	MessageCounts   []int64  `json:"message_counts"`
}

// Render 渲染模板
func (h *AdminHandler) Render(c *gin.Context, templateFile string, data interface{}) {
	// [SECURITY] 使用template/html而不是template/js来防止XSS
	tmpl := template.Must(template.ParseGlob("admin/templates/*.html"))
	tmpl = template.Must(tmpl.ParseGlob("admin/templates/**/*.html"))

	// 添加自定义函数 - [SECURITY] 移除不安全的json函数，使用html模板
	funcMap := template.FuncMap{
		"formatTime": func(t time.Time) string {
			return t.Format("2006-01-02 15:04:05")
		},
		"add": func(a, b int) int {
			return a + b
		},
		"sub": func(a, b int) int {
			return a - b
		},
		// [SECURITY] 安全的HTML转义函数
		"escape": func(s string) string {
			return escapeHTML(s)
		},
	}

	tmpl = tmpl.Funcs(funcMap)
	c.Header("Content-Type", "text/html; charset=utf-8")
	tmpl.ExecuteTemplate(c.Writer, templateFile, data)
}

// Dashboard 控制台首页
func (h *AdminHandler) Dashboard(c *gin.Context) {
	var totalUsers, totalGroups, totalMessages int64
	var onlineUsers int64

	h.db.Model(&model.User{}).Count(&totalUsers)
	h.db.Model(&model.Conversation{}).Where("type = ?", "group").Count(&totalGroups)
	h.db.Model(&model.Message{}).Count(&totalMessages)

	if h.redis != nil {
		onlineUsers, _ = h.redis.SCard(c.Request.Context(), "online:users").Result()
	}

	// 最近7天消息趋势
	var dates []string
	var messageCounts []int64
	for i := 6; i >= 0; i-- {
		date := time.Now().AddDate(0, 0, -i)
		dates = append(dates, date.Format("01-02"))
		
		var count int64
		startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
		endOfDay := startOfDay.AddDate(0, 0, 1)
		h.db.Model(&model.Message{}).Where("created_at BETWEEN ? AND ?", startOfDay, endOfDay).Count(&count)
		messageCounts = append(messageCounts, count)
	}

	// 最近操作日志
	var logs []model.AdminLog
	h.db.Order("created_at DESC").Limit(10).Find(&logs)

	data := PageData{
		ActivePage: "dashboard",
		Stats: DashboardStats{
			TotalUsers:    totalUsers,
			TotalGroups:   totalGroups,
			TotalMessages: totalMessages,
			OnlineUsers:   onlineUsers,
			Dates:         dates,
			MessageCounts: messageCounts,
		},
		Logs: logs,
	}

	h.Render(c, "dashboard.html", data)
}

// Users 用户管理列表
func (h *AdminHandler) Users(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize := 20
	// [SECURITY] SQL注入防护：过滤搜索关键词
	keyword := sanitizeSearchKeyword(c.Query("keyword"))
	role := c.Query("role")
	status := c.Query("status")

	if page < 1 {
		page = 1
	}

	var users []model.User
	var total int64

	query := h.db.Model(&model.User{})
	if keyword != "" {
		// [SECURITY] 使用过滤后的关键词
		escapedKeyword := "%" + escapeHTML(keyword) + "%"
		query = query.Where("username LIKE ? OR nickname LIKE ? OR email LIKE ?", escapedKeyword, escapedKeyword, escapedKeyword)
	}
	if role != "" {
		query = query.Where("role = ?", role)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	query.Count(&total)
	query.Offset((page - 1) * pageSize).Limit(pageSize).Order("created_at DESC").Find(&users)

	data := gin.H{
		"ActivePage": "users",
		"Users":      users,
		"Total":      total,
		"Page":       page,
		"TotalPages": (int(total) + pageSize - 1) / pageSize,
		"Keyword":    keyword,
		"Role":       role,
		"Status":     status,
	}

	h.Render(c, "users/list.html", data)
}

// BanUser 封禁用户
func (h *AdminHandler) BanUser(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	reason := c.PostForm("reason")

	h.db.Model(&model.User{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status": "banned",
	})

	// 记录日志
	adminID, _ := c.Get("user_id")
	username, _ := c.Get("username")
	h.db.Create(&model.AdminLog{
		AdminID:       adminID.(int64),
		AdminUsername: username.(string),
		Action:        "ban_user",
		TargetType:    "user",
		TargetID:      id,
		Details:       reason,
		IP:            c.ClientIP(),
	})

	c.Redirect(http.StatusFound, "/admin/users")
}

// UnbanUser 解封用户
func (h *AdminHandler) UnbanUser(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	h.db.Model(&model.User{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status": "active",
	})

	adminID, _ := c.Get("user_id")
	username, _ := c.Get("username")
	h.db.Create(&model.AdminLog{
		AdminID:       adminID.(int64),
		AdminUsername: username.(string),
		Action:        "unban_user",
		TargetType:    "user",
		TargetID:      id,
		Details:       "解封用户",
		IP:            c.ClientIP(),
	})

	c.Redirect(http.StatusFound, "/admin/users")
}

// DeleteUser 删除用户
func (h *AdminHandler) DeleteUser(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	h.db.Delete(&model.User{}, id)

	adminID, _ := c.Get("user_id")
	username, _ := c.Get("username")
	h.db.Create(&model.AdminLog{
		AdminID:       adminID.(int64),
		AdminUsername: username.(string),
		Action:        "delete_user",
		TargetType:    "user",
		TargetID:      id,
		Details:       "删除用户",
		IP:            c.ClientIP(),
	})

	c.Redirect(http.StatusFound, "/admin/users")
}

// Groups 群组管理列表
func (h *AdminHandler) Groups(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize := 20
	// [SECURITY] SQL注入防护：过滤搜索关键词
	groupName := sanitizeSearchKeyword(c.Query("name"))
	groupType := c.Query("type")

	if page < 1 {
		page = 1
	}

	var groups []model.Conversation
	var total int64

	query := h.db.Model(&model.Conversation{})
	if groupName != "" {
		// [SECURITY] 使用过滤后的关键词
		escapedName := "%" + escapeHTML(groupName) + "%"
		query = query.Where("name LIKE ?", escapedName)
	}
	if groupType != "" {
		query = query.Where("type = ?", groupType)
	} else {
		query = query.Where("type = ?", "group")
	}

	query.Count(&total)
	query.Offset((page - 1) * pageSize).Limit(pageSize).Order("created_at DESC").Find(&groups)

	// 填充群主和成员数
	type GroupInfo struct {
		model.Conversation
		OwnerName    string `json:"owner_name"`
		MemberCount  int64  `json:"member_count"`
	}

	var groupInfos []GroupInfo
	for _, g := range groups {
		var owner model.User
		h.db.First(&owner, g.CreatorID)
		var memberCount int64
		h.db.Model(&model.ConversationMember{}).Where("conversation_id = ?", g.ID).Count(&memberCount)
		groupInfos = append(groupInfos, GroupInfo{
			Conversation: g,
			OwnerName:    owner.Username,
			MemberCount:  memberCount,
		})
	}

	data := gin.H{
		"ActivePage": "groups",
		"Groups":     groupInfos,
		"Total":      total,
		"Page":       page,
		"TotalPages": (int(total) + pageSize - 1) / pageSize,
		"GroupName":  groupName,
		"GroupType":  groupType,
	}

	h.Render(c, "groups/list.html", data)
}

// GetGroup 获取群组详情
func (h *AdminHandler) GetGroup(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	var group model.Conversation
	if err := h.db.First(&group, id).Error; err != nil {
		c.JSON(404, gin.H{"code": 404, "msg": "群组不存在"})
		return
	}

	var owner model.User
	h.db.First(&owner, group.CreatorID)

	var members []model.ConversationMember
	h.db.Where("conversation_id = ?", id).Find(&members)

	type MemberWithUser struct {
		model.ConversationMember
		Username string `json:"username"`
		Nickname string `json:"nickname"`
	}

	var membersWithUser []MemberWithUser
	for _, m := range members {
		var user model.User
		h.db.First(&user, m.UserID)
		membersWithUser = append(membersWithUser, MemberWithUser{
			ConversationMember: m,
			Username:           user.Username,
			Nickname:           user.Nickname,
		})
	}

	c.JSON(200, gin.H{
		"code": 0,
		"data": gin.H{
			"id":           group.ID,
			"name":         group.Name,
			"type":         group.Type,
			"creator_id":   group.CreatorID,
			"creator_name": owner.Username,
			"member_count": len(members),
			"members":      membersWithUser,
			"created_at":  group.CreatedAt,
		},
	})
}

// DismissGroup 解散群组
func (h *AdminHandler) DismissGroup(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	reason := c.PostForm("reason")

	var group model.Conversation
	if err := h.db.First(&group, id).Error; err != nil {
		c.Redirect(http.StatusFound, "/admin/groups")
		return
	}

	// 删除群组成员
	h.db.Where("conversation_id = ?", id).Delete(&model.ConversationMember{})
	// 删除群组
	h.db.Delete(&group)

	adminID, _ := c.Get("user_id")
	username, _ := c.Get("username")
	h.db.Create(&model.AdminLog{
		AdminID:       adminID.(int64),
		AdminUsername: username.(string),
		Action:        "dismiss_group",
		TargetType:    "group",
		TargetID:      id,
		Details:       reason,
		IP:            c.ClientIP(),
	})

	c.Redirect(http.StatusFound, "/admin/groups")
}

// Messages 消息管理列表
func (h *AdminHandler) Messages(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize := 20
	userID, _ := strconv.ParseInt(c.Query("user_id"), 10, 64)
	conversationID, _ := strconv.ParseInt(c.Query("conversation_id"), 10, 64)
	contentType := c.Query("content_type")

	if page < 1 {
		page = 1
	}

	var messages []model.Message
	var total int64

	query := h.db.Model(&model.Message{})
	if userID > 0 {
		query = query.Where("sender_id = ?", userID)
	}
	if conversationID > 0 {
		query = query.Where("conversation_id = ?", conversationID)
	}
	if contentType != "" {
		query = query.Where("content_type = ?", contentType)
	}

	query.Count(&total)
	query.Offset((page - 1) * pageSize).Limit(pageSize).Order("created_at DESC").Find(&messages)

	// 填充发送者用户名
	type MessageWithSender struct {
		model.Message
		SenderUsername string `json:"sender_username"`
	}

	var messagesWithSender []MessageWithSender
	for _, m := range messages {
		var sender model.User
		h.db.First(&sender, m.SenderID)
		messagesWithSender = append(messagesWithSender, MessageWithSender{
			Message:        m,
			SenderUsername: sender.Username,
		})
	}

	data := gin.H{
		"ActivePage":    "messages",
		"Messages":      messagesWithSender,
		"Total":         total,
		"Page":          page,
		"TotalPages":    (int(total) + pageSize - 1) / pageSize,
		"UserID":        userID,
		"ConversationID": conversationID,
		"ContentType":   contentType,
	}

	h.Render(c, "messages/list.html", data)
}

// RevokeMessage 撤回消息
func (h *AdminHandler) RevokeMessage(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	reason := c.PostForm("reason")

	h.db.Model(&model.Message{}).Where("id = ?", id).Updates(map[string]interface{}{
		"is_recalled": true,
		"content":     "该消息已被撤回",
	})

	adminID, _ := c.Get("user_id")
	username, _ := c.Get("username")
	h.db.Create(&model.AdminLog{
		AdminID:       adminID.(int64),
		AdminUsername: username.(string),
		Action:        "revoke_message",
		TargetType:    "message",
		TargetID:      id,
		Details:       reason,
		IP:            c.ClientIP(),
	})

	c.Redirect(http.StatusFound, "/admin/messages")
}

// Stats 数据统计页面
func (h *AdminHandler) Stats(c *gin.Context) {
	startDate := c.DefaultQuery("start_date", time.Now().AddDate(0, 0, -30).Format("2006-01-02"))
	endDate := c.DefaultQuery("end_date", time.Now().Format("2006-01-02"))

	start, _ := time.Parse("2006-01-02", startDate)
	end, _ := time.Parse("2006-01-02", endDate)
	end = end.AddDate(0, 0, 1)

	// 用户统计
	var totalUsers, newUsers, activeUsers int64
	h.db.Model(&model.User{}).Count(&totalUsers)
	h.db.Model(&model.User{}).Where("created_at BETWEEN ? AND ?", start, end).Count(&newUsers)
	h.db.Model(&model.User{}).Where("last_login_at > ?", time.Now().AddDate(0, 0, -30)).Count(&activeUsers)

	// 每日新增用户
	var dates []string
	var dailyNewUsers []int64
	for d := start; d.Before(end); d = d.AddDate(0, 0, 1) {
		dates = append(dates, d.Format("01-02"))
		dNext := d.AddDate(0, 0, 1)
		var count int64
		h.db.Model(&model.User{}).Where("created_at BETWEEN ? AND ?", d, dNext).Count(&count)
		dailyNewUsers = append(dailyNewUsers, count)
	}

	// 消息统计
	var totalMessages, periodMessages int64
	h.db.Model(&model.Message{}).Count(&totalMessages)
	h.db.Model(&model.Message{}).Where("created_at BETWEEN ? AND ?", start, end).Count(&periodMessages)

	// 每日消息数
	var messageDates []string
	var dailyMessages []int64
	for d := start; d.Before(end); d = d.AddDate(0, 0, 1) {
		messageDates = append(messageDates, d.Format("01-02"))
		dNext := d.AddDate(0, 0, 1)
		var count int64
		h.db.Model(&model.Message{}).Where("created_at BETWEEN ? AND ?", d, dNext).Count(&count)
		dailyMessages = append(dailyMessages, count)
	}

	// 消息类型分布
	types := []string{"text", "image", "voice", "video", "file"}
	var typeCounts []int64
	for _, t := range types {
		var count int64
		h.db.Model(&model.Message{}).Where("content_type = ?", t).Count(&count)
		typeCounts = append(typeCounts, count)
	}

	data := gin.H{
		"ActivePage": "stats",
		"StartDate":  startDate,
		"EndDate":    endDate,
		"UserStats": gin.H{
			"TotalUsers":     totalUsers,
			"NewUsers":       newUsers,
			"ActiveUsers":    activeUsers,
			"Dates":          dates,
			"DailyNewUsers":  dailyNewUsers,
		},
		"MessageStats": gin.H{
			"TotalMessages":   totalMessages,
			"PeriodMessages":  periodMessages,
			"Types":           types,
			"TypeCounts":      typeCounts,
			"Dates":           messageDates,
			"DailyMessages":   dailyMessages,
		},
	}

	h.Render(c, "stats/users.html", data)
}

// Settings 系统设置页面
func (h *AdminHandler) Settings(c *gin.Context) {
	// 这里可以从数据库或配置文件加载设置
	settings := gin.H{
		"SystemName":       "IM 即时通讯系统",
		"AllowRegister":    true,
		"DefaultRole":      "user",
		"PasswordMinLength": 6,
		"TokenExpireHours": 168,
		"MaxLoginFailures": 5,
		"MaxMessageLength": 5000,
		"MaxFileSize":      10,
		"RevokeTimeLimit":  5,
		"SensitiveWords":   []string{"敏感词1", "敏感词2"},
	}

	data := gin.H{
		"ActivePage": "settings",
		"Settings":   settings,
	}

	h.Render(c, "settings/system.html", data)
}

// Login 登录页面
func (h *AdminHandler) Login(c *gin.Context) {
	h.Render(c, "login.html", nil)
}

// Logout 退出登录
func (h *AdminHandler) Logout(c *gin.Context) {
	c.SetCookie("token", "", -1, "/", "", false, false)
	c.Redirect(http.StatusFound, "/admin/login")
}

// 辅助函数
func toJSON(v interface{}) string {
	// 简化的 JSON 转换
	return service.ToJSON(v)
}
