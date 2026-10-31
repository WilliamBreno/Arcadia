package handler

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
	"github.com/WilliamBreno/Arcadia/backend/internal/service"
)

// Descoberta (item 6.2): páginas por cidade e categoria, "Em alta",
// "Quase esgotando", "Novos" e coleções curadas pelo admin.

// slugCidade: "Maceió" + "AL" -> "maceio-al". Testado.
func slugCidade(cidade, uf string) string {
	return service.Slugify(cidade) + "-" + strings.ToLower(uf)
}

type cidadeResposta struct {
	Cidade  string `json:"cidade"`
	UF      string `json:"uf"`
	Slug    string `json:"slug"`
	Eventos int64  `json:"eventos"`
}

type categoriaResposta struct {
	Nome    string `json:"nome"`
	Slug    string `json:"slug"`
	Eventos int64  `json:"eventos"`
}

// resolverCidade transforma o slug da URL em cidade/UF a partir das cidades
// que têm eventos (assim a URL nunca aponta para uma cidade vazia).
func resolverCidade(cidades []repository.CidadeContagem, slug string) *cidadeResposta {
	for _, c := range cidades {
		if slugCidade(c.Cidade, c.UF) == slug {
			return &cidadeResposta{Cidade: c.Cidade, UF: c.UF, Slug: slug, Eventos: c.Eventos}
		}
	}
	return nil
}

// Descoberta é GET /api/v1/descoberta?cidade=<slug>: cidades com eventos e
// categorias (da cidade, se informada). Base dos menus e das URLs.
func (h *PublicoHandler) Descoberta(c *gin.Context) {
	cidades, err := h.eventos.ListarCidadesComEventos()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao listar cidades"})
		return
	}
	var atual *cidadeResposta
	if slug := c.Query("cidade"); slug != "" {
		if atual = resolverCidade(cidades, slug); atual == nil {
			c.JSON(http.StatusNotFound, gin.H{"erro": "nenhum evento nesta cidade"})
			return
		}
	}
	cidade, uf := "", ""
	if atual != nil {
		cidade, uf = atual.Cidade, atual.UF
	}
	categorias, err := h.eventos.ListarCategoriasComEventos(cidade, uf)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao listar categorias"})
		return
	}

	respCidades := make([]cidadeResposta, 0, len(cidades))
	for _, ci := range cidades {
		respCidades = append(respCidades, cidadeResposta{Cidade: ci.Cidade, UF: ci.UF, Slug: slugCidade(ci.Cidade, ci.UF), Eventos: ci.Eventos})
	}
	respCategorias := make([]categoriaResposta, 0, len(categorias))
	for _, ca := range categorias {
		respCategorias = append(respCategorias, categoriaResposta{Nome: ca.Categoria, Slug: service.Slugify(ca.Categoria), Eventos: ca.Eventos})
	}
	c.Header("Cache-Control", "public, max-age=300")
	c.JSON(http.StatusOK, gin.H{"cidades": respCidades, "categorias": respCategorias, "cidade": atual})
}

// Destaques é GET /api/v1/destaques?cidade=<slug>: "Em alta" (mais vendidos
// nos últimos 7 dias), "Quase esgotando" e "Novos" (publicados na semana).
func (h *PublicoHandler) Destaques(c *gin.Context) {
	cidade, uf := "", ""
	if slug := c.Query("cidade"); slug != "" {
		cidades, err := h.eventos.ListarCidadesComEventos()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao listar cidades"})
			return
		}
		if atual := resolverCidade(cidades, slug); atual != nil {
			cidade, uf = atual.Cidade, atual.UF
		}
	}
	semana := time.Now().Add(-7 * 24 * time.Hour)
	const limite = 8

	listas := map[string][]domain.Evento{}
	var err error
	if listas["em_alta"], err = h.eventos.EmAlta(cidade, uf, semana, limite); err == nil {
		if listas["quase_esgotando"], err = h.eventos.QuaseEsgotando(cidade, uf, limite); err == nil {
			listas["novos"], err = h.eventos.Novos(cidade, uf, semana, limite)
		}
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao calcular destaques"})
		return
	}
	resp := gin.H{}
	for nome, eventos := range listas {
		itens, err := h.paraItensPublicos(eventos, true)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao calcular preços"})
			return
		}
		resp[nome] = itens
	}
	c.Header("Cache-Control", "public, max-age=120")
	c.JSON(http.StatusOK, resp)
}

type colecaoPublica struct {
	Slug      string              `json:"slug"`
	Titulo    string              `json:"titulo"`
	Descricao string              `json:"descricao"`
	Icone     string              `json:"icone"` // item 11.7
	Eventos   []eventoPublicoItem `json:"eventos"`
}

type colecaoGrade struct {
	Slug   string `json:"slug"`
	Titulo string `json:"titulo"`
	Icone  string `json:"icone"`
}

// Colecoes é GET /api/v1/colecoes: coleções ativas, no período e com ao
// menos um evento. Sem parâmetro, só as que vão para a página inicial (com
// os eventos); com ?grade=1, todas, só com título e ícone (item 11.7).
func (h *PublicoHandler) Colecoes(c *gin.Context) {
	cs, err := h.eventos.ListarColecoes(true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao listar coleções"})
		return
	}
	agora := time.Now()
	if c.Query("grade") == "1" {
		grade := make([]colecaoGrade, 0, len(cs))
		for i := range cs {
			col := &cs[i]
			if !col.VisivelEm(agora) {
				continue
			}
			if eventos, err := h.eventos.EventosDaColecao(col, 1); err != nil || len(eventos) == 0 {
				continue
			}
			grade = append(grade, colecaoGrade{Slug: col.Slug, Titulo: col.Titulo, Icone: col.Icone})
		}
		c.Header("Cache-Control", "public, max-age=300")
		c.JSON(http.StatusOK, grade)
		return
	}
	resp := make([]colecaoPublica, 0, len(cs))
	for i := range cs {
		col := &cs[i]
		if !col.NaHome || !col.VisivelEm(agora) {
			continue
		}
		eventos, err := h.eventos.EventosDaColecao(col, 8)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao listar coleções"})
			return
		}
		if len(eventos) == 0 {
			continue
		}
		itens, err := h.paraItensPublicos(eventos, true)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao calcular preços"})
			return
		}
		resp = append(resp, colecaoPublica{Slug: col.Slug, Titulo: col.Titulo, Descricao: col.Descricao, Icone: col.Icone, Eventos: itens})
	}
	c.Header("Cache-Control", "public, max-age=300")
	c.JSON(http.StatusOK, resp)
}

// Colecao é GET /api/v1/colecoes/:slug.
func (h *PublicoHandler) Colecao(c *gin.Context) {
	col, err := h.eventos.BuscarColecao(c.Param("slug"))
	if err != nil || !col.VisivelEm(time.Now()) {
		c.JSON(http.StatusNotFound, gin.H{"erro": "coleção não encontrada"})
		return
	}
	eventos, err := h.eventos.EventosDaColecao(col, 100)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao listar a coleção"})
		return
	}
	itens, err := h.paraItensPublicos(eventos, true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao calcular preços"})
		return
	}
	c.JSON(http.StatusOK, colecaoPublica{Slug: col.Slug, Titulo: col.Titulo, Descricao: col.Descricao, Icone: col.Icone, Eventos: itens})
}

// --- Admin das coleções ---

type colecaoAdmin struct {
	ID        int64    `json:"id"`
	Slug      string   `json:"slug"`
	Titulo    string   `json:"titulo"`
	Descricao string   `json:"descricao"`
	Ordem     int      `json:"ordem"`
	Ativa     bool     `json:"ativa"`
	Eventos   []string `json:"eventos"` // slugs, na ordem
	// Item 11.7.
	Icone             string                              `json:"icone"`
	ExibirDe          *string                             `json:"exibir_de"`
	ExibirAte         *string                             `json:"exibir_ate"`
	Regra             string                              `json:"regra"`
	RegraValor        string                              `json:"regra_valor"`
	AbertaOrganizador bool                                `json:"aberta_organizador"`
	NaHome            bool                                `json:"na_home"`
	VisivelAgora      bool                                `json:"visivel_agora"`
	Marcados          []repository.MarcadoPeloOrganizador `json:"marcados_organizador"`
}

type colecaoRequest struct {
	Titulo    string   `json:"titulo" binding:"required,min=2,max=120"`
	Descricao string   `json:"descricao" binding:"max=500"`
	Ordem     int      `json:"ordem"`
	Ativa     *bool    `json:"ativa"`
	Eventos   []string `json:"eventos"`
	// Item 11.7 (datas em AAAA-MM-DD; vazio = sem limite). Campo ausente
	// mantém o valor atual (na criação, o padrão).
	Icone             *string `json:"icone"`
	ExibirDe          *string `json:"exibir_de"`
	ExibirAte         *string `json:"exibir_ate"`
	Regra             *string `json:"regra"`
	RegraValor        *string `json:"regra_valor"`
	AbertaOrganizador *bool   `json:"aberta_organizador"`
	NaHome            *bool   `json:"na_home"`
}

func dataColecao(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.UTC().Format("2006-01-02")
	return &s
}

func lerDataColecao(s string) (*time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// ListarColecoesAdmin é GET /api/v1/admin/colecoes.
func (h *PublicoHandler) ListarColecoesAdmin(c *gin.Context) {
	cs, err := h.eventos.ListarColecoes(false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao listar coleções"})
		return
	}
	resp := make([]colecaoAdmin, 0, len(cs))
	agora := time.Now()
	for i := range cs {
		col := &cs[i]
		slugs, err := h.eventos.SlugsDaColecao(col.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao listar coleções"})
			return
		}
		if slugs == nil {
			slugs = []string{}
		}
		marcados, err := h.eventos.MarcadosPeloOrganizador(col.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao listar coleções"})
			return
		}
		resp = append(resp, colecaoAdmin{ID: col.ID, Slug: col.Slug, Titulo: col.Titulo, Descricao: col.Descricao, Ordem: col.Ordem, Ativa: col.Ativa, Eventos: slugs,
			Icone: col.Icone, ExibirDe: dataColecao(col.ExibirDe), ExibirAte: dataColecao(col.ExibirAte), Regra: col.Regra, RegraValor: col.RegraValor,
			AbertaOrganizador: col.AbertaOrganizador, NaHome: col.NaHome, VisivelAgora: col.VisivelEm(agora), Marcados: marcados})
	}
	c.JSON(http.StatusOK, resp)
}

// SalvarColecao é POST /api/v1/admin/colecoes e PUT /api/v1/admin/colecoes/:id.
// Eventos são informados por slug (o que o admin copia da URL do evento);
// slugs que não existem são devolvidos em "ignorados".
func (h *PublicoHandler) SalvarColecao(c *gin.Context) {
	var req colecaoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "informe um título (2 a 120 caracteres)"})
		return
	}
	col := &domain.Colecao{Ativa: true, NaHome: true, Icone: "estrela", Regra: domain.RegraColecaoManual, CriadoEm: time.Now()}
	if c.Param("id") != "" {
		id, ok := idDaURL(c)
		if !ok {
			return
		}
		existente, err := h.eventos.BuscarColecaoPorID(id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"erro": "coleção não encontrada"})
			return
		}
		col = existente
	}
	col.Titulo, col.Descricao, col.Ordem = strings.TrimSpace(req.Titulo), strings.TrimSpace(req.Descricao), req.Ordem
	if req.Ativa != nil {
		col.Ativa = *req.Ativa
	}
	// Item 11.7: ícone, período e regra (ausente = mantém).
	if req.Icone != nil {
		col.Icone = *req.Icone
	}
	if req.Regra != nil {
		col.Regra = *req.Regra
	}
	if req.RegraValor != nil {
		col.RegraValor = strings.TrimSpace(*req.RegraValor)
	}
	if col.Regra != domain.RegraColecaoCategoria {
		col.RegraValor = ""
	}
	var errData error
	if req.ExibirDe != nil {
		col.ExibirDe, errData = lerDataColecao(*req.ExibirDe)
	}
	if req.ExibirAte != nil && errData == nil {
		col.ExibirAte, errData = lerDataColecao(*req.ExibirAte)
	}
	if req.AbertaOrganizador != nil {
		col.AbertaOrganizador = *req.AbertaOrganizador
	}
	if req.NaHome != nil {
		col.NaHome = *req.NaHome
	}
	switch {
	case !domain.IconesColecao[col.Icone]:
		c.JSON(http.StatusUnprocessableEntity, gin.H{"erro": "ícone inválido"})
		return
	case !domain.RegrasColecao[col.Regra]:
		c.JSON(http.StatusUnprocessableEntity, gin.H{"erro": "regra inválida"})
		return
	case col.Regra == domain.RegraColecaoCategoria && (col.RegraValor == "" || len(col.RegraValor) > 60):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"erro": "informe a categoria da coleção"})
		return
	case errData != nil:
		c.JSON(http.StatusUnprocessableEntity, gin.H{"erro": "datas no formato AAAA-MM-DD"})
		return
	case col.ExibirDe != nil && col.ExibirAte != nil && col.ExibirDe.After(*col.ExibirAte):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"erro": "o início do período deve vir antes do fim"})
		return
	case col.AbertaOrganizador && col.Regra != domain.RegraColecaoManual:
		c.JSON(http.StatusUnprocessableEntity, gin.H{"erro": "só coleção manual pode ser aberta para o organizador marcar"})
		return
	}
	if col.Slug == "" {
		base := service.Slugify(col.Titulo)
		slug := base
		for i := 2; ; i++ {
			existe, err := h.eventos.SlugColecaoExiste(slug, col.ID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao salvar coleção"})
				return
			}
			if !existe {
				break
			}
			slug = base + "-" + itoa(i)
		}
		col.Slug = slug
	}
	if err := h.eventos.SalvarColecao(col); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao salvar coleção"})
		return
	}

	slugs := make([]string, 0, len(req.Eventos))
	vistos := map[string]bool{}
	for _, s := range req.Eventos {
		s = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(s), "/e/"))
		if i := strings.LastIndex(s, "/e/"); i >= 0 { // aceita a URL inteira do evento
			s = s[i+3:]
		}
		s = strings.Trim(s, "/")
		if s != "" && !vistos[s] {
			vistos[s] = true
			slugs = append(slugs, s)
		}
	}
	ids := []int64{}
	ignorados := []string{}
	if len(slugs) > 0 {
		mapa, err := h.eventos.IDsPorSlugs(slugs)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao salvar coleção"})
			return
		}
		for _, s := range slugs {
			if id, ok := mapa[s]; ok {
				ids = append(ids, id)
			} else {
				ignorados = append(ignorados, s)
			}
		}
	}
	if err := h.eventos.DefinirEventosDaColecao(col.ID, ids); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao salvar eventos da coleção"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": col.ID, "slug": col.Slug, "ignorados": ignorados})
}

// ExcluirColecao é DELETE /api/v1/admin/colecoes/:id.
func (h *PublicoHandler) ExcluirColecao(c *gin.Context) {
	id, ok := idDaURL(c)
	if !ok {
		return
	}
	if _, err := h.eventos.BuscarColecaoPorID(id); errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"erro": "coleção não encontrada"})
		return
	}
	if err := h.eventos.ExcluirColecao(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao excluir coleção"})
		return
	}
	c.Status(http.StatusNoContent)
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b []byte
	for i > 0 {
		b = append([]byte{byte('0' + i%10)}, b...)
		i /= 10
	}
	return string(b)
}

// BloquearEventoDaColecao é POST /api/v1/admin/colecoes/:id/marcados/:slug
// (item 11.7): o admin tira (bloqueado=true) ou devolve um evento que o
// organizador marcou no tema.
func (h *PublicoHandler) BloquearEventoDaColecao(c *gin.Context) {
	id, ok := idDaURL(c)
	if !ok {
		return
	}
	var req struct {
		Bloqueado bool `json:"bloqueado"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "dados inválidos"})
		return
	}
	achou, err := h.eventos.BloquearNaColecao(id, c.Param("slug"), req.Bloqueado)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao salvar"})
		return
	}
	if !achou {
		c.JSON(http.StatusNotFound, gin.H{"erro": "evento não marcado nesta coleção"})
		return
	}
	c.Status(http.StatusNoContent)
}
