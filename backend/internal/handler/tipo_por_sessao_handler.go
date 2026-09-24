package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/WilliamBreno/Arcadia/backend/internal/service"
)

func idsOuVazio(ids []int64) []int64 {
	if ids == nil {
		return []int64{}
	}
	return ids
}

type precoPorSessaoRequest struct {
	SessaoID      int64 `json:"sessao_id" binding:"required"`
	PrecoCentavos int64 `json:"preco_centavos" binding:"gte=0"`
	Quantidade    int   `json:"quantidade" binding:"required,gt=0"`
}

type criarPorSessaoRequest struct {
	Nome         string                  `json:"nome" binding:"required,min=2"`
	Descricao    string                  `json:"descricao"`
	MeiaEntrada  bool                    `json:"meia_entrada"`
	MinPorPedido int                     `json:"min_por_pedido"`
	MaxPorPedido int                     `json:"max_por_pedido"`
	Sessoes      []precoPorSessaoRequest `json:"sessoes" binding:"required,min=1,dive"`
}

// CriarPorSessao é POST /org/eventos/:id/ingressos/por-sessao: um ingresso
// por sessão, cada um com preço e quantidade próprios.
func (h *TipoIngressoHandler) CriarPorSessao(c *gin.Context) {
	organizador, ok := h.organizadorHandler.ObterOrganizadorAtual(c)
	if !ok {
		return
	}
	eventoID, ok := h.eventoIDDaURL(c)
	if !ok {
		return
	}
	var req criarPorSessaoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "dados inválidos"})
		return
	}
	linhas := make([]service.PrecoPorSessao, 0, len(req.Sessoes))
	for _, s := range req.Sessoes {
		linhas = append(linhas, service.PrecoPorSessao{SessaoID: s.SessaoID, PrecoCentavos: s.PrecoCentavos, Quantidade: s.Quantidade})
	}
	criados, err := h.service.CriarPorSessao(organizador.ID, eventoID, service.TipoIngressoDados{
		Nome: req.Nome, Descricao: req.Descricao, MeiaEntrada: req.MeiaEntrada,
		MinPorPedido: req.MinPorPedido, MaxPorPedido: req.MaxPorPedido, Ativo: true,
	}, linhas)
	if err != nil {
		h.responderErro(c, err)
		return
	}
	resp := make([]tipoIngressoResposta, 0, len(criados))
	for i := range criados {
		resp = append(resp, paraTipoIngressoResposta(&criados[i]))
	}
	c.JSON(http.StatusCreated, resp)
}
