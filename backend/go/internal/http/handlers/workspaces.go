// backend/go/internal/http/handlers/workspaces.go
package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

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

	ws := model.Workspace{Name: in.Name}
	if err := h.db.Create(&ws).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": "create workspace failed"})
		return
	}

	// 作成者=owner
	wm := model.WorkspaceMember{
		UserID:      uuid.MustParse(uid),
		WorkspaceID: ws.ID,
		Role:        "owner",
	}
	_ = h.db.Create(&wm).Error

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
