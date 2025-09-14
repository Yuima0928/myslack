// internal/http/middleware/channel_access.go
package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

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
func RequireChannelWritable(db *gorm.DB) gin.HandlerFunc {
	// もし「public でも書き込みは channel member だけ」にしたいなら、
	// public 分岐でも channel_members をチェックするように変えてください。
	return RequireChannelReadable(db)
}
