package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ExigirCronSecret protege rotas de job (seção 4 do plano: "rota
// protegida por X-Cron-Secret") chamadas por um agendador externo
// (cron do Render, GitHub Actions schedule, etc.), não por usuários.
func ExigirCronSecret(segredo string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if segredo == "" || c.GetHeader("X-Cron-Secret") != segredo {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"erro": "não autorizado"})
			return
		}
		c.Next()
	}
}
