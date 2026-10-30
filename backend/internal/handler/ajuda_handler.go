package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/WilliamBreno/Arcadia/backend/internal/middleware"
	"github.com/WilliamBreno/Arcadia/backend/internal/service"
	"gorm.io/gorm"
)

// Central de ajuda, verificação de ingresso e status (item 9.9).
type AjudaHandler struct {
	svc *service.AjudaService
	db  *gorm.DB
}

func NovoAjudaHandler(s *service.AjudaService, db *gorm.DB) *AjudaHandler {
	return &AjudaHandler{svc: s, db: db}
}

// Verificar é GET /verificar-ingresso?codigo=... (código, QR ou link).
func (h *AjudaHandler) Verificar(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, service.VerificarIngresso(h.db, c.Query("codigo"), time.Now()))
}

// Contato é POST /suporte/contato (com ou sem login).
func (h *AjudaHandler) Contato(c *gin.Context) {
	var req struct {
		Nome     string `json:"nome"`
		Email    string `json:"email"`
		Assunto  string `json:"assunto"`
		Pedido   string `json:"pedido"`
		Mensagem string `json:"mensagem"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "dados inválidos"})
		return
	}
	protocolo, err := h.svc.EnviarContato(c.GetInt64(middleware.ChaveContextoUsuarioID), req.Nome, req.Email, req.Assunto, req.Pedido, req.Mensagem)
	if err != nil {
		if errors.Is(err, service.ErrDadosContato) {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"erro": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "não foi possível enviar agora; tente de novo em instantes"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"protocolo": protocolo})
}

// Status é GET /status.
func (h *AjudaHandler) Status(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, h.svc.Status(time.Now()))
}

// Avisos é GET /status/avisos (faixa no site).
func (h *AjudaHandler) Avisos(c *gin.Context) {
	c.Header("Cache-Control", "public, max-age=60")
	c.JSON(http.StatusOK, h.svc.AvisosAbertos())
}

// ---------- admin ----------

// CriarAviso é POST /admin/avisos-status.
func (h *AjudaHandler) CriarAviso(c *gin.Context) {
	var req struct {
		Titulo     string `json:"titulo"`
		Mensagem   string `json:"mensagem"`
		Nivel      string `json:"nivel"`
		Componente string `json:"componente"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "dados inválidos"})
		return
	}
	a, err := h.svc.CriarAviso(c.GetInt64(middleware.ChaveContextoUsuarioID), req.Titulo, req.Mensagem, req.Nivel, req.Componente)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"erro": "informe título (até 120 letras), nível e componente válidos"})
		return
	}
	c.JSON(http.StatusCreated, a)
}

// EncerrarAviso é POST /admin/avisos-status/:id/encerrar.
func (h *AjudaHandler) EncerrarAviso(c *gin.Context) {
	id, ok := idDaURL(c)
	if !ok {
		return
	}
	if err := h.svc.EncerrarAviso(id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"erro": "aviso não encontrado ou já encerrado"})
		return
	}
	c.Status(http.StatusNoContent)
}

// Contatos é GET /admin/suporte[?status=aberto].
func (h *AjudaHandler) Contatos(c *gin.Context) {
	lista, err := h.svc.ListarContatos(c.Query("status"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao listar"})
		return
	}
	resp := make([]gin.H, 0, len(lista))
	for _, x := range lista {
		resp = append(resp, gin.H{"id": x.ID, "nome": x.Nome, "email": x.Email, "assunto": x.Assunto, "assunto_rotulo": service.AssuntosSuporte[x.Assunto],
			"pedido": x.Pedido, "mensagem": x.Mensagem, "status": x.Status, "criado_em": x.CriadoEm, "com_conta": x.UsuarioID != nil})
	}
	c.JSON(http.StatusOK, resp)
}

// Responder é POST /admin/suporte/:id/respondido.
func (h *AjudaHandler) Respondido(c *gin.Context) {
	id, ok := idDaURL(c)
	if !ok {
		return
	}
	if err := h.svc.MarcarRespondido(id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"erro": "mensagem não encontrada"})
		return
	}
	c.Status(http.StatusNoContent)
}
