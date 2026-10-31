package service

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
)

// Item 10.5: feeds públicos (RSS e calendário) para sites, apps de agenda
// e parceiros divulgarem os eventos fora do Evve.

// EventoFeed: o evento com o lugar já em texto.
type EventoFeed struct {
	Evento *domain.Evento
	Onde   string
}

// CalendarioICS monta um calendário com vários eventos (assinável no
// Google Agenda, Apple e Outlook). Testada.
func CalendarioICS(nome string, eventos []EventoFeed, frontendURL string, agora time.Time) string {
	base := strings.TrimRight(frontendURL, "/")
	linhas := []string{"BEGIN:VCALENDAR", "VERSION:2.0", "PRODID:-//Evve//Eventos//PT-BR", "CALSCALE:GREGORIAN", "METHOD:PUBLISH",
		"X-WR-CALNAME:" + escaparICS(nome), "X-PUBLISHED-TTL:PT6H"}
	for _, ef := range eventos {
		ini, fim, ok := periodoCalendario(ef.Evento)
		if !ok {
			continue
		}
		link := base + "/e/" + ef.Evento.Slug
		linhas = append(linhas, "BEGIN:VEVENT",
			fmt.Sprintf("UID:evento-%d@evve", ef.Evento.ID),
			"DTSTAMP:"+agora.UTC().Format(formatoICS),
			"DTSTART:"+ini.Format(formatoICS),
			"DTEND:"+fim.Format(formatoICS),
			"SUMMARY:"+escaparICS(ef.Evento.Titulo),
			"LOCATION:"+escaparICS(ef.Onde),
			"DESCRIPTION:"+escaparICS("Ingressos e detalhes: "+link),
			"URL:"+link,
			"END:VEVENT")
	}
	linhas = append(linhas, "END:VCALENDAR")
	for i := range linhas {
		linhas[i] = dobrarICS(linhas[i])
	}
	return strings.Join(linhas, "\r\n") + "\r\n"
}

type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	GUID        string `xml:"guid"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate,omitempty"`
	Enclosure   *struct {
		URL  string `xml:"url,attr"`
		Type string `xml:"type,attr"`
	} `xml:"enclosure,omitempty"`
}

// FeedRSS monta o RSS 2.0 com os eventos (o texto é escapado pelo XML). Testada.
func FeedRSS(titulo, link, descricao string, eventos []EventoFeed, frontendURL string) ([]byte, error) {
	base := strings.TrimRight(frontendURL, "/")
	type canal struct {
		Title       string    `xml:"title"`
		Link        string    `xml:"link"`
		Description string    `xml:"description"`
		Language    string    `xml:"language"`
		Items       []rssItem `xml:"item"`
	}
	type rss struct {
		XMLName xml.Name `xml:"rss"`
		Version string   `xml:"version,attr"`
		Channel canal    `xml:"channel"`
	}
	c := canal{Title: titulo, Link: link, Description: descricao, Language: "pt-BR"}
	for _, ef := range eventos {
		e := ef.Evento
		url := base + "/e/" + e.Slug
		desc := ef.Onde
		if d := DataDoEvento(e); d != "" {
			desc = d + " · " + desc
		}
		it := rssItem{Title: e.Titulo, Link: url, GUID: url, Description: desc}
		if e.CriadoEm.Year() > 2000 {
			it.PubDate = e.CriadoEm.UTC().Format(time.RFC1123Z)
		}
		if e.CapaURL != "" {
			it.Enclosure = &struct {
				URL  string `xml:"url,attr"`
				Type string `xml:"type,attr"`
			}{URL: e.CapaURL, Type: "image/jpeg"}
		}
		c.Items = append(c.Items, it)
	}
	corpo, err := xml.MarshalIndent(rss{Version: "2.0", Channel: c}, "", "  ")
	if err != nil {
		return nil, err
	}
	return append([]byte(xml.Header), corpo...), nil
}
