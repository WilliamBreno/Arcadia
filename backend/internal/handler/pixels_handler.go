package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
	"github.com/WilliamBreno/Arcadia/backend/internal/service"
)

// Pixels de anúncio por evento (item 7.5), no painel do organizador.
type PixelsHandler struct {
	db      *gorm.DB
	eventos *repository.EventoRepository
	org     *OrganizadorHandler
}

func NovoPixelsHandler(db *gorm.DB, e *repository.EventoRepository, org *OrganizadorHandler) *PixelsHandler {
	return &PixelsHandler{db: db, eventos: e, org: org}
}

func (h *PixelsHandler) eventoDoOrganizador(c *gin.Context) (*domain.Evento, bool) {
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

// Obter é GET /org/eventos/:id/pixels.
func (h *PixelsHandler) Obter(c *gin.Context) {
	e, ok := h.eventoDoOrganizador(c)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, service.Pixels{Meta: e.PixelMeta, GA4: e.PixelGA4, TikTok: e.PixelTikTok})
}

// Salvar é PUT /org/eventos/:id/pixels.
func (h *PixelsHandler) Salvar(c *gin.Context) {
	e, ok := h.eventoDoOrganizador(c)
	if !ok {
		return
	}
	var req service.Pixels
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "dados inválidos"})
		return
	}
	p, err := service.NormalizarPixels(req)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"erro": err.Error()})
		return
	}
	if err := h.db.Model(&domain.Evento{}).Where("id = ?", e.ID).
		Updates(map[string]any{"pixel_meta": p.Meta, "pixel_ga4": p.GA4, "pixel_tiktok": p.TikTok}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao salvar"})
		return
	}
	c.JSON(http.StatusOK, p)
}
