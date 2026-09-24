package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
	"github.com/WilliamBreno/Arcadia/backend/internal/service"
)

type RepasseHandler struct {
	organizadorHandler *OrganizadorHandler
	service            *service.RepasseService
}

func NovoRepasseHandler(o *OrganizadorHandler, s *service.RepasseService) *RepasseHandler {
	return &RepasseHandler{organizadorHandler: o, service: s}
}

type repasseResposta struct {
	ID                      int64      `json:"id"`
	EventoID                int64      `json:"evento_id"`
	EventoTitulo            string     `json:"evento_titulo,omitempty"`
	OrganizadorNome         string     `json:"organizador_nome,omitempty"`
	ChavePix                string     `json:"chave_pix,omitempty"`
	TipoChavePix            string     `json:"tipo_chave_pix,omitempty"`
	ValorBrutoCentavos      int64      `json:"valor_bruto_centavos"`
	TaxaProcessadorCentavos int64      `json:"taxa_processador_centavos"`
	ValorLiquidoCentavos    int64      `json:"valor_liquido_centavos"`
	Status                  string     `json:"status"`
	LiberarEm               time.Time  `json:"liberar_em"`
	PagoEm                  *time.Time `json:"pago_em"`
	ComprovanteURL          string     `json:"comprovante_url"`
	Observacao              string     `json:"observacao"`
}

func paraRepasseResposta(r *repository.RepasseDetalhado, comPix bool) repasseResposta {
	resp := repasseResposta{
		ID: r.ID, EventoID: r.EventoID, EventoTitulo: r.EventoTitulo,
		ValorBrutoCentavos: r.ValorBrutoCentavos, TaxaProcessadorCentavos: r.TaxaProcessadorCentavos,
		ValorLiquidoCentavos: r.ValorLiquidoCentavos, Status: string(r.Status), LiberarEm: r.LiberarEm,
		PagoEm: r.PagoEm, ComprovanteURL: r.ComprovanteURL, Observacao: r.Observacao,
	}
	if comPix {
		resp.OrganizadorNome, resp.ChavePix, resp.TipoChavePix = r.OrganizadorNome, r.ChavePix, r.TipoChavePix
	}
	return resp
}

// FinanceiroEvento é GET /org/eventos/:id/financeiro.
func (h *RepasseHandler) FinanceiroEvento(c *gin.Context) {
	org, ok := h.organizadorHandler.ObterOrganizadorDono(c)
	if !ok {
		return
	}
	eventoID, ok := idDaURL(c)
	if !ok {
		return
	}
	f, err := h.service.FinanceiroDoEvento(org.ID, eventoID)
	if err != nil {
		if errors.Is(err, service.ErrEventoNaoPertenceAoOrganizador) {
			c.JSON(http.StatusForbidden, gin.H{"erro": "evento não pertence a este organizador"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao calcular financeiro"})
		return
	}
	resp := gin.H{
		"bruto_centavos": f.BrutoCentavos, "taxa_processador_centavos": f.TaxaProcessadorCentavos,
		"liquido_centavos": f.LiquidoCentavos, "liberar_em": f.LiberarEm, "repasse_status": nil,
	}
	if f.Repasse != nil {
		resp["repasse_status"] = f.Repasse.Status
	}
	c.JSON(http.StatusOK, resp)
}

// Extrato é GET /org/repasses: a receber + histórico.
func (h *RepasseHandler) Extrato(c *gin.Context) {
	org, ok := h.organizadorHandler.ObterOrganizadorDono(c)
	if !ok {
		return
	}
	lista, err := h.service.ListarDoOrganizador(org.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao listar repasses"})
		return
	}
	var aReceber int64
	itens := make([]repasseResposta, 0, len(lista))
	for i := range lista {
		if lista[i].Status == "pendente" {
			aReceber += lista[i].ValorLiquidoCentavos
		}
		itens = append(itens, paraRepasseResposta(&lista[i], false))
	}
	c.JSON(http.StatusOK, gin.H{"a_receber_centavos": aReceber, "repasses": itens})
}

// ListarAdmin é GET /admin/repasses?status=.
func (h *RepasseHandler) ListarAdmin(c *gin.Context) {
	lista, err := h.service.ListarAdmin(c.Query("status"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao listar repasses"})
		return
	}
	itens := make([]repasseResposta, 0, len(lista))
	for i := range lista {
		itens = append(itens, paraRepasseResposta(&lista[i], true))
	}
	c.JSON(http.StatusOK, itens)
}

type pagarRepasseRequest struct {
	ComprovanteURL string `json:"comprovante_url"`
	Observacao     string `json:"observacao"`
}

// Pagar é POST /admin/repasses/:id/pagar.
func (h *RepasseHandler) Pagar(c *gin.Context) {
	id, ok := idDaURL(c)
	if !ok {
		return
	}
	var req pagarRepasseRequest
	_ = c.ShouldBindJSON(&req)

	rep, err := h.service.MarcarPago(id, req.ComprovanteURL, req.Observacao)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrRepasseNaoEncontrado):
			c.JSON(http.StatusNotFound, gin.H{"erro": err.Error()})
		case errors.Is(err, service.ErrRepasseNaoPendente):
			c.JSON(http.StatusConflict, gin.H{"erro": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao marcar repasse como pago"})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": rep.ID, "status": rep.Status, "pago_em": rep.PagoEm})
}

// GerarRepasses é o job POST /jobs/gerar-repasses.
func (h *RepasseHandler) GerarRepasses(c *gin.Context) {
	n, err := h.service.GerarRepasses()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao gerar repasses"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"criados": n})
}
