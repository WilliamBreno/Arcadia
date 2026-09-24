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

type CronogramaHandler struct {
	organizadorHandler *OrganizadorHandler
	service            *service.CronogramaService
	fichas             *service.FichaService
}

func NovoCronogramaHandler(o *OrganizadorHandler, s *service.CronogramaService, f *service.FichaService) *CronogramaHandler {
	return &CronogramaHandler{organizadorHandler: o, service: s, fichas: f}
}

type cronogramaResposta struct {
	ID        int64      `json:"id"`
	Titulo    string     `json:"titulo"`
	Descricao string     `json:"descricao"`
	Local     string     `json:"local"`
	InicioEm  time.Time  `json:"inicio_em"`
	FimEm     *time.Time `json:"fim_em"`
}

func paraCronogramaResposta(i *domain.CronogramaItem) cronogramaResposta {
	return cronogramaResposta{ID: i.ID, Titulo: i.Titulo, Descricao: i.Descricao, Local: i.Local, InicioEm: i.InicioEm, FimEm: i.FimEm}
}

type cronogramaRequest struct {
	Titulo    string     `json:"titulo" binding:"required"`
	Descricao string     `json:"descricao"`
	Local     string     `json:"local"`
	InicioEm  time.Time  `json:"inicio_em" binding:"required"`
	FimEm     *time.Time `json:"fim_em"`
}

func (r cronogramaRequest) dados() service.CronogramaDados {
	return service.CronogramaDados{Titulo: r.Titulo, Descricao: r.Descricao, Local: r.Local, InicioEm: r.InicioEm, FimEm: r.FimEm}
}

func (h *CronogramaHandler) responderErro(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrEventoNaoPertenceAoOrganizador):
		c.JSON(http.StatusForbidden, gin.H{"erro": "evento não pertence a este organizador"})
	case errors.Is(err, service.ErrCronogramaNaoEncontrado), errors.Is(err, service.ErrFichaNaoEncontrada):
		c.JSON(http.StatusNotFound, gin.H{"erro": err.Error()})
	default:
		c.JSON(http.StatusUnprocessableEntity, gin.H{"erro": err.Error()})
	}
}

func (h *CronogramaHandler) Listar(c *gin.Context) {
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
	resp := make([]cronogramaResposta, 0, len(lista))
	for i := range lista {
		resp = append(resp, paraCronogramaResposta(&lista[i]))
	}
	c.JSON(http.StatusOK, resp)
}

func (h *CronogramaHandler) Criar(c *gin.Context) {
	org, ok := h.organizadorHandler.ObterOrganizadorAtual(c)
	if !ok {
		return
	}
	eventoID, ok := idDaURL(c)
	if !ok {
		return
	}
	var req cronogramaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "dados inválidos"})
		return
	}
	item, err := h.service.Criar(org.ID, eventoID, req.dados())
	if err != nil {
		h.responderErro(c, err)
		return
	}
	c.JSON(http.StatusCreated, paraCronogramaResposta(item))
}

func (h *CronogramaHandler) Atualizar(c *gin.Context) {
	org, ok := h.organizadorHandler.ObterOrganizadorAtual(c)
	if !ok {
		return
	}
	eventoID, ok := idDaURL(c)
	if !ok {
		return
	}
	itemID, err := strconv.ParseInt(c.Param("itemId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "id inválido"})
		return
	}
	var req cronogramaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "dados inválidos"})
		return
	}
	item, err := h.service.Atualizar(org.ID, eventoID, itemID, req.dados())
	if err != nil {
		h.responderErro(c, err)
		return
	}
	c.JSON(http.StatusOK, paraCronogramaResposta(item))
}

func (h *CronogramaHandler) Excluir(c *gin.Context) {
	org, ok := h.organizadorHandler.ObterOrganizadorAtual(c)
	if !ok {
		return
	}
	eventoID, ok := idDaURL(c)
	if !ok {
		return
	}
	itemID, err := strconv.ParseInt(c.Param("itemId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "id inválido"})
		return
	}
	if err := h.service.Excluir(org.ID, eventoID, itemID); err != nil {
		h.responderErro(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

type ordemRequest struct {
	FichaIDs []int64 `json:"ficha_ids" binding:"required"`
}

// DefinirOrdem é PUT /org/eventos/:id/ordem-apresentacao: a lista define
// a ordem 1..n dos participantes aprovados.
func (h *CronogramaHandler) DefinirOrdem(c *gin.Context) {
	org, ok := h.organizadorHandler.ObterOrganizadorAtual(c)
	if !ok {
		return
	}
	eventoID, ok := idDaURL(c)
	if !ok {
		return
	}
	var req ordemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "informe ficha_ids"})
		return
	}
	if err := h.fichas.DefinirOrdem(org.ID, eventoID, req.FichaIDs); err != nil {
		h.responderErro(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
