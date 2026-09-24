package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/WilliamBreno/Arcadia/backend/internal/middleware"
	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
	"github.com/WilliamBreno/Arcadia/backend/internal/service"
)

// AdminHandler agrupa relatórios e reembolsos do painel admin (item 2.3).
type AdminHandler struct {
	reembolsos   *repository.ReembolsoRepository
	itensPedido  *repository.ItemPedidoRepository
	cancelamento *service.CancelamentoService
}

func NovoAdminHandler(r *repository.ReembolsoRepository, i *repository.ItemPedidoRepository, c *service.CancelamentoService) *AdminHandler {
	return &AdminHandler{reembolsos: r, itensPedido: i, cancelamento: c}
}

// Relatorios é GET /admin/relatorios?de=YYYY-MM-DD&ate=YYYY-MM-DD.
func (h *AdminHandler) Relatorios(c *gin.Context) {
	de, ate := parseDataQuery(c.Query("de")), parseDataQuery(c.Query("ate"))

	receita, err := h.itensPedido.ReceitaPlataforma(de, ate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao calcular receita"})
		return
	}
	resumo, err := h.reembolsos.Resumo()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao resumir reembolsos"})
		return
	}

	type linha struct {
		Tipo          string `json:"tipo"`
		Status        string `json:"status"`
		Quantidade    int64  `json:"quantidade"`
		ValorCentavos int64  `json:"valor_centavos"`
	}
	reembolsos := make([]linha, 0, len(resumo))
	var falhas int64
	for _, r := range resumo {
		reembolsos = append(reembolsos, linha{r.Tipo, r.Status, r.Quantidade, r.ValorCentavos})
		if r.Status == "falhou" {
			falhas += r.Quantidade
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"receita": gin.H{
			"itens": receita.Itens, "taxa_centavos": receita.TaxaCentavos,
			"garantia_centavos": receita.GarantiaCentavos, "total_centavos": receita.TaxaCentavos + receita.GarantiaCentavos,
		},
		"reembolsos": reembolsos, "reembolsos_falhos": falhas,
	})
}

type reembolsoResposta struct {
	ID            int64     `json:"id"`
	EventoTitulo  string    `json:"evento_titulo"`
	Codigo        string    `json:"codigo"`
	TitularNome   string    `json:"titular_nome"`
	ValorCentavos int64     `json:"valor_centavos"`
	Tipo          string    `json:"tipo"`
	Status        string    `json:"status"`
	Motivo        string    `json:"motivo"`
	CriadoEm      time.Time `json:"criado_em"`
}

// ListarReembolsos é GET /admin/reembolsos?status=falhou.
func (h *AdminHandler) ListarReembolsos(c *gin.Context) {
	lista, err := h.reembolsos.ListarDetalhados(c.Query("status"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao listar reembolsos"})
		return
	}
	resp := make([]reembolsoResposta, 0, len(lista))
	for _, r := range lista {
		resp = append(resp, reembolsoResposta{
			ID: r.ID, EventoTitulo: r.EventoTitulo, Codigo: r.Codigo, TitularNome: r.TitularNome,
			ValorCentavos: r.ValorCentavos, Tipo: string(r.Tipo), Status: string(r.Status), Motivo: r.Motivo, CriadoEm: r.CriadoEm,
		})
	}
	c.JSON(http.StatusOK, resp)
}

// ReprocessarReembolso é POST /admin/reembolsos/:id/reprocessar.
func (h *AdminHandler) ReprocessarReembolso(c *gin.Context) {
	id, ok := idDaURL(c)
	if !ok {
		return
	}
	adminID := c.GetInt64(middleware.ChaveContextoUsuarioID)
	if err := h.cancelamento.ReprocessarReembolso(id, adminID); err != nil {
		if errors.Is(err, service.ErrReembolsoNaoReprocessavel) {
			c.JSON(http.StatusConflict, gin.H{"erro": err.Error()})
			return
		}
		c.JSON(http.StatusBadGateway, gin.H{"erro": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ReprocessarFalhas é o job POST /jobs/reprocessar-reembolsos.
func (h *AdminHandler) ReprocessarFalhas(c *gin.Context) {
	sucessos, falhas, err := h.cancelamento.ReprocessarFalhas()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao reprocessar reembolsos"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"sucessos": sucessos, "falhas": falhas})
}
