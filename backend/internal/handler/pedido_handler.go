package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/WilliamBreno/Arcadia/backend/internal/middleware"
	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
	"github.com/WilliamBreno/Arcadia/backend/internal/service"
)

type PedidoHandler struct {
	eventos  *repository.EventoRepository
	usuarios *repository.UsuarioRepository
	service  *service.CheckoutService
}

func NovoPedidoHandler(eventos *repository.EventoRepository, usuarios *repository.UsuarioRepository, s *service.CheckoutService) *PedidoHandler {
	return &PedidoHandler{eventos: eventos, usuarios: usuarios, service: s}
}

type itemPedidoResposta struct {
	ID                 int64      `json:"id"`
	TipoIngressoID     int64      `json:"tipo_ingresso_id"`
	TitularNome        string     `json:"titular_nome"`
	TitularEmail       string     `json:"titular_email"`
	PrecoCentavos      int64      `json:"preco_centavos"`
	TaxaCentavos       int64      `json:"taxa_plataforma_centavos"`
	GarantiaContratada bool       `json:"garantia_contratada"`
	GarantiaCentavos   int64      `json:"garantia_centavos"`
	DescontoCentavos   int64      `json:"desconto_centavos"`
	TotalCentavos      int64      `json:"total_centavos"`
	Status             string     `json:"status"`
	Codigo             string     `json:"codigo"`
	QRToken            string     `json:"qr_token,omitempty"`
	UtilizadoEm        *time.Time `json:"utilizado_em"`
}

func paraItemPedidoResposta(i *domain.ItemPedido) itemPedidoResposta {
	return itemPedidoResposta{
		ID: i.ID, TipoIngressoID: i.TipoIngressoID, TitularNome: i.TitularNome, TitularEmail: i.TitularEmail,
		PrecoCentavos: i.PrecoCentavos, TaxaCentavos: i.TaxaPlataformaCentavos,
		GarantiaContratada: i.GarantiaContratada, GarantiaCentavos: i.GarantiaCentavos, DescontoCentavos: i.DescontoCentavos, TotalCentavos: i.TotalCentavos,
		Status: string(i.Status), Codigo: i.Codigo, QRToken: i.QRToken, UtilizadoEm: i.UtilizadoEm,
	}
}

type pedidoResposta struct {
	ID            int64                `json:"id"`
	EventoID      int64                `json:"evento_id"`
	Status        string               `json:"status"`
	TotalCentavos int64                `json:"total_centavos"`
	ExpiraEm      *time.Time           `json:"expira_em"`
	Itens         []itemPedidoResposta `json:"itens"`
}

func paraPedidoResposta(p *domain.Pedido, itens []domain.ItemPedido) pedidoResposta {
	itensResp := make([]itemPedidoResposta, 0, len(itens))
	for i := range itens {
		itensResp = append(itensResp, paraItemPedidoResposta(&itens[i]))
	}
	return pedidoResposta{
		ID: p.ID, EventoID: p.EventoID, Status: string(p.Status), TotalCentavos: p.TotalCentavos,
		ExpiraEm: p.ExpiraEm, Itens: itensResp,
	}
}

type itemRequest struct {
	TipoIngressoID     int64 `json:"tipo_ingresso_id" binding:"required"`
	Quantidade         int   `json:"quantidade" binding:"required,gt=0"`
	GarantiaContratada bool  `json:"garantia_contratada"`
}

type criarPedidoRequest struct {
	Itens []itemRequest `json:"itens" binding:"required,min=1,dive"`
	Cupom string        `json:"cupom"`
}

// Criar é POST /eventos/:slug/pedidos — reserva + cálculo (seção 8).
func (h *PedidoHandler) Criar(c *gin.Context) {
	evento, err := h.eventos.BuscarPorSlug(c.Param("slug"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"erro": "evento não encontrado"})
		return
	}

	var req criarPedidoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "dados inválidos"})
		return
	}

	usuarioID := c.GetInt64(middleware.ChaveContextoUsuarioID)
	comprador, err := h.usuarios.BuscarPorID(usuarioID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao identificar comprador"})
		return
	}

	itensReq := make([]service.ItemRequisitado, len(req.Itens))
	for i, ir := range req.Itens {
		itensReq[i] = service.ItemRequisitado{
			TipoIngressoID: ir.TipoIngressoID, Quantidade: ir.Quantidade, GarantiaContratada: ir.GarantiaContratada,
		}
	}

	pedido, itens, err := h.service.Reservar(usuarioID, evento.ID, itensReq, comprador.Nome, comprador.Email, req.Cupom)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"erro": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, paraPedidoResposta(pedido, itens))
}

// Obter é GET /pedidos/:id.
func (h *PedidoHandler) Obter(c *gin.Context) {
	pedidoID, ok := idDaURL(c)
	if !ok {
		return
	}
	usuarioID := c.GetInt64(middleware.ChaveContextoUsuarioID)

	pedido, itens, err := h.service.ObterPedido(usuarioID, pedidoID)
	if err != nil {
		h.responderErro(c, err)
		return
	}
	c.JSON(http.StatusOK, paraPedidoResposta(pedido, itens))
}

// Pagar é POST /pedidos/:id/pagar — cria a preference no Mercado Pago e
// devolve o link de checkout para o frontend redirecionar o comprador.
func (h *PedidoHandler) Pagar(c *gin.Context) {
	pedidoID, ok := idDaURL(c)
	if !ok {
		return
	}
	usuarioID := c.GetInt64(middleware.ChaveContextoUsuarioID)

	initPoint, err := h.service.IniciarPagamento(usuarioID, pedidoID)
	if err != nil {
		if errors.Is(err, service.ErrMercadoPagoNaoConfigurado) {
			c.JSON(http.StatusServiceUnavailable, gin.H{"erro": "pagamento não configurado"})
			return
		}
		h.responderErro(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"checkout_url": initPoint})
}

func (h *PedidoHandler) responderErro(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrPedidoNaoPertenceAoUsuario):
		c.JSON(http.StatusForbidden, gin.H{"erro": "pedido não pertence a este usuário"})
	case errors.Is(err, service.ErrPedidoNaoEncontrado):
		c.JSON(http.StatusNotFound, gin.H{"erro": "pedido não encontrado"})
	default:
		c.JSON(http.StatusUnprocessableEntity, gin.H{"erro": err.Error()})
	}
}
