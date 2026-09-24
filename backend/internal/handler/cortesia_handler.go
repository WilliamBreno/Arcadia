package handler

import (
	"crypto/subtle"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/WilliamBreno/Arcadia/backend/internal/middleware"
	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
	"github.com/WilliamBreno/Arcadia/backend/internal/service"
)

type CortesiaHandler struct {
	organizadorHandler *OrganizadorHandler
	service            *service.CortesiaService
	fichas             *service.FichaService
	itensPedido        *repository.ItemPedidoRepository
	tipos              *repository.TipoIngressoRepository
	eventos            *repository.EventoRepository
}

func NovoCortesiaHandler(
	o *OrganizadorHandler, s *service.CortesiaService, f *service.FichaService,
	i *repository.ItemPedidoRepository, t *repository.TipoIngressoRepository, e *repository.EventoRepository,
) *CortesiaHandler {
	return &CortesiaHandler{organizadorHandler: o, service: s, fichas: f, itensPedido: i, tipos: t, eventos: e}
}

func (h *CortesiaHandler) responderErro(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrEventoNaoPertenceAoOrganizador):
		c.JSON(http.StatusForbidden, gin.H{"erro": "evento não pertence a este organizador"})
	case errors.Is(err, service.ErrCortesiaNaoRevogavel):
		c.JSON(http.StatusConflict, gin.H{"erro": err.Error()})
	default:
		c.JSON(http.StatusUnprocessableEntity, gin.H{"erro": err.Error()})
	}
}

type emitirCortesiaRequest struct {
	TipoIngressoID int64  `json:"tipo_ingresso_id" binding:"required"`
	Nome           string `json:"nome" binding:"required"`
	Email          string `json:"email" binding:"required,email"`
	Quantidade     int    `json:"quantidade" binding:"required"`
}

// Emitir é POST /org/eventos/:id/cortesias.
func (h *CortesiaHandler) Emitir(c *gin.Context) {
	org, ok := h.organizadorHandler.ObterOrganizadorAtual(c)
	if !ok {
		return
	}
	eventoID, ok := idDaURL(c)
	if !ok {
		return
	}
	var req emitirCortesiaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "dados inválidos"})
		return
	}
	usuarioID := c.GetInt64(middleware.ChaveContextoUsuarioID)
	itens, err := h.service.Emitir(org.ID, usuarioID, eventoID, req.TipoIngressoID, req.Nome, req.Email, req.Quantidade)
	if err != nil {
		h.responderErro(c, err)
		return
	}
	codigos := make([]string, len(itens))
	for i := range itens {
		codigos[i] = itens[i].Codigo
	}
	c.JSON(http.StatusCreated, gin.H{"emitidas": len(itens), "codigos": codigos})
}

// Listar é GET /org/eventos/:id/cortesias.
func (h *CortesiaHandler) Listar(c *gin.Context) {
	org, ok := h.organizadorHandler.ObterOrganizadorAtual(c)
	if !ok {
		return
	}
	eventoID, ok := idDaURL(c)
	if !ok {
		return
	}
	itens, err := h.service.Listar(org.ID, eventoID)
	if err != nil {
		h.responderErro(c, err)
		return
	}
	resp := make([]vendaItemResposta, 0, len(itens))
	for i := range itens {
		resp = append(resp, paraVendaItemResposta(&itens[i]))
	}
	c.JSON(http.StatusOK, resp)
}

// Revogar é DELETE /org/eventos/:id/cortesias/:itemId.
func (h *CortesiaHandler) Revogar(c *gin.Context) {
	org, ok := h.organizadorHandler.ObterOrganizadorAtual(c)
	if !ok {
		return
	}
	eventoID, ok := idDaURL(c)
	if !ok {
		return
	}
	itemID, err := strconv.ParseInt(c.Param("itemId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "id inválido"})
		return
	}
	if err := h.service.Revogar(org.ID, eventoID, itemID); err != nil {
		h.responderErro(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ExportarCSV é GET /org/eventos/:id/exportar.csv?tipo=compradores|participantes.
func (h *CortesiaHandler) ExportarCSV(c *gin.Context) {
	org, ok := h.organizadorHandler.ObterOrganizadorAtual(c)
	if !ok {
		return
	}
	eventoID, ok := idDaURL(c)
	if !ok {
		return
	}

	tipo := c.DefaultQuery("tipo", "compradores")
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", `attachment; filename="`+tipo+`-evento-`+strconv.FormatInt(eventoID, 10)+`.csv"`)

	var err error
	switch tipo {
	case "compradores":
		_, _ = c.Writer.Write([]byte("\xEF\xBB\xBF")) // BOM: Excel abre com acentos certos
		err = h.service.ExportarCompradores(c.Writer, org.ID, eventoID)
	case "participantes":
		fichas, errLista := h.fichas.ListarDoOrganizador(org.ID, eventoID)
		if errLista != nil {
			h.responderErro(c, errLista)
			return
		}
		_, _ = c.Writer.Write([]byte("\xEF\xBB\xBF"))
		err = service.EscreverParticipantesCSV(c.Writer, fichas)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"erro": "tipo deve ser compradores ou participantes"})
		return
	}
	if err != nil && !c.Writer.Written() {
		h.responderErro(c, err)
	}
}

// IngressoPublico é GET /ingressos/:codigo/:token — página do ingresso para
// quem recebeu cortesia por e-mail sem ter conta. Exige o qr_token (segredo
// que só quem recebeu o link tem), comparado em tempo constante.
func (h *CortesiaHandler) IngressoPublico(c *gin.Context) {
	item, err := h.itensPedido.BuscarPorCodigo(c.Param("codigo"))
	if err != nil || subtle.ConstantTimeCompare([]byte(item.QRToken), []byte(c.Param("token"))) != 1 {
		c.JSON(http.StatusNotFound, gin.H{"erro": "ingresso não encontrado"})
		return
	}
	tipo, err := h.tipos.BuscarPorID(item.TipoIngressoID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"erro": "ingresso não encontrado"})
		return
	}
	evento, err := h.eventos.BuscarPorID(tipo.EventoID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"erro": "ingresso não encontrado"})
		return
	}
	var inicio *time.Time = evento.InicioEm
	c.JSON(http.StatusOK, gin.H{
		"codigo": item.Codigo, "qr_token": item.QRToken, "titular_nome": item.TitularNome,
		"tipo_ingresso_nome": tipo.Nome, "status": item.Status, "utilizado_em": item.UtilizadoEm,
		"evento_titulo": evento.Titulo, "evento_slug": evento.Slug, "evento_inicio_em": inicio,
	})
}
