package service

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"im-backend/model"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type GroupService struct {
	db    *gorm.DB
	redis *redis.Client
}

func NewGroupService(db *gorm.DB, redis *redis.Client) *GroupService {
	return &GroupService{db: db, redis: redis}
}

// CreateGroup 创建群组
func (s *GroupService) CreateGroup(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		Avatar      string `json:"avatar"`
		MemberIDs   []int64 `json:"memberIds"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 400, "msg": "参数错误"})
		return
	}

	// 创建群组会话
	conv := model.Conversation{
		Type:      "group",
		Name:      req.Name,
		CreatorID: userID.(int64),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.db.Create(&conv).Error; err != nil {
		c.JSON(500, gin.H{"code": 500, "msg": "创建失败"})
		return
	}

	// 创建群组成员（创建者为 owner）
	ownerMember := model.ConversationMember{
		ConversationID: conv.ID,
		UserID:         userID.(int64),
		Role:           "owner",
		JoinedAt:       time.Now(),
	}
	s.db.Create(&ownerMember)

	// 添加其他成员
	for _, memberID := range req.MemberIDs {
		member := model.ConversationMember{
			ConversationID: conv.ID,
			UserID:          memberID,
			Role:            "member",
			JoinedAt:        time.Now(),
		}
		s.db.Create(&member)
	}

	c.JSON(200, gin.H{
		"code": 0,
		"data": gin.H{
			"conversationId": conv.ID,
			"groupId":         conv.ID,
		},
	})
}

// GetGroupInfo 获取群信息
func (s *GroupService) GetGroupInfo(c *gin.Context) {
	userID, _ := c.Get("user_id")
	groupID, _ := c.Get("id")

	var conv model.Conversation
	if err := s.db.First(&conv, groupID).Error; err != nil {
		c.JSON(404, gin.H{"code": 404, "msg": "群组不存在"})
		return
	}

	// 检查是否是成员
	var member model.ConversationMember
	if err := s.db.Where("conversation_id = ? AND user_id = ?", groupID, userID).First(&member).Error; err != nil {
		c.JSON(403, gin.H{"code": 403, "msg": "不是群成员"})
		return
	}

	// 获取所有成员
	var members []model.ConversationMember
	s.db.Where("conversation_id = ?", groupID).Find(&members)

	var memberList []gin.H
	for _, m := range members {
		var user model.User
		s.db.First(&user, m.UserID)
		memberList = append(memberList, gin.H{
			"userId":   user.ID,
			"nickname": user.Nickname,
			"avatar":   user.Avatar,
			"role":     m.Role,
		})
	}

	c.JSON(200, gin.H{
		"code": 0,
		"data": gin.H{
			"id":          conv.ID,
			"name":        conv.Name,
			"type":        conv.Type,
			"memberCount": len(members),
			"members":     memberList,
		},
	})
}

// AddMember 添加群成员
func (s *GroupService) AddMember(c *gin.Context) {
	userID, _ := c.Get("user_id")
	groupID, _ := c.Get("id")

	var req struct {
		UserID int64 `json:"userId" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 400, "msg": "参数错误"})
		return
	}

	// 检查权限（只有 owner 和 admin 可以添加成员）
	var selfMember model.ConversationMember
	if err := s.db.Where("conversation_id = ? AND user_id = ?", groupID, userID).First(&selfMember).Error; err != nil {
		c.JSON(403, gin.H{"code": 403, "msg": "无权限"})
		return
	}

	if selfMember.Role != "owner" && selfMember.Role != "admin" {
		c.JSON(403, gin.H{"code": 403, "msg": "只有群主和管理员可以添加成员"})
		return
	}

	// 检查用户是否已在群中
	var existMember model.ConversationMember
	if err := s.db.Where("conversation_id = ? AND user_id = ?", groupID, req.UserID).First(&existMember).Error; err == nil {
		c.JSON(400, gin.H{"code": 400, "msg": "用户已在群中"})
		return
	}

	// 添加成员
	member := model.ConversationMember{
		ConversationID: groupID.(int64),
		UserID:          req.UserID,
		Role:            "member",
		JoinedAt:        time.Now(),
	}
	s.db.Create(&member)

	c.JSON(200, gin.H{"code": 0, "msg": "添加成功"})
}

// RemoveMember 移除群成员
func (s *GroupService) RemoveMember(c *gin.Context) {
	userID, _ := c.Get("user_id")
	groupID, _ := c.Get("id")
	uid := c.Param("uid")
	targetUserID, _ := strconv.ParseInt(uid, 10, 64)

	// 检查权限
	var selfMember model.ConversationMember
	if err := s.db.Where("conversation_id = ? AND user_id = ?", groupID, userID).First(&selfMember).Error; err != nil {
		c.JSON(403, gin.H{"code": 403, "msg": "无权限"})
		return
	}

	// 不能移除群主
	var targetMember model.ConversationMember
	if err := s.db.Where("conversation_id = ? AND user_id = ?", groupID, targetUserID).First(&targetMember).Error; err != nil {
		c.JSON(404, gin.H{"code": 404, "msg": "成员不存在"})
		return
	}

	if targetMember.Role == "owner" {
		c.JSON(403, gin.H{"code": 403, "msg": "不能移除群主"})
		return
	}

	// 群主可以移除任何人，管理员只能移除普通成员
	if selfMember.Role != "owner" && selfMember.Role == "member" {
		c.JSON(403, gin.H{"code": 403, "msg": "无权限"})
		return
	}
	if selfMember.Role == "admin" && targetMember.Role == "admin" {
		c.JSON(403, gin.H{"code": 403, "msg": "无权限"})
		return
	}

	// 移除成员
	s.db.Delete(&targetMember)

	c.JSON(200, gin.H{"code": 0, "msg": "移除成功"})
}

// MuteMember 禁言成员
func (s *GroupService) MuteMember(c *gin.Context) {
	userID, _ := c.Get("user_id")
	groupID, _ := c.Get("id")

	var req struct {
		UserID   int64 `json:"userId" binding:"required"`
		Duration int   `json:"duration"` // 禁言时长（秒），0 表示解除
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 400, "msg": "参数错误"})
		return
	}

	// 检查权限
	var selfMember model.ConversationMember
	if err := s.db.Where("conversation_id = ? AND user_id = ?", groupID, userID).First(&selfMember).Error; err != nil {
		c.JSON(403, gin.H{"code": 403, "msg": "无权限"})
		return
	}

	if selfMember.Role != "owner" && selfMember.Role != "admin" {
		c.JSON(403, gin.H{"code": 403, "msg": "只有群主和管理员可以禁言"})
		return
	}

	// 设置禁言
	var targetMember model.ConversationMember
	if err := s.db.Where("conversation_id = ? AND user_id = ?", groupID, req.UserID).First(&targetMember).Error; err != nil {
		c.JSON(404, gin.H{"code": 404, "msg": "成员不存在"})
		return
	}

	updates := map[string]interface{}{
		"is_muted": req.Duration > 0,
	}
	if req.Duration > 0 {
		updates["mute_until"] = time.Now().Add(time.Duration(req.Duration) * time.Second)
	} else {
		updates["mute_until"] = nil
	}

	s.db.Model(&targetMember).Updates(updates)

	c.JSON(200, gin.H{"code": 0, "msg": "设置成功"})
}

// GetOnlineMembers 获取群在线成员
func (s *GroupService) GetOnlineMembers(c *gin.Context) {
	groupID, _ := c.Get("id")

	// 从 Redis 获取群成员
	key := fmt.Sprintf("group:%d:members", groupID)
	members, err := s.redis.SMembers(context.Background(), key).Result()
	if err != nil {
		members = []string{}
	}

	c.JSON(200, gin.H{
		"code": 0,
		"data": gin.H{
			"onlineCount": len(members),
			"members":     members,
		},
	})
}
