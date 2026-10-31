package handler

import (
	"bytes"
	"errors"
	"image"
	"image/jpeg"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/WilliamBreno/Arcadia/backend/internal/service"
	"github.com/gin-gonic/gin"
)

type ChatEventoHandler struct{ s *service.ChatEventoService }

func NovoChatEventoHandler(s *service.ChatEventoService) *ChatEventoHandler {
	return &ChatEventoHandler{s}
}
func chatErro(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrChatLimite):
		c.Header("Retry-After", "60")
		c.JSON(429, gin.H{"erro": err.Error()})
	case errors.Is(err, service.ErrChatAcesso):
		c.JSON(403, gin.H{"erro": err.Error()})
	case errors.Is(err, service.ErrChatDados):
		c.JSON(422, gin.H{"erro": err.Error()})
	case errors.Is(err, service.ErrChatNaoEncontrado):
		c.JSON(404, gin.H{"erro": err.Error()})
	default:
		c.JSON(500, gin.H{"erro": "não foi possível concluir a operação no chat"})
	}
}
func (h *ChatEventoHandler) Canais(c *gin.Context) {
	r, e := h.s.Canais(usuarioDe(c))
	if e != nil {
		chatErro(c, e)
		return
	}
	c.JSON(200, r)
}
func (h *ChatEventoHandler) Pagina(c *gin.Context) {
	id, ok := paramID(c, "id")
	if !ok {
		return
	}
	antes, _ := strconv.ParseInt(c.Query("antes"), 10, 64)
	r, e := h.s.Pagina(usuarioDe(c), id, antes)
	if e != nil {
		chatErro(c, e)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(200, r)
}
func (h *ChatEventoHandler) Enviar(c *gin.Context) {
	id, ok := paramID(c, "id")
	if !ok {
		return
	}
	if e := h.s.Acesso(usuarioDe(c), id); e != nil {
		chatErro(c, e)
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 6*1024*1024)
	var req struct {
		Texto string `json:"texto"`
		Chave string `json:"chave"`
	}
	var foto []byte
	if c.ContentType() == "multipart/form-data" {
		if e := c.Request.ParseMultipartForm(6 * 1024 * 1024); e != nil {
			c.JSON(413, gin.H{"erro": "envie uma foto JPEG ou PNG de até 5 MB"})
			return
		}
		if c.Request.MultipartForm != nil {
			defer c.Request.MultipartForm.RemoveAll()
		}
		req.Texto, req.Chave = c.PostForm("texto"), c.PostForm("chave")
		arq, cab, e := c.Request.FormFile("arquivo")
		if e != nil {
			c.JSON(422, gin.H{"erro": "envie uma foto JPEG ou PNG"})
			return
		}
		defer arq.Close()
		if cab.Size > 5*1024*1024 {
			c.JSON(413, gin.H{"erro": "foto maior que 5 MB"})
			return
		}
		original, e := io.ReadAll(io.LimitReader(arq, 5*1024*1024+1))
		if e != nil || len(original) > 5*1024*1024 {
			c.JSON(413, gin.H{"erro": "foto maior que 5 MB"})
			return
		}
		config, formato, e := image.DecodeConfig(bytes.NewReader(original))
		if e != nil || (formato != "jpeg" && formato != "png") || config.Width > 6000 || config.Height > 6000 || int64(config.Width)*int64(config.Height) > 20000000 {
			c.JSON(422, gin.H{"erro": "foto inválida ou com resolução muito alta (máximo 20 megapixels)"})
			return
		}
		img, _, e := image.Decode(bytes.NewReader(original))
		if e != nil {
			c.JSON(422, gin.H{"erro": "foto inválida"})
			return
		}
		var buf bytes.Buffer
		if e = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 80}); e != nil {
			chatErro(c, e)
			return
		}
		foto = buf.Bytes()
		if len(foto) > 5*1024*1024 {
			c.JSON(413, gin.H{"erro": "foto processada maior que 5 MB"})
			return
		}
	} else if e := c.ShouldBindJSON(&req); e != nil {
		c.JSON(400, gin.H{"erro": "mensagem inválida"})
		return
	}
	mid, e := h.s.Enviar(usuarioDe(c), id, req.Texto, req.Chave, foto)
	if e != nil {
		chatErro(c, e)
		return
	}
	c.JSON(201, gin.H{"id": mid})
}
func (h *ChatEventoHandler) Ler(c *gin.Context) {
	id, ok := paramID(c, "id")
	if !ok {
		return
	}
	var r struct {
		Ate   int64 `json:"ate"`
		Email *bool `json:"email_ativo"`
	}
	if c.ShouldBindJSON(&r) != nil {
		c.JSON(400, gin.H{"erro": "leitura inválida"})
		return
	}
	if e := h.s.Ler(usuarioDe(c), id, r.Ate, r.Email); e != nil {
		chatErro(c, e)
		return
	}
	c.Status(204)
}
func (h *ChatEventoHandler) Fixar(c *gin.Context) {
	id, ok := paramID(c, "id")
	if !ok {
		return
	}
	mid, ok := paramID(c, "mensagem")
	if !ok {
		return
	}
	var r struct {
		Fixada bool `json:"fixada"`
	}
	if c.ShouldBindJSON(&r) != nil {
		c.JSON(400, gin.H{"erro": "aviso inválido"})
		return
	}
	if e := h.s.Fixar(usuarioDe(c), id, mid, r.Fixada); e != nil {
		chatErro(c, e)
		return
	}
	c.Status(204)
}
func (h *ChatEventoHandler) Foto(c *gin.Context) {
	id, ok := paramID(c, "id")
	if !ok {
		return
	}
	mid, ok := paramID(c, "mensagem")
	if !ok {
		return
	}
	foto, e := h.s.Foto(usuarioDe(c), id, mid)
	if e != nil {
		chatErro(c, e)
		return
	}
	c.Header("Cache-Control", "no-store, private")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Data(200, "image/jpeg", foto)
}
func (h *ChatEventoHandler) Apoio(c *gin.Context) {
	if c.Request.Method == "POST" {
		var r struct {
			Texto string `json:"texto"`
		}
		if c.ShouldBindJSON(&r) != nil {
			c.JSON(400, gin.H{"erro": "pedido inválido"})
			return
		}
		if e := h.s.PedirAtencao(usuarioDe(c), c.Param("slug"), r.Texto); e != nil {
			chatErro(c, e)
			return
		}
	}
	r, e := h.s.Apoio(usuarioDe(c), c.Param("slug"))
	if e != nil {
		chatErro(c, e)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(200, r)
}
func (h *ChatEventoHandler) Atender(c *gin.Context) {
	id, ok := paramID(c, "id")
	if !ok {
		return
	}
	aid, ok := paramID(c, "atencao")
	if !ok {
		return
	}
	if e := h.s.Atender(usuarioDe(c), id, aid); e != nil {
		chatErro(c, e)
		return
	}
	c.Status(204)
}
func (h *ChatEventoHandler) Resumos(c *gin.Context) {
	n, e := h.s.Resumos(time.Now())
	if e != nil {
		chatErro(c, e)
		return
	}
	c.JSON(200, gin.H{"resumos": n})
}
