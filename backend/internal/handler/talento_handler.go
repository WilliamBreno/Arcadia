package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/WilliamBreno/Arcadia/backend/internal/middleware"
	"github.com/WilliamBreno/Arcadia/backend/internal/service"
)

// Perfil público do talento (item 8.1).
type TalentoHandler struct {
	svc *service.PerfilTalentoService
}

func NovoTalentoHandler(s *service.PerfilTalentoService) *TalentoHandler {
	return &TalentoHandler{svc: s}
}

type meuPerfilResposta struct {
	Perfil        *service.DadosPerfil   `json:"perfil"`
	Slug          string                 `json:"slug,omitempty"`
	Situacao      service.SituacaoPerfil `json:"situacao"`
	Autorizado    bool                   `json:"autorizado"`
	AutorizaFoto  bool                   `json:"autoriza_foto"`
	Participacoes []service.Participacao `json:"participacoes"`
}

func (h *TalentoHandler) responderMeu(c *gin.Context, uid int64, status int) {
	parts, _ := h.svc.MinhasParticipacoes(uid)
	if parts == nil {
		parts = []service.Participacao{}
	}
	p, sit, err := h.svc.Meu(uid)
	if errors.Is(err, service.ErrPerfilNaoEncontrado) {
		c.JSON(status, meuPerfilResposta{Participacoes: parts})
		return
	}
	publico := p.Publico
	c.JSON(status, meuPerfilResposta{
		Perfil: &service.DadosPerfil{NomeArtistico: p.NomeArtistico, Bio: p.Bio, FotoURL: p.FotoURL, Cidade: p.Cidade, UF: p.UF,
			Instagram: p.Instagram, TikTok: p.TikTok, YouTube: p.YouTube, DataNascimento: p.DataNascimento.Format("2006-01-02"),
			Publico: &publico, ResponsavelNome: p.ResponsavelNome, ResponsavelEmail: p.ResponsavelEmail},
		Slug: p.Slug, Situacao: sit, Autorizado: p.AutorizadoEm != nil, AutorizaFoto: p.AutorizaFoto, Participacoes: parts,
	})
}

// Meu é GET /me/talento.
func (h *TalentoHandler) Meu(c *gin.Context) {
	h.responderMeu(c, c.GetInt64(middleware.ChaveContextoUsuarioID), http.StatusOK)
}

// Salvar é PUT /me/talento.
func (h *TalentoHandler) Salvar(c *gin.Context) {
	var d service.DadosPerfil
	if err := c.ShouldBindJSON(&d); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "dados inválidos"})
		return
	}
	uid := c.GetInt64(middleware.ChaveContextoUsuarioID)
	if _, _, err := h.svc.Salvar(uid, d); err != nil {
		if errors.Is(err, service.ErrPerfilInvalido) || errors.Is(err, service.ErrMenorDe12) {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"erro": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao salvar o perfil"})
		return
	}
	h.responderMeu(c, uid, http.StatusOK)
}

// Ocultar é POST /me/talento/participacoes/:ficha {"ocultar": bool}.
func (h *TalentoHandler) Ocultar(c *gin.Context) {
	fichaID, err := strconv.ParseInt(c.Param("ficha"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "participação inválida"})
		return
	}
	var req struct {
		Ocultar bool `json:"ocultar"`
	}
	_ = c.ShouldBindJSON(&req)
	uid := c.GetInt64(middleware.ChaveContextoUsuarioID)
	if err := h.svc.Ocultar(uid, fichaID, req.Ocultar); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"erro": "participação não encontrada"})
		return
	}
	h.responderMeu(c, uid, http.StatusOK)
}

// Publico é GET /talentos/:slug.
func (h *TalentoHandler) Publico(c *gin.Context) {
	p, err := h.svc.Publico(c.Param("slug"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"erro": "perfil não encontrado"})
		return
	}
	if !p.Indexar {
		c.Header("X-Robots-Tag", "noindex, nofollow")
	}
	c.JSON(http.StatusOK, p)
}

// VerAutorizacao é GET /talentos/autorizacao?t= (link do e-mail do responsável).
func (h *TalentoHandler) VerAutorizacao(c *gin.Context) {
	r, err := h.svc.VerAutorizacao(c.Query("t"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"erro": "link inválido ou substituído por um mais novo"})
		return
	}
	c.Header("X-Robots-Tag", "noindex, nofollow")
	c.JSON(http.StatusOK, r)
}

// Decidir é POST /talentos/autorizacao {"t", "autorizar", "foto"}.
func (h *TalentoHandler) Decidir(c *gin.Context) {
	var req struct {
		T         string `json:"t" binding:"required"`
		Autorizar bool   `json:"autorizar"`
		Foto      bool   `json:"foto"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "link incompleto"})
		return
	}
	r, err := h.svc.Decidir(req.T, req.Autorizar, req.Foto)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"erro": "link inválido ou substituído por um mais novo"})
		return
	}
	c.JSON(http.StatusOK, r)
}
