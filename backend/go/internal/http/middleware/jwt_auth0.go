package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"slackgo/internal/auth"
	"slackgo/internal/model"
)

// JWTWithVerifier は auth.Verifier を使って JWT を検証し、JITでユーザーを作成します。
// 成功時は c.Set("user_id", "<uuid-string>") / c.Set("user_email", *string|nil) を設定します。
func JWTAuth0(db *gorm.DB, v *auth.Verifier) gin.HandlerFunc {
	return func(c *gin.Context) {
		authz := c.GetHeader("Authorization")
		if !strings.HasPrefix(authz, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"detail": "Missing bearer"})
			return
		}
		raw := strings.TrimPrefix(authz, "Bearer ")

		// ← ここで auth.Verifier を利用（aud/iss/alg 検証含む）
		claims, err := v.Verify(raw)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"detail": "invalid token"})
			return
		}

		// --- JIT プロビジョニング（既存と同じ挙動） ---
		var u model.User
		if err := db.Where("external_id = ?", claims.Sub).First(&u).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				u = model.User{
					ExternalID:  strPtrOrNil(claims.Sub),
					Email:       strPtrOrNil(claims.Email),
					DisplayName: strPtrOrNil(claims.Name),
				}
				if err := db.Create(&u).Error; err != nil {
					c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"detail": "user upsert failed"})
					return
				}
			} else {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"detail": "user lookup failed"})
				return
			}
		}

		c.Set("user_id", u.ID.String())
		c.Set("user_email", u.Email)
		c.Next()
	}
}

func strPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
