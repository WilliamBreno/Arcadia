package handler

import (
	"encoding/xml"
	"strings"
	"testing"
	"time"

	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
)

func TestMontarSitemap(t *testing.T) {
	d := time.Date(2026, 10, 26, 15, 0, 0, 0, time.UTC)
	s := montarSitemap("https://evve.app",
		[]repository.ItemSitemap{{Slug: "cosplay-maceio", AtualizadoEm: d}},
		[]repository.ItemSitemap{{Slug: "org-geek", AtualizadoEm: d}})
	b, err := xml.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	x := string(b)
	for _, esperado := range []string{
		`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`,
		"<loc>https://evve.app/</loc>",
		"<loc>https://evve.app/e/cosplay-maceio</loc><lastmod>2026-10-26</lastmod>",
		"<loc>https://evve.app/o/org-geek</loc>",
	} {
		if !strings.Contains(x, esperado) {
			t.Errorf("sitemap sem %q:\n%s", esperado, x)
		}
	}
}

func TestMontarRobots(t *testing.T) {
	r := montarRobots("https://evve.app")
	for _, esperado := range []string{"Disallow: /admin/", "Disallow: /organizador/", "Sitemap: https://evve.app/sitemap.xml"} {
		if !strings.Contains(r, esperado) {
			t.Errorf("robots sem %q", esperado)
		}
	}
	for _, linha := range strings.Split(r, "\n") {
		if linha == "Disallow: /e/" || linha == "Disallow: /o/" || linha == "Disallow: /" {
			t.Errorf("página pública bloqueada: %q", linha)
		}
	}
	if !strings.Contains(r, "Disallow: /e/*/jurado") {
		t.Error("área do jurado deveria estar fora do Google")
	}
}
