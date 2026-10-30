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

// Modo palco/telão e camarim (item 8.3). Organizador e equipe de check-in
// operam; o telão abre por link secreto, sem login.
type PalcoHandler struct {
	svc     *service.PalcoService
	checkin *service.CheckinService
	eventos *repository.EventoRepository
	org     *OrganizadorHandler
}

func NovoPalcoHandler(s *service.PalcoService, c *service.CheckinService, e *repository.EventoRepository, org *OrganizadorHandler) *PalcoHandler {
	return &PalcoHandler{svc: s, checkin: c, eventos: e, org: org}
}

// acesso: organizador do evento ou equipe de check-in confirmada.
func (h *PalcoHandler) acesso(c *gin.Context) (int64, bool) {
	id, ok := idDaURL(c)
	if !ok {
		return 0, false
	}
	if !h.checkin.TemAcesso(c.GetInt64(middleware.ChaveContextoUsuarioID), id) {
		c.JSON(http.StatusForbidden, gin.H{"erro": "você não faz parte da equipe deste evento"})
		return 0, false
	}
	return id, true
}

func (h *PalcoHandler) responder(c *gin.Context, eventoID int64) {
	e, err := h.svc.Estado(eventoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao ler o palco"})
		return
	}
	c.JSON(http.StatusOK, e)
}

func (h *PalcoHandler) erro(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrQRParticipante), errors.Is(err, service.ErrQROutroEvento), errors.Is(err, service.ErrStatusPalco),
		errors.Is(err, service.ErrFichaNaoNaFila):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"erro": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro no palco"})
	}
}

// Estado é GET /palco/eventos/:id.
func (h *PalcoHandler) Estado(c *gin.Context) {
	if id, ok := h.acesso(c); ok {
		h.responder(c, id)
	}
}

// Status é POST /palco/eventos/:id/fichas/:ficha/status {"status"}.
func (h *PalcoHandler) Status(c *gin.Context) {
	id, ok := h.acesso(c)
	if !ok {
		return
	}
	ficha, err := strconv.ParseInt(c.Param("ficha"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "participante inválido"})
		return
	}
	var req struct {
		Status string `json:"status"`
	}
	_ = c.ShouldBindJSON(&req)
	if err := h.svc.MarcarStatus(id, ficha, req.Status); err != nil {
		h.erro(c, err)
		return
	}
	h.responder(c, id)
}

// QR é POST /palco/eventos/:id/camarim/qr {"payload"}.
func (h *PalcoHandler) QR(c *gin.Context) {
	id, ok := h.acesso(c)
	if !ok {
		return
	}
	var req struct {
		Payload string `json:"payload"`
	}
	_ = c.ShouldBindJSON(&req)
	item, novo, err := h.svc.CheckinQR(id, req.Payload)
	if err != nil {
		h.erro(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"participante": item, "chegou_agora": novo})
}

// Chamar é POST /palco/eventos/:id/chamar {"ficha_id"}.
func (h *PalcoHandler) Chamar(c *gin.Context) {
	id, ok := h.acesso(c)
	if !ok {
		return
	}
	var req struct {
		FichaID int64 `json:"ficha_id"`
	}
	_ = c.ShouldBindJSON(&req)
	if err := h.svc.Chamar(id, req.FichaID); err != nil {
		h.erro(c, err)
		return
	}
	h.responder(c, id)
}

// Proximo é POST /palco/eventos/:id/proximo.
func (h *PalcoHandler) Proximo(c *gin.Context) {
	id, ok := h.acesso(c)
	if !ok {
		return
	}
	if _, err := h.svc.Proximo(id); err != nil {
		h.erro(c, err)
		return
	}
	h.responder(c, id)
}

// Encerrar é POST /palco/eventos/:id/encerrar.
func (h *PalcoHandler) Encerrar(c *gin.Context) {
	id, ok := h.acesso(c)
	if !ok {
		return
	}
	if err := h.svc.EncerrarAtual(id); err != nil {
		h.erro(c, err)
		return
	}
	h.responder(c, id)
}

// Modo é POST /palco/eventos/:id/modo {"modo": intervalo|apresentacao|resultado}.
func (h *PalcoHandler) Modo(c *gin.Context) {
	id, ok := h.acesso(c)
	if !ok {
		return
	}
	var req struct {
		Modo string `json:"modo"`
	}
	_ = c.ShouldBindJSON(&req)
	if err := h.svc.DefinirModo(id, req.Modo); err != nil {
		h.erro(c, err)
		return
	}
	h.responder(c, id)
}

// LinkTelao é POST /org/eventos/:id/telao/link (só o organizador): gera um
// link secreto novo; o anterior para de funcionar.
func (h *PalcoHandler) LinkTelao(c *gin.Context) {
	org, ok := h.org.ObterOrganizadorAtual(c)
	if !ok {
		return
	}
	id, ok := idDaURL(c)
	if !ok {
		return
	}
	e, err := h.eventos.BuscarPorID(id)
	if err != nil || e.OrganizadorID != org.ID {
		c.JSON(http.StatusNotFound, gin.H{"erro": "evento não encontrado"})
		return
	}
	token, err := h.svc.NovoLinkTelao(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao gerar o link"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token})
}

// Telao é GET /telao/:token (sem login).
func (h *PalcoHandler) Telao(c *gin.Context) {
	e, err := h.svc.Telao(c.Param("token"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"erro": err.Error()})
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Header("X-Robots-Tag", "noindex, nofollow")
	c.JSON(http.StatusOK, e)
}

// MeuQR é GET /eventos/:slug/meu-qr-participante.
func (h *PalcoHandler) MeuQR(c *gin.Context) {
	e, err := h.eventos.BuscarPublicoPorSlug(c.Param("slug"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"erro": "evento não encontrado"})
		return
	}
	qrs, err := h.svc.MeusQRs(e.ID, c.GetInt64(middleware.ChaveContextoUsuarioID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao ler"})
		return
	}
	c.JSON(http.StatusOK, qrs)
}
