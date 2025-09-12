package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"slackgo/internal/jwt"
	"slackgo/internal/model"
)

type AuthHandler struct {
	db  *gorm.DB
	jwt *jwtutil.Maker
}

func NewAuthHandler(db *gorm.DB, jm *jwtutil.Maker) *AuthHandler {
	return &AuthHandler{db: db, jwt: jm}
}

type SignUpIn struct {
	Email       string `json:"email" binding:"required,email"`
	Password    string `json:"password" binding:"required,min=4"`
	DisplayName string `json:"display_name"`
}
type TokenOut struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

func (h *AuthHandler) SignUp(c *gin.Context) {
	var in SignUpIn
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"detail": err.Error()})
		return
	}
	var exist model.User
	if err := h.db.Where("email = ?", in.Email).First(&exist).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"detail": "Email already registered"})
		return
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	u := model.User{Email: in.Email, PasswordHash: string(hash)}
	if in.DisplayName != "" {
		u.DisplayName = &in.DisplayName
	}
	if err := h.db.Create(&u).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": "create user failed"})
		return
	}
	t, _ := h.jwt.Create(u.ID.String())
	c.JSON(http.StatusOK, TokenOut{AccessToken: t, TokenType: "bearer"})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var in SignUpIn
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"detail": err.Error()})
		return
	}
	var u model.User
	if err := h.db.Where("email = ?", in.Email).First(&u).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"detail": "Invalid credentials"})
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(in.Password)) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"detail": "Invalid credentials"})
		return
	}
	t, _ := h.jwt.Create(u.ID.String())
	c.JSON(http.StatusOK, TokenOut{AccessToken: t, TokenType: "bearer"})
}

func (h *AuthHandler) Me(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"id": c.GetString("user_id")})
}
