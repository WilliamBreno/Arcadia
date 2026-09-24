package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/WilliamBreno/Arcadia/backend/internal/service"
)

type AfiliadoHandler struct {
	organizadorHandler *OrganizadorHandler
	service            *service.AfiliadoService
}

func NovoAfiliadoHandler(o *OrganizadorHandler, s *service.AfiliadoService) *AfiliadoHandler {
	return &AfiliadoHandler{organizadorHandler: o, service: s}
}

func (h *AfiliadoHandler) responderErro(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrEventoNaoPertenceAoOrganizador):
		c.JSON(http.StatusForbidden, gin.H{"erro": "evento não pertence a este organizador"})
	case errors.Is(err, service.ErrAfiliadoNaoEncontrado):
		c.JSON(http.StatusNotFound, gin.H{"erro": err.Error()})
	default:
		c.JSON(http.StatusUnprocessableEntity, gin.H{"erro": err.Error()})
	}
}

func (h *AfiliadoHandler) Listar(c *gin.Context) {
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
	resp := make([]gin.H, 0, len(lista))
	for _, a := range lista {
		resp = append(resp, gin.H{"id": a.ID, "nome": a.Nome, "codigo": a.Codigo, "ativo": a.Ativo,
			"pedidos": a.Pedidos, "itens": a.Itens, "receita_centavos": a.Receita})
	}
	c.JSON(http.StatusOK, resp)
}

type criarAfiliadoRequest struct {
	Nome string `json:"nome" binding:"required"`
}

func (h *AfiliadoHandler) Criar(c *gin.Context) {
	org, ok := h.organizadorHandler.ObterOrganizadorAtual(c)
	if !ok {
		return
	}
	eventoID, ok := idDaURL(c)
	if !ok {
		return
	}
	var req criarAfiliadoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "informe o nome"})
		return
	}
	a, err := h.service.Criar(org.ID, eventoID, req.Nome)
	if err != nil {
		h.responderErro(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": a.ID, "nome": a.Nome, "codigo": a.Codigo})
}

func (h *AfiliadoHandler) Desativar(c *gin.Context) {
	org, ok := h.organizadorHandler.ObterOrganizadorAtual(c)
	if !ok {
		return
	}
	eventoID, ok := idDaURL(c)
	if !ok {
		return
	}
	afID, err := strconv.ParseInt(c.Param("afiliadoId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "id inválido"})
		return
	}
	if err := h.service.Desativar(org.ID, eventoID, afID); err != nil {
		h.responderErro(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
