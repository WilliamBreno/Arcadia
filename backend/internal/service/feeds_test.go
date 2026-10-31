package service

import (
	"strings"
	"testing"
	"time"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
)

func TestFeeds(t *testing.T) {
	ini := time.Date(2026, 12, 5, 20, 0, 0, 0, time.UTC)
	e := &domain.Evento{ID: 7, Titulo: "Rock & <Roll>", Slug: "rock", InicioEm: &ini, CapaURL: "https://x/capa.jpg", CriadoEm: ini.AddDate(0, -1, 0)}
	itens := []EventoFeed{{Evento: e, Onde: "Teatro, Maceió/AL"}}
	ics := CalendarioICS("Eventos em Maceió", itens, "https://evve.app/", ini)
	for _, trecho := range []string{"BEGIN:VCALENDAR\r\n", "X-WR-CALNAME:Eventos em Maceió", "UID:evento-7@evve", "URL:https://evve.app/e/rock", "END:VCALENDAR\r\n"} {
		if !strings.Contains(ics, trecho) {
			t.Errorf("ics sem %q:\n%s", trecho, ics)
		}
	}
	rss, err := FeedRSS("Eventos", "https://evve.app", "Próximos", itens, "https://evve.app")
	if err != nil {
		t.Fatal(err)
	}
	s := string(rss)
	if !strings.Contains(s, "Rock &amp; &lt;Roll&gt;") || !strings.Contains(s, `<enclosure url="https://x/capa.jpg"`) || !strings.Contains(s, "<link>https://evve.app/e/rock</link>") {
		t.Fatalf("rss:\n%s", s)
	}
}
