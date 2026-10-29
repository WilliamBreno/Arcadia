package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"html"
	"log/slog"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/WilliamBreno/Arcadia/backend/internal/mail"
	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
)

// Seguir organizador e categoria (item 6.4). Seguir é o consentimento para
// receber, por e-mail, o aviso de evento novo; todo aviso traz um link de
// "deixar de seguir" que funciona sem login e um de "não receber mais".

var (
	ErrSeguirInvalido = errors.New("não é possível seguir isso")
	ErrLinkInvalido   = errors.New("link de descadastro inválido")
)

const (
	SeguirOrganizador = "organizador"
	SeguirCategoria   = "categoria"
)

var reAlvoCategoria = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*(@[a-z0-9]+(-[a-z0-9]+)*-[a-z]{2})?$`)

type Seguindo struct {
	ID        int64 `gorm:"primaryKey"`
	UsuarioID int64
	Tipo      string
	Alvo      string
	Rotulo    string
	CriadoEm  time.Time
}

func (Seguindo) TableName() string { return "seguindo" }

type SeguirService struct {
	db            *gorm.DB
	organizadores *repository.OrganizadorRepository
	locais        *repository.LocalRepository
	mail          *mail.Cliente
	segredo       string
	frontendURL   string
	nome          string
}

func NovoSeguirService(db *gorm.DB, organizadores *repository.OrganizadorRepository, locais *repository.LocalRepository, m *mail.Cliente, segredo, frontendURL, nome string) *SeguirService {
	return &SeguirService{db: db, organizadores: organizadores, locais: locais, mail: m, segredo: segredo, frontendURL: strings.TrimRight(frontendURL, "/"), nome: nome}
}

// Seguir cria (ou mantém) o seguir. Idempotente.
func (s *SeguirService) Seguir(usuarioID int64, tipo, alvo, rotulo string) error {
	switch tipo {
	case SeguirOrganizador:
		id, err := strconv.ParseInt(alvo, 10, 64)
		if err != nil {
			return ErrSeguirInvalido
		}
		org, err := s.organizadores.BuscarPorID(id)
		if err != nil {
			return ErrSeguirInvalido
		}
		if org.UsuarioID == usuarioID {
			return fmt.Errorf("%w: você não precisa seguir o próprio perfil", ErrSeguirInvalido)
		}
		rotulo = org.NomePublico
	case SeguirCategoria:
		if !reAlvoCategoria.MatchString(alvo) {
			return ErrSeguirInvalido
		}
		rotulo = strings.TrimSpace(rotulo)
		if len(rotulo) > 200 {
			rotulo = rotulo[:200]
		}
	default:
		return ErrSeguirInvalido
	}
	return s.db.Exec(`INSERT INTO seguindo (usuario_id, tipo, alvo, rotulo) VALUES (?, ?, ?, ?)
		ON CONFLICT (usuario_id, tipo, alvo) DO NOTHING`, usuarioID, tipo, alvo, rotulo).Error
}

func (s *SeguirService) Deixar(usuarioID int64, tipo, alvo string) error {
	return s.db.Where("usuario_id = ? AND tipo = ? AND alvo = ?", usuarioID, tipo, alvo).Delete(&Seguindo{}).Error
}

func (s *SeguirService) Listar(usuarioID int64) ([]Seguindo, error) {
	var lista []Seguindo
	return lista, s.db.Where("usuario_id = ?", usuarioID).Order("criado_em DESC").Find(&lista).Error
}

func (s *SeguirService) ContarSeguidores(tipo, alvo string) (int64, error) {
	var n int64
	return n, s.db.Model(&Seguindo{}).Where("tipo = ? AND alvo = ?", tipo, alvo).Count(&n).Error
}

// --- Descadastro sem login ---

// tokenDescadastro assina (usuário, tipo, alvo) — quem tem o link só
// consegue mexer naquele seguir daquela pessoa. Testado.
func tokenDescadastro(segredo string, usuarioID int64, tipo, alvo string) string {
	m := hmac.New(sha256.New, []byte(segredo))
	fmt.Fprintf(m, "descadastro:%d:%s:%s", usuarioID, tipo, alvo)
	return hex.EncodeToString(m.Sum(nil))[:32]
}

func (s *SeguirService) linkDescadastro(usuarioID int64, tipo, alvo string) string {
	q := url.Values{"u": {strconv.FormatInt(usuarioID, 10)}, "tipo": {tipo}, "alvo": {alvo}, "t": {tokenDescadastro(s.segredo, usuarioID, tipo, alvo)}}
	return s.frontendURL + "/descadastrar?" + q.Encode()
}

// Descadastrar deixa de seguir (e, com tudo=true, desliga todas as
// novidades por e-mail). Confere o token em tempo constante.
func (s *SeguirService) Descadastrar(usuarioID int64, tipo, alvo, token string, tudo bool) error {
	esperado := tokenDescadastro(s.segredo, usuarioID, tipo, alvo)
	if !hmac.Equal([]byte(esperado), []byte(token)) {
		return ErrLinkInvalido
	}
	if err := s.Deixar(usuarioID, tipo, alvo); err != nil {
		return err
	}
	if tudo {
		return s.db.Model(&domain.Usuario{}).Where("id = ?", usuarioID).Update("receber_novidades", false).Error
	}
	return nil
}

// --- Aviso de evento novo ---

type destinatarioAviso struct {
	UsuarioID int64
	Nome      string
	Email     string
	Tipo      string
	Alvo      string
	Rotulo    string
}

// alvosDoEvento: o que, seguido, faz a pessoa receber o aviso deste evento.
func alvosDoEvento(organizadorID int64, categoriaSlug, cidadeSlug string) (org string, categorias []string) {
	org = strconv.FormatInt(organizadorID, 10)
	if categoriaSlug != "" {
		categorias = append(categorias, categoriaSlug)
		if cidadeSlug != "" {
			categorias = append(categorias, categoriaSlug+"@"+cidadeSlug)
		}
	}
	return org, categorias
}

// NotificarPublicacao avisa os seguidores de um evento recém-publicado,
// uma vez por evento (tabela avisos_publicacao) e uma vez por pessoa mesmo
// que ela siga o organizador e a categoria. Roda em segundo plano.
func (s *SeguirService) NotificarPublicacao(evento *domain.Evento) {
	if evento.Visibilidade != domain.VisibilidadePublico {
		return
	}
	go func() {
		if err := s.notificar(evento); err != nil {
			slog.Error("erro ao avisar seguidores", "evento_id", evento.ID, "erro", err)
		}
	}()
}

func (s *SeguirService) notificar(evento *domain.Evento) error {
	res := s.db.Exec("INSERT INTO avisos_publicacao (evento_id) VALUES (?) ON CONFLICT DO NOTHING", evento.ID)
	if res.Error != nil || res.RowsAffected == 0 {
		return res.Error // já avisado antes
	}

	cidadeSlug, onde := "", "Online"
	if evento.LocalID != nil {
		if l, err := s.locais.BuscarPorID(*evento.LocalID); err == nil {
			cidadeSlug = Slugify(l.Cidade) + "-" + strings.ToLower(l.UF)
			onde = l.Nome + " · " + l.Cidade + "/" + l.UF
		}
	}
	org, categorias := alvosDoEvento(evento.OrganizadorID, Slugify(evento.Categoria), cidadeSlug)
	organizador, _ := s.organizadores.BuscarPorID(evento.OrganizadorID)

	var linhas []destinatarioAviso
	q := s.db.Table("seguindo").
		Select("seguindo.usuario_id, usuarios.nome, usuarios.email, seguindo.tipo, seguindo.alvo, seguindo.rotulo").
		Joins("JOIN usuarios ON usuarios.id = seguindo.usuario_id").
		Where("usuarios.receber_novidades AND usuarios.email_verificado_em IS NOT NULL").
		Where("(seguindo.tipo = ? AND seguindo.alvo = ?) OR (seguindo.tipo = ? AND seguindo.alvo IN ?)", SeguirOrganizador, org, SeguirCategoria, append(categorias, "")).
		Order("seguindo.tipo DESC") // "organizador" antes de "categoria": o motivo mais forte fica
	if organizador != nil {
		q = q.Where("seguindo.usuario_id <> ?", organizador.UsuarioID)
	}
	if err := q.Scan(&linhas).Error; err != nil {
		return err
	}

	vistos := map[int64]bool{}
	enviados := 0
	for _, d := range linhas {
		if vistos[d.UsuarioID] {
			continue
		}
		vistos[d.UsuarioID] = true
		motivo := "Você segue " + d.Rotulo + "."
		if d.Tipo == SeguirOrganizador && organizador != nil {
			motivo = "Você segue " + organizador.NomePublico + "."
		}
		corpo := corpoAvisoEvento(d.Nome, motivo, evento, onde, s.frontendURL, s.linkDescadastro(d.UsuarioID, d.Tipo, d.Alvo))
		if err := s.mail.Enviar(d.Email, "Evento novo: "+evento.Titulo, corpo); err == nil {
			enviados++
		}
	}
	return s.db.Exec("UPDATE avisos_publicacao SET enviados = ? WHERE evento_id = ?", enviados, evento.ID).Error
}

func corpoAvisoEvento(nome, motivo string, evento *domain.Evento, onde, frontendURL, descadastro string) string {
	var b strings.Builder
	link := frontendURL + "/e/" + evento.Slug
	fmt.Fprintf(&b, `<div style="font-family:Arial,sans-serif;max-width:560px;margin:auto;color:#111">`)
	fmt.Fprintf(&b, `<p>Olá, %s!</p><p>%s Acabou de sair um evento novo:</p>`, html.EscapeString(primeiroNome(nome)), html.EscapeString(motivo))
	fmt.Fprintf(&b, `<h2 style="margin:16px 0 4px">%s</h2>`, html.EscapeString(evento.Titulo))
	fmt.Fprintf(&b, `<p style="margin:0 0 4px">%s</p>`, html.EscapeString(DataDoEvento(evento)))
	fmt.Fprintf(&b, `<p style="margin:0 0 16px;color:#555">%s</p>`, html.EscapeString(onde))
	fmt.Fprintf(&b, `<p><a href="%s" style="background:#111;color:#fff;padding:10px 16px;border-radius:8px;text-decoration:none">Ver evento</a></p>`, link)
	fmt.Fprintf(&b, `<p style="color:#777;font-size:12px;margin-top:24px">Você recebe este e-mail porque segue isso no %s. <a href="%s">Deixar de seguir</a>.</p></div>`, "Evve", descadastro)
	return b.String()
}

func primeiroNome(nome string) string {
	if p := strings.Fields(nome); len(p) > 0 {
		return p[0]
	}
	return "tudo bem"
}

// FusoDoEvento: o fuso cadastrado no evento (padrão Maceió).
func FusoDoEvento(e *domain.Evento) *time.Location {
	if e.Timezone != "" {
		if l, err := time.LoadLocation(e.Timezone); err == nil {
			return l
		}
	}
	if l, err := time.LoadLocation("America/Maceio"); err == nil {
		return l
	}
	return time.FixedZone("BRT", -3*3600)
}

// DataDoEvento: "sábado, 07/11/2026 às 20:00" no fuso do evento (o
// servidor roda em UTC; usar o fuso dele mostrava 23:00 para um evento às 20:00).
func DataDoEvento(e *domain.Evento) string {
	if e.InicioEm == nil {
		return ""
	}
	t := e.InicioEm.In(FusoDoEvento(e))
	dias := [...]string{"domingo", "segunda-feira", "terça-feira", "quarta-feira", "quinta-feira", "sexta-feira", "sábado"}
	return dias[t.Weekday()] + ", " + t.Format("02/01/2006 às 15:04")
}
