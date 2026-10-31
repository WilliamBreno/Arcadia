package service

import (
	"net"
	"strings"
	"testing"
)

func TestExtrairEventoJSONLD(t *testing.T) {
	pagina := `<html><head><script type="application/ld+json">{"@context":"https://schema.org","@graph":[{"@type":"Organization","name":"X"},
	{"@type":"MusicEvent","name":"Noite &amp; Rock","startDate":"2026-12-05T20:00:00-03:00","description":"<p>Show <b>ao vivo</b></p>",
	"image":["https://cdn.x/capa.jpg"],"location":{"@type":"Place","name":"Teatro Deodoro","address":{"@type":"PostalAddress",
	"streetAddress":"Praça Marechal Deodoro","addressLocality":"Maceió","addressRegion":"al"}}}]}</script></head></html>`
	ev := ExtrairEvento(pagina)
	if ev == nil || ev.Titulo != "Noite & Rock" || ev.Cidade != "Maceió" || ev.UF != "AL" || ev.LocalNome != "Teatro Deodoro" {
		t.Fatalf("%+v", ev)
	}
	if ev.InicioEm == nil || ev.InicioEm.Day() != 5 || ev.ImagemURL != "https://cdn.x/capa.jpg" || strings.Contains(ev.Descricao, "<") {
		t.Fatalf("%+v", ev)
	}
}

func TestExtrairEventoOpenGraph(t *testing.T) {
	ev := ExtrairEvento(`<meta property="og:title" content="Feira Geek"><meta content="Tudo de anime" property="og:description">`)
	if ev == nil || ev.Titulo != "Feira Geek" || ev.Descricao != "Tudo de anime" {
		t.Fatalf("%+v", ev)
	}
	if ExtrairEvento(`<html><title>nada</title></html>`) != nil {
		t.Fatal("sem dados de evento deveria voltar nil")
	}
}

func TestLinkSeguro(t *testing.T) {
	for _, ruim := range []string{"ftp://x.com", "http://localhost/a", "http://127.0.0.1/", "https://site.local/x", "https://x.com:8080/",
		"https://user:senha@x.com/", "https://www.instagram.com/p/1", "javascript:alert(1)"} {
		if _, err := validarLink(ruim); err == nil {
			t.Errorf("deveria recusar %s", ruim)
		}
	}
	if _, err := validarLink("https://www.sympla.com.br/evento/x/123"); err != nil {
		t.Fatal(err)
	}
	for _, ip := range []string{"10.0.0.1", "192.168.1.1", "172.16.0.1", "127.0.0.1", "169.254.169.254", "100.64.0.1", "::1", "fd00::1"} {
		if !ipProibido(net.ParseIP(ip)) {
			t.Errorf("%s deveria ser proibido", ip)
		}
	}
	if ipProibido(net.ParseIP("8.8.8.8")) {
		t.Error("IP público é permitido")
	}
}

func TestLimparRascunho(t *testing.T) {
	r := limparRascunho(RascunhoIA{Lotes: []LoteSugerido{{Nome: "1º lote", PrecoCentavos: 5000, Quantidade: 100},
		{Nome: "", PrecoCentavos: 1, Quantidade: 1}, {Nome: "Ruim", PrecoCentavos: -5, Quantidade: 10},
		{Nome: "2º", PrecoCentavos: 6000, Quantidade: 50}, {Nome: "3º", PrecoCentavos: 7000, Quantidade: 50}, {Nome: "4º", PrecoCentavos: 8000, Quantidade: 50}}})
	if len(r.Lotes) != 3 || r.Lotes[0].Nome != "1º lote" {
		t.Fatalf("%+v", r.Lotes)
	}
}
