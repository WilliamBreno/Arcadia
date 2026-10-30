package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
	"github.com/WilliamBreno/Arcadia/backend/internal/service"
)

// Perguntas personalizadas do checkout (item 7.6), no painel do organizador.
type PerguntasHandler struct {
	svc     *service.PerguntasService
	eventos *repository.EventoRepository
	org     *OrganizadorHandler
}

func NovoPerguntasHandler(s *service.PerguntasService, e *repository.EventoRepository, org *OrganizadorHandler) *PerguntasHandler {
	return &PerguntasHandler{svc: s, eventos: e, org: org}
}

func (h *PerguntasHandler) evento(c *gin.Context) (*domain.Evento, bool) {
	org, ok := h.org.ObterOrganizadorAtual(c)
	if !ok {
		return nil, false
	}
	id, ok := idDaURL(c)
	if !ok {
		return nil, false
	}
	e, err := h.eventos.BuscarPorID(id)
	if err != nil || e.OrganizadorID != org.ID {
		c.JSON(http.StatusNotFound, gin.H{"erro": "evento não encontrado"})
		return nil, false
	}
	return e, true
}

// Listar é GET /org/eventos/:id/perguntas.
func (h *PerguntasHandler) Listar(c *gin.Context) {
	e, ok := h.evento(c)
	if !ok {
		return
	}
	ps, err := h.svc.Listar(e.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao listar"})
		return
	}
	c.JSON(http.StatusOK, ps)
}

// Salvar é POST /org/eventos/:id/perguntas e PUT /org/eventos/:id/perguntas/:pergunta.
func (h *PerguntasHandler) Salvar(c *gin.Context) {
	e, ok := h.evento(c)
	if !ok {
		return
	}
	var p service.Pergunta
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "dados inválidos"})
		return
	}
	p.ID = 0
	if s := c.Param("pergunta"); s != "" {
		id, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"erro": "pergunta inválida"})
			return
		}
		p.ID = id
	}
	salva, err := h.svc.Salvar(e.ID, p)
	switch {
	case errors.Is(err, service.ErrPerguntaInvalida), errors.Is(err, service.ErrMuitasPerguntas):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"erro": err.Error()})
	case errors.Is(err, gorm.ErrRecordNotFound):
		c.JSON(http.StatusNotFound, gin.H{"erro": "pergunta não encontrada"})
	case err != nil:
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao salvar"})
	default:
		c.JSON(http.StatusOK, salva)
	}
}

// Remover é DELETE /org/eventos/:id/perguntas/:pergunta.
func (h *PerguntasHandler) Remover(c *gin.Context) {
	e, ok := h.evento(c)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(c.Param("pergunta"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "pergunta inválida"})
		return
	}
	if err := h.svc.Remover(e.ID, id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"erro": "pergunta não encontrada"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// Exportar é GET /org/eventos/:id/respostas.csv.
func (h *PerguntasHandler) Exportar(c *gin.Context) {
	e, ok := h.evento(c)
	if !ok {
		return
	}
	perguntas, linhas, err := h.svc.Respostas(e.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao gerar a planilha"})
		return
	}
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", `attachment; filename="respostas-evento-`+strconv.FormatInt(e.ID, 10)+`.csv"`)
	_, _ = c.Writer.Write([]byte("\xEF\xBB\xBF")) // BOM: Excel abre com acentos certos
	_ = service.EscreverRespostasCSV(c.Writer, perguntas, linhas)
}
