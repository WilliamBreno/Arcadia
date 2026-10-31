package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
	"github.com/WilliamBreno/Arcadia/backend/internal/service"
)

// Item 11.6: limite de jurados e participantes por evento.
type VagasHandler struct {
	organizadorHandler *OrganizadorHandler
	eventos            *repository.EventoRepository
	service            *service.VagasService
}

func NovoVagasHandler(o *OrganizadorHandler, eventos *repository.EventoRepository, s *service.VagasService) *VagasHandler {
	return &VagasHandler{organizadorHandler: o, eventos: eventos, service: s}
}

type vagasRequest struct {
	Jurados       *int `json:"jurados"`
	Participantes *int `json:"participantes"`
	Cosplay       *int `json:"cosplay"`
	Danca         *int `json:"danca"`
	Canto         *int `json:"canto"`
	Atuacao       *int `json:"atuacao"`
	ListaEspera   bool `json:"lista_espera"`
}

// responderErroVagas responde 409 quando o erro é de vaga preenchida.
func responderErroVagas(c *gin.Context, err error) bool {
	if service.EhErroVagas(err) {
		c.JSON(http.StatusConflict, gin.H{"erro": err.Error(), "vagas_esgotadas": true})
		return true
	}
	return false
}

// Obter é GET /org/eventos/:id/vagas.
func (h *VagasHandler) Obter(c *gin.Context) {
	organizador, ok := h.organizadorHandler.ObterOrganizadorAtual(c)
	if !ok {
		return
	}
	eventoID, ok := idDaURL(c)
	if !ok {
		return
	}
	r, err := h.service.ResumoDoOrganizador(organizador.ID, eventoID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"erro": "evento não encontrado"})
		return
	}
	c.JSON(http.StatusOK, r)
}

// Definir é PUT /org/eventos/:id/vagas.
func (h *VagasHandler) Definir(c *gin.Context) {
	organizador, ok := h.organizadorHandler.ObterOrganizadorAtual(c)
	if !ok {
		return
	}
	eventoID, ok := idDaURL(c)
	if !ok {
		return
	}
	var req vagasRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "dados inválidos"})
		return
	}
	_, err := h.service.Definir(organizador.ID, eventoID, service.VagasEvento{
		Jurados: req.Jurados, Participantes: req.Participantes,
		Cosplay: req.Cosplay, Danca: req.Danca, Canto: req.Canto, Atuacao: req.Atuacao,
		ListaEspera: req.ListaEspera,
	})
	switch {
	case errors.Is(err, service.ErrVagasInvalidas):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"erro": err.Error()})
		return
	case err != nil:
		c.JSON(http.StatusNotFound, gin.H{"erro": "evento não encontrado"})
		return
	}
	r, err := h.service.Resumo(eventoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro interno"})
		return
	}
	c.JSON(http.StatusOK, r)
}

// Publico é GET /eventos/:slug/vagas: só participantes (a quantidade de
// jurados não é pública).
func (h *VagasHandler) Publico(c *gin.Context) {
	e, err := h.eventos.BuscarPublicoPorSlug(c.Param("slug"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"erro": "evento não encontrado"})
		return
	}
	r, err := h.service.Resumo(e.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro interno"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"participantes": r.Participantes, "por_tipo": r.PorTipo, "lista_espera": r.ListaEspera})
}
