package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/WilliamBreno/Arcadia/backend/internal/middleware"
	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
	"github.com/WilliamBreno/Arcadia/backend/internal/service"
)

type ConviteHandler struct {
	organizadorHandler *OrganizadorHandler
	eventos            *repository.EventoRepository
	service            *service.ConviteService
}

func NovoConviteHandler(organizadorHandler *OrganizadorHandler, eventos *repository.EventoRepository, s *service.ConviteService) *ConviteHandler {
	return &ConviteHandler{organizadorHandler: organizadorHandler, eventos: eventos, service: s}
}

type conviteResposta struct {
	ID         int64      `json:"id"`
	Tipo       string     `json:"tipo"`
	Token      string     `json:"token,omitempty"`
	MaxUsos    *int       `json:"max_usos"`
	Usos       int        `json:"usos"`
	ExpiraEm   *time.Time `json:"expira_em"`
	RevogadoEm *time.Time `json:"revogado_em"`
}

func paraConviteResposta(c *domain.Convite) conviteResposta {
	return conviteResposta{ID: c.ID, Tipo: string(c.Tipo), MaxUsos: c.MaxUsos, Usos: c.Usos, ExpiraEm: c.ExpiraEm, RevogadoEm: c.RevogadoEm}
}

type gerarConviteRequest struct {
	Tipo     string     `json:"tipo" binding:"required,oneof=jurado participante_especial"`
	MaxUsos  *int       `json:"max_usos"`
	ExpiraEm *time.Time `json:"expira_em"`
}

// --- Organizador ---

func (h *ConviteHandler) Listar(c *gin.Context) {
	organizador, ok := h.organizadorHandler.ObterOrganizadorAtual(c)
	if !ok {
		return
	}
	eventoID, ok := idDaURL(c)
	if !ok {
		return
	}

	convites, err := h.service.Listar(organizador.ID, eventoID)
	if err != nil {
		h.responderErro(c, err)
		return
	}

	resposta := make([]conviteResposta, 0, len(convites))
	for i := range convites {
		resposta = append(resposta, paraConviteResposta(&convites[i]))
	}
	c.JSON(http.StatusOK, resposta)
}

func (h *ConviteHandler) Gerar(c *gin.Context) {
	organizador, ok := h.organizadorHandler.ObterOrganizadorAtual(c)
	if !ok {
		return
	}
	eventoID, ok := idDaURL(c)
	if !ok {
		return
	}

	var req gerarConviteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "dados inválidos"})
		return
	}

	usuarioID := c.GetInt64(middleware.ChaveContextoUsuarioID)
	convite, tokenPlano, err := h.service.Gerar(organizador.ID, eventoID, domain.TipoConvite(req.Tipo), req.MaxUsos, req.ExpiraEm, usuarioID)
	if err != nil {
		h.responderErro(c, err)
		return
	}

	resposta := paraConviteResposta(convite)
	resposta.Token = tokenPlano
	c.JSON(http.StatusCreated, resposta)
}

func (h *ConviteHandler) Revogar(c *gin.Context) {
	organizador, ok := h.organizadorHandler.ObterOrganizadorAtual(c)
	if !ok {
		return
	}
	eventoID, ok := idDaURL(c)
	if !ok {
		return
	}
	conviteID, ok := parseParamInt64(c, "conviteId")
	if !ok {
		return
	}

	if err := h.service.Revogar(organizador.ID, eventoID, conviteID); err != nil {
		h.responderErro(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *ConviteHandler) responderErro(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrEventoNaoPertenceAoOrganizador), errors.Is(err, service.ErrConviteNaoPertence):
		c.JSON(http.StatusForbidden, gin.H{"erro": "recurso não pertence a este organizador"})
	default:
		c.JSON(http.StatusNotFound, gin.H{"erro": "não encontrado"})
	}
}

// --- Público / convidado ---

type consultarConviteResposta struct {
	Tipo         string `json:"tipo"`
	EventoTitulo string `json:"evento_titulo"`
	EventoSlug   string `json:"evento_slug"`
}

func (h *ConviteHandler) Consultar(c *gin.Context) {
	convite, err := h.service.ConsultarPorToken(c.Param("token"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"erro": "convite inválido, expirado ou revogado"})
		return
	}

	evento, err := h.eventos.BuscarPorID(convite.EventoID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"erro": "evento não encontrado"})
		return
	}

	c.JSON(http.StatusOK, consultarConviteResposta{Tipo: string(convite.Tipo), EventoTitulo: evento.Titulo, EventoSlug: evento.Slug})
}

type aceitarConviteRequest struct {
	Nome                      string         `json:"nome" binding:"required,min=2"`
	NomeArtistico             string         `json:"nome_artistico"`
	Instagram                 string         `json:"instagram"`
	DataNascimento            *time.Time     `json:"data_nascimento"`
	FotoURL                   string         `json:"foto_url"`
	Telefone                  string         `json:"telefone"`
	TipoApresentacao          *string        `json:"tipo_apresentacao"`
	Dados                     map[string]any `json:"dados"`
	ResponsavelNome           string         `json:"responsavel_nome"`
	ResponsavelContato        string         `json:"responsavel_contato"`
	AutorizacaoResponsavelURL string         `json:"autorizacao_responsavel_url"`
}

func (req aceitarConviteRequest) paraDados() service.FichaDados {
	var tipoApresentacao *domain.TipoApresentacao
	if req.TipoApresentacao != nil {
		t := domain.TipoApresentacao(*req.TipoApresentacao)
		tipoApresentacao = &t
	}

	var dadosJSON datatypes.JSON
	if req.Dados != nil {
		if bruto, err := json.Marshal(req.Dados); err == nil {
			dadosJSON = datatypes.JSON(bruto)
		}
	}

	return service.FichaDados{
		Nome:                      req.Nome,
		NomeArtistico:             req.NomeArtistico,
		Instagram:                 req.Instagram,
		DataNascimento:            req.DataNascimento,
		FotoURL:                   req.FotoURL,
		Telefone:                  req.Telefone,
		TipoApresentacao:          tipoApresentacao,
		Dados:                     dadosJSON,
		ResponsavelNome:           req.ResponsavelNome,
		ResponsavelContato:        req.ResponsavelContato,
		AutorizacaoResponsavelURL: req.AutorizacaoResponsavelURL,
	}
}

func (h *ConviteHandler) Aceitar(c *gin.Context) {
	var req aceitarConviteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "dados inválidos"})
		return
	}

	usuarioID := c.GetInt64(middleware.ChaveContextoUsuarioID)
	_, ficha, err := h.service.Aceitar(c.Param("token"), usuarioID, req.paraDados())
	if err != nil {
		if errors.Is(err, service.ErrMenorSemAutorizacao) {
			c.JSON(http.StatusBadRequest, gin.H{"erro": err.Error()})
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"erro": "convite inválido, expirado ou revogado"})
		return
	}

	c.JSON(http.StatusOK, paraFichaResposta(ficha))
}

func parseParamInt64(c *gin.Context, nome string) (int64, bool) {
	id, err := strconv.ParseInt(c.Param(nome), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "id inválido"})
		return 0, false
	}
	return id, true
}
