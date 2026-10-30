package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/WilliamBreno/Arcadia/backend/internal/middleware"
	"github.com/WilliamBreno/Arcadia/backend/internal/service"
)

// Venda na porta pelo celular (item 9.4).
type PortaHandler struct {
	svc *service.CheckoutService
}

func NovoPortaHandler(s *service.CheckoutService) *PortaHandler { return &PortaHandler{svc: s} }

func (h *PortaHandler) erro(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrPortaSemAcesso):
		c.JSON(http.StatusForbidden, gin.H{"erro": err.Error()})
	case errors.Is(err, service.ErrPedidoNaoEncontrado):
		c.JSON(http.StatusNotFound, gin.H{"erro": "venda não encontrada"})
	case errors.Is(err, service.ErrEstoqueInsuficiente):
		c.JSON(http.StatusConflict, gin.H{"erro": err.Error()})
	case errors.Is(err, service.ErrMercadoPagoNaoConfigurado):
		c.JSON(http.StatusServiceUnavailable, gin.H{"erro": "Pix indisponível agora; registre como dinheiro ou tente de novo"})
	case errors.Is(err, service.ErrPortaFechada), errors.Is(err, service.ErrPortaInvalida), errors.Is(err, service.ErrPortaEmail),
		errors.Is(err, service.ErrTipoIndisponivel), errors.Is(err, service.ErrReservaExpirada), errors.Is(err, service.ErrPedidoNaoDisponivelParaPagamento):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"erro": err.Error()})
	default:
		slog.Error("venda na porta", "erro", err)
		c.JSON(http.StatusBadGateway, gin.H{"erro": "não foi possível concluir agora; tente de novo"})
	}
}

func usuarioDaPorta(c *gin.Context) int64 { return c.GetInt64(middleware.ChaveContextoUsuarioID) }

// Tipos é GET /checkin/eventos/:id/porta/tipos.
func (h *PortaHandler) Tipos(c *gin.Context) {
	id, ok := idDaURL(c)
	if !ok {
		return
	}
	ts, err := h.svc.TiposPorta(usuarioDaPorta(c), id)
	if err != nil {
		h.erro(c, err)
		return
	}
	out := make([]gin.H, 0, len(ts))
	for _, t := range ts {
		out = append(out, gin.H{"id": t.ID, "nome": t.Nome, "preco_centavos": t.PrecoCentavos, "total_pix_centavos": t.TotalPix, "disponivel": t.Disponivel})
	}
	c.JSON(http.StatusOK, out)
}

type vendaPortaResposta struct {
	ID            int64               `json:"id"`
	Status        string              `json:"status"`
	Metodo        string              `json:"metodo"`
	TotalCentavos int64               `json:"total_centavos"`
	ExpiraEm      *time.Time          `json:"expira_em"`
	Itens         []itemPortaResposta `json:"itens"`
}

type itemPortaResposta struct {
	Codigo string `json:"codigo"`
	Status string `json:"status"`
}

func paraVendaPorta(p *domain.Pedido, itens []domain.ItemPedido) vendaPortaResposta {
	r := vendaPortaResposta{ID: p.ID, Status: string(p.Status), TotalCentavos: p.TotalCentavos, ExpiraEm: p.ExpiraEm, Itens: []itemPortaResposta{}}
	if p.MetodoPorta != nil {
		r.Metodo = *p.MetodoPorta
	}
	for _, i := range itens {
		r.Itens = append(r.Itens, itemPortaResposta{Codigo: i.Codigo, Status: string(i.Status)})
	}
	return r
}

// Vender é POST /checkin/eventos/:id/porta/vendas.
func (h *PortaHandler) Vender(c *gin.Context) {
	id, ok := idDaURL(c)
	if !ok {
		return
	}
	var req struct {
		Itens []struct {
			TipoIngressoID int64 `json:"tipo_ingresso_id"`
			Quantidade     int   `json:"quantidade"`
		} `json:"itens"`
		Metodo string `json:"metodo"`
		Nome   string `json:"nome"`
		Email  string `json:"email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "dados inválidos"})
		return
	}
	itens := make([]service.ItemPorta, 0, len(req.Itens))
	for _, i := range req.Itens {
		itens = append(itens, service.ItemPorta{TipoIngressoID: i.TipoIngressoID, Quantidade: i.Quantidade})
	}
	p, its, err := h.svc.VenderNaPorta(usuarioDaPorta(c), id, itens, req.Metodo, req.Nome, req.Email)
	if err != nil {
		h.erro(c, err)
		return
	}
	c.JSON(http.StatusCreated, paraVendaPorta(p, its))
}

// Pix é POST /checkin/eventos/:id/porta/vendas/:pid/pix.
func (h *PortaHandler) Pix(c *gin.Context) {
	id, ok := idDaURL(c)
	if !ok {
		return
	}
	pid, ok := paramInt(c, "pid")
	if !ok {
		return
	}
	pix, err := h.svc.GerarPixPorta(usuarioDaPorta(c), id, pid)
	if err != nil {
		h.erro(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"copia_cola": pix.CopiaCola, "expira_em": pix.ExpiraEm})
}

// Situacao é GET /checkin/eventos/:id/porta/vendas/:pid.
func (h *PortaHandler) Situacao(c *gin.Context) {
	id, ok := idDaURL(c)
	if !ok {
		return
	}
	pid, ok := paramInt(c, "pid")
	if !ok {
		return
	}
	p, its, err := h.svc.SituacaoPorta(usuarioDaPorta(c), id, pid)
	if err != nil {
		h.erro(c, err)
		return
	}
	c.JSON(http.StatusOK, paraVendaPorta(p, its))
}

// Resumo é GET /checkin/eventos/:id/porta/resumo.
func (h *PortaHandler) Resumo(c *gin.Context) {
	id, ok := idDaURL(c)
	if !ok {
		return
	}
	uid := usuarioDaPorta(c)
	todos := h.svc.EhOrganizadorDoEvento(uid, id)
	ls, err := h.svc.ResumoPorta(uid, id, todos)
	if err != nil {
		h.erro(c, err)
		return
	}
	out := make([]gin.H, 0, len(ls))
	for _, l := range ls {
		out = append(out, gin.H{"vendedor": l.Vendedor, "ingressos": l.Ingressos, "pix_centavos": l.PixCentavos,
			"dinheiro_centavos": l.DinheiroCentavos, "taxa_dinheiro_centavos": l.TaxaDinheiroCentavos})
	}
	c.JSON(http.StatusOK, gin.H{"organizador": todos, "linhas": out})
}
