package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"slackgo/internal/model"
)

type ChannelsHandler struct {
	db *gorm.DB
}

func NewChannelsHandler(db *gorm.DB) *ChannelsHandler {
	return &ChannelsHandler{db: db}
}

type CreateChannelIn struct {
	WorkspaceID string `json:"workspace_id" binding:"required,uuid"`
	Name        string `json:"name" binding:"required"`
	IsPrivate   bool   `json:"is_private"`
}

func (h *ChannelsHandler) Create(c *gin.Context) {
	var in CreateChannelIn
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"detail": err.Error()})
		return
	}
	uid := c.GetString("user_id")

	ch := model.Channel{
		WorkspaceID: uuid.MustParse(in.WorkspaceID),
		Name:        in.Name,
		IsPrivate:   in.IsPrivate,
		CreatedBy:   uuidPtr(uuid.MustParse(uid)),
	}
	if err := h.db.Create(&ch).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": "create channel failed"})
		return
	}

	cm := model.ChannelMember{
		UserID:    uuid.MustParse(uid),
		ChannelID: ch.ID,
		Role:      "owner",
	}
	_ = h.db.Create(&cm).Error

	c.JSON(http.StatusOK, gin.H{"id": ch.ID})
}

func uuidPtr(v uuid.UUID) *uuid.UUID { return &v }
