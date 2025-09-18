package handlers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	jwtutil "slackgo/internal/jwt"
	"slackgo/internal/model"
)

type AuthHandler struct {
	db  *gorm.DB
	jwt *jwtutil.Maker
}

func NewAuthHandler(db *gorm.DB, jm *jwtutil.Maker) *AuthHandler {
	return &AuthHandler{db: db, jwt: jm}
}

// Me godoc
// @Summary Me
// @Tags    auth
// @Produce json
// @Success 200 {object} map[string]string
// @Security Bearer
// @Router  /auth/me [get]
func (h *AuthHandler) Me(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"id": c.GetString("user_id")})
}

// POST /auth/bootstrap  {email?, display_name?}
type bootstrapReq struct {
	Email       *string `json:"email"`
	DisplayName *string `json:"display_name"`
}

func (h *AuthHandler) Bootstrap(c *gin.Context) {
	uid := c.GetString("user_id")
	if uid == "" {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	var body bootstrapReq
	if err := c.BindJSON(&body); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"detail": "bad body"})
		return
	}
	if err := h.db.Model(&model.User{}).Where("id = ?", uid).
		Updates(map[string]any{
			"email":        body.Email, // nilなら変更なしにしたければ条件分岐
			"display_name": body.DisplayName,
		}).Error; err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"detail": "update failed"})
		return
	}
	log.Printf("[bootstrap] uid=%s body=%+v", uid, body)
	c.Status(http.StatusNoContent)
}
