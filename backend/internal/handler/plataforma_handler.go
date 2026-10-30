package handler

import (
	"bytes"
	"encoding/csv"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/WilliamBreno/Arcadia/backend/internal/middleware"
	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
	"github.com/WilliamBreno/Arcadia/backend/internal/service"
)

// Plano Pro, destaque na vitrine e as cobranças da Evve (item 10.4).
type PlataformaHandler struct {
	svc      *service.PlataformaService
	org      *OrganizadorHandler
	itens    *repository.ItemPedidoRepository
	publicos *PublicoHandler
}

func NovoPlataformaHandler(s *service.PlataformaService, org *OrganizadorHandler, itens *repository.ItemPedidoRepository, p *PublicoHandler) *PlataformaHandler {
	return &PlataformaHandler{svc: s, org: org, itens: itens, publicos: p}
}

func (h *PlataformaHandler) erro(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrEventoNaoPertenceAoOrganizador):
		c.JSON(http.StatusForbidden, gin.H{"erro": "sem acesso a este evento"})
	case errors.Is(err, service.ErrCobrancaNaoEncontrada):
		c.JSON(http.StatusNotFound, gin.H{"erro": err.Error()})
	case errors.Is(err, service.ErrCobrancaPaga), errors.Is(err, service.ErrCobrancaExpirada), errors.Is(err, service.ErrDestaqueLotado):
		c.JSON(http.StatusConflict, gin.H{"erro": err.Error()})
	case errors.Is(err, service.ErrDadosDestaque), errors.Is(err, service.ErrDadosCartaoIncompletos):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"erro": err.Error()})
	default:
		slog.Error("plataforma", "erro", err)
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao processar a cobrança"})
	}
}

type cobrancaResposta struct {
	ID            int64      `json:"id"`
	Tipo          string     `json:"tipo"`
	Descricao     string     `json:"descricao"`
	ValorCentavos int64      `json:"valor_centavos"`
	Status        string     `json:"status"`
	ExpiraEm      time.Time  `json:"expira_em"`
	PagoEm        *time.Time `json:"pago_em"`
}

func paraCobranca(c *service.CobrancaPlataforma) cobrancaResposta {
	status := c.Status
	if status == "pendente" && !time.Now().Before(c.ExpiraEm) {
		status = "expirada"
	}
	return cobrancaResposta{ID: c.ID, Tipo: c.Tipo, Descricao: c.Descricao, ValorCentavos: c.ValorCentavos, Status: status, ExpiraEm: c.ExpiraEm, PagoEm: c.PagoEm}
}

// Situacao é GET /org/pro.
func (h *PlataformaHandler) Situacao(c *gin.Context) {
	org, ok := h.org.ObterOrganizadorAtual(c)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, h.svc.Situacao(org.ID))
}

// ComprarPro é POST /org/pro/cobrancas {anual}.
func (h *PlataformaHandler) ComprarPro(c *gin.Context) {
	org, ok := h.org.ObterOrganizadorDono(c)
	if !ok {
		return
	}
	var req struct {
		Anual bool `json:"anual"`
	}
	_ = c.ShouldBindJSON(&req)
	cb, err := h.svc.CriarCobrancaPro(org.ID, c.GetInt64(middleware.ChaveContextoUsuarioID), req.Anual)
	if err != nil {
		h.erro(c, err)
		return
	}
	c.JSON(http.StatusCreated, paraCobranca(cb))
}

// ComprarDestaque é POST /org/destaques {evento_id, alvo_tipo, dias}.
func (h *PlataformaHandler) ComprarDestaque(c *gin.Context) {
	org, ok := h.org.ObterOrganizadorDono(c)
	if !ok {
		return
	}
	var req struct {
		EventoID int64    `json:"evento_id"`
		AlvoTipo string   `json:"alvo_tipo"`
		Dias     []string `json:"dias"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "dados inválidos"})
		return
	}
	cb, err := h.svc.CriarCobrancaDestaque(org.ID, c.GetInt64(middleware.ChaveContextoUsuarioID),
		service.DadosDestaque{EventoID: req.EventoID, AlvoTipo: req.AlvoTipo, Dias: req.Dias})
	if err != nil {
		h.erro(c, err)
		return
	}
	c.JSON(http.StatusCreated, paraCobranca(cb))
}

// MeusDestaques é GET /org/destaques.
func (h *PlataformaHandler) MeusDestaques(c *gin.Context) {
	org, ok := h.org.ObterOrganizadorAtual(c)
	if !ok {
		return
	}
	l, err := h.svc.MeusDestaques(org.ID)
	if err != nil {
		h.erro(c, err)
		return
	}
	c.JSON(http.StatusOK, l)
}

// Disponibilidade é GET /org/eventos/:id/destaques?alvo=cidade|categoria.
func (h *PlataformaHandler) Disponibilidade(c *gin.Context) {
	org, ok := h.org.ObterOrganizadorAtual(c)
	if !ok {
		return
	}
	id, ok := idDaURL(c)
	if !ok {
		return
	}
	d, err := h.svc.Disponibilidade(org.ID, id, c.Query("alvo"))
	if err != nil {
		h.erro(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"vagas": d, "preco_dia_centavos": h.svc.Precos().DestaqueDia})
}

func (h *PlataformaHandler) cobrancaID(c *gin.Context) (int64, int64, bool) {
	org, ok := h.org.ObterOrganizadorDono(c)
	if !ok {
		return 0, 0, false
	}
	id, ok := idDaURL(c)
	return org.ID, id, ok
}

// Cobranca é GET /org/cobrancas/:id.
func (h *PlataformaHandler) Cobranca(c *gin.Context) {
	orgID, id, ok := h.cobrancaID(c)
	if !ok {
		return
	}
	cb, err := h.svc.Cobranca(orgID, id)
	if err != nil {
		h.erro(c, err)
		return
	}
	c.JSON(http.StatusOK, paraCobranca(cb))
}

// Pix é POST /org/cobrancas/:id/pix.
func (h *PlataformaHandler) Pix(c *gin.Context) {
	orgID, id, ok := h.cobrancaID(c)
	if !ok {
		return
	}
	p, err := h.svc.GerarPix(orgID, id)
	if err != nil {
		h.erro(c, err)
		return
	}
	c.JSON(http.StatusOK, p)
}

// Cartao é POST /org/cobrancas/:id/cartao (mesmo corpo do pedido).
func (h *PlataformaHandler) Cartao(c *gin.Context) {
	orgID, id, ok := h.cobrancaID(c)
	if !ok {
		return
	}
	var req struct {
		Token           string `json:"token"`
		PaymentMethodID string `json:"payment_method_id"`
		IssuerID        string `json:"issuer_id"`
		DocumentoTipo   string `json:"documento_tipo"`
		DocumentoNumero string `json:"documento_numero"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "dados inválidos"})
		return
	}
	r, err := h.svc.PagarCartao(orgID, id, service.DadosCartao{Token: req.Token, BandeiraID: req.PaymentMethodID, EmissorID: req.IssuerID,
		DocumentoTipo: req.DocumentoTipo, DocumentoNumero: req.DocumentoNumero})
	if err != nil {
		h.erro(c, err)
		return
	}
	c.JSON(http.StatusOK, r)
}

// Verificar é POST /org/cobrancas/:id/verificar {pagamento_id}.
func (h *PlataformaHandler) Verificar(c *gin.Context) {
	orgID, id, ok := h.cobrancaID(c)
	if !ok {
		return
	}
	var req struct {
		PagamentoID string `json:"pagamento_id"`
	}
	_ = c.ShouldBindJSON(&req)
	cb, err := h.svc.Verificar(orgID, id, req.PagamentoID)
	if err != nil {
		h.erro(c, err)
		return
	}
	r := paraCobranca(cb)
	// O componente de pagamento fala a língua do pedido: "pago" quando pago.
	status := "aguardando_pagamento"
	if cb.Status == "paga" {
		status = "pago"
	}
	c.JSON(http.StatusOK, gin.H{"status": status, "cobranca": r})
}

// RelatoriosCSV é GET /org/relatorios.csv (Pro): todos os eventos em planilha.
func (h *PlataformaHandler) RelatoriosCSV(c *gin.Context) {
	org, ok := h.org.ObterOrganizadorFinanceiro(c)
	if !ok {
		return
	}
	if !h.svc.Situacao(org.ID).Pro {
		c.JSON(http.StatusForbidden, gin.H{"erro": "a planilha de relatórios é do Evve Pro"})
		return
	}
	linhas, err := h.itens.RelatorioPorEvento(org.ID)
	if err != nil {
		h.erro(c, err)
		return
	}
	var buf bytes.Buffer
	buf.Write([]byte{0xEF, 0xBB, 0xBF})
	w := csv.NewWriter(&buf)
	w.Comma = ';'
	_ = w.Write([]string{"evento", "situacao", "vendidos", "cortesias", "receita_bruta_reais", "check-ins", "cancelados"})
	for _, l := range linhas {
		_ = w.Write([]string{celulaSegura(l.Titulo), l.Status, strconv.FormatInt(l.Vendidos, 10), strconv.FormatInt(l.Cortesias, 10),
			strconv.FormatFloat(float64(l.Receita)/100, 'f', 2, 64), strconv.FormatInt(l.Checkins, 10), strconv.FormatInt(l.Cancelados, 10)})
	}
	w.Flush()
	c.Header("Content-Disposition", `attachment; filename="relatorio-eventos.csv"`)
	c.Data(http.StatusOK, "text/csv; charset=utf-8", buf.Bytes())
}

// celulaSegura impede fórmula na planilha (=, +, -, @ no começo).
func celulaSegura(s string) string {
	if s != "" && (s[0] == '=' || s[0] == '+' || s[0] == '-' || s[0] == '@') {
		return "'" + s
	}
	return s
}

// Patrocinados é GET /patrocinados?cidade=<slug>&categoria=<slug>.
func (h *PlataformaHandler) Patrocinados(c *gin.Context) {
	tipo, valor := "cidade", c.Query("cidade")
	if valor == "" {
		tipo, valor = "categoria", c.Query("categoria")
	}
	if valor == "" {
		c.JSON(http.StatusOK, []eventoPublicoItem{})
		return
	}
	eventos, err := h.svc.Patrocinados(tipo, valor, 3)
	if err != nil {
		h.erro(c, err)
		return
	}
	itens, err := h.publicos.paraItensPublicos(eventos, true)
	if err != nil {
		h.erro(c, err)
		return
	}
	c.Header("Cache-Control", "public, max-age=300")
	c.JSON(http.StatusOK, itens)
}

// AvisarVencimento é o job POST /jobs/pro-vencimento (diário).
func (h *PlataformaHandler) AvisarVencimento(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"avisados": h.svc.AvisarVencimentoPro(time.Now())})
}
