package handler

import (
	"encoding/xml"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
	"github.com/WilliamBreno/Arcadia/backend/internal/service"
)

// SEOHandler serve o sitemap.xml e o robots.txt (item 6.1). A Vercel
// reescreve /sitemap.xml e /robots.txt do site para estas rotas.
type SEOHandler struct {
	eventos     *repository.EventoRepository
	frontendURL string
}

func NovoSEOHandler(eventos *repository.EventoRepository, frontendURL string) *SEOHandler {
	return &SEOHandler{eventos: eventos, frontendURL: strings.TrimRight(frontendURL, "/")}
}

type urlSitemap struct {
	Loc        string `xml:"loc"`
	LastMod    string `xml:"lastmod,omitempty"`
	ChangeFreq string `xml:"changefreq,omitempty"`
}

type urlSet struct {
	XMLName xml.Name     `xml:"urlset"`
	XMLNS   string       `xml:"xmlns,attr"`
	URLs    []urlSitemap `xml:"url"`
}

// O protocolo aceita até 50 mil URLs por arquivo.
const limiteSitemap = 45000

// montarSitemap é puro (testado): páginas fixas, páginas de descoberta
// (cidade, cidade + categoria, coleções — item 6.2), eventos e organizadores.
func montarSitemap(base string, descoberta []string, eventos, organizadores []repository.ItemSitemap) urlSet {
	s := urlSet{XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9"}
	s.URLs = append(s.URLs,
		urlSitemap{Loc: base + "/", ChangeFreq: "daily"},
		urlSitemap{Loc: base + "/mapa", ChangeFreq: "daily"},
		urlSitemap{Loc: base + "/quanto-custa", ChangeFreq: "monthly"},
	)
	for _, caminho := range descoberta {
		s.URLs = append(s.URLs, urlSitemap{Loc: base + caminho, ChangeFreq: "daily"})
	}
	for _, e := range eventos {
		s.URLs = append(s.URLs, urlSitemap{Loc: base + "/e/" + e.Slug, LastMod: e.AtualizadoEm.UTC().Format(time.DateOnly), ChangeFreq: "daily"})
	}
	for _, o := range organizadores {
		s.URLs = append(s.URLs, urlSitemap{Loc: base + "/o/" + o.Slug, LastMod: o.AtualizadoEm.UTC().Format(time.DateOnly), ChangeFreq: "weekly"})
	}
	return s
}

// Sitemap é GET /api/v1/seo/sitemap.xml.
func (h *SEOHandler) Sitemap(c *gin.Context) {
	eventos, err := h.eventos.ListarParaSitemap(limiteSitemap)
	if err != nil {
		c.String(http.StatusInternalServerError, "erro ao gerar sitemap")
		return
	}
	organizadores, err := h.eventos.ListarOrganizadoresParaSitemap(limiteSitemap - len(eventos))
	if err != nil {
		c.String(http.StatusInternalServerError, "erro ao gerar sitemap")
		return
	}
	descoberta, err := h.paginasDescoberta()
	if err != nil {
		c.String(http.StatusInternalServerError, "erro ao gerar sitemap")
		return
	}
	corpo, err := xml.MarshalIndent(montarSitemap(h.frontendURL, descoberta, eventos, organizadores), "", "  ")
	if err != nil {
		c.String(http.StatusInternalServerError, "erro ao gerar sitemap")
		return
	}
	c.Header("Cache-Control", "public, max-age=3600")
	c.Data(http.StatusOK, "application/xml; charset=utf-8", append([]byte(xml.Header), corpo...))
}

// montarRobots: libera as páginas públicas e esconde as áreas logadas.
func montarRobots(base string) string {
	return strings.Join([]string{
		"User-agent: *",
		"Allow: /",
		"Disallow: /organizador/",
		"Disallow: /admin/",
		"Disallow: /pedidos/",
		"Disallow: /checkin/",
		"Disallow: /convite/",
		"Disallow: /ingresso/",
		"Disallow: /meus-ingressos",
		"Disallow: /meus-eventos",
		"Disallow: /api/",
		"Disallow: /login",
		"Disallow: /cadastro",
		"Disallow: /e/*/jurado",
		"",
		"Sitemap: " + base + "/sitemap.xml",
		"",
	}, "\n")
}

// Robots é GET /api/v1/seo/robots.txt.
func (h *SEOHandler) Robots(c *gin.Context) {
	c.Header("Cache-Control", "public, max-age=86400")
	c.String(http.StatusOK, montarRobots(h.frontendURL))
}

// paginasDescoberta: cada cidade com eventos, cada categoria dentro dela e
// cada coleção ativa com evento (item 6.2). Só entram páginas com conteúdo.
func (h *SEOHandler) paginasDescoberta() ([]string, error) {
	var caminhos []string
	cidades, err := h.eventos.ListarCidadesComEventos()
	if err != nil {
		return nil, err
	}
	for _, ci := range cidades {
		base := "/eventos/" + slugCidade(ci.Cidade, ci.UF)
		caminhos = append(caminhos, base)
		categorias, err := h.eventos.ListarCategoriasComEventos(ci.Cidade, ci.UF)
		if err != nil {
			return nil, err
		}
		for _, ca := range categorias {
			caminhos = append(caminhos, base+"/"+service.Slugify(ca.Categoria))
		}
	}
	colecoes, err := h.eventos.ListarColecoes(true)
	if err != nil {
		return nil, err
	}
	for _, col := range colecoes {
		if eventos, err := h.eventos.EventosDaColecao(col.ID, 1); err == nil && len(eventos) > 0 {
			caminhos = append(caminhos, "/c/"+col.Slug)
		}
	}
	return caminhos, nil
}
