package handler

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/WilliamBreno/Arcadia/backend/internal/middleware"
	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
	"github.com/WilliamBreno/Arcadia/backend/internal/service"
)

// Seguir, descadastro, calendário e lembretes (item 6.4).
type SeguirHandler struct {
	seguir      *service.SeguirService
	lembretes   *service.LembretesService
	convites    *service.AvaliacaoPublicoService
	eventos     *repository.EventoRepository
	locais      *repository.LocalRepository
	frontendURL string
}

func NovoSeguirHandler(s *service.SeguirService, l *service.LembretesService, e *repository.EventoRepository, lo *repository.LocalRepository, frontendURL string) *SeguirHandler {
	return &SeguirHandler{seguir: s, lembretes: l, eventos: e, locais: lo, frontendURL: frontendURL}
}

type seguirRequest struct {
	Tipo   string `json:"tipo" binding:"required"`
	Alvo   string `json:"alvo" binding:"required"`
	Rotulo string `json:"rotulo"`
}

// Seguir é POST /api/v1/seguindo.
func (h *SeguirHandler) Seguir(c *gin.Context) {
	var req seguirRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "dados inválidos"})
		return
	}
	if err := h.seguir.Seguir(c.GetInt64(middleware.ChaveContextoUsuarioID), req.Tipo, req.Alvo, req.Rotulo); err != nil {
		if errors.Is(err, service.ErrSeguirInvalido) {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"erro": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao seguir"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"seguindo": true})
}

// Deixar é DELETE /api/v1/seguindo?tipo=&alvo=.
func (h *SeguirHandler) Deixar(c *gin.Context) {
	if err := h.seguir.Deixar(c.GetInt64(middleware.ChaveContextoUsuarioID), c.Query("tipo"), c.Query("alvo")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao deixar de seguir"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"seguindo": false})
}

type seguindoResposta struct {
	Tipo     string    `json:"tipo"`
	Alvo     string    `json:"alvo"`
	Rotulo   string    `json:"rotulo"`
	CriadoEm time.Time `json:"criado_em"`
}

// Meus é GET /api/v1/me/seguindo.
func (h *SeguirHandler) Meus(c *gin.Context) {
	lista, err := h.seguir.Listar(c.GetInt64(middleware.ChaveContextoUsuarioID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao listar"})
		return
	}
	resp := make([]seguindoResposta, 0, len(lista))
	for _, s := range lista {
		resp = append(resp, seguindoResposta{Tipo: s.Tipo, Alvo: s.Alvo, Rotulo: s.Rotulo, CriadoEm: s.CriadoEm})
	}
	c.JSON(http.StatusOK, resp)
}

type descadastroRequest struct {
	U    string `json:"u" binding:"required"`
	Tipo string `json:"tipo" binding:"required"`
	Alvo string `json:"alvo" binding:"required"`
	T    string `json:"t" binding:"required"`
	Tudo bool   `json:"tudo"`
}

// Descadastrar é POST /api/v1/descadastrar (link do e-mail, sem login). É
// POST de propósito: leitores de e-mail que abrem links sozinhos (antivírus)
// não descadastram ninguém por engano — a página pede um clique.
func (h *SeguirHandler) Descadastrar(c *gin.Context) {
	var req descadastroRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "link incompleto"})
		return
	}
	uid, err := strconv.ParseInt(req.U, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "link inválido"})
		return
	}
	if err := h.seguir.Descadastrar(uid, req.Tipo, req.Alvo, req.T, req.Tudo); err != nil {
		if errors.Is(err, service.ErrLinkInvalido) {
			c.JSON(http.StatusForbidden, gin.H{"erro": "link inválido ou alterado"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao descadastrar"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// Calendario é GET /api/v1/eventos/:slug/calendario.ics.
func (h *SeguirHandler) Calendario(c *gin.Context) {
	e, err := h.eventos.BuscarPublicoPorSlug(c.Param("slug"))
	if err != nil {
		c.String(http.StatusNotFound, "evento não encontrado")
		return
	}
	onde := "Online"
	if e.LocalID != nil {
		if l, err := h.locais.BuscarPorID(*e.LocalID); err == nil {
			onde = service.EnderecoDoLocal(l)
		}
	}
	ics, ok := service.ArquivoICS(e, onde, h.frontendURL, time.Now())
	if !ok {
		c.String(http.StatusNotFound, "evento sem data")
		return
	}
	c.Header("Content-Disposition", `attachment; filename="`+e.Slug+`.ics"`)
	c.Data(http.StatusOK, "text/calendar; charset=utf-8", []byte(ics))
}

// DefinirConvites liga o convite para avaliar depois do evento (6.5).
func (h *SeguirHandler) DefinirConvites(s *service.AvaliacaoPublicoService) { h.convites = s }

// Lembretes é o job POST /api/v1/jobs/lembretes (de hora em hora).
func (h *SeguirHandler) Lembretes(c *gin.Context) {
	agora := time.Now()
	n, err := h.lembretes.Enviar(agora)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao enviar lembretes"})
		return
	}
	convites := 0
	if h.convites != nil {
		// Item 6.5: "Como foi?" para quem fez check-in, depois do evento.
		if convites, err = h.convites.ConvidarParaAvaliar(agora); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao enviar convites de avaliação"})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"enviados": n, "convites_avaliacao": convites})
}
