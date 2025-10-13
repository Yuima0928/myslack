package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"slackgo/internal/model"
)

type WorkspacesHandler struct {
	db *gorm.DB
}

func NewWorkspacesHandler(db *gorm.DB) *WorkspacesHandler {
	return &WorkspacesHandler{db: db}
}

type CreateWorkspaceIn struct {
	// ワークスペース名
	Name string `json:"name" binding:"required" example:"my-team"`
}

// Swagger 表示用の行構造体（メンバー一覧）
type WorkspaceMemberRow struct {
	UserID uuid.UUID `json:"user_id"       example:"6d4c2f52-1f1c-4e7d-92a2-4b2d4a3d9a10"`
	Role   string    `json:"role"          example:"owner"`
	Email  string    `json:"email"         example:"alice@example.com"`
	Name   *string   `json:"display_name"  example:"Alice"`
}

// Create godoc
// @Summary  Create workspace
// @Tags     workspaces
// @Accept   json
// @Produce  json
// @Param    body  body     CreateWorkspaceIn true "workspace payload"
// @Success  200   {object} map[string]string "id: UUID string"
// @Failure  401   {object} map[string]string
// @Failure  422   {object} map[string]string
// @Security Bearer
// @Router   /workspaces [post]
// POST /workspaces  （作成者=ownerで workspace_members へ追加）
func (h *WorkspacesHandler) Create(c *gin.Context) {
	var in CreateWorkspaceIn
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"detail": err.Error()})
		return
	}
	uid := c.GetString("user_id")
	if uid == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"detail": "unauthorized"})
		return
	}

	var ws model.Workspace
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		ws = model.Workspace{Name: in.Name}
		if err := tx.Create(&ws).Error; err != nil {
			return err
		}
		wm := model.WorkspaceMember{
			UserID:      uuid.MustParse(uid),
			WorkspaceID: ws.ID,
			Role:        "owner",
		}
		if err := tx.Create(&wm).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": "create workspace failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"id": ws.ID.String()})
}

// ListMine godoc
// @Summary  List my workspaces
// @Tags     workspaces
// @Produce  json
// @Success  200 {array}  model.Workspace
// @Failure  401 {object} map[string]string
// @Security Bearer
// @Router   /workspaces [get]
// GET /workspaces （自分が属するWS一覧）
func (h *WorkspacesHandler) ListMine(c *gin.Context) {
	uid := c.GetString("user_id")
	if uid == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"detail": "unauthorized"})
		return
	}

	var ws []model.Workspace
	if err := h.db.
		Table("workspaces").
		Joins("JOIN workspace_members wm ON wm.workspace_id = workspaces.id AND wm.user_id = ?", uid).
		Find(&ws).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": "list failed"})
		return
	}
	c.JSON(http.StatusOK, ws)
}

// ListMembers godoc
// @Summary  List workspace members
// @Tags     workspaces
// @Produce  json
// @Param    ws_id path string true "Workspace ID (UUID)"
// @Success  200 {array}  handlers.WorkspaceMemberRow
// @Failure  401 {object} map[string]string
// @Failure  404 {object} map[string]string
// @Security Bearer
// @Router   /workspaces/{ws_id}/members [get]
// GET /workspaces/:ws_id/members
func (h *WorkspacesHandler) ListMembers(c *gin.Context) {
	wsID := c.Param("ws_id")
	if wsID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "ws_id required"})
		return
	}

	var rows []WorkspaceMemberRow
	if err := h.db.
		Table("workspace_members wm").
		Select("wm.user_id, wm.role, u.email, u.display_name").
		Joins("JOIN users u ON u.id = wm.user_id").
		Where("wm.workspace_id = ?", wsID).
		Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": "list failed"})
		return
	}
	c.JSON(http.StatusOK, rows)
}

type AddWorkspaceMemberIn struct {
	// 追加するユーザのUUID
	UserID string `json:"user_id" binding:"required,uuid" example:"6d4c2f52-1f1c-4e7d-92a2-4b2d4a3d9a10"`
	// 役割（未指定は member）
	Role string `json:"role" binding:"omitempty,oneof=owner member" example:"member"`
}

// AddMember godoc
// @Summary  Add member to workspace
// @Tags     workspaces
// @Accept   json
// @Produce  json
// @Param    ws_id path string true "Workspace ID (UUID)"
// @Param    body  body  AddWorkspaceMemberIn true "member payload"
// @Success  200 {object} map[string]bool "ok: true"
// @Failure  400 {object} map[string]string
// @Failure  401 {object} map[string]string
// @Failure  404 {object} map[string]string
// @Failure  422 {object} map[string]string
// @Security Bearer
// @Router   /workspaces/{ws_id}/members [post]
func (h *WorkspacesHandler) AddMember(c *gin.Context) {
	wsID := c.Param("ws_id")
	if wsID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "ws_id required"})
		return
	}
	var in AddWorkspaceMemberIn
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"detail": err.Error()})
		return
	}
	role := in.Role
	if role == "" {
		role = "member"
	}

	wsUUID, err := uuid.Parse(wsID)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"detail": "invalid ws_id"})
		return
	}
	uuidTarget, err := uuid.Parse(in.UserID)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"detail": "invalid user_id"})
		return
	}

	// workspace_members に登録（重複なら何もしない）
	rec := model.WorkspaceMember{
		UserID:      uuidTarget,
		WorkspaceID: wsUUID,
		Role:        role,
	}
	if err := h.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&rec).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": "add member failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// --- 追加: 登録ユーザー検索（email / display_name 部分一致） ---

type UserRow struct {
	ID          uuid.UUID `json:"id"`
	Email       string    `json:"email"`
	DisplayName *string   `json:"display_name,omitempty"`
}

// SearchUsers godoc
// @Summary  Search registered users (email/display_name contains q)
// @Tags     users
// @Produce  json
// @Param    q     query string true  "query string (part of email or display_name)"
// @Param    limit query int    false "limit (max 50, default 20)"
// @Success  200 {array}  handlers.UserRow
// @Failure  400 {object} map[string]string
// @Security Bearer
// @Router   /users/search [get]
func (h *WorkspacesHandler) SearchUsers(c *gin.Context) {
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

	like := "%" + q + "%"
	rows := []UserRow{}
	if err := h.db.
		Table("users").
		Select("id, email, display_name").
		Where("email ILIKE ? OR display_name ILIKE ?", like, like).
		Limit(limit).
		Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": "search failed"})
		return
	}
	c.JSON(http.StatusOK, rows)
}

// 小ヘルパー
func parsePositiveInt(s string, min, max int) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	if err != nil {
		return 0, err
	}
	if n < min {
		n = min
	}
	if n > max {
		n = max
	}
	return n, nil
}
