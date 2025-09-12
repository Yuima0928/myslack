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
	WorkspaceID uuid.UUID `json:"workspace_id" binding:"required"`
	ChannelID   uuid.UUID `json:"channel_id" binding:"required"`
	Text        string    `json:"text" binding:"required,min=1"`
}
type MsgOut struct {
	ID          uuid.UUID `json:"id"`
	WorkspaceID uuid.UUID `json:"workspace_id"`
	ChannelID   uuid.UUID `json:"channel_id"`
	UserID      uuid.UUID `json:"user_id"`
	Text        string    `json:"text"`
}

func (h *MessagesHandler) Create(c *gin.Context) {
	var in MsgCreateIn
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"detail": err.Error()})
		return
	}
	uid := uuid.MustParse(c.GetString("user_id"))
	text := in.Text
	m := model.Message{
		WorkspaceID: in.WorkspaceID,
		ChannelID:   in.ChannelID,
		UserID:      &uid,
		Text:        &text,
	}
	if err := h.db.Create(&m).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "insert failed"})
		return
	}
	out := MsgOut{ID: m.ID, WorkspaceID: m.WorkspaceID, ChannelID: m.ChannelID, UserID: *m.UserID, Text: *m.Text}
	c.JSON(http.StatusOK, out)

	ev := map[string]any{"type": "message_created", "message": out}
	b, _ := json.Marshal(ev)
	h.hub.Broadcast(m.ChannelID.String(), b)
}

func (h *MessagesHandler) List(c *gin.Context) {
	chID := c.Query("channel_id")
	if chID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "channel_id is required"})
		return
	}
	var rows []model.Message
	if err := h.db.Where("channel_id = ?", chID).
		Order("created_at DESC").Limit(50).Offset(0).Find(&rows).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "list failed"})
		return
	}
	out := make([]MsgOut, 0, len(rows))
	for _, r := range rows {
		out = append(out, MsgOut{
			ID: r.ID, WorkspaceID: r.WorkspaceID, ChannelID: r.ChannelID, UserID: *r.UserID, Text: deref(r.Text),
		})
	}
	c.JSON(http.StatusOK, out)
}

func deref(s *string) string { if s==nil { return ""}; return *s }
