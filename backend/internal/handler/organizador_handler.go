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

type OrganizadorHandler struct {
	organizadores *repository.OrganizadorRepository
	service       *service.OrganizadorService
}

func NovoOrganizadorHandler(organizadores *repository.OrganizadorRepository, s *service.OrganizadorService) *OrganizadorHandler {
	return &OrganizadorHandler{organizadores: organizadores, service: s}
}

type organizadorResposta struct {
	ID           int64  `json:"id"`
	NomePublico  string `json:"nome_publico"`
	Slug         string `json:"slug"`
	Descricao    string `json:"descricao"`
	LogoURL      string `json:"logo_url"`
	TipoPessoa   string `json:"tipo_pessoa"`
	Documento    string `json:"documento"`
	ChavePix     string `json:"chave_pix"`
	TipoChavePix string `json:"tipo_chave_pix"`
	Instagram    string `json:"instagram"`
	Site         string `json:"site"`
	Status       string `json:"status"`
}

func paraOrganizadorResposta(o *domain.Organizador) organizadorResposta {
	return organizadorResposta{
		ID:           o.ID,
		NomePublico:  o.NomePublico,
		Slug:         o.Slug,
		Descricao:    o.Descricao,
		LogoURL:      o.LogoURL,
		TipoPessoa:   string(o.TipoPessoa),
		Documento:    o.Documento,
		ChavePix:     o.ChavePix,
		TipoChavePix: o.TipoChavePix,
		Instagram:    o.Instagram,
		Site:         o.Site,
		Status:       string(o.Status),
	}
}

// ObterOrganizadorAtual busca o perfil de organizador do usuário logado.
// Handlers de sub-recursos (locais, eventos, ...) chamam isso para
// garantir escopo por dono antes de qualquer operação.
func (h *OrganizadorHandler) ObterOrganizadorAtual(c *gin.Context) (*domain.Organizador, bool) {
	usuarioID := c.GetInt64(middleware.ChaveContextoUsuarioID)
	organizador, err := h.organizadores.BuscarPorUsuarioID(usuarioID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"erro": "crie seu perfil de organizador primeiro"})
		return nil, false
	}
	return organizador, true
}

type criarPerfilRequest struct {
	NomePublico string `json:"nome_publico" binding:"required,min=2"`
	TipoPessoa  string `json:"tipo_pessoa" binding:"required,oneof=pf pj"`
	Documento   string `json:"documento" binding:"required"`
}

func (h *OrganizadorHandler) CriarPerfil(c *gin.Context) {
	var req criarPerfilRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "dados inválidos"})
		return
	}

	usuarioID := c.GetInt64(middleware.ChaveContextoUsuarioID)
	organizador, err := h.service.CriarPerfil(usuarioID, req.NomePublico, domain.TipoPessoa(req.TipoPessoa), req.Documento)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrOrganizadorJaExiste):
			c.JSON(http.StatusConflict, gin.H{"erro": "você já tem um perfil de organizador"})
		case errors.Is(err, service.ErrDocumentoInvalido):
			c.JSON(http.StatusBadRequest, gin.H{"erro": "documento inválido para o tipo de pessoa"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao criar perfil"})
		}
		return
	}

	c.JSON(http.StatusCreated, paraOrganizadorResposta(organizador))
}

func (h *OrganizadorHandler) MeuPerfil(c *gin.Context) {
	organizador, ok := h.ObterOrganizadorAtual(c)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, paraOrganizadorResposta(organizador))
}

type atualizarPerfilRequest struct {
	NomePublico  string `json:"nome_publico" binding:"required,min=2"`
	Descricao    string `json:"descricao"`
	LogoURL      string `json:"logo_url"`
	ChavePix     string `json:"chave_pix"`
	TipoChavePix string `json:"tipo_chave_pix"`
	Instagram    string `json:"instagram"`
	Site         string `json:"site"`
}

func (h *OrganizadorHandler) AtualizarPerfil(c *gin.Context) {
	var req atualizarPerfilRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "dados inválidos"})
		return
	}

	usuarioID := c.GetInt64(middleware.ChaveContextoUsuarioID)
	organizador, err := h.service.AtualizarPerfil(usuarioID, service.AtualizarPerfilDados{
		NomePublico:  req.NomePublico,
		Descricao:    req.Descricao,
		LogoURL:      req.LogoURL,
		ChavePix:     req.ChavePix,
		TipoChavePix: req.TipoChavePix,
		Instagram:    req.Instagram,
		Site:         req.Site,
	})
	if err != nil {
		if errors.Is(err, service.ErrOrganizadorNaoExiste) {
			c.JSON(http.StatusNotFound, gin.H{"erro": "crie seu perfil de organizador primeiro"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao atualizar perfil"})
		return
	}

	c.JSON(http.StatusOK, paraOrganizadorResposta(organizador))
}
