package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/WilliamBreno/Arcadia/backend/internal/middleware"
	"github.com/WilliamBreno/Arcadia/backend/internal/service"
)

// Dia do evento (item 9.6): painel ao vivo, portões, importados e
// credenciais impressas.
type DiaDoEventoHandler struct {
	checkin *service.CheckinService
	svc     *service.DiaDoEventoService
	org     *OrganizadorHandler
}

func NovoDiaDoEventoHandler(c *service.CheckinService, s *service.DiaDoEventoService, org *OrganizadorHandler) *DiaDoEventoHandler {
	return &DiaDoEventoHandler{checkin: c, svc: s, org: org}
}

func (h *DiaDoEventoHandler) erro(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrSemAcessoCheckin), errors.Is(err, service.ErrEventoNaoPertenceAoOrganizador):
		c.JSON(http.StatusForbidden, gin.H{"erro": "sem acesso a este evento"})
	case errors.Is(err, service.ErrDadosDiaEvento):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"erro": err.Error()})
	case errors.Is(err, service.ErrCredencialEvento):
		c.JSON(http.StatusNotFound, gin.H{"erro": err.Error()})
	default:
		slog.Error("dia do evento", "erro", err)
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro no dia do evento"})
	}
}

// ---------- equipe de check-in ----------

// Portoes é GET /checkin/eventos/:id/portoes.
func (h *DiaDoEventoHandler) Portoes(c *gin.Context) {
	id, ok := idDaURL(c)
	if !ok {
		return
	}
	nomes, err := h.checkin.Portoes(c.GetInt64(middleware.ChaveContextoUsuarioID), id)
	if err != nil {
		h.erro(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"portoes": nomes})
}

// Importado é POST /checkin/eventos/:id/importado: o texto lido do QR de
// outra plataforma.
func (h *DiaDoEventoHandler) Importado(c *gin.Context) {
	id, ok := idDaURL(c)
	if !ok {
		return
	}
	var req struct {
		Codigo string `json:"codigo" binding:"required"`
		Portao string `json:"portao"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "dados inválidos"})
		return
	}
	r, err := h.checkin.ValidarImportado(c.GetInt64(middleware.ChaveContextoUsuarioID), id, req.Codigo, req.Portao)
	if err != nil {
		h.erro(c, err)
		return
	}
	resp := gin.H{"resultado": r.Resultado, "titular_nome": r.Nome}
	if r.Resultado != service.ResultadoNaoEncontrado {
		resp["tipo_ingresso_nome"] = service.RotuloImportado(r.Origem, r.Tipo)
	}
	if r.UtilizadoEm != nil {
		resp["utilizado_em"] = r.UtilizadoEm.Format("15:04:05")
	}
	c.JSON(http.StatusOK, resp)
}

// Credencial é POST /checkin/eventos/:id/credencial-evento (EVVEC:...).
func (h *DiaDoEventoHandler) Credencial(c *gin.Context) {
	id, ok := idDaURL(c)
	if !ok {
		return
	}
	var req struct {
		Texto  string `json:"texto" binding:"required"`
		Portao string `json:"portao"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "dados inválidos"})
		return
	}
	r, err := h.checkin.LerCredencialEvento(c.GetInt64(middleware.ChaveContextoUsuarioID), id, req.Texto, req.Portao)
	if err != nil {
		h.erro(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"nome": r.Nome, "funcao": r.Funcao, "primeira_vez": r.PrimeiraVez})
}

// AoVivo é GET /checkin/eventos/:id/ao-vivo.
func (h *DiaDoEventoHandler) AoVivo(c *gin.Context) {
	id, ok := idDaURL(c)
	if !ok {
		return
	}
	p, err := h.checkin.AoVivo(c.GetInt64(middleware.ChaveContextoUsuarioID), id)
	if err != nil {
		h.erro(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, p)
}

// ---------- organizador ----------

func (h *DiaDoEventoHandler) orgEvento(c *gin.Context) (int64, int64, bool) {
	org, ok := h.org.ObterOrganizadorAtual(c)
	if !ok {
		return 0, 0, false
	}
	id, ok := idDaURL(c)
	return org.ID, id, ok
}

// OrgPortoes é GET /org/eventos/:id/portoes.
func (h *DiaDoEventoHandler) OrgPortoes(c *gin.Context) {
	orgID, id, ok := h.orgEvento(c)
	if !ok {
		return
	}
	nomes, err := h.svc.Portoes(orgID, id)
	if err != nil {
		h.erro(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"portoes": nomes})
}

// DefinirPortoes é PUT /org/eventos/:id/portoes.
func (h *DiaDoEventoHandler) DefinirPortoes(c *gin.Context) {
	orgID, id, ok := h.orgEvento(c)
	if !ok {
		return
	}
	var req struct {
		Portoes []string `json:"portoes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "dados inválidos"})
		return
	}
	if err := h.svc.DefinirPortoes(orgID, id, req.Portoes); err != nil {
		h.erro(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// Importados é GET /org/eventos/:id/importados.
func (h *DiaDoEventoHandler) Importados(c *gin.Context) {
	orgID, id, ok := h.orgEvento(c)
	if !ok {
		return
	}
	r, err := h.svc.ResumoImportados(orgID, id)
	if err != nil {
		h.erro(c, err)
		return
	}
	c.JSON(http.StatusOK, r)
}

const limiteCSVImportacao = 8 << 20

// Importar é POST /org/eventos/:id/importados {csv, origem}.
func (h *DiaDoEventoHandler) Importar(c *gin.Context) {
	orgID, id, ok := h.orgEvento(c)
	if !ok {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limiteCSVImportacao)
	var req struct {
		CSV    string `json:"csv" binding:"required"`
		Origem string `json:"origem"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "arquivo vazio ou maior que 8 MB"})
		return
	}
	r, err := h.svc.Importar(orgID, id, req.CSV, req.Origem)
	if err != nil {
		h.erro(c, err)
		return
	}
	c.JSON(http.StatusOK, r)
}

// ApagarImportados é DELETE /org/eventos/:id/importados[?todos=1].
func (h *DiaDoEventoHandler) ApagarImportados(c *gin.Context) {
	orgID, id, ok := h.orgEvento(c)
	if !ok {
		return
	}
	n, err := h.svc.ApagarImportados(orgID, id, c.Query("todos") == "1")
	if err != nil {
		h.erro(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"apagados": n})
}

// Credenciais é GET /org/eventos/:id/credenciais (para imprimir).
func (h *DiaDoEventoHandler) Credenciais(c *gin.Context) {
	orgID, id, ok := h.orgEvento(c)
	if !ok {
		return
	}
	lista, err := h.svc.Credenciais(orgID, id)
	if err != nil {
		h.erro(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"credenciais": lista})
}

// CriarCredencial é POST /org/eventos/:id/credenciais {nome, funcao}.
func (h *DiaDoEventoHandler) CriarCredencial(c *gin.Context) {
	orgID, id, ok := h.orgEvento(c)
	if !ok {
		return
	}
	var req struct {
		Nome   string `json:"nome"`
		Funcao string `json:"funcao"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "dados inválidos"})
		return
	}
	cred, err := h.svc.CriarCredencial(orgID, id, req.Nome, req.Funcao)
	if err != nil {
		h.erro(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": cred.ID})
}

// DesativarCredencial é DELETE /org/eventos/:id/credenciais/:cid.
func (h *DiaDoEventoHandler) DesativarCredencial(c *gin.Context) {
	orgID, id, ok := h.orgEvento(c)
	if !ok {
		return
	}
	cid, ok := paramInt(c, "cid")
	if !ok {
		return
	}
	if err := h.svc.DesativarCredencial(orgID, id, cid); err != nil {
		h.erro(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
