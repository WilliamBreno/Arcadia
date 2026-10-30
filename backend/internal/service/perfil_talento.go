package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/WilliamBreno/Arcadia/backend/internal/mail"
)

// Perfil público do talento (item 8.1). Regras do dono (30/10/2026):
//   - 18+: público quando quiser;
//   - 12 a 17: público só depois que o responsável autoriza pelo link no
//     e-mail dele; ele pode desautorizar a qualquer momento;
//   - menos de 12: sem perfil público;
//   - menor mostra só nome artístico, bio, foto (se autorizada),
//     personagens e colocações; nunca nome real, idade, cidade, redes,
//     contato; e não vai para o Google;
//   - os concursos aprovados entram sozinhos; o talento oculta o que quiser.

const (
	IdadeAdulto       = 18
	IdadeMinimaPerfil = 12
)

var (
	ErrPerfilInvalido      = errors.New("perfil inválido")
	ErrMenorDe12           = errors.New("perfis públicos são para quem tem 12 anos ou mais")
	ErrPerfilNaoEncontrado = errors.New("perfil não encontrado")
	ErrLinkAutorizacao     = errors.New("link de autorização inválido")
)

type PerfilTalento struct {
	ID                   int64 `gorm:"primaryKey"`
	UsuarioID            int64
	Slug                 string
	NomeArtistico        string
	Bio                  string
	FotoURL              string
	Cidade               string
	UF                   string `gorm:"column:uf"`
	Instagram            string
	TikTok               string `gorm:"column:tiktok"`
	YouTube              string `gorm:"column:youtube"`
	DataNascimento       time.Time
	Publico              bool
	ResponsavelNome      string
	ResponsavelEmail     string
	AutorizacaoTokenHash *string
	AutorizadoEm         *time.Time
	AutorizaFoto         bool
	CriadoEm             time.Time
	AtualizadoEm         time.Time
}

func (PerfilTalento) TableName() string { return "perfis_talento" }

// Idade em anos completos na data. Testada.
func Idade(nascimento, agora time.Time) int {
	anos := agora.Year() - nascimento.Year()
	if agora.Month() < nascimento.Month() || (agora.Month() == nascimento.Month() && agora.Day() < nascimento.Day()) {
		anos--
	}
	return anos
}

// SituacaoPerfil resume o que o perfil pode mostrar agora. Testada.
type SituacaoPerfil struct {
	Visivel    bool   `json:"visivel"`
	Menor      bool   `json:"menor"`
	Indexar    bool   `json:"indexar"`
	MostraFoto bool   `json:"mostra_foto"`
	Motivo     string `json:"motivo"` // por que não está visível (para o próprio talento)
}

func Situacao(p *PerfilTalento, nascimento time.Time, agora time.Time) SituacaoPerfil {
	idade := Idade(nascimento, agora)
	s := SituacaoPerfil{Menor: idade < IdadeAdulto}
	switch {
	case idade < IdadeMinimaPerfil:
		s.Motivo = "Perfis públicos são para quem tem 12 anos ou mais. Seu nome artístico continua aparecendo nos resultados dos concursos."
	case !p.Publico:
		s.Motivo = "Você deixou o perfil oculto."
	case idade < IdadeAdulto && p.AutorizadoEm == nil:
		s.Motivo = "Aguardando a autorização do responsável pelo link enviado ao e-mail dele."
	default:
		s.Visivel = true
	}
	s.Indexar = s.Visivel && !s.Menor
	s.MostraFoto = s.Visivel && p.FotoURL != "" && (!s.Menor || p.AutorizaFoto)
	return s
}

// --- Normalização ---

var (
	reHandle = regexp.MustCompile(`^[A-Za-z0-9._]{1,40}$`)
	reUF     = regexp.MustCompile(`^[A-Z]{2}$`)
)

// limparHandle aceita "@nome", "nome" ou o link do perfil e devolve "nome".
// Testada.
func limparHandle(s string) string {
	s = strings.TrimSpace(s)
	for _, pref := range []string{"https://", "http://", "www.", "instagram.com/", "tiktok.com/", "youtube.com/"} {
		if strings.HasPrefix(strings.ToLower(s), pref) {
			s = s[len(pref):]
		}
	}
	if i := strings.IndexAny(s, "?#"); i >= 0 {
		s = s[:i]
	}
	return strings.Trim(s, "@/ ")
}

// DadosPerfil é o que o talento envia.
type DadosPerfil struct {
	NomeArtistico    string `json:"nome_artistico"`
	Bio              string `json:"bio"`
	FotoURL          string `json:"foto_url"`
	Cidade           string `json:"cidade"`
	UF               string `json:"uf"`
	Instagram        string `json:"instagram"`
	TikTok           string `json:"tiktok"`
	YouTube          string `json:"youtube"`
	DataNascimento   string `json:"data_nascimento"` // AAAA-MM-DD
	Publico          *bool  `json:"publico"`
	ResponsavelNome  string `json:"responsavel_nome"`
	ResponsavelEmail string `json:"responsavel_email"`
}

// NormalizarPerfil valida e limpa. Para menor, campos que nunca aparecem
// (cidade e redes) são descartados em vez de guardados. Testada.
func NormalizarPerfil(d DadosPerfil, agora time.Time) (DadosPerfil, time.Time, error) {
	d.NomeArtistico = limpar(d.NomeArtistico)
	d.Bio = strings.TrimSpace(d.Bio)
	d.Cidade = limpar(d.Cidade)
	d.UF = strings.ToUpper(strings.TrimSpace(d.UF))
	d.ResponsavelNome = limpar(d.ResponsavelNome)
	d.ResponsavelEmail = NormalizarEmail(d.ResponsavelEmail)
	nasc, err := time.Parse("2006-01-02", strings.TrimSpace(d.DataNascimento))
	switch {
	case utf8.RuneCountInString(d.NomeArtistico) < 2 || utf8.RuneCountInString(d.NomeArtistico) > 60:
		return d, nasc, fmt.Errorf("%w: o nome artístico tem de 2 a 60 caracteres", ErrPerfilInvalido)
	case utf8.RuneCountInString(d.Bio) > 500:
		return d, nasc, fmt.Errorf("%w: a bio tem no máximo 500 caracteres", ErrPerfilInvalido)
	case err != nil || nasc.After(agora) || nasc.Before(agora.AddDate(-120, 0, 0)):
		return d, nasc, fmt.Errorf("%w: informe a data de nascimento", ErrPerfilInvalido)
	case d.FotoURL != "" && !strings.HasPrefix(d.FotoURL, "http"):
		return d, nasc, fmt.Errorf("%w: foto inválida", ErrPerfilInvalido)
	}
	idade := Idade(nasc, agora)
	if idade < IdadeMinimaPerfil {
		return d, nasc, ErrMenorDe12
	}
	if idade < IdadeAdulto {
		d.Cidade, d.UF, d.Instagram, d.TikTok, d.YouTube = "", "", "", "", ""
		if utf8.RuneCountInString(d.ResponsavelNome) < 3 || !strings.Contains(d.ResponsavelEmail, "@") || !strings.Contains(d.ResponsavelEmail, ".") {
			return d, nasc, fmt.Errorf("%w: informe o nome e o e-mail do responsável", ErrPerfilInvalido)
		}
		return d, nasc, nil
	}
	d.ResponsavelNome, d.ResponsavelEmail = "", ""
	if d.UF != "" && !reUF.MatchString(d.UF) {
		return d, nasc, fmt.Errorf("%w: UF com 2 letras", ErrPerfilInvalido)
	}
	for _, h := range []*string{&d.Instagram, &d.TikTok, &d.YouTube} {
		*h = limparHandle(*h)
		if *h != "" && !reHandle.MatchString(*h) {
			return d, nasc, fmt.Errorf("%w: use só o @ das redes (letras, números, ponto e _)", ErrPerfilInvalido)
		}
	}
	return d, nasc, nil
}

// --- Serviço ---

type PerfilTalentoService struct {
	db          *gorm.DB
	avaliacao   *AvaliacaoService
	mail        *mail.Cliente
	frontendURL string
	nome        string
}

func NovoPerfilTalentoService(db *gorm.DB, a *AvaliacaoService, m *mail.Cliente, frontendURL, nome string) *PerfilTalentoService {
	return &PerfilTalentoService{db: db, avaliacao: a, mail: m, frontendURL: strings.TrimRight(frontendURL, "/"), nome: nome}
}

// nascimentoEfetivo: a data mais nova entre a do perfil e as das fichas de
// concurso da pessoa (nunca trata como adulto quem declarou ser menor).
func (s *PerfilTalentoService) nascimentoEfetivo(p *PerfilTalento) time.Time {
	var maisNova *time.Time
	s.db.Table("fichas_participacao").Select("MAX(data_nascimento)").Where("usuario_id = ? AND data_nascimento IS NOT NULL", p.UsuarioID).Scan(&maisNova)
	if maisNova != nil && maisNova.After(p.DataNascimento) {
		return *maisNova
	}
	return p.DataNascimento
}

func (s *PerfilTalentoService) Meu(usuarioID int64) (*PerfilTalento, SituacaoPerfil, error) {
	var p PerfilTalento
	if err := s.db.Where("usuario_id = ?", usuarioID).First(&p).Error; err != nil {
		return nil, SituacaoPerfil{}, ErrPerfilNaoEncontrado
	}
	return &p, Situacao(&p, s.nascimentoEfetivo(&p), time.Now()), nil
}

func (s *PerfilTalentoService) slugLivre(base string, usuarioID int64) string {
	base = Slugify(base)
	if base == "" {
		base = "talento"
	}
	slug := base
	for i := 2; ; i++ {
		var n int64
		s.db.Model(&PerfilTalento{}).Where("slug = ? AND usuario_id <> ?", slug, usuarioID).Count(&n)
		if n == 0 {
			return slug
		}
		slug = fmt.Sprintf("%s-%d", base, i)
	}
}

func novoTokenAutorizacao() (plano, hash string) {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	plano = hex.EncodeToString(b)
	h := sha256.Sum256([]byte(plano))
	return plano, hex.EncodeToString(h[:])
}

// Salvar cria ou atualiza o perfil do próprio usuário. Se o responsável
// mudou (ou é a primeira vez), a autorização zera e sai um e-mail novo.
func (s *PerfilTalentoService) Salvar(usuarioID int64, d DadosPerfil) (*PerfilTalento, SituacaoPerfil, error) {
	agora := time.Now()
	d, nasc, err := NormalizarPerfil(d, agora)
	if err != nil {
		return nil, SituacaoPerfil{}, err
	}
	var p PerfilTalento
	novo := s.db.Where("usuario_id = ?", usuarioID).First(&p).Error != nil
	if novo {
		p = PerfilTalento{UsuarioID: usuarioID, CriadoEm: agora, Publico: true}
	}
	if d.Publico != nil {
		p.Publico = *d.Publico
	}
	if novo || Slugify(p.NomeArtistico) != Slugify(d.NomeArtistico) {
		p.Slug = s.slugLivre(d.NomeArtistico, usuarioID)
	}
	trocouResponsavel := p.ResponsavelEmail != d.ResponsavelEmail
	p.NomeArtistico, p.Bio, p.FotoURL = d.NomeArtistico, d.Bio, d.FotoURL
	p.Cidade, p.UF, p.Instagram, p.TikTok, p.YouTube = d.Cidade, d.UF, d.Instagram, d.TikTok, d.YouTube
	p.DataNascimento, p.ResponsavelNome, p.ResponsavelEmail, p.AtualizadoEm = nasc, d.ResponsavelNome, d.ResponsavelEmail, agora

	var tokenPlano string
	menor := Idade(s.nascimentoEfetivo(&p), agora) < IdadeAdulto
	if menor && (trocouResponsavel || p.AutorizacaoTokenHash == nil) && p.ResponsavelEmail != "" {
		var hash string
		tokenPlano, hash = novoTokenAutorizacao()
		p.AutorizacaoTokenHash, p.AutorizadoEm, p.AutorizaFoto = &hash, nil, false
	}
	if !menor {
		p.AutorizacaoTokenHash, p.AutorizadoEm, p.AutorizaFoto = nil, nil, false
	}
	if err := s.db.Save(&p).Error; err != nil {
		return nil, SituacaoPerfil{}, err
	}
	if tokenPlano != "" {
		s.pedirAutorizacao(&p, tokenPlano)
	}
	return &p, Situacao(&p, s.nascimentoEfetivo(&p), agora), nil
}

func (s *PerfilTalentoService) pedirAutorizacao(p *PerfilTalento, token string) {
	link := s.frontendURL + "/autorizar-perfil?t=" + token
	corpo := fmt.Sprintf(`<div style="font-family:Arial,sans-serif;max-width:560px;margin:auto;color:#111">`+
		`<p>Olá, %s.</p>`+
		`<p>Um(a) jovem pelo qual você é responsável quer criar um perfil público de talento no %s com o nome artístico <strong>%s</strong>, para mostrar personagens e colocações em concursos.</p>`+
		`<p>O perfil só fica visível se você autorizar. Ele nunca mostra nome real, idade, cidade, redes sociais ou contato, e não aparece no Google. Você escolhe se a foto aparece, e pode desautorizar quando quiser pelo mesmo link.</p>`+
		`<p><a href="%s" style="background:#111;color:#fff;padding:10px 16px;border-radius:8px;text-decoration:none">Ver e decidir</a></p>`+
		`<p style="color:#777;font-size:12px">Se você não reconhece este pedido, ignore este e-mail: nada fica público sem a sua autorização.</p></div>`,
		html.EscapeString(primeiroNome(p.ResponsavelNome)), html.EscapeString(s.nome), html.EscapeString(p.NomeArtistico), link)
	_ = s.mail.Enviar(p.ResponsavelEmail, "Autorização de perfil público — "+p.NomeArtistico, corpo)
}

func hashToken(t string) string {
	h := sha256.Sum256([]byte(strings.TrimSpace(t)))
	return hex.EncodeToString(h[:])
}

// PedidoAutorizacao é o que o responsável vê antes de decidir.
type PedidoAutorizacao struct {
	NomeArtistico string `json:"nome_artistico"`
	Bio           string `json:"bio"`
	TemFoto       bool   `json:"tem_foto"`
	Autorizado    bool   `json:"autorizado"`
	AutorizaFoto  bool   `json:"autoriza_foto"`
	Responsavel   string `json:"responsavel"`
}

func (s *PerfilTalentoService) perfilDoToken(token string) (*PerfilTalento, error) {
	var p PerfilTalento
	if len(strings.TrimSpace(token)) < 20 || s.db.Where("autorizacao_token_hash = ?", hashToken(token)).First(&p).Error != nil {
		return nil, ErrLinkAutorizacao
	}
	return &p, nil
}

func (s *PerfilTalentoService) VerAutorizacao(token string) (*PedidoAutorizacao, error) {
	p, err := s.perfilDoToken(token)
	if err != nil {
		return nil, err
	}
	return &PedidoAutorizacao{NomeArtistico: p.NomeArtistico, Bio: p.Bio, TemFoto: p.FotoURL != "", Autorizado: p.AutorizadoEm != nil,
		AutorizaFoto: p.AutorizaFoto, Responsavel: p.ResponsavelNome}, nil
}

// Decidir: o responsável autoriza (com ou sem foto) ou desautoriza.
func (s *PerfilTalentoService) Decidir(token string, autorizar, foto bool) (*PedidoAutorizacao, error) {
	p, err := s.perfilDoToken(token)
	if err != nil {
		return nil, err
	}
	upd := map[string]any{"autoriza_foto": autorizar && foto, "atualizado_em": time.Now()}
	if autorizar {
		upd["autorizado_em"] = time.Now()
	} else {
		upd["autorizado_em"] = nil
	}
	if err := s.db.Model(&PerfilTalento{}).Where("id = ?", p.ID).Updates(upd).Error; err != nil {
		return nil, err
	}
	return s.VerAutorizacao(token)
}

// --- Participações ---

type Participacao struct {
	FichaID      int64      `json:"ficha_id"`
	EventoTitulo string     `json:"evento_titulo"`
	EventoSlug   string     `json:"evento_slug"`
	Data         *time.Time `json:"data"`
	Tipo         string     `json:"tipo"`
	Personagem   string     `json:"personagem,omitempty"`
	Obra         string     `json:"obra,omitempty"`
	Posicao      int        `json:"posicao,omitempty"`
	Total        int        `json:"total,omitempty"`
	Oculta       bool       `json:"oculta,omitempty"`
}

// participacoes: concursos aprovados do talento (todos para ele; só os não
// ocultos em público), com a colocação quando o resultado foi liberado.
func (s *PerfilTalentoService) participacoes(usuarioID int64, incluirOcultas bool) ([]Participacao, error) {
	var fichas []domain.FichaParticipacao
	q := s.db.Joins("JOIN eventos ON eventos.id = fichas_participacao.evento_id").
		Where("fichas_participacao.usuario_id = ? AND fichas_participacao.papel = ? AND fichas_participacao.status = ? AND eventos.status <> ?",
			usuarioID, domain.PapelParticipante, domain.StatusFichaAprovado, domain.StatusEventoCancelado)
	if !incluirOcultas {
		q = q.Where("NOT fichas_participacao.oculta_no_perfil")
	}
	if err := q.Order("eventos.inicio_em DESC").Limit(60).Find(&fichas).Error; err != nil {
		return nil, err
	}
	var ocultas map[int64]bool
	if incluirOcultas {
		ocultas = map[int64]bool{}
		var ids []int64
		s.db.Table("fichas_participacao").Where("usuario_id = ? AND oculta_no_perfil", usuarioID).Pluck("id", &ids)
		for _, id := range ids {
			ocultas[id] = true
		}
	}
	out := make([]Participacao, 0, len(fichas))
	for i := range fichas {
		f := &fichas[i]
		var e domain.Evento
		if s.db.First(&e, f.EventoID).Error != nil {
			continue
		}
		var dados map[string]any
		_ = json.Unmarshal(f.Dados, &dados)
		texto := func(k string) string {
			v, _ := dados[k].(string)
			return limpar(v)
		}
		p := Participacao{FichaID: f.ID, EventoTitulo: e.Titulo, EventoSlug: e.Slug, Data: e.InicioEm, Tipo: tipoDaFicha(f),
			Personagem: texto("personagem"), Obra: texto("obra_origem"), Oculta: ocultas[f.ID]}
		if s.avaliacao != nil && e.ResultadoLiberadoEm != nil {
			if ranking, err := s.avaliacao.ResultadoPublico(&e); err == nil {
				for _, linhas := range ranking {
					for _, l := range linhas {
						if l.FichaID == f.ID {
							p.Posicao, p.Total = l.Posicao, len(linhas)
						}
					}
				}
			}
		}
		out = append(out, p)
	}
	return out, nil
}

func (s *PerfilTalentoService) MinhasParticipacoes(usuarioID int64) ([]Participacao, error) {
	return s.participacoes(usuarioID, true)
}

func (s *PerfilTalentoService) Ocultar(usuarioID, fichaID int64, ocultar bool) error {
	res := s.db.Table("fichas_participacao").Where("id = ? AND usuario_id = ?", fichaID, usuarioID).Update("oculta_no_perfil", ocultar)
	if res.Error == nil && res.RowsAffected == 0 {
		return ErrPerfilNaoEncontrado
	}
	return res.Error
}

// PerfilPublico é o que a página /t/:slug recebe: só o que pode aparecer.
type PerfilPublico struct {
	Slug          string         `json:"slug"`
	NomeArtistico string         `json:"nome_artistico"`
	Bio           string         `json:"bio"`
	FotoURL       string         `json:"foto_url,omitempty"`
	Cidade        string         `json:"cidade,omitempty"`
	UF            string         `json:"uf,omitempty"`
	Instagram     string         `json:"instagram,omitempty"`
	TikTok        string         `json:"tiktok,omitempty"`
	YouTube       string         `json:"youtube,omitempty"`
	Indexar       bool           `json:"indexar"`
	Participacoes []Participacao `json:"participacoes"`
	// Item 8.7: fotos da galeria em que o talento (adulto) se marcou.
	Fotos []FotoDoPerfil `json:"fotos,omitempty"`
}

// Montar aplica as regras de menor. Testada.
func Montar(p *PerfilTalento, sit SituacaoPerfil, parts []Participacao) PerfilPublico {
	out := PerfilPublico{Slug: p.Slug, NomeArtistico: p.NomeArtistico, Bio: p.Bio, Indexar: sit.Indexar, Participacoes: parts}
	if sit.MostraFoto {
		out.FotoURL = p.FotoURL
	}
	if !sit.Menor {
		out.Cidade, out.UF, out.Instagram, out.TikTok, out.YouTube = p.Cidade, p.UF, p.Instagram, p.TikTok, p.YouTube
	}
	if out.Participacoes == nil {
		out.Participacoes = []Participacao{}
	}
	return out
}

func (s *PerfilTalentoService) Publico(slug string) (*PerfilPublico, error) {
	var p PerfilTalento
	if s.db.Where("slug = ?", strings.ToLower(strings.TrimSpace(slug))).First(&p).Error != nil {
		return nil, ErrPerfilNaoEncontrado
	}
	sit := Situacao(&p, s.nascimentoEfetivo(&p), time.Now())
	if !sit.Visivel {
		return nil, ErrPerfilNaoEncontrado
	}
	parts, err := s.participacoes(p.UsuarioID, false)
	if err != nil {
		return nil, err
	}
	out := Montar(&p, sit, parts)
	if !sit.Menor {
		out.Fotos = FotosDoTalento(s.db, p.UsuarioID)
	}
	return &out, nil
}

// SlugsVisiveis: para ligar o nome no resultado do concurso ao perfil.
func (s *PerfilTalentoService) SlugsVisiveis(usuarioIDs []int64) map[int64]string {
	out := map[int64]string{}
	if len(usuarioIDs) == 0 {
		return out
	}
	var ps []PerfilTalento
	s.db.Where("usuario_id IN ?", usuarioIDs).Find(&ps)
	agora := time.Now()
	for i := range ps {
		if Situacao(&ps[i], s.nascimentoEfetivo(&ps[i]), agora).Visivel {
			out[ps[i].UsuarioID] = ps[i].Slug
		}
	}
	return out
}

// ParaSitemap: perfis visíveis de adultos (os de menores não vão ao Google).
func (s *PerfilTalentoService) ParaSitemap() []string {
	var ps []PerfilTalento
	s.db.Where("publico").Order("id").Limit(5000).Find(&ps)
	agora := time.Now()
	var out []string
	for i := range ps {
		if Situacao(&ps[i], s.nascimentoEfetivo(&ps[i]), agora).Indexar {
			out = append(out, ps[i].Slug)
		}
	}
	return out
}
