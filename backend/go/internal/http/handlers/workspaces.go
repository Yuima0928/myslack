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
	Name string `json:"name" binding:"required"`
}

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

	c.JSON(http.StatusOK, gin.H{"id": ws.ID})
}

// GET /workspaces （自分が属するWS一覧）
func (h *WorkspacesHandler) ListMine(c *gin.Context) {
	uid := c.GetString("user_id")
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

// GET /workspaces/:ws_id/members
func (h *WorkspacesHandler) ListMembers(c *gin.Context) {
	wsID := c.Param("ws_id")
	type Row struct {
		UserID uuid.UUID `json:"user_id"`
		Role   string    `json:"role"`
		Email  string    `json:"email"`
		Name   *string   `json:"display_name"`
	}
	var rows []Row
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
