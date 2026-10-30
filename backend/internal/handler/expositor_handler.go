package handler

import (
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

// Artist alley e expositores (item 8.6).
type ExpositorHandler struct {
	svc     *service.ExpositorService
	eventos *repository.EventoRepository
	org     *OrganizadorHandler
}

func NovoExpositorHandler(s *service.ExpositorService, e *repository.EventoRepository, org *OrganizadorHandler) *ExpositorHandler {
	return &ExpositorHandler{svc: s, eventos: e, org: org}
}

func (h *ExpositorHandler) erro(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrExpositorNaoEncontrado), errors.Is(err, service.ErrEspacoSemPagamento):
		c.JSON(http.StatusNotFound, gin.H{"erro": err.Error()})
	case errors.Is(err, service.ErrEventoNaoPertenceAoOrganizador):
		c.JSON(http.StatusForbidden, gin.H{"erro": "sem acesso a este evento"})
	case errors.Is(err, service.ErrJaCandidatado), errors.Is(err, service.ErrEspacoOcupado), errors.Is(err, service.ErrTipoEspacoEmUso):
		c.JSON(http.StatusConflict, gin.H{"erro": err.Error()})
	case errors.Is(err, service.ErrExpositoresFechados), errors.Is(err, service.ErrDadosExpositor), errors.Is(err, service.ErrEspacoInvalido),
		errors.Is(err, service.ErrQRCredencial), errors.Is(err, service.ErrPrazoInscricaoVencido), errors.Is(err, service.ErrGarantiaNaoDisponivel):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"erro": err.Error()})
	default:
		slog.Error("expositores", "erro", err)
		c.JSON(http.StatusUnprocessableEntity, gin.H{"erro": err.Error()})
	}
}

type tipoEspacoResposta struct {
	ID            int64  `json:"id"`
	Nome          string `json:"nome"`
	Descricao     string `json:"descricao"`
	PrecoCentavos int64  `json:"preco_centavos"`
	Credenciais   int    `json:"credenciais"`
	Ativo         bool   `json:"ativo"`
}

func paraTipoEspaco(t service.TipoEspaco) tipoEspacoResposta {
	return tipoEspacoResposta{ID: t.ID, Nome: t.Nome, Descricao: t.Descricao, PrecoCentavos: t.PrecoCentavos, Credenciais: t.Credenciais, Ativo: t.Ativo}
}

// ---------- público ----------

// Publico é GET /eventos/:slug/expositores: mapa e lista de quem expõe.
func (h *ExpositorHandler) Publico(c *gin.Context) {
	e, err := h.eventos.BuscarPublicoPorSlug(c.Param("slug"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"erro": "evento não encontrado"})
		return
	}
	p, err := h.svc.Publico(e)
	if err != nil {
		h.erro(c, err)
		return
	}
	type mesa struct {
		Codigo  string `json:"codigo"`
		Setor   string `json:"setor"`
		Tipo    string `json:"tipo"`
		Ocupada bool   `json:"ocupada"`
		Marca   string `json:"marca,omitempty"`
	}
	type expositor struct {
		Marca     string `json:"marca"`
		Categoria string `json:"categoria"`
		Descricao string `json:"descricao"`
		Instagram string `json:"instagram"`
		Mesa      string `json:"mesa"`
	}
	tipos := make([]tipoEspacoResposta, 0, len(p.Tipos))
	for _, t := range p.Tipos {
		tipos = append(tipos, paraTipoEspaco(t))
	}
	mesas := make([]mesa, 0, len(p.Mesas))
	for _, m := range p.Mesas {
		mesas = append(mesas, mesa{Codigo: m.Codigo, Setor: m.Setor, Tipo: m.Tipo, Ocupada: m.Ocupada, Marca: m.Marca})
	}
	exps := make([]expositor, 0, len(p.Expositores))
	for _, x := range p.Expositores {
		exps = append(exps, expositor{Marca: x.Marca, Categoria: x.Categoria, Descricao: x.Descricao, Instagram: x.Instagram, Mesa: x.Mesa})
	}
	c.JSON(http.StatusOK, gin.H{"aberto": p.Aberto, "tipos": tipos, "mesas": mesas, "expositores": exps})
}

// ---------- expositor ----------

type candidaturaRequest struct {
	TipoEspacoID    int64  `json:"tipo_espaco_id" binding:"required"`
	Marca           string `json:"marca" binding:"required"`
	Categoria       string `json:"categoria"`
	Descricao       string `json:"descricao"`
	PortfolioURL    string `json:"portfolio_url"`
	Instagram       string `json:"instagram"`
	ResponsavelNome string `json:"responsavel_nome" binding:"required"`
	Telefone        string `json:"telefone"`
}

// Candidatar é POST /eventos/:slug/expositores.
func (h *ExpositorHandler) Candidatar(c *gin.Context) {
	e, err := h.eventos.BuscarPublicoPorSlug(c.Param("slug"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"erro": "evento não encontrado"})
		return
	}
	var req candidaturaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "preencha o tipo de espaço, a marca e o responsável"})
		return
	}
	x, err := h.svc.Candidatar(c.GetInt64(middleware.ChaveContextoUsuarioID), e, service.DadosExpositor{
		TipoEspacoID: req.TipoEspacoID, Marca: req.Marca, Categoria: req.Categoria, Descricao: req.Descricao,
		PortfolioURL: req.PortfolioURL, Instagram: req.Instagram, ResponsavelNome: req.ResponsavelNome, Telefone: req.Telefone,
	})
	if err != nil {
		h.erro(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": x.ID, "status": x.Status})
}

type credencialResposta struct {
	ID       int64      `json:"id"`
	Nome     string     `json:"nome"`
	Payload  string     `json:"payload"`
	EntrouEm *time.Time `json:"entrou_em"`
}

// Meu é GET /eventos/:slug/meu-espaco.
func (h *ExpositorHandler) Meu(c *gin.Context) {
	e, err := h.eventos.BuscarPorSlug(c.Param("slug"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"erro": "evento não encontrado"})
		return
	}
	m, err := h.svc.Meu(c.GetInt64(middleware.ChaveContextoUsuarioID), e)
	if err != nil {
		h.erro(c, err)
		return
	}
	out := gin.H{
		"id": m.Expositor.ID, "status": m.Expositor.Status, "marca": m.Expositor.Marca, "motivo_rejeicao": m.Expositor.MotivoRejeicao,
		"tipo": paraTipoEspaco(m.Tipo), "mesa": nil, "valores": nil, "credenciais": []credencialResposta{},
	}
	if m.Mesa != nil {
		out["mesa"] = m.Mesa.Codigo
	}
	if v := m.Valores; v != nil {
		out["valores"] = valoresInscricaoResposta{
			PrecoCentavos: v.PrecoCentavos, TaxaCentavos: v.TaxaCentavos, GarantiaCentavos: v.GarantiaCentavos,
			GarantiaDisponivel: v.GarantiaDisponivel, PrazoEm: v.PrazoEm, PedidoAbertoID: v.PedidoAbertoID,
			Pago: v.Pago, ItemPagoID: v.ItemPagoID, TotalPagoCentavos: v.TotalPagoCentavos, GarantiaContratada: v.GarantiaContratada,
		}
	}
	creds := make([]credencialResposta, 0, len(m.Credenciais))
	for _, cr := range m.Credenciais {
		creds = append(creds, credencialResposta{ID: cr.ID, Nome: cr.Nome, Payload: cr.Payload, EntrouEm: cr.PrimeiraEntradaEm})
	}
	out["credenciais"] = creds
	c.JSON(http.StatusOK, out)
}

// Pagar é POST /eventos/:slug/meu-espaco/pagamento.
func (h *ExpositorHandler) Pagar(c *gin.Context) {
	e, err := h.eventos.BuscarPorSlug(c.Param("slug"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"erro": "evento não encontrado"})
		return
	}
	var req struct {
		GarantiaContratada bool `json:"garantia_contratada"`
	}
	_ = c.ShouldBindJSON(&req)
	p, itens, err := h.svc.ReservarEspaco(c.GetInt64(middleware.ChaveContextoUsuarioID), e, req.GarantiaContratada)
	if err != nil {
		h.erro(c, err)
		return
	}
	c.JSON(http.StatusCreated, paraPedidoResposta(p, itens))
}

// RenomearCredencial é PUT /credenciais-expositor/:id.
func (h *ExpositorHandler) RenomearCredencial(c *gin.Context) {
	id, ok := idDaURL(c)
	if !ok {
		return
	}
	var req struct {
		Nome string `json:"nome"`
	}
	_ = c.ShouldBindJSON(&req)
	if err := h.svc.RenomearCredencial(c.GetInt64(middleware.ChaveContextoUsuarioID), id, req.Nome); err != nil {
		h.erro(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// LerCredencial é POST /checkin/eventos/:id/credencial (organizador ou equipe).
func (h *ExpositorHandler) LerCredencial(c *gin.Context) {
	id, ok := idDaURL(c)
	if !ok {
		return
	}
	var req struct {
		Payload string `json:"payload" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "leia o QR da credencial"})
		return
	}
	r, err := h.svc.LerCredencial(c.GetInt64(middleware.ChaveContextoUsuarioID), id, req.Payload)
	if err != nil {
		h.erro(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"nome": r.Nome, "marca": r.Marca, "mesa": r.Mesa, "primeira_vez": r.PrimeiraVez, "entrou_em": r.EntrouEm})
}

// ---------- organizador ----------

func (h *ExpositorHandler) orgEvento(c *gin.Context) (int64, int64, bool) {
	org, ok := h.org.ObterOrganizadorAtual(c)
	if !ok {
		return 0, 0, false
	}
	id, ok := idDaURL(c)
	return org.ID, id, ok
}

func paramInt(c *gin.Context, nome string) (int64, bool) {
	v, err := strconv.ParseInt(c.Param(nome), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "id inválido"})
		return 0, false
	}
	return v, true
}

// Painel é GET /org/eventos/:id/expositores.
func (h *ExpositorHandler) Painel(c *gin.Context) {
	orgID, id, ok := h.orgEvento(c)
	if !ok {
		return
	}
	p, err := h.svc.Painel(orgID, id)
	if err != nil {
		h.erro(c, err)
		return
	}
	type mesa struct {
		ID           int64  `json:"id"`
		TipoEspacoID int64  `json:"tipo_espaco_id"`
		Setor        string `json:"setor"`
		Codigo       string `json:"codigo"`
	}
	type expositor struct {
		ID               int64      `json:"id"`
		TipoEspacoID     int64      `json:"tipo_espaco_id"`
		EspacoID         *int64     `json:"espaco_id"`
		Marca            string     `json:"marca"`
		Categoria        string     `json:"categoria"`
		Descricao        string     `json:"descricao"`
		PortfolioURL     string     `json:"portfolio_url"`
		Instagram        string     `json:"instagram"`
		ResponsavelNome  string     `json:"responsavel_nome"`
		Telefone         string     `json:"telefone"`
		Status           string     `json:"status"`
		MotivoRejeicao   string     `json:"motivo_rejeicao"`
		PrazoPagamentoEm *time.Time `json:"prazo_pagamento_em"`
	}
	tipos := make([]tipoEspacoResposta, 0, len(p.Tipos))
	for _, t := range p.Tipos {
		tipos = append(tipos, paraTipoEspaco(t))
	}
	mesas := make([]mesa, 0, len(p.Mesas))
	for _, m := range p.Mesas {
		mesas = append(mesas, mesa{ID: m.ID, TipoEspacoID: m.TipoEspacoID, Setor: m.Setor, Codigo: m.Codigo})
	}
	exps := make([]expositor, 0, len(p.Expositores))
	for _, x := range p.Expositores {
		exps = append(exps, expositor{ID: x.ID, TipoEspacoID: x.TipoEspacoID, EspacoID: x.EspacoID, Marca: x.Marca, Categoria: x.Categoria,
			Descricao: x.Descricao, PortfolioURL: x.PortfolioURL, Instagram: x.Instagram, ResponsavelNome: x.ResponsavelNome,
			Telefone: x.Telefone, Status: x.Status, MotivoRejeicao: x.MotivoRejeicao, PrazoPagamentoEm: x.PrazoPagamentoEm})
	}
	c.JSON(http.StatusOK, gin.H{"tipos": tipos, "mesas": mesas, "expositores": exps})
}

type tipoEspacoRequest struct {
	Nome          string `json:"nome" binding:"required"`
	Descricao     string `json:"descricao"`
	PrecoCentavos int64  `json:"preco_centavos"`
	Credenciais   int    `json:"credenciais" binding:"required"`
	Ativo         *bool  `json:"ativo"`
}

func (r tipoEspacoRequest) dados() service.DadosTipoEspaco {
	ativo := true
	if r.Ativo != nil {
		ativo = *r.Ativo
	}
	return service.DadosTipoEspaco{Nome: r.Nome, Descricao: r.Descricao, PrecoCentavos: r.PrecoCentavos, Credenciais: r.Credenciais, Ativo: ativo}
}

// CriarTipo é POST /org/eventos/:id/tipos-espaco.
func (h *ExpositorHandler) CriarTipo(c *gin.Context) {
	orgID, id, ok := h.orgEvento(c)
	if !ok {
		return
	}
	var req tipoEspacoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "informe nome, preço e credenciais"})
		return
	}
	t, err := h.svc.CriarTipo(orgID, id, req.dados())
	if err != nil {
		h.erro(c, err)
		return
	}
	c.JSON(http.StatusCreated, paraTipoEspaco(*t))
}

// AtualizarTipo é PUT /org/eventos/:id/tipos-espaco/:tid.
func (h *ExpositorHandler) AtualizarTipo(c *gin.Context) {
	orgID, id, ok := h.orgEvento(c)
	if !ok {
		return
	}
	tid, ok := paramInt(c, "tid")
	if !ok {
		return
	}
	var req tipoEspacoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "informe nome, preço e credenciais"})
		return
	}
	t, err := h.svc.AtualizarTipo(orgID, id, tid, req.dados())
	if err != nil {
		h.erro(c, err)
		return
	}
	c.JSON(http.StatusOK, paraTipoEspaco(*t))
}

// ExcluirTipo é DELETE /org/eventos/:id/tipos-espaco/:tid.
func (h *ExpositorHandler) ExcluirTipo(c *gin.Context) {
	orgID, id, ok := h.orgEvento(c)
	if !ok {
		return
	}
	tid, ok := paramInt(c, "tid")
	if !ok {
		return
	}
	if err := h.svc.ExcluirTipo(orgID, id, tid); err != nil {
		h.erro(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// CriarMesas é POST /org/eventos/:id/mesas.
func (h *ExpositorHandler) CriarMesas(c *gin.Context) {
	orgID, id, ok := h.orgEvento(c)
	if !ok {
		return
	}
	var req struct {
		TipoEspacoID int64  `json:"tipo_espaco_id" binding:"required"`
		Setor        string `json:"setor" binding:"required"`
		De           int    `json:"de" binding:"required"`
		Ate          int    `json:"ate" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "informe tipo, setor e a faixa de números"})
		return
	}
	n, err := h.svc.CriarMesas(orgID, id, req.TipoEspacoID, req.Setor, req.De, req.Ate)
	if err != nil {
		h.erro(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"criadas": n})
}

// ExcluirMesa é DELETE /org/eventos/:id/mesas/:mid.
func (h *ExpositorHandler) ExcluirMesa(c *gin.Context) {
	orgID, id, ok := h.orgEvento(c)
	if !ok {
		return
	}
	mid, ok := paramInt(c, "mid")
	if !ok {
		return
	}
	if err := h.svc.ExcluirMesa(orgID, id, mid); err != nil {
		h.erro(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// Aprovar é POST /org/eventos/:id/expositores/:eid/aprovar.
func (h *ExpositorHandler) Aprovar(c *gin.Context) {
	orgID, id, ok := h.orgEvento(c)
	if !ok {
		return
	}
	eid, ok := paramInt(c, "eid")
	if !ok {
		return
	}
	var req struct {
		EspacoID int64 `json:"espaco_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "escolha a mesa"})
		return
	}
	x, err := h.svc.Aprovar(orgID, id, eid, req.EspacoID)
	if err != nil {
		h.erro(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": x.Status, "prazo_pagamento_em": x.PrazoPagamentoEm})
}

// Rejeitar é POST /org/eventos/:id/expositores/:eid/rejeitar.
func (h *ExpositorHandler) Rejeitar(c *gin.Context) {
	orgID, id, ok := h.orgEvento(c)
	if !ok {
		return
	}
	eid, ok := paramInt(c, "eid")
	if !ok {
		return
	}
	var req struct {
		Motivo string `json:"motivo"`
	}
	_ = c.ShouldBindJSON(&req)
	x, err := h.svc.Rejeitar(orgID, id, eid, req.Motivo)
	if err != nil {
		h.erro(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": x.Status})
}
