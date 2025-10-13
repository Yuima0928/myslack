package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Health godoc
// @Summary Health check
// @Tags    health
// @Success 200 {object} map[string]bool
// @Router  /health [get]
func Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
