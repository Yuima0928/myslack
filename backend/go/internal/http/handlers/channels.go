package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"slackgo/internal/model"
)

type ChannelsHandler struct {
	db *gorm.DB
}

func NewChannelsHandler(db *gorm.DB) *ChannelsHandler {
	return &ChannelsHandler{db: db}
}

type CreateChannelIn struct {
	Name      string `json:"name" binding:"required"`
	IsPrivate bool   `json:"is_private"`
}

// POST /workspaces/:ws_id/channels  （作成者=ownerで channel_members 追加）
func (h *ChannelsHandler) Create(c *gin.Context) {
	uid := c.GetString("user_id")
	wsID := c.Param("ws_id")
	var in CreateChannelIn
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"detail": err.Error()})
		return
	}

	ch := model.Channel{
		WorkspaceID: uuid.MustParse(wsID),
		Name:        in.Name,
		IsPrivate:   in.IsPrivate,
		CreatedBy:   uuidPtr(uuid.MustParse(uid)),
	}
	if err := h.db.Create(&ch).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": "create channel failed"})
		return
	}
	// 作成者=ownerとして channel_members へ
	cm := model.ChannelMember{
		UserID:    uuid.MustParse(uid),
		ChannelID: ch.ID,
		Role:      "owner",
	}
	_ = h.db.Create(&cm).Error

	c.JSON(http.StatusOK, gin.H{"id": ch.ID})
}

type AddMemberIn struct {
	UserID string `json:"user_id" binding:"required,uuid"`
	Role   string `json:"role" binding:"omitempty,oneof=owner member"`
}

// POST /channels/:channel_id/members  （owner限定）
func (h *ChannelsHandler) AddMember(c *gin.Context) {
	chID := c.Param("channel_id")
	var in AddMemberIn
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"detail": err.Error()})
		return
	}
	role := in.Role
	if role == "" {
		role = "member"
	}

	rec := model.ChannelMember{
		UserID:    uuid.MustParse(in.UserID),
		ChannelID: uuid.MustParse(chID),
		Role:      role,
	}
	// 既存なら何もしない（エラーにしない）
	if err := h.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&rec).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": "add member failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func uuidPtr(v uuid.UUID) *uuid.UUID { return &v }
