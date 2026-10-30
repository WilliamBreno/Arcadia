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

// Certificados e pesquisa pós-evento (item 9.7).
type PosEventoHandler struct {
	svc      *service.PosEventoService
	eventos  *repository.EventoRepository
	usuarios *repository.UsuarioRepository
	org      *OrganizadorHandler
}

func NovoPosEventoHandler(s *service.PosEventoService, e *repository.EventoRepository, u *repository.UsuarioRepository, org *OrganizadorHandler) *PosEventoHandler {
	return &PosEventoHandler{svc: s, eventos: e, usuarios: u, org: org}
}

func (h *PosEventoHandler) erro(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrCertificadoNaoEncontrado), errors.Is(err, service.ErrPesquisaInvalida):
		c.JSON(http.StatusNotFound, gin.H{"erro": err.Error()})
	case errors.Is(err, service.ErrNomeJaCorrigido), errors.Is(err, service.ErrPesquisaFechada):
		c.JSON(http.StatusConflict, gin.H{"erro": err.Error()})
	case errors.Is(err, service.ErrDadosCertificado), errors.Is(err, service.ErrEventoNaoTerminou):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"erro": err.Error()})
	default:
		slog.Error("pós-evento", "erro", err)
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao processar"})
	}
}

// ---------- público ----------

// Validar é GET /certificados/:codigo[?t=token].
func (h *PosEventoHandler) Validar(c *gin.Context) {
	r, err := h.svc.Validar(c.Param("codigo"), c.Query("t"))
	if err != nil {
		h.erro(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, r)
}

// PDF é GET /certificados/:codigo/pdf.
func (h *PosEventoHandler) PDF(c *gin.Context) {
	dados, nome, err := h.svc.PDF(c.Param("codigo"))
	if err != nil {
		h.erro(c, err)
		return
	}
	c.Header("Content-Disposition", `attachment; filename="`+nome+`"`)
	c.Header("Cache-Control", "no-store")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Data(http.StatusOK, "application/pdf", dados)
}

// CorrigirNome é POST /certificados/:codigo/nome {t, nome} (link do e-mail).
func (h *PosEventoHandler) CorrigirNome(c *gin.Context) {
	var req struct {
		T    string `json:"t"`
		Nome string `json:"nome"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "dados inválidos"})
		return
	}
	if err := h.svc.CorrigirNome(c.Param("codigo"), req.T, 0, req.Nome); err != nil {
		h.erro(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// CorrigirNomeLogado é POST /me/certificados/:codigo/nome {nome}.
func (h *PosEventoHandler) CorrigirNomeLogado(c *gin.Context) {
	var req struct {
		Nome string `json:"nome"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "dados inválidos"})
		return
	}
	usuarioID := c.GetInt64(middleware.ChaveContextoUsuarioID)
	// Certificado do ingresso pelo e-mail da conta também vale.
	token := ""
	if u, err := h.usuarios.BuscarPorID(usuarioID); err == nil {
		if lista, err := h.svc.Meus(usuarioID, u.Email); err == nil {
			for _, ce := range lista {
				if ce.Codigo == c.Param("codigo") {
					token = h.svc.TokenNome(ce.Codigo)
				}
			}
		}
	}
	if err := h.svc.CorrigirNome(c.Param("codigo"), token, usuarioID, req.Nome); err != nil {
		h.erro(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// Meus é GET /me/certificados.
func (h *PosEventoHandler) Meus(c *gin.Context) {
	usuarioID := c.GetInt64(middleware.ChaveContextoUsuarioID)
	u, err := h.usuarios.BuscarPorID(usuarioID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"erro": "sessão inválida"})
		return
	}
	lista, err := h.svc.Meus(usuarioID, u.Email)
	if err != nil {
		h.erro(c, err)
		return
	}
	c.JSON(http.StatusOK, lista)
}

// Pesquisa é GET /pesquisa/:token.
func (h *PosEventoHandler) Pesquisa(c *gin.Context) {
	p, err := h.svc.Pesquisa(c.Param("token"))
	if err != nil {
		h.erro(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, p)
}

// Responder é POST /pesquisa/:token {nota, comentario}.
func (h *PosEventoHandler) Responder(c *gin.Context) {
	var req struct {
		Nota       *int   `json:"nota"`
		Comentario string `json:"comentario"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Nota == nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "escolha uma nota de 0 a 10"})
		return
	}
	if err := h.svc.Responder(c.Param("token"), *req.Nota, req.Comentario); err != nil {
		h.erro(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ---------- organizador ----------

func (h *PosEventoHandler) eventoDoOrg(c *gin.Context) (*domain.Evento, bool) {
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
		c.JSON(http.StatusForbidden, gin.H{"erro": "sem acesso a este evento"})
		return nil, false
	}
	return e, true
}

type configCertificadoJSON struct {
	Ativo           bool   `json:"ativo"`
	Publico         bool   `json:"publico"`
	Participantes   bool   `json:"participantes"`
	Jurados         bool   `json:"jurados"`
	Equipe          bool   `json:"equipe"`
	CargaHorariaMin int    `json:"carga_horaria_min"`
	Texto           string `json:"texto"`
	AssinaturaNome  string `json:"assinatura_nome"`
	AssinaturaCargo string `json:"assinatura_cargo"`
}

// OrgResumo é GET /org/eventos/:id/certificados (config + emitidos).
func (h *PosEventoHandler) OrgResumo(c *gin.Context) {
	e, ok := h.eventoDoOrg(c)
	if !ok {
		return
	}
	cfg := h.svc.Config(e.ID)
	r, err := h.svc.Resumo(e)
	if err != nil {
		h.erro(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"config": configCertificadoJSON{Ativo: cfg.Ativo, Publico: cfg.Publico, Participantes: cfg.Participantes, Jurados: cfg.Jurados, Equipe: cfg.Equipe,
			CargaHorariaMin: cfg.CargaHorariaMin, Texto: cfg.Texto, AssinaturaNome: cfg.AssinaturaNome, AssinaturaCargo: cfg.AssinaturaCargo},
		"resumo": r,
	})
}

// OrgSalvarConfig é PUT /org/eventos/:id/certificados/config.
func (h *PosEventoHandler) OrgSalvarConfig(c *gin.Context) {
	e, ok := h.eventoDoOrg(c)
	if !ok {
		return
	}
	var req configCertificadoJSON
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "dados inválidos"})
		return
	}
	err := h.svc.SalvarConfig(service.ConfigCertificado{EventoID: e.ID, Ativo: req.Ativo, Publico: req.Publico, Participantes: req.Participantes,
		Jurados: req.Jurados, Equipe: req.Equipe, CargaHorariaMin: req.CargaHorariaMin, Texto: req.Texto,
		AssinaturaNome: req.AssinaturaNome, AssinaturaCargo: req.AssinaturaCargo})
	if err != nil {
		h.erro(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// OrgEmitir é POST /org/eventos/:id/certificados/emitir (depois do fim;
// o e-mail continua saindo pelo job).
func (h *PosEventoHandler) OrgEmitir(c *gin.Context) {
	e, ok := h.eventoDoOrg(c)
	if !ok {
		return
	}
	if !h.svc.Config(e.ID).Ativo {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"erro": "ligue os certificados primeiro"})
		return
	}
	n, err := h.svc.Emitir(e, time.Now())
	if err != nil {
		h.erro(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"emitidos": n})
}

// OrgPesquisa é GET /org/eventos/:id/pesquisa.
func (h *PosEventoHandler) OrgPesquisa(c *gin.Context) {
	e, ok := h.eventoDoOrg(c)
	if !ok {
		return
	}
	r, err := h.svc.ResultadoPesquisa(e.ID)
	if err != nil {
		h.erro(c, err)
		return
	}
	c.Header("X-Total-Respostas", strconv.FormatInt(r.Respostas, 10))
	c.JSON(http.StatusOK, r)
}
