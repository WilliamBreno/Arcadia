package middleware

import (
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// Contador é onde as tentativas ficam guardadas. Em produção é o Postgres
// (repository.LimiteTaxaRepository), compartilhado entre instâncias.
type Contador interface {
	Registrar(chave string, janela time.Duration, agora time.Time) (int, error)
}

// LimitarPorJanela permite no máximo `limite` requisições por IP e por rota
// a cada `janela`, contadas no Contador compartilhado (item 5.5). O limite
// por IP é folgado de propósito: num evento, centenas de pessoas saem pelo
// mesmo IP do Wi-Fi do local. A proteção forte de cada conta é o limite por
// e-mail no login (AuthHandler). Se o contador
// falhar (banco fora), deixa passar e registra o erro — o limite protege
// contra abuso, não pode derrubar o login de todo mundo.
func LimitarPorJanela(contador Contador, nome string, limite int, janela time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		n, err := contador.Registrar(nome+":"+c.FullPath()+":ip:"+c.ClientIP(), janela, time.Now())
		if err != nil {
			slog.Error("limite de taxa indisponível", "erro", err, "limite", nome)
			c.Next()
			return
		}
		if n > limite {
			c.Header("Retry-After", strconv.Itoa(int(janela.Seconds())))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"erro": "muitas tentativas, tente novamente em instantes"})
			return
		}
		c.Next()
	}
}

// LimitarTaxaPorIP é o limite antigo, em memória (uma instância só).
// Mantido para testes e uso local; as rotas usam LimitarPorJanela.
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
