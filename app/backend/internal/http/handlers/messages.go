package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

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
	Text     string  `json:"text" binding:"required,min=1"`
	ParentID *string `json:"parent_id,omitempty"` // 追加: 返信先（UUID文字列）
}

type MsgOut struct {
	ID               uuid.UUID  `json:"id"`
	WorkspaceID      uuid.UUID  `json:"workspace_id"`
	ChannelID        uuid.UUID  `json:"channel_id"`
	UserID           uuid.UUID  `json:"user_id"`
	UserDisplayName  *string    `json:"user_display_name,omitempty"`
	UserAvatarFileID *uuid.UUID `json:"user_avatar_file_id,omitempty"`
	Text             string     `json:"text"`
	ParentID         *uuid.UUID `json:"parent_id,omitempty"`
	ThreadRootID     *uuid.UUID `json:"thread_root_id,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
}

// Create message godoc
// @Summary  Create message (supports thread replies)
// @Tags     messages
// @Accept   json
// @Produce  json
// @Param    channel_id path string true  "Channel ID (UUID)"
// @Param    body       body  MsgCreateIn true "message"
// @Success  200 {object} MsgOut
// @Failure  400 {object} map[string]string
// @Failure  401 {object} map[string]string
// @Failure  404 {object} map[string]string
// @Security Bearer
// @Router   /channels/{channel_id}/messages [post]
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

	// チャンネル存在 & WS解決
	var ch model.Channel
	if err := h.db.First(&ch, "id = ?", chIDStr).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"detail": "channel not found"})
		return
	}

	uid := uuid.MustParse(uidStr)
	chID := uuid.MustParse(chIDStr)
	text := in.Text

	var parentID *uuid.UUID
	var rootID *uuid.UUID

	// 返信の場合、親を検証
	if in.ParentID != nil && *in.ParentID != "" {
		pid, err := uuid.Parse(*in.ParentID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"detail": "invalid parent_id"})
			return
		}
		var parent model.Message
		if err := h.db.First(&parent, "id = ?", pid).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"detail": "parent message not found"})
			return
		}
		// 同一チャンネルであることを保証
		if parent.ChannelID != chID {
			c.JSON(http.StatusBadRequest, gin.H{"detail": "parent message channel mismatch"})
			return
		}
		parentID = &pid

		// ThreadRootID: 親に既にルートがあればそれを継承、なければ親をルートとする
		if parent.ThreadRootID != nil {
			rootID = parent.ThreadRootID
		} else {
			rootID = &parent.ID
		}
	}

	msg := model.Message{
		WorkspaceID:  ch.WorkspaceID,
		ChannelID:    chID,
		UserID:       &uid,
		Text:         &text,
		ParentID:     parentID,
		ThreadRootID: rootID,
	}

	if err := h.db.Create(&msg).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": "create message failed"})
		return
	}

	var disp *string
	var avatarID *uuid.UUID

	_ = h.db.Table("users").
		Select("display_name, avatar_file_id").
		Where("id = ?", uid).
		Row().
		Scan(&disp, &avatarID)

	out := MsgOut{
		ID:               msg.ID,
		WorkspaceID:      msg.WorkspaceID,
		ChannelID:        msg.ChannelID,
		UserID:           *msg.UserID,
		UserDisplayName:  disp,
		UserAvatarFileID: avatarID,
		Text:             text,
		ParentID:         msg.ParentID,
		ThreadRootID:     msg.ThreadRootID,
		CreatedAt:        msg.CreatedAt,
	}
	c.JSON(http.StatusOK, out)

	// WSイベント
	ev := map[string]any{"type": "message_created", "message": out}
	if b, err := json.Marshal(ev); err == nil {
		h.hub.Broadcast(chID.String(), b)
	}
}

// List messages godoc
// @Summary  List messages (channel timeline or thread replies)
// @Tags     messages
// @Produce  json
// @Param    channel_id     path  string true  "Channel ID (UUID)"
// @Param    thread_root_id query string false "If set, returns replies under the thread root"
// @Param    root_only      query bool   false "If true, returns only top-level messages"
// @Param    limit          query int    false "limit"
// @Param    offset         query int    false "offset"
// @Success  200 {array} MsgOut
// @Failure  400 {object} map[string]string
// @Security Bearer
// @Router   /channels/{channel_id}/messages [get]
func (h *MessagesHandler) List(c *gin.Context) {
	chIDStr := c.Param("channel_id")
	if chIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "channel_id required"})
		return
	}
	chID := uuid.MustParse(chIDStr)

	threadRootIDStr := c.Query("thread_root_id")
	rootOnly := c.Query("root_only") == "true"

	limit := 100
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 500 {
			limit = n
		}
	}
	offset := 0
	if v := c.Query("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}

	type row struct {
		ID               uuid.UUID
		WorkspaceID      uuid.UUID
		ChannelID        uuid.UUID
		UserID           *uuid.UUID
		Text             *string
		ParentID         *uuid.UUID
		ThreadRootID     *uuid.UUID
		UserDisplayName  *string
		UserAvatarFileID *uuid.UUID
		CreatedAt        time.Time
	}

	var rows []row

	q := h.db.Table("messages m").
		Select(`m.id, m.workspace_id, m.channel_id, m.user_id, m.text, m.parent_id, m.thread_root_id, m.created_at,
			u.display_name AS user_display_name, u.avatar_file_id AS user_avatar_file_id`).
		Joins("LEFT JOIN users u ON u.id = m.user_id").
		Where("m.channel_id = ?", chID)

	if threadRootIDStr != "" {
		if tid, err := uuid.Parse(threadRootIDStr); err == nil {
			q = q.Where("m.thread_root_id = ?", tid)
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"detail": "invalid thread_root_id"})
			return
		}
	} else if rootOnly {
		q = q.Where("m.thread_root_id IS NULL")
	}

	if err := q.Order("m.created_at ASC").Limit(limit).Offset(offset).Scan(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": "query failed"})
		return
	}

	out := make([]MsgOut, 0, len(rows))
	for _, r := range rows {
		out = append(out, MsgOut{
			ID:               r.ID,
			WorkspaceID:      r.WorkspaceID,
			ChannelID:        r.ChannelID,
			UserID:           derefUUID(r.UserID),
			UserDisplayName:  r.UserDisplayName,
			UserAvatarFileID: r.UserAvatarFileID, // ← 追加
			Text:             derefStr(r.Text),
			ParentID:         r.ParentID,
			ThreadRootID:     r.ThreadRootID,
			CreatedAt:        r.CreatedAt,
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
