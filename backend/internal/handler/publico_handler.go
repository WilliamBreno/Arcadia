package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
)

type PublicoHandler struct {
	eventos       *repository.EventoRepository
	tiposIngresso *repository.TipoIngressoRepository
	locais        *repository.LocalRepository
	organizadores *repository.OrganizadorRepository
	config        *repository.ConfigPlataformaRepository
}

func NovoPublicoHandler(
	eventos *repository.EventoRepository,
	tiposIngresso *repository.TipoIngressoRepository,
	locais *repository.LocalRepository,
	organizadores *repository.OrganizadorRepository,
	config *repository.ConfigPlataformaRepository,
) *PublicoHandler {
	return &PublicoHandler{eventos: eventos, tiposIngresso: tiposIngresso, locais: locais, organizadores: organizadores, config: config}
}

type eventoPublicoItem struct {
	Slug                 string  `json:"slug"`
	Titulo               string  `json:"titulo"`
	CapaURL              string  `json:"capa_url"`
	Categoria            string  `json:"categoria"`
	Cidade               string  `json:"cidade,omitempty"`
	UF                   string  `json:"uf,omitempty"`
	InicioEm             *string `json:"inicio_em"`
	PrecoAPartirCentavos *int64  `json:"preco_a_partir_centavos"`
	Gratuito             bool    `json:"gratuito"`
}

// atalhoParaIntervalo converte hoje|amanha|fim-de-semana num intervalo de
// datas. Usa o horário do servidor (UTC) — simplificação registrada na
// seção 14; o ideal seria por fuso do evento/usuário.
func atalhoParaIntervalo(atalho string) (de, ate *time.Time) {
	agora := time.Now().UTC()
	inicioHoje := time.Date(agora.Year(), agora.Month(), agora.Day(), 0, 0, 0, 0, time.UTC)

	switch atalho {
	case "hoje":
		fim := inicioHoje.Add(24 * time.Hour)
		return &inicioHoje, &fim
	case "amanha":
		inicio := inicioHoje.Add(24 * time.Hour)
		fim := inicio.Add(24 * time.Hour)
		return &inicio, &fim
	case "fim-de-semana":
		diasAteSabado := (6 - int(agora.Weekday()) + 7) % 7
		sabado := inicioHoje.Add(time.Duration(diasAteSabado) * 24 * time.Hour)
		segunda := sabado.Add(2 * 24 * time.Hour)
		return &sabado, &segunda
	default:
		return nil, nil
	}
}

func parseDataQuery(valor string) *time.Time {
	if valor == "" {
		return nil
	}
	t, err := time.Parse("2006-01-02", valor)
	if err != nil {
		return nil
	}
	return &t
}

func (h *PublicoHandler) ListarEventos(c *gin.Context) {
	filtros := repository.FiltrosEventoPublico{
		Busca:     c.Query("q"),
		Cidade:    c.Query("cidade"),
		UF:        c.Query("uf"),
		Categoria: c.Query("categoria"),
		Ordenar:   c.DefaultQuery("ordenar", "data"),
	}
	filtros.DataDe = parseDataQuery(c.Query("data_de"))
	filtros.DataAte = parseDataQuery(c.Query("data_ate"))
	if atalho := c.Query("atalho"); atalho != "" {
		filtros.DataDe, filtros.DataAte = atalhoParaIntervalo(atalho)
	}
	filtros.SomenteGratuitos = c.Query("gratuito") == "true"
	filtros.Pagina, _ = strconv.Atoi(c.DefaultQuery("pagina", "1"))
	filtros.PorPagina, _ = strconv.Atoi(c.DefaultQuery("por_pagina", "20"))

	eventos, total, err := h.eventos.ListarPublicos(filtros)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao listar eventos"})
		return
	}

	itens, err := h.paraItensPublicos(eventos, true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao calcular preços"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"eventos": itens, "total": total, "pagina": filtros.Pagina, "por_pagina": filtros.PorPagina})
}

// paraItensPublicos monta os cards de evento (usado na listagem e na
// página do organizador), com preço mínimo calculado em lote (evita
// N+1) e, opcionalmente, cidade/UF do local.
func (h *PublicoHandler) paraItensPublicos(eventos []domain.Evento, comLocal bool) ([]eventoPublicoItem, error) {
	ids := make([]int64, len(eventos))
	for i, e := range eventos {
		ids[i] = e.ID
	}
	precos, err := h.tiposIngresso.PrecoMinimoPorEvento(ids)
	if err != nil {
		return nil, err
	}

	itens := make([]eventoPublicoItem, 0, len(eventos))
	for i := range eventos {
		e := &eventos[i]
		item := eventoPublicoItem{
			Slug:      e.Slug,
			Titulo:    e.Titulo,
			CapaURL:   e.CapaURL,
			Categoria: e.Categoria,
		}
		if e.InicioEm != nil {
			s := e.InicioEm.Format(time.RFC3339)
			item.InicioEm = &s
		}
		if comLocal && e.LocalID != nil {
			if local, err := h.locais.BuscarPorID(*e.LocalID); err == nil {
				item.Cidade = local.Cidade
				item.UF = local.UF
			}
		}
		if preco, ok := precos[e.ID]; ok {
			item.PrecoAPartirCentavos = &preco
			item.Gratuito = preco == 0
		}
		itens = append(itens, item)
	}
	return itens, nil
}

func (h *PublicoHandler) Categorias(c *gin.Context) {
	categorias, err := h.eventos.ListarCategorias()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao listar categorias"})
		return
	}
	c.JSON(http.StatusOK, categorias)
}

type localPublico struct {
	Nome       string   `json:"nome"`
	Logradouro string   `json:"logradouro"`
	Numero     string   `json:"numero"`
	Bairro     string   `json:"bairro"`
	Cidade     string   `json:"cidade"`
	UF         string   `json:"uf"`
	Latitude   *float64 `json:"latitude"`
	Longitude  *float64 `json:"longitude"`
}

type organizadorPublico struct {
	NomePublico string `json:"nome_publico"`
	Slug        string `json:"slug"`
	Descricao   string `json:"descricao"`
	LogoURL     string `json:"logo_url"`
	Instagram   string `json:"instagram"`
	Site        string `json:"site"`
}

func (h *PublicoHandler) ObterEvento(c *gin.Context) {
	evento, err := h.eventos.BuscarPublicoPorSlug(c.Param("slug"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"erro": "evento não encontrado"})
		return
	}

	tipos, err := h.tiposIngresso.ListarAtivosPublicoPorEvento(evento.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao carregar ingressos"})
		return
	}
	tiposResposta := make([]tipoIngressoResposta, 0, len(tipos))
	for i := range tipos {
		tiposResposta = append(tiposResposta, paraTipoIngressoResposta(&tipos[i]))
	}

	var localResposta *localPublico
	if evento.LocalID != nil {
		if local, err := h.locais.BuscarPorID(*evento.LocalID); err == nil {
			localResposta = &localPublico{
				Nome: local.Nome, Logradouro: local.Logradouro, Numero: local.Numero,
				Bairro: local.Bairro, Cidade: local.Cidade, UF: local.UF,
				Latitude: local.Latitude, Longitude: local.Longitude,
			}
		}
	}

	var organizadorResposta *organizadorPublico
	if organizador, err := h.organizadores.BuscarPorID(evento.OrganizadorID); err == nil {
		organizadorResposta = &organizadorPublico{
			NomePublico: organizador.NomePublico, Slug: organizador.Slug,
			Descricao: organizador.Descricao, LogoURL: organizador.LogoURL,
			Instagram: organizador.Instagram, Site: organizador.Site,
		}
	}

	chaveTaxa := domain.ChaveTaxaIngressoCentavos
	if evento.TipoAcesso == domain.TipoAcessoCadastro {
		chaveTaxa = domain.ChaveTaxaCadastroCentavos
	}
	taxaPlataforma, _ := h.config.BuscarInt64(chaveTaxa)
	garantiaCentavos, _ := h.config.BuscarInt64(domain.ChaveGarantiaCentavos)

	c.JSON(http.StatusOK, gin.H{
		"evento":                   paraEventoResposta(evento),
		"local":                    localResposta,
		"organizador":              organizadorResposta,
		"ingressos":                tiposResposta,
		"taxa_plataforma_centavos": taxaPlataforma,
		"garantia_centavos":        garantiaCentavos,
	})
}

func (h *PublicoHandler) ObterOrganizador(c *gin.Context) {
	organizador, err := h.organizadores.BuscarPorSlug(c.Param("slug"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"erro": "organizador não encontrado"})
		return
	}

	eventos, _, err := h.eventos.ListarPublicos(repository.FiltrosEventoPublico{OrganizadorID: &organizador.ID, PorPagina: 100})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao listar eventos"})
		return
	}

	eventosDoOrganizador, err := h.paraItensPublicos(eventos, false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao calcular preços"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"organizador": organizadorPublico{
			NomePublico: organizador.NomePublico, Slug: organizador.Slug,
			Descricao: organizador.Descricao, LogoURL: organizador.LogoURL,
			Instagram: organizador.Instagram, Site: organizador.Site,
		},
		"eventos": eventosDoOrganizador,
	})
}
