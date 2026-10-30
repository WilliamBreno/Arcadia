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

// Galeria oficial do evento (item 8.7).
type GaleriaHandler struct {
	svc     *service.GaleriaService
	eventos *repository.EventoRepository
	org     *OrganizadorHandler
}

func NovoGaleriaHandler(s *service.GaleriaService, e *repository.EventoRepository, org *OrganizadorHandler) *GaleriaHandler {
	return &GaleriaHandler{svc: s, eventos: e, org: org}
}

func (h *GaleriaHandler) erro(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrFotoNaoEncontrada), errors.Is(err, service.ErrRemocaoNaoEncontra):
		c.JSON(http.StatusNotFound, gin.H{"erro": err.Error()})
	case errors.Is(err, service.ErrSemAcessoGaleria), errors.Is(err, service.ErrEventoNaoPertenceAoOrganizador):
		c.JSON(http.StatusForbidden, gin.H{"erro": err.Error()})
	case errors.Is(err, service.ErrFotoGrande):
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"erro": err.Error()})
	case errors.Is(err, service.ErrFotoInvalida), errors.Is(err, service.ErrLimiteFotos), errors.Is(err, service.ErrMarcacaoSoAdulto),
		errors.Is(err, service.ErrUsuarioNaoEncontrado):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"erro": err.Error()})
	default:
		slog.Error("galeria", "erro", err)
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro na galeria"})
	}
}

func (h *GaleriaHandler) evento(c *gin.Context) (int64, bool) {
	e, err := h.eventos.BuscarPorSlug(c.Param("slug"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"erro": "evento não encontrado"})
		return 0, false
	}
	return e.ID, true
}

type fotoResposta struct {
	ID       int64            `json:"id"`
	URL      string           `json:"url"`
	Credito  string           `json:"credito"`
	Marcados []marcadoReposta `json:"marcados"`
}

type marcadoReposta struct {
	Nome string `json:"nome"`
	Slug string `json:"slug"`
}

// Publica é GET /eventos/:slug/galeria.
func (h *GaleriaHandler) Publica(c *gin.Context) {
	e, err := h.eventos.BuscarPublicoPorSlug(c.Param("slug"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"erro": "evento não encontrado"})
		return
	}
	fs, err := h.svc.Publica(e.ID)
	if err != nil {
		h.erro(c, err)
		return
	}
	out := make([]fotoResposta, 0, len(fs))
	for _, f := range fs {
		m := make([]marcadoReposta, 0, len(f.Marcados))
		for _, x := range f.Marcados {
			m = append(m, marcadoReposta{Nome: x.Nome, Slug: x.Slug})
		}
		out = append(out, fotoResposta{ID: f.ID, URL: f.URL, Credito: f.Credito, Marcados: m})
	}
	c.JSON(http.StatusOK, out)
}

// Eu é GET /eventos/:slug/galeria/eu: pode enviar? em quais fotos me marquei?
func (h *GaleriaHandler) Eu(c *gin.Context) {
	id, ok := h.evento(c)
	if !ok {
		return
	}
	uid := c.GetInt64(middleware.ChaveContextoUsuarioID)
	marcadas := h.svc.MinhasMarcacoes(uid, id)
	if marcadas == nil {
		marcadas = []int64{}
	}
	c.JSON(http.StatusOK, gin.H{"pode_enviar": h.svc.PodeEnviar(uid, id), "marcadas": marcadas})
}

// Enviar é POST /eventos/:slug/galeria (multipart, campo "arquivo").
func (h *GaleriaHandler) Enviar(c *gin.Context) {
	id, ok := h.evento(c)
	if !ok {
		return
	}
	arquivo, cab, err := c.Request.FormFile("arquivo")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "envie a foto no campo 'arquivo'"})
		return
	}
	defer arquivo.Close()
	f, err := h.svc.Enviar(c.GetInt64(middleware.ChaveContextoUsuarioID), id, arquivo, cab.Size)
	if err != nil {
		h.erro(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": f.ID, "url": f.URL})
}

// Excluir é DELETE /eventos/:slug/galeria/:fid.
func (h *GaleriaHandler) Excluir(c *gin.Context) {
	id, ok := h.evento(c)
	if !ok {
		return
	}
	fid, ok := paramInt(c, "fid")
	if !ok {
		return
	}
	if err := h.svc.Excluir(c.GetInt64(middleware.ChaveContextoUsuarioID), id, fid); err != nil {
		h.erro(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// Painel é GET /eventos/:slug/galeria/painel (organizador e fotógrafos).
func (h *GaleriaHandler) Painel(c *gin.Context) {
	id, ok := h.evento(c)
	if !ok {
		return
	}
	p, err := h.svc.Painel(c.GetInt64(middleware.ChaveContextoUsuarioID), id)
	if err != nil {
		h.erro(c, err)
		return
	}
	type foto struct {
		ID     int64  `json:"id"`
		URL    string `json:"url"`
		Oculta bool   `json:"oculta"`
	}
	type remocao struct {
		ID       int64     `json:"id"`
		FotoID   int64     `json:"foto_id"`
		URL      string    `json:"url"`
		Motivo   string    `json:"motivo"`
		CriadoEm time.Time `json:"criado_em"`
	}
	fotos := make([]foto, 0, len(p.Fotos))
	for _, f := range p.Fotos {
		if f.URL != "" {
			fotos = append(fotos, foto{ID: f.ID, URL: f.URL, Oculta: f.Oculta})
		}
	}
	rems := make([]remocao, 0, len(p.Remocoes))
	for _, r := range p.Remocoes {
		rems = append(rems, remocao{ID: r.ID, FotoID: r.FotoID, URL: r.URL, Motivo: r.Motivo, CriadoEm: r.CriadoEm})
	}
	c.JSON(http.StatusOK, gin.H{"fotos": fotos, "remocoes": rems, "limite": service.MaxFotosPorEvento})
}

// DecidirRemocao é POST /eventos/:slug/galeria/remocoes/:rid.
func (h *GaleriaHandler) DecidirRemocao(c *gin.Context) {
	id, ok := h.evento(c)
	if !ok {
		return
	}
	rid, ok := paramInt(c, "rid")
	if !ok {
		return
	}
	var req struct {
		Manter bool `json:"manter"`
	}
	_ = c.ShouldBindJSON(&req)
	if err := h.svc.DecidirRemocao(c.GetInt64(middleware.ChaveContextoUsuarioID), id, rid, req.Manter); err != nil {
		h.erro(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// Marcar é POST /galeria/fotos/:id/marcar {"marcar": true|false}.
func (h *GaleriaHandler) Marcar(c *gin.Context) {
	fid, ok := idDaURL(c)
	if !ok {
		return
	}
	var req struct {
		Marcar bool `json:"marcar"`
	}
	_ = c.ShouldBindJSON(&req)
	if err := h.svc.Marcar(c.GetInt64(middleware.ChaveContextoUsuarioID), fid, req.Marcar); err != nil {
		h.erro(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// PedirRemocao é POST /galeria/fotos/:id/remocao (sem login).
func (h *GaleriaHandler) PedirRemocao(c *gin.Context) {
	fid, ok := idDaURL(c)
	if !ok {
		return
	}
	var req struct {
		Motivo string `json:"motivo"`
	}
	_ = c.ShouldBindJSON(&req)
	if err := h.svc.PedirRemocao(nil, fid, req.Motivo); err != nil {
		h.erro(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// Fotógrafos (organizador): GET/POST /org/eventos/:id/fotografos, DELETE .../:usuarioId.
func (h *GaleriaHandler) Fotografos(c *gin.Context) {
	org, ok := h.org.ObterOrganizadorAtual(c)
	if !ok {
		return
	}
	id, ok := idDaURL(c)
	if !ok {
		return
	}
	fs, err := h.svc.Fotografos(org.ID, id)
	if err != nil {
		h.erro(c, err)
		return
	}
	type f struct {
		UsuarioID int64  `json:"usuario_id"`
		Nome      string `json:"nome"`
		Email     string `json:"email"`
	}
	out := make([]f, 0, len(fs))
	for _, x := range fs {
		out = append(out, f{UsuarioID: x.UsuarioID, Nome: x.Nome, Email: x.Email})
	}
	c.JSON(http.StatusOK, out)
}

func (h *GaleriaHandler) AdicionarFotografo(c *gin.Context) {
	org, ok := h.org.ObterOrganizadorAtual(c)
	if !ok {
		return
	}
	id, ok := idDaURL(c)
	if !ok {
		return
	}
	var req struct {
		Email string `json:"email" binding:"required,email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "informe o e-mail do fotógrafo"})
		return
	}
	f, err := h.svc.AdicionarFotografo(org.ID, id, req.Email)
	if err != nil {
		h.erro(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"usuario_id": f.UsuarioID, "nome": f.Nome, "email": f.Email})
}

func (h *GaleriaHandler) RemoverFotografo(c *gin.Context) {
	org, ok := h.org.ObterOrganizadorAtual(c)
	if !ok {
		return
	}
	id, ok := idDaURL(c)
	if !ok {
		return
	}
	uid, ok := paramInt(c, "usuarioId")
	if !ok {
		return
	}
	if err := h.svc.RemoverFotografo(org.ID, id, uid); err != nil {
		h.erro(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
