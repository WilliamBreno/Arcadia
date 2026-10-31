package handler

import (
	"github.com/WilliamBreno/Arcadia/backend/internal/service"
	"github.com/gin-gonic/gin"
	"net/http"
)

func (h *GruposHandler) Gestao(c *gin.Context) {
	id, ok := paramID(c, "id")
	if !ok {
		return
	}
	v, err := h.service.Gestao(usuarioDe(c), id)
	if err != nil {
		h.erro(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, v)
}
func (h *GruposHandler) ConvidarGestao(c *gin.Context) {
	id, ok := paramID(c, "id")
	if !ok {
		return
	}
	var d struct {
		Email          string `json:"email"`
		Papel          string `json:"papel"`
		AntigoCogestor bool   `json:"antigo_cogestor"`
	}
	if c.ShouldBindJSON(&d) != nil {
		c.JSON(400, gin.H{"erro": "dados inválidos"})
		return
	}
	if err := h.service.ConvidarGestao(usuarioDe(c), id, d.Email, d.Papel, d.AntigoCogestor); err != nil {
		h.erro(c, err)
		return
	}
	c.JSON(201, gin.H{"ok": true})
}
func (h *GruposHandler) VerGestaoConvite(c *gin.Context) {
	v, err := h.service.VerGestaoConvite(usuarioDe(c), c.Param("token"))
	if err != nil {
		h.erro(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(200, v)
}
func (h *GruposHandler) DecidirGestaoConvite(c *gin.Context) {
	var d struct {
		Aceitar *bool `json:"aceitar"`
	}
	if c.ShouldBindJSON(&d) != nil || d.Aceitar == nil {
		c.JSON(400, gin.H{"erro": "informe se aceita o convite"})
		return
	}
	id, err := h.service.DecidirGestaoConvite(usuarioDe(c), c.Param("token"), *d.Aceitar)
	if err != nil {
		h.erro(c, err)
		return
	}
	c.JSON(200, gin.H{"grupo_id": id})
}
func (h *GruposHandler) RemoverGestao(c *gin.Context) {
	id, ok := paramID(c, "id")
	if !ok {
		return
	}
	alvo, ok := paramID(c, "alvo")
	if !ok {
		return
	}
	if err := h.service.RemoverGestao(usuarioDe(c), id, alvo, c.Query("convite") == "1"); err != nil {
		h.erro(c, err)
		return
	}
	c.JSON(200, gin.H{"ok": true})
}
func (h *GruposHandler) Agenda(c *gin.Context) {
	id, ok := paramID(c, "id")
	if !ok {
		return
	}
	v, err := h.service.Agenda(usuarioDe(c), id)
	if err != nil {
		h.erro(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(200, v)
}
func (h *GruposHandler) SalvarAgenda(c *gin.Context) {
	id, ok := paramID(c, "id")
	if !ok {
		return
	}
	var aid int64
	if c.Param("agenda") != "" {
		aid, ok = paramID(c, "agenda")
		if !ok {
			return
		}
	}
	var d service.AgendaArtista
	if c.ShouldBindJSON(&d) != nil {
		c.JSON(400, gin.H{"erro": "dados inválidos"})
		return
	}
	if err := h.service.SalvarAgenda(usuarioDe(c), id, aid, d); err != nil {
		h.erro(c, err)
		return
	}
	c.JSON(200, gin.H{"ok": true})
}
func (h *GruposHandler) RemoverAgenda(c *gin.Context) {
	id, ok := paramID(c, "id")
	if !ok {
		return
	}
	aid, ok := paramID(c, "agenda")
	if !ok {
		return
	}
	if err := h.service.RemoverAgenda(usuarioDe(c), id, aid); err != nil {
		h.erro(c, err)
		return
	}
	c.JSON(200, gin.H{"ok": true})
}
func (h *GruposHandler) MidiaKit(c *gin.Context) {
	id, ok := paramID(c, "id")
	if !ok {
		return
	}
	b, err := h.service.MidiaKit(usuarioDe(c), id)
	if err != nil {
		h.erro(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Header("Content-Disposition", `attachment; filename="midia-kit.pdf"`)
	c.Data(200, "application/pdf", b)
}
