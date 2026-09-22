package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Healthz responde com o status da API, usado por health checks de deploy.
func Healthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
