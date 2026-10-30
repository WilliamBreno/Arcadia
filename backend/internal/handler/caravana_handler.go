package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/WilliamBreno/Arcadia/backend/internal/middleware"
	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
	"github.com/WilliamBreno/Arcadia/backend/internal/service"
)

// Caravanas (item 8.7).
type CaravanaHandler struct {
	svc     *service.CaravanaService
	eventos *repository.EventoRepository
	org     *OrganizadorHandler
}

func NovoCaravanaHandler(s *service.CaravanaService, e *repository.EventoRepository, org *OrganizadorHandler) *CaravanaHandler {
	return &CaravanaHandler{svc: s, eventos: e, org: org}
}

func (h *CaravanaHandler) erro(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrCaravanaNaoEncontrada):
		c.JSON(http.StatusNotFound, gin.H{"erro": err.Error()})
	case errors.Is(err, service.ErrSoResponsavel), errors.Is(err, service.ErrEventoNaoPertenceAoOrganizador):
		c.JSON(http.StatusForbidden, gin.H{"erro": err.Error()})
	case errors.Is(err, service.ErrCaravanaLotada):
		c.JSON(http.StatusConflict, gin.H{"erro": err.Error()})
	case errors.Is(err, service.ErrDadosCaravana), errors.Is(err, service.ErrAceiteCaravana):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"erro": err.Error()})
	default:
		slog.Error("caravana", "erro", err)
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro na caravana"})
	}
}

type caravanaRequest struct {
	Nome          string     `json:"nome" binding:"required"`
	Cidade        string     `json:"cidade" binding:"required"`
	UF            string     `json:"uf"`
	PontoEmbarque string     `json:"ponto_embarque" binding:"required"`
	SaidaEm       *time.Time `json:"saida_em"`
	Vagas         *int       `json:"vagas"`
	Telefone      string     `json:"telefone"`
}

func (r caravanaRequest) dados() service.DadosCaravana {
	return service.DadosCaravana{Nome: r.Nome, Cidade: r.Cidade, UF: r.UF, PontoEmbarque: r.PontoEmbarque, SaidaEm: r.SaidaEm, Vagas: r.Vagas, Telefone: r.Telefone}
}

func uid(c *gin.Context) int64 { return c.GetInt64(middleware.ChaveContextoUsuarioID) }

// Criar é POST /eventos/:slug/caravanas.
func (h *CaravanaHandler) Criar(c *gin.Context) {
	e, err := h.eventos.BuscarPublicoPorSlug(c.Param("slug"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"erro": "evento não encontrado"})
		return
	}
	var req caravanaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "informe nome, cidade e ponto de embarque"})
		return
	}
	cv, err := h.svc.Criar(uid(c), e, req.dados())
	if err != nil {
		h.erro(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"codigo": cv.Codigo})
}

// Minhas é GET /eventos/:slug/minhas-caravanas.
func (h *CaravanaHandler) Minhas(c *gin.Context) {
	e, err := h.eventos.BuscarPorSlug(c.Param("slug"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"erro": "evento não encontrado"})
		return
	}
	ms, err := h.svc.Minhas(uid(c), e.ID)
	if err != nil {
		h.erro(c, err)
		return
	}
	type m struct {
		Codigo      string     `json:"codigo"`
		Nome        string     `json:"nome"`
		Cidade      string     `json:"cidade"`
		Ponto       string     `json:"ponto_embarque"`
		SaidaEm     *time.Time `json:"saida_em"`
		Responsavel bool       `json:"responsavel"`
	}
	out := make([]m, 0, len(ms))
	for _, x := range ms {
		out = append(out, m{Codigo: x.Codigo, Nome: x.Nome, Cidade: x.Cidade, Ponto: x.Ponto, SaidaEm: x.SaidaEm, Responsavel: x.Responsavel})
	}
	c.JSON(http.StatusOK, out)
}

// Ver é GET /caravanas/:codigo.
func (h *CaravanaHandler) Ver(c *gin.Context) {
	v, err := h.svc.Ver(uid(c), c.Param("codigo"))
	if err != nil {
		h.erro(c, err)
		return
	}
	cv := v.Caravana
	out := gin.H{
		"codigo": cv.Codigo, "nome": cv.Nome, "cidade": cv.Cidade, "uf": cv.UF, "ponto_embarque": cv.PontoEmbarque,
		"saida_em": cv.SaidaEm, "vagas": cv.Vagas, "membros": v.Membros, "evento_titulo": v.EventoTitulo, "evento_slug": v.EventoSlug,
		"responsavel": v.Responsavel, "membro": v.Membro, "responsavel_nome": v.ResponsavelNome,
	}
	// Telefone do responsável: só para quem já está na caravana.
	if v.Membro || v.Responsavel {
		out["telefone"] = cv.Telefone
	}
	c.JSON(http.StatusOK, out)
}

// Atualizar é PUT /caravanas/:codigo.
func (h *CaravanaHandler) Atualizar(c *gin.Context) {
	var req caravanaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "informe nome, cidade e ponto de embarque"})
		return
	}
	if _, err := h.svc.Atualizar(uid(c), c.Param("codigo"), req.dados()); err != nil {
		h.erro(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// Encerrar é DELETE /caravanas/:codigo.
func (h *CaravanaHandler) Encerrar(c *gin.Context) {
	if err := h.svc.Encerrar(uid(c), c.Param("codigo")); err != nil {
		h.erro(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// Entrar é POST /caravanas/:codigo/entrar.
func (h *CaravanaHandler) Entrar(c *gin.Context) {
	var req struct {
		Nome     string `json:"nome"`
		Telefone string `json:"telefone"`
		Aceite   bool   `json:"aceite"`
	}
	_ = c.ShouldBindJSON(&req)
	if err := h.svc.Entrar(uid(c), c.Param("codigo"), req.Nome, req.Telefone, req.Aceite); err != nil {
		h.erro(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// Sair é DELETE /caravanas/:codigo/eu.
func (h *CaravanaHandler) Sair(c *gin.Context) {
	if err := h.svc.Sair(uid(c), c.Param("codigo")); err != nil {
		h.erro(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// Lista é GET /caravanas/:codigo/lista (só o responsável).
func (h *CaravanaHandler) Lista(c *gin.Context) {
	ls, err := h.svc.Lista(uid(c), c.Param("codigo"))
	if err != nil {
		h.erro(c, err)
		return
	}
	type l struct {
		ID          int64      `json:"id"`
		Nome        string     `json:"nome"`
		Telefone    string     `json:"telefone"`
		TemIngresso bool       `json:"tem_ingresso"`
		EmbarcouEm  *time.Time `json:"embarcou_em"`
	}
	out := make([]l, 0, len(ls))
	for _, x := range ls {
		out = append(out, l{ID: x.ID, Nome: x.Nome, Telefone: x.Telefone, TemIngresso: x.TemIngresso, EmbarcouEm: x.EmbarcouEm})
	}
	c.JSON(http.StatusOK, out)
}

// Embarque é POST /caravanas/:codigo/membros/:mid/embarque {"embarcou": bool}.
func (h *CaravanaHandler) Embarque(c *gin.Context) {
	mid, ok := paramInt(c, "mid")
	if !ok {
		return
	}
	var req struct {
		Embarcou bool `json:"embarcou"`
	}
	_ = c.ShouldBindJSON(&req)
	if err := h.svc.MarcarEmbarque(uid(c), c.Param("codigo"), mid, req.Embarcou); err != nil {
		h.erro(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// DoOrganizador é GET /org/eventos/:id/caravanas.
func (h *CaravanaHandler) DoOrganizador(c *gin.Context) {
	org, ok := h.org.ObterOrganizadorAtual(c)
	if !ok {
		return
	}
	id, ok := idDaURL(c)
	if !ok {
		return
	}
	rs, err := h.svc.DoOrganizador(org.ID, id)
	if err != nil {
		h.erro(c, err)
		return
	}
	type r struct {
		Nome        string     `json:"nome"`
		Cidade      string     `json:"cidade"`
		UF          string     `json:"uf"`
		SaidaEm     *time.Time `json:"saida_em"`
		Membros     int        `json:"membros"`
		ComIngresso int        `json:"com_ingresso"`
		Responsavel string     `json:"responsavel"`
	}
	out := make([]r, 0, len(rs))
	for _, x := range rs {
		out = append(out, r{Nome: x.Nome, Cidade: x.Cidade, UF: x.UF, SaidaEm: x.SaidaEm, Membros: x.Membros, ComIngresso: x.ComIngresso, Responsavel: x.Responsavel})
	}
	c.JSON(http.StatusOK, out)
}
