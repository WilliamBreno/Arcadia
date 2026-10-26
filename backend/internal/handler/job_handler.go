package handler

import (
	"github.com/WilliamBreno/Arcadia/backend/internal/mail"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/WilliamBreno/Arcadia/backend/internal/service"
)

type JobHandler struct {
	checkout *service.CheckoutService
	mail     *mail.Cliente
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

// EnviarEmails é o job POST /jobs/enviar-emails (item 5.5): esvazia a fila
// de e-mails (o processo em segundo plano já faz isso a cada poucos
// segundos; o job cobre instâncias que dormem por inatividade).
func (h *JobHandler) EnviarEmails(c *gin.Context) {
	if h.mail == nil {
		c.JSON(http.StatusOK, gin.H{"enviados": 0, "falhas": 0})
		return
	}
	enviados, falhas, err := h.mail.ProcessarFila(100)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao processar a fila de e-mails"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"enviados": enviados, "falhas": falhas})
}

// DefinirMail liga o job da fila de e-mails.
func (h *JobHandler) DefinirMail(m *mail.Cliente) { h.mail = m }
