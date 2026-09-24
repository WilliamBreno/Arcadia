package middleware

import "github.com/gin-gonic/gin"

// CabecalhosSeguranca aplica cabeçalhos defensivos em toda resposta da API
// (que só devolve JSON ou arquivos enviados por usuários — nunca HTML).
func CabecalhosSeguranca() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		c.Next()
	}
}
