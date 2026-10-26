package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestAlertadorRespeitaIntervalo(t *testing.T) {
	a := NovoAlertador("http://exemplo")
	t0 := time.Date(2026, 10, 26, 10, 0, 0, 0, time.UTC)
	if !a.deveAvisar("GET /x", t0) {
		t.Fatal("primeiro aviso deveria sair")
	}
	if a.deveAvisar("GET /x", t0.Add(time.Minute)) {
		t.Error("segundo aviso em 1 min deveria ser segurado")
	}
	if !a.deveAvisar("POST /y", t0.Add(time.Minute)) {
		t.Error("outra rota deveria avisar")
	}
	if !a.deveAvisar("GET /x", t0.Add(6*time.Minute)) {
		t.Error("depois do intervalo deveria avisar de novo")
	}
}

func TestAlertarErrosAvisaWebhookEm500(t *testing.T) {
	recebido := make(chan string, 2)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var corpo map[string]string
		_ = json.NewDecoder(r.Body).Decode(&corpo)
		recebido <- corpo["content"]
	}))
	defer srv.Close()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(IDRequisicao(), AlertarErros(NovoAlertador(srv.URL)), TratadorDeErros())
	r.GET("/quebra", func(c *gin.Context) { panic("teste") })
	r.GET("/ok", func(c *gin.Context) { c.Status(http.StatusOK) })

	for _, rota := range []string{"/ok", "/quebra"} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, rota, nil))
	}
	select {
	case msg := <-recebido:
		if !strings.Contains(msg, "500") || !strings.Contains(msg, "/quebra") {
			t.Errorf("mensagem inesperada: %q", msg)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("webhook não recebeu o alerta do erro 500")
	}
	select {
	case msg := <-recebido:
		t.Errorf("rota com 200 não deveria alertar: %q", msg)
	case <-time.After(300 * time.Millisecond):
	}
}
