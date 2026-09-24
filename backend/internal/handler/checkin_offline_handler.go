package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/WilliamBreno/Arcadia/backend/internal/middleware"
	"github.com/WilliamBreno/Arcadia/backend/internal/service"
)

func (h *CheckinHandler) erroOffline(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrSemAcessoCheckin):
		c.JSON(http.StatusForbidden, gin.H{"erro": err.Error()})
	case errors.Is(err, service.ErrOfflineIndisponivelRotativo), errors.Is(err, service.ErrLoteOfflineInvalido):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"erro": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro no check-in offline"})
	}
}

// Pacote é GET /checkin/eventos/:id/pacote — dados para validar sem internet.
func (h *CheckinHandler) Pacote(c *gin.Context) {
	eventoID, ok := idDaURL(c)
	if !ok {
		return
	}
	p, err := h.service.Pacote(c.GetInt64(middleware.ChaveContextoUsuarioID), eventoID)
	if err != nil {
		h.erroOffline(c, err)
		return
	}
	sessoes := make([]sessaoResposta, 0, len(p.Sessoes))
	for i := range p.Sessoes {
		sessoes = append(sessoes, paraSessaoResposta(&p.Sessoes[i]))
	}
	ingressos := make([]gin.H, 0, len(p.Ingressos))
	for _, i := range p.Ingressos {
		ingressos = append(ingressos, gin.H{
			"codigo": i.Codigo, "hash_token": i.HashToken, "nome": i.Nome, "tipo": i.TipoNome, "meia_entrada": i.MeiaEntrada,
			"status": i.Status, "utilizado_em": i.UtilizadoEm, "sessao_ids": idsOuVazio(i.SessaoIDs),
		})
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"evento_id": p.EventoID, "gerado_em": p.GeradoEm, "sessoes": sessoes, "ingressos": ingressos})
}

type entradaOfflineRequest struct {
	EntradaID string    `json:"entrada_id" binding:"required,max=64"`
	Codigo    string    `json:"codigo" binding:"required"`
	SessaoID  *int64    `json:"sessao_id"`
	LidoEm    time.Time `json:"lido_em" binding:"required"`
}

type sincronizarRequest struct {
	EventoID int64                   `json:"evento_id" binding:"required"`
	Entradas []entradaOfflineRequest `json:"entradas" binding:"required"`
}

// Sincronizar é POST /checkin/sincronizar.
func (h *CheckinHandler) Sincronizar(c *gin.Context) {
	var req sincronizarRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "dados inválidos"})
		return
	}
	entradas := make([]service.EntradaOffline, 0, len(req.Entradas))
	for _, e := range req.Entradas {
		entradas = append(entradas, service.EntradaOffline{EntradaID: e.EntradaID, Codigo: e.Codigo, SessaoID: e.SessaoID, LidoEm: e.LidoEm})
	}
	res, err := h.service.Sincronizar(c.GetInt64(middleware.ChaveContextoUsuarioID), req.EventoID, entradas)
	if err != nil {
		h.erroOffline(c, err)
		return
	}
	resp := make([]gin.H, 0, len(res))
	for _, r := range res {
		resp = append(resp, gin.H{"entrada_id": r.EntradaID, "codigo": r.Codigo, "resultado": r.Resultado})
	}
	c.JSON(http.StatusOK, gin.H{"resultados": resp})
}

// Conflitos é GET /checkin/eventos/:id/conflitos.
func (h *CheckinHandler) Conflitos(c *gin.Context) {
	eventoID, ok := idDaURL(c)
	if !ok {
		return
	}
	lista, err := h.service.Conflitos(c.GetInt64(middleware.ChaveContextoUsuarioID), eventoID)
	if err != nil {
		h.erroOffline(c, err)
		return
	}
	resp := make([]gin.H, 0, len(lista))
	for _, l := range lista {
		resp = append(resp, gin.H{"codigo": l.Codigo, "titular": l.TitularNome, "lido_em": l.LidoEm, "staff": l.StaffNome, "sessao_id": l.SessaoID})
	}
	c.JSON(http.StatusOK, resp)
}
