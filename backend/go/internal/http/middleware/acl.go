package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RequireWorkspaceMember(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		uid := c.GetString("user_id")
		wsID := c.Param("ws_id")
		if wsID == "" {
			wsID = c.Query("workspace_id")
		}
		if uid == "" || wsID == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"detail": "workspace_id required"})
			return
		}
		var n int64
		if err := db.Table("workspace_members").
			Where("user_id = ? AND workspace_id = ?", uid, wsID).
			Count(&n).Error; err != nil || n == 0 {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"detail": "forbidden"})
			return
		}
		c.Next()
	}
}

func RequireWorkspaceOwner(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		uid := c.GetString("user_id")
		wsID := c.Param("ws_id")
		if wsID == "" {
			wsID = c.Query("workspace_id")
		}
		if uid == "" || wsID == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"detail": "workspace_id required"})
			return
		}
		var n int64
		if err := db.Table("workspace_members").
			Where("user_id = ? AND workspace_id = ? AND role = 'owner'", uid, wsID).
			Count(&n).Error; err != nil || n == 0 {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"detail": "owner only"})
			return
		}
		c.Next()
	}
}

// 今は使っていない
func RequireChannelMember(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		uid := c.GetString("user_id")
		chID := c.Param("channel_id")
		if chID == "" {
			chID = c.Query("channel_id")
		}
		if uid == "" || chID == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"detail": "channel_id required"})
			return
		}
		var n int64
		if err := db.Table("channel_members").
			Where("user_id = ? AND channel_id = ?", uid, chID).
			Count(&n).Error; err != nil || n == 0 {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"detail": "forbidden"})
			return
		}
		c.Next()
	}
}

func RequireChannelOwner(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		uid := c.GetString("user_id")
		chID := c.Param("channel_id")
		if chID == "" {
			chID = c.Query("channel_id")
		}
		if uid == "" || chID == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"detail": "channel_id required"})
			return
		}
		var n int64
		if err := db.Table("channel_members").
			Where("user_id = ? AND channel_id = ? AND role = 'owner'", uid, chID).
			Count(&n).Error; err != nil || n == 0 {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"detail": "owner only"})
			return
		}
		c.Next()
	}
}
