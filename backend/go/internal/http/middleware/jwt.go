package middleware

import (
    "net/http"
    "strings"

    "github.com/gin-gonic/gin"

    jwtutil "slackgo/internal/jwt"
)

func JWT(m *jwtutil.Maker) gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"detail": "Missing bearer"})
			return
		}
		raw := strings.TrimPrefix(auth, "Bearer ")
		claims, err := m.Parse(raw)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"detail": "Invalid token"})
			return
		}
		sub, _ := claims["sub"].(string)
		if sub == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"detail": "Invalid sub"})
			return
		}
		c.Set("user_id", sub)
		c.Next()
	}
}
