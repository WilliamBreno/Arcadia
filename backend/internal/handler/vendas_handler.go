package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
	"github.com/WilliamBreno/Arcadia/backend/internal/service"
)

type VendasHandler struct {
	organizadorHandler *OrganizadorHandler
	service            *service.VendasService
}

func NovoVendasHandler(organizadorHandler *OrganizadorHandler, s *service.VendasService) *VendasHandler {
	return &VendasHandler{organizadorHandler: organizadorHandler, service: s}
}

type vendaItemResposta struct {
	ID               int64      `json:"id"`
	TitularNome      string     `json:"titular_nome"`
	TitularEmail     string     `json:"titular_email"`
	TipoIngressoNome string     `json:"tipo_ingresso_nome"`
	PrecoCentavos    int64      `json:"preco_centavos"`
	Status           string     `json:"status"`
	Codigo           string     `json:"codigo"`
	CriadoEm         time.Time  `json:"criado_em"`
	UtilizadoEm      *time.Time `json:"utilizado_em"`
}

type resumoVendasTipoResposta struct {
	TipoIngressoID   int64  `json:"tipo_ingresso_id"`
	TipoIngressoNome string `json:"tipo_ingresso_nome"`
	Quantidade       int    `json:"quantidade"`
	ReceitaCentavos  int64  `json:"receita_centavos"`
}

type vendasResposta struct {
	TotalVendido    int                        `json:"total_vendido"`
	ReceitaCentavos int64                      `json:"receita_centavos"`
	PorTipo         []resumoVendasTipoResposta `json:"por_tipo"`
	Itens           []vendaItemResposta        `json:"itens"`
}

// Listar é GET /org/eventos/:id/vendas — painel básico da seção 1.11:
// quantidade vendida, receita bruta e lista de compradores.
func (h *VendasHandler) Listar(c *gin.Context) {
	organizador, ok := h.organizadorHandler.ObterOrganizadorAtual(c)
	if !ok {
		return
	}
	eventoID, ok := idDaURL(c)
	if !ok {
		return
	}

	resultado, err := h.service.Listar(organizador.ID, eventoID)
	if err != nil {
		if errors.Is(err, service.ErrEventoNaoPertenceAoOrganizador) {
			c.JSON(http.StatusForbidden, gin.H{"erro": "evento não pertence a este organizador"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao carregar vendas"})
		return
	}

	porTipo := make([]resumoVendasTipoResposta, 0, len(resultado.Resumo.PorTipo))
	for _, t := range resultado.Resumo.PorTipo {
		porTipo = append(porTipo, resumoVendasTipoResposta{
			TipoIngressoID: t.TipoIngressoID, TipoIngressoNome: t.TipoIngressoNome,
			Quantidade: t.Quantidade, ReceitaCentavos: t.ReceitaCentavos,
		})
	}

	itens := make([]vendaItemResposta, 0, len(resultado.Itens))
	for _, i := range resultado.Itens {
		itens = append(itens, paraVendaItemResposta(&i))
	}

	c.JSON(http.StatusOK, vendasResposta{
		TotalVendido: resultado.Resumo.TotalVendido, ReceitaCentavos: resultado.Resumo.ReceitaCentavos,
		PorTipo: porTipo, Itens: itens,
	})
}

func paraVendaItemResposta(i *repository.ItemVenda) vendaItemResposta {
	return vendaItemResposta{
		ID: i.ID, TitularNome: i.TitularNome, TitularEmail: i.TitularEmail, TipoIngressoNome: i.TipoIngressoNome,
		PrecoCentavos: i.PrecoCentavos, Status: string(i.Status), Codigo: i.Codigo,
		CriadoEm: i.CriadoEm, UtilizadoEm: i.UtilizadoEm,
	}
}
