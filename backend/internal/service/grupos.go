package service

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/WilliamBreno/Arcadia/backend/internal/mail"
)

// Item 11.1: grupos (dança, cosplay, canto, teatro) e artistas solo, com
// integrantes que aceitam com a própria conta, portfólio, selo do
// Instagram, resultados de concursos e Atrações de eventos.

type Grupo struct {
	ID                    int64 `gorm:"primaryKey"`
	Slug                  string
	Nome                  string
	Tipo                  string
	Solo                  bool
	Bio                   string
	Cidade                string
	UF                    string `gorm:"column:uf"`
	Instagram             string
	TikTok                string `gorm:"column:tiktok"`
	YouTube               string `gorm:"column:youtube"`
	LogoURL               string
	CapaURL               string
	LiderUsuarioID        int64
	Publico               bool
	InstagramCodigo       string
	InstagramPedidoEm     *time.Time
	InstagramVerificadoEm *time.Time
	CriadoEm              time.Time
	AtualizadoEm          time.Time
}

func (Grupo) TableName() string { return "grupos" }

type GrupoIntegrante struct {
	ID                   int64 `gorm:"primaryKey"`
	GrupoID              int64
	UsuarioID            *int64
	Email                string
	ConviteTokenHash     *string
	NomeArtistico        string
	Funcao               string
	DesdeAno             *int
	FotoURL              string
	Instagram            string
	DataNascimento       *time.Time
	ResponsavelNome      string
	ResponsavelEmail     string
	AutorizacaoTokenHash *string
	AutorizadoEm         *time.Time
	AutorizaFoto         bool
	Status               string
	Ordem                int
	CriadoEm             time.Time
	AceitoEm             *time.Time
}

func (GrupoIntegrante) TableName() string { return "grupo_integrantes" }

func (i *GrupoIntegrante) situacao(agora time.Time) SituacaoIntegrante {
	return CalcularSituacaoIntegrante(i.Status, i.DataNascimento, i.AutorizadoEm != nil, i.AutorizaFoto, i.FotoURL != "", agora)
}

type GrupoMidia struct {
	ID       int64 `gorm:"primaryKey"`
	GrupoID  int64
	Tipo     string
	URL      string `gorm:"column:url"`
	Legenda  string
	EventoID *int64
	Ordem    int
	CriadoEm time.Time
}

func (GrupoMidia) TableName() string { return "grupo_midias" }

type Atracao struct {
	ID           int64 `gorm:"primaryKey"`
	EventoID     int64
	GrupoID      int64
	Status       string
	ConvidadoPor *int64
	CriadoEm     time.Time
	DecididoEm   *time.Time
}

func (Atracao) TableName() string { return "atracoes" }

type GruposService struct {
	db          *gorm.DB
	mail        *mail.Cliente
	avaliacao   *AvaliacaoService
	eventos     *EventoService
	frontendURL string
	nome        string
}

func NovoGruposService(db *gorm.DB, m *mail.Cliente, a *AvaliacaoService, eventos *EventoService, frontendURL, nome string) *GruposService {
	return &GruposService{db: db, mail: m, avaliacao: a, eventos: eventos, frontendURL: strings.TrimRight(frontendURL, "/"), nome: nome}
}

func (s *GruposService) envelope(conteudo string) string {
	return `<div style="font-family:Arial,sans-serif;max-width:560px;margin:auto;color:#111">` + conteudo + `</div>`
}

func (s *GruposService) botao(link, texto string) string {
	return fmt.Sprintf(`<p><a href="%s" style="background:#111;color:#fff;padding:10px 16px;border-radius:8px;text-decoration:none">%s</a></p>`, link, html.EscapeString(texto))
}

// --- Grupo ---

func (s *GruposService) slugLivre(base string, excetoID int64) string {
	base = Slugify(base)
	if base == "" {
		base = "artista"
	}
	if len(base) > 80 {
		base = strings.Trim(base[:80], "-")
	}
	slug := base
	for i := 2; ; i++ {
		var n int64
		s.db.Model(&Grupo{}).Where("slug = ? AND id <> ?", slug, excetoID).Count(&n)
		if n == 0 {
			return slug
		}
		slug = fmt.Sprintf("%s-%d", base, i)
	}
}

func (s *GruposService) aplicar(g *Grupo, d DadosGrupo) {
	trocouInsta := g.Instagram != d.Instagram
	g.Nome, g.Tipo, g.Bio, g.Cidade, g.UF = d.Nome, d.Tipo, d.Bio, d.Cidade, d.UF
	g.Instagram, g.TikTok, g.YouTube, g.LogoURL, g.CapaURL = d.Instagram, d.TikTok, d.YouTube, d.LogoURL, d.CapaURL
	if d.Publico != nil {
		g.Publico = *d.Publico
	}
	if trocouInsta {
		// Outro @: o selo e o pedido valem só para o anterior.
		g.InstagramVerificadoEm, g.InstagramPedidoEm = nil, nil
	}
}

// Criar: quem cria vira o líder. No perfil solo, a própria pessoa entra
// como integrante (completa os dados ao aceitar).
func (s *GruposService) Criar(usuarioID int64, d DadosGrupo) (*Grupo, error) {
	d, err := NormalizarGrupo(d)
	if err != nil {
		return nil, err
	}
	var n int64
	s.db.Model(&Grupo{}).Where("lider_usuario_id = ?", usuarioID).Count(&n)
	if n >= MaxGruposPorLider {
		return nil, ErrLimiteArtistas
	}
	agora := time.Now()
	g := &Grupo{Slug: s.slugLivre(d.Nome, 0), Solo: d.Solo, LiderUsuarioID: usuarioID, Publico: true, CriadoEm: agora, AtualizadoEm: agora}
	s.aplicar(g, d)
	if err := s.db.Create(g).Error; err != nil {
		return nil, err
	}
	return g, nil
}

func (s *GruposService) doLider(usuarioID, grupoID int64) (*Grupo, error) {
	var g Grupo
	if err := s.db.First(&g, grupoID).Error; err != nil {
		return nil, ErrArtistaNaoEncontrado
	}
	if g.LiderUsuarioID != usuarioID {
		return nil, ErrArtistaSemPermissao
	}
	return &g, nil
}

func (s *GruposService) Atualizar(usuarioID, grupoID int64, d DadosGrupo) (*Grupo, error) {
	g, err := s.doLider(usuarioID, grupoID)
	if err != nil {
		return nil, err
	}
	d, err = NormalizarGrupo(d)
	if err != nil {
		return nil, err
	}
	if Slugify(d.Nome) != Slugify(g.Nome) {
		g.Slug = s.slugLivre(d.Nome, g.ID)
	}
	d.Solo = g.Solo // não muda depois de criado
	s.aplicar(g, d)
	g.AtualizadoEm = time.Now()
	return g, s.db.Save(g).Error
}

// MeuGrupoResumo aparece em "Meus grupos".
type MeuGrupoResumo struct {
	ID                int64  `json:"id"`
	Slug              string `json:"slug"`
	Nome              string `json:"nome"`
	Tipo              string `json:"tipo"`
	Solo              bool   `json:"solo"`
	LogoURL           string `json:"logo_url"`
	Lider             bool   `json:"lider"`
	MeuStatus         string `json:"meu_status,omitempty"` // integrante: ativo | convidado
	Integrantes       int    `json:"integrantes"`
	AtracoesPendentes int    `json:"atracoes_pendentes"`
}

func (s *GruposService) MeusGrupos(usuarioID int64) ([]MeuGrupoResumo, error) {
	var usuario domain.Usuario
	if err := s.db.First(&usuario, usuarioID).Error; err != nil {
		return nil, err
	}
	var linhas []MeuGrupoResumo
	err := s.db.Raw(`
		SELECT g.id, g.slug, g.nome, g.tipo, g.solo, g.logo_url, (g.lider_usuario_id = ?) AS lider,
		       COALESCE((SELECT gi.status FROM grupo_integrantes gi WHERE gi.grupo_id = g.id AND (gi.usuario_id = ? OR gi.email = ?) AND gi.status IN ('ativo','convidado') LIMIT 1), '') AS meu_status,
		       (SELECT count(*) FROM grupo_integrantes gi WHERE gi.grupo_id = g.id AND gi.status = 'ativo') AS integrantes,
		       (SELECT count(*) FROM atracoes a WHERE a.grupo_id = g.id AND a.status = 'convidado') AS atracoes_pendentes
		FROM grupos g
		WHERE g.lider_usuario_id = ?
		   OR EXISTS (SELECT 1 FROM grupo_integrantes gi WHERE gi.grupo_id = g.id AND gi.usuario_id = ? AND gi.status = 'ativo')
		ORDER BY lider DESC, g.nome`, usuarioID, usuarioID, NormalizarEmail(usuario.Email), usuarioID, usuarioID).Scan(&linhas).Error
	if linhas == nil {
		linhas = []MeuGrupoResumo{}
	}
	for i := range linhas {
		if !linhas[i].Lider {
			linhas[i].AtracoesPendentes = 0
		}
	}
	return linhas, err
}

// IntegranteVisao é o integrante como o líder vê (painel).
type IntegranteVisao struct {
	ID            int64  `json:"id"`
	Email         string `json:"email"`
	NomeArtistico string `json:"nome_artistico"`
	Funcao        string `json:"funcao"`
	DesdeAno      *int   `json:"desde_ano"`
	Anos          int    `json:"anos"`
	FotoURL       string `json:"foto_url"`
	Status        string `json:"status"`
	Visivel       bool   `json:"visivel"`
	Menor         bool   `json:"menor"`
	Pendente      string `json:"pendente,omitempty"`
	SouEu         bool   `json:"sou_eu"`
}

type MidiaVisao struct {
	ID           int64  `json:"id"`
	Tipo         string `json:"tipo"`
	URL          string `json:"url"`
	Embed        string `json:"embed,omitempty"`
	Plataforma   string `json:"plataforma,omitempty"`
	Legenda      string `json:"legenda"`
	EventoSlug   string `json:"evento_slug,omitempty"`
	EventoTitulo string `json:"evento_titulo,omitempty"`
}

type AtracaoVisao struct {
	ID           int64      `json:"id"`
	Status       string     `json:"status"`
	EventoSlug   string     `json:"evento_slug"`
	EventoTitulo string     `json:"evento_titulo"`
	EventoInicio *time.Time `json:"evento_inicio"`
	Cidade       string     `json:"cidade"`
	UF           string     `json:"uf"`
}

type GrupoPainel struct {
	Grupo               *Grupo            `json:"-"`
	Integrantes         []IntegranteVisao `json:"integrantes"`
	Midias              []MidiaVisao      `json:"midias"`
	Atracoes            []AtracaoVisao    `json:"atracoes"`
	InstagramCodigo     string            `json:"instagram_codigo"`
	InstagramPedido     bool              `json:"instagram_pedido"`
	InstagramVerificado bool              `json:"instagram_verificado"`
}

func (s *GruposService) midias(grupoID int64) []MidiaVisao {
	var ms []GrupoMidia
	s.db.Where("grupo_id = ?", grupoID).Order("ordem, id").Find(&ms)
	out := make([]MidiaVisao, 0, len(ms))
	for _, m := range ms {
		v := MidiaVisao{ID: m.ID, Tipo: m.Tipo, URL: m.URL, Legenda: m.Legenda}
		if m.Tipo == "video" {
			v.Embed, v.Plataforma = EmbedDoVideo(m.URL)
		}
		if m.EventoID != nil {
			var e domain.Evento
			if s.db.Select("slug, titulo").Where("status = ? AND visibilidade = ?", domain.StatusEventoPublicado, domain.VisibilidadePublico).First(&e, *m.EventoID).Error == nil {
				v.EventoSlug, v.EventoTitulo = e.Slug, e.Titulo
			}
		}
		out = append(out, v)
	}
	return out
}

func (s *GruposService) atracoes(grupoID int64, status []string, soPublicos bool) []AtracaoVisao {
	var out []AtracaoVisao
	s.db.Raw(`SELECT a.id, a.status, e.slug AS evento_slug, e.titulo AS evento_titulo, e.inicio_em AS evento_inicio,
	                 COALESCE(l.cidade, '') AS cidade, COALESCE(l.uf, '') AS uf
	          FROM atracoes a JOIN eventos e ON e.id = a.evento_id LEFT JOIN locais l ON l.id = e.local_id
	          WHERE a.grupo_id = ? AND a.status IN ? AND e.status <> 'cancelado'
	            AND (NOT ? OR (e.status = 'publicado' AND e.visibilidade = 'publico'))
	          ORDER BY e.inicio_em DESC NULLS LAST`, grupoID, status, soPublicos).Scan(&out)
	if out == nil {
		out = []AtracaoVisao{}
	}
	return out
}

// Painel: tudo o que o líder gerencia.
func (s *GruposService) Painel(usuarioID, grupoID int64) (*GrupoPainel, error) {
	g, err := s.doLider(usuarioID, grupoID)
	if err != nil {
		return nil, err
	}
	agora := time.Now()
	var is []GrupoIntegrante
	s.db.Where("grupo_id = ? AND status IN ('convidado','ativo')", g.ID).Order("ordem, id").Find(&is)
	p := &GrupoPainel{Grupo: g, Integrantes: []IntegranteVisao{}, Midias: s.midias(g.ID), Atracoes: s.atracoes(g.ID, []string{"convidado", "confirmado", "recusado"}, false),
		InstagramCodigo: g.InstagramCodigo, InstagramPedido: g.InstagramPedidoEm != nil, InstagramVerificado: g.InstagramVerificadoEm != nil}
	s.ajustarNascimentos(is)
	for i := range is {
		x := &is[i]
		sit := x.situacao(agora)
		foto := x.FotoURL
		if sit.Menor && !sit.MostraFoto {
			foto = "" // sem a autorização do responsável, nem o líder vê
		}
		p.Integrantes = append(p.Integrantes, IntegranteVisao{ID: x.ID, Email: x.Email, NomeArtistico: x.NomeArtistico, Funcao: x.Funcao, DesdeAno: x.DesdeAno,
			Anos: AnosDeExperiencia(x.DesdeAno, agora), FotoURL: foto, Status: x.Status, Visivel: sit.Visivel, Menor: sit.Menor, Pendente: sit.Pendente,
			SouEu: x.UsuarioID != nil && *x.UsuarioID == usuarioID})
	}
	return p, nil
}

// --- Idade (regra do item 8.1) ---

// nascimentosConhecidos: a data de nascimento mais nova que cada pessoa já
// declarou no perfil de talento e nas fichas de concurso. Nunca tratamos
// como adulto quem já declarou ser menor em outro lugar.
func (s *GruposService) nascimentosConhecidos(ids []int64) map[int64]time.Time {
	m := map[int64]time.Time{}
	if len(ids) == 0 {
		return m
	}
	var linhas []struct {
		UsuarioID int64
		Nasc      time.Time
	}
	s.db.Raw(`SELECT usuario_id, max(d) AS nasc FROM (
		SELECT usuario_id, data_nascimento::date AS d FROM perfis_talento WHERE usuario_id IN ?
		UNION ALL
		SELECT usuario_id, data_nascimento::date AS d FROM fichas_participacao WHERE usuario_id IN ? AND data_nascimento IS NOT NULL
	) x GROUP BY usuario_id`, ids, ids).Scan(&linhas)
	for _, l := range linhas {
		m[l.UsuarioID] = l.Nasc
	}
	return m
}

// ajustarNascimentos troca, em memória, a data do integrante pela mais nova conhecida.
func (s *GruposService) ajustarNascimentos(is []GrupoIntegrante) {
	ids := []int64{}
	for _, i := range is {
		if i.UsuarioID != nil {
			ids = append(ids, *i.UsuarioID)
		}
	}
	conhecidos := s.nascimentosConhecidos(ids)
	for k := range is {
		if is[k].UsuarioID == nil {
			continue
		}
		if n, ok := conhecidos[*is[k].UsuarioID]; ok && (is[k].DataNascimento == nil || n.After(*is[k].DataNascimento)) {
			nn := n
			is[k].DataNascimento = &nn
		}
	}
}

// situacaoSolo: o perfil solo segue a situação da própria pessoa. Sem o
// aceite dela (com a data de nascimento), o perfil não aparece.
func (s *GruposService) situacaoSolo(g *Grupo, agora time.Time) SituacaoIntegrante {
	var is []GrupoIntegrante
	s.db.Where("grupo_id = ? AND usuario_id = ? AND status = 'ativo'", g.ID, g.LiderUsuarioID).Limit(1).Find(&is)
	if len(is) == 0 {
		return SituacaoIntegrante{Pendente: "Confirme os seus dados para o perfil aparecer."}
	}
	s.ajustarNascimentos(is)
	return is[0].situacao(agora)
}

// condSoloVisivel (SQL, alias g): perfil de grupo, ou solo com a pessoa
// adulta, ou de 12 a 17 com a autorização do responsável.
const condSoloVisivel = `(NOT g.solo OR EXISTS (
	SELECT 1 FROM grupo_integrantes gi
	WHERE gi.grupo_id = g.id AND gi.usuario_id = g.lider_usuario_id AND gi.status = 'ativo' AND gi.data_nascimento IS NOT NULL
	  AND (
	    GREATEST(gi.data_nascimento, (SELECT max(x.d) FROM (
	        SELECT pt.data_nascimento AS d FROM perfis_talento pt WHERE pt.usuario_id = gi.usuario_id
	        UNION ALL SELECT f.data_nascimento::date FROM fichas_participacao f WHERE f.usuario_id = gi.usuario_id) x)) <= CURRENT_DATE - interval '18 years'
	    OR (gi.autorizado_em IS NOT NULL AND
	    GREATEST(gi.data_nascimento, (SELECT max(x.d) FROM (
	        SELECT pt.data_nascimento AS d FROM perfis_talento pt WHERE pt.usuario_id = gi.usuario_id
	        UNION ALL SELECT f.data_nascimento::date FROM fichas_participacao f WHERE f.usuario_id = gi.usuario_id) x)) <= CURRENT_DATE - interval '12 years')
	  )))`

// limparSoloMenor: perfil solo de menor não mostra redes, cidade nem
// imagens sem a autorização da foto.
func (s *GruposService) limparSoloMenor(id, lider int64, solo bool, insta, cidade, uf, logo, capa *string) {
	if !solo {
		return
	}
	sit := s.situacaoSolo(&Grupo{ID: id, LiderUsuarioID: lider}, time.Now())
	if !sit.Menor {
		return
	}
	*insta, *cidade, *uf = "", "", ""
	if !sit.MostraFoto {
		*logo, *capa = "", ""
	}
}

// --- Integrantes ---

func novoTokenGrupo() (plano, hash string) { return novoTokenAutorizacao() }

// ConvidarIntegrante: o líder convida pelo e-mail. Se for o próprio
// e-mail do líder, devolve o token para ele aceitar na hora.
func (s *GruposService) ConvidarIntegrante(usuarioID, grupoID int64, d DadosIntegrante) (*GrupoIntegrante, string, error) {
	g, err := s.doLider(usuarioID, grupoID)
	if err != nil {
		return nil, "", err
	}
	d, err = NormalizarIntegrante(d, time.Now())
	if err != nil {
		return nil, "", err
	}
	var lider domain.Usuario
	if err := s.db.First(&lider, usuarioID).Error; err != nil {
		return nil, "", err
	}
	proprio := NormalizarEmail(lider.Email) == d.Email
	if g.Solo && !proprio {
		return nil, "", ErrSoloUmIntegrante
	}
	var ativos int64
	s.db.Model(&GrupoIntegrante{}).Where("grupo_id = ? AND status IN ('convidado','ativo')", g.ID).Count(&ativos)
	if ativos >= MaxIntegrantesGrupo {
		return nil, "", ErrLimiteIntegrantes
	}
	plano, hash := novoTokenGrupo()
	var existente GrupoIntegrante
	if s.db.Where("grupo_id = ? AND email = ?", g.ID, d.Email).First(&existente).Error == nil {
		if existente.Status == "ativo" || existente.Status == "convidado" {
			return nil, "", ErrIntegranteDuplicado
		}
		// Quem saiu, recusou ou foi removido pode ser convidado de novo.
		existente.NomeArtistico, existente.Funcao, existente.DesdeAno = d.NomeArtistico, d.Funcao, d.DesdeAno
		existente.Status, existente.ConviteTokenHash, existente.UsuarioID, existente.AceitoEm = "convidado", &hash, nil, nil
		if err := s.db.Save(&existente).Error; err != nil {
			return nil, "", err
		}
		s.enviarConvite(g, &existente, plano, proprio)
		return &existente, s.tokenParaLider(proprio, plano), nil
	}
	i := &GrupoIntegrante{GrupoID: g.ID, Email: d.Email, ConviteTokenHash: &hash, NomeArtistico: d.NomeArtistico, Funcao: d.Funcao,
		DesdeAno: d.DesdeAno, Status: "convidado", Ordem: int(ativos), CriadoEm: time.Now()}
	if err := s.db.Create(i).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate") {
			return nil, "", ErrIntegranteDuplicado
		}
		return nil, "", err
	}
	s.enviarConvite(g, i, plano, proprio)
	return i, s.tokenParaLider(proprio, plano), nil
}

func (s *GruposService) tokenParaLider(proprio bool, plano string) string {
	if proprio {
		return plano
	}
	return ""
}

func (s *GruposService) enviarConvite(g *Grupo, i *GrupoIntegrante, token string, proprio bool) {
	if proprio || s.mail == nil {
		return
	}
	link := s.frontendURL + "/grupo-convite/" + token
	corpo := s.envelope(fmt.Sprintf(`<p>Olá, %s!</p><p>O grupo <strong>%s</strong> quer mostrar você como integrante no perfil público dele no %s, como <strong>%s</strong>.</p>`+
		`<p>Você só aparece se aceitar. Antes de aceitar, você vê exatamente o que fica público e pode sair quando quiser.</p>%s`+
		`<p style="color:#777;font-size:12px">Para aceitar, entre com a conta Evve deste e-mail (ou crie uma). Se não conhece o grupo, ignore este e-mail.</p>`,
		html.EscapeString(primeiroNome(i.NomeArtistico)), html.EscapeString(g.Nome), html.EscapeString(s.nome), html.EscapeString(textoOu(i.Funcao, "integrante")),
		s.botao(link, "Ver o convite")))
	_ = s.mail.Enviar(i.Email, "Convite para o perfil de "+g.Nome, corpo)
}

func textoOu(s, padrao string) string {
	if strings.TrimSpace(s) == "" {
		return padrao
	}
	return s
}

// ConviteVisao é o que a pessoa vê antes de aceitar.
type ConviteVisao struct {
	GrupoNome     string `json:"grupo_nome"`
	GrupoTipo     string `json:"grupo_tipo"`
	GrupoSlug     string `json:"grupo_slug"`
	Solo          bool   `json:"solo"`
	NomeArtistico string `json:"nome_artistico"`
	Funcao        string `json:"funcao"`
	DesdeAno      *int   `json:"desde_ano"`
	EmailConvite  string `json:"email_convite"`             // mascarado
	Nascimento    string `json:"data_nascimento,omitempty"` // do perfil de talento, se houver
}

func mascararEmail(e string) string {
	at := strings.Index(e, "@")
	if at <= 1 {
		return e
	}
	return e[:1] + strings.Repeat("•", min(at-1, 6)) + e[at:]
}

func (s *GruposService) integranteDoConvite(token string) (*GrupoIntegrante, *Grupo, error) {
	var i GrupoIntegrante
	if len(strings.TrimSpace(token)) < 20 || s.db.Where("convite_token_hash = ? AND status = 'convidado'", hashToken(token)).First(&i).Error != nil {
		return nil, nil, ErrConviteArtista
	}
	var g Grupo
	if s.db.First(&g, i.GrupoID).Error != nil {
		return nil, nil, ErrConviteArtista
	}
	return &i, &g, nil
}

func (s *GruposService) VerConvite(token string, usuarioID int64) (*ConviteVisao, error) {
	i, g, err := s.integranteDoConvite(token)
	if err != nil {
		return nil, err
	}
	v := &ConviteVisao{GrupoNome: g.Nome, GrupoTipo: g.Tipo, GrupoSlug: g.Slug, Solo: g.Solo, NomeArtistico: i.NomeArtistico, Funcao: i.Funcao, DesdeAno: i.DesdeAno,
		EmailConvite: mascararEmail(i.Email)}
	if usuarioID != 0 {
		var p PerfilTalento
		if s.db.Where("usuario_id = ?", usuarioID).First(&p).Error == nil {
			v.Nascimento = p.DataNascimento.Format("2006-01-02")
		}
	}
	return v, nil
}

// DadosAceite: o que a própria pessoa completa ao aceitar.
type DadosAceite struct {
	NomeArtistico    string `json:"nome_artistico"`
	FotoURL          string `json:"foto_url"`
	Instagram        string `json:"instagram"`
	DataNascimento   string `json:"data_nascimento"`
	ResponsavelNome  string `json:"responsavel_nome"`
	ResponsavelEmail string `json:"responsavel_email"`
}

// normalizarAceite aplica as regras de idade do item 8.1. Testada.
func normalizarAceite(d DadosAceite, agora time.Time) (DadosAceite, time.Time, error) {
	d.NomeArtistico = limpar(d.NomeArtistico)
	d.ResponsavelNome = limpar(d.ResponsavelNome)
	d.ResponsavelEmail = NormalizarEmail(d.ResponsavelEmail)
	nasc, err := time.Parse("2006-01-02", strings.TrimSpace(d.DataNascimento))
	switch {
	case utf8.RuneCountInString(d.NomeArtistico) < 2 || utf8.RuneCountInString(d.NomeArtistico) > 60:
		return d, nasc, fmt.Errorf("%w: o nome tem de 2 a 60 caracteres", ErrIntegranteInvalido)
	case err != nil || nasc.After(agora) || nasc.Before(agora.AddDate(-120, 0, 0)):
		return d, nasc, fmt.Errorf("%w: informe a data de nascimento", ErrIntegranteInvalido)
	case !urlImagemValida(d.FotoURL):
		return d, nasc, fmt.Errorf("%w: foto inválida", ErrIntegranteInvalido)
	}
	idade := Idade(nasc, agora)
	if idade < IdadeAdulto {
		d.Instagram = "" // menor: redes nunca aparecem nem são guardadas
		if idade < IdadeMinimaPerfil {
			d.FotoURL, d.ResponsavelNome, d.ResponsavelEmail = "", "", ""
			return d, nasc, nil
		}
		if utf8.RuneCountInString(d.ResponsavelNome) < 3 || !strings.Contains(d.ResponsavelEmail, "@") || !strings.Contains(d.ResponsavelEmail, ".") {
			return d, nasc, fmt.Errorf("%w: informe o nome e o e-mail do responsável", ErrIntegranteInvalido)
		}
		return d, nasc, nil
	}
	d.ResponsavelNome, d.ResponsavelEmail = "", ""
	d.Instagram = limparHandle(d.Instagram)
	if d.Instagram != "" && !reHandle.MatchString(d.Instagram) {
		return d, nasc, fmt.Errorf("%w: use só o @ do Instagram", ErrIntegranteInvalido)
	}
	return d, nasc, nil
}

// AceitarConvite: só com a conta do e-mail convidado.
func (s *GruposService) AceitarConvite(token string, usuarioID int64, d DadosAceite) (*GrupoIntegrante, error) {
	i, g, err := s.integranteDoConvite(token)
	if err != nil {
		return nil, err
	}
	var u domain.Usuario
	if err := s.db.First(&u, usuarioID).Error; err != nil {
		return nil, err
	}
	if NormalizarEmail(u.Email) != i.Email {
		return nil, ErrConviteOutroEmail
	}
	agora := time.Now()
	// Se a pessoa já declarou ser mais nova em outro lugar, vale essa data.
	if n, ok := s.nascimentosConhecidos([]int64{usuarioID})[usuarioID]; ok {
		if dec, err := time.Parse("2006-01-02", strings.TrimSpace(d.DataNascimento)); err != nil || n.After(dec) {
			d.DataNascimento = n.Format("2006-01-02")
		}
	}
	d, nasc, err := normalizarAceite(d, agora)
	if err != nil {
		return nil, err
	}
	var tokenResp string
	trocouResponsavel := i.ResponsavelEmail != d.ResponsavelEmail
	i.UsuarioID, i.NomeArtistico, i.FotoURL, i.Instagram, i.DataNascimento = &usuarioID, d.NomeArtistico, d.FotoURL, d.Instagram, &nasc
	i.ResponsavelNome, i.ResponsavelEmail = d.ResponsavelNome, d.ResponsavelEmail
	i.Status, i.ConviteTokenHash, i.AceitoEm = "ativo", nil, &agora
	idade := Idade(nasc, agora)
	if idade >= IdadeMinimaPerfil && idade < IdadeAdulto {
		if trocouResponsavel || i.AutorizacaoTokenHash == nil {
			var hash string
			tokenResp, hash = novoTokenAutorizacao()
			i.AutorizacaoTokenHash, i.AutorizadoEm, i.AutorizaFoto = &hash, nil, false
		}
	} else {
		i.AutorizacaoTokenHash, i.AutorizadoEm, i.AutorizaFoto = nil, nil, false
	}
	// Só grava se o convite ainda é o mesmo e está aberto: se o líder removeu
	// ou outro aceite chegou antes, nada muda.
	res := s.db.Model(&GrupoIntegrante{}).Where("id = ? AND status = 'convidado' AND convite_token_hash = ?", i.ID, hashToken(token)).
		Updates(map[string]any{"usuario_id": usuarioID, "nome_artistico": i.NomeArtistico, "foto_url": i.FotoURL, "instagram": i.Instagram,
			"data_nascimento": nasc, "responsavel_nome": i.ResponsavelNome, "responsavel_email": i.ResponsavelEmail, "status": "ativo",
			"convite_token_hash": nil, "aceito_em": agora, "autorizacao_token_hash": i.AutorizacaoTokenHash, "autorizado_em": i.AutorizadoEm,
			"autoriza_foto": i.AutorizaFoto})
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, ErrConviteArtista
	}
	if tokenResp != "" {
		s.pedirAutorizacaoIntegrante(g, i, tokenResp)
	}
	return i, nil
}

func (s *GruposService) RecusarConvite(token string, usuarioID int64) error {
	i, _, err := s.integranteDoConvite(token)
	if err != nil {
		return err
	}
	var u domain.Usuario
	if err := s.db.First(&u, usuarioID).Error; err != nil {
		return err
	}
	if NormalizarEmail(u.Email) != i.Email {
		return ErrConviteOutroEmail
	}
	res := s.db.Model(&GrupoIntegrante{}).Where("id = ? AND status = 'convidado' AND convite_token_hash = ?", i.ID, hashToken(token)).
		Updates(map[string]any{"status": "recusou", "convite_token_hash": nil})
	if res.RowsAffected == 0 {
		return ErrConviteArtista
	}
	return res.Error
}

func (s *GruposService) pedirAutorizacaoIntegrante(g *Grupo, i *GrupoIntegrante, token string) {
	if s.mail == nil {
		return
	}
	link := s.frontendURL + "/autorizar-integrante?t=" + token
	corpo := s.envelope(fmt.Sprintf(`<p>Olá, %s.</p><p>Um(a) jovem pelo qual você é responsável quer aparecer como integrante do grupo <strong>%s</strong> no perfil público do %s, com o nome artístico <strong>%s</strong>.</p>`+
		`<p>Só aparece se você autorizar: nome artístico, função e anos de experiência. Nunca nome real, idade, cidade, redes ou contato. Você escolhe se a foto aparece e pode desautorizar quando quiser pelo mesmo link.</p>%s`+
		`<p style="color:#777;font-size:12px">Se você não reconhece este pedido, ignore este e-mail: nada fica público sem a sua autorização.</p>`,
		html.EscapeString(primeiroNome(i.ResponsavelNome)), html.EscapeString(g.Nome), html.EscapeString(s.nome), html.EscapeString(i.NomeArtistico), s.botao(link, "Ver e decidir")))
	_ = s.mail.Enviar(i.ResponsavelEmail, "Autorização de integrante — "+g.Nome, corpo)
}

// AutorizacaoIntegrante é o que o responsável vê.
type AutorizacaoIntegrante struct {
	GrupoNome     string `json:"grupo_nome"`
	NomeArtistico string `json:"nome_artistico"`
	Funcao        string `json:"funcao"`
	TemFoto       bool   `json:"tem_foto"`
	Autorizado    bool   `json:"autorizado"`
	AutorizaFoto  bool   `json:"autoriza_foto"`
	Responsavel   string `json:"responsavel"`
}

func (s *GruposService) integranteDaAutorizacao(token string) (*GrupoIntegrante, *Grupo, error) {
	var i GrupoIntegrante
	if len(strings.TrimSpace(token)) < 20 || s.db.Where("autorizacao_token_hash = ?", hashToken(token)).First(&i).Error != nil {
		return nil, nil, ErrLinkAutorizacao
	}
	var g Grupo
	if s.db.First(&g, i.GrupoID).Error != nil {
		return nil, nil, ErrLinkAutorizacao
	}
	return &i, &g, nil
}

func (s *GruposService) VerAutorizacao(token string) (*AutorizacaoIntegrante, error) {
	i, g, err := s.integranteDaAutorizacao(token)
	if err != nil {
		return nil, err
	}
	return &AutorizacaoIntegrante{GrupoNome: g.Nome, NomeArtistico: i.NomeArtistico, Funcao: i.Funcao, TemFoto: i.FotoURL != "",
		Autorizado: i.AutorizadoEm != nil, AutorizaFoto: i.AutorizaFoto, Responsavel: i.ResponsavelNome}, nil
}

func (s *GruposService) DecidirAutorizacao(token string, autorizar, foto bool) (*AutorizacaoIntegrante, error) {
	i, _, err := s.integranteDaAutorizacao(token)
	if err != nil {
		return nil, err
	}
	upd := map[string]any{"autoriza_foto": autorizar && foto, "autorizado_em": nil}
	if autorizar {
		upd["autorizado_em"] = time.Now()
	}
	if err := s.db.Model(&GrupoIntegrante{}).Where("id = ?", i.ID).Updates(upd).Error; err != nil {
		return nil, err
	}
	return s.VerAutorizacao(token)
}

// DadosEdicaoIntegrante: o líder ajusta nome, função e ano de início.
func (s *GruposService) EditarIntegrante(usuarioID, grupoID, integranteID int64, d DadosIntegrante) error {
	if _, err := s.doLider(usuarioID, grupoID); err != nil {
		return err
	}
	var i GrupoIntegrante
	if s.db.Where("id = ? AND grupo_id = ? AND status IN ('convidado','ativo')", integranteID, grupoID).First(&i).Error != nil {
		return ErrIntegranteInvalido
	}
	d.Email = i.Email
	d, err := NormalizarIntegrante(d, time.Now())
	if err != nil {
		return err
	}
	return s.db.Model(&i).Updates(map[string]any{"nome_artistico": d.NomeArtistico, "funcao": d.Funcao, "desde_ano": d.DesdeAno}).Error
}

func (s *GruposService) RemoverIntegrante(usuarioID, grupoID, integranteID int64) error {
	g, err := s.doLider(usuarioID, grupoID)
	if err != nil {
		return err
	}
	var i GrupoIntegrante
	if s.db.Where("id = ? AND grupo_id = ?", integranteID, grupoID).First(&i).Error != nil {
		return ErrIntegranteInvalido
	}
	if i.UsuarioID != nil && *i.UsuarioID == g.LiderUsuarioID && g.Solo {
		return ErrSoloUmIntegrante
	}
	return s.db.Model(&i).Updates(map[string]any{"status": "removido", "convite_token_hash": nil, "autorizacao_token_hash": nil}).Error
}

// Sair: o integrante deixa o grupo (some do perfil na hora).
func (s *GruposService) Sair(usuarioID, grupoID int64) error {
	var g Grupo
	if s.db.First(&g, grupoID).Error != nil {
		return ErrArtistaNaoEncontrado
	}
	if g.LiderUsuarioID == usuarioID {
		return ErrLiderNaoSai
	}
	res := s.db.Model(&GrupoIntegrante{}).Where("grupo_id = ? AND usuario_id = ? AND status = 'ativo'", grupoID, usuarioID).
		Updates(map[string]any{"status": "saiu", "autorizacao_token_hash": nil})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrArtistaNaoEncontrado
	}
	return nil
}

// --- Portfólio ---

type DadosMidia struct {
	Tipo     string `json:"tipo"`
	URL      string `json:"url"`
	Legenda  string `json:"legenda"`
	EventoID *int64 `json:"evento_id"`
}

func (s *GruposService) AdicionarMidia(usuarioID, grupoID int64, d DadosMidia) (*GrupoMidia, error) {
	if _, err := s.doLider(usuarioID, grupoID); err != nil {
		return nil, err
	}
	d.Legenda = limpar(d.Legenda)
	if utf8.RuneCountInString(d.Legenda) > maxLegendaMidia {
		return nil, fmt.Errorf("%w: legenda com no máximo 140 caracteres", ErrMidiaInvalida)
	}
	switch d.Tipo {
	case "foto":
		if !strings.HasPrefix(d.URL, "http") {
			return nil, fmt.Errorf("%w: envie a foto", ErrMidiaInvalida)
		}
	case "video":
		v, err := NormalizarVideo(d.URL)
		if err != nil {
			return nil, err
		}
		d.URL = v.URL
	default:
		return nil, fmt.Errorf("%w: tipo", ErrMidiaInvalida)
	}
	if d.EventoID != nil {
		var n int64
		s.db.Model(&domain.Evento{}).Where("id = ? AND status = ?", *d.EventoID, domain.StatusEventoPublicado).Count(&n)
		if n == 0 {
			d.EventoID = nil
		}
	}
	var total int64
	s.db.Model(&GrupoMidia{}).Where("grupo_id = ?", grupoID).Count(&total)
	if total >= MaxMidiasGrupo {
		return nil, ErrLimiteMidias
	}
	m := &GrupoMidia{GrupoID: grupoID, Tipo: d.Tipo, URL: d.URL, Legenda: d.Legenda, EventoID: d.EventoID, Ordem: int(total), CriadoEm: time.Now()}
	return m, s.db.Create(m).Error
}

func (s *GruposService) RemoverMidia(usuarioID, grupoID, midiaID int64) error {
	if _, err := s.doLider(usuarioID, grupoID); err != nil {
		return err
	}
	res := s.db.Where("id = ? AND grupo_id = ?", midiaID, grupoID).Delete(&GrupoMidia{})
	if res.RowsAffected == 0 {
		return ErrMidiaInvalida
	}
	return res.Error
}

// --- Selo do Instagram ---

// PedirVerificacaoInstagram gera o código (se ainda não houver) e marca o
// pedido para o admin conferir.
func (s *GruposService) PedirVerificacaoInstagram(usuarioID, grupoID int64, soCodigo bool) (*Grupo, error) {
	g, err := s.doLider(usuarioID, grupoID)
	if err != nil {
		return nil, err
	}
	if g.Instagram == "" {
		return nil, fmt.Errorf("%w: informe o Instagram do grupo antes", ErrArtistaInvalido)
	}
	upd := map[string]any{}
	if g.InstagramCodigo == "" {
		b := make([]byte, TamanhoCodigoInsta)
		_, _ = rand.Read(b)
		g.InstagramCodigo = CodigoInstagram(b)
		upd["instagram_codigo"] = g.InstagramCodigo
	}
	if !soCodigo && g.InstagramVerificadoEm == nil {
		agora := time.Now()
		g.InstagramPedidoEm = &agora
		upd["instagram_pedido_em"] = agora
	}
	if len(upd) > 0 {
		if err := s.db.Model(&Grupo{}).Where("id = ?", g.ID).Updates(upd).Error; err != nil {
			return nil, err
		}
	}
	return g, nil
}

type PedidoInstagram struct {
	GrupoID   int64     `json:"grupo_id"`
	Nome      string    `json:"nome"`
	Slug      string    `json:"slug"`
	Instagram string    `json:"instagram"`
	Codigo    string    `json:"codigo"`
	PedidoEm  time.Time `json:"pedido_em"`
}

func (s *GruposService) PedidosInstagram() ([]PedidoInstagram, error) {
	out := []PedidoInstagram{}
	err := s.db.Raw(`SELECT id AS grupo_id, nome, slug, instagram, instagram_codigo AS codigo, instagram_pedido_em AS pedido_em
		FROM grupos WHERE instagram_pedido_em IS NOT NULL AND instagram_verificado_em IS NULL ORDER BY instagram_pedido_em`).Scan(&out).Error
	return out, err
}

// DecidirInstagram: o admin confere a bio e aprova (selo) ou recusa (o
// pedido some; o líder pode pedir de novo).
func (s *GruposService) DecidirInstagram(grupoID int64, aprovar bool, instagram string) error {
	upd := map[string]any{"instagram_pedido_em": nil}
	if aprovar {
		upd["instagram_verificado_em"] = time.Now()
	}
	// Se o líder trocou o @ depois que o admin abriu o pedido, nada muda.
	res := s.db.Model(&Grupo{}).Where("id = ? AND instagram_pedido_em IS NOT NULL AND instagram = ?", grupoID, limparHandle(instagram)).Updates(upd)
	if res.RowsAffected == 0 {
		return ErrArtistaNaoEncontrado
	}
	return res.Error
}

// --- Atrações ---

var (
	ErrAtracaoInvalida = errors.New("atração inválida")
)

// ConvidarAtracao: o organizador convida um grupo público para o evento.
func (s *GruposService) ConvidarAtracao(organizadorID, eventoID, usuarioID int64, grupoSlug string) (*Atracao, error) {
	e, err := s.eventos.BuscarDoOrganizador(organizadorID, eventoID)
	if err != nil {
		return nil, err
	}
	var g Grupo
	if s.db.Where("slug = ? AND publico", strings.TrimSpace(grupoSlug)).First(&g).Error != nil {
		return nil, ErrArtistaNaoEncontrado
	}
	a := &Atracao{EventoID: eventoID, GrupoID: g.ID, Status: "convidado", ConvidadoPor: &usuarioID, CriadoEm: time.Now()}
	res := s.db.Exec(`INSERT INTO atracoes (evento_id, grupo_id, status, convidado_por) VALUES (?, ?, 'convidado', ?)
		ON CONFLICT (evento_id, grupo_id) DO UPDATE SET status = 'convidado', decidido_em = NULL, convidado_por = EXCLUDED.convidado_por
		WHERE atracoes.status = 'recusado'`, eventoID, g.ID, usuarioID)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, fmt.Errorf("%w: o grupo já foi convidado", ErrAtracaoInvalida)
	}
	s.db.Where("evento_id = ? AND grupo_id = ?", eventoID, g.ID).First(a)
	if s.mail != nil {
		var lider domain.Usuario
		if s.db.First(&lider, g.LiderUsuarioID).Error == nil {
			corpo := s.envelope(fmt.Sprintf(`<p>Olá, %s!</p><p>O evento <strong>%s</strong> quer anunciar <strong>%s</strong> nas atrações.</p><p>Só aparece na página do evento se você confirmar.</p>%s`,
				html.EscapeString(primeiroNome(lider.Nome)), html.EscapeString(e.Titulo), html.EscapeString(g.Nome), s.botao(s.frontendURL+"/meus-artistas/"+fmt.Sprint(g.ID), "Ver o convite")))
			_ = s.mail.Enviar(lider.Email, "Convite de atração — "+e.Titulo, corpo)
		}
	}
	return a, nil
}

func (s *GruposService) RemoverAtracao(organizadorID, eventoID, grupoID int64) error {
	if _, err := s.eventos.BuscarDoOrganizador(organizadorID, eventoID); err != nil {
		return err
	}
	return s.db.Where("evento_id = ? AND grupo_id = ?", eventoID, grupoID).Delete(&Atracao{}).Error
}

// DecidirAtracao: o líder confirma ou recusa (pode desfazer depois).
func (s *GruposService) DecidirAtracao(usuarioID, grupoID, atracaoID int64, aceitar bool) error {
	if _, err := s.doLider(usuarioID, grupoID); err != nil {
		return err
	}
	status := "recusado"
	if aceitar {
		status = "confirmado"
	}
	res := s.db.Model(&Atracao{}).Where("id = ? AND grupo_id = ?", atracaoID, grupoID).
		Updates(map[string]any{"status": status, "decidido_em": time.Now()})
	if res.RowsAffected == 0 {
		return ErrAtracaoInvalida
	}
	return res.Error
}

// AtracaoDoEvento aparece no painel do evento (todas) e na página pública
// (só as confirmadas).
type AtracaoDoEvento struct {
	LiderID    int64  `json:"-"`
	GrupoID    int64  `json:"grupo_id"`
	Slug       string `json:"slug"`
	Nome       string `json:"nome"`
	Tipo       string `json:"tipo"`
	Solo       bool   `json:"solo"`
	LogoURL    string `json:"logo_url"`
	Instagram  string `json:"instagram"`
	Verificado bool   `json:"verificado"`
	Status     string `json:"status"`
}

func (s *GruposService) atracoesDoEvento(eventoID int64, soConfirmadas bool) ([]AtracaoDoEvento, error) {
	out := []AtracaoDoEvento{}
	q := `SELECT g.id AS grupo_id, g.lider_usuario_id AS lider_id, g.slug, g.nome, g.tipo, g.solo, g.logo_url, g.instagram, (g.instagram_verificado_em IS NOT NULL) AS verificado, a.status
		FROM atracoes a JOIN grupos g ON g.id = a.grupo_id WHERE a.evento_id = ?`
	if soConfirmadas {
		q += ` AND a.status = 'confirmado' AND g.publico AND ` + condSoloVisivel
	}
	err := s.db.Raw(q+` ORDER BY a.criado_em`, eventoID).Scan(&out).Error
	s.limparAtracoes(out)
	return out, err
}

func (s *GruposService) limparAtracoes(out []AtracaoDoEvento) {
	for k := range out {
		a := &out[k]
		var cidade, uf, capa string
		s.limparSoloMenor(a.GrupoID, a.LiderID, a.Solo, &a.Instagram, &cidade, &uf, &a.LogoURL, &capa)
		if a.Solo && a.Instagram == "" {
			a.Verificado = false
		}
	}
}

func (s *GruposService) AtracoesDoEventoOrg(organizadorID, eventoID int64) ([]AtracaoDoEvento, error) {
	if _, err := s.eventos.BuscarDoOrganizador(organizadorID, eventoID); err != nil {
		return nil, err
	}
	return s.atracoesDoEvento(eventoID, false)
}

func (s *GruposService) AtracoesPublicas(eventoID int64) ([]AtracaoDoEvento, error) {
	return s.atracoesDoEvento(eventoID, true)
}

// BuscarParaAtracao: grupos públicos pelo nome (para o organizador convidar).
func (s *GruposService) BuscarParaAtracao(termo string) ([]AtracaoDoEvento, error) {
	out := []AtracaoDoEvento{}
	termo = limpar(termo)
	if utf8.RuneCountInString(termo) < 2 {
		return out, nil
	}
	err := s.db.Raw(`SELECT g.id AS grupo_id, g.lider_usuario_id AS lider_id, g.slug, g.nome, g.tipo, g.solo, g.logo_url, g.instagram,
		(g.instagram_verificado_em IS NOT NULL) AS verificado, '' AS status
		FROM grupos g WHERE g.publico AND `+condSoloVisivel+` AND (g.nome ILIKE ? OR g.instagram ILIKE ?) ORDER BY g.nome LIMIT 10`,
		"%"+termo+"%", "%"+strings.TrimPrefix(termo, "@")+"%").Scan(&out).Error
	s.limparAtracoes(out)
	return out, err
}

// --- Resultados de concursos (inscrição pelo grupo) ---

// PodeInscreverPeloGrupo: o líder ou um integrante ativo.
func (s *GruposService) PodeInscreverPeloGrupo(usuarioID, grupoID int64) bool {
	var n int64
	s.db.Raw(`SELECT count(*) FROM grupos g WHERE g.id = ? AND (g.lider_usuario_id = ?
		OR EXISTS (SELECT 1 FROM grupo_integrantes gi WHERE gi.grupo_id = g.id AND gi.usuario_id = ? AND gi.status = 'ativo'))`, grupoID, usuarioID, usuarioID).Scan(&n)
	return n > 0
}

type ResultadoGrupo struct {
	EventoTitulo string     `json:"evento_titulo"`
	EventoSlug   string     `json:"evento_slug"`
	Data         *time.Time `json:"data"`
	Tipo         string     `json:"tipo"`
	Posicao      int        `json:"posicao,omitempty"`
	Total        int        `json:"total,omitempty"`
}

func (s *GruposService) resultados(grupoID int64) []ResultadoGrupo {
	var fichas []domain.FichaParticipacao
	s.db.Joins("JOIN eventos ON eventos.id = fichas_participacao.evento_id").
		Where("fichas_participacao.grupo_id = ? AND fichas_participacao.status = ? AND eventos.status = ? AND eventos.visibilidade = ?",
			grupoID, domain.StatusFichaAprovado, domain.StatusEventoPublicado, domain.VisibilidadePublico).
		Order("eventos.inicio_em DESC").Limit(40).Find(&fichas)
	out := []ResultadoGrupo{}
	for i := range fichas {
		f := &fichas[i]
		var e domain.Evento
		if s.db.First(&e, f.EventoID).Error != nil {
			continue
		}
		r := ResultadoGrupo{EventoTitulo: e.Titulo, EventoSlug: e.Slug, Data: e.InicioEm, Tipo: tipoDaFicha(f)}
		if s.avaliacao != nil && e.ResultadoLiberadoEm != nil {
			if ranking, err := s.avaliacao.ResultadoPublico(&e); err == nil {
				for _, linhas := range ranking {
					for _, l := range linhas {
						if l.FichaID == f.ID {
							r.Posicao, r.Total = l.Posicao, len(linhas)
						}
					}
				}
			}
		}
		out = append(out, r)
	}
	return out
}

// --- Público ---

type IntegrantePublico struct {
	NomeArtistico string `json:"nome_artistico"`
	Funcao        string `json:"funcao"`
	Anos          int    `json:"anos"`
	FotoURL       string `json:"foto_url,omitempty"`
	Instagram     string `json:"instagram,omitempty"`
}

type GrupoPublico struct {
	Slug              string              `json:"slug"`
	Nome              string              `json:"nome"`
	Tipo              string              `json:"tipo"`
	TipoNome          string              `json:"tipo_nome"`
	Solo              bool                `json:"solo"`
	Bio               string              `json:"bio"`
	Cidade            string              `json:"cidade"`
	UF                string              `json:"uf"`
	Instagram         string              `json:"instagram"`
	TikTok            string              `json:"tiktok"`
	YouTube           string              `json:"youtube"`
	LogoURL           string              `json:"logo_url"`
	CapaURL           string              `json:"capa_url"`
	Verificado        bool                `json:"verificado"`
	Integrantes       []IntegrantePublico `json:"integrantes"`
	OutrosIntegrantes int                 `json:"outros_integrantes"` // menores de 12 (sem nome)
	Midias            []MidiaVisao        `json:"midias"`
	Proximas          []AtracaoVisao      `json:"proximas"`
	Passadas          []AtracaoVisao      `json:"passadas"`
	Resultados        []ResultadoGrupo    `json:"resultados"`
	Indexar           bool                `json:"indexar"`
}

func (s *GruposService) integrantesPublicos(grupoID int64, agora time.Time) ([]IntegrantePublico, int, bool) {
	var is []GrupoIntegrante
	s.db.Where("grupo_id = ? AND status = 'ativo'", grupoID).Order("ordem, id").Find(&is)
	s.ajustarNascimentos(is)
	out := []IntegrantePublico{}
	anonimos, temMenor := 0, false
	for i := range is {
		x := &is[i]
		sit := x.situacao(agora)
		if sit.Menor {
			temMenor = true
		}
		switch {
		case !sit.Visivel:
			continue
		case sit.Anonimo:
			anonimos++
			continue
		}
		p := IntegrantePublico{NomeArtistico: x.NomeArtistico, Funcao: x.Funcao, Anos: AnosDeExperiencia(x.DesdeAno, agora)}
		if sit.MostraFoto {
			p.FotoURL = x.FotoURL
		}
		if sit.MostraInstagram {
			p.Instagram = x.Instagram
		}
		out = append(out, p)
	}
	return out, anonimos, temMenor
}

func (s *GruposService) Publico(slug string) (*GrupoPublico, error) {
	var g Grupo
	if s.db.Where("slug = ? AND publico", slug).First(&g).Error != nil {
		return nil, ErrArtistaNaoEncontrado
	}
	agora := time.Now()
	var sitSolo SituacaoIntegrante
	if g.Solo {
		sitSolo = s.situacaoSolo(&g, agora)
		if !sitSolo.Visivel || sitSolo.Anonimo {
			return nil, ErrArtistaNaoEncontrado
		}
	}
	ints, anon, temMenor := s.integrantesPublicos(g.ID, agora)
	p := &GrupoPublico{Slug: g.Slug, Nome: g.Nome, Tipo: g.Tipo, TipoNome: TiposGrupo[g.Tipo], Solo: g.Solo, Bio: g.Bio, Cidade: g.Cidade, UF: g.UF,
		Instagram: g.Instagram, TikTok: g.TikTok, YouTube: g.YouTube, LogoURL: g.LogoURL, CapaURL: g.CapaURL, Verificado: g.InstagramVerificadoEm != nil,
		Integrantes: ints, OutrosIntegrantes: anon, Midias: s.midias(g.ID), Proximas: []AtracaoVisao{}, Passadas: []AtracaoVisao{},
		Resultados: s.resultados(g.ID)}
	for _, a := range s.atracoes(g.ID, []string{"confirmado"}, true) {
		if a.EventoInicio != nil && a.EventoInicio.Before(agora.Add(-12*time.Hour)) {
			p.Passadas = append(p.Passadas, a)
		} else {
			p.Proximas = append(p.Proximas, a)
		}
	}
	// Perfil solo de menor (regras do item 8.1): sem redes, cidade nem
	// imagens sem autorização da foto, e fora do Google.
	if g.Solo && sitSolo.Menor {
		p.Instagram, p.TikTok, p.YouTube, p.Cidade, p.UF, p.Verificado = "", "", "", "", "", false
		if !sitSolo.MostraFoto {
			p.LogoURL, p.CapaURL = "", ""
			fotos := p.Midias[:0]
			for _, m := range p.Midias {
				if m.Tipo != "foto" {
					fotos = append(fotos, m)
				}
			}
			p.Midias = fotos
		}
	}
	p.Indexar = !(g.Solo && temMenor)
	return p, nil
}

// --- Vitrine ---

type FiltrosArtistas struct {
	Tipo   string
	UF     string
	Cidade string
	Busca  string
	Solo   *bool
	Pagina int
}

type ArtistaItem struct {
	ID          int64  `json:"-"`
	LiderID     int64  `json:"-"`
	Slug        string `json:"slug"`
	Nome        string `json:"nome"`
	Tipo        string `json:"tipo"`
	Solo        bool   `json:"solo"`
	Cidade      string `json:"cidade"`
	UF          string `json:"uf"`
	LogoURL     string `json:"logo_url"`
	CapaURL     string `json:"capa_url"`
	Instagram   string `json:"instagram"`
	Verificado  bool   `json:"verificado"`
	Integrantes int    `json:"integrantes"`
}

func (s *GruposService) Vitrine(f FiltrosArtistas) ([]ArtistaItem, int64, error) {
	q := s.db.Table("grupos g").Where("g.publico").Where(condSoloVisivel)
	if TiposGrupo[f.Tipo] != "" {
		q = q.Where("g.tipo = ?", f.Tipo)
	}
	if reUF.MatchString(strings.ToUpper(f.UF)) {
		q = q.Where("g.uf = ?", strings.ToUpper(f.UF))
	}
	if c := limpar(f.Cidade); c != "" {
		q = q.Where("g.cidade ILIKE ?", c)
	}
	if b := limpar(f.Busca); b != "" {
		q = q.Where("g.nome ILIKE ? OR g.bio ILIKE ?", "%"+b+"%", "%"+b+"%")
	}
	if f.Solo != nil {
		q = q.Where("g.solo = ?", *f.Solo)
	}
	var total int64
	if err := q.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	pagina := max(f.Pagina, 1)
	out := []ArtistaItem{}
	err := q.Select(`g.id, g.lider_usuario_id AS lider_id, g.slug, g.nome, g.tipo, g.solo, g.cidade, g.uf, g.logo_url, g.capa_url, g.instagram, (g.instagram_verificado_em IS NOT NULL) AS verificado,
		(SELECT count(*) FROM grupo_integrantes gi WHERE gi.grupo_id = g.id AND gi.status = 'ativo') AS integrantes`).
		Order("(g.instagram_verificado_em IS NOT NULL) DESC, g.atualizado_em DESC").Offset((pagina - 1) * 24).Limit(24).Scan(&out).Error
	for k := range out {
		a := &out[k]
		s.limparSoloMenor(a.ID, a.LiderID, a.Solo, &a.Instagram, &a.Cidade, &a.UF, &a.LogoURL, &a.CapaURL)
		if a.Solo && a.Instagram == "" {
			a.Verificado = false
		}
	}
	return out, total, err
}

// MeuStatusNoGrupo: integrante logado vendo os próprios dados.
func (s *GruposService) Grupo(grupoID int64) (*Grupo, error) {
	var g Grupo
	if err := s.db.First(&g, grupoID).Error; err != nil {
		return nil, ErrArtistaNaoEncontrado
	}
	return &g, nil
}

// paraJSON é usado pelo handler para devolver o grupo ao líder.
func (g *Grupo) ParaLider() map[string]any {
	b, _ := json.Marshal(struct {
		ID        int64  `json:"id"`
		Slug      string `json:"slug"`
		Nome      string `json:"nome"`
		Tipo      string `json:"tipo"`
		Solo      bool   `json:"solo"`
		Bio       string `json:"bio"`
		Cidade    string `json:"cidade"`
		UF        string `json:"uf"`
		Instagram string `json:"instagram"`
		TikTok    string `json:"tiktok"`
		YouTube   string `json:"youtube"`
		LogoURL   string `json:"logo_url"`
		CapaURL   string `json:"capa_url"`
		Publico   bool   `json:"publico"`
	}{g.ID, g.Slug, g.Nome, g.Tipo, g.Solo, g.Bio, g.Cidade, g.UF, g.Instagram, g.TikTok, g.YouTube, g.LogoURL, g.CapaURL, g.Publico})
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	return m
}
