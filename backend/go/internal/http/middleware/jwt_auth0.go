// internal/http/middleware/jwt_auth0.go
package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"slackgo/internal/model"

	keyfunc "github.com/MicahParks/keyfunc/v3" // ← ルートを import
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

type Auth0Config struct {
	Domain   string // e.g. your-tenant.us.auth0.com
	Audience string // e.g. https://api.myslack.local
}

func JWTAuth0(db *gorm.DB, cfg Auth0Config) gin.HandlerFunc {
	jwksURL := "https://" + cfg.Domain + "/.well-known/jwks.json"

	// v3: NewDefaultCtx で自動リフレッシュ付きの Keyfunc を作成
	ctx := context.Background() // サーバのライフサイクルに合わせた ctx を使ってOK（終了時に cancel でも可）
	kf, err := keyfunc.NewDefaultCtx(ctx, []string{jwksURL})
	if err != nil {
		panic(err)
	}

	return func(c *gin.Context) {
		authz := c.GetHeader("Authorization")
		if !strings.HasPrefix(authz, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"detail": "Missing bearer"})
			return
		}
		raw := strings.TrimPrefix(authz, "Bearer ")

		// jwt.Parse のオプションで aud/iss/alg を厳密に検証
		token, err := jwt.Parse(raw, kf.Keyfunc,
			jwt.WithAudience(cfg.Audience),
			jwt.WithIssuer("https://"+cfg.Domain+"/"), // 末尾スラッシュ必須
			jwt.WithValidMethods([]string{"RS256"}),
			jwt.WithLeeway(30*time.Second),
		)
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"detail": "invalid token"})
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"detail": "invalid claims"})
			return
		}

		sub, _ := claims["sub"].(string)
		email, _ := claims["email"].(string) // Access Token だと無いことが多い点は留意
		name, _ := claims["name"].(string)
		if sub == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"detail": "missing sub"})
			return
		}

		// --- JIT プロビジョニング ---
		var u model.User
		if err := db.Where("external_id = ?", sub).First(&u).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				u = model.User{
					ExternalID:  &sub,
					Email:       strPtrOrNil(email),
					DisplayName: strPtrOrNil(name),
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
