package handler

import (
	"github.com/WilliamBreno/Arcadia/backend/internal/mail"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/WilliamBreno/Arcadia/backend/internal/service"
)

type JobHandler struct {
	checkout  *service.CheckoutService
	mail      *mail.Cliente
	carrinho  *service.RecuperacaoCarrinhoService
	fila      *service.FilaEsperaService
	revenda   *service.RevendaService
	perguntas *service.PerguntasService
}

// DefinirPerguntas liga a limpeza das respostas (item 7.6, 90 dias após o evento).
func (h *JobHandler) DefinirPerguntas(p *service.PerguntasService) { h.perguntas = p }

// DefinirRevenda liga a revenda oficial (item 7.4) ao job de reservas.
func (h *JobHandler) DefinirRevenda(r *service.RevendaService) { h.revenda = r }

// DefinirFila liga a lista de espera (item 7.3) ao job de reservas.
func (h *JobHandler) DefinirFila(f *service.FilaEsperaService) { h.fila = f }

// DefinirRecuperacao liga o e-mail de carrinho (item 7.2) ao job de reservas.
func (h *JobHandler) DefinirRecuperacao(s *service.RecuperacaoCarrinhoService) { h.carrinho = s }

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
	// Item 7.1: tenta de novo os estornos automáticos que o MP não confirmou.
	devolucoes := h.checkout.ReprocessarDevolucoes()
	carrinhos := 0
	if h.carrinho != nil {
		carrinhos, _ = h.carrinho.Enviar(time.Now())
	}
	var revendasEncerradas int64
	if h.revenda != nil {
		revendasEncerradas, _ = h.revenda.Encerrar(time.Now())
	}
	if h.perguntas != nil {
		_, _ = h.perguntas.ApagarVencidas(time.Now())
	}
	convidados := 0
	if h.fila != nil {
		convidados, _ = h.fila.Processar(time.Now())
	}
	c.JSON(http.StatusOK, gin.H{"expirados": total, "devolucoes_tentadas": devolucoes, "carrinhos": carrinhos, "fila_convidados": convidados, "revendas_encerradas": revendasEncerradas})
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
