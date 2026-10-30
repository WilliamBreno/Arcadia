package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/WilliamBreno/Arcadia/backend/internal/middleware"
	"github.com/WilliamBreno/Arcadia/backend/internal/service"
)

// Revenda oficial (item 7.4).
type RevendaHandler struct {
	svc *service.RevendaService
}

func NovoRevendaHandler(s *service.RevendaService) *RevendaHandler { return &RevendaHandler{svc: s} }

// Anunciar é POST /itens/:id/revenda.
func (h *RevendaHandler) Anunciar(c *gin.Context) {
	id, ok := idDaURL(c)
	if !ok {
		return
	}
	uid := c.GetInt64(middleware.ChaveContextoUsuarioID)
	a, err := h.svc.Anunciar(uid, id)
	if err != nil {
		h.erro(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"item_id": a.ItemID, "status": a.Status, "valor_devolver_centavos": a.ValorDevolverCentavos})
}

// Retirar é DELETE /itens/:id/revenda.
func (h *RevendaHandler) Retirar(c *gin.Context) {
	id, ok := idDaURL(c)
	if !ok {
		return
	}
	if err := h.svc.Retirar(c.GetInt64(middleware.ChaveContextoUsuarioID), id); err != nil {
		h.erro(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"item_id": id, "status": service.RevendaRetirada})
}

// Minhas é GET /me/revendas.
func (h *RevendaHandler) Minhas(c *gin.Context) {
	lista, err := h.svc.Minhas(c.GetInt64(middleware.ChaveContextoUsuarioID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao ler suas revendas"})
		return
	}
	c.JSON(http.StatusOK, lista)
}

func (h *RevendaHandler) erro(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrItemNaoEncontrado), errors.Is(err, service.ErrRevendaNaoEncontrada):
		c.JSON(http.StatusNotFound, gin.H{"erro": err.Error()})
	case errors.Is(err, service.ErrItemNaoPertenceAoUsuario):
		c.JSON(http.StatusForbidden, gin.H{"erro": err.Error()})
	case errors.Is(err, service.ErrRevendaJaAnunciada), errors.Is(err, service.ErrRevendaEmAndamento), errors.Is(err, service.ErrRevendaSemEsgotar):
		c.JSON(http.StatusConflict, gin.H{"erro": err.Error()})
	case errors.Is(err, service.ErrRevendaNaoPermitida):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"erro": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro na revenda"})
	}
}
