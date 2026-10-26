package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
)

// DefinirContestacoes liga a lista de contestações do painel admin (item 5.3).
func (h *AdminHandler) DefinirContestacoes(c *repository.ContestacaoRepository) { h.contestacoes = c }

type contestacaoResposta struct {
	ID                int64     `json:"id"`
	PedidoID          int64     `json:"pedido_id"`
	EventoTitulo      string    `json:"evento_titulo"`
	UsuarioEmail      string    `json:"usuario_email"`
	PagamentoExterno  string    `json:"pagamento_externo"`
	ValorCentavos     int64     `json:"valor_centavos"`
	Status            string    `json:"status"`
	StatusProcessador string    `json:"status_processador"`
	CriadoEm          time.Time `json:"criado_em"`
	AtualizadoEm      time.Time `json:"atualizado_em"`
}

// ListarContestacoes é GET /admin/contestacoes?status=em_disputa|perdida|resolvida.
func (h *AdminHandler) ListarContestacoes(c *gin.Context) {
	lista, err := h.contestacoes.Listar(c.Query("status"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao listar contestações"})
		return
	}
	resp := make([]contestacaoResposta, 0, len(lista))
	for _, l := range lista {
		resp = append(resp, contestacaoResposta{
			ID: l.ID, PedidoID: l.PedidoID, EventoTitulo: l.EventoTitulo, UsuarioEmail: l.UsuarioEmail,
			PagamentoExterno: l.MPPaymentID, ValorCentavos: l.ValorCentavos, Status: l.Status,
			StatusProcessador: l.StatusProcessador, CriadoEm: l.CriadoEm, AtualizadoEm: l.AtualizadoEm,
		})
	}
	c.JSON(http.StatusOK, resp)
}
