package handler

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/WilliamBreno/Arcadia/backend/internal/service"
)

type SessaoHandler struct {
	organizadorHandler *OrganizadorHandler
	service            *service.SessaoService
}

func NovoSessaoHandler(o *OrganizadorHandler, s *service.SessaoService) *SessaoHandler {
	return &SessaoHandler{organizadorHandler: o, service: s}
}

type sessaoResposta struct {
	ID       int64      `json:"id"`
	Titulo   string     `json:"titulo"`
	InicioEm time.Time  `json:"inicio_em"`
	FimEm    *time.Time `json:"fim_em"`
	Status   string     `json:"status"`
}

func paraSessaoResposta(s *domain.Sessao) sessaoResposta {
	return sessaoResposta{ID: s.ID, Titulo: s.Titulo, InicioEm: s.InicioEm, FimEm: s.FimEm, Status: string(s.Status)}
}

func (h *SessaoHandler) responderErro(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrEventoNaoPertenceAoOrganizador):
		c.JSON(http.StatusForbidden, gin.H{"erro": "evento não pertence a este organizador"})
	case errors.Is(err, service.ErrSessaoNaoEncontrada):
		c.JSON(http.StatusNotFound, gin.H{"erro": err.Error()})
	case errors.Is(err, service.ErrSessaoJaCancelada), errors.Is(err, service.ErrSessaoEmUso):
		c.JSON(http.StatusConflict, gin.H{"erro": err.Error()})
	default:
		c.JSON(http.StatusUnprocessableEntity, gin.H{"erro": err.Error()})
	}
}

type sessaoRequest struct {
	Titulo   string     `json:"titulo"`
	InicioEm time.Time  `json:"inicio_em" binding:"required"`
	FimEm    *time.Time `json:"fim_em"`
}

func (r sessaoRequest) dados() service.SessaoDados {
	return service.SessaoDados{Titulo: r.Titulo, InicioEm: r.InicioEm, FimEm: r.FimEm}
}

func (h *SessaoHandler) Listar(c *gin.Context) {
	org, ok := h.organizadorHandler.ObterOrganizadorAtual(c)
	if !ok {
		return
	}
	eventoID, ok := idDaURL(c)
	if !ok {
		return
	}
	lista, err := h.service.Listar(org.ID, eventoID)
	if err != nil {
		h.responderErro(c, err)
		return
	}
	resp := make([]sessaoResposta, 0, len(lista))
	for i := range lista {
		resp = append(resp, paraSessaoResposta(&lista[i]))
	}
	c.JSON(http.StatusOK, resp)
}

func (h *SessaoHandler) Criar(c *gin.Context) {
	org, ok := h.organizadorHandler.ObterOrganizadorAtual(c)
	if !ok {
		return
	}
	eventoID, ok := idDaURL(c)
	if !ok {
		return
	}
	var req sessaoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "informe o início da sessão"})
		return
	}
	s, err := h.service.Criar(org.ID, eventoID, req.dados())
	if err != nil {
		h.responderErro(c, err)
		return
	}
	c.JSON(http.StatusCreated, paraSessaoResposta(s))
}

func (h *SessaoHandler) sessaoIDDaURL(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("sessaoId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "id inválido"})
		return 0, false
	}
	return id, true
}

func (h *SessaoHandler) Atualizar(c *gin.Context) {
	org, ok := h.organizadorHandler.ObterOrganizadorAtual(c)
	if !ok {
		return
	}
	eventoID, ok := idDaURL(c)
	if !ok {
		return
	}
	sessaoID, ok := h.sessaoIDDaURL(c)
	if !ok {
		return
	}
	var req sessaoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "dados inválidos"})
		return
	}
	s, err := h.service.Atualizar(org.ID, eventoID, sessaoID, req.dados())
	if err != nil {
		h.responderErro(c, err)
		return
	}
	c.JSON(http.StatusOK, paraSessaoResposta(s))
}

func (h *SessaoHandler) Excluir(c *gin.Context) {
	org, ok := h.organizadorHandler.ObterOrganizadorAtual(c)
	if !ok {
		return
	}
	eventoID, ok := idDaURL(c)
	if !ok {
		return
	}
	sessaoID, ok := h.sessaoIDDaURL(c)
	if !ok {
		return
	}
	if err := h.service.Excluir(org.ID, eventoID, sessaoID); err != nil {
		h.responderErro(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

type cancelarSessaoRequest struct {
	Motivo string `json:"motivo"`
}

// Cancelar é POST /org/eventos/:id/sessoes/:sessaoId/cancelar (só dono —
// dispara reembolsos).
func (h *SessaoHandler) Cancelar(c *gin.Context) {
	org, ok := h.organizadorHandler.ObterOrganizadorDono(c)
	if !ok {
		return
	}
	eventoID, ok := idDaURL(c)
	if !ok {
		return
	}
	sessaoID, ok := h.sessaoIDDaURL(c)
	if !ok {
		return
	}
	var req cancelarSessaoRequest
	_ = c.ShouldBindJSON(&req)
	sucessos, falhas, err := h.service.Cancelar(org.ID, eventoID, sessaoID, req.Motivo)
	if err != nil {
		h.responderErro(c, err)
		return
	}
	if falhas == nil {
		falhas = []int64{}
	}
	c.JSON(http.StatusOK, gin.H{"reembolsos_concluidos": sucessos, "falhas": falhas})
}
