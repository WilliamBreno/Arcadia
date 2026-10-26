package middleware

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// Monitoramento de erros (item 5.5).

// ChaveContextoRequestID guarda o id da requisição (também devolvido no
// cabeçalho X-Request-ID), para achar o log de um erro relatado por alguém.
const ChaveContextoRequestID = "request_id"

var requestIDValido = regexp.MustCompile(`^[A-Za-z0-9-]{8,64}$`)

// IDRequisicao aceita o X-Request-ID de quem chamou (se bem formado) ou
// gera um novo.
func IDRequisicao() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Request-ID")
		if !requestIDValido.MatchString(id) {
			b := make([]byte, 8)
			_, _ = rand.Read(b)
			id = hex.EncodeToString(b)
		}
		c.Set(ChaveContextoRequestID, id)
		c.Header("X-Request-ID", id)
		c.Next()
	}
}

// Alertador avisa um webhook (Discord ou Slack) sobre erros 500, no máximo
// uma vez a cada `intervalo` por rota, para não inundar o canal.
type Alertador struct {
	url       string
	intervalo time.Duration
	http      *http.Client
	mu        sync.Mutex
	ultimo    map[string]time.Time
}

func NovoAlertador(url string) *Alertador {
	return &Alertador{url: url, intervalo: 5 * time.Minute, http: &http.Client{Timeout: 5 * time.Second}, ultimo: map[string]time.Time{}}
}

// deveAvisar aplica o intervalo mínimo por chave. Testado.
func (a *Alertador) deveAvisar(chave string, agora time.Time) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	if t, ok := a.ultimo[chave]; ok && agora.Sub(t) < a.intervalo {
		return false
	}
	a.ultimo[chave] = agora
	return true
}

// Avisar envia a mensagem em segundo plano. Sem URL configurada, só loga.
func (a *Alertador) Avisar(chave, mensagem string) {
	if a == nil || a.url == "" || !a.deveAvisar(chave, time.Now()) {
		return
	}
	// "content" é o campo do Discord; "text", o do Slack.
	corpo, _ := json.Marshal(map[string]string{"content": mensagem, "text": mensagem})
	go func() {
		resp, err := a.http.Post(a.url, "application/json", bytes.NewReader(corpo))
		if err != nil {
			slog.Warn("falha ao enviar alerta", "erro", err)
			return
		}
		resp.Body.Close()
	}()
}

// AlertarErros avisa quando uma requisição termina em 5xx (inclui panics
// recuperados pelo TratadorDeErros, que deve vir depois na cadeia).
func AlertarErros(a *Alertador) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if c.Writer.Status() < 500 {
			return
		}
		rota := c.FullPath()
		if rota == "" {
			rota = c.Request.URL.Path
		}
		a.Avisar(c.Request.Method+" "+rota, fmt.Sprintf("⚠️ Evve API: %d em %s %s (request_id %s)",
			c.Writer.Status(), c.Request.Method, rota, c.GetString(ChaveContextoRequestID)))
	}
}
