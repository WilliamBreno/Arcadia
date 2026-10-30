package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/WilliamBreno/Arcadia/backend/internal/middleware"
	"github.com/WilliamBreno/Arcadia/backend/internal/service"
)

// CRM do organizador (item 9.3).
type CRMHandler struct {
	svc *service.CRMService
	org *OrganizadorHandler
}

func NovoCRMHandler(s *service.CRMService, org *OrganizadorHandler) *CRMHandler {
	return &CRMHandler{svc: s, org: org}
}

func (h *CRMHandler) erro(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrCampanhaInvalida), errors.Is(err, service.ErrSegmentoInvalido), errors.Is(err, service.ErrSemContatos):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"erro": err.Error()})
	case errors.Is(err, service.ErrCotaEmails), errors.Is(err, service.ErrCampanhasHoje):
		c.JSON(http.StatusTooManyRequests, gin.H{"erro": err.Error()})
	default:
		slog.Error("crm", "erro", err)
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro no CRM"})
	}
}

func eventoDaQuery(v string) *int64 {
	if id, err := strconv.ParseInt(v, 10, 64); err == nil && id > 0 {
		return &id
	}
	return nil
}

// Resumo é GET /org/crm.
func (h *CRMHandler) Resumo(c *gin.Context) {
	org, ok := h.org.ObterOrganizadorAtual(c)
	if !ok {
		return
	}
	r, err := h.svc.Resumo(org.ID)
	if err != nil {
		h.erro(c, err)
		return
	}
	type campanha struct {
		Assunto       string    `json:"assunto"`
		Segmento      string    `json:"segmento"`
		EventoTitulo  string    `json:"evento_titulo"`
		Destinatarios int       `json:"destinatarios"`
		EnviadaEm     time.Time `json:"enviada_em"`
	}
	cs := make([]campanha, 0, len(r.Campanhas))
	for _, x := range r.Campanhas {
		cs = append(cs, campanha{Assunto: x.Assunto, Segmento: x.Segmento, EventoTitulo: x.EventoTitulo, Destinatarios: x.Destinatarios, EnviadaEm: x.EnviadaEm})
	}
	c.JSON(http.StatusOK, gin.H{"contatos": r.Contatos, "enviados_mes": r.EnviadosMes, "cota_mes": service.CotaEmailsMes,
		"cota_restante": r.CotaRestante, "campanhas_hoje": r.CampanhasHoje, "campanhas_por_dia": service.CampanhasPorDia, "campanhas": cs})
}

// Previa é GET /org/crm/previa?segmento=&evento_id=.
func (h *CRMHandler) Previa(c *gin.Context) {
	org, ok := h.org.ObterOrganizadorAtual(c)
	if !ok {
		return
	}
	n, err := h.svc.Previa(org.ID, c.DefaultQuery("segmento", service.SegmentoTodos), eventoDaQuery(c.Query("evento_id")))
	if err != nil {
		h.erro(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"destinatarios": n})
}

// Enviar é POST /org/crm/campanhas.
func (h *CRMHandler) Enviar(c *gin.Context) {
	org, ok := h.org.ObterOrganizadorAtual(c)
	if !ok {
		return
	}
	var req struct {
		Assunto  string `json:"assunto"`
		Corpo    string `json:"corpo"`
		Segmento string `json:"segmento"`
		EventoID *int64 `json:"evento_id"`
		Teste    bool   `json:"teste"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "dados inválidos"})
		return
	}
	n, err := h.svc.Enviar(org.ID, c.GetInt64(middleware.ChaveContextoUsuarioID), service.DadosCampanha{
		Assunto: req.Assunto, Corpo: req.Corpo, Segmento: req.Segmento, EventoID: req.EventoID, SoTeste: req.Teste,
	})
	if err != nil {
		h.erro(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"destinatarios": n})
}

// ExportarCSV é GET /org/crm/contatos.csv (Mailchimp/Brevo).
func (h *CRMHandler) ExportarCSV(c *gin.Context) {
	org, ok := h.org.ObterOrganizadorAtual(c)
	if !ok {
		return
	}
	b, err := h.svc.ExportarCSV(org.ID)
	if err != nil {
		h.erro(c, err)
		return
	}
	c.Header("Content-Disposition", `attachment; filename="contatos-evve.csv"`)
	c.Header("Cache-Control", "no-store")
	c.Data(http.StatusOK, "text/csv; charset=utf-8", b)
}
