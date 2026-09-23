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

type ContaHandler struct {
	service *service.ContaService
}

func NovoContaHandler(s *service.ContaService) *ContaHandler {
	return &ContaHandler{service: s}
}

type meuEventoResposta struct {
	EventoID int64      `json:"evento_id"`
	Titulo   string     `json:"titulo"`
	Slug     string     `json:"slug"`
	InicioEm *time.Time `json:"inicio_em"`
	Selos    []string   `json:"selos"`
}

// MeusEventos é GET /me/eventos — cada evento em que a pessoa tem algum
// papel, com selo (seção 3 do plano).
func (h *ContaHandler) MeusEventos(c *gin.Context) {
	usuarioID := c.GetInt64(middleware.ChaveContextoUsuarioID)

	meusEventos, err := h.service.MeusEventos(usuarioID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao listar seus eventos"})
		return
	}

	resposta := make([]meuEventoResposta, 0, len(meusEventos))
	for _, me := range meusEventos {
		selos := make([]string, 0, len(me.Selos))
		for _, s := range me.Selos {
			selos = append(selos, string(s))
		}
		resposta = append(resposta, meuEventoResposta{
			EventoID: me.Evento.ID, Titulo: me.Evento.Titulo, Slug: me.Evento.Slug,
			InicioEm: me.Evento.InicioEm, Selos: selos,
		})
	}
	c.JSON(http.StatusOK, resposta)
}

type meuIngressoResposta struct {
	itemPedidoResposta
	EventoTitulo   string     `json:"evento_titulo"`
	EventoSlug     string     `json:"evento_slug"`
	EventoInicioEm *time.Time `json:"evento_inicio_em"`
}

func paraMeuIngressoResposta(i *repository.ItemComEvento) meuIngressoResposta {
	return meuIngressoResposta{
		itemPedidoResposta: paraItemPedidoResposta(&i.ItemPedido),
		EventoTitulo:       i.EventoTitulo,
		EventoSlug:         i.EventoSlug,
		EventoInicioEm:     i.EventoInicioEm,
	}
}

// MeusIngressos é GET /me/ingressos (seção 8).
func (h *ContaHandler) MeusIngressos(c *gin.Context) {
	usuarioID := c.GetInt64(middleware.ChaveContextoUsuarioID)

	itens, err := h.service.MeusIngressos(usuarioID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao listar seus ingressos"})
		return
	}

	resposta := make([]meuIngressoResposta, 0, len(itens))
	for i := range itens {
		resposta = append(resposta, paraMeuIngressoResposta(&itens[i]))
	}
	c.JSON(http.StatusOK, resposta)
}

// MeuIngresso é GET /me/ingressos/:id — detalhe com o QR (seção 8).
func (h *ContaHandler) MeuIngresso(c *gin.Context) {
	itemID, ok := idDaURL(c)
	if !ok {
		return
	}
	usuarioID := c.GetInt64(middleware.ChaveContextoUsuarioID)

	item, err := h.service.MeuIngresso(usuarioID, itemID)
	if err != nil {
		if errors.Is(err, service.ErrItemNaoPertenceAoUsuario) {
			c.JSON(http.StatusForbidden, gin.H{"erro": "ingresso não pertence a este usuário"})
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"erro": "ingresso não encontrado"})
		return
	}
	c.JSON(http.StatusOK, paraItemPedidoResposta(item))
}
