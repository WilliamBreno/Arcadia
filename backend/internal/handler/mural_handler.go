package handler

import (
	"bytes"
	"context"
	"errors"
	"image"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/WilliamBreno/Arcadia/backend/internal/service"
	"github.com/gin-gonic/gin"
)

type MuralHandler struct{ s *service.MuralService }

func NovoMuralHandler(s *service.MuralService) *MuralHandler { return &MuralHandler{s} }
func muralErro(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrMuralAcesso):
		c.JSON(403, gin.H{"erro": err.Error()})
	case errors.Is(err, service.ErrMuralNaoEncontrado):
		c.JSON(404, gin.H{"erro": err.Error()})
	case errors.Is(err, service.ErrMuralLimite):
		c.JSON(429, gin.H{"erro": err.Error()})
	case errors.Is(err, service.ErrMuralDados), errors.Is(err, service.ErrFotoInvalida):
		c.JSON(422, gin.H{"erro": err.Error()})
	default:
		c.JSON(500, gin.H{"erro": "não foi possível concluir a operação no mural"})
	}
}
func (h *MuralHandler) Publico(c *gin.Context) {
	antes, _ := strconv.ParseInt(c.Query("antes"), 10, 64)
	r, e := h.s.Publico(c.Param("slug"), antes)
	if e != nil {
		muralErro(c, e)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(200, r)
}
func (h *MuralHandler) Eu(c *gin.Context) {
	r, e := h.s.Eu(usuarioDe(c), c.Param("slug"))
	if e != nil {
		muralErro(c, e)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(200, r)
}
func (h *MuralHandler) Criar(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 27*1024*1024)
	if err := c.Request.ParseMultipartForm(27 * 1024 * 1024); err != nil {
		c.JSON(413, gin.H{"erro": "envie até cinco fotos de no máximo 5 MB cada"})
		return
	}
	defer c.Request.MultipartForm.RemoveAll()
	arquivos := c.Request.MultipartForm.File["fotos"]
	if len(arquivos) > 5 {
		c.JSON(422, gin.H{"erro": "cada publicação aceita até cinco fotos"})
		return
	}
	fotos := [][]byte{}
	for _, cab := range arquivos {
		if cab.Size > 5*1024*1024 {
			c.JSON(413, gin.H{"erro": "cada foto deve ter até 5 MB"})
			return
		}
		arq, err := cab.Open()
		if err != nil {
			muralErro(c, err)
			return
		}
		b, err := io.ReadAll(io.LimitReader(arq, 5*1024*1024+1))
		arq.Close()
		if err != nil || len(b) > 5*1024*1024 {
			c.JSON(413, gin.H{"erro": "foto maior que 5 MB"})
			return
		}
		cfg, formato, err := image.DecodeConfig(bytes.NewReader(b))
		if err != nil || (formato != "jpeg" && formato != "png") || int64(cfg.Width)*int64(cfg.Height) > 20000000 {
			c.JSON(422, gin.H{"erro": "envie JPEG ou PNG com no máximo 20 megapixels"})
			return
		}
		jpg, err := service.ReduzirParaJPEG(b, 2048)
		if err != nil {
			muralErro(c, err)
			return
		}
		fotos = append(fotos, jpg)
	}
	id, err := h.s.Criar(c.Request.Context(), usuarioDe(c), c.Param("slug"), c.PostForm("texto"), c.PostForm("chave"), fotos, c.PostForm("consentiu") == "true", c.PostForm("participando") == "true")
	if err != nil {
		muralErro(c, err)
		return
	}
	c.JSON(201, gin.H{"id": id, "status": "pendente"})
}
func (h *MuralHandler) Foto(c *gin.Context) {
	pid, ok := paramID(c, "post")
	if !ok {
		return
	}
	fid, ok := paramID(c, "foto")
	if !ok {
		return
	}
	b, e := h.s.Foto(usuarioDe(c), c.Param("slug"), pid, fid)
	if e != nil {
		muralErro(c, e)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Data(200, "image/jpeg", b)
}
func (h *MuralHandler) RemoverMeu(c *gin.Context) {
	id, ok := paramID(c, "post")
	if !ok {
		return
	}
	if e := h.s.RemoverMeu(usuarioDe(c), c.Param("slug"), id); e != nil {
		muralErro(c, e)
		return
	}
	c.Status(204)
}
func (h *MuralHandler) Curtir(c *gin.Context) {
	id, ok := paramID(c, "post")
	if !ok {
		return
	}
	var r struct {
		Curtida bool `json:"curtida"`
	}
	if c.ShouldBindJSON(&r) != nil {
		c.JSON(400, gin.H{"erro": "curtida inválida"})
		return
	}
	if e := h.s.Curtir(usuarioDe(c), c.Param("slug"), id, r.Curtida); e != nil {
		muralErro(c, e)
		return
	}
	c.Status(204)
}
func (h *MuralHandler) Denunciar(c *gin.Context) {
	id, ok := paramID(c, "post")
	if !ok {
		return
	}
	var r struct {
		Motivo string `json:"motivo"`
	}
	if c.ShouldBindJSON(&r) != nil {
		c.JSON(400, gin.H{"erro": "informe o motivo da denúncia"})
		return
	}
	if e := h.s.Denunciar(usuarioDe(c), c.Param("slug"), id, r.Motivo); e != nil {
		muralErro(c, e)
		return
	}
	c.Status(204)
}
func (h *MuralHandler) Moderacao(c *gin.Context) {
	antes, _ := strconv.ParseInt(c.Query("antes"), 10, 64)
	r, e := h.s.Moderacao(usuarioDe(c), c.Param("slug"), c.DefaultQuery("status", "pendente"), antes)
	if e != nil {
		muralErro(c, e)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(200, r)
}
func (h *MuralHandler) Decidir(c *gin.Context) {
	id, ok := paramID(c, "post")
	if !ok {
		return
	}
	var r struct {
		Acao   string `json:"acao"`
		Motivo string `json:"motivo"`
		Ciente bool   `json:"ciente_alertas"`
	}
	if c.ShouldBindJSON(&r) != nil {
		c.JSON(400, gin.H{"erro": "decisão inválida"})
		return
	}
	if e := h.s.Decidir(usuarioDe(c), c.Param("slug"), id, r.Acao, r.Motivo, r.Ciente); e != nil {
		muralErro(c, e)
		return
	}
	c.Status(204)
}
func (h *MuralHandler) Denuncias(c *gin.Context) {
	id, ok := paramID(c, "post")
	if !ok {
		return
	}
	r, e := h.s.Denuncias(usuarioDe(c), c.Param("slug"), id)
	if e != nil {
		muralErro(c, e)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(200, r)
}
func (h *MuralHandler) Analisar(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 55*time.Second)
	defer cancel()
	n, e := h.s.AnalisarPendentes(ctx)
	if e != nil {
		muralErro(c, e)
		return
	}
	c.JSON(200, gin.H{"analisados": n})
}
