// backend/go/internal/http/handlers/channels.go
package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
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

	// 名前の正規化（前後空白カット）
	name := strings.TrimSpace(in.Name)
	if name == "" {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"detail": "name must not be empty"})
		return
	}

	ch := model.Channel{
		WorkspaceID: uuid.MustParse(wsID),
		Name:        name,
		IsPrivate:   in.IsPrivate,
		CreatedBy:   uuidPtr(uuid.MustParse(uid)),
	}

	if err := h.db.Create(&ch).Error; err != nil {
		// ← 一意制約違反（23505）を 409 で返す
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			// どの制約か見分けたい場合は pgErr.ConstraintName を確認
			// if pgErr.ConstraintName == "uq_channel_ws_name" { ... }
			c.JSON(http.StatusConflict, gin.H{
				"detail": "channel name already exists in this workspace",
				"code":   "channel_name_conflict",
			})
			return
		}

		// その他のDBエラーは 500
		c.JSON(http.StatusInternalServerError, gin.H{"detail": "create channel failed"})
		return
	}

	// 作成者=ownerとして channel_members へ（失敗しても致命ではないのでログだけでもOK）
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

func (h *ChannelsHandler) JoinSelf(c *gin.Context) {
	uid := c.GetString("user_id")
	wsID := c.Param("ws_id")
	chID := c.Param("channel_id")
	if uid == "" || wsID == "" || chID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "bad params"})
		return
	}

	// チャンネルの存在＆WS一致＆公開かどうか確認
	var ch struct {
		ID          uuid.UUID
		WorkspaceID uuid.UUID
		IsPrivate   bool
	}
	if err := h.db.
		Table("channels").
		Select("id, workspace_id, is_private").
		Where("id = ?", chID).
		Take(&ch).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"detail": "channel not found"})
		return
	}
	if ch.WorkspaceID.String() != wsID {
		c.JSON(http.StatusForbidden, gin.H{"detail": "forbidden"})
		return
	}
	if ch.IsPrivate {
		// 自己参加はNG。招待API(オーナー権限)でのみ追加させる
		c.JSON(http.StatusForbidden, gin.H{"detail": "cannot self-join private channel"})
		return
	}

	rec := model.ChannelMember{
		UserID:    uuid.MustParse(uid),
		ChannelID: uuid.MustParse(chID),
		Role:      "member",
	}
	if err := h.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&rec).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": "join failed"})
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

func (h *ChannelsHandler) IsMember(c *gin.Context) {
	uidStr := c.GetString("user_id")
	wsID := c.Param("ws_id") // WS付きルート互換
	chID := c.Param("channel_id")
	if uidStr == "" || chID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "bad params"})
		return
	}
	uID, err := uuid.Parse(uidStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"detail": "bad user id"})
		return
	}

	// チャンネル情報
	var ch struct {
		ID          uuid.UUID
		WorkspaceID uuid.UUID
		IsPrivate   bool
		CreatedBy   *uuid.UUID
	}
	if err := h.db.
		Table("channels").
		Select("id, workspace_id, is_private, created_by").
		Where("id = ?", chID).
		Take(&ch).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"detail": "channel not found"})
		return
	}

	// WS付きルートでWS不一致は403
	if wsID != "" && ch.WorkspaceID.String() != wsID {
		c.JSON(http.StatusForbidden, gin.H{"detail": "forbidden"})
		return
	}

	// 作成者＝メンバー扱い
	isOwner := ch.CreatedBy != nil && *ch.CreatedBy == uID

	// channel_members
	var chCnt int64
	if err := h.db.Table("channel_members").
		Where("channel_id = ? AND user_id = ?", ch.ID, uID).
		Count(&chCnt).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": "lookup failed"})
		return
	}
	isChMember := chCnt > 0

	// workspace_members
	var wsCnt int64
	if err := h.db.Table("workspace_members").
		Where("workspace_id = ? AND user_id = ?", ch.WorkspaceID, uID).
		Count(&wsCnt).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": "lookup failed"})
		return
	}
	isWsMember := wsCnt > 0

	// 読み取り/書き込み/メンバー判定
	canRead := (!ch.IsPrivate && isWsMember) || (ch.IsPrivate && (isOwner || isChMember))
	canPost := isOwner || isChMember
	isMember := isOwner || isChMember
	role := "none"
	if isOwner {
		role = "owner"
	} else if isChMember {
		role = "member"
	}

	c.JSON(http.StatusOK, gin.H{
		"is_member":  isMember,
		"can_read":   canRead,
		"can_post":   canPost,
		"role":       role,
		"is_private": ch.IsPrivate,
	})
}

func uuidPtr(v uuid.UUID) *uuid.UUID { return &v }
