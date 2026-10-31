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

// Item 10.5: criação rápida (IA e importação), recomendações e feeds.
type CriacaoRapidaHandler struct {
	svc           *service.CriacaoRapidaService
	org           *OrganizadorHandler
	eventos       *repository.EventoRepository
	locais        *repository.LocalRepository
	organizadores *repository.OrganizadorRepository
	publicos      *PublicoHandler
	frontendURL   string
}

func NovoCriacaoRapidaHandler(s *service.CriacaoRapidaService, org *OrganizadorHandler, e *repository.EventoRepository, l *repository.LocalRepository,
	o *repository.OrganizadorRepository, p *PublicoHandler, frontendURL string) *CriacaoRapidaHandler {
	return &CriacaoRapidaHandler{svc: s, org: org, eventos: e, locais: l, organizadores: o, publicos: p, frontendURL: frontendURL}
}

func (h *CriacaoRapidaHandler) erro(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrIADesligada):
		c.JSON(http.StatusServiceUnavailable, gin.H{"erro": err.Error()})
	case errors.Is(err, service.ErrLimiteIA):
		c.JSON(http.StatusTooManyRequests, gin.H{"erro": err.Error()})
	case errors.Is(err, service.ErrDadosRascunho), errors.Is(err, service.ErrLinkImportarInvalido), errors.Is(err, service.ErrLinkSemEvento):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"erro": err.Error()})
	case errors.Is(err, service.ErrIAIndisponivel), errors.Is(err, service.ErrLinkInacessivel):
		c.JSON(http.StatusBadGateway, gin.H{"erro": err.Error()})
	default:
		slog.Error("criação rápida", "erro", err)
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao processar"})
	}
}

// Situacao é GET /org/criacao-rapida: IA ligada e quantos rascunhos restam hoje.
func (h *CriacaoRapidaHandler) Situacao(c *gin.Context) {
	org, ok := h.org.ObterOrganizadorAtual(c)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, gin.H{"ia_ligada": h.svc.IALigada(), "restantes_hoje": h.svc.RestantesHoje(org.ID), "limite_dia": service.LimiteIAPorDia})
}

// GerarIA é POST /org/criacao-rapida/ia.
func (h *CriacaoRapidaHandler) GerarIA(c *gin.Context) {
	org, ok := h.org.ObterOrganizadorAtual(c)
	if !ok {
		return
	}
	var req service.EntradaRascunho
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "dados inválidos"})
		return
	}
	r, err := h.svc.GerarRascunho(c.Request.Context(), org.ID, c.GetInt64(middleware.ChaveContextoUsuarioID), req)
	if err != nil {
		h.erro(c, err)
		return
	}
	c.JSON(http.StatusOK, r)
}

// Importar é POST /org/criacao-rapida/importar {url}.
func (h *CriacaoRapidaHandler) Importar(c *gin.Context) {
	if _, ok := h.org.ObterOrganizadorAtual(c); !ok {
		return
	}
	var req struct {
		URL string `json:"url"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "informe o link"})
		return
	}
	ev, err := h.svc.ImportarLink(c.Request.Context(), req.URL)
	if err != nil {
		h.erro(c, err)
		return
	}
	c.JSON(http.StatusOK, ev)
}

// Recomendados é GET /eventos/:slug/recomendados.
func (h *CriacaoRapidaHandler) Recomendados(c *gin.Context) {
	e, err := h.eventos.BuscarPublicoPorSlug(c.Param("slug"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"erro": "evento não encontrado"})
		return
	}
	tambem, err := h.eventos.TambemCompraram(e.ID, 3, 6)
	if err != nil {
		h.erro(c, err)
		return
	}
	cidade, uf := "", ""
	if e.LocalID != nil {
		if l, err := h.locais.BuscarPorID(*e.LocalID); err == nil {
			cidade, uf = l.Cidade, l.UF
		}
	}
	parecidos, err := h.eventos.Parecidos(e, cidade, uf, 6)
	if err != nil {
		h.erro(c, err)
		return
	}
	// O que já está em "também compraram" não se repete em "parecidos".
	vistos := map[int64]bool{}
	for _, x := range tambem {
		vistos[x.ID] = true
	}
	filtrados := parecidos[:0]
	for _, x := range parecidos {
		if !vistos[x.ID] {
			filtrados = append(filtrados, x)
		}
	}
	a, err := h.publicos.paraItensPublicos(tambem, true)
	if err != nil {
		h.erro(c, err)
		return
	}
	b, err := h.publicos.paraItensPublicos(filtrados, true)
	if err != nil {
		h.erro(c, err)
		return
	}
	c.Header("Cache-Control", "public, max-age=600")
	c.JSON(http.StatusOK, gin.H{"tambem_compraram": a, "parecidos": b})
}

// ParaVoce é GET /me/para-voce.
func (h *CriacaoRapidaHandler) ParaVoce(c *gin.Context) {
	eventos, err := h.eventos.ParaVoce(c.GetInt64(middleware.ChaveContextoUsuarioID), 8)
	if err != nil {
		h.erro(c, err)
		return
	}
	itens, err := h.publicos.paraItensPublicos(eventos, true)
	if err != nil {
		h.erro(c, err)
		return
	}
	c.JSON(http.StatusOK, itens)
}

func (h *CriacaoRapidaHandler) comLocal(eventos []domain.Evento) []service.EventoFeed {
	out := make([]service.EventoFeed, 0, len(eventos))
	for i := range eventos {
		onde := "Online"
		if eventos[i].LocalID != nil {
			if l, err := h.locais.BuscarPorID(*eventos[i].LocalID); err == nil {
				onde = service.EnderecoDoLocal(l)
			}
		}
		out = append(out, service.EventoFeed{Evento: &eventos[i], Onde: onde})
	}
	return out
}

func (h *CriacaoRapidaHandler) responderFeed(c *gin.Context, formato, nome, link, descricao string, eventos []domain.Evento) {
	itens := h.comLocal(eventos)
	c.Header("Cache-Control", "public, max-age=1800")
	if formato == "ics" {
		c.Data(http.StatusOK, "text/calendar; charset=utf-8", []byte(service.CalendarioICS(nome, itens, h.frontendURL, time.Now())))
		return
	}
	corpo, err := service.FeedRSS(nome, link, descricao, itens, h.frontendURL)
	if err != nil {
		h.erro(c, err)
		return
	}
	c.Data(http.StatusOK, "application/rss+xml; charset=utf-8", corpo)
}

const limiteFeed = 100

// FeedOrganizador é GET /feeds/organizadores/:slug/eventos.rss e .ics.
func (h *CriacaoRapidaHandler) FeedOrganizador(formato string) gin.HandlerFunc {
	return func(c *gin.Context) {
		o, err := h.organizadores.BuscarPorSlug(c.Param("slug"))
		if err != nil {
			c.String(http.StatusNotFound, "organizador não encontrado")
			return
		}
		eventos, err := h.eventos.FeedOrganizador(o.ID, limiteFeed)
		if err != nil {
			h.erro(c, err)
			return
		}
		h.responderFeed(c, formato, "Eventos de "+o.NomePublico, h.frontendURL+"/o/"+o.Slug, "Próximos eventos de "+o.NomePublico+" no Evve", eventos)
	}
}

// FeedCidade é GET /feeds/cidades/:slug/eventos.rss e .ics.
func (h *CriacaoRapidaHandler) FeedCidade(formato string) gin.HandlerFunc {
	return func(c *gin.Context) {
		cidades, err := h.eventos.ListarCidadesComEventos()
		if err != nil {
			h.erro(c, err)
			return
		}
		atual := resolverCidade(cidades, c.Param("slug"))
		if atual == nil {
			c.String(http.StatusNotFound, "nenhum evento nesta cidade")
			return
		}
		eventos, err := h.eventos.FeedCidade(atual.Cidade, atual.UF, limiteFeed)
		if err != nil {
			h.erro(c, err)
			return
		}
		h.responderFeed(c, formato, "Eventos em "+atual.Cidade, h.frontendURL+"/eventos/"+atual.Slug,
			"Próximos eventos em "+atual.Cidade+"/"+atual.UF+" no Evve ("+strconv.Itoa(len(eventos))+")", eventos)
	}
}
