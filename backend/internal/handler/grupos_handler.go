package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/WilliamBreno/Arcadia/backend/internal/middleware"
	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
	"github.com/WilliamBreno/Arcadia/backend/internal/service"
)

// Item 11.1: grupos e artistas solo.
type GruposHandler struct {
	organizadorHandler *OrganizadorHandler
	eventos            *repository.EventoRepository
	service            *service.GruposService
}

func NovoGruposHandler(o *OrganizadorHandler, eventos *repository.EventoRepository, s *service.GruposService) *GruposHandler {
	return &GruposHandler{organizadorHandler: o, eventos: eventos, service: s}
}

func (h *GruposHandler) erro(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrIBGEArtista):
		c.JSON(http.StatusServiceUnavailable, gin.H{"erro": err.Error()})
	case errors.Is(err, service.ErrContratacaoEstado):
		c.JSON(http.StatusConflict, gin.H{"erro": err.Error()})
	case errors.Is(err, service.ErrArtistaNaoEncontrado), errors.Is(err, service.ErrConviteArtista), errors.Is(err, service.ErrLinkAutorizacao):
		c.JSON(http.StatusNotFound, gin.H{"erro": err.Error()})
	case errors.Is(err, service.ErrArtistaSemPermissao), errors.Is(err, service.ErrConviteOutroEmail):
		c.JSON(http.StatusForbidden, gin.H{"erro": err.Error()})
	case errors.Is(err, service.ErrIntegranteDuplicado), errors.Is(err, service.ErrLiderNaoSai), errors.Is(err, service.ErrAtracaoInvalida):
		c.JSON(http.StatusConflict, gin.H{"erro": err.Error()})
	case errors.Is(err, service.ErrArtistaInvalido), errors.Is(err, service.ErrIntegranteInvalido), errors.Is(err, service.ErrMidiaInvalida),
		errors.Is(err, service.ErrContratacao),
		errors.Is(err, service.ErrGestaoArtista), errors.Is(err, service.ErrAgendaArtista),
		errors.Is(err, service.ErrLimiteArtistas), errors.Is(err, service.ErrLimiteIntegrantes), errors.Is(err, service.ErrLimiteMidias),
		errors.Is(err, service.ErrSoloUmIntegrante):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"erro": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro interno"})
	}
}

func usuarioDe(c *gin.Context) int64 { return c.GetInt64(middleware.ChaveContextoUsuarioID) }

func (h *GruposHandler) Excluir(c *gin.Context) {
	id, ok := paramID(c, "id")
	if !ok {
		return
	}
	if err := h.service.Excluir(usuarioDe(c), id); err != nil {
		if errors.Is(err, service.ErrExcluirArtista) {
			c.JSON(http.StatusConflict, gin.H{"erro": err.Error()})
			return
		}
		h.erro(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func paramID(c *gin.Context, nome string) (int64, bool) {
	id, err := strconv.ParseInt(c.Param(nome), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "id inválido"})
		return 0, false
	}
	return id, true
}

// --- Público ---

// Vitrine é GET /artistas.
func (h *GruposHandler) Vitrine(c *gin.Context) {
	pagina, _ := strconv.Atoi(c.Query("pagina"))
	f := service.FiltrosArtistas{Tipo: c.Query("tipo"), UF: c.Query("uf"), Cidade: c.Query("cidade"), Busca: c.Query("q"), Pagina: pagina, DataLivre: c.Query("livre_em")}
	if v := c.Query("solo"); v == "1" || v == "0" {
		solo := v == "1"
		f.Solo = &solo
	}
	itens, total, err := h.service.Vitrine(f)
	if err != nil {
		h.erro(c, err)
		return
	}
	c.Header("Cache-Control", "public, max-age=60")
	c.JSON(http.StatusOK, gin.H{"itens": itens, "total": total, "tipos": service.TiposGrupo})
}

// Publico é GET /artistas/:slug.
func (h *GruposHandler) Publico(c *gin.Context) {
	p, err := h.service.Publico(c.Param("slug"))
	if err != nil {
		h.erro(c, err)
		return
	}
	c.JSON(http.StatusOK, p)
}

// AtracoesDoEvento é GET /eventos/:slug/atracoes.
func (h *GruposHandler) AtracoesDoEvento(c *gin.Context) {
	e, err := h.eventos.BuscarPublicoPorSlug(c.Param("slug"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"erro": "evento não encontrado"})
		return
	}
	lista, err := h.service.AtracoesPublicas(e.ID)
	if err != nil {
		h.erro(c, err)
		return
	}
	c.JSON(http.StatusOK, lista)
}

// VerConvite é GET /grupos/convites/:token (público) e GET /me/grupos/convites/:token.
func (h *GruposHandler) VerConvite(c *gin.Context) {
	v, err := h.service.VerConvite(c.Param("token"), usuarioDe(c))
	if err != nil {
		h.erro(c, err)
		return
	}
	c.JSON(http.StatusOK, v)
}

// VerAutorizacao é GET /grupos/autorizacao?t=.
func (h *GruposHandler) VerAutorizacao(c *gin.Context) {
	v, err := h.service.VerAutorizacao(c.Query("t"))
	if err != nil {
		h.erro(c, err)
		return
	}
	c.JSON(http.StatusOK, v)
}

// DecidirAutorizacao é POST /grupos/autorizacao.
func (h *GruposHandler) DecidirAutorizacao(c *gin.Context) {
	var req struct {
		Token     string `json:"token"`
		Autorizar bool   `json:"autorizar"`
		Foto      bool   `json:"foto"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "dados inválidos"})
		return
	}
	v, err := h.service.DecidirAutorizacao(req.Token, req.Autorizar, req.Foto)
	if err != nil {
		h.erro(c, err)
		return
	}
	c.JSON(http.StatusOK, v)
}

// --- Conta (líder e integrantes) ---

// Meus é GET /me/grupos.
func (h *GruposHandler) Meus(c *gin.Context) {
	lista, err := h.service.MeusGrupos(usuarioDe(c))
	if err != nil {
		h.erro(c, err)
		return
	}
	c.JSON(http.StatusOK, lista)
}

// Criar é POST /me/grupos.
func (h *GruposHandler) Criar(c *gin.Context) {
	var d service.DadosGrupo
	if err := c.ShouldBindJSON(&d); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "dados inválidos"})
		return
	}
	g, err := h.service.Criar(usuarioDe(c), d)
	if err != nil {
		h.erro(c, err)
		return
	}
	c.JSON(http.StatusCreated, g.ParaLider())
}

func (h *GruposHandler) responderPainel(c *gin.Context, grupoID int64) {
	p, err := h.service.Painel(usuarioDe(c), grupoID)
	if err != nil {
		h.erro(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"grupo": p.Grupo.ParaLider(), "lider": p.Grupo.LiderUsuarioID == usuarioDe(c), "integrantes": p.Integrantes, "midias": p.Midias, "atracoes": p.Atracoes,
		"instagram_codigo": p.InstagramCodigo, "instagram_pedido": p.InstagramPedido, "instagram_verificado": p.InstagramVerificado})
}

// Painel é GET /me/grupos/:id.
func (h *GruposHandler) Painel(c *gin.Context) {
	id, ok := paramID(c, "id")
	if !ok {
		return
	}
	h.responderPainel(c, id)
}

// Atualizar é PUT /me/grupos/:id.
func (h *GruposHandler) Atualizar(c *gin.Context) {
	id, ok := paramID(c, "id")
	if !ok {
		return
	}
	var d service.DadosGrupo
	if err := c.ShouldBindJSON(&d); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "dados inválidos"})
		return
	}
	if _, err := h.service.Atualizar(usuarioDe(c), id, d); err != nil {
		h.erro(c, err)
		return
	}
	h.responderPainel(c, id)
}

// ConvidarIntegrante é POST /me/grupos/:id/integrantes.
func (h *GruposHandler) ConvidarIntegrante(c *gin.Context) {
	id, ok := paramID(c, "id")
	if !ok {
		return
	}
	var d service.DadosIntegrante
	if err := c.ShouldBindJSON(&d); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "dados inválidos"})
		return
	}
	_, token, err := h.service.ConvidarIntegrante(usuarioDe(c), id, d)
	if err != nil {
		h.erro(c, err)
		return
	}
	// token só volta quando o líder convida o próprio e-mail (aceita na hora).
	c.JSON(http.StatusCreated, gin.H{"token_proprio": token})
}

// EditarIntegrante é PUT /me/grupos/:id/integrantes/:integrante.
func (h *GruposHandler) EditarIntegrante(c *gin.Context) {
	id, ok := paramID(c, "id")
	if !ok {
		return
	}
	iid, ok := paramID(c, "integrante")
	if !ok {
		return
	}
	var d service.DadosIntegrante
	if err := c.ShouldBindJSON(&d); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "dados inválidos"})
		return
	}
	if err := h.service.EditarIntegrante(usuarioDe(c), id, iid, d); err != nil {
		h.erro(c, err)
		return
	}
	h.responderPainel(c, id)
}

// RemoverIntegrante é DELETE /me/grupos/:id/integrantes/:integrante.
func (h *GruposHandler) RemoverIntegrante(c *gin.Context) {
	id, ok := paramID(c, "id")
	if !ok {
		return
	}
	iid, ok := paramID(c, "integrante")
	if !ok {
		return
	}
	if err := h.service.RemoverIntegrante(usuarioDe(c), id, iid); err != nil {
		h.erro(c, err)
		return
	}
	h.responderPainel(c, id)
}

// Sair é POST /me/grupos/:id/sair.
func (h *GruposHandler) Sair(c *gin.Context) {
	id, ok := paramID(c, "id")
	if !ok {
		return
	}
	if err := h.service.Sair(usuarioDe(c), id); err != nil {
		h.erro(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// AceitarConvite é POST /me/grupos/convites/:token/aceitar.
func (h *GruposHandler) AceitarConvite(c *gin.Context) {
	var d service.DadosAceite
	if err := c.ShouldBindJSON(&d); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "dados inválidos"})
		return
	}
	i, err := h.service.AceitarConvite(c.Param("token"), usuarioDe(c), d)
	if err != nil {
		h.erro(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"grupo_id": i.GrupoID, "aguardando_responsavel": i.AutorizacaoTokenHash != nil && i.AutorizadoEm == nil})
}

// RecusarConvite é POST /me/grupos/convites/:token/recusar.
func (h *GruposHandler) RecusarConvite(c *gin.Context) {
	if err := h.service.RecusarConvite(c.Param("token"), usuarioDe(c)); err != nil {
		h.erro(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// AdicionarMidia é POST /me/grupos/:id/midias.
func (h *GruposHandler) AdicionarMidia(c *gin.Context) {
	id, ok := paramID(c, "id")
	if !ok {
		return
	}
	var d service.DadosMidia
	if err := c.ShouldBindJSON(&d); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "dados inválidos"})
		return
	}
	if _, err := h.service.AdicionarMidia(usuarioDe(c), id, d); err != nil {
		h.erro(c, err)
		return
	}
	h.responderPainel(c, id)
}

// RemoverMidia é DELETE /me/grupos/:id/midias/:midia.
func (h *GruposHandler) RemoverMidia(c *gin.Context) {
	id, ok := paramID(c, "id")
	if !ok {
		return
	}
	mid, ok := paramID(c, "midia")
	if !ok {
		return
	}
	if err := h.service.RemoverMidia(usuarioDe(c), id, mid); err != nil {
		h.erro(c, err)
		return
	}
	h.responderPainel(c, id)
}

// Instagram é POST /me/grupos/:id/instagram {pedir: bool}: sem pedir só gera o código.
func (h *GruposHandler) Instagram(c *gin.Context) {
	id, ok := paramID(c, "id")
	if !ok {
		return
	}
	var req struct {
		Pedir bool `json:"pedir"`
	}
	_ = c.ShouldBindJSON(&req)
	if _, err := h.service.PedirVerificacaoInstagram(usuarioDe(c), id, !req.Pedir); err != nil {
		h.erro(c, err)
		return
	}
	h.responderPainel(c, id)
}

// DecidirAtracao é POST /me/grupos/:id/atracoes/:atracao {aceitar}.
func (h *GruposHandler) DecidirAtracao(c *gin.Context) {
	id, ok := paramID(c, "id")
	if !ok {
		return
	}
	aid, ok := paramID(c, "atracao")
	if !ok {
		return
	}
	var req struct {
		Aceitar bool `json:"aceitar"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "dados inválidos"})
		return
	}
	if err := h.service.DecidirAtracao(usuarioDe(c), id, aid, req.Aceitar); err != nil {
		h.erro(c, err)
		return
	}
	h.responderPainel(c, id)
}

// --- Organizador (Atrações) ---

func (h *GruposHandler) orgEvento(c *gin.Context) (int64, int64, bool) {
	organizador, ok := h.organizadorHandler.ObterOrganizadorAtual(c)
	if !ok {
		return 0, 0, false
	}
	eventoID, ok := idDaURL(c)
	if !ok {
		return 0, 0, false
	}
	return organizador.ID, eventoID, true
}

// AtracoesOrg é GET /org/eventos/:id/atracoes.
func (h *GruposHandler) AtracoesOrg(c *gin.Context) {
	orgID, eventoID, ok := h.orgEvento(c)
	if !ok {
		return
	}
	lista, err := h.service.AtracoesDoEventoOrg(orgID, eventoID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"erro": "evento não encontrado"})
		return
	}
	c.JSON(http.StatusOK, lista)
}

// ConvidarAtracao é POST /org/eventos/:id/atracoes {grupo_slug}.
func (h *GruposHandler) ConvidarAtracao(c *gin.Context) {
	orgID, eventoID, ok := h.orgEvento(c)
	if !ok {
		return
	}
	var req struct {
		GrupoSlug string `json:"grupo_slug"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "dados inválidos"})
		return
	}
	if _, err := h.service.ConvidarAtracao(orgID, eventoID, usuarioDe(c), req.GrupoSlug); err != nil {
		if errors.Is(err, service.ErrArtistaNaoEncontrado) || errors.Is(err, service.ErrAtracaoInvalida) {
			h.erro(c, err)
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"erro": "evento não encontrado"})
		return
	}
	h.AtracoesOrg(c)
}

// RemoverAtracao é DELETE /org/eventos/:id/atracoes/:grupo.
func (h *GruposHandler) RemoverAtracao(c *gin.Context) {
	orgID, eventoID, ok := h.orgEvento(c)
	if !ok {
		return
	}
	gid, ok := paramID(c, "grupo")
	if !ok {
		return
	}
	if err := h.service.RemoverAtracao(orgID, eventoID, gid); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"erro": "evento não encontrado"})
		return
	}
	h.AtracoesOrg(c)
}

// BuscarParaAtracao é GET /org/grupos/busca?q=.
func (h *GruposHandler) BuscarParaAtracao(c *gin.Context) {
	if _, ok := h.organizadorHandler.ObterOrganizadorAtual(c); !ok {
		return
	}
	lista, err := h.service.BuscarParaAtracao(c.Query("q"))
	if err != nil {
		h.erro(c, err)
		return
	}
	c.JSON(http.StatusOK, lista)
}

// --- Admin (selo do Instagram) ---

// PedidosInstagram é GET /admin/grupos/instagram.
func (h *GruposHandler) PedidosInstagram(c *gin.Context) {
	lista, err := h.service.PedidosInstagram()
	if err != nil {
		h.erro(c, err)
		return
	}
	c.JSON(http.StatusOK, lista)
}

// DecidirInstagram é POST /admin/grupos/:id/instagram {aprovar}.
func (h *GruposHandler) DecidirInstagram(c *gin.Context) {
	id, ok := paramID(c, "id")
	if !ok {
		return
	}
	var req struct {
		Aprovar   bool   `json:"aprovar"`
		Instagram string `json:"instagram"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "dados inválidos"})
		return
	}
	if err := h.service.DecidirInstagram(id, req.Aprovar, req.Instagram); err != nil {
		if errors.Is(err, service.ErrArtistaNaoEncontrado) {
			c.JSON(http.StatusConflict, gin.H{"erro": "o pedido mudou (o Instagram pode ter sido trocado); recarregue a lista"})
			return
		}
		h.erro(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
