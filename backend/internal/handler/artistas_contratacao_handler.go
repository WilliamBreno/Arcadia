package handler

import (
	"github.com/WilliamBreno/Arcadia/backend/internal/service"
	"github.com/gin-gonic/gin"
	"strconv"
)

func (h *GruposHandler) Oferta(c *gin.Context) {
	id, ok := paramID(c, "id")
	if !ok {
		return
	}
	if c.Request.Method == "PUT" {
		var d service.ArtistaOferta
		if c.ShouldBindJSON(&d) != nil {
			c.JSON(400, gin.H{"erro": "dados inválidos"})
			return
		}
		if e := h.service.SalvarOferta(usuarioDe(c), id, d); e != nil {
			h.erro(c, e)
			return
		}
	}
	v, e := h.service.Oferta(usuarioDe(c), id)
	if e != nil {
		h.erro(c, e)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(200, v)
}
func (h *GruposHandler) Municipios(c *gin.Context) {
	v, e := h.service.Municipios(c.Query("uf"))
	if e != nil {
		h.erro(c, e)
		return
	}
	c.Header("Cache-Control", "public, max-age=86400")
	c.JSON(200, v)
}
func (h *GruposHandler) ViagemConfig(c *gin.Context) {
	if c.Request.Method == "PUT" {
		var d service.ArtistaViagemConfig
		if c.ShouldBindJSON(&d) != nil {
			c.JSON(400, gin.H{"erro": "dados inválidos"})
			return
		}
		if e := h.service.SalvarViagemConfig(usuarioDe(c), d); e != nil {
			h.erro(c, e)
			return
		}
	}
	v, e := h.service.ViagemConfig()
	if e != nil {
		h.erro(c, e)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(200, v)
}
func (h *GruposHandler) EstimarViagem(c *gin.Context) {
	id, e := strconv.Atoi(c.Query("destino_id"))
	if e != nil || id <= 0 {
		c.JSON(400, gin.H{"erro": "escolha a cidade de destino"})
		return
	}
	v, e := h.service.EstimarViagem(c.Param("slug"), id)
	if e != nil {
		h.erro(c, e)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(200, v)
}
func (h *GruposHandler) CriarOrcamento(c *gin.Context) {
	var d service.ArtistaOrcamento
	if c.ShouldBindJSON(&d) != nil {
		c.JSON(400, gin.H{"erro": "dados inválidos"})
		return
	}
	v, e := h.service.CriarOrcamento(usuarioDe(c), c.Param("slug"), d)
	if e != nil {
		h.erro(c, e)
		return
	}
	c.JSON(201, gin.H{"id": v.ID})
}
func (h *GruposHandler) Orcamentos(c *gin.Context) {
	var id int64
	if c.Param("id") != "" {
		var ok bool
		id, ok = paramID(c, "id")
		if !ok {
			return
		}
	}
	pagina, _ := strconv.Atoi(c.Query("pagina"))
	if pagina < 1 {
		pagina = 1
	}
	if pagina > 100000 {
		c.JSON(400, gin.H{"erro": "página inválida"})
		return
	}
	v, e := h.service.Orcamentos(usuarioDe(c), id, pagina)
	if e != nil {
		h.erro(c, e)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(200, v)
}
func (h *GruposHandler) Conversa(c *gin.Context) {
	id, ok := paramID(c, "pedido")
	if !ok {
		return
	}
	antes, _ := strconv.ParseInt(c.Query("antes"), 10, 64)
	o, ms, e := h.service.Conversa(usuarioDe(c), id, antes)
	if e != nil {
		h.erro(c, e)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(200, gin.H{"pedido": o, "mensagens": ms})
}
func (h *GruposHandler) Mensagem(c *gin.Context) {
	id, ok := paramID(c, "pedido")
	if !ok {
		return
	}
	var d struct {
		Texto string `json:"texto"`
	}
	if c.ShouldBindJSON(&d) != nil {
		c.JSON(400, gin.H{"erro": "dados inválidos"})
		return
	}
	if e := h.service.Mensagem(usuarioDe(c), id, d.Texto); e != nil {
		h.erro(c, e)
		return
	}
	c.JSON(201, gin.H{"ok": true})
}
func (h *GruposHandler) AcaoOrcamento(c *gin.Context) {
	id, ok := paramID(c, "pedido")
	if !ok {
		return
	}
	var d service.AcaoOrcamento
	if c.ShouldBindJSON(&d) != nil {
		c.JSON(400, gin.H{"erro": "dados inválidos"})
		return
	}
	if e := h.service.AcaoOrcamento(usuarioDe(c), id, d); e != nil {
		h.erro(c, e)
		return
	}
	c.JSON(200, gin.H{"ok": true})
}
