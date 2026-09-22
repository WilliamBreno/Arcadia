package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/WilliamBreno/Arcadia/backend/internal/service"
)

type LocalHandler struct {
	organizadorHandler *OrganizadorHandler
	service            *service.LocalService
}

func NovoLocalHandler(organizadorHandler *OrganizadorHandler, s *service.LocalService) *LocalHandler {
	return &LocalHandler{organizadorHandler: organizadorHandler, service: s}
}

type localResposta struct {
	ID          int64    `json:"id"`
	Nome        string   `json:"nome"`
	Logradouro  string   `json:"logradouro"`
	Numero      string   `json:"numero"`
	Bairro      string   `json:"bairro"`
	Cidade      string   `json:"cidade"`
	UF          string   `json:"uf"`
	CEP         string   `json:"cep"`
	Latitude    *float64 `json:"latitude"`
	Longitude   *float64 `json:"longitude"`
	Capacidade  *int     `json:"capacidade"`
	Observacoes string   `json:"observacoes"`
}

func paraLocalResposta(l *domain.Local) localResposta {
	return localResposta{
		ID:          l.ID,
		Nome:        l.Nome,
		Logradouro:  l.Logradouro,
		Numero:      l.Numero,
		Bairro:      l.Bairro,
		Cidade:      l.Cidade,
		UF:          l.UF,
		CEP:         l.CEP,
		Latitude:    l.Latitude,
		Longitude:   l.Longitude,
		Capacidade:  l.Capacidade,
		Observacoes: l.Observacoes,
	}
}

type localRequest struct {
	Nome        string   `json:"nome" binding:"required,min=2"`
	Logradouro  string   `json:"logradouro"`
	Numero      string   `json:"numero"`
	Bairro      string   `json:"bairro"`
	Cidade      string   `json:"cidade" binding:"required"`
	UF          string   `json:"uf" binding:"required,len=2"`
	CEP         string   `json:"cep"`
	Latitude    *float64 `json:"latitude"`
	Longitude   *float64 `json:"longitude"`
	Capacidade  *int     `json:"capacidade"`
	Observacoes string   `json:"observacoes"`
}

func (req localRequest) paraDados() service.LocalDados {
	return service.LocalDados{
		Nome:        req.Nome,
		Logradouro:  req.Logradouro,
		Numero:      req.Numero,
		Bairro:      req.Bairro,
		Cidade:      req.Cidade,
		UF:          req.UF,
		CEP:         req.CEP,
		Latitude:    req.Latitude,
		Longitude:   req.Longitude,
		Capacidade:  req.Capacidade,
		Observacoes: req.Observacoes,
	}
}

func (h *LocalHandler) Listar(c *gin.Context) {
	organizador, ok := h.organizadorHandler.ObterOrganizadorAtual(c)
	if !ok {
		return
	}

	locais, err := h.service.Listar(organizador.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao listar locais"})
		return
	}

	resposta := make([]localResposta, 0, len(locais))
	for i := range locais {
		resposta = append(resposta, paraLocalResposta(&locais[i]))
	}
	c.JSON(http.StatusOK, resposta)
}

func (h *LocalHandler) Criar(c *gin.Context) {
	organizador, ok := h.organizadorHandler.ObterOrganizadorAtual(c)
	if !ok {
		return
	}

	var req localRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "dados inválidos"})
		return
	}

	local, err := h.service.Criar(organizador.ID, req.paraDados())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao criar local"})
		return
	}

	c.JSON(http.StatusCreated, paraLocalResposta(local))
}

func idDaURL(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "id inválido"})
		return 0, false
	}
	return id, true
}

func (h *LocalHandler) Atualizar(c *gin.Context) {
	organizador, ok := h.organizadorHandler.ObterOrganizadorAtual(c)
	if !ok {
		return
	}
	localID, ok := idDaURL(c)
	if !ok {
		return
	}

	var req localRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "dados inválidos"})
		return
	}

	local, err := h.service.Atualizar(organizador.ID, localID, req.paraDados())
	if err != nil {
		h.responderErroDeAcesso(c, err)
		return
	}

	c.JSON(http.StatusOK, paraLocalResposta(local))
}

func (h *LocalHandler) Excluir(c *gin.Context) {
	organizador, ok := h.organizadorHandler.ObterOrganizadorAtual(c)
	if !ok {
		return
	}
	localID, ok := idDaURL(c)
	if !ok {
		return
	}

	if err := h.service.Excluir(organizador.ID, localID); err != nil {
		h.responderErroDeAcesso(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *LocalHandler) responderErroDeAcesso(c *gin.Context, err error) {
	if errors.Is(err, service.ErrLocalNaoPertenceAoOrganizador) {
		c.JSON(http.StatusForbidden, gin.H{"erro": "local não pertence a este organizador"})
		return
	}
	c.JSON(http.StatusNotFound, gin.H{"erro": "local não encontrado"})
}
