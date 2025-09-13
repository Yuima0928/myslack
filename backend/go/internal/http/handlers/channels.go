// backend/go/internal/http/handlers/channels.go
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
	// チャンネル名
	Name string `json:"name" binding:"required" example:"general"`
	// プライベートかどうか
	IsPrivate bool `json:"is_private" example:"false"`
}

// Create godoc
// @Summary  Create channel
// @Tags     channels
// @Accept   json
// @Produce  json
// @Param    ws_id path string true "Workspace ID (UUID)"
// @Param    body  body  CreateChannelIn true "channel payload"
// @Success  200   {object} map[string]string "id: UUID string"
// @Failure  400   {object} map[string]string
// @Failure  401   {object} map[string]string
// @Failure  422   {object} map[string]string
// @Security Bearer
// @Router   /workspaces/{ws_id}/channels [post]
// POST /workspaces/:ws_id/channels  （作成者=ownerで channel_members 追加）
func (h *ChannelsHandler) Create(c *gin.Context) {
	uid := c.GetString("user_id")
	wsID := c.Param("ws_id")
	if uid == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"detail": "unauthorized"})
		return
	}
	if wsID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "ws_id required"})
		return
	}

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

	c.JSON(http.StatusOK, gin.H{"id": ch.ID.String()})
}

type AddMemberIn struct {
	// 追加するユーザのUUID
	UserID string `json:"user_id" binding:"required,uuid" example:"6d4c2f52-1f1c-4e7d-92a2-4b2d4a3d9a10"`
	// 役割（未指定は member）
	Role string `json:"role" binding:"omitempty,oneof=owner member" example:"member"`
}

// AddMember godoc
// @Summary  Add member to channel
// @Tags     channels
// @Accept   json
// @Produce  json
// @Param    channel_id path string true "Channel ID (UUID)"
// @Param    body       body  AddMemberIn true "member payload"
// @Success  200 {object} map[string]bool "ok: true"
// @Failure  400 {object} map[string]string
// @Failure  401 {object} map[string]string
// @Failure  422 {object} map[string]string
// @Security Bearer
// @Router   /channels/{channel_id}/members [post]
// POST /channels/:channel_id/members  （owner限定）
func (h *ChannelsHandler) AddMember(c *gin.Context) {
	chID := c.Param("channel_id")
	if chID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "channel_id required"})
		return
	}

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
