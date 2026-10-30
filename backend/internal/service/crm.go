package service

// Item 9.3: CRM do organizador. Decisões do dono (30/10/2026):
//   - só recebe quem aceitou: quem segue o organizador (6.4), inclusive
//     quem marca "quero receber novidades" na compra (vira seguir);
//   - cota de 5.000 e-mails por mês e 2 campanhas por dia por organizador;
//   - exportação CSV compatível com Mailchimp e Brevo (sem guardar chaves
//     de terceiros);
//   - WhatsApp oficial fica para depois.
// O organizador escreve texto simples (sem HTML): a Evve escapa tudo, faz
// os parágrafos, transforma links https em clicáveis e põe o descadastro.

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"html"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/WilliamBreno/Arcadia/backend/internal/mail"
	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
)

const (
	CotaEmailsMes      = 5000
	CampanhasPorDia    = 2
	SegmentoTodos      = "todos"
	SegmentoCompraram  = "compraram"
	SegmentoForam      = "foram"
	maxAssuntoCampanha = 120
	maxCorpoCampanha   = 5000
)

var (
	ErrCampanhaInvalida = errors.New("campanha inválida: assunto de até 120 caracteres e texto de até 5.000")
	ErrSegmentoInvalido = errors.New("segmento inválido")
	ErrCotaEmails       = errors.New("cota de e-mails do mês esgotada")
	ErrCampanhasHoje    = fmt.Errorf("limite de %d campanhas por dia atingido; tente amanhã", CampanhasPorDia)
	ErrSemContatos      = errors.New("nenhum contato neste segmento")
)

type Contato struct {
	UsuarioID int64
	Nome      string
	Email     string
	Eventos   []string // títulos dos eventos deste organizador em que comprou
}

type CRMService struct {
	db            *gorm.DB
	organizadores *repository.OrganizadorRepository
	seguir        *SeguirService
	mailCliente   *mail.Cliente
	frontendURL   string
}

func NovoCRMService(db *gorm.DB, o *repository.OrganizadorRepository, s *SeguirService, m *mail.Cliente, frontendURL string) *CRMService {
	return &CRMService{db: db, organizadores: o, seguir: s, mailCliente: m, frontendURL: strings.TrimRight(frontendURL, "/")}
}

func (s *CRMService) validarSegmento(organizadorID int64, segmento string, eventoID *int64) error {
	switch segmento {
	case SegmentoTodos:
		return nil
	case SegmentoCompraram, SegmentoForam:
		if eventoID == nil {
			return ErrSegmentoInvalido
		}
		var n int64
		s.db.Model(&domain.Evento{}).Where("id = ? AND organizador_id = ?", *eventoID, organizadorID).Count(&n)
		if n == 0 {
			return ErrSegmentoInvalido
		}
		return nil
	}
	return ErrSegmentoInvalido
}

// contatos: seguidores do organizador com novidades ligadas e e-mail
// verificado (sem o próprio organizador), filtrados pelo segmento.
func (s *CRMService) contatos(organizadorID int64, segmento string, eventoID *int64) ([]Contato, error) {
	org, err := s.organizadores.BuscarPorID(organizadorID)
	if err != nil {
		return nil, err
	}
	q := s.db.Table("seguindo sg").Select("u.id AS usuario_id, u.nome, u.email").
		Joins("JOIN usuarios u ON u.id = sg.usuario_id").
		Where("sg.tipo = ? AND sg.alvo = ? AND u.receber_novidades AND u.email_verificado_em IS NOT NULL AND u.id <> ?",
			SeguirOrganizador, strconv.FormatInt(organizadorID, 10), org.UsuarioID)
	itemValido := `SELECT 1 FROM pedidos p JOIN itens_pedido i ON i.pedido_id = p.id
		WHERE p.usuario_id = u.id AND p.evento_id = ? AND i.ficha_id IS NULL AND i.expositor_id IS NULL AND i.status IN ?`
	switch segmento {
	case SegmentoCompraram:
		q = q.Where("EXISTS ("+itemValido+")", *eventoID, []string{"pago", "utilizado"})
	case SegmentoForam:
		q = q.Where("EXISTS ("+itemValido+")", *eventoID, []string{"utilizado"})
	}
	var cs []Contato
	if err := q.Order("u.nome").Scan(&cs).Error; err != nil {
		return nil, err
	}
	if len(cs) == 0 {
		return cs, nil
	}
	ids := make([]int64, len(cs))
	for i := range cs {
		ids[i] = cs[i].UsuarioID
	}
	var compras []struct {
		UsuarioID int64
		Titulo    string
	}
	s.db.Raw(`SELECT DISTINCT p.usuario_id, e.titulo FROM pedidos p JOIN eventos e ON e.id = p.evento_id
		JOIN itens_pedido i ON i.pedido_id = p.id
		WHERE e.organizador_id = ? AND p.usuario_id IN ? AND i.status IN ('pago','utilizado') AND i.ficha_id IS NULL AND i.expositor_id IS NULL
		ORDER BY e.titulo`, organizadorID, ids).Scan(&compras)
	idx := map[int64]int{}
	for i := range cs {
		idx[cs[i].UsuarioID] = i
	}
	for _, c := range compras {
		i := idx[c.UsuarioID]
		cs[i].Eventos = append(cs[i].Eventos, c.Titulo)
	}
	return cs, nil
}

// Resumo do painel do CRM.
type ResumoCRM struct {
	Contatos      int
	EnviadosMes   int
	CotaRestante  int
	CampanhasHoje int
	Campanhas     []CampanhaResumo
}

type CampanhaResumo struct {
	Assunto       string
	Segmento      string
	EventoTitulo  string
	Destinatarios int
	EnviadaEm     time.Time
}

func inicioDoMes(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
}

func inicioDoDia(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func (s *CRMService) usoDoMes(organizadorID int64, agora time.Time) (enviados int, hoje int) {
	s.db.Raw(`SELECT COALESCE(SUM(destinatarios), 0) FROM campanhas_email WHERE organizador_id = ? AND enviada_em >= ?`,
		organizadorID, inicioDoMes(agora)).Scan(&enviados)
	s.db.Raw(`SELECT COUNT(*) FROM campanhas_email WHERE organizador_id = ? AND enviada_em >= ?`,
		organizadorID, inicioDoDia(agora)).Scan(&hoje)
	return
}

func (s *CRMService) Resumo(organizadorID int64) (*ResumoCRM, error) {
	cs, err := s.contatos(organizadorID, SegmentoTodos, nil)
	if err != nil {
		return nil, err
	}
	agora := time.Now().In(fusoPadrao())
	r := &ResumoCRM{Contatos: len(cs)}
	r.EnviadosMes, r.CampanhasHoje = s.usoDoMes(organizadorID, agora)
	r.CotaRestante = CotaEmailsMes - r.EnviadosMes
	if r.CotaRestante < 0 {
		r.CotaRestante = 0
	}
	s.db.Raw(`SELECT c.assunto, c.segmento, COALESCE(e.titulo, '') AS evento_titulo, c.destinatarios, c.enviada_em
		FROM campanhas_email c LEFT JOIN eventos e ON e.id = c.segmento_evento_id
		WHERE c.organizador_id = ? ORDER BY c.enviada_em DESC LIMIT 30`, organizadorID).Scan(&r.Campanhas)
	return r, nil
}

func fusoPadrao() *time.Location {
	return FusoDoEvento(&domain.Evento{})
}

// Previa: quantas pessoas receberiam.
func (s *CRMService) Previa(organizadorID int64, segmento string, eventoID *int64) (int, error) {
	if err := s.validarSegmento(organizadorID, segmento, eventoID); err != nil {
		return 0, err
	}
	cs, err := s.contatos(organizadorID, segmento, eventoID)
	return len(cs), err
}

var reLink = regexp.MustCompile(`https://[^\s<>"']+`)

// CorpoCampanha transforma o texto do organizador em HTML seguro:
// escapa tudo, parágrafos por linha em branco, quebra de linha simples e
// links https clicáveis. Testada.
func CorpoCampanha(texto string) string {
	var b strings.Builder
	for _, par := range strings.Split(strings.ReplaceAll(strings.TrimSpace(texto), "\r\n", "\n"), "\n\n") {
		par = strings.TrimSpace(par)
		if par == "" {
			continue
		}
		esc := html.EscapeString(par)
		esc = reLink.ReplaceAllStringFunc(esc, func(u string) string {
			return fmt.Sprintf(`<a href="%s">%s</a>`, u, u)
		})
		b.WriteString("<p>" + strings.ReplaceAll(esc, "\n", "<br>") + "</p>")
	}
	return b.String()
}

func (s *CRMService) emailCampanha(nomeOrg, nome, corpoHTML, descadastro string) string {
	return fmt.Sprintf(`<div style="font-family:Arial,sans-serif;max-width:560px;margin:auto;color:#111"><p>Olá, %s!</p>%s
<p style="color:#777;font-size:12px;margin-top:24px">Você recebe este e-mail porque segue %s no Evve. <a href="%s">Deixar de seguir</a>.</p></div>`,
		html.EscapeString(primeiroNome(nome)), corpoHTML, html.EscapeString(nomeOrg), descadastro)
}

type DadosCampanha struct {
	Assunto  string
	Corpo    string
	Segmento string
	EventoID *int64
	SoTeste  bool // manda só para o próprio organizador (não conta na cota)
}

// Enviar põe os e-mails na fila. Devolve quantos destinatários.
func (s *CRMService) Enviar(organizadorID, usuarioID int64, d DadosCampanha) (int, error) {
	d.Assunto, d.Corpo = strings.TrimSpace(d.Assunto), strings.TrimSpace(d.Corpo)
	if d.Assunto == "" || d.Corpo == "" || len([]rune(d.Assunto)) > maxAssuntoCampanha || len([]rune(d.Corpo)) > maxCorpoCampanha ||
		strings.ContainsAny(d.Assunto, "\r\n") {
		return 0, ErrCampanhaInvalida
	}
	if err := s.validarSegmento(organizadorID, d.Segmento, d.EventoID); err != nil {
		return 0, err
	}
	org, err := s.organizadores.BuscarPorID(organizadorID)
	if err != nil {
		return 0, err
	}
	corpo := CorpoCampanha(d.Corpo)
	if d.SoTeste {
		var u domain.Usuario
		if err := s.db.First(&u, usuarioID).Error; err != nil {
			return 0, err
		}
		return 1, s.mailCliente.Enviar(u.Email, "[Teste] "+d.Assunto, s.emailCampanha(org.NomePublico, u.Nome, corpo, s.frontendURL))
	}
	cs, err := s.contatos(organizadorID, d.Segmento, d.EventoID)
	if err != nil {
		return 0, err
	}
	if len(cs) == 0 {
		return 0, ErrSemContatos
	}
	err = s.db.Transaction(func(tx *gorm.DB) error {
		// Trava o organizador: duas campanhas ao mesmo tempo não furam a cota.
		if err := tx.Exec(`SELECT id FROM organizadores WHERE id = ? FOR UPDATE`, organizadorID).Error; err != nil {
			return err
		}
		agora := time.Now().In(fusoPadrao())
		var enviados, hoje int
		tx.Raw(`SELECT COALESCE(SUM(destinatarios), 0) FROM campanhas_email WHERE organizador_id = ? AND enviada_em >= ?`, organizadorID, inicioDoMes(agora)).Scan(&enviados)
		tx.Raw(`SELECT COUNT(*) FROM campanhas_email WHERE organizador_id = ? AND enviada_em >= ?`, organizadorID, inicioDoDia(agora)).Scan(&hoje)
		if hoje >= CampanhasPorDia {
			return ErrCampanhasHoje
		}
		if enviados+len(cs) > CotaEmailsMes {
			return fmt.Errorf("%w: restam %d este mês e o segmento tem %d pessoas", ErrCotaEmails, CotaEmailsMes-enviados, len(cs))
		}
		return tx.Exec(`INSERT INTO campanhas_email (organizador_id, criado_por, assunto, corpo_texto, segmento, segmento_evento_id, destinatarios, enviada_em)
			VALUES (?, ?, ?, ?, ?, ?, ?, now())`, organizadorID, usuarioID, d.Assunto, d.Corpo, d.Segmento, d.EventoID, len(cs)).Error
	})
	if err != nil {
		return 0, err
	}
	alvo := strconv.FormatInt(organizadorID, 10)
	for _, c := range cs {
		_ = s.mailCliente.Enviar(c.Email, d.Assunto, s.emailCampanha(org.NomePublico, c.Nome, corpo, s.seguir.LinkDescadastro(c.UsuarioID, SeguirOrganizador, alvo)))
	}
	return len(cs), nil
}

// ExportarCSV: contatos com consentimento, colunas que o Mailchimp e o
// Brevo reconhecem na importação (Email Address / First Name / Last Name /
// Tags). Células protegidas contra fórmula.
func (s *CRMService) ExportarCSV(organizadorID int64) ([]byte, error) {
	cs, err := s.contatos(organizadorID, SegmentoTodos, nil)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	buf.Write([]byte{0xEF, 0xBB, 0xBF}) // marca UTF-8: o Excel abre com acentos
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"Email Address", "First Name", "Last Name", "Tags"})
	for _, c := range cs {
		partes := strings.Fields(c.Nome)
		primeiro, resto := "", ""
		if len(partes) > 0 {
			primeiro, resto = partes[0], strings.Join(partes[1:], " ")
		}
		tags := make([]string, 0, len(c.Eventos))
		for _, e := range c.Eventos {
			tags = append(tags, strings.ReplaceAll(e, ",", " "))
		}
		_ = w.Write([]string{celulaCSV(c.Email), celulaCSV(primeiro), celulaCSV(resto), celulaCSV(strings.Join(tags, ","))})
	}
	w.Flush()
	return buf.Bytes(), w.Error()
}
