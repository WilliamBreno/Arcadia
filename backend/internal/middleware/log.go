package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// LogRequisicoes registra cada requisição em log estruturado (slog),
// substituindo o logger de texto padrão do Gin.
func LogRequisicoes() gin.HandlerFunc {
	return func(c *gin.Context) {
		inicio := time.Now()
		caminho := c.Request.URL.Path

		c.Next()

		slog.Info("requisicao",
			"metodo", c.Request.Method,
			"caminho", caminho,
			"status", c.Writer.Status(),
			"duracao_ms", time.Since(inicio).Milliseconds(),
		)
	}
}
