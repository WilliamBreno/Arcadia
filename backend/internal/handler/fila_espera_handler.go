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

// Lista de espera de ingresso esgotado (item 7.3).
type FilaEsperaHandler struct {
	fila    *service.FilaEsperaService
	eventos *repository.EventoRepository
	org     *OrganizadorHandler
}

func NovoFilaEsperaHandler(f *service.FilaEsperaService, e *repository.EventoRepository, org *OrganizadorHandler) *FilaEsperaHandler {
	return &FilaEsperaHandler{fila: f, eventos: e, org: org}
}

type entrarFilaRequest struct {
	TipoIngressoID int64 `json:"tipo_ingresso_id" binding:"required"`
	Quantidade     int   `json:"quantidade" binding:"required"`
}

// Entrar é POST /eventos/:slug/fila.
func (h *FilaEsperaHandler) Entrar(c *gin.Context) {
	e, err := h.eventos.BuscarPublicoPorSlug(c.Param("slug"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"erro": "evento não encontrado"})
		return
	}
	var req entrarFilaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "escolha o ingresso e a quantidade"})
		return
	}
	uid := c.GetInt64(middleware.ChaveContextoUsuarioID)
	if _, err := h.fila.Entrar(uid, e, req.TipoIngressoID, req.Quantidade); err != nil {
		h.erro(c, err)
		return
	}
	h.responderMinhas(c, uid, e.ID, http.StatusCreated)
}

// Minhas é GET /eventos/:slug/fila: onde a pessoa está em cada fila do evento.
func (h *FilaEsperaHandler) Minhas(c *gin.Context) {
	e, err := h.eventos.BuscarPublicoPorSlug(c.Param("slug"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"erro": "evento não encontrado"})
		return
	}
	h.responderMinhas(c, c.GetInt64(middleware.ChaveContextoUsuarioID), e.ID, http.StatusOK)
}

// Sair é DELETE /eventos/:slug/fila/:tipo.
func (h *FilaEsperaHandler) Sair(c *gin.Context) {
	e, err := h.eventos.BuscarPublicoPorSlug(c.Param("slug"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"erro": "evento não encontrado"})
		return
	}
	tipoID, err := strconv.ParseInt(c.Param("tipo"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "ingresso inválido"})
		return
	}
	uid := c.GetInt64(middleware.ChaveContextoUsuarioID)
	if err := h.fila.Sair(uid, tipoID); err != nil {
		h.erro(c, err)
		return
	}
	h.responderMinhas(c, uid, e.ID, http.StatusOK)
}

func (h *FilaEsperaHandler) responderMinhas(c *gin.Context, uid, eventoID int64, status int) {
	lista, err := h.fila.Minhas(uid, eventoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao ler a lista de espera"})
		return
	}
	c.JSON(status, lista)
}

// Resumo é GET /org/eventos/:id/fila: quantas pessoas esperam por tipo.
func (h *FilaEsperaHandler) Resumo(c *gin.Context) {
	org, ok := h.org.ObterOrganizadorAtual(c)
	if !ok {
		return
	}
	eventoID, ok := idDaURL(c)
	if !ok {
		return
	}
	e, err := h.eventos.BuscarPorID(eventoID)
	if err != nil || e.OrganizadorID != org.ID {
		c.JSON(http.StatusNotFound, gin.H{"erro": "evento não encontrado"})
		return
	}
	r, err := h.fila.ResumoDoEvento(e.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao ler a lista de espera"})
		return
	}
	c.JSON(http.StatusOK, r)
}

func (h *FilaEsperaHandler) erro(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrAindaHaVaga), errors.Is(err, service.ErrJaNaFila):
		c.JSON(http.StatusConflict, gin.H{"erro": err.Error()})
	case errors.Is(err, service.ErrFilaIndisponivel), errors.Is(err, service.ErrQuantidadeFila):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"erro": err.Error()})
	case errors.Is(err, service.ErrNaoEstaNaFila):
		c.JSON(http.StatusNotFound, gin.H{"erro": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro na lista de espera"})
	}
}
