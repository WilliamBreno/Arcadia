package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/WilliamBreno/Arcadia/backend/internal/middleware"
	"github.com/WilliamBreno/Arcadia/backend/internal/service"
)

type CheckinHandler struct {
	service *service.CheckinService
}

func NovoCheckinHandler(s *service.CheckinService) *CheckinHandler {
	return &CheckinHandler{service: s}
}

type validarCheckinRequest struct {
	EventoID int64  `json:"evento_id" binding:"required"`
	Codigo   string `json:"codigo" binding:"required"`
	QRToken  string `json:"qr_token" binding:"required"`
}

type validarCheckinResposta struct {
	Resultado        string `json:"resultado"`
	TitularNome      string `json:"titular_nome,omitempty"`
	TipoIngressoNome string `json:"tipo_ingresso_nome,omitempty"`
	Codigo           string `json:"codigo,omitempty"`
	UtilizadoEm      string `json:"utilizado_em,omitempty"`
}

// Validar é POST /checkin/validar — o leitor de QR chama isso a cada
// leitura (seção 7.10).
func (h *CheckinHandler) Validar(c *gin.Context) {
	var req validarCheckinRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "dados inválidos"})
		return
	}

	usuarioID := c.GetInt64(middleware.ChaveContextoUsuarioID)
	resultado, err := h.service.Validar(usuarioID, req.EventoID, req.Codigo, req.QRToken)
	if err != nil {
		if errors.Is(err, service.ErrSemAcessoCheckin) {
			c.JSON(http.StatusForbidden, gin.H{"erro": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao validar check-in"})
		return
	}

	resp := validarCheckinResposta{Resultado: string(resultado.Resultado), TipoIngressoNome: resultado.TipoIngressoNome}
	if resultado.Item != nil {
		resp.TitularNome = resultado.Item.TitularNome
		resp.Codigo = resultado.Item.Codigo
		if resultado.Item.UtilizadoEm != nil {
			resp.UtilizadoEm = resultado.Item.UtilizadoEm.Format("15:04:05")
		}
	}
	c.JSON(http.StatusOK, resp)
}

type buscaCheckinItem struct {
	ID               int64  `json:"id"`
	TitularNome      string `json:"titular_nome"`
	TitularEmail     string `json:"titular_email"`
	Codigo           string `json:"codigo"`
	Status           string `json:"status"`
	TipoIngressoNome string `json:"tipo_ingresso_nome"`
}

// Buscar é GET /checkin/eventos/:id/busca — busca manual por nome,
// e-mail ou código (seção 7.10).
func (h *CheckinHandler) Buscar(c *gin.Context) {
	eventoID, ok := idDaURL(c)
	if !ok {
		return
	}
	usuarioID := c.GetInt64(middleware.ChaveContextoUsuarioID)
	texto := c.Query("q")

	itens, err := h.service.Buscar(usuarioID, eventoID, texto)
	if err != nil {
		if errors.Is(err, service.ErrSemAcessoCheckin) {
			c.JSON(http.StatusForbidden, gin.H{"erro": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao buscar"})
		return
	}

	resposta := make([]buscaCheckinItem, 0, len(itens))
	for _, i := range itens {
		resposta = append(resposta, buscaCheckinItem{
			ID: i.ID, TitularNome: i.TitularNome, TitularEmail: i.TitularEmail,
			Codigo: i.Codigo, Status: string(i.Status), TipoIngressoNome: i.TipoIngressoNome,
		})
	}
	c.JSON(http.StatusOK, resposta)
}

// Resumo é GET /checkin/eventos/:id/resumo — contador em tempo real.
func (h *CheckinHandler) Resumo(c *gin.Context) {
	eventoID, ok := idDaURL(c)
	if !ok {
		return
	}
	usuarioID := c.GetInt64(middleware.ChaveContextoUsuarioID)

	resumo, err := h.service.Resumo(usuarioID, eventoID)
	if err != nil {
		if errors.Is(err, service.ErrSemAcessoCheckin) {
			c.JSON(http.StatusForbidden, gin.H{"erro": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao carregar resumo"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"total_pagos": resumo.TotalPagos, "total_utilizados": resumo.TotalUtilizados})
}

// --- Organizador: gestão de staff (seção 3) ---

type StaffHandler struct {
	organizadorHandler *OrganizadorHandler
	service            *service.StaffService
}

func NovoStaffHandler(organizadorHandler *OrganizadorHandler, s *service.StaffService) *StaffHandler {
	return &StaffHandler{organizadorHandler: organizadorHandler, service: s}
}

type staffResposta struct {
	UsuarioID int64  `json:"usuario_id"`
	Nome      string `json:"nome"`
	Email     string `json:"email"`
}

func paraStaffResposta(s *service.StaffComUsuario) staffResposta {
	return staffResposta{UsuarioID: s.Papel.UsuarioID, Nome: s.Nome, Email: s.Email}
}

type adicionarStaffRequest struct {
	Email string `json:"email" binding:"required,email"`
}

func (h *StaffHandler) Adicionar(c *gin.Context) {
	organizador, ok := h.organizadorHandler.ObterOrganizadorAtual(c)
	if !ok {
		return
	}
	eventoID, ok := idDaURL(c)
	if !ok {
		return
	}

	var req adicionarStaffRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "informe um e-mail válido"})
		return
	}

	staff, err := h.service.Adicionar(organizador.ID, eventoID, req.Email)
	if err != nil {
		h.responderErro(c, err)
		return
	}
	c.JSON(http.StatusCreated, paraStaffResposta(staff))
}

func (h *StaffHandler) Listar(c *gin.Context) {
	organizador, ok := h.organizadorHandler.ObterOrganizadorAtual(c)
	if !ok {
		return
	}
	eventoID, ok := idDaURL(c)
	if !ok {
		return
	}

	lista, err := h.service.Listar(organizador.ID, eventoID)
	if err != nil {
		h.responderErro(c, err)
		return
	}
	resposta := make([]staffResposta, 0, len(lista))
	for i := range lista {
		resposta = append(resposta, paraStaffResposta(&lista[i]))
	}
	c.JSON(http.StatusOK, resposta)
}

func (h *StaffHandler) Remover(c *gin.Context) {
	organizador, ok := h.organizadorHandler.ObterOrganizadorAtual(c)
	if !ok {
		return
	}
	eventoID, ok := idDaURL(c)
	if !ok {
		return
	}
	usuarioID, err := strconv.ParseInt(c.Param("usuarioId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "id inválido"})
		return
	}

	if err := h.service.Remover(organizador.ID, eventoID, usuarioID); err != nil {
		h.responderErro(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *StaffHandler) responderErro(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrEventoNaoPertenceAoOrganizador):
		c.JSON(http.StatusForbidden, gin.H{"erro": "evento não pertence a este organizador"})
	case errors.Is(err, service.ErrUsuarioNaoEncontrado), errors.Is(err, service.ErrStaffNaoEncontrado):
		c.JSON(http.StatusNotFound, gin.H{"erro": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro inesperado"})
	}
}
