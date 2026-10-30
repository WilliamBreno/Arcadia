package service

import (
	"fmt"
	"html"
	"net/url"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/WilliamBreno/Arcadia/backend/internal/mail"
	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
)

// Lembretes antes do evento e "adicionar ao calendário" (item 6.4).

// --- Calendário ---

// periodoCalendario: início e fim (sem fim: 3 horas depois do início).
func periodoCalendario(e *domain.Evento) (time.Time, time.Time, bool) {
	if e.InicioEm == nil {
		return time.Time{}, time.Time{}, false
	}
	fim := e.InicioEm.Add(3 * time.Hour)
	if e.FimEm != nil && e.FimEm.After(*e.InicioEm) {
		fim = *e.FimEm
	}
	return e.InicioEm.UTC(), fim.UTC(), true
}

const formatoICS = "20060102T150405Z"

// escaparICS segue a RFC 5545 (vírgula, ponto e vírgula, barra e quebra).
func escaparICS(s string) string {
	r := strings.NewReplacer(`\`, `\\`, ";", `\;`, ",", `\,`, "\r\n", `\n`, "\n", `\n`)
	return r.Replace(s)
}

// dobrarICS quebra linhas em 75 octetos (RFC 5545 §3.1).
func dobrarICS(linha string) string {
	if len(linha) <= 75 {
		return linha
	}
	var b strings.Builder
	for len(linha) > 75 {
		corte := 75
		for corte > 0 && (linha[corte]&0xC0) == 0x80 { // não corta no meio de um caractere UTF-8
			corte--
		}
		b.WriteString(linha[:corte] + "\r\n ")
		linha = linha[corte:]
	}
	b.WriteString(linha)
	return b.String()
}

// ArquivoICS monta o .ics de um evento (Apple Calendar, Outlook, Google).
func ArquivoICS(e *domain.Evento, onde, frontendURL string, agora time.Time) (string, bool) {
	ini, fim, ok := periodoCalendario(e)
	if !ok {
		return "", false
	}
	link := strings.TrimRight(frontendURL, "/") + "/e/" + e.Slug
	linhas := []string{
		"BEGIN:VCALENDAR", "VERSION:2.0", "PRODID:-//Evve//Eventos//PT-BR", "CALSCALE:GREGORIAN", "METHOD:PUBLISH",
		"BEGIN:VEVENT",
		fmt.Sprintf("UID:evento-%d@evve", e.ID),
		"DTSTAMP:" + agora.UTC().Format(formatoICS),
		"DTSTART:" + ini.Format(formatoICS),
		"DTEND:" + fim.Format(formatoICS),
		"SUMMARY:" + escaparICS(e.Titulo),
		"LOCATION:" + escaparICS(onde),
		"DESCRIPTION:" + escaparICS("Ingressos e detalhes: "+link),
		"URL:" + link,
		"BEGIN:VALARM", "TRIGGER:-PT3H", "ACTION:DISPLAY", "DESCRIPTION:" + escaparICS(e.Titulo), "END:VALARM",
		"END:VEVENT", "END:VCALENDAR",
	}
	for i := range linhas {
		linhas[i] = dobrarICS(linhas[i])
	}
	return strings.Join(linhas, "\r\n") + "\r\n", true
}

// LinkGoogleAgenda: link "adicionar ao Google Agenda" já preenchido.
func LinkGoogleAgenda(e *domain.Evento, onde, frontendURL string) string {
	ini, fim, ok := periodoCalendario(e)
	if !ok {
		return ""
	}
	q := url.Values{
		"action":   {"TEMPLATE"},
		"text":     {e.Titulo},
		"dates":    {ini.Format(formatoICS) + "/" + fim.Format(formatoICS)},
		"location": {onde},
		"details":  {"Ingressos e detalhes: " + strings.TrimRight(frontendURL, "/") + "/e/" + e.Slug},
	}
	return "https://calendar.google.com/calendar/render?" + q.Encode()
}

// EnderecoDoLocal: texto do local para calendário e e-mail.
func EnderecoDoLocal(l *domain.Local) string {
	if l == nil {
		return "Online"
	}
	partes := []string{l.Nome}
	rua := strings.TrimSpace(strings.Trim(l.Logradouro+", "+l.Numero, ", "))
	if rua != "" {
		partes = append(partes, rua)
	}
	if l.Bairro != "" {
		partes = append(partes, l.Bairro)
	}
	partes = append(partes, l.Cidade+"/"+l.UF)
	return strings.Join(partes, " · ")
}

// --- Lembretes ---

// tipoLembrete decide qual lembrete cabe agora. "1d": falta 26 h ou menos;
// "7d": faltam entre 6 e 7 dias. Fora disso, nenhum. Testado.
func tipoLembrete(inicio, agora time.Time) string {
	falta := inicio.Sub(agora)
	switch {
	case falta <= 0:
		return ""
	case falta <= 26*time.Hour:
		return "1d"
	case falta >= 6*24*time.Hour && falta <= 7*24*time.Hour:
		return "7d"
	}
	return ""
}

type LembretesService struct {
	db          *gorm.DB
	locais      *repository.LocalRepository
	mail        *mail.Cliente
	frontendURL string
}

func NovoLembretesService(db *gorm.DB, locais *repository.LocalRepository, m *mail.Cliente, frontendURL string) *LembretesService {
	return &LembretesService{db: db, locais: locais, mail: m, frontendURL: strings.TrimRight(frontendURL, "/")}
}

type itemLembrete struct {
	Email  string
	Nome   string
	Codigo string
	Token  string
}

// Enviar é o job /jobs/lembretes (de hora em hora). Idempotente: cada
// e-mail recebe cada lembrete uma vez por evento (lembretes_enviados).
func (s *LembretesService) Enviar(agora time.Time) (enviados int, err error) {
	var eventos []domain.Evento
	if err := s.db.Where("status = ? AND inicio_em > ? AND inicio_em <= ?", domain.StatusEventoPublicado, agora, agora.Add(7*24*time.Hour)).
		Find(&eventos).Error; err != nil {
		return 0, err
	}
	for i := range eventos {
		e := &eventos[i]
		tipo := tipoLembrete(*e.InicioEm, agora)
		if tipo == "" {
			continue
		}
		var itens []itemLembrete
		if err := s.db.Table("itens_pedido").
			Select("itens_pedido.titular_email as email, itens_pedido.titular_nome as nome, itens_pedido.codigo, itens_pedido.qr_token as token").
			Joins("JOIN tipos_ingresso ON tipos_ingresso.id = itens_pedido.tipo_ingresso_id").
			Where("tipos_ingresso.evento_id = ? AND itens_pedido.status = ? AND itens_pedido.titular_email <> '' AND itens_pedido.ficha_id IS NULL AND itens_pedido.expositor_id IS NULL AND NOT itens_pedido.adicional", e.ID, domain.StatusItemPago).
			Order("itens_pedido.titular_email, itens_pedido.id").
			Scan(&itens).Error; err != nil {
			return enviados, err
		}
		porEmail := map[string][]itemLembrete{}
		ordem := []string{}
		for _, it := range itens {
			chave := strings.ToLower(it.Email)
			if _, ok := porEmail[chave]; !ok {
				ordem = append(ordem, chave)
			}
			porEmail[chave] = append(porEmail[chave], it)
		}
		var local *domain.Local
		if e.LocalID != nil {
			local, _ = s.locais.BuscarPorID(*e.LocalID)
		}
		for _, email := range ordem {
			res := s.db.Exec("INSERT INTO lembretes_enviados (evento_id, email, tipo) VALUES (?, ?, ?) ON CONFLICT DO NOTHING", e.ID, email, tipo)
			if res.Error != nil {
				return enviados, res.Error
			}
			if res.RowsAffected == 0 {
				continue // já lembrado
			}
			itensDoEmail := porEmail[email]
			assunto := "Falta uma semana: " + e.Titulo
			if tipo == "1d" {
				assunto = "É amanhã: " + e.Titulo
			}
			if err := s.mail.Enviar(itensDoEmail[0].Email, assunto, s.corpoLembrete(e, local, tipo, itensDoEmail)); err == nil {
				enviados++
			}
		}
	}
	return enviados, nil
}

func (s *LembretesService) corpoLembrete(e *domain.Evento, l *domain.Local, tipo string, itens []itemLembrete) string {
	onde := EnderecoDoLocal(l)
	var b strings.Builder
	fmt.Fprintf(&b, `<div style="font-family:Arial,sans-serif;max-width:560px;margin:auto;color:#111">`)
	intro := "Falta uma semana para o seu evento."
	if tipo == "1d" {
		intro = "Seu evento está chegando."
	}
	fmt.Fprintf(&b, `<p>Olá, %s! %s</p>`, html.EscapeString(primeiroNome(itens[0].Nome)), intro)
	fmt.Fprintf(&b, `<h2 style="margin:16px 0 4px">%s</h2>`, html.EscapeString(e.Titulo))
	fmt.Fprintf(&b, `<p style="margin:0 0 4px"><strong>%s</strong></p>`, html.EscapeString(DataDoEvento(e)))
	fmt.Fprintf(&b, `<p style="margin:0 0 4px">%s</p>`, html.EscapeString(onde))
	if l != nil && l.Latitude != nil && l.Longitude != nil {
		fmt.Fprintf(&b, `<p style="margin:0 0 12px"><a href="https://www.google.com/maps/search/?api=1&query=%f,%f">Como chegar</a></p>`, *l.Latitude, *l.Longitude)
	}
	b.WriteString(`<p style="margin:16px 0 4px">Seus ingressos:</p><ul>`)
	for _, it := range itens {
		fmt.Fprintf(&b, `<li>%s — <a href="%s/ingresso/%s/%s">abrir ingresso</a></li>`, html.EscapeString(it.Nome), s.frontendURL, it.Codigo, it.Token)
	}
	b.WriteString(`</ul>`)
	if g := LinkGoogleAgenda(e, onde, s.frontendURL); g != "" {
		fmt.Fprintf(&b, `<p style="font-size:13px">Adicionar ao calendário: <a href="%s">Google Agenda</a> · <a href="%s/api/v1/eventos/%s/calendario.ics">Apple / Outlook</a></p>`, g, s.frontendURL, url.PathEscape(e.Slug))
	}
	fmt.Fprintf(&b, `<p style="color:#555;font-size:13px">Apresente o QR na entrada. Se não puder ir, veja em "Meus ingressos" se ainda dá para cancelar ou transferir.</p></div>`)
	return b.String()
}
