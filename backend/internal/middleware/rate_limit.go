package middleware

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// LimitarTaxaPorIP restringe requisições por IP (seção 12: "rate limit em
// login e cadastro"). Guarda em memória — suficiente para uma instância;
// se o backend escalar horizontalmente, precisa migrar para um store
// compartilhado (ex.: Redis).
func LimitarTaxaPorIP(requisicoesPorSegundo float64, burst int) gin.HandlerFunc {
	var mu sync.Mutex
	limitadores := make(map[string]*rate.Limiter)

	obterLimitador := func(ip string) *rate.Limiter {
		mu.Lock()
		defer mu.Unlock()
		l, ok := limitadores[ip]
		if !ok {
			l = rate.NewLimiter(rate.Limit(requisicoesPorSegundo), burst)
			limitadores[ip] = l
		}
		return l
	}

	return func(c *gin.Context) {
		if !obterLimitador(c.ClientIP()).Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"erro": "muitas tentativas, tente novamente em instantes"})
			return
		}
		c.Next()
	}
}
