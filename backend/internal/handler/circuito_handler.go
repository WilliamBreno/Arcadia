package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
	"github.com/WilliamBreno/Arcadia/backend/internal/service"
)

// Circuitos, ranking por temporada e seletivas (item 8.5).
type CircuitoHandler struct {
	svc     *service.CircuitoService
	eventos *repository.EventoRepository
	org     *OrganizadorHandler
}

func NovoCircuitoHandler(s *service.CircuitoService, e *repository.EventoRepository, org *OrganizadorHandler) *CircuitoHandler {
	return &CircuitoHandler{svc: s, eventos: e, org: org}
}

type circuitoResposta struct {
	ID              int64  `json:"id"`
	Nome            string `json:"nome"`
	Slug            string `json:"slug"`
	Descricao       string `json:"descricao"`
	TemporadaInicio string `json:"temporada_inicio"`
	TemporadaFim    string `json:"temporada_fim"`
	Status          string `json:"status,omitempty"`
}

func paraCircuitoResposta(c service.Circuito) circuitoResposta {
	return circuitoResposta{ID: c.ID, Nome: c.Nome, Slug: c.Slug, Descricao: c.Descricao,
		TemporadaInicio: c.TemporadaInicio.Format("2006-01-02"), TemporadaFim: c.TemporadaFim.Format("2006-01-02")}
}

type seletivaResposta struct {
	ID       int64      `json:"id"`
	Vagas    int        `json:"vagas"`
	Status   string     `json:"status"`
	Titulo   string     `json:"titulo"`
	Slug     string     `json:"slug"`
	InicioEm *time.Time `json:"inicio_em"`
}

func paraSeletivas(ss []service.SeletivaInfo) []seletivaResposta {
	out := make([]seletivaResposta, 0, len(ss))
	for _, s := range ss {
		out = append(out, seletivaResposta{ID: s.ID, Vagas: s.Vagas, Status: s.Status, Titulo: s.Titulo, Slug: s.Slug, InicioEm: s.InicioEm})
	}
	return out
}

func (h *CircuitoHandler) erro(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrCircuitoNaoEncontrado), errors.Is(err, service.ErrPedidoNaoEncontrado8):
		c.JSON(http.StatusNotFound, gin.H{"erro": err.Error()})
	case errors.Is(err, service.ErrEventoNaoPertenceAoOrganizador):
		c.JSON(http.StatusForbidden, gin.H{"erro": "evento não pertence a este organizador"})
	case errors.Is(err, service.ErrEventoJaEmCircuito):
		c.JSON(http.StatusConflict, gin.H{"erro": err.Error()})
	case errors.Is(err, service.ErrCircuitoInvalido), errors.Is(err, service.ErrForaDaTemporada), errors.Is(err, service.ErrSeletivaInvalida):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"erro": err.Error()})
	default:
		slog.Error("circuito", "erro", err)
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao processar o circuito"})
	}
}

// ---------- público ----------

type linhaCircuitoResposta struct {
	Posicao    int    `json:"posicao"`
	Nome       string `json:"nome"`
	Pontos     int    `json:"pontos"`
	Vitorias   int    `json:"vitorias"`
	Eventos    int    `json:"eventos"`
	PerfilSlug string `json:"perfil_slug,omitempty"`
}

// Publico é GET /circuitos/:slug.
func (h *CircuitoHandler) Publico(c *gin.Context) {
	p, err := h.svc.Publico(c.Param("slug"))
	if err != nil {
		h.erro(c, err)
		return
	}
	type etapa struct {
		Titulo   string     `json:"titulo"`
		Slug     string     `json:"slug"`
		InicioEm *time.Time `json:"inicio_em"`
		Liberado bool       `json:"resultado_liberado"`
	}
	etapas := make([]etapa, 0, len(p.Etapas))
	for _, e := range p.Etapas {
		etapas = append(etapas, etapa{Titulo: e.Titulo, Slug: e.Slug, InicioEm: e.InicioEm, Liberado: e.Liberado})
	}
	ranking := map[string][]linhaCircuitoResposta{}
	for tipo, ls := range p.Ranking {
		for _, l := range ls {
			ranking[tipo] = append(ranking[tipo], linhaCircuitoResposta{Posicao: l.Posicao, Nome: l.Nome, Pontos: l.Pontos,
				Vitorias: l.Vitorias, Eventos: l.Eventos, PerfilSlug: l.PerfilSlug})
		}
	}
	c.JSON(http.StatusOK, gin.H{"circuito": paraCircuitoResposta(p.Circuito), "etapas": etapas, "ranking": ranking,
		"pontos_por_posicao": service.PontosPorPosicao})
}

// DoEventoPublico é GET /eventos/:slug/circuito: selos da página do evento.
func (h *CircuitoHandler) DoEventoPublico(c *gin.Context) {
	e, err := h.eventos.BuscarPublicoPorSlug(c.Param("slug"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"erro": "evento não encontrado"})
		return
	}
	out := gin.H{}
	if ce, err := h.svc.DoEvento(e.ID); err == nil && ce.Status == "aceito" {
		out["circuito"] = paraCircuitoResposta(ce.Circuito)
	}
	if fs, err := h.svc.ComoSeletiva(e.ID); err == nil {
		out["classifica_para"] = soAceitas(paraSeletivas(fs))
	}
	if ss, err := h.svc.DaFinal(e.ID); err == nil {
		out["seletivas"] = soAceitas(paraSeletivas(ss))
	}
	c.JSON(http.StatusOK, out)
}

func soAceitas(ss []seletivaResposta) []seletivaResposta {
	out := make([]seletivaResposta, 0, len(ss))
	for _, s := range ss {
		if s.Status == "aceita" {
			out = append(out, s)
		}
	}
	return out
}

// ---------- organizador ----------

type criarCircuitoRequest struct {
	Nome            string `json:"nome" binding:"required"`
	Descricao       string `json:"descricao"`
	TemporadaInicio string `json:"temporada_inicio" binding:"required"`
	TemporadaFim    string `json:"temporada_fim" binding:"required"`
}

// Criar é POST /org/circuitos.
func (h *CircuitoHandler) Criar(c *gin.Context) {
	org, ok := h.org.ObterOrganizadorAtual(c)
	if !ok {
		return
	}
	var req criarCircuitoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "informe nome e temporada"})
		return
	}
	ini, e1 := time.Parse("2006-01-02", req.TemporadaInicio)
	fim, e2 := time.Parse("2006-01-02", req.TemporadaFim)
	if e1 != nil || e2 != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "datas da temporada inválidas"})
		return
	}
	ci, err := h.svc.Criar(org.ID, service.DadosCircuito{Nome: req.Nome, Descricao: req.Descricao, TemporadaInicio: ini, TemporadaFim: fim})
	if err != nil {
		h.erro(c, err)
		return
	}
	c.JSON(http.StatusCreated, paraCircuitoResposta(*ci))
}

// Meus é GET /org/circuitos: circuitos do organizador com os pedidos.
func (h *CircuitoHandler) Meus(c *gin.Context) {
	org, ok := h.org.ObterOrganizadorAtual(c)
	if !ok {
		return
	}
	cs, err := h.svc.Meus(org.ID)
	if err != nil {
		h.erro(c, err)
		return
	}
	type pedido struct {
		ID           int64      `json:"id"`
		EventoTitulo string     `json:"evento_titulo"`
		EventoSlug   string     `json:"evento_slug"`
		InicioEm     *time.Time `json:"inicio_em"`
		Status       string     `json:"status"`
	}
	type item struct {
		circuitoResposta
		Pedidos []pedido `json:"pedidos"`
	}
	out := make([]item, 0, len(cs))
	for _, ci := range cs {
		it := item{circuitoResposta: paraCircuitoResposta(ci.Circuito), Pedidos: []pedido{}}
		for _, p := range ci.Pedidos {
			it.Pedidos = append(it.Pedidos, pedido{ID: p.ID, EventoTitulo: p.EventoTitulo, EventoSlug: p.EventoSlug, InicioEm: p.InicioEm, Status: p.Status})
		}
		out = append(out, it)
	}
	c.JSON(http.StatusOK, out)
}

// Abertos é GET /org/circuitos/abertos.
func (h *CircuitoHandler) Abertos(c *gin.Context) {
	cs, err := h.svc.Abertos()
	if err != nil {
		h.erro(c, err)
		return
	}
	out := make([]circuitoResposta, 0, len(cs))
	for _, ci := range cs {
		out = append(out, paraCircuitoResposta(ci))
	}
	c.JSON(http.StatusOK, out)
}

type decidirCircuitoRequest struct {
	Aceitar bool `json:"aceitar"`
}

// DecidirPedido é POST /org/circuitos/pedidos/:id/decidir.
func (h *CircuitoHandler) DecidirPedido(c *gin.Context) {
	org, ok := h.org.ObterOrganizadorAtual(c)
	if !ok {
		return
	}
	id, ok := idDaURL(c)
	if !ok {
		return
	}
	var req decidirCircuitoRequest
	_ = c.ShouldBindJSON(&req)
	if err := h.svc.Decidir(org.ID, id, req.Aceitar); err != nil {
		h.erro(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// DoEvento é GET /org/eventos/:id/circuito: circuito e seletivas do evento.
func (h *CircuitoHandler) DoEvento(c *gin.Context) {
	org, ok := h.org.ObterOrganizadorAtual(c)
	if !ok {
		return
	}
	id, ok := idDaURL(c)
	if !ok {
		return
	}
	e, err := h.eventos.BuscarPorID(id)
	if err != nil || e.OrganizadorID != org.ID {
		h.erro(c, service.ErrEventoNaoPertenceAoOrganizador)
		return
	}
	out := gin.H{"circuito": nil}
	if ce, err := h.svc.DoEvento(id); err == nil {
		r := paraCircuitoResposta(ce.Circuito)
		r.Status = ce.Status
		out["circuito"] = r
	}
	ss, _ := h.svc.DaFinal(id)
	fs, _ := h.svc.ComoSeletiva(id)
	out["seletivas"] = paraSeletivas(ss)
	out["classifica_para"] = paraSeletivas(fs)
	c.JSON(http.StatusOK, out)
}

// EntrarCircuito é POST /org/eventos/:id/circuito.
func (h *CircuitoHandler) EntrarCircuito(c *gin.Context) {
	org, ok := h.org.ObterOrganizadorAtual(c)
	if !ok {
		return
	}
	id, ok := idDaURL(c)
	if !ok {
		return
	}
	var req struct {
		CircuitoID int64 `json:"circuito_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "escolha o circuito"})
		return
	}
	ce, err := h.svc.PedirEntrada(org.ID, id, req.CircuitoID)
	if err != nil {
		h.erro(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"status": ce.Status})
}

// SairCircuito é DELETE /org/eventos/:id/circuito.
func (h *CircuitoHandler) SairCircuito(c *gin.Context) {
	org, ok := h.org.ObterOrganizadorAtual(c)
	if !ok {
		return
	}
	id, ok := idDaURL(c)
	if !ok {
		return
	}
	if err := h.svc.SairDoCircuito(org.ID, id); err != nil {
		h.erro(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// CriarSeletiva é POST /org/eventos/:id/seletivas (o :id é a final).
func (h *CircuitoHandler) CriarSeletiva(c *gin.Context) {
	org, ok := h.org.ObterOrganizadorAtual(c)
	if !ok {
		return
	}
	id, ok := idDaURL(c)
	if !ok {
		return
	}
	var req struct {
		Slug  string `json:"slug" binding:"required"`
		Vagas int    `json:"vagas" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "informe o endereço do evento e as vagas"})
		return
	}
	st, err := h.svc.CriarSeletiva(org.ID, id, req.Slug, req.Vagas)
	if err != nil {
		h.erro(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": st.ID, "status": st.Status})
}

func seletivaDaURL(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("sid"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "id inválido"})
		return 0, false
	}
	return id, true
}

// RemoverSeletiva é DELETE /org/eventos/:id/seletivas/:sid.
func (h *CircuitoHandler) RemoverSeletiva(c *gin.Context) {
	org, ok := h.org.ObterOrganizadorAtual(c)
	if !ok {
		return
	}
	id, ok := idDaURL(c)
	if !ok {
		return
	}
	sid, ok := seletivaDaURL(c)
	if !ok {
		return
	}
	if err := h.svc.RemoverSeletiva(org.ID, id, sid); err != nil {
		h.erro(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// DecidirSeletiva é POST /org/eventos/:id/seletivas/:sid/decidir (o :id é a seletiva).
func (h *CircuitoHandler) DecidirSeletiva(c *gin.Context) {
	org, ok := h.org.ObterOrganizadorAtual(c)
	if !ok {
		return
	}
	id, ok := idDaURL(c)
	if !ok {
		return
	}
	sid, ok := seletivaDaURL(c)
	if !ok {
		return
	}
	var req decidirCircuitoRequest
	_ = c.ShouldBindJSON(&req)
	if err := h.svc.DecidirSeletiva(org.ID, id, sid, req.Aceitar); err != nil {
		h.erro(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
