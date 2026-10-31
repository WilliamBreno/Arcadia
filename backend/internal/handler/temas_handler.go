package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
	"github.com/WilliamBreno/Arcadia/backend/internal/service"
)

// Item 11.7: o organizador marca o evento em temas da época (Halloween,
// Réveillon...) abertos pelo admin. Até 3 por evento.
const maxTemasPorEvento = 3

type TemasHandler struct {
	organizadorHandler *OrganizadorHandler
	eventosSvc         *service.EventoService
	eventos            *repository.EventoRepository
}

func NovoTemasHandler(o *OrganizadorHandler, svc *service.EventoService, eventos *repository.EventoRepository) *TemasHandler {
	return &TemasHandler{organizadorHandler: o, eventosSvc: svc, eventos: eventos}
}

type temaResposta struct {
	ID         int64   `json:"id"`
	Slug       string  `json:"slug"`
	Titulo     string  `json:"titulo"`
	Icone      string  `json:"icone"`
	ExibirDe   *string `json:"exibir_de"`
	ExibirAte  *string `json:"exibir_ate"`
	Marcado    bool    `json:"marcado"`
	PelaEquipe bool    `json:"pela_equipe"` // o admin incluiu: fica marcado e o organizador não tira
	Bloqueado  bool    `json:"bloqueado"`   // o admin tirou o evento deste tema
	Aberto     bool    `json:"aberto"`      // ainda aceita marcação
}

func (h *TemasHandler) evento(c *gin.Context) (int64, bool) {
	organizador, ok := h.organizadorHandler.ObterOrganizadorAtual(c)
	if !ok {
		return 0, false
	}
	eventoID, ok := idDaURL(c)
	if !ok {
		return 0, false
	}
	if _, err := h.eventosSvc.BuscarDoOrganizador(organizador.ID, eventoID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"erro": "evento não encontrado"})
		return 0, false
	}
	return eventoID, true
}

func (h *TemasHandler) listar(eventoID int64) ([]temaResposta, error) {
	cs, err := h.eventos.ListarColecoes(true)
	if err != nil {
		return nil, err
	}
	marcados, err := h.eventos.TemasDoEvento(eventoID)
	if err != nil {
		return nil, err
	}
	agora := time.Now()
	lista := []temaResposta{}
	for i := range cs {
		col := &cs[i]
		m, presente := marcados[col.ID]
		aberto := col.AceitaMarcacaoEm(agora)
		if !aberto && !presente {
			continue
		}
		if !col.AbertaOrganizador && m.Origem != "organizador" {
			continue // coleção montada só pelo admin: não é tema
		}
		lista = append(lista, temaResposta{ID: col.ID, Slug: col.Slug, Titulo: col.Titulo, Icone: col.Icone,
			ExibirDe: dataColecao(col.ExibirDe), ExibirAte: dataColecao(col.ExibirAte),
			Marcado: presente && !m.Bloqueado, PelaEquipe: presente && m.Origem == "admin" && !m.Bloqueado,
			Bloqueado: presente && m.Bloqueado, Aberto: aberto})
	}
	return lista, nil
}

// Listar é GET /org/eventos/:id/temas.
func (h *TemasHandler) Listar(c *gin.Context) {
	eventoID, ok := h.evento(c)
	if !ok {
		return
	}
	lista, err := h.listar(eventoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro interno"})
		return
	}
	c.JSON(http.StatusOK, lista)
}

// Definir é PUT /org/eventos/:id/temas {colecao_ids: [...]}.
func (h *TemasHandler) Definir(c *gin.Context) {
	eventoID, ok := h.evento(c)
	if !ok {
		return
	}
	var req struct {
		ColecaoIDs []int64 `json:"colecao_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "dados inválidos"})
		return
	}
	ids := []int64{}
	vistos := map[int64]bool{}
	for _, id := range req.ColecaoIDs {
		if !vistos[id] {
			vistos[id] = true
			ids = append(ids, id)
		}
	}
	atuais, err := h.listar(eventoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro interno"})
		return
	}
	porID := map[int64]temaResposta{}
	for _, t := range atuais {
		porID[t.ID] = t
	}
	// Os incluídos pela equipe não contam no limite nem são regravados.
	doOrganizador := []int64{}
	for _, id := range ids {
		if !porID[id].PelaEquipe {
			doOrganizador = append(doOrganizador, id)
		}
	}
	ids = doOrganizador
	if len(ids) > maxTemasPorEvento {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"erro": "escolha até 3 temas"})
		return
	}
	for _, id := range ids {
		t, existe := porID[id]
		switch {
		case !existe:
			c.JSON(http.StatusUnprocessableEntity, gin.H{"erro": "tema indisponível"})
			return
		case t.Bloqueado:
			c.JSON(http.StatusUnprocessableEntity, gin.H{"erro": "a equipe da Evve tirou este evento do tema " + t.Titulo})
			return
		case !t.Aberto && !t.Marcado:
			c.JSON(http.StatusUnprocessableEntity, gin.H{"erro": "o tema " + t.Titulo + " já terminou"})
			return
		}
	}
	if err := h.eventos.DefinirTemasDoEvento(eventoID, ids); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao salvar"})
		return
	}
	lista, err := h.listar(eventoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro interno"})
		return
	}
	c.JSON(http.StatusOK, lista)
}
