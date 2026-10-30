package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/WilliamBreno/Arcadia/backend/internal/middleware"
	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
	"github.com/WilliamBreno/Arcadia/backend/internal/service"
)

// Adicionais, pacotes e compra em grupo (item 10.3).
type ExtrasHandler struct {
	svc     *service.CheckoutService
	checkin *service.CheckinService
	eventos *repository.EventoRepository
	org     *OrganizadorHandler
}

func NovoExtrasHandler(s *service.CheckoutService, c *service.CheckinService, e *repository.EventoRepository, org *OrganizadorHandler) *ExtrasHandler {
	return &ExtrasHandler{svc: s, checkin: c, eventos: e, org: org}
}

func (h *ExtrasHandler) erro(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrEventoNaoPertenceAoOrganizador), errors.Is(err, service.ErrSemAcessoCheckin):
		c.JSON(http.StatusForbidden, gin.H{"erro": "sem acesso a este evento"})
	case errors.Is(err, service.ErrAdicionalNaoEncontrado), errors.Is(err, service.ErrGrupoInvalido), errors.Is(err, service.ErrPacoteIndisponivel):
		c.JSON(http.StatusNotFound, gin.H{"erro": err.Error()})
	case errors.Is(err, service.ErrAdicionalNaoPago):
		c.JSON(http.StatusConflict, gin.H{"erro": err.Error()})
	case errors.Is(err, service.ErrDadosAdicional), errors.Is(err, service.ErrDadosPacote), errors.Is(err, service.ErrEstoqueInsuficiente),
		errors.Is(err, service.ErrQuantidadeInvalida), errors.Is(err, service.ErrTipoIndisponivel), errors.Is(err, service.ErrVendasEncerradas),
		errors.Is(err, service.ErrEventoNaoDisponivelParaCompra), errors.Is(err, service.ErrPrecisaAprovacao):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"erro": err.Error()})
	default:
		slog.Error("adicionais/pacotes/grupos", "erro", err)
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao processar"})
	}
}

// ---------- público ----------

// Extras é GET /eventos/:slug/extras: adicionais e pacotes à venda.
func (h *ExtrasHandler) Extras(c *gin.Context) {
	e, err := h.eventos.BuscarPorSlug(c.Param("slug"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"erro": "evento não encontrado"})
		return
	}
	adicionais, err := h.svc.Adicionais(e.ID, true)
	if err != nil {
		h.erro(c, err)
		return
	}
	pacotes, err := h.svc.Pacotes(e.ID, true)
	if err != nil {
		h.erro(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"adicionais": adicionais, "pacotes": pacotes})
}

// CriarGrupo é POST /eventos/:slug/grupos {tipo_ingresso_id, quantidade}.
func (h *ExtrasHandler) CriarGrupo(c *gin.Context) {
	e, err := h.eventos.BuscarPorSlug(c.Param("slug"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"erro": "evento não encontrado"})
		return
	}
	var req struct {
		TipoIngressoID int64 `json:"tipo_ingresso_id"`
		Quantidade     int   `json:"quantidade"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "dados inválidos"})
		return
	}
	g, err := h.svc.CriarGrupo(c.GetInt64(middleware.ChaveContextoUsuarioID), e.ID, req.TipoIngressoID, req.Quantidade)
	if err != nil {
		if errors.Is(err, service.ErrGrupoInvalido) {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"erro": err.Error()})
			return
		}
		h.erro(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"token": g.Token, "expira_em": g.ExpiraEm})
}

// VerGrupo é GET /grupos/:token (com ou sem login).
func (h *ExtrasHandler) VerGrupo(c *gin.Context) {
	v, err := h.svc.VerGrupo(c.Param("token"), c.GetInt64(middleware.ChaveContextoUsuarioID))
	if err != nil {
		h.erro(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, v)
}

// MeusGrupos é GET /me/grupos.
func (h *ExtrasHandler) MeusGrupos(c *gin.Context) {
	l, err := h.svc.MeusGrupos(c.GetInt64(middleware.ChaveContextoUsuarioID))
	if err != nil {
		h.erro(c, err)
		return
	}
	c.JSON(http.StatusOK, l)
}

// CancelarGrupo é POST /grupos/:token/cancelar (quem criou).
func (h *ExtrasHandler) CancelarGrupo(c *gin.Context) {
	if err := h.svc.CancelarGrupo(c.GetInt64(middleware.ChaveContextoUsuarioID), c.Param("token")); err != nil {
		h.erro(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// MeusAdicionais é GET /me/adicionais.
func (h *ExtrasHandler) MeusAdicionais(c *gin.Context) {
	l, err := h.svc.MeusAdicionais(c.GetInt64(middleware.ChaveContextoUsuarioID))
	if err != nil {
		h.erro(c, err)
		return
	}
	c.JSON(http.StatusOK, l)
}

// ---------- portaria ----------

func (h *ExtrasHandler) acessoCheckin(c *gin.Context) (int64, bool) {
	id, ok := idDaURL(c)
	if !ok {
		return 0, false
	}
	if !h.checkin.TemAcesso(c.GetInt64(middleware.ChaveContextoUsuarioID), id) {
		c.JSON(http.StatusForbidden, gin.H{"erro": "você não tem acesso ao check-in deste evento"})
		return 0, false
	}
	return id, true
}

// Entregar é POST /checkin/eventos/:id/adicionais/entregar {texto} (QR
// EVVEA:...) ou {codigo} (busca manual).
func (h *ExtrasHandler) Entregar(c *gin.Context) {
	id, ok := h.acessoCheckin(c)
	if !ok {
		return
	}
	var req struct {
		Texto  string `json:"texto"`
		Codigo string `json:"codigo"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || (req.Texto == "" && req.Codigo == "") {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "dados inválidos"})
		return
	}
	texto, comToken := req.Texto, true
	if texto == "" {
		texto, comToken = req.Codigo, false
	}
	r, err := h.svc.EntregarAdicional(id, texto, comToken)
	if err != nil {
		h.erro(c, err)
		return
	}
	c.JSON(http.StatusOK, r)
}

// BuscarAdicionais é GET /checkin/eventos/:id/adicionais?q=.
func (h *ExtrasHandler) BuscarAdicionais(c *gin.Context) {
	id, ok := h.acessoCheckin(c)
	if !ok {
		return
	}
	l, err := h.svc.BuscarAdicionais(id, c.Query("q"))
	if err != nil {
		h.erro(c, err)
		return
	}
	c.JSON(http.StatusOK, l)
}

// ---------- organizador ----------

func (h *ExtrasHandler) orgEvento(c *gin.Context) (int64, int64, bool) {
	org, ok := h.org.ObterOrganizadorAtual(c)
	if !ok {
		return 0, 0, false
	}
	id, ok := idDaURL(c)
	return org.ID, id, ok
}

// OrgListar é GET /org/eventos/:id/extras (adicionais e pacotes, inclusive inativos).
func (h *ExtrasHandler) OrgListar(c *gin.Context) {
	orgID, id, ok := h.orgEvento(c)
	if !ok {
		return
	}
	e, err := h.eventos.BuscarPorID(id)
	if err != nil || e.OrganizadorID != orgID {
		h.erro(c, service.ErrEventoNaoPertenceAoOrganizador)
		return
	}
	adicionais, err := h.svc.Adicionais(id, false)
	if err != nil {
		h.erro(c, err)
		return
	}
	pacotes, err := h.svc.Pacotes(id, false)
	if err != nil {
		h.erro(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"adicionais": adicionais, "pacotes": pacotes})
}

type adicionalRequest struct {
	Nome          string `json:"nome"`
	Descricao     string `json:"descricao"`
	PrecoCentavos int64  `json:"preco_centavos"`
	Quantidade    int    `json:"quantidade"`
	MaxPorPedido  int    `json:"max_por_pedido"`
	Ativo         bool   `json:"ativo"`
}

// SalvarAdicional é POST /org/eventos/:id/adicionais e PUT .../adicionais/:aid.
func (h *ExtrasHandler) SalvarAdicional(c *gin.Context) {
	orgID, id, ok := h.orgEvento(c)
	if !ok {
		return
	}
	var aid int64
	if c.Param("aid") != "" {
		if aid, ok = paramInt(c, "aid"); !ok {
			return
		}
	}
	var req adicionalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "dados inválidos"})
		return
	}
	novo, err := h.svc.SalvarAdicional(orgID, id, aid, service.DadosAdicional{Nome: req.Nome, Descricao: req.Descricao,
		PrecoCentavos: req.PrecoCentavos, Quantidade: req.Quantidade, MaxPorPedido: req.MaxPorPedido, Ativo: req.Ativo})
	if err != nil {
		h.erro(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": novo})
}

// ExcluirAdicional é DELETE /org/eventos/:id/adicionais/:aid.
func (h *ExtrasHandler) ExcluirAdicional(c *gin.Context) {
	orgID, id, ok := h.orgEvento(c)
	if !ok {
		return
	}
	aid, ok := paramInt(c, "aid")
	if !ok {
		return
	}
	if err := h.svc.ExcluirAdicional(orgID, id, aid); err != nil {
		h.erro(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

type pacoteRequest struct {
	Nome          string `json:"nome"`
	Descricao     string `json:"descricao"`
	PrecoCentavos int64  `json:"preco_centavos"`
	Quantidade    int    `json:"quantidade"`
	MaxPorPedido  int    `json:"max_por_pedido"`
	Ativo         bool   `json:"ativo"`
	Itens         []struct {
		TipoIngressoID int64 `json:"tipo_ingresso_id"`
		Quantidade     int   `json:"quantidade"`
	} `json:"itens"`
}

// SalvarPacote é POST /org/eventos/:id/pacotes e PUT .../pacotes/:pid.
func (h *ExtrasHandler) SalvarPacote(c *gin.Context) {
	orgID, id, ok := h.orgEvento(c)
	if !ok {
		return
	}
	var pid int64
	if c.Param("pid") != "" {
		if pid, ok = paramInt(c, "pid"); !ok {
			return
		}
	}
	var req pacoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "dados inválidos"})
		return
	}
	d := service.DadosPacote{Nome: req.Nome, Descricao: req.Descricao, PrecoCentavos: req.PrecoCentavos, Quantidade: req.Quantidade,
		MaxPorPedido: req.MaxPorPedido, Ativo: req.Ativo}
	for _, it := range req.Itens {
		d.Itens = append(d.Itens, service.PacoteItem{TipoIngressoID: it.TipoIngressoID, Quantidade: it.Quantidade})
	}
	novo, err := h.svc.SalvarPacote(orgID, id, pid, d)
	if err != nil {
		h.erro(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": novo})
}

// ExcluirPacote é DELETE /org/eventos/:id/pacotes/:pid.
func (h *ExtrasHandler) ExcluirPacote(c *gin.Context) {
	orgID, id, ok := h.orgEvento(c)
	if !ok {
		return
	}
	pid, ok := paramInt(c, "pid")
	if !ok {
		return
	}
	if err := h.svc.ExcluirPacote(orgID, id, pid); err != nil {
		h.erro(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
