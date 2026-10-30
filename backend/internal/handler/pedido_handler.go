package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
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
	TaxaAbsorvida      bool       `json:"taxa_absorvida"`
	DescontoCentavos   int64      `json:"desconto_centavos"`
	TotalCentavos      int64      `json:"total_centavos"`
	Status             string     `json:"status"`
	Codigo             string     `json:"codigo"`
	QRToken            string     `json:"qr_token,omitempty"`
	UtilizadoEm        *time.Time `json:"utilizado_em"`
	Lugar              string     `json:"lugar,omitempty"`     // item 9.5
	Adicional          bool       `json:"adicional,omitempty"` // item 10.3
	PacoteID           *int64     `json:"pacote_id,omitempty"`
}

func paraItemPedidoResposta(i *domain.ItemPedido) itemPedidoResposta {
	return itemPedidoResposta{
		ID: i.ID, TipoIngressoID: i.TipoIngressoID, TitularNome: i.TitularNome, TitularEmail: i.TitularEmail,
		PrecoCentavos: i.PrecoCentavos, TaxaCentavos: i.TaxaPlataformaCentavos,
		GarantiaContratada: i.GarantiaContratada, GarantiaCentavos: i.GarantiaCentavos, TaxaAbsorvida: i.TaxaAbsorvida, DescontoCentavos: i.DescontoCentavos, TotalCentavos: i.TotalCentavos,
		Status: string(i.Status), Codigo: i.Codigo, QRToken: i.QRToken, UtilizadoEm: i.UtilizadoEm, Lugar: i.LugarRotulo,
		Adicional: i.Adicional, PacoteID: i.PacoteID,
	}
}

type pedidoResposta struct {
	ID            int64                `json:"id"`
	EventoID      int64                `json:"evento_id"`
	Status        string               `json:"status"`
	TotalCentavos int64                `json:"total_centavos"`
	ExpiraEm      *time.Time           `json:"expira_em"`
	Itens         []itemPedidoResposta `json:"itens"`
	Pixels        *service.Pixels      `json:"pixels,omitempty"`
	// Item 7.6: até quantas vezes o cartão pode dividir este pedido.
	MaxParcelas int `json:"max_parcelas"`
	// Endereço do evento, para o botão "Voltar ao evento".
	EventoSlug string `json:"evento_slug,omitempty"`
	// Item 8.4: pedido da inscrição paga de talento (não é ingresso).
	Inscricao bool `json:"inscricao,omitempty"`
	// Item 8.6: pedido do espaço de expositor.
	Espaco bool `json:"espaco,omitempty"`
}

func paraPedidoResposta(p *domain.Pedido, itens []domain.ItemPedido) pedidoResposta {
	itensResp := make([]itemPedidoResposta, 0, len(itens))
	inscricao, espaco := false, false
	for i := range itens {
		inscricao = inscricao || itens[i].FichaID != nil
		espaco = espaco || itens[i].ExpositorID != nil
		itensResp = append(itensResp, paraItemPedidoResposta(&itens[i]))
	}
	return pedidoResposta{
		ID: p.ID, EventoID: p.EventoID, Status: string(p.Status), TotalCentavos: p.TotalCentavos,
		ExpiraEm: p.ExpiraEm, MaxParcelas: service.MaxParcelas(p.TotalCentavos), Itens: itensResp,
		Inscricao: inscricao, Espaco: espaco,
	}
}

type itemRequest struct {
	TipoIngressoID     int64 `json:"tipo_ingresso_id"`
	Quantidade         int   `json:"quantidade" binding:"required,gt=0"`
	GarantiaContratada bool  `json:"garantia_contratada"`
	// Item 7.6: [{"<pergunta_id>": "resposta"}, ...], um objeto por ingresso.
	Respostas []map[string]string `json:"respostas"`
	// Item 9.5: cadeiras escolhidas no mapa, uma por ingresso.
	LugarIDs []int64 `json:"lugar_ids"`
	// Item 10.3: pacote (no lugar do tipo) e parte de um grupo de compra.
	PacoteID   int64  `json:"pacote_id"`
	GrupoToken string `json:"grupo_token"`
}

// respostasPorUnidade converte as chaves (ids de pergunta em texto).
func respostasPorUnidade(rs []map[string]string) []map[int64]string {
	out := make([]map[int64]string, len(rs))
	for i, r := range rs {
		out[i] = map[int64]string{}
		for k, v := range r {
			if id, err := strconv.ParseInt(k, 10, 64); err == nil {
				out[i][id] = v
			}
		}
	}
	return out
}

type criarPedidoRequest struct {
	Itens []itemRequest `json:"itens" binding:"required,min=1,dive"`
	Cupom string        `json:"cupom"`
	Ref   string        `json:"ref"`
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
			Respostas: respostasPorUnidade(ir.Respostas), LugarIDs: ir.LugarIDs,
			PacoteID: ir.PacoteID, GrupoToken: ir.GrupoToken,
		}
		if ir.TipoIngressoID == 0 && ir.PacoteID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"erro": "dados inválidos"})
			return
		}
	}

	pedido, itens, err := h.service.Reservar(usuarioID, evento.ID, itensReq, comprador.Nome, comprador.Email, req.Cupom, req.Ref)
	if errors.Is(err, service.ErrGrupoJaParticipa) || errors.Is(err, service.ErrGrupoCompleto) || errors.Is(err, service.ErrGrupoEncerrado) {
		c.JSON(http.StatusConflict, gin.H{"erro": err.Error()})
		return
	}
	if errors.Is(err, service.ErrLugarOcupado) {
		// 409: a tela recarrega o mapa e pede outra cadeira.
		c.JSON(http.StatusConflict, gin.H{"erro": err.Error()})
		return
	}
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
	resp := paraPedidoResposta(pedido, itens)
	// Item 7.5: a página do pedido dispara a "compra concluída" nos pixels
	// do evento (se o visitante aceitou os cookies).
	if e, err := h.eventos.BuscarPorID(pedido.EventoID); err == nil {
		resp.Pixels = &service.Pixels{Meta: e.PixelMeta, GA4: e.PixelGA4, TikTok: e.PixelTikTok}
		resp.EventoSlug = e.Slug
	}
	c.JSON(http.StatusOK, resp)
}

// Pagar é POST /pedidos/:id/pagar — cria a cobrança no processador e
// devolve o link de checkout para o frontend redirecionar o comprador.
func (h *PedidoHandler) Pagar(c *gin.Context) {
	pedidoID, ok := idDaURL(c)
	if !ok {
		return
	}
	usuarioID := c.GetInt64(middleware.ChaveContextoUsuarioID)

	// Corpo opcional: {"metodo": "pix" | "cartao"} — padrão Pix (item 5.2).
	var corpo struct {
		Metodo string `json:"metodo"`
	}
	_ = c.ShouldBindJSON(&corpo)

	initPoint, err := h.service.IniciarPagamento(usuarioID, pedidoID, corpo.Metodo)
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

// Pix é POST /pedidos/:id/pix: gera (ou devolve) o QR do Pix na própria
// página (item 7.1). A reserva passa a terminar junto com o QR.
func (h *PedidoHandler) Pix(c *gin.Context) {
	pedidoID, ok := idDaURL(c)
	if !ok {
		return
	}
	pix, err := h.service.GerarPix(c.GetInt64(middleware.ChaveContextoUsuarioID), pedidoID)
	if err != nil {
		h.responderErroPagamento(c, err)
		return
	}
	c.JSON(http.StatusOK, pix)
}

type cartaoRequest struct {
	Token           string `json:"token" binding:"required"`
	BandeiraID      string `json:"payment_method_id" binding:"required"`
	EmissorID       string `json:"issuer_id"`
	DocumentoTipo   string `json:"documento_tipo"`
	DocumentoNumero string `json:"documento_numero"`
	Parcelas        int    `json:"installments"`
}

// Cartao é POST /pedidos/:id/cartao com o token do formulário do Mercado
// Pago (o número do cartão nunca passa pelo Evve).
func (h *PedidoHandler) Cartao(c *gin.Context) {
	pedidoID, ok := idDaURL(c)
	if !ok {
		return
	}
	var req cartaoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "dados do cartão incompletos"})
		return
	}
	res, err := h.service.PagarCartao(c.GetInt64(middleware.ChaveContextoUsuarioID), pedidoID, service.DadosCartao{
		Token: req.Token, BandeiraID: req.BandeiraID, EmissorID: req.EmissorID,
		DocumentoTipo: req.DocumentoTipo, DocumentoNumero: req.DocumentoNumero, Parcelas: req.Parcelas,
	})
	if err != nil {
		h.responderErroPagamento(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// Verificar é POST /pedidos/:id/verificar {"pagamento_id": "..."}: reconsulta
// o pagamento no Mercado Pago (depois da confirmação do banco ou quando a
// notificação atrasa). Sem pagamento_id, confere o Pix do pedido.
func (h *PedidoHandler) Verificar(c *gin.Context) {
	pedidoID, ok := idDaURL(c)
	if !ok {
		return
	}
	var req struct {
		PagamentoID string `json:"pagamento_id"`
	}
	_ = c.ShouldBindJSON(&req)
	usuarioID := c.GetInt64(middleware.ChaveContextoUsuarioID)
	if _, err := h.service.VerificarPagamento(usuarioID, pedidoID, req.PagamentoID); err != nil {
		h.responderErroPagamento(c, err)
		return
	}
	pedido, itens, err := h.service.ObterPedido(usuarioID, pedidoID)
	if err != nil {
		h.responderErro(c, err)
		return
	}
	c.JSON(http.StatusOK, paraPedidoResposta(pedido, itens))
}

func (h *PedidoHandler) responderErroPagamento(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrMercadoPagoNaoConfigurado):
		c.JSON(http.StatusServiceUnavailable, gin.H{"erro": "pagamento não configurado"})
	case errors.Is(err, service.ErrCartaoBloqueado):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"erro": err.Error()})
	case errors.Is(err, service.ErrReservaExpirada):
		c.JSON(http.StatusGone, gin.H{"erro": "a reserva expirou. Volte ao evento para comprar de novo."})
	case errors.Is(err, service.ErrPedidoNaoDisponivelParaPagamento), errors.Is(err, service.ErrDadosCartaoIncompletos),
		errors.Is(err, service.ErrParcelasInvalidas),
		errors.Is(err, service.ErrPagamentoNaoEncontradoNoPedido):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"erro": err.Error()})
	case errors.Is(err, service.ErrPedidoNaoPertenceAoUsuario), errors.Is(err, service.ErrPedidoNaoEncontrado):
		h.responderErro(c, err)
	default:
		slog.Error("pagamento na página", "erro", err)
		c.JSON(http.StatusBadGateway, gin.H{"erro": "não foi possível falar com o Mercado Pago agora. Tente de novo em instantes."})
	}
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

// Item 8.4: inscrição paga de talento.

type valoresInscricaoResposta struct {
	PrecoCentavos      int64      `json:"preco_centavos"`
	TaxaCentavos       int64      `json:"taxa_centavos"`
	GarantiaCentavos   int64      `json:"garantia_centavos"`
	GarantiaDisponivel bool       `json:"garantia_disponivel"`
	PrazoEm            *time.Time `json:"prazo_em"`
	PedidoAbertoID     *int64     `json:"pedido_aberto_id"`
	Pago               bool       `json:"pago"`
	ItemPagoID         *int64     `json:"item_pago_id"`
	TotalPagoCentavos  int64      `json:"total_pago_centavos"`
	GarantiaContratada bool       `json:"garantia_contratada"`
}

func (h *PedidoHandler) eventoDoSlug(c *gin.Context) (int64, bool) {
	e, err := h.eventos.BuscarPorSlug(c.Param("slug"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"erro": "evento não encontrado"})
		return 0, false
	}
	return e.ID, true
}

func (h *PedidoHandler) responderErroInscricao(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrInscricaoSemPagamento):
		c.JSON(http.StatusNotFound, gin.H{"erro": err.Error()})
	case errors.Is(err, service.ErrPrazoInscricaoVencido), errors.Is(err, service.ErrGarantiaNaoDisponivel):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"erro": err.Error()})
	default:
		slog.Error("inscrição paga", "erro", err)
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao preparar o pagamento da inscrição"})
	}
}

// ValoresInscricao é GET /eventos/:slug/inscricao/pagamento.
func (h *PedidoHandler) ValoresInscricao(c *gin.Context) {
	eventoID, ok := h.eventoDoSlug(c)
	if !ok {
		return
	}
	v, err := h.service.ValoresInscricao(c.GetInt64(middleware.ChaveContextoUsuarioID), eventoID)
	if err != nil {
		h.responderErroInscricao(c, err)
		return
	}
	c.JSON(http.StatusOK, valoresInscricaoResposta{
		PrecoCentavos: v.PrecoCentavos, TaxaCentavos: v.TaxaCentavos, GarantiaCentavos: v.GarantiaCentavos,
		GarantiaDisponivel: v.GarantiaDisponivel, PrazoEm: v.PrazoEm, PedidoAbertoID: v.PedidoAbertoID,
		Pago: v.Pago, ItemPagoID: v.ItemPagoID, TotalPagoCentavos: v.TotalPagoCentavos, GarantiaContratada: v.GarantiaContratada,
	})
}

// PagarInscricao é POST /eventos/:slug/inscricao/pagamento: abre o pedido.
func (h *PedidoHandler) PagarInscricao(c *gin.Context) {
	eventoID, ok := h.eventoDoSlug(c)
	if !ok {
		return
	}
	var req struct {
		GarantiaContratada bool `json:"garantia_contratada"`
	}
	_ = c.ShouldBindJSON(&req)
	pedido, itens, err := h.service.ReservarInscricao(c.GetInt64(middleware.ChaveContextoUsuarioID), eventoID, req.GarantiaContratada)
	if err != nil {
		h.responderErroInscricao(c, err)
		return
	}
	c.JSON(http.StatusCreated, paraPedidoResposta(pedido, itens))
}
