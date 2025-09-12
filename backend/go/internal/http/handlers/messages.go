package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"slackgo/internal/model"
	"slackgo/internal/ws"
)

type MessagesHandler struct {
	db  *gorm.DB
	hub *ws.Hub
}

func NewMessagesHandler(db *gorm.DB, hub *ws.Hub) *MessagesHandler {
	return &MessagesHandler{db: db, hub: hub}
}

type MsgCreateIn struct {
	Text string `json:"text" binding:"required,min=1"`
}

type MsgOut struct {
	ID          uuid.UUID `json:"id"`
	WorkspaceID uuid.UUID `json:"workspace_id"`
	ChannelID   uuid.UUID `json:"channel_id"`
	UserID      uuid.UUID `json:"user_id"`
	Text        string    `json:"text"`
}

// POST /channels/:channel_id/messages
func (h *MessagesHandler) Create(c *gin.Context) {
	var in MsgCreateIn
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"detail": err.Error()})
		return
	}

	uidStr := c.GetString("user_id")
	chIDStr := c.Param("channel_id")
	if uidStr == "" || chIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "channel_id required"})
		return
	}

	// Channel を取得して WorkspaceID を解決
	var ch model.Channel
	if err := h.db.First(&ch, "id = ?", chIDStr).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"detail": "channel not found"})
		return
	}

	uid := uuid.MustParse(uidStr)
	chID := uuid.MustParse(chIDStr)
	text := in.Text

	msg := model.Message{
		WorkspaceID: ch.WorkspaceID,
		ChannelID:   chID,
		UserID:      &uid,
		Text:        &text,
	}

	if err := h.db.Create(&msg).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": "create message failed"})
		return
	}

	out := MsgOut{
		ID:          msg.ID,
		WorkspaceID: msg.WorkspaceID,
		ChannelID:   msg.ChannelID,
		UserID:      *msg.UserID,
		Text:        text,
	}
	c.JSON(http.StatusOK, out)

	// WSへ通知（任意）
	ev := map[string]any{"type": "message_created", "message": out}
	if b, err := json.Marshal(ev); err == nil {
		h.hub.Broadcast(chID.String(), b)
	}
}

// GET /channels/:channel_id/messages
func (h *MessagesHandler) List(c *gin.Context) {
	chIDStr := c.Param("channel_id")
	if chIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "channel_id required"})
		return
	}

	var rows []model.Message
	if err := h.db.
		Where("channel_id = ?", chIDStr).
		Order("created_at ASC").
		Limit(100).
		Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": "query failed"})
		return
	}

	out := make([]MsgOut, 0, len(rows))
	for _, r := range rows {
		out = append(out, MsgOut{
			ID:          r.ID,
			WorkspaceID: r.WorkspaceID,
			ChannelID:   r.ChannelID,
			UserID:      derefUUID(r.UserID),
			Text:        derefStr(r.Text),
		})
	}
	c.JSON(http.StatusOK, out)
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func derefUUID(u *uuid.UUID) uuid.UUID {
	if u == nil {
		return uuid.Nil
	}
	return *u
}
