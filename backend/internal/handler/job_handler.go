package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/WilliamBreno/Arcadia/backend/internal/service"
)

type JobHandler struct {
	checkout *service.CheckoutService
}

func NovoJobHandler(checkout *service.CheckoutService) *JobHandler {
	return &JobHandler{checkout: checkout}
}

// ExpirarReservas é POST /jobs/expirar-reservas — deve rodar a cada
// minuto (seção 7.2: "job a cada minuto") via agendador externo.
func (h *JobHandler) ExpirarReservas(c *gin.Context) {
	total, err := h.checkout.ExpirarReservas()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao expirar reservas"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"expirados": total})
}
