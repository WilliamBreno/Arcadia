package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/WilliamBreno/Arcadia/backend/internal/middleware"
	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
	"github.com/WilliamBreno/Arcadia/backend/internal/service"
)

// Avaliações do público (item 6.5).
type AvaliacaoPublicoHandler struct {
	svc     *service.AvaliacaoPublicoService
	eventos *repository.EventoRepository
}

func NovoAvaliacaoPublicoHandler(s *service.AvaliacaoPublicoService, e *repository.EventoRepository) *AvaliacaoPublicoHandler {
	return &AvaliacaoPublicoHandler{svc: s, eventos: e}
}

// DoEvento é GET /api/v1/eventos/:slug/avaliacoes.
func (h *AvaliacaoPublicoHandler) DoEvento(c *gin.Context) {
	e, err := h.eventos.BuscarPublicoPorSlug(c.Param("slug"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"erro": "evento não encontrado"})
		return
	}
	resumo, err := h.svc.ResumoDoEvento(e.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao ler avaliações"})
		return
	}
	itens, err := h.svc.ListarDoEvento(e.ID, 30)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao ler avaliações"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"resumo": resumo, "avaliacoes": itens})
}

// Minha é GET /api/v1/eventos/:slug/minha-avaliacao: se pode avaliar e o que já avaliou.
func (h *AvaliacaoPublicoHandler) Minha(c *gin.Context) {
	e, err := h.eventos.BuscarPublicoPorSlug(c.Param("slug"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"erro": "evento não encontrado"})
		return
	}
	uid := c.GetInt64(middleware.ChaveContextoUsuarioID)
	pode, motivo, err := h.svc.PodeAvaliar(e, uid, time.Now())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao verificar"})
		return
	}
	minha, err := h.svc.Minha(e.ID, uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao ler"})
		return
	}
	resp := gin.H{"pode_avaliar": pode, "motivo": motivo, "avaliacao": nil}
	if minha != nil {
		resp["avaliacao"] = gin.H{"nota": minha.Nota, "comentario": minha.Comentario, "ocultada": minha.Ocultada}
	}
	c.JSON(http.StatusOK, resp)
}

type avaliarPublicoRequest struct {
	Nota       int    `json:"nota" binding:"required"`
	Comentario string `json:"comentario"`
}

// Salvar é PUT /api/v1/eventos/:slug/minha-avaliacao.
func (h *AvaliacaoPublicoHandler) Salvar(c *gin.Context) {
	e, err := h.eventos.BuscarPublicoPorSlug(c.Param("slug"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"erro": "evento não encontrado"})
		return
	}
	var req avaliarPublicoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "escolha uma nota de 1 a 5"})
		return
	}
	a, err := h.svc.Salvar(e, c.GetInt64(middleware.ChaveContextoUsuarioID), req.Nota, req.Comentario)
	if err != nil {
		if errors.Is(err, service.ErrNaoPodeAvaliar) {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"erro": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao salvar avaliação"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"nota": a.Nota, "comentario": a.Comentario})
}

// ListarAdmin é GET /api/v1/admin/avaliacoes.
func (h *AvaliacaoPublicoHandler) ListarAdmin(c *gin.Context) {
	lista, err := h.svc.ListarAdmin(200)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao listar"})
		return
	}
	if lista == nil {
		lista = []service.AvaliacaoAdmin{}
	}
	c.JSON(http.StatusOK, lista)
}

// Ocultar é POST /api/v1/admin/avaliacoes/:id/ocultar {"ocultar": bool}.
func (h *AvaliacaoPublicoHandler) Ocultar(c *gin.Context) {
	id, ok := idDaURL(c)
	if !ok {
		return
	}
	var req struct {
		Ocultar *bool `json:"ocultar" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "informe ocultar: true ou false"})
		return
	}
	achou, err := h.svc.DefinirOcultada(id, *req.Ocultar)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao salvar"})
		return
	}
	if !achou {
		c.JSON(http.StatusNotFound, gin.H{"erro": "avaliação não encontrada"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "ocultada": *req.Ocultar})
}
