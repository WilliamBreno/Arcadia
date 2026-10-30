package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/WilliamBreno/Arcadia/backend/internal/middleware"
	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
	"github.com/WilliamBreno/Arcadia/backend/internal/service"
)

// Voto do público e devolutiva dos jurados (item 8.2).
type ConcursoPublicoHandler struct {
	svc       *service.ConcursoPublicoService
	avaliacao *service.AvaliacaoService
	perfis    *service.PerfilTalentoService
	eventos   *repository.EventoRepository
	org       *OrganizadorHandler
}

func NovoConcursoPublicoHandler(s *service.ConcursoPublicoService, a *service.AvaliacaoService, p *service.PerfilTalentoService,
	e *repository.EventoRepository, org *OrganizadorHandler) *ConcursoPublicoHandler {
	return &ConcursoPublicoHandler{svc: s, avaliacao: a, perfis: p, eventos: e, org: org}
}

func (h *ConcursoPublicoHandler) eventoPublico(c *gin.Context) (*domain.Evento, bool) {
	e, err := h.eventos.BuscarPublicoPorSlug(c.Param("slug"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"erro": "evento não encontrado"})
		return nil, false
	}
	return e, true
}

func (h *ConcursoPublicoHandler) eventoDoOrganizador(c *gin.Context) (*domain.Evento, bool) {
	org, ok := h.org.ObterOrganizadorAtual(c)
	if !ok {
		return nil, false
	}
	id, ok := idDaURL(c)
	if !ok {
		return nil, false
	}
	e, err := h.eventos.BuscarPorID(id)
	if err != nil || e.OrganizadorID != org.ID {
		c.JSON(http.StatusNotFound, gin.H{"erro": "evento não encontrado"})
		return nil, false
	}
	return e, true
}

// Votacao é GET /eventos/:slug/votacao: se está aberta e os candidatos.
func (h *ConcursoPublicoHandler) Votacao(c *gin.Context) {
	e, ok := h.eventoPublico(c)
	if !ok {
		return
	}
	cands, err := h.svc.Candidatos(e.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao ler a votação"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"aberta": service.VotacaoAberta(e), "categorias": cands})
}

// MeusVotos é GET /eventos/:slug/meus-votos.
func (h *ConcursoPublicoHandler) MeusVotos(c *gin.Context) {
	e, ok := h.eventoPublico(c)
	if !ok {
		return
	}
	votos, n, err := h.svc.MeusVotos(e.ID, c.GetInt64(middleware.ChaveContextoUsuarioID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao ler seus votos"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"votos": votos, "ingressos": n})
}

// Votar é POST /eventos/:slug/votos {"tipo", "ficha_id"}.
func (h *ConcursoPublicoHandler) Votar(c *gin.Context) {
	e, ok := h.eventoPublico(c)
	if !ok {
		return
	}
	var req struct {
		Tipo    string `json:"tipo"`
		FichaID int64  `json:"ficha_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "escolha em quem votar"})
		return
	}
	n, err := h.svc.Votar(e, c.GetInt64(middleware.ChaveContextoUsuarioID), req.Tipo, req.FichaID)
	switch {
	case errors.Is(err, service.ErrVotacaoFechada), errors.Is(err, service.ErrSemCheckin), errors.Is(err, service.ErrCandidatoInvalido):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"erro": err.Error()})
	case err != nil:
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao votar"})
	default:
		c.JSON(http.StatusOK, gin.H{"votos": n})
	}
}

// MinhaDevolutiva é GET /eventos/:slug/minha-devolutiva.
func (h *ConcursoPublicoHandler) MinhaDevolutiva(c *gin.Context) {
	e, ok := h.eventoPublico(c)
	if !ok {
		return
	}
	var ranking map[string][]service.LinhaRanking
	if e.ResultadoLiberadoEm != nil {
		ranking, _ = h.avaliacao.ResultadoPublico(e)
	}
	d, err := h.svc.MinhaDevolutiva(e, c.GetInt64(middleware.ChaveContextoUsuarioID), ranking)
	switch {
	case errors.Is(err, service.ErrDevolutivaOculta), errors.Is(err, service.ErrSemDevolutiva):
		c.JSON(http.StatusNotFound, gin.H{"erro": err.Error()})
	case err != nil:
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao montar a devolutiva"})
	default:
		c.JSON(http.StatusOK, d)
	}
}

// Situacao é GET /org/eventos/:id/votacao: estado e contagem (só o organizador vê antes).
func (h *ConcursoPublicoHandler) Situacao(c *gin.Context) {
	e, ok := h.eventoDoOrganizador(c)
	if !ok {
		return
	}
	cont, err := h.svc.Contagem(e.ID, h.perfis)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao contar"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"aberta": service.VotacaoAberta(e), "aberta_em": e.VotacaoAbertaEm, "fechada_em": e.VotacaoFechadaEm,
		"devolutiva_liberada": e.DevolutivaLiberadaEm != nil, "contagem": cont,
	})
}

// DefinirVotacao é POST /org/eventos/:id/votacao {"aberta": bool}.
func (h *ConcursoPublicoHandler) DefinirVotacao(c *gin.Context) {
	e, ok := h.eventoDoOrganizador(c)
	if !ok {
		return
	}
	var req struct {
		Aberta bool `json:"aberta"`
	}
	_ = c.ShouldBindJSON(&req)
	if err := h.svc.DefinirVotacao(e.ID, req.Aberta); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao salvar"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"aberta": req.Aberta})
}

// DefinirDevolutiva é POST /org/eventos/:id/devolutiva {"liberar": bool}.
func (h *ConcursoPublicoHandler) DefinirDevolutiva(c *gin.Context) {
	e, ok := h.eventoDoOrganizador(c)
	if !ok {
		return
	}
	var req struct {
		Liberar bool `json:"liberar"`
	}
	_ = c.ShouldBindJSON(&req)
	if err := h.svc.DefinirDevolutiva(e.ID, req.Liberar); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao salvar"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"devolutiva_liberada": req.Liberar})
}
