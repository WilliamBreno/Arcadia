package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/WilliamBreno/Arcadia/backend/internal/service"
)

// Coprodução com divisão de repasse (item 9.8). Só o dono do perfil de
// organizador mexe (é dinheiro).
type CoproducaoHandler struct {
	svc *service.CoproducaoService
	org *OrganizadorHandler
}

func NovoCoproducaoHandler(s *service.CoproducaoService, org *OrganizadorHandler) *CoproducaoHandler {
	return &CoproducaoHandler{svc: s, org: org}
}

func (h *CoproducaoHandler) erro(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrEventoNaoPertenceAoOrganizador):
		c.JSON(http.StatusForbidden, gin.H{"erro": "só o organizador principal do evento mexe na coprodução"})
	case errors.Is(err, service.ErrCoproducaoNaoEncontrada):
		c.JSON(http.StatusNotFound, gin.H{"erro": err.Error()})
	case errors.Is(err, service.ErrCoproducaoTravada):
		c.JSON(http.StatusConflict, gin.H{"erro": err.Error()})
	case errors.Is(err, service.ErrCoproducao):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"erro": err.Error()})
	default:
		slog.Error("coprodução", "erro", err)
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro na coprodução"})
	}
}

// DoEvento é GET /org/eventos/:id/coproducao.
func (h *CoproducaoHandler) DoEvento(c *gin.Context) {
	org, ok := h.org.ObterOrganizadorDono(c)
	if !ok {
		return
	}
	id, ok := idDaURL(c)
	if !ok {
		return
	}
	r, err := h.svc.DoEvento(org.ID, id)
	if err != nil {
		h.erro(c, err)
		return
	}
	c.JSON(http.StatusOK, r)
}

// Convidar é POST /org/eventos/:id/coproducao {email, percentual_bp}.
func (h *CoproducaoHandler) Convidar(c *gin.Context) {
	org, ok := h.org.ObterOrganizadorDono(c)
	if !ok {
		return
	}
	id, ok := idDaURL(c)
	if !ok {
		return
	}
	var req struct {
		Email        string `json:"email" binding:"required"`
		PercentualBP int64  `json:"percentual_bp"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "informe o e-mail e o percentual"})
		return
	}
	cp, err := h.svc.Convidar(org.ID, id, req.Email, req.PercentualBP)
	if err != nil {
		h.erro(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": cp.ID})
}

// Propor é PUT /org/eventos/:id/coproducao/:cid {percentual_bp} (0 = tirar).
func (h *CoproducaoHandler) Propor(c *gin.Context) {
	org, ok := h.org.ObterOrganizadorDono(c)
	if !ok {
		return
	}
	id, ok := idDaURL(c)
	if !ok {
		return
	}
	cid, ok := paramInt(c, "cid")
	if !ok {
		return
	}
	var req struct {
		PercentualBP int64 `json:"percentual_bp"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "dados inválidos"})
		return
	}
	if err := h.svc.Propor(org.ID, id, cid, req.PercentualBP); err != nil {
		h.erro(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// Minhas é GET /org/coproducoes (convites e coproduções em que participo).
func (h *CoproducaoHandler) Minhas(c *gin.Context) {
	org, ok := h.org.ObterOrganizadorDono(c)
	if !ok {
		return
	}
	lista, err := h.svc.Minhas(org.ID)
	if err != nil {
		h.erro(c, err)
		return
	}
	c.JSON(http.StatusOK, lista)
}

// Responder é POST /org/coproducoes/:cid/responder {aceitar}.
func (h *CoproducaoHandler) Responder(c *gin.Context) {
	org, ok := h.org.ObterOrganizadorDono(c)
	if !ok {
		return
	}
	cid, ok := paramInt(c, "cid")
	if !ok {
		return
	}
	var req struct {
		Aceitar bool `json:"aceitar"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "dados inválidos"})
		return
	}
	if err := h.svc.Responder(org.ID, cid, req.Aceitar); err != nil {
		h.erro(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// Sair é POST /org/coproducoes/:cid/sair.
func (h *CoproducaoHandler) Sair(c *gin.Context) {
	org, ok := h.org.ObterOrganizadorDono(c)
	if !ok {
		return
	}
	cid, ok := paramInt(c, "cid")
	if !ok {
		return
	}
	if err := h.svc.Sair(org.ID, cid); err != nil {
		h.erro(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
