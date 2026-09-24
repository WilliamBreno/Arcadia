package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/WilliamBreno/Arcadia/backend/internal/middleware"
	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
	"github.com/WilliamBreno/Arcadia/backend/internal/service"
)

type AvaliacaoHandler struct {
	organizadorHandler *OrganizadorHandler
	eventos            *repository.EventoRepository
	service            *service.AvaliacaoService
}

func NovoAvaliacaoHandler(o *OrganizadorHandler, e *repository.EventoRepository, s *service.AvaliacaoService) *AvaliacaoHandler {
	return &AvaliacaoHandler{organizadorHandler: o, eventos: e, service: s}
}

func (h *AvaliacaoHandler) responderErro(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrEventoNaoPertenceAoOrganizador), errors.Is(err, service.ErrNaoEhJuradoConfirmado):
		c.JSON(http.StatusForbidden, gin.H{"erro": err.Error()})
	case errors.Is(err, service.ErrCriterioNaoEncontrado), errors.Is(err, service.ErrFichaNaoEncontrada):
		c.JSON(http.StatusNotFound, gin.H{"erro": err.Error()})
	case errors.Is(err, service.ErrCriterioEmUso), errors.Is(err, service.ErrAvaliacaoFinalizada):
		c.JSON(http.StatusConflict, gin.H{"erro": err.Error()})
	case errors.Is(err, service.ErrResultadoOculto):
		c.JSON(http.StatusNotFound, gin.H{"erro": err.Error()})
	default:
		c.JSON(http.StatusUnprocessableEntity, gin.H{"erro": err.Error()})
	}
}

type criterioResposta struct {
	ID               int64   `json:"id"`
	Nome             string  `json:"nome"`
	Peso             float64 `json:"peso"`
	NotaMin          float64 `json:"nota_min"`
	NotaMax          float64 `json:"nota_max"`
	Passo            float64 `json:"passo"`
	TipoApresentacao *string `json:"tipo_apresentacao"`
	Ordem            int     `json:"ordem"`
}

func paraCriterioResposta(c *domain.CriterioAvaliacao) criterioResposta {
	var tipo *string
	if c.TipoApresentacao != nil {
		t := string(*c.TipoApresentacao)
		tipo = &t
	}
	return criterioResposta{ID: c.ID, Nome: c.Nome, Peso: c.Peso, NotaMin: c.NotaMin, NotaMax: c.NotaMax, Passo: c.Passo, TipoApresentacao: tipo, Ordem: c.Ordem}
}

type criterioRequest struct {
	Nome             string  `json:"nome" binding:"required"`
	Peso             float64 `json:"peso" binding:"required"`
	NotaMin          float64 `json:"nota_min"`
	NotaMax          float64 `json:"nota_max" binding:"required"`
	Passo            float64 `json:"passo" binding:"required"`
	TipoApresentacao *string `json:"tipo_apresentacao"`
	Ordem            int     `json:"ordem"`
}

func (r criterioRequest) dados() service.CriterioDados {
	d := service.CriterioDados{Nome: r.Nome, Peso: r.Peso, NotaMin: r.NotaMin, NotaMax: r.NotaMax, Passo: r.Passo, Ordem: r.Ordem}
	if r.TipoApresentacao != nil && *r.TipoApresentacao != "" {
		t := domain.TipoApresentacao(*r.TipoApresentacao)
		d.TipoApresentacao = &t
	}
	return d
}

func (h *AvaliacaoHandler) ListarCriterios(c *gin.Context) {
	org, ok := h.organizadorHandler.ObterOrganizadorAtual(c)
	if !ok {
		return
	}
	eventoID, ok := idDaURL(c)
	if !ok {
		return
	}
	lista, err := h.service.ListarCriterios(org.ID, eventoID)
	if err != nil {
		h.responderErro(c, err)
		return
	}
	resp := make([]criterioResposta, 0, len(lista))
	for i := range lista {
		resp = append(resp, paraCriterioResposta(&lista[i]))
	}
	c.JSON(http.StatusOK, resp)
}

func (h *AvaliacaoHandler) CriarCriterio(c *gin.Context) {
	org, ok := h.organizadorHandler.ObterOrganizadorAtual(c)
	if !ok {
		return
	}
	eventoID, ok := idDaURL(c)
	if !ok {
		return
	}
	var req criterioRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "dados inválidos"})
		return
	}
	crit, err := h.service.CriarCriterio(org.ID, eventoID, req.dados())
	if err != nil {
		h.responderErro(c, err)
		return
	}
	c.JSON(http.StatusCreated, paraCriterioResposta(crit))
}

func (h *AvaliacaoHandler) AtualizarCriterio(c *gin.Context) {
	org, ok := h.organizadorHandler.ObterOrganizadorAtual(c)
	if !ok {
		return
	}
	eventoID, ok := idDaURL(c)
	if !ok {
		return
	}
	critID, err := strconv.ParseInt(c.Param("criterioId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "id inválido"})
		return
	}
	var req criterioRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "dados inválidos"})
		return
	}
	crit, err := h.service.AtualizarCriterio(org.ID, eventoID, critID, req.dados())
	if err != nil {
		h.responderErro(c, err)
		return
	}
	c.JSON(http.StatusOK, paraCriterioResposta(crit))
}

func (h *AvaliacaoHandler) ExcluirCriterio(c *gin.Context) {
	org, ok := h.organizadorHandler.ObterOrganizadorAtual(c)
	if !ok {
		return
	}
	eventoID, ok := idDaURL(c)
	if !ok {
		return
	}
	critID, err := strconv.ParseInt(c.Param("criterioId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "id inválido"})
		return
	}
	if err := h.service.ExcluirCriterio(org.ID, eventoID, critID); err != nil {
		h.responderErro(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

type linhaRankingResposta struct {
	Posicao       int     `json:"posicao"`
	FichaID       int64   `json:"ficha_id"`
	Nome          string  `json:"nome"`
	NomeArtistico string  `json:"nome_artistico"`
	NotaFinal     float64 `json:"nota_final"`
	Jurados       int     `json:"jurados"`
}

func paraRankingResposta(r map[string][]service.LinhaRanking, publico bool) map[string][]linhaRankingResposta {
	resp := map[string][]linhaRankingResposta{}
	for tipo, linhas := range r {
		lista := make([]linhaRankingResposta, 0, len(linhas))
		for _, l := range linhas {
			item := linhaRankingResposta{Posicao: l.Posicao, FichaID: l.FichaID, Nome: l.Nome,
				NomeArtistico: l.NomeArtistico, NotaFinal: l.NotaFinal, Jurados: l.Jurados}
			if publico {
				item = linhaRankingResposta{Posicao: l.Posicao, Nome: service.NomePublico(l), NotaFinal: l.NotaFinal}
			}
			lista = append(lista, item)
		}
		resp[tipo] = lista
	}
	return resp
}

// Ranking é GET /org/eventos/:id/ranking (parcial, a qualquer momento).
func (h *AvaliacaoHandler) Ranking(c *gin.Context) {
	org, ok := h.organizadorHandler.ObterOrganizadorAtual(c)
	if !ok {
		return
	}
	eventoID, ok := idDaURL(c)
	if !ok {
		return
	}
	r, liberado, err := h.service.RankingDoOrganizador(org.ID, eventoID)
	if err != nil {
		h.responderErro(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ranking": paraRankingResposta(r, false), "resultado_liberado_em": liberado})
}

type liberarRequest struct {
	Liberar bool `json:"liberar"`
}

// Liberar é POST /org/eventos/:id/resultado {liberar}.
func (h *AvaliacaoHandler) Liberar(c *gin.Context) {
	org, ok := h.organizadorHandler.ObterOrganizadorAtual(c)
	if !ok {
		return
	}
	eventoID, ok := idDaURL(c)
	if !ok {
		return
	}
	var req liberarRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "dados inválidos"})
		return
	}
	if err := h.service.DefinirResultadoLiberado(org.ID, eventoID, req.Liberar); err != nil {
		h.responderErro(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"liberado": req.Liberar})
}

// ResultadoPublico é GET /eventos/:slug/resultado (404 até liberar).
func (h *AvaliacaoHandler) ResultadoPublico(c *gin.Context) {
	evento, err := h.eventos.BuscarPorSlug(c.Param("slug"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"erro": "evento não encontrado"})
		return
	}
	r, err := h.service.ResultadoPublico(evento)
	if err != nil {
		h.responderErro(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ranking": paraRankingResposta(r, true)})
}

// --- jurado ---

func (h *AvaliacaoHandler) eventoEFicha(c *gin.Context) (*domain.Evento, int64, bool) {
	evento, err := h.eventos.BuscarPorSlug(c.Param("slug"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"erro": "evento não encontrado"})
		return nil, 0, false
	}
	fichaID, err := strconv.ParseInt(c.Param("fichaId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "id inválido"})
		return nil, 0, false
	}
	return evento, fichaID, true
}

type notaResposta struct {
	CriterioID int64   `json:"criterio_id"`
	Nota       float64 `json:"nota"`
	Comentario string  `json:"comentario"`
	Finalizada bool    `json:"finalizada"`
}

func paraNotasResposta(avs []domain.Avaliacao) []notaResposta {
	resp := make([]notaResposta, 0, len(avs))
	for _, a := range avs {
		resp = append(resp, notaResposta{CriterioID: a.CriterioID, Nota: a.Nota, Comentario: a.Comentario, Finalizada: a.Finalizada})
	}
	return resp
}

// FormularioJurado é GET /eventos/:slug/participantes/:fichaId/avaliacao:
// critérios aplicáveis + as notas já dadas pelo próprio jurado.
func (h *AvaliacaoHandler) FormularioJurado(c *gin.Context) {
	evento, fichaID, ok := h.eventoEFicha(c)
	if !ok {
		return
	}
	usuarioID := c.GetInt64(middleware.ChaveContextoUsuarioID)
	criterios, err := h.service.CriteriosParaJurado(usuarioID, evento.ID, fichaID)
	if err != nil {
		h.responderErro(c, err)
		return
	}
	minhas, err := h.service.MinhaAvaliacao(usuarioID, evento.ID, fichaID)
	if err != nil {
		h.responderErro(c, err)
		return
	}
	crit := make([]criterioResposta, 0, len(criterios))
	for i := range criterios {
		crit = append(crit, paraCriterioResposta(&criterios[i]))
	}
	c.JSON(http.StatusOK, gin.H{"criterios": crit, "notas": paraNotasResposta(minhas)})
}

type avaliarRequest struct {
	Notas []struct {
		CriterioID int64   `json:"criterio_id"`
		Nota       float64 `json:"nota"`
		Comentario string  `json:"comentario"`
	} `json:"notas"`
	Finalizar bool `json:"finalizar"`
}

// Avaliar é PUT /eventos/:slug/participantes/:fichaId/avaliacao.
func (h *AvaliacaoHandler) Avaliar(c *gin.Context) {
	evento, fichaID, ok := h.eventoEFicha(c)
	if !ok {
		return
	}
	var req avaliarRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "dados inválidos"})
		return
	}
	notas := make([]service.NotaInput, 0, len(req.Notas))
	for _, n := range req.Notas {
		notas = append(notas, service.NotaInput{CriterioID: n.CriterioID, Nota: n.Nota, Comentario: n.Comentario})
	}
	avs, err := h.service.Avaliar(c.GetInt64(middleware.ChaveContextoUsuarioID), evento.ID, fichaID, notas, req.Finalizar)
	if err != nil {
		h.responderErro(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"notas": paraNotasResposta(avs)})
}
