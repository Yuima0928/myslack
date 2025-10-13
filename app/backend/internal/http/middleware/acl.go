package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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

type channelInfo struct {
	WorkspaceID uuid.UUID
	IsPrivate   bool
}

// 読み取り可：private→channel member 必須、public→workspace member でOK
func RequireChannelReadable(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString("user_id")
		chID := c.Param("channel_id")
		if userID == "" || chID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"detail": "unauthorized"})
			return
		}

		var ch channelInfo
		if err := db.Raw(`
			SELECT workspace_id, is_private
			FROM channels
			WHERE id = ?`, chID).Scan(&ch).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"detail": "channel not found"})
			return
		}

		// private は channel_members が必要
		if ch.IsPrivate {
			var ok int
			if err := db.Raw(`
				SELECT 1 FROM channel_members
				WHERE channel_id = ? AND user_id = ? LIMIT 1`, chID, userID).Scan(&ok).Error; err == nil && ok == 1 {
				c.Next()
				return
			}
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"detail": "not a channel member"})
			return
		}

		// public は workspace_members でOK
		var ok int
		if err := db.Raw(`
			SELECT 1 FROM workspace_members
			WHERE workspace_id = ? AND user_id = ? LIMIT 1`, ch.WorkspaceID, userID).Scan(&ok).Error; err == nil && ok == 1 {
			c.Next()
			return
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"detail": "not a workspace member"})
	}
}

// 書き込み可：方針に合わせて調整（ここでは public→WS member、private→channel member）
// 書き込み可 : public, privateともにchannel memberの必要がある
func RequireChannelWritable(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString("user_id")
		chID := c.Param("channel_id")
		if userID == "" || chID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"detail": "unauthorized"})
			return
		}

		var ch channelInfo
		if err := db.Raw(`
			SELECT workspace_id, is_private
			FROM channels
			WHERE id = ?`, chID).Scan(&ch).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"detail": "channel not found"})
			return
		}

		var ok int
		if err := db.Raw(`
				SELECT 1 FROM channel_members
				WHERE channel_id = ? AND user_id = ? LIMIT 1`, chID, userID).Scan(&ok).Error; err == nil && ok == 1 {
			c.Next()
			return
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"detail": "not a channel member"})
	}
}
