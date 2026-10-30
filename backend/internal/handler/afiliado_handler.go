package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/WilliamBreno/Arcadia/backend/internal/middleware"
	"github.com/WilliamBreno/Arcadia/backend/internal/service"
)

type AfiliadoHandler struct {
	organizadorHandler *OrganizadorHandler
	service            *service.AfiliadoService
}

func NovoAfiliadoHandler(o *OrganizadorHandler, s *service.AfiliadoService) *AfiliadoHandler {
	return &AfiliadoHandler{organizadorHandler: o, service: s}
}

func (h *AfiliadoHandler) responderErro(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrEventoNaoPertenceAoOrganizador):
		c.JSON(http.StatusForbidden, gin.H{"erro": "evento não pertence a este organizador"})
	case errors.Is(err, service.ErrAfiliadoNaoEncontrado), errors.Is(err, service.ErrPainelDivulgador):
		c.JSON(http.StatusNotFound, gin.H{"erro": err.Error()})
	default:
		c.JSON(http.StatusUnprocessableEntity, gin.H{"erro": err.Error()})
	}
}

func (h *AfiliadoHandler) Listar(c *gin.Context) {
	org, ok := h.organizadorHandler.ObterOrganizadorAtual(c)
	if !ok {
		return
	}
	eventoID, ok := idDaURL(c)
	if !ok {
		return
	}
	lista, err := h.service.Listar(org.ID, eventoID)
	if err != nil {
		h.responderErro(c, err)
		return
	}
	resp := make([]gin.H, 0, len(lista))
	for _, a := range lista {
		item := gin.H{"id": a.ID, "nome": a.Nome, "codigo": a.Codigo, "ativo": a.Ativo,
			"pedidos": a.Pedidos, "itens": a.Itens, "receita_centavos": a.Receita,
			// Item 9.2: regra e saldo da comissão.
			"comissao_tipo": a.ComissaoTipo, "comissao_bp": a.ComissaoBP, "comissao_fixa_centavos": a.ComissaoFixaCentavos,
			"painel_ativo": a.PainelTokenHash != nil}
		if saldo, err := h.service.SaldoDe(eventoID, a.ID); err == nil {
			item["comissao_centavos"], item["pago_centavos"], item["a_pagar_centavos"], item["liberado"] =
				saldo.ComissaoCentavos, saldo.PagoCentavos, saldo.APagarCentavos, saldo.Liberado
		}
		resp = append(resp, item)
	}
	c.JSON(http.StatusOK, resp)
}

type criarAfiliadoRequest struct {
	Nome string `json:"nome" binding:"required"`
	regraRequest
}

// regraRequest (item 9.2): tipo nenhuma|percentual|fixo; percentual em
// pontos-base (1000 = 10%); fixo em centavos por ingresso.
type regraRequest struct {
	ComissaoTipo         string `json:"comissao_tipo"`
	ComissaoBP           int    `json:"comissao_bp"`
	ComissaoFixaCentavos int64  `json:"comissao_fixa_centavos"`
}

func (r regraRequest) regra() service.RegraComissao {
	t := r.ComissaoTipo
	if t == "" {
		t = "nenhuma"
	}
	return service.RegraComissao{Tipo: t, BP: r.ComissaoBP, FixaCentavos: r.ComissaoFixaCentavos}
}

func (h *AfiliadoHandler) Criar(c *gin.Context) {
	org, ok := h.organizadorHandler.ObterOrganizadorAtual(c)
	if !ok {
		return
	}
	eventoID, ok := idDaURL(c)
	if !ok {
		return
	}
	var req criarAfiliadoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "informe o nome"})
		return
	}
	a, err := h.service.Criar(org.ID, eventoID, req.Nome, req.regra())
	if err != nil {
		h.responderErro(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": a.ID, "nome": a.Nome, "codigo": a.Codigo})
}

func (h *AfiliadoHandler) Desativar(c *gin.Context) {
	org, ok := h.organizadorHandler.ObterOrganizadorAtual(c)
	if !ok {
		return
	}
	eventoID, ok := idDaURL(c)
	if !ok {
		return
	}
	afID, err := strconv.ParseInt(c.Param("afiliadoId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "id inválido"})
		return
	}
	if err := h.service.Desativar(org.ID, eventoID, afID); err != nil {
		h.responderErro(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *AfiliadoHandler) orgEAfiliado(c *gin.Context) (orgID, eventoID, afID int64, ok bool) {
	org, ok := h.organizadorHandler.ObterOrganizadorAtual(c)
	if !ok {
		return
	}
	eventoID, ok = idDaURL(c)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(c.Param("afiliadoId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "id inválido"})
		return 0, 0, 0, false
	}
	return org.ID, eventoID, id, true
}

// DefinirComissao é PUT /org/eventos/:id/afiliados/:afiliadoId/comissao (item 9.2).
func (h *AfiliadoHandler) DefinirComissao(c *gin.Context) {
	orgID, eventoID, afID, ok := h.orgEAfiliado(c)
	if !ok {
		return
	}
	var req regraRequest
	_ = c.ShouldBindJSON(&req)
	a, err := h.service.DefinirComissao(orgID, eventoID, afID, req.regra())
	if err != nil {
		h.responderErro(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"comissao_tipo": a.ComissaoTipo, "comissao_bp": a.ComissaoBP, "comissao_fixa_centavos": a.ComissaoFixaCentavos})
}

// RegistrarPagamento é POST /org/eventos/:id/afiliados/:afiliadoId/pagamentos.
func (h *AfiliadoHandler) RegistrarPagamento(c *gin.Context) {
	orgID, eventoID, afID, ok := h.orgEAfiliado(c)
	if !ok {
		return
	}
	var req struct {
		ValorCentavos int64  `json:"valor_centavos" binding:"required"`
		Observacao    string `json:"observacao"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "informe o valor pago"})
		return
	}
	if err := h.service.RegistrarPagamento(orgID, c.GetInt64(middleware.ChaveContextoUsuarioID), eventoID, afID, req.ValorCentavos, req.Observacao); err != nil {
		h.responderErro(c, err)
		return
	}
	c.Status(http.StatusCreated)
}

// Pagamentos é GET /org/eventos/:id/afiliados/:afiliadoId/pagamentos.
func (h *AfiliadoHandler) Pagamentos(c *gin.Context) {
	orgID, eventoID, afID, ok := h.orgEAfiliado(c)
	if !ok {
		return
	}
	ps, err := h.service.PagamentosDe(orgID, eventoID, afID)
	if err != nil {
		h.responderErro(c, err)
		return
	}
	c.JSON(http.StatusOK, paraPagamentos(ps))
}

func paraPagamentos(ps []service.PagamentoComissao) []gin.H {
	out := make([]gin.H, 0, len(ps))
	for _, p := range ps {
		out = append(out, gin.H{"valor_centavos": p.ValorCentavos, "observacao": p.Observacao, "pago_em": p.PagoEm})
	}
	return out
}

// LinkPainel é POST /org/eventos/:id/afiliados/:afiliadoId/painel: gera
// (ou troca) o link secreto do divulgador.
func (h *AfiliadoHandler) LinkPainel(c *gin.Context) {
	orgID, eventoID, afID, ok := h.orgEAfiliado(c)
	if !ok {
		return
	}
	token, err := h.service.GerarLinkPainel(orgID, eventoID, afID)
	if err != nil {
		h.responderErro(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"token": token})
}

// PainelDivulgador é GET /divulgador/:token (sem login; nada de compradores).
func (h *AfiliadoHandler) PainelDivulgador(c *gin.Context) {
	p, err := h.service.Painel(c.Param("token"))
	if err != nil {
		h.responderErro(c, err)
		return
	}
	c.Header("X-Robots-Tag", "noindex, nofollow")
	c.JSON(http.StatusOK, gin.H{
		"nome": p.Nome, "evento_titulo": p.EventoTitulo, "evento_slug": p.EventoSlug, "inicio_em": p.InicioEm, "codigo": p.Codigo,
		"comissao_tipo": p.Regra.Tipo, "comissao_bp": p.Regra.BP, "comissao_fixa_centavos": p.Regra.FixaCentavos,
		"ingressos": p.Saldo.Ingressos, "comissao_centavos": p.Saldo.ComissaoCentavos, "pago_centavos": p.Saldo.PagoCentavos,
		"a_pagar_centavos": p.Saldo.APagarCentavos, "liberado": p.Saldo.Liberado, "pagamentos": paraPagamentos(p.Pagamentos),
	})
}
