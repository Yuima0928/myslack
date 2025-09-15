// backend/go/internal/http/handlers/channels.go
package handlers

import (
	"net/http"
	"strings"

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

type MemberCandidateRow struct {
	ID          uuid.UUID `json:"id"`
	Email       string    `json:"email"`
	DisplayName *string   `json:"display_name,omitempty"`
}

// SearchWorkspaceMembers godoc
// @Summary  Search members of the workspace which the channel belongs to (exclude current channel members)
// @Tags     channels
// @Produce  json
// @Param    channel_id path string true  "Channel ID (UUID)"
// @Param    q          query string true "query string (part of email or display_name)"
// @Param    limit      query int    false "limit (max 50, default 20)"
// @Success  200 {array}  handlers.MemberCandidateRow
// @Failure  400 {object} map[string]string
// @Failure  401 {object} map[string]string
// @Failure  404 {object} map[string]string
// @Security Bearer
// @Router   /channels/{channel_id}/members/search [get]
func (h *ChannelsHandler) SearchWorkspaceMembers(c *gin.Context) {
	chID := c.Param("channel_id")
	if chID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "channel_id required"})
		return
	}
	q := strings.TrimSpace(c.Query("q"))
	if q == "" {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "q required"})
		return
	}
	limit := 20
	if v := c.Query("limit"); v != "" {
		if n, err := parsePositiveInt(v, 1, 50); err == nil {
			limit = n
		}
	}

	// 1) チャンネルの属する workspace_id を引く
	var wsIDStr string
	if err := h.db.
		Table("channels").
		Select("workspace_id").
		Where("id = ?", chID).
		Limit(1).
		Row().
		Scan(&wsIDStr); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": "lookup channel failed"})
		return
	}
	wsID, err := uuid.Parse(wsIDStr)
	if err != nil || wsID == uuid.Nil {
		c.JSON(http.StatusNotFound, gin.H{"detail": "channel not found"})
		return
	}

	// 2) workspace_members に居るユーザーの中から検索
	//    すでに channel_members のユーザーは除外
	like := "%" + q + "%"
	rows := []MemberCandidateRow{}
	if err := h.db.Raw(`
		SELECT u.id, u.email, u.display_name
		FROM workspace_members wm
		JOIN users u ON u.id = wm.user_id
		WHERE wm.workspace_id = ?
		AND (u.email ILIKE ? OR u.display_name ILIKE ?)
		AND NOT EXISTS (
		SELECT 1 FROM channel_members cm
		WHERE cm.channel_id = ? AND cm.user_id = u.id
		)
		ORDER BY u.email
		LIMIT ?`,
		wsID, like, like, chID, limit,
	).Scan(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": "search failed"})
		return
	}

	c.JSON(http.StatusOK, rows)
}

type AddMemberIn struct {
	// 追加するユーザのUUID
	UserID string `json:"user_id" binding:"required,uuid" example:"6d4c2f52-1f1c-4e7d-92a2-4b2d4a3d9a10"`
	// 役割（未指定は member）
	Role string `json:"role" binding:"omitempty,oneof=owner member" example:"member"`
}

// 既存: AddMember（WSメンバーであることのチェックを入れていない場合は下のように強化）
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

	// チャンネルのWS取得
	var wsIDStr string
	if err := h.db.
		Table("channels").
		Select("workspace_id").
		Where("id = ?", chID).
		Limit(1).
		Row().
		Scan(&wsIDStr); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": "lookup channel failed"})
		return
	}
	wsID, err := uuid.Parse(wsIDStr)
	if err != nil || wsID == uuid.Nil {
		c.JSON(http.StatusNotFound, gin.H{"detail": "channel not found"})
		return
	}

	// 対象ユーザーがWSメンバーであることを検証
	var ok int
	if err := h.db.Raw(`
		SELECT 1 FROM workspace_members
		WHERE workspace_id = ? AND user_id = ? LIMIT 1
	`, wsID, in.UserID).Scan(&ok).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": "lookup membership failed"})
		return
	}
	if ok != 1 {
		c.JSON(http.StatusForbidden, gin.H{"detail": "user is not a workspace member"})
		return
	}

	rec := model.ChannelMember{
		UserID:    uuid.MustParse(in.UserID),
		ChannelID: uuid.MustParse(chID),
		Role:      role,
	}
	if err := h.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&rec).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": "add member failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ListByWorkspace godoc
// @Summary  List channels visible in a workspace
// @Tags     channels
// @Produce  json
// @Param    ws_id path string true "Workspace ID (UUID)"
// @Success  200 {array} map[string]any
// @Failure  401 {object} map[string]string
// @Failure  403 {object} map[string]string
// @Security Bearer
// @Router   /workspaces/{ws_id}/channels [get]
func (h *ChannelsHandler) ListByWorkspace(c *gin.Context) {
	uid := c.GetString("user_id")
	wsID := c.Param("ws_id")
	if uid == "" || wsID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"detail": "unauthorized"})
		return
	}

	// パブリック or 自分がメンバーのプライベート
	type row struct {
		ID        uuid.UUID `json:"id"`
		Name      string    `json:"name"`
		IsPrivate bool      `json:"is_private"`
	}
	var rows []row
	if err := h.db.
		Table("channels c").
		Select("c.id, c.name, c.is_private").
		Where("c.workspace_id = ?", wsID).
		Where(`
            c.is_private = false
            OR EXISTS (
			SELECT 1 FROM channel_members cm
			WHERE cm.channel_id = c.id AND cm.user_id = ?
            )`, uid).
		Order("c.name ASC").
		Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": "list failed"})
		return
	}
	c.JSON(http.StatusOK, rows)
}

func uuidPtr(v uuid.UUID) *uuid.UUID { return &v }
