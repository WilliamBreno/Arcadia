package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/WilliamBreno/Arcadia/backend/internal/middleware"
	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
	"github.com/WilliamBreno/Arcadia/backend/internal/service"
)

// Verificação de organizadores e reserva de risco no painel admin (item 5.4).

// DefinirOrganizadores liga a lista de verificação.
func (h *AdminHandler) DefinirOrganizadores(o *repository.OrganizadorRepository) { h.organizadores = o }

type organizadorAdminResposta struct {
	ID            int64      `json:"id"`
	NomePublico   string     `json:"nome_publico"`
	Slug          string     `json:"slug"`
	UsuarioEmail  string     `json:"usuario_email"`
	TipoPessoa    string     `json:"tipo_pessoa"`
	Documento     string     `json:"documento"`
	ChavePix      string     `json:"chave_pix"`
	TipoChavePix  string     `json:"tipo_chave_pix"`
	Eventos       int64      `json:"eventos"`
	RepassesPagos int64      `json:"repasses_pagos"`
	VerificadoEm  *time.Time `json:"verificado_em"`
	CriadoEm      time.Time  `json:"criado_em"`
}

// ListarOrganizadores é GET /admin/organizadores?pendentes=1.
func (h *AdminHandler) ListarOrganizadores(c *gin.Context) {
	lista, err := h.organizadores.ListarAdmin(c.Query("pendentes") == "1")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao listar organizadores"})
		return
	}
	resp := make([]organizadorAdminResposta, 0, len(lista))
	for _, o := range lista {
		resp = append(resp, organizadorAdminResposta{
			ID: o.ID, NomePublico: o.NomePublico, Slug: o.Slug, UsuarioEmail: o.UsuarioEmail,
			TipoPessoa: string(o.TipoPessoa), Documento: o.Documento, ChavePix: o.ChavePix, TipoChavePix: o.TipoChavePix,
			Eventos: o.Eventos, RepassesPagos: o.RepassesPagos, VerificadoEm: o.VerificadoEm, CriadoEm: o.CriadoEm,
		})
	}
	c.JSON(http.StatusOK, resp)
}

// DefinirVerificacao é POST /admin/organizadores/:id/verificacao {"verificado": bool}.
func (h *AdminHandler) DefinirVerificacao(c *gin.Context) {
	id, ok := idDaURL(c)
	if !ok {
		return
	}
	var req struct {
		Verificado *bool `json:"verificado" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "informe verificado: true ou false"})
		return
	}
	if _, err := h.organizadores.BuscarPorID(id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"erro": "organizador não encontrado"})
		return
	}
	adminID := c.GetInt64(middleware.ChaveContextoUsuarioID)
	if err := h.organizadores.DefinirVerificacao(id, &adminID, *req.Verificado); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao salvar verificação"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "verificado": *req.Verificado})
}

// PagarReserva é POST /admin/repasses/:id/reserva/pagar.
func (h *RepasseHandler) PagarReserva(c *gin.Context) {
	id, ok := idDaURL(c)
	if !ok {
		return
	}
	var req pagarRepasseRequest
	_ = c.ShouldBindJSON(&req)
	rep, err := h.service.PagarReserva(id, req.ComprovanteURL, req.Observacao)
	h.responderReserva(c, rep, err)
}

// UsarReserva é POST /admin/repasses/:id/reserva/usar {"observacao": "..."}.
// Decisão manual do admin para cobrir uma contestação perdida.
func (h *RepasseHandler) UsarReserva(c *gin.Context) {
	id, ok := idDaURL(c)
	if !ok {
		return
	}
	var req struct {
		Observacao string `json:"observacao" binding:"required,min=5"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "explique o motivo (ex.: qual contestação a reserva cobre)"})
		return
	}
	rep, err := h.service.UsarReserva(id, req.Observacao)
	h.responderReserva(c, rep, err)
}

func (h *RepasseHandler) responderReserva(c *gin.Context, rep any, err error) {
	switch {
	case err == nil:
		c.JSON(http.StatusOK, rep)
	case errors.Is(err, service.ErrRepasseNaoEncontrado):
		c.JSON(http.StatusNotFound, gin.H{"erro": err.Error()})
	case errors.Is(err, service.ErrReservaIndisponivel), errors.Is(err, service.ErrReservaNoPrazo), errors.Is(err, service.ErrRepasseAindaPendente):
		c.JSON(http.StatusConflict, gin.H{"erro": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao atualizar a reserva"})
	}
}
