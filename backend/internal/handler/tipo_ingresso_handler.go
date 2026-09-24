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

type TipoIngressoHandler struct {
	organizadorHandler *OrganizadorHandler
	service            *service.TipoIngressoService
}

func NovoTipoIngressoHandler(organizadorHandler *OrganizadorHandler, s *service.TipoIngressoService) *TipoIngressoHandler {
	return &TipoIngressoHandler{organizadorHandler: organizadorHandler, service: s}
}

type tipoIngressoResposta struct {
	ID            int64      `json:"id"`
	Nome          string     `json:"nome"`
	Descricao     string     `json:"descricao"`
	PrecoCentavos int64      `json:"preco_centavos"`
	Quantidade    int        `json:"quantidade"`
	VendasInicio  *time.Time `json:"vendas_inicio"`
	VendasFim     *time.Time `json:"vendas_fim"`
	MinPorPedido  int        `json:"min_por_pedido"`
	MaxPorPedido  int        `json:"max_por_pedido"`
	Ordem         int        `json:"ordem"`
	LoteGrupo     string     `json:"lote_grupo"`
	MeiaEntrada   bool       `json:"meia_entrada"`
	SessaoIDs     []int64    `json:"sessao_ids"`
	Ativo         bool       `json:"ativo"`
}

func paraTipoIngressoResposta(t *domain.TipoIngresso) tipoIngressoResposta {
	return tipoIngressoResposta{
		ID:            t.ID,
		Nome:          t.Nome,
		Descricao:     t.Descricao,
		PrecoCentavos: t.PrecoCentavos,
		Quantidade:    t.Quantidade,
		VendasInicio:  t.VendasInicio,
		VendasFim:     t.VendasFim,
		MinPorPedido:  t.MinPorPedido,
		MaxPorPedido:  t.MaxPorPedido,
		Ordem:         t.Ordem,
		LoteGrupo:     t.LoteGrupo,
		MeiaEntrada:   t.MeiaEntrada,
		SessaoIDs:     idsOuVazio(t.SessaoIDs),
		Ativo:         t.Ativo,
	}
}

type tipoIngressoRequest struct {
	Nome          string     `json:"nome" binding:"required,min=2"`
	Descricao     string     `json:"descricao"`
	PrecoCentavos int64      `json:"preco_centavos" binding:"gte=0"`
	Quantidade    int        `json:"quantidade" binding:"required,gt=0"`
	VendasInicio  *time.Time `json:"vendas_inicio"`
	VendasFim     *time.Time `json:"vendas_fim"`
	MinPorPedido  int        `json:"min_por_pedido"`
	MaxPorPedido  int        `json:"max_por_pedido"`
	Ordem         int        `json:"ordem"`
	LoteGrupo     string     `json:"lote_grupo"`
	MeiaEntrada   bool       `json:"meia_entrada"`
	SessaoIDs     []int64    `json:"sessao_ids"`
	Ativo         bool       `json:"ativo"`
}

func (req tipoIngressoRequest) paraDados() service.TipoIngressoDados {
	return service.TipoIngressoDados{
		Nome:          req.Nome,
		Descricao:     req.Descricao,
		PrecoCentavos: req.PrecoCentavos,
		Quantidade:    req.Quantidade,
		VendasInicio:  req.VendasInicio,
		VendasFim:     req.VendasFim,
		MinPorPedido:  req.MinPorPedido,
		MaxPorPedido:  req.MaxPorPedido,
		Ordem:         req.Ordem,
		LoteGrupo:     req.LoteGrupo,
		MeiaEntrada:   req.MeiaEntrada,
		SessaoIDs:     req.SessaoIDs,
		Ativo:         req.Ativo,
	}
}

func (h *TipoIngressoHandler) eventoIDDaURL(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "id do evento inválido"})
		return 0, false
	}
	return id, true
}

func (h *TipoIngressoHandler) tipoIDDaURL(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("ingressoId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "id do tipo de ingresso inválido"})
		return 0, false
	}
	return id, true
}

func (h *TipoIngressoHandler) Listar(c *gin.Context) {
	organizador, ok := h.organizadorHandler.ObterOrganizadorAtual(c)
	if !ok {
		return
	}
	eventoID, ok := h.eventoIDDaURL(c)
	if !ok {
		return
	}

	tipos, err := h.service.Listar(organizador.ID, eventoID)
	if err != nil {
		h.responderErro(c, err)
		return
	}

	resposta := make([]tipoIngressoResposta, 0, len(tipos))
	for i := range tipos {
		resposta = append(resposta, paraTipoIngressoResposta(&tipos[i]))
	}
	c.JSON(http.StatusOK, resposta)
}

func (h *TipoIngressoHandler) Criar(c *gin.Context) {
	organizador, ok := h.organizadorHandler.ObterOrganizadorAtual(c)
	if !ok {
		return
	}
	eventoID, ok := h.eventoIDDaURL(c)
	if !ok {
		return
	}

	var req tipoIngressoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "dados inválidos"})
		return
	}

	tipo, err := h.service.Criar(organizador.ID, eventoID, req.paraDados())
	if err != nil {
		h.responderErro(c, err)
		return
	}

	c.JSON(http.StatusCreated, paraTipoIngressoResposta(tipo))
}

func (h *TipoIngressoHandler) Atualizar(c *gin.Context) {
	organizador, ok := h.organizadorHandler.ObterOrganizadorAtual(c)
	if !ok {
		return
	}
	eventoID, ok := h.eventoIDDaURL(c)
	if !ok {
		return
	}
	tipoID, ok := h.tipoIDDaURL(c)
	if !ok {
		return
	}

	var req tipoIngressoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "dados inválidos"})
		return
	}

	tipo, err := h.service.Atualizar(organizador.ID, eventoID, tipoID, req.paraDados())
	if err != nil {
		h.responderErro(c, err)
		return
	}

	c.JSON(http.StatusOK, paraTipoIngressoResposta(tipo))
}

func (h *TipoIngressoHandler) Excluir(c *gin.Context) {
	organizador, ok := h.organizadorHandler.ObterOrganizadorAtual(c)
	if !ok {
		return
	}
	eventoID, ok := h.eventoIDDaURL(c)
	if !ok {
		return
	}
	tipoID, ok := h.tipoIDDaURL(c)
	if !ok {
		return
	}

	if err := h.service.Excluir(organizador.ID, eventoID, tipoID); err != nil {
		h.responderErro(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *TipoIngressoHandler) responderErro(c *gin.Context, err error) {
	if errors.Is(err, service.ErrEventoNaoPertenceAoOrganizador) {
		c.JSON(http.StatusForbidden, gin.H{"erro": "recurso não pertence a este organizador"})
		return
	}
	if errors.Is(err, service.ErrSessaoInvalidaParaTipo) {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"erro": err.Error()})
		return
	}
	if errors.Is(err, service.ErrCotaMeiaExcedida) {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"erro": err.Error()})
		return
	}
	c.JSON(http.StatusNotFound, gin.H{"erro": "não encontrado"})
}
