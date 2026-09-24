package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/WilliamBreno/Arcadia/backend/internal/middleware"
	"github.com/WilliamBreno/Arcadia/backend/internal/service"
)

type CancelamentoHandler struct {
	service *service.CancelamentoService
}

func NovoCancelamentoHandler(s *service.CancelamentoService) *CancelamentoHandler {
	return &CancelamentoHandler{service: s}
}

type decisaoCancelamentoResposta struct {
	Pode                   bool   `json:"pode"`
	Motivo                 string `json:"motivo"`
	ValorReembolsoCentavos int64  `json:"valor_reembolso_centavos"`
}

func paraDecisaoResposta(d service.DecisaoCancelamento) decisaoCancelamentoResposta {
	return decisaoCancelamentoResposta{Pode: d.Pode, Motivo: d.Motivo, ValorReembolsoCentavos: d.ValorReembolsoCentavos}
}

// Simular é GET /itens/:id/cancelamento — "pode cancelar? quanto volta?"
func (h *CancelamentoHandler) Simular(c *gin.Context) {
	itemID, ok := idDaURL(c)
	if !ok {
		return
	}
	usuarioID := c.GetInt64(middleware.ChaveContextoUsuarioID)

	decisao, err := h.service.Simular(itemID, usuarioID)
	if err != nil {
		h.responderErro(c, err)
		return
	}
	c.JSON(http.StatusOK, paraDecisaoResposta(decisao))
}

// Cancelar é POST /itens/:id/cancelar.
func (h *CancelamentoHandler) Cancelar(c *gin.Context) {
	itemID, ok := idDaURL(c)
	if !ok {
		return
	}
	usuarioID := c.GetInt64(middleware.ChaveContextoUsuarioID)

	item, err := h.service.Cancelar(itemID, usuarioID)
	if err != nil {
		if errors.Is(err, service.ErrCancelamentoNaoPermitido) {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"erro": err.Error()})
			return
		}
		h.responderErro(c, err)
		return
	}
	c.JSON(http.StatusOK, paraItemPedidoResposta(item))
}

func (h *CancelamentoHandler) responderErro(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrItemNaoPertenceAoUsuario):
		c.JSON(http.StatusForbidden, gin.H{"erro": "item não pertence a este usuário"})
	case errors.Is(err, service.ErrItemNaoEncontrado), errors.Is(err, service.ErrPagamentoNaoEncontrado):
		c.JSON(http.StatusNotFound, gin.H{"erro": "item não encontrado"})
	default:
		slog.Error("erro ao processar cancelamento", "erro", err)
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao processar cancelamento, tente novamente em instantes"})
	}
}

// --- Organizador: cancelar evento inteiro (seção 7.5) ---

type cancelarEventoRequest struct {
	Motivo string `json:"motivo" binding:"required,min=3"`
}

type cancelarEventoResposta struct {
	Sucessos int     `json:"sucessos"`
	Falhas   []int64 `json:"falhas"`
}

func (h *EventoHandler) Cancelar(c *gin.Context) {
	organizador, ok := h.organizadorHandler.ObterOrganizadorDono(c)
	if !ok {
		return
	}
	eventoID, ok := idDaURL(c)
	if !ok {
		return
	}

	var req cancelarEventoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "informe o motivo do cancelamento"})
		return
	}

	sucessos, falhas, err := h.cancelamentoService.CancelarEvento(organizador.ID, eventoID, req.Motivo)
	if err != nil {
		if errors.Is(err, service.ErrEventoNaoPertenceAoOrganizador) {
			c.JSON(http.StatusForbidden, gin.H{"erro": "evento não pertence a este organizador"})
			return
		}
		slog.Error("erro ao cancelar evento", "erro", err, "evento_id", eventoID)
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao cancelar evento"})
		return
	}

	if falhas == nil {
		falhas = []int64{}
	}
	c.JSON(http.StatusOK, cancelarEventoResposta{Sucessos: sucessos, Falhas: falhas})
}
