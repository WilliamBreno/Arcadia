package handler

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
	"github.com/WilliamBreno/Arcadia/backend/internal/service"
)

type CupomHandler struct {
	organizadorHandler *OrganizadorHandler
	eventos            *repository.EventoRepository
	service            *service.CupomService
}

func NovoCupomHandler(o *OrganizadorHandler, e *repository.EventoRepository, s *service.CupomService) *CupomHandler {
	return &CupomHandler{organizadorHandler: o, eventos: e, service: s}
}

type cupomResposta struct {
	ID        int64      `json:"id"`
	Codigo    string     `json:"codigo"`
	Tipo      string     `json:"tipo"`
	Valor     int64      `json:"valor"`
	MaxUsos   *int       `json:"max_usos"`
	Usos      int        `json:"usos"`
	ValidoDe  *time.Time `json:"valido_de"`
	ValidoAte *time.Time `json:"valido_ate"`
	Ativo     bool       `json:"ativo"`
}

func paraCupomResposta(c *domain.Cupom) cupomResposta {
	return cupomResposta{ID: c.ID, Codigo: c.Codigo, Tipo: string(c.Tipo), Valor: c.Valor, MaxUsos: c.MaxUsos,
		Usos: c.Usos, ValidoDe: c.ValidoDe, ValidoAte: c.ValidoAte, Ativo: c.Ativo}
}

type cupomRequest struct {
	Codigo    string     `json:"codigo" binding:"required"`
	Tipo      string     `json:"tipo" binding:"required"`
	Valor     int64      `json:"valor" binding:"required"`
	MaxUsos   *int       `json:"max_usos"`
	ValidoDe  *time.Time `json:"valido_de"`
	ValidoAte *time.Time `json:"valido_ate"`
}

func (h *CupomHandler) responderErro(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrEventoNaoPertenceAoOrganizador):
		c.JSON(http.StatusForbidden, gin.H{"erro": "evento não pertence a este organizador"})
	case errors.Is(err, service.ErrCupomJaExiste):
		c.JSON(http.StatusConflict, gin.H{"erro": err.Error()})
	case errors.Is(err, service.ErrCupomDadosInvalidos), errors.Is(err, service.ErrCupomInvalido):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"erro": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro inesperado"})
	}
}

func (h *CupomHandler) Listar(c *gin.Context) {
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
	resp := make([]cupomResposta, 0, len(lista))
	for i := range lista {
		resp = append(resp, paraCupomResposta(&lista[i]))
	}
	c.JSON(http.StatusOK, resp)
}

func (h *CupomHandler) Criar(c *gin.Context) {
	org, ok := h.organizadorHandler.ObterOrganizadorAtual(c)
	if !ok {
		return
	}
	eventoID, ok := idDaURL(c)
	if !ok {
		return
	}
	var req cupomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "dados inválidos"})
		return
	}
	cupom, err := h.service.Criar(org.ID, eventoID, service.CupomDados{
		Codigo: req.Codigo, Tipo: domain.TipoCupom(req.Tipo), Valor: req.Valor,
		MaxUsos: req.MaxUsos, ValidoDe: req.ValidoDe, ValidoAte: req.ValidoAte,
	})
	if err != nil {
		h.responderErro(c, err)
		return
	}
	c.JSON(http.StatusCreated, paraCupomResposta(cupom))
}

func (h *CupomHandler) Desativar(c *gin.Context) {
	org, ok := h.organizadorHandler.ObterOrganizadorAtual(c)
	if !ok {
		return
	}
	eventoID, ok := idDaURL(c)
	if !ok {
		return
	}
	cupomID, err := strconv.ParseInt(c.Param("cupomId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "id inválido"})
		return
	}
	if err := h.service.Desativar(org.ID, eventoID, cupomID); err != nil {
		h.responderErro(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

type validarCupomRequest struct {
	Codigo string `json:"codigo" binding:"required"`
}

// Validar é POST /eventos/:slug/cupom — prévia do desconto no checkout.
func (h *CupomHandler) Validar(c *gin.Context) {
	evento, err := h.eventos.BuscarPorSlug(c.Param("slug"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"erro": "evento não encontrado"})
		return
	}
	var req validarCupomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "dados inválidos"})
		return
	}
	cupom, err := h.service.Validar(evento.ID, req.Codigo)
	if err != nil {
		h.responderErro(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"codigo": cupom.Codigo, "tipo": cupom.Tipo, "valor": cupom.Valor})
}
