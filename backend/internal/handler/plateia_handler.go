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

type PlateiaHandler struct {
	organizadorHandler *OrganizadorHandler
	eventos            *repository.EventoRepository
	service            *service.PlateiaService
}

func NovoPlateiaHandler(o *OrganizadorHandler, e *repository.EventoRepository, s *service.PlateiaService) *PlateiaHandler {
	return &PlateiaHandler{organizadorHandler: o, eventos: e, service: s}
}

func (h *PlateiaHandler) responderErro(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrEventoNaoPertenceAoOrganizador):
		c.JSON(http.StatusForbidden, gin.H{"erro": "evento não pertence a este organizador"})
	case errors.Is(err, service.ErrSolicitacaoNaoEncontrada):
		c.JSON(http.StatusNotFound, gin.H{"erro": err.Error()})
	case errors.Is(err, service.ErrSolicitacaoJaExiste):
		c.JSON(http.StatusConflict, gin.H{"erro": err.Error()})
	default:
		c.JSON(http.StatusUnprocessableEntity, gin.H{"erro": err.Error()})
	}
}

// Solicitar é POST /eventos/:slug/solicitacao.
func (h *PlateiaHandler) Solicitar(c *gin.Context) {
	evento, err := h.eventos.BuscarPorSlug(c.Param("slug"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"erro": "evento não encontrado"})
		return
	}
	sol, err := h.service.Solicitar(c.GetInt64(middleware.ChaveContextoUsuarioID), evento)
	if err != nil {
		h.responderErro(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"status": sol.Status})
}

// Minha é GET /eventos/:slug/solicitacao — 404 se ainda não solicitou.
func (h *PlateiaHandler) Minha(c *gin.Context) {
	evento, err := h.eventos.BuscarPorSlug(c.Param("slug"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"erro": "evento não encontrado"})
		return
	}
	sol, err := h.service.Minha(c.GetInt64(middleware.ChaveContextoUsuarioID), evento.ID)
	if err != nil {
		h.responderErro(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": sol.Status})
}

// Listar é GET /org/eventos/:id/solicitacoes.
func (h *PlateiaHandler) Listar(c *gin.Context) {
	org, ok := h.organizadorHandler.ObterOrganizadorAtual(c)
	if !ok {
		return
	}
	eventoID, ok := idDaURL(c)
	if !ok {
		return
	}
	lista, err := h.service.Listar(org.ID, eventoID)
	if err != nil {
		h.responderErro(c, err)
		return
	}
	resp := make([]gin.H, 0, len(lista))
	for _, s := range lista {
		resp = append(resp, gin.H{"id": s.ID, "usuario_nome": s.UsuarioNome, "usuario_email": s.UsuarioEmail, "status": s.Status, "criado_em": s.CriadoEm})
	}
	c.JSON(http.StatusOK, resp)
}

type decidirRequest struct {
	Decisao string `json:"decisao" binding:"required"`
}

// Decidir é POST /org/eventos/:id/solicitacoes/:solicitacaoId.
func (h *PlateiaHandler) Decidir(c *gin.Context) {
	org, ok := h.organizadorHandler.ObterOrganizadorAtual(c)
	if !ok {
		return
	}
	eventoID, ok := idDaURL(c)
	if !ok {
		return
	}
	solID, err := strconv.ParseInt(c.Param("solicitacaoId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "id inválido"})
		return
	}
	var req decidirRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "dados inválidos"})
		return
	}
	sol, err := h.service.Decidir(org.ID, eventoID, solID, req.Decisao)
	if err != nil {
		h.responderErro(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": sol.ID, "status": sol.Status})
}
