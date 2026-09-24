package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/WilliamBreno/Arcadia/backend/internal/middleware"
	"github.com/WilliamBreno/Arcadia/backend/internal/service"
)

type TransferenciaHandler struct {
	service *service.TransferenciaService
}

func NovoTransferenciaHandler(s *service.TransferenciaService) *TransferenciaHandler {
	return &TransferenciaHandler{service: s}
}

type transferirRequest struct {
	Nome  string `json:"nome" binding:"required"`
	Email string `json:"email" binding:"required,email"`
}

// Transferir é POST /itens/:id/transferir.
func (h *TransferenciaHandler) Transferir(c *gin.Context) {
	itemID, ok := idDaURL(c)
	if !ok {
		return
	}
	var req transferirRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "informe nome e e-mail válidos do novo titular"})
		return
	}
	usuarioID := c.GetInt64(middleware.ChaveContextoUsuarioID)

	item, err := h.service.Transferir(itemID, usuarioID, req.Nome, req.Email)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrItemNaoPertenceAoUsuario):
			c.JSON(http.StatusForbidden, gin.H{"erro": "ingresso não pertence a este usuário"})
		case errors.Is(err, service.ErrItemNaoEncontrado):
			c.JSON(http.StatusNotFound, gin.H{"erro": err.Error()})
		default:
			c.JSON(http.StatusUnprocessableEntity, gin.H{"erro": err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, paraItemPedidoResposta(item))
}
