package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/WilliamBreno/Arcadia/backend/internal/middleware"
	"github.com/WilliamBreno/Arcadia/backend/internal/service"
)

type QRHandler struct {
	service *service.QRService
}

func NovoQRHandler(s *service.QRService) *QRHandler { return &QRHandler{service: s} }

func (h *QRHandler) responder(c *gin.Context, info *service.QRInfo, err error) {
	if err != nil {
		if errors.Is(err, service.ErrItemNaoPertenceAoUsuario) {
			c.JSON(http.StatusForbidden, gin.H{"erro": "ingresso não pertence a este usuário"})
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"erro": "ingresso não encontrado"})
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"payload": info.Payload, "rotativo": info.Rotativo, "renova_em_segundos": info.RenovaEm})
}

// DoComprador é GET /me/ingressos/:id/qr.
func (h *QRHandler) DoComprador(c *gin.Context) {
	id, ok := idDaURL(c)
	if !ok {
		return
	}
	info, err := h.service.DoComprador(c.GetInt64(middleware.ChaveContextoUsuarioID), id)
	h.responder(c, info, err)
}

// PorLink é GET /ingressos/:codigo/:token/qr (público, com o token do link).
func (h *QRHandler) PorLink(c *gin.Context) {
	info, err := h.service.PorLink(c.Param("codigo"), c.Param("token"))
	h.responder(c, info, err)
}
