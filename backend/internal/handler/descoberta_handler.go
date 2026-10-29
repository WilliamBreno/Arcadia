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
	Eventos   []eventoPublicoItem `json:"eventos"`
}

// Colecoes é GET /api/v1/colecoes: coleções ativas com ao menos um evento.
func (h *PublicoHandler) Colecoes(c *gin.Context) {
	cs, err := h.eventos.ListarColecoes(true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao listar coleções"})
		return
	}
	resp := make([]colecaoPublica, 0, len(cs))
	for _, col := range cs {
		eventos, err := h.eventos.EventosDaColecao(col.ID, 8)
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
		resp = append(resp, colecaoPublica{Slug: col.Slug, Titulo: col.Titulo, Descricao: col.Descricao, Eventos: itens})
	}
	c.Header("Cache-Control", "public, max-age=300")
	c.JSON(http.StatusOK, resp)
}

// Colecao é GET /api/v1/colecoes/:slug.
func (h *PublicoHandler) Colecao(c *gin.Context) {
	col, err := h.eventos.BuscarColecao(c.Param("slug"))
	if err != nil || !col.Ativa {
		c.JSON(http.StatusNotFound, gin.H{"erro": "coleção não encontrada"})
		return
	}
	eventos, err := h.eventos.EventosDaColecao(col.ID, 100)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao listar a coleção"})
		return
	}
	itens, err := h.paraItensPublicos(eventos, true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao calcular preços"})
		return
	}
	c.JSON(http.StatusOK, colecaoPublica{Slug: col.Slug, Titulo: col.Titulo, Descricao: col.Descricao, Eventos: itens})
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
}

type colecaoRequest struct {
	Titulo    string   `json:"titulo" binding:"required,min=2,max=120"`
	Descricao string   `json:"descricao" binding:"max=500"`
	Ordem     int      `json:"ordem"`
	Ativa     *bool    `json:"ativa"`
	Eventos   []string `json:"eventos"`
}

// ListarColecoesAdmin é GET /api/v1/admin/colecoes.
func (h *PublicoHandler) ListarColecoesAdmin(c *gin.Context) {
	cs, err := h.eventos.ListarColecoes(false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao listar coleções"})
		return
	}
	resp := make([]colecaoAdmin, 0, len(cs))
	for _, col := range cs {
		slugs, err := h.eventos.SlugsDaColecao(col.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao listar coleções"})
			return
		}
		if slugs == nil {
			slugs = []string{}
		}
		resp = append(resp, colecaoAdmin{ID: col.ID, Slug: col.Slug, Titulo: col.Titulo, Descricao: col.Descricao, Ordem: col.Ordem, Ativa: col.Ativa, Eventos: slugs})
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
	col := &domain.Colecao{Ativa: true, CriadoEm: time.Now()}
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
