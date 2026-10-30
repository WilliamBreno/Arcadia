package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/WilliamBreno/Arcadia/backend/internal/service"
)

// Mapa de assentos e mesas (item 9.5).
type LugaresHandler struct {
	svc *service.LugaresService
	org *OrganizadorHandler
}

func NovoLugaresHandler(s *service.LugaresService, org *OrganizadorHandler) *LugaresHandler {
	return &LugaresHandler{svc: s, org: org}
}

func (h *LugaresHandler) erro(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrMapaNaoEncontrado):
		c.JSON(http.StatusNotFound, gin.H{"erro": err.Error()})
	case errors.Is(err, service.ErrEventoNaoPertenceAoOrganizador):
		c.JSON(http.StatusForbidden, gin.H{"erro": "sem acesso a este evento"})
	case errors.Is(err, service.ErrSetorComIngressos):
		c.JSON(http.StatusConflict, gin.H{"erro": err.Error()})
	case errors.Is(err, service.ErrDadosSetor):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"erro": err.Error()})
	default:
		slog.Error("mapa de lugares", "erro", err)
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao carregar o mapa"})
	}
}

// Publico é GET /eventos/:slug/lugares: cadeiras livres e ocupadas.
func (h *LugaresHandler) Publico(c *gin.Context) {
	setores, err := h.svc.Publico(c.Param("slug"))
	if err != nil {
		h.erro(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"setores": setores})
}

func (h *LugaresHandler) orgEvento(c *gin.Context) (int64, int64, bool) {
	org, ok := h.org.ObterOrganizadorAtual(c)
	if !ok {
		return 0, 0, false
	}
	id, ok := idDaURL(c)
	return org.ID, id, ok
}

// Painel é GET /org/eventos/:id/lugares.
func (h *LugaresHandler) Painel(c *gin.Context) {
	orgID, id, ok := h.orgEvento(c)
	if !ok {
		return
	}
	setores, err := h.svc.Painel(orgID, id)
	if err != nil {
		h.erro(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"setores": setores})
}

type setorRequest struct {
	Nome              string  `json:"nome"`
	Formato           string  `json:"formato"`
	TipoIngressoIDs   []int64 `json:"tipo_ingresso_ids"`
	Fileiras          int     `json:"fileiras"`
	LugaresPorFileira int     `json:"lugares_por_fileira"`
	Mesas             int     `json:"mesas"`
	PrimeiraMesa      int     `json:"primeira_mesa"`
	CadeirasPorMesa   int     `json:"cadeiras_por_mesa"`
	MesaInteira       bool    `json:"mesa_inteira"`
}

// CriarSetor é POST /org/eventos/:id/setores.
func (h *LugaresHandler) CriarSetor(c *gin.Context) {
	orgID, id, ok := h.orgEvento(c)
	if !ok {
		return
	}
	var req setorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "dados inválidos"})
		return
	}
	setorID, err := h.svc.CriarSetor(orgID, id, service.DadosSetor{
		Nome: req.Nome, Formato: req.Formato, TipoIngressoIDs: req.TipoIngressoIDs,
		Fileiras: req.Fileiras, LugaresPorFileira: req.LugaresPorFileira,
		Mesas: req.Mesas, PrimeiraMesa: req.PrimeiraMesa, CadeirasPorMesa: req.CadeirasPorMesa, MesaInteira: req.MesaInteira,
	})
	if err != nil {
		h.erro(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": setorID})
}

// AtualizarSetor é PUT /org/eventos/:id/setores/:sid (nome e tipos).
func (h *LugaresHandler) AtualizarSetor(c *gin.Context) {
	orgID, id, ok := h.orgEvento(c)
	if !ok {
		return
	}
	sid, ok := paramInt(c, "sid")
	if !ok {
		return
	}
	var req setorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "dados inválidos"})
		return
	}
	if err := h.svc.AtualizarSetor(orgID, id, sid, req.Nome, req.TipoIngressoIDs); err != nil {
		h.erro(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ExcluirSetor é DELETE /org/eventos/:id/setores/:sid.
func (h *LugaresHandler) ExcluirSetor(c *gin.Context) {
	orgID, id, ok := h.orgEvento(c)
	if !ok {
		return
	}
	sid, ok := paramInt(c, "sid")
	if !ok {
		return
	}
	if err := h.svc.ExcluirSetor(orgID, id, sid); err != nil {
		h.erro(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// MesaInteira é PUT /org/eventos/:id/setores/:sid/mesas/:grupo.
func (h *LugaresHandler) MesaInteira(c *gin.Context) {
	orgID, id, ok := h.orgEvento(c)
	if !ok {
		return
	}
	sid, ok := paramInt(c, "sid")
	if !ok {
		return
	}
	var req struct {
		MesaInteira bool `json:"mesa_inteira"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "dados inválidos"})
		return
	}
	if err := h.svc.DefinirMesaInteira(orgID, id, sid, c.Param("grupo"), req.MesaInteira); err != nil {
		h.erro(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
