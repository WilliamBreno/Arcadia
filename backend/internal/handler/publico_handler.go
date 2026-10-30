package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
	"github.com/WilliamBreno/Arcadia/backend/internal/service"
)

type PublicoHandler struct {
	eventos       *repository.EventoRepository
	tiposIngresso *repository.TipoIngressoRepository
	locais        *repository.LocalRepository
	organizadores *repository.OrganizadorRepository
	config        *repository.ConfigPlataformaRepository
	itensPedido   *repository.ItemPedidoRepository
	cronograma    *repository.CronogramaRepository
	sessoes       *repository.SessaoRepository
	taxas         *service.TaxasService
	seguir        *service.SeguirService
	avaliacoes    *service.AvaliacaoPublicoService
	perguntasDB   *gorm.DB // item 7.6
}

// DefinirTaxas liga o cálculo de taxas por faixa (item 5.1).
func (h *PublicoHandler) DefinirTaxas(t *service.TaxasService) { h.taxas = t }

// regraTaxasPublica vai para a página do evento: o front mostra a taxa
// de cada ingresso (e com cupom) com a mesma regra do servidor — só para
// exibição; o valor cobrado é sempre recalculado no checkout.
type regraTaxasPublica struct {
	Faixas               []service.FaixaTaxa `json:"faixas"`
	GarantiaPercentualBP int64               `json:"garantia_percentual_bp"`
	GarantiaMinCentavos  int64               `json:"garantia_min_centavos"`
	GarantiaMaxCentavos  int64               `json:"garantia_max_centavos"`
	LimiteBP             *int64              `json:"limite_bp"`
	TaxaPagaPor          string              `json:"taxa_paga_por"`
}

func NovoPublicoHandler(
	eventos *repository.EventoRepository,
	tiposIngresso *repository.TipoIngressoRepository,
	locais *repository.LocalRepository,
	organizadores *repository.OrganizadorRepository,
	config *repository.ConfigPlataformaRepository,
	itensPedido *repository.ItemPedidoRepository,
	cronograma *repository.CronogramaRepository,
	sessoes *repository.SessaoRepository,
) *PublicoHandler {
	return &PublicoHandler{sessoes: sessoes, cronograma: cronograma, eventos: eventos, tiposIngresso: tiposIngresso, locais: locais, organizadores: organizadores, config: config, itensPedido: itensPedido}
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
	// Preço final do ingresso mais barato, já com a taxa que o comprador paga
	// (item 6.3: "preço final em destaque"). Igual ao preço quando o
	// organizador absorve a taxa ou o ingresso é gratuito.
	PrecoFinalAPartirCentavos *int64 `json:"preco_final_a_partir_centavos,omitempty"`
	// Coordenadas do local (mapa "perto de mim", item 6.2).
	Latitude  *float64 `json:"latitude,omitempty"`
	Longitude *float64 `json:"longitude,omitempty"`
}

// fusoPadrao: fuso dos atalhos de data (o padrão dos eventos, seção 5).
var fusoPadrao = func() *time.Location {
	if l, err := time.LoadLocation("America/Maceio"); err == nil {
		return l
	}
	return time.FixedZone("BRT", -3*3600)
}()

// atalhoParaIntervalo converte hoje|amanha|fim-de-semana num intervalo, no
// fuso de Brasília/Maceió (item 6.2; antes usava UTC e "hoje" virava às 21h).
// "fim-de-semana" é de sábado 00h a segunda 00h; no próprio domingo vale o
// fim de semana corrente (antes pulava para o sábado seguinte).
func atalhoParaIntervalo(atalho string) (de, ate *time.Time) {
	return intervaloDoAtalho(atalho, time.Now())
}

func intervaloDoAtalho(atalho string, agora time.Time) (de, ate *time.Time) {
	local := agora.In(fusoPadrao)
	inicioHoje := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, fusoPadrao)

	switch atalho {
	case "hoje":
		fim := inicioHoje.AddDate(0, 0, 1)
		return &inicioHoje, &fim
	case "amanha":
		inicio := inicioHoje.AddDate(0, 0, 1)
		fim := inicio.AddDate(0, 0, 1)
		return &inicio, &fim
	case "fim-de-semana":
		var sabado time.Time
		if local.Weekday() == time.Sunday {
			sabado = inicioHoje.AddDate(0, 0, -1)
		} else {
			sabado = inicioHoje.AddDate(0, 0, int(time.Saturday-local.Weekday()))
		}
		segunda := sabado.AddDate(0, 0, 2)
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
	// Item 6.2: a listagem pública mostra só o que ainda não terminou.
	filtros.SomenteFuturos = c.Query("passados") != "true"
	if lat, errLat := strconv.ParseFloat(c.Query("lat"), 64); errLat == nil {
		if lng, errLng := strconv.ParseFloat(c.Query("lng"), 64); errLng == nil && lat >= -90 && lat <= 90 && lng >= -180 && lng <= 180 {
			filtros.Lat, filtros.Lng = &lat, &lng
			filtros.RaioKm, _ = strconv.ParseFloat(c.Query("raio_km"), 64)
		}
	}
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
	var regra *service.RegraTaxas
	if h.taxas != nil {
		if r, err := h.taxas.Regra(); err == nil {
			regra = &r
		}
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
				item.Latitude, item.Longitude = local.Latitude, local.Longitude
			}
		}
		if preco, ok := precos[e.ID]; ok {
			item.PrecoAPartirCentavos = &preco
			item.Gratuito = preco == 0
			if regra != nil {
				final := h.taxas.ParaEventoComRegra(*regra, e).Calcular(preco, false).TotalCentavos
				item.PrecoFinalAPartirCentavos = &final
			}
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
	ID          int64  `json:"id"` // para seguir (item 6.4)
	NomePublico string `json:"nome_publico"`
	Slug        string `json:"slug"`
	Descricao   string `json:"descricao"`
	LogoURL     string `json:"logo_url"`
	Instagram   string `json:"instagram"`
	Site        string `json:"site"`
	Verificado  bool   `json:"verificado"` // selo "Verificado" (item 5.4)
	Pro         bool   `json:"pro"`        // selo "Pro" (item 10.4)
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
	ids := make([]int64, len(tipos))
	for i := range tipos {
		ids[i] = tipos[i].ID
	}
	vendidos, err := h.itensPedido.ContarAtivosPorTipos(nil, ids)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao carregar ingressos"})
		return
	}
	tipos = service.FiltrarLotes(tipos, vendidos, time.Now())
	guardadas, _ := service.VagasGuardadasPorTipos(h.itensPedido.DB(), ids)
	revendas, _ := service.RevendasAVendaPorTipos(h.itensPedido.DB(), ids)
	tiposResposta := make([]tipoIngressoResposta, 0, len(tipos))
	for i := range tipos {
		r := paraTipoIngressoResposta(&tipos[i])
		esgotado := service.VagasParaComprar(int64(tipos[i].Quantidade)-vendidos[tipos[i].ID], revendas[tipos[i].ID], guardadas[tipos[i].ID]) <= 0
		r.Esgotado = &esgotado
		tiposResposta = append(tiposResposta, r)
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
			ID: organizador.ID, NomePublico: organizador.NomePublico, Slug: organizador.Slug,
			Descricao: organizador.Descricao, LogoURL: organizador.LogoURL,
			Instagram: organizador.Instagram, Site: organizador.Site,
			Verificado: organizador.VerificadoEm != nil, Pro: h.perguntasDB != nil && service.EhPro(h.perguntasDB, organizador.ID),
		}
	}

	taxasEvento, err := h.taxas.ParaEvento(evento)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao calcular taxas"})
		return
	}
	taxaPagaPor := domain.TaxaPagaPorComprador
	if taxasEvento.Absorvida {
		taxaPagaPor = domain.TaxaPagaPorOrganizador
	}
	regraTaxas := regraTaxasPublica{
		Faixas:               taxasEvento.Regra.Faixas,
		GarantiaPercentualBP: taxasEvento.Regra.GarantiaPercentualBP,
		GarantiaMinCentavos:  taxasEvento.Regra.GarantiaMinCentavos,
		GarantiaMaxCentavos:  taxasEvento.Regra.GarantiaMaxCentavos,
		LimiteBP:             taxasEvento.LimiteBP,
		TaxaPagaPor:          taxaPagaPor,
	}

	itensCronograma, _ := h.cronograma.ListarPorEvento(evento.ID)
	cronogramaResp := make([]cronogramaResposta, 0, len(itensCronograma))
	for i := range itensCronograma {
		cronogramaResp = append(cronogramaResp, paraCronogramaResposta(&itensCronograma[i]))
	}

	listaSessoes, _ := h.sessoes.ListarPorEvento(evento.ID)
	sessoesResp := make([]sessaoResposta, 0, len(listaSessoes))
	for i := range listaSessoes {
		sessoesResp = append(sessoesResp, paraSessaoResposta(&listaSessoes[i]))
	}

	c.JSON(http.StatusOK, gin.H{
		"sessoes":           sessoesResp,
		"cronograma":        cronogramaResp,
		"evento":            paraEventoResposta(evento),
		"local":             localResposta,
		"organizador":       organizadorResposta,
		"ingressos":         tiposResposta,
		"taxas":             regraTaxas,
		"vendas_encerradas": vendasEncerradas(evento, tipos, listaSessoes),
		"pixels":            service.Pixels{Meta: evento.PixelMeta, GA4: evento.PixelGA4, TikTok: evento.PixelTikTok},
		"perguntas":         h.perguntasDoEvento(evento.ID),
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
		"seguidores": h.contarSeguidores(organizador.ID),
		"avaliacoes": h.resumoOrganizador(organizador.ID),
		"organizador": organizadorPublico{
			ID: organizador.ID, NomePublico: organizador.NomePublico, Slug: organizador.Slug,
			Descricao: organizador.Descricao, LogoURL: organizador.LogoURL,
			Instagram: organizador.Instagram, Site: organizador.Site,
			Verificado: organizador.VerificadoEm != nil, Pro: h.perguntasDB != nil && service.EhPro(h.perguntasDB, organizador.ID),
		},
		"eventos": eventosDoOrganizador,
	})
}

// vendasEncerradas: nenhum tipo de ingresso pode mais ser vendido (o evento,
// ou todos os dias dele, já terminou — item 5.5).
func vendasEncerradas(evento *domain.Evento, tipos []domain.TipoIngresso, sessoes []domain.Sessao) bool {
	agora := time.Now()
	for i := range tipos {
		if service.VendaAberta(evento, tipos[i].SessaoIDs, sessoes, agora) {
			return false
		}
	}
	return len(tipos) > 0 || !service.VendaAberta(evento, nil, sessoes, agora)
}

// Taxas é GET /api/v1/taxas: a tabela pública de taxas (página "Quanto
// custa", item 6.3). Sem limites estaduais: esses aparecem no evento.
func (h *PublicoHandler) Taxas(c *gin.Context) {
	regra, err := h.taxas.Regra()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao ler as taxas"})
		return
	}
	c.Header("Cache-Control", "public, max-age=600")
	c.JSON(http.StatusOK, gin.H{
		"faixas":                 regra.Faixas,
		"garantia_percentual_bp": regra.GarantiaPercentualBP,
		"garantia_min_centavos":  regra.GarantiaMinCentavos,
		"garantia_max_centavos":  regra.GarantiaMaxCentavos,
	})
}

// DefinirAvaliacoes liga a nota média do público na página do organizador (6.5).
func (h *PublicoHandler) DefinirAvaliacoes(s *service.AvaliacaoPublicoService) { h.avaliacoes = s }

func (h *PublicoHandler) resumoOrganizador(organizadorID int64) service.ResumoAvaliacoes {
	if h.avaliacoes == nil {
		return service.ResumoAvaliacoes{}
	}
	r, _ := h.avaliacoes.ResumoDoOrganizador(organizadorID)
	return r
}

// DefinirSeguir liga a contagem de seguidores na página do organizador.
func (h *PublicoHandler) DefinirSeguir(s *service.SeguirService) { h.seguir = s }

func (h *PublicoHandler) contarSeguidores(organizadorID int64) int64 {
	if h.seguir == nil {
		return 0
	}
	n, _ := h.seguir.ContarSeguidores(service.SeguirOrganizador, strconv.FormatInt(organizadorID, 10))
	return n
}

// DefinirPerguntas liga as perguntas do checkout na página pública (item 7.6).
func (h *PublicoHandler) DefinirPerguntas(db *gorm.DB) { h.perguntasDB = db }

func (h *PublicoHandler) perguntasDoEvento(eventoID int64) []service.Pergunta {
	if h.perguntasDB == nil {
		return []service.Pergunta{}
	}
	ps, err := service.PerguntasAtivas(h.perguntasDB, eventoID)
	if err != nil || ps == nil {
		return []service.Pergunta{}
	}
	return ps
}
