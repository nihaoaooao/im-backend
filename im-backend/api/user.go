package api

import (
	"net/http"
	"strconv"

	"im-backend/middleware"
	"im-backend/models"
	"im-backend/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// UserHandler 用户处理器
type UserHandler struct {
	db *gorm.DB
}

// NewUserHandler 创建用户处理器
func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{
		db: userService.DB,
	}
}

// ListUsers 获取用户列表（管理员）
func (h *UserHandler) ListUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	var users []models.User
	err := h.db.WithContext(c.Request.Context()).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&users).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    50001,
			"message": "获取用户列表失败",
		})
		return
	}

	var total int64
	h.db.Model(&models.User{}).Count(&total)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"users": users,
			"total": total,
			"page":  page,
		},
	})
}

// GetUser 获取用户详情（管理员）
func (h *UserHandler) GetUser(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    40001,
			"message": "无效的用户ID",
		})
		return
	}

	var user models.User
	err = h.db.WithContext(c.Request.Context()).First(&user, userID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"code":    40401,
				"message": "用户不存在",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    50001,
			"message": "获取用户失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": user,
	})
}

// DeleteUser 删除用户（管理员）
func (h *UserHandler) DeleteUser(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    40001,
			"message": "无效的用户ID",
		})
		return
	}

	// 不能删除自己
	currentUserID := middleware.GetUserID(c)
	if userID == currentUserID {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    40002,
			"message": "不能删除自己的账号",
		})
		return
	}

	err = h.db.WithContext(c.Request.Context()).Delete(&models.User{}, userID).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    50001,
			"message": "删除用户失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "删除成功",
	})
}
