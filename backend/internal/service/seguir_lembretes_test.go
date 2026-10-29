package service

import (
	"strings"
	"testing"
	"time"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
)

func TestTipoLembrete(t *testing.T) {
	agora := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	casos := []struct {
		falta time.Duration
		tipo  string
	}{
		{-time.Hour, ""},
		{2 * time.Hour, "1d"},
		{26 * time.Hour, "1d"},
		{27 * time.Hour, ""},
		{3 * 24 * time.Hour, ""},
		{6*24*time.Hour + time.Hour, "7d"},
		{7 * 24 * time.Hour, "7d"},
		{8 * 24 * time.Hour, ""},
	}
	for _, c := range casos {
		if got := tipoLembrete(agora.Add(c.falta), agora); got != c.tipo {
			t.Errorf("falta %v: %q, esperado %q", c.falta, got, c.tipo)
		}
	}
}

func eventoTeste() *domain.Evento {
	ini := time.Date(2026, 11, 7, 23, 0, 0, 0, time.UTC) // 20:00 em Maceió
	return &domain.Evento{ID: 42, Slug: "cosplay-fest", Titulo: "Cosplay, Dança; e mais", InicioEm: &ini, Timezone: "America/Maceio"}
}

func TestArquivoICS(t *testing.T) {
	ics, ok := ArquivoICS(eventoTeste(), "Centro de Convenções · Maceió/AL", "https://evve.app", time.Date(2026, 10, 29, 0, 0, 0, 0, time.UTC))
	if !ok {
		t.Fatal("deveria gerar")
	}
	for _, esperado := range []string{
		"BEGIN:VCALENDAR\r\n", "UID:evento-42@evve", "DTSTART:20261107T230000Z", "DTEND:20261108T020000Z",
		`SUMMARY:Cosplay\, Dança\; e mais`, "URL:https://evve.app/e/cosplay-fest", "END:VCALENDAR\r\n",
	} {
		if !strings.Contains(ics, esperado) {
			t.Errorf("ics sem %q:\n%s", esperado, ics)
		}
	}
	for _, linha := range strings.Split(ics, "\r\n") {
		if len(linha) > 75 {
			t.Errorf("linha com mais de 75 octetos: %q", linha)
		}
	}
	if _, ok := ArquivoICS(&domain.Evento{Slug: "x"}, "", "https://evve.app", time.Now()); ok {
		t.Error("evento sem data não gera .ics")
	}
}

func TestLinkGoogleAgenda(t *testing.T) {
	l := LinkGoogleAgenda(eventoTeste(), "Maceió/AL", "https://evve.app")
	if !strings.HasPrefix(l, "https://calendar.google.com/calendar/render?") || !strings.Contains(l, "dates=20261107T230000Z%2F20261108T020000Z") {
		t.Errorf("link inesperado: %s", l)
	}
}

func TestDataDoEventoNoFusoDoEvento(t *testing.T) {
	if got := DataDoEvento(eventoTeste()); got != "sábado, 07/11/2026 às 20:00" {
		t.Errorf("DataDoEvento = %q", got)
	}
}

func TestTokenDescadastro(t *testing.T) {
	a := tokenDescadastro("segredo", 7, "organizador", "3")
	if a != tokenDescadastro("segredo", 7, "organizador", "3") || len(a) != 32 {
		t.Error("token deveria ser estável e ter 32 caracteres")
	}
	for _, outro := range []string{
		tokenDescadastro("segredo", 8, "organizador", "3"),
		tokenDescadastro("segredo", 7, "organizador", "4"),
		tokenDescadastro("segredo", 7, "categoria", "3"),
		tokenDescadastro("outro", 7, "organizador", "3"),
	} {
		if outro == a {
			t.Error("token de outra pessoa/alvo/segredo não pode coincidir")
		}
	}
}

func TestAlvosDoEvento(t *testing.T) {
	org, cats := alvosDoEvento(5, "danca", "maceio-al")
	if org != "5" || len(cats) != 2 || cats[0] != "danca" || cats[1] != "danca@maceio-al" {
		t.Errorf("alvos = %q %v", org, cats)
	}
	if _, cats := alvosDoEvento(5, "danca", ""); len(cats) != 1 {
		t.Errorf("evento online: só a categoria geral, %v", cats)
	}
}

func TestReAlvoCategoria(t *testing.T) {
	for _, ok := range []string{"danca", "games-e-geek", "danca@maceio-al", "games-e-geek@nossa-senhora-do-socorro-se"} {
		if !reAlvoCategoria.MatchString(ok) {
			t.Errorf("%q deveria valer", ok)
		}
	}
	for _, ruim := range []string{"", "Dança", "danca@", "danca@maceio", "<script>", "danca@maceio-al@x"} {
		if reAlvoCategoria.MatchString(ruim) {
			t.Errorf("%q não deveria valer", ruim)
		}
	}
}
