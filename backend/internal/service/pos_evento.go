package service

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"html"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-pdf/fpdf"
	"github.com/skip2/go-qrcode"
	"gorm.io/gorm"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/WilliamBreno/Arcadia/backend/internal/mail"
)

// Item 9.7: certificados automáticos (PDF com código de validação) e
// pesquisa pós-evento (NPS, anônima para o organizador). Os dois vão no
// mesmo e-mail "Como foi?" do item 6.5, que sai depois do fim do evento.

var (
	ErrCertificadoNaoEncontrado = errors.New("certificado não encontrado")
	ErrNomeJaCorrigido          = errors.New("o nome deste certificado já foi corrigido uma vez")
	ErrDadosCertificado         = errors.New("dados do certificado inválidos")
	ErrEventoNaoTerminou        = errors.New("os certificados saem depois do fim do evento")
	ErrPesquisaInvalida         = errors.New("link da pesquisa inválido")
	ErrPesquisaFechada          = errors.New("a pesquisa deste evento já fechou")
)

const (
	FuncaoCertPublico      = "publico"
	FuncaoCertParticipante = "participante"
	FuncaoCertJurado       = "jurado"
	FuncaoCertEquipe       = "equipe"

	MinimoRespostasPesquisa = 3
	DiasPesquisaAberta      = 30
)

type ConfigCertificado struct {
	EventoID        int64 `gorm:"primaryKey"`
	Ativo           bool
	Publico         bool
	Participantes   bool
	Jurados         bool
	Equipe          bool
	CargaHorariaMin int
	Texto           string
	AssinaturaNome  string
	AssinaturaCargo string
	AtualizadoEm    time.Time
}

func (ConfigCertificado) TableName() string { return "config_certificado" }

type Certificado struct {
	ID            int64 `gorm:"primaryKey"`
	EventoID      int64
	Funcao        string
	ItemID        *int64
	FichaID       *int64
	PapelID       *int64
	UsuarioID     *int64
	Email         string
	Nome          string
	NomeCorrigido bool
	Detalhe       string
	Codigo        string
	EmitidoEm     time.Time `gorm:"default:now()"`
}

func (Certificado) TableName() string { return "certificados" }

type PosEventoService struct {
	db          *gorm.DB
	mail        *mail.Cliente
	segredo     string
	frontendURL string
	plataforma  string
}

func NovoPosEventoService(db *gorm.DB, m *mail.Cliente, segredo, frontendURL, plataforma string) *PosEventoService {
	return &PosEventoService{db: db, mail: m, segredo: segredo, frontendURL: strings.TrimSuffix(frontendURL, "/"), plataforma: plataforma}
}

// FimDoEvento: fim informado ou 3 h depois do início.
func FimDoEvento(e *domain.Evento) time.Time {
	if e.FimEm != nil {
		return *e.FimEm
	}
	if e.InicioEm != nil {
		return e.InicioEm.Add(3 * time.Hour)
	}
	return time.Time{}
}

// ---------- configuração ----------

func (s *PosEventoService) Config(eventoID int64) ConfigCertificado {
	c := ConfigCertificado{EventoID: eventoID, Publico: true}
	s.db.Where("evento_id = ?", eventoID).First(&c)
	return c
}

func (s *PosEventoService) SalvarConfig(c ConfigCertificado) error {
	c.Texto = strings.TrimSpace(c.Texto)
	c.AssinaturaNome = strings.TrimSpace(c.AssinaturaNome)
	c.AssinaturaCargo = strings.TrimSpace(c.AssinaturaCargo)
	if utf8.RuneCountInString(c.Texto) > 400 || utf8.RuneCountInString(c.AssinaturaNome) > 120 || utf8.RuneCountInString(c.AssinaturaCargo) > 120 ||
		c.CargaHorariaMin < 0 || c.CargaHorariaMin > 60000 {
		return fmt.Errorf("%w: texto até 400 letras, assinatura até 120 e carga horária até 1.000 h", ErrDadosCertificado)
	}
	if c.Ativo && !c.Publico && !c.Participantes && !c.Jurados && !c.Equipe {
		return fmt.Errorf("%w: escolha pelo menos um grupo", ErrDadosCertificado)
	}
	c.AtualizadoEm = time.Now()
	return s.db.Save(&c).Error
}

// ---------- emissão ----------

func codigoCertificado() (string, error) {
	a, err := gerarCodigoItem()
	if err != nil {
		return "", err
	}
	b, err := gerarCodigoItem()
	if err != nil {
		return "", err
	}
	return a + b[:4], nil
}

type candidatoCert struct {
	ItemID, FichaID, PapelID, UsuarioID *int64
	Email, Nome, Detalhe                string
}

// Emitir cria os certificados que faltam (idempotente). Só depois do fim
// do evento e com a configuração ligada.
func (s *PosEventoService) Emitir(e *domain.Evento, agora time.Time) (int, error) {
	c := s.Config(e.ID)
	if !c.Ativo {
		return 0, nil
	}
	if FimDoEvento(e).IsZero() || agora.Before(FimDoEvento(e)) {
		return 0, ErrEventoNaoTerminou
	}
	novos := 0
	grupos := []struct {
		ligado bool
		funcao string
		sql    string
	}{
		// Público: cada ingresso usado, no nome do titular.
		{c.Publico, FuncaoCertPublico, `SELECT i.id AS item_id, NULL::bigint AS ficha_id, NULL::bigint AS papel_id,
			(SELECT u.id FROM usuarios u WHERE lower(u.email) = lower(i.titular_email) LIMIT 1) AS usuario_id,
			lower(i.titular_email) AS email, i.titular_nome AS nome, '' AS detalhe
			FROM itens_pedido i JOIN pedidos p ON p.id = i.pedido_id
			WHERE p.evento_id = ? AND i.status = 'utilizado' AND i.ficha_id IS NULL AND i.expositor_id IS NULL AND NOT i.adicional
			AND trim(i.titular_nome) <> '' AND i.titular_email LIKE '%@%'
			AND NOT EXISTS (SELECT 1 FROM certificados c WHERE c.item_id = i.id)`},
		// Participantes aprovados que estiveram lá (camarim/palco); se o
		// evento não usou o camarim, vale a aprovação.
		{c.Participantes, FuncaoCertParticipante, `SELECT NULL::bigint AS item_id, f.id AS ficha_id, NULL::bigint AS papel_id, f.usuario_id,
			lower(u.email) AS email, f.nome,
			COALESCE((SELECT r.posicao || 'º lugar' FROM resultados_evento r WHERE r.ficha_id = f.id LIMIT 1), '') AS detalhe
			FROM fichas_participacao f JOIN usuarios u ON u.id = f.usuario_id
			WHERE f.evento_id = ? AND f.status = 'aprovado' AND f.status_palco <> 'ausente'
			AND (f.presente_em IS NOT NULL OR f.apresentou_em IS NOT NULL OR NOT EXISTS (
				SELECT 1 FROM fichas_participacao g WHERE g.evento_id = f.evento_id AND (g.presente_em IS NOT NULL OR g.apresentou_em IS NOT NULL)))
			AND NOT EXISTS (SELECT 1 FROM certificados c WHERE c.ficha_id = f.id)`},
		{c.Jurados, FuncaoCertJurado, `SELECT NULL::bigint AS item_id, NULL::bigint AS ficha_id, p.id AS papel_id, p.usuario_id,
			lower(u.email) AS email, u.nome, '' AS detalhe
			FROM papeis_evento p JOIN usuarios u ON u.id = p.usuario_id
			WHERE p.evento_id = ? AND p.papel = 'jurado' AND p.status = 'confirmado'
			AND NOT EXISTS (SELECT 1 FROM certificados c WHERE c.papel_id = p.id)`},
		{c.Equipe, FuncaoCertEquipe, `SELECT NULL::bigint AS item_id, NULL::bigint AS ficha_id, p.id AS papel_id, p.usuario_id,
			lower(u.email) AS email, u.nome, '' AS detalhe
			FROM papeis_evento p JOIN usuarios u ON u.id = p.usuario_id
			WHERE p.evento_id = ? AND p.papel = 'staff' AND p.status = 'confirmado'
			AND NOT EXISTS (SELECT 1 FROM certificados c WHERE c.papel_id = p.id)`},
	}
	for _, g := range grupos {
		if !g.ligado {
			continue
		}
		var cands []candidatoCert
		if err := s.db.Raw(g.sql, e.ID).Scan(&cands).Error; err != nil {
			return novos, err
		}
		for _, k := range cands {
			codigo, err := codigoCertificado()
			if err != nil {
				return novos, err
			}
			cert := Certificado{EventoID: e.ID, Funcao: g.funcao, ItemID: k.ItemID, FichaID: k.FichaID, PapelID: k.PapelID,
				UsuarioID: k.UsuarioID, Email: k.Email, Nome: limitarTexto(strings.TrimSpace(k.Nome), 160), Detalhe: k.Detalhe, Codigo: codigo}
			res := s.db.Exec(`INSERT INTO certificados (evento_id, funcao, item_id, ficha_id, papel_id, usuario_id, email, nome, detalhe, codigo)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT DO NOTHING`,
				cert.EventoID, cert.Funcao, cert.ItemID, cert.FichaID, cert.PapelID, cert.UsuarioID, cert.Email, cert.Nome, cert.Detalhe, cert.Codigo)
			if res.Error != nil {
				return novos, res.Error
			}
			novos += int(res.RowsAffected)
		}
	}
	return novos, nil
}

// ---------- links assinados ----------

func (s *PosEventoService) assinar(texto string, n int) string {
	m := hmac.New(sha256.New, []byte(s.segredo))
	m.Write([]byte(texto))
	return hex.EncodeToString(m.Sum(nil))[:n]
}

// TokenNome autoriza a correção do nome (vai só no e-mail da pessoa).
func (s *PosEventoService) TokenNome(codigo string) string { return s.assinar("cert-nome:"+codigo, 24) }

// respondente: hash do e-mail (a pesquisa não guarda quem respondeu).
func respondente(email string) string {
	h := sha256.Sum256([]byte("pesquisa:" + strings.ToLower(strings.TrimSpace(email))))
	return hex.EncodeToString(h[:])[:32]
}

// TokenPesquisa: <evento>.<respondente>.<assinatura> (sem dado pessoal).
func (s *PosEventoService) TokenPesquisa(eventoID int64, email string) string {
	base := strconv.FormatInt(eventoID, 10) + "." + respondente(email)
	return base + "." + s.assinar("pesquisa:"+base, 24)
}

func (s *PosEventoService) lerTokenPesquisa(token string) (int64, string, error) {
	p := strings.Split(token, ".")
	if len(p) != 3 {
		return 0, "", ErrPesquisaInvalida
	}
	base := p[0] + "." + p[1]
	if !hmac.Equal([]byte(p[2]), []byte(s.assinar("pesquisa:"+base, 24))) {
		return 0, "", ErrPesquisaInvalida
	}
	id, err := strconv.ParseInt(p[0], 10, 64)
	if err != nil {
		return 0, "", ErrPesquisaInvalida
	}
	return id, p[1], nil
}

// ---------- certificado público ----------

type CertificadoPublico struct {
	Codigo          string    `json:"codigo"`
	Nome            string    `json:"nome"`
	Funcao          string    `json:"funcao"`
	Detalhe         string    `json:"detalhe,omitempty"`
	EventoTitulo    string    `json:"evento_titulo"`
	EventoSlug      string    `json:"evento_slug"`
	EventoData      string    `json:"evento_data"`
	Organizador     string    `json:"organizador"`
	CargaHorariaMin int       `json:"carga_horaria_min"`
	EmitidoEm       time.Time `json:"emitido_em"`
	PodeCorrigir    bool      `json:"pode_corrigir"`
}

type dadosCert struct {
	cert   Certificado
	evento domain.Evento
	org    string
	cfg    ConfigCertificado
}

func (s *PosEventoService) carregar(codigo string) (*dadosCert, error) {
	codigo = strings.ToUpper(strings.TrimSpace(codigo))
	var d dadosCert
	if codigo == "" || s.db.Where("codigo = ?", codigo).First(&d.cert).Error != nil {
		return nil, ErrCertificadoNaoEncontrado
	}
	if s.db.First(&d.evento, d.cert.EventoID).Error != nil {
		return nil, ErrCertificadoNaoEncontrado
	}
	s.db.Table("organizadores").Where("id = ?", d.evento.OrganizadorID).Pluck("nome_publico", &d.org)
	d.cfg = s.Config(d.evento.ID)
	return &d, nil
}

func (s *PosEventoService) Validar(codigo, token string) (*CertificadoPublico, error) {
	d, err := s.carregar(codigo)
	if err != nil {
		return nil, err
	}
	data := ""
	if d.evento.InicioEm != nil {
		data = d.evento.InicioEm.In(FusoDoEvento(&d.evento)).Format("02/01/2006")
	}
	return &CertificadoPublico{
		Codigo: d.cert.Codigo, Nome: d.cert.Nome, Funcao: d.cert.Funcao, Detalhe: d.cert.Detalhe,
		EventoTitulo: d.evento.Titulo, EventoSlug: d.evento.Slug, EventoData: data, Organizador: d.org,
		CargaHorariaMin: d.cfg.CargaHorariaMin, EmitidoEm: d.cert.EmitidoEm,
		PodeCorrigir: !d.cert.NomeCorrigido && token != "" && hmac.Equal([]byte(token), []byte(s.TokenNome(d.cert.Codigo))),
	}, nil
}

// CorrigirNome: uma vez, pelo link do e-mail ou pela própria conta.
func (s *PosEventoService) CorrigirNome(codigo, token string, usuarioID int64, nome string) error {
	d, err := s.carregar(codigo)
	if err != nil {
		return err
	}
	dono := usuarioID > 0 && d.cert.UsuarioID != nil && *d.cert.UsuarioID == usuarioID
	if !dono && (token == "" || !hmac.Equal([]byte(token), []byte(s.TokenNome(d.cert.Codigo)))) {
		return ErrCertificadoNaoEncontrado
	}
	nome = strings.Join(strings.Fields(nome), " ")
	if utf8.RuneCountInString(nome) < 3 || utf8.RuneCountInString(nome) > 160 {
		return fmt.Errorf("%w: o nome precisa ter de 3 a 160 letras", ErrDadosCertificado)
	}
	res := s.db.Model(&Certificado{}).Where("id = ? AND NOT nome_corrigido", d.cert.ID).Updates(map[string]any{"nome": nome, "nome_corrigido": true})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNomeJaCorrigido
	}
	return nil
}

// Meus: certificados da conta (pelo usuário ou pelo e-mail do ingresso).
func (s *PosEventoService) Meus(usuarioID int64, email string) ([]CertificadoPublico, error) {
	var certs []Certificado
	if err := s.db.Where("usuario_id = ? OR (email <> '' AND lower(email) = lower(?))", usuarioID, email).
		Order("emitido_em DESC").Limit(200).Find(&certs).Error; err != nil {
		return nil, err
	}
	out := make([]CertificadoPublico, 0, len(certs))
	for _, c := range certs {
		var e domain.Evento
		if s.db.First(&e, c.EventoID).Error != nil {
			continue
		}
		data := ""
		if e.InicioEm != nil {
			data = e.InicioEm.In(FusoDoEvento(&e)).Format("02/01/2006")
		}
		out = append(out, CertificadoPublico{Codigo: c.Codigo, Nome: c.Nome, Funcao: c.Funcao, Detalhe: c.Detalhe,
			EventoTitulo: e.Titulo, EventoSlug: e.Slug, EventoData: data, EmitidoEm: c.EmitidoEm, PodeCorrigir: !c.NomeCorrigido})
	}
	return out, nil
}

// ---------- PDF ----------

func cargaHoraria(min int) string {
	h, m := min/60, min%60
	switch {
	case min <= 0:
		return ""
	case m == 0 && h == 1:
		return "1 hora"
	case m == 0:
		return fmt.Sprintf("%d horas", h)
	case h == 0:
		return fmt.Sprintf("%d minutos", m)
	default:
		return fmt.Sprintf("%dh%02d", h, m)
	}
}

func fraseFuncao(funcao, detalhe string) string {
	switch funcao {
	case FuncaoCertParticipante:
		if detalhe != "" {
			return "participou como competidor(a), conquistando o " + detalhe + ", no evento"
		}
		return "participou como competidor(a) no evento"
	case FuncaoCertJurado:
		return "atuou como jurado(a) no evento"
	case FuncaoCertEquipe:
		return "integrou a equipe de organização do evento"
	}
	return "participou do evento"
}

// PDF gera o certificado (A4 deitado) com o QR de validação.
func (s *PosEventoService) PDF(codigo string) ([]byte, string, error) {
	d, err := s.carregar(codigo)
	if err != nil {
		return nil, "", err
	}
	pdf := fpdf.New("L", "mm", "A4", "")
	pdf.SetTitle("Certificado "+d.cert.Codigo, true)
	pdf.SetAuthor(s.plataforma, true)
	pdf.SetMargins(20, 20, 20)
	pdf.SetAutoPageBreak(false, 0)
	tr := pdf.UnicodeTranslatorFromDescriptor("cp1252")
	pdf.AddPage()

	pdf.SetDrawColor(30, 30, 30)
	pdf.SetLineWidth(0.8)
	pdf.Rect(10, 10, 277, 190, "D")
	pdf.SetLineWidth(0.2)
	pdf.Rect(13, 13, 271, 184, "D")

	pdf.SetTextColor(20, 20, 20)
	pdf.SetFont("Helvetica", "B", 30)
	pdf.SetXY(20, 42)
	pdf.CellFormat(257, 14, tr("CERTIFICADO"), "", 1, "C", false, 0, "")
	pdf.Ln(6)
	pdf.SetFont("Helvetica", "", 13)
	pdf.SetX(20)
	pdf.CellFormat(257, 12, tr("Certificamos que"), "", 1, "C", false, 0, "")
	pdf.SetFont("Helvetica", "B", 24)
	pdf.SetX(20)
	pdf.MultiCell(257, 12, tr(d.cert.Nome), "", "C", false)

	data := DataDoEvento(&d.evento)
	if i := strings.Index(data, " às "); i > 0 {
		data = data[:i]
	}
	frase := fraseFuncao(d.cert.Funcao, d.cert.Detalhe) + " " + d.evento.Titulo
	if data != "" {
		frase += ", realizado em " + data
	}
	if ch := cargaHoraria(d.cfg.CargaHorariaMin); ch != "" {
		frase += ", com carga horária de " + ch
	}
	frase += "."
	pdf.Ln(8)
	pdf.SetFont("Helvetica", "", 14)
	pdf.SetX(35)
	pdf.MultiCell(227, 8, tr(frase), "", "C", false)
	if d.cfg.Texto != "" {
		pdf.Ln(2)
		pdf.SetFont("Helvetica", "I", 12)
		pdf.SetX(35)
		pdf.MultiCell(227, 7, tr(d.cfg.Texto), "", "C", false)
	}

	assinatura, cargo := d.cfg.AssinaturaNome, d.cfg.AssinaturaCargo
	if assinatura == "" {
		assinatura, cargo = d.org, "Organização"
	}
	pdf.SetLineWidth(0.3)
	pdf.Line(98, 160, 199, 160)
	pdf.SetFont("Helvetica", "B", 12)
	pdf.SetXY(70, 162)
	pdf.CellFormat(157, 6, tr(assinatura), "", 1, "C", false, 0, "")
	pdf.SetFont("Helvetica", "", 10)
	pdf.SetX(70)
	pdf.CellFormat(157, 5, tr(cargo), "", 1, "C", false, 0, "")

	link := s.frontendURL + "/certificado/" + d.cert.Codigo
	if png, err := qrcode.Encode(link, qrcode.Medium, 256); err == nil {
		pdf.RegisterImageOptionsReader("qr", fpdf.ImageOptions{ImageType: "PNG"}, bytes.NewReader(png))
		pdf.ImageOptions("qr", 248, 165, 28, 28, false, fpdf.ImageOptions{ImageType: "PNG"}, 0, "")
	}
	pdf.SetFont("Helvetica", "", 8)
	pdf.SetTextColor(90, 90, 90)
	pdf.SetXY(20, 184)
	pdf.CellFormat(220, 4, tr("Código de validação: "+d.cert.Codigo), "", 1, "L", false, 0, "")
	pdf.SetX(20)
	pdf.CellFormat(220, 4, tr("Confira em "+link+" · emitido por "+s.plataforma), "", 1, "L", false, 0, "")

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, "", err
	}
	return buf.Bytes(), "certificado-" + d.cert.Codigo + ".pdf", nil
}

// ---------- pesquisa ----------

type PesquisaPublica struct {
	EventoTitulo string `json:"evento_titulo"`
	EventoSlug   string `json:"evento_slug"`
	JaRespondeu  bool   `json:"ja_respondeu"`
	Nota         *int   `json:"nota"`
	Comentario   string `json:"comentario"`
	Aberta       bool   `json:"aberta"`
}

func (s *PosEventoService) Pesquisa(token string) (*PesquisaPublica, error) {
	eventoID, resp, err := s.lerTokenPesquisa(token)
	if err != nil {
		return nil, err
	}
	var e domain.Evento
	if s.db.First(&e, eventoID).Error != nil {
		return nil, ErrPesquisaInvalida
	}
	p := &PesquisaPublica{EventoTitulo: e.Titulo, EventoSlug: e.Slug, Aberta: time.Since(FimDoEvento(&e)) <= DiasPesquisaAberta*24*time.Hour}
	var r struct {
		Nota       int
		Comentario string
	}
	if res := s.db.Raw(`SELECT nota, comentario FROM respostas_pesquisa WHERE evento_id = ? AND respondente = ?`, eventoID, resp).Scan(&r); res.RowsAffected > 0 {
		p.JaRespondeu, p.Nota, p.Comentario = true, &r.Nota, r.Comentario
	}
	return p, nil
}

// Responder grava ou atualiza a resposta (enquanto a pesquisa estiver aberta).
func (s *PosEventoService) Responder(token string, nota int, comentario string) error {
	eventoID, resp, err := s.lerTokenPesquisa(token)
	if err != nil {
		return err
	}
	var e domain.Evento
	if s.db.First(&e, eventoID).Error != nil {
		return ErrPesquisaInvalida
	}
	if time.Since(FimDoEvento(&e)) > DiasPesquisaAberta*24*time.Hour || time.Now().Before(FimDoEvento(&e)) {
		return ErrPesquisaFechada
	}
	if nota < 0 || nota > 10 {
		return fmt.Errorf("%w: nota de 0 a 10", ErrDadosCertificado)
	}
	comentario = limitarTexto(strings.TrimSpace(comentario), 1000)
	return s.db.Exec(`INSERT INTO respostas_pesquisa (evento_id, respondente, nota, comentario) VALUES (?, ?, ?, ?)
		ON CONFLICT (evento_id, respondente) DO UPDATE SET nota = EXCLUDED.nota, comentario = EXCLUDED.comentario, criado_em = now()`,
		eventoID, resp, nota, comentario).Error
}

type ResultadoPesquisa struct {
	Respostas    int64              `json:"respostas"`
	Minimo       int                `json:"minimo"`
	NPS          *int               `json:"nps"`
	Promotores   int64              `json:"promotores"`
	Neutros      int64              `json:"neutros"`
	Detratores   int64              `json:"detratores"`
	Distribuicao []int64            `json:"distribuicao"`
	Comentarios  []ComentarioAnonim `json:"comentarios"`
}

type ComentarioAnonim struct {
	Nota       int    `json:"nota"`
	Comentario string `json:"comentario"`
	Data       string `json:"data"`
}

// ResultadoPesquisa: só números e comentários sem nome; com menos de 3
// respostas nada é mostrado (para ninguém ser reconhecido pela nota).
func (s *PosEventoService) ResultadoPesquisa(eventoID int64) (*ResultadoPesquisa, error) {
	r := &ResultadoPesquisa{Minimo: MinimoRespostasPesquisa, Distribuicao: make([]int64, 11), Comentarios: []ComentarioAnonim{}}
	var linhas []struct {
		Nota int
		N    int64
	}
	if err := s.db.Raw(`SELECT nota, count(*) AS n FROM respostas_pesquisa WHERE evento_id = ? GROUP BY nota`, eventoID).Scan(&linhas).Error; err != nil {
		return nil, err
	}
	for _, l := range linhas {
		r.Respostas += l.N
		r.Distribuicao[l.Nota] = l.N
		switch {
		case l.Nota >= 9:
			r.Promotores += l.N
		case l.Nota >= 7:
			r.Neutros += l.N
		default:
			r.Detratores += l.N
		}
	}
	if r.Respostas < MinimoRespostasPesquisa {
		return &ResultadoPesquisa{Respostas: r.Respostas, Minimo: MinimoRespostasPesquisa, Distribuicao: make([]int64, 11), Comentarios: []ComentarioAnonim{}}, nil
	}
	nps := int((float64(r.Promotores-r.Detratores)/float64(r.Respostas))*100 + 0.5*sinal(r.Promotores-r.Detratores))
	r.NPS = &nps
	var coms []struct {
		Nota       int
		Comentario string
		CriadoEm   time.Time
	}
	s.db.Raw(`SELECT nota, comentario, criado_em FROM respostas_pesquisa WHERE evento_id = ? AND comentario <> '' ORDER BY random() LIMIT 200`, eventoID).Scan(&coms)
	// Ordem aleatória e só a data (sem hora): não dá para ligar à pessoa.
	sort.SliceStable(coms, func(i, j int) bool { return coms[i].Nota < coms[j].Nota })
	for _, c := range coms {
		r.Comentarios = append(r.Comentarios, ComentarioAnonim{Nota: c.Nota, Comentario: c.Comentario, Data: c.CriadoEm.Format("02/01/2006")})
	}
	return r, nil
}

func sinal(n int64) float64 {
	if n < 0 {
		return -1
	}
	return 1
}

// ---------- e-mail pós-evento (substitui o "Como foi?" do 6.5) ----------

type destinoPos struct {
	nome      string
	comprador bool // pode avaliar na página (6.5)
	publico   bool // responde a pesquisa
	certs     []Certificado
}

// Processar: depois do fim do evento, emite os certificados e manda um
// e-mail por pessoa com avaliação, pesquisa e certificados.
func (s *PosEventoService) Processar(agora time.Time) (int, error) {
	var eventos []domain.Evento
	if err := s.db.Where("status = ? AND inicio_em BETWEEN ? AND ?", domain.StatusEventoPublicado, agora.AddDate(0, 0, -40), agora).
		Find(&eventos).Error; err != nil {
		return 0, err
	}
	enviados := 0
	for i := range eventos {
		e := &eventos[i]
		if !deveConvidar(e, agora) {
			continue
		}
		if _, err := s.Emitir(e, agora); err != nil && !errors.Is(err, ErrEventoNaoTerminou) {
			return enviados, err
		}
		destinos := map[string]*destinoPos{}
		pegar := func(email, nome string) *destinoPos {
			email = strings.ToLower(strings.TrimSpace(email))
			if !strings.Contains(email, "@") {
				return nil
			}
			d := destinos[email]
			if d == nil {
				d = &destinoPos{nome: nome}
				destinos[email] = d
			}
			if d.nome == "" {
				d.nome = nome
			}
			return d
		}
		var compradores []struct{ Nome, Email string }
		s.db.Raw(`SELECT DISTINCT u.nome, u.email FROM itens_pedido i JOIN pedidos p ON p.id = i.pedido_id JOIN usuarios u ON u.id = p.usuario_id
			WHERE p.evento_id = ? AND i.status = 'utilizado' AND NOT i.cortesia AND p.canal = 'online' AND i.ficha_id IS NULL AND i.expositor_id IS NULL AND NOT i.adicional`, e.ID).Scan(&compradores)
		for _, c := range compradores {
			if d := pegar(c.Email, c.Nome); d != nil {
				d.comprador, d.publico = true, true
			}
		}
		var titulares []struct{ Nome, Email string }
		s.db.Raw(`SELECT DISTINCT i.titular_nome AS nome, i.titular_email AS email FROM itens_pedido i JOIN pedidos p ON p.id = i.pedido_id
			WHERE p.evento_id = ? AND i.status = 'utilizado' AND i.ficha_id IS NULL AND i.expositor_id IS NULL AND NOT i.adicional`, e.ID).Scan(&titulares)
		for _, t := range titulares {
			if d := pegar(t.Email, t.Nome); d != nil {
				d.publico = true
			}
		}
		var certs []Certificado
		s.db.Where("evento_id = ?", e.ID).Order("id").Find(&certs)
		for _, c := range certs {
			if d := pegar(c.Email, c.Nome); d != nil {
				d.certs = append(d.certs, c)
			}
		}
		emails := make([]string, 0, len(destinos))
		for em := range destinos {
			emails = append(emails, em)
		}
		sort.Strings(emails)
		for _, email := range emails {
			d := destinos[email]
			res := s.db.Exec("INSERT INTO lembretes_enviados (evento_id, email, tipo) VALUES (?, ?, 'pos') ON CONFLICT DO NOTHING", e.ID, email)
			if res.Error != nil {
				return enviados, res.Error
			}
			if res.RowsAffected == 0 {
				continue
			}
			if err := s.mail.Enviar(email, "Como foi o "+e.Titulo+"?", s.corpoPos(e, email, d)); err == nil {
				enviados++
			}
		}
	}
	return enviados, nil
}

func (s *PosEventoService) corpoPos(e *domain.Evento, email string, d *destinoPos) string {
	var b strings.Builder
	esc := html.EscapeString
	fmt.Fprintf(&b, `<div style="font-family:Arial,sans-serif;max-width:560px;margin:auto;color:#111"><p>Olá, %s! Obrigado por fazer parte do <strong>%s</strong>.</p>`,
		esc(primeiroNome(d.nome)), esc(e.Titulo))
	if d.publico {
		token := s.TokenPesquisa(e.ID, email)
		b.WriteString(`<p style="margin:20px 0 8px"><strong>De 0 a 10, quanto você recomendaria este evento a um amigo?</strong></p><p style="margin:0">`)
		for n := 0; n <= 10; n++ {
			fmt.Fprintf(&b, `<a href="%s/pesquisa/%s?nota=%d" style="display:inline-block;min-width:26px;margin:2px;padding:6px 0;border:1px solid #ccc;border-radius:6px;text-align:center;color:#111;text-decoration:none">%d</a>`,
				s.frontendURL, token, n, n)
		}
		b.WriteString(`</p><p style="color:#777;font-size:12px;margin:6px 0 0">Resposta anônima: o organizador vê só as notas e os comentários, sem o seu nome.</p>`)
	}
	if d.comprador {
		fmt.Fprintf(&b, `<p style="margin-top:20px">Quer contar para quem ainda vai? <a href="%s/e/%s#avaliar">Avalie o evento na página</a> (até %s).</p>`,
			s.frontendURL, e.Slug, fimParaAvaliar(e).In(FusoDoEvento(e)).Format("02/01/2006"))
	}
	if len(d.certs) > 0 {
		b.WriteString(`<p style="margin-top:20px"><strong>Seus certificados</strong></p><ul style="padding-left:18px">`)
		for _, c := range d.certs {
			link := s.frontendURL + "/certificado/" + c.Codigo + "?t=" + s.TokenNome(c.Codigo)
			fmt.Fprintf(&b, `<li style="margin-bottom:6px"><a href="%s">%s</a> · %s</li>`, link, esc(c.Nome), esc(rotuloFuncaoCert(c.Funcao)))
		}
		b.WriteString(`</ul><p style="color:#777;font-size:12px">Confira se o nome está certo antes de baixar: dá para corrigir uma vez pelo link.</p>`)
	}
	b.WriteString(`</div>`)
	return b.String()
}

func rotuloFuncaoCert(f string) string {
	switch f {
	case FuncaoCertParticipante:
		return "participante"
	case FuncaoCertJurado:
		return "jurado"
	case FuncaoCertEquipe:
		return "equipe"
	}
	return "participação"
}

// ---------- organizador ----------

type ResumoCertificados struct {
	Total          int64                `json:"total"`
	PorFuncao      map[string]int       `json:"por_funcao"`
	EventoTerminou bool                 `json:"evento_terminou"`
	Lista          []CertificadoPublico `json:"lista"`
}

func (s *PosEventoService) Resumo(e *domain.Evento) (*ResumoCertificados, error) {
	r := &ResumoCertificados{PorFuncao: map[string]int{}, EventoTerminou: !FimDoEvento(e).IsZero() && time.Now().After(FimDoEvento(e)), Lista: []CertificadoPublico{}}
	var certs []Certificado
	if err := s.db.Where("evento_id = ?", e.ID).Order("funcao, nome").Limit(2000).Find(&certs).Error; err != nil {
		return nil, err
	}
	s.db.Model(&Certificado{}).Where("evento_id = ?", e.ID).Count(&r.Total)
	for _, c := range certs {
		r.PorFuncao[c.Funcao]++
		r.Lista = append(r.Lista, CertificadoPublico{Codigo: c.Codigo, Nome: c.Nome, Funcao: c.Funcao, Detalhe: c.Detalhe, EmitidoEm: c.EmitidoEm})
	}
	return r, nil
}
