package service

import (
	"context"
	"errors"
	"fmt"
	"html"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/WilliamBreno/Arcadia/backend/internal/mail"
)

// Item 9.9: verificação pública de ingresso (contra golpe), mensagens ao
// suporte e página de status.

var (
	ErrDadosContato = errors.New("dados do contato inválidos")
	ErrDadosAviso   = errors.New("dados do aviso inválidos")
)

// ---------- verificação de ingresso ----------

const (
	SituacaoValido       = "valido"
	SituacaoUtilizado    = "utilizado"
	SituacaoCancelado    = "cancelado"
	SituacaoNaRevenda    = "na_revenda"
	SituacaoTransferido  = "transferido"
	SituacaoNaoPago      = "nao_pago"
	SituacaoNaoExiste    = "nao_encontrado"
	SituacaoEventoPassou = "evento_encerrado"
)

type VerificacaoIngresso struct {
	Situacao     string     `json:"situacao"`
	EventoTitulo string     `json:"evento_titulo,omitempty"`
	EventoSlug   string     `json:"evento_slug,omitempty"`
	EventoInicio *time.Time `json:"evento_inicio_em,omitempty"`
	Tipo         string     `json:"tipo,omitempty"`
}

var reCodigoIngresso = regexp.MustCompile(`^[A-Z0-9]{8}$`)

// CodigoDoTexto aceita o código, "CODIGO:TOKEN" (o QR) ou o link
// /ingresso/CODIGO/TOKEN. Testada.
func CodigoDoTexto(texto string) string {
	t := strings.ToUpper(strings.TrimSpace(texto))
	if i := strings.Index(t, "/INGRESSO/"); i >= 0 {
		t = t[i+len("/INGRESSO/"):]
		if j := strings.Index(t, "/"); j >= 0 {
			t = t[:j]
		}
	}
	if i := strings.Index(t, ":"); i >= 0 {
		t = t[:i]
	}
	t = strings.ReplaceAll(strings.ReplaceAll(t, " ", ""), "-", "")
	if !reCodigoIngresso.MatchString(t) {
		return ""
	}
	return t
}

// VerificarIngresso diz a situação do ingresso sem nunca mostrar o nome
// de quem tem. Inscrição e espaço de expositor não são ingresso.
func VerificarIngresso(db *gorm.DB, texto string, agora time.Time) VerificacaoIngresso {
	codigo := CodigoDoTexto(texto)
	if codigo == "" {
		return VerificacaoIngresso{Situacao: SituacaoNaoExiste}
	}
	var linha struct {
		ItemID      int64
		Status      string
		Tipo        string
		EventoID    int64
		Titulo      string
		Slug        string
		InicioEm    *time.Time
		FimEm       *time.Time
		StatusEvent string
	}
	res := db.Raw(`SELECT i.id AS item_id, i.status, t.nome AS tipo, e.id AS evento_id, e.titulo, e.slug, e.inicio_em, e.fim_em, e.status AS status_event
		FROM itens_pedido i JOIN tipos_ingresso t ON t.id = i.tipo_ingresso_id JOIN eventos e ON e.id = t.evento_id
		WHERE i.codigo = ? AND i.ficha_id IS NULL AND i.expositor_id IS NULL AND NOT i.adicional LIMIT 1`, codigo).Scan(&linha)
	if res.Error != nil || res.RowsAffected == 0 {
		// Código antigo de um ingresso que foi transferido: o print não vale.
		var n int64
		db.Raw(`SELECT count(*) FROM transferencias_ingresso WHERE codigo_anterior = ?`, codigo).Scan(&n)
		if n > 0 {
			return VerificacaoIngresso{Situacao: SituacaoTransferido}
		}
		return VerificacaoIngresso{Situacao: SituacaoNaoExiste}
	}
	v := VerificacaoIngresso{EventoTitulo: linha.Titulo, EventoSlug: linha.Slug, EventoInicio: linha.InicioEm, Tipo: linha.Tipo}
	fim := linha.FimEm
	if fim == nil && linha.InicioEm != nil {
		f := linha.InicioEm.Add(12 * time.Hour)
		fim = &f
	}
	switch {
	case linha.StatusEvent == string(domain.StatusEventoCancelado):
		v.Situacao = SituacaoCancelado
	case linha.Status == string(domain.StatusItemUtilizado):
		v.Situacao = SituacaoUtilizado
	case linha.Status == string(domain.StatusItemPago):
		v.Situacao = SituacaoValido
		if a, err := AnuncioAtivo(db, linha.ItemID); err == nil && a != nil {
			v.Situacao = SituacaoNaRevenda
		} else if fim != nil && agora.After(*fim) {
			v.Situacao = SituacaoEventoPassou
		}
	case linha.Status == string(domain.StatusItemReservado):
		v.Situacao = SituacaoNaoPago
	default:
		v.Situacao = SituacaoCancelado
	}
	return v
}

// ---------- suporte ----------

var AssuntosSuporte = map[string]string{
	"compra":      "Compra e pagamento",
	"ingresso":    "Ingresso, QR ou transferência",
	"reembolso":   "Cancelamento e reembolso",
	"golpe":       "Suspeita de golpe",
	"organizador": "Sou organizador",
	"repasse":     "Repasse e financeiro",
	"outro":       "Outro assunto",
}

type ContatoSuporte struct {
	ID        int64 `gorm:"primaryKey"`
	UsuarioID *int64
	Nome      string
	Email     string
	Assunto   string
	Pedido    string
	Mensagem  string
	Status    string    `gorm:"default:aberto"`
	CriadoEm  time.Time `gorm:"default:now()"`
}

func (ContatoSuporte) TableName() string { return "contatos_suporte" }

type AjudaService struct {
	db            *gorm.DB
	mail          *mail.Cliente
	suporteEmail  string
	plataforma    string
	mpBaseURL     string
	mpToken       string
	httpCliente   *http.Client
	mu            sync.Mutex
	statusCache   *StatusPlataforma
	statusCacheEm time.Time
}

func NovoAjudaService(db *gorm.DB, m *mail.Cliente, suporteEmail, plataforma, mpBaseURL, mpToken string) *AjudaService {
	if mpBaseURL == "" {
		mpBaseURL = "https://api.mercadopago.com"
	}
	return &AjudaService{db: db, mail: m, suporteEmail: strings.TrimSpace(suporteEmail), plataforma: plataforma,
		mpBaseURL: strings.TrimSuffix(mpBaseURL, "/"), mpToken: mpToken, httpCliente: &http.Client{Timeout: 4 * time.Second}}
}

// EnviarContato grava a mensagem e manda para o e-mail do suporte.
// Devolve o número do protocolo.
func (s *AjudaService) EnviarContato(usuarioID int64, nome, email, assunto, pedido, mensagem string) (int64, error) {
	nome = strings.TrimSpace(nome)
	email = strings.ToLower(strings.TrimSpace(email))
	pedido = strings.TrimSpace(pedido)
	mensagem = strings.TrimSpace(mensagem)
	rotulo, ok := AssuntosSuporte[assunto]
	switch {
	case !ok:
		return 0, fmt.Errorf("%w: escolha o assunto", ErrDadosContato)
	case utf8.RuneCountInString(nome) < 2 || utf8.RuneCountInString(nome) > 120:
		return 0, fmt.Errorf("%w: informe o seu nome", ErrDadosContato)
	case !strings.Contains(email, "@") || len(email) > 200 || strings.ContainsAny(email, " <>\r\n"):
		return 0, fmt.Errorf("%w: e-mail inválido", ErrDadosContato)
	case utf8.RuneCountInString(mensagem) < 10 || utf8.RuneCountInString(mensagem) > 4000:
		return 0, fmt.Errorf("%w: escreva a mensagem (de 10 a 4.000 letras)", ErrDadosContato)
	case utf8.RuneCountInString(pedido) > 40:
		return 0, fmt.Errorf("%w: número do pedido muito longo", ErrDadosContato)
	}
	c := ContatoSuporte{Nome: nome, Email: email, Assunto: assunto, Pedido: pedido, Mensagem: mensagem}
	if usuarioID > 0 {
		c.UsuarioID = &usuarioID
	}
	if err := s.db.Create(&c).Error; err != nil {
		return 0, err
	}
	if s.suporteEmail == "" {
		slog.Warn("suporte: SUPORTE_EMAIL vazio, mensagem só gravada no banco", "protocolo", c.ID)
		return c.ID, nil
	}
	conta := "sem conta (não logado)"
	if usuarioID > 0 {
		conta = fmt.Sprintf("usuário #%d", usuarioID)
	}
	esc := html.EscapeString
	corpo := fmt.Sprintf(`<div style="font-family:Arial,sans-serif;color:#111"><p><strong>Protocolo #%d</strong> · %s</p>
		<p>De: %s &lt;%s&gt; · %s</p>%s<pre style="white-space:pre-wrap;font-family:inherit;border-left:3px solid #ccc;padding-left:12px">%s</pre>
		<p style="color:#777;font-size:12px">Responda direto para o e-mail da pessoa.</p></div>`,
		c.ID, esc(rotulo), esc(nome), esc(email), esc(conta), func() string {
			if pedido == "" {
				return ""
			}
			return "<p>Pedido: " + esc(pedido) + "</p>"
		}(), esc(mensagem))
	if err := s.mail.Enviar(s.suporteEmail, fmt.Sprintf("[%s] #%d %s", s.plataforma, c.ID, rotulo), corpo); err != nil {
		slog.Error("suporte: e-mail não foi para a fila", "protocolo", c.ID, "erro", err)
	}
	return c.ID, nil
}

func (s *AjudaService) ListarContatos(status string) ([]ContatoSuporte, error) {
	var lista []ContatoSuporte
	q := s.db.Order("criado_em DESC").Limit(200)
	if status != "" {
		q = q.Where("status = ?", status)
	}
	return lista, q.Find(&lista).Error
}

func (s *AjudaService) MarcarRespondido(id int64) error {
	res := s.db.Model(&ContatoSuporte{}).Where("id = ?", id).Update("status", "respondido")
	if res.Error == nil && res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return res.Error
}

// ---------- status ----------

type AvisoStatus struct {
	ID         int64      `gorm:"primaryKey" json:"id"`
	Titulo     string     `json:"titulo"`
	Mensagem   string     `json:"mensagem"`
	Nivel      string     `json:"nivel"`
	Componente string     `json:"componente"`
	InicioEm   time.Time  `gorm:"default:now()" json:"inicio_em"`
	FimEm      *time.Time `json:"fim_em"`
	CriadoPor  *int64     `json:"-"`
}

func (AvisoStatus) TableName() string { return "avisos_status" }

const (
	StatusOperacional = "operacional"
	StatusDegradado   = "degradado"
	StatusFalha       = "falha"
)

type ComponenteStatus struct {
	Chave   string `json:"chave"`
	Nome    string `json:"nome"`
	Status  string `json:"status"`
	Detalhe string `json:"detalhe,omitempty"`
}

type StatusPlataforma struct {
	Geral        string             `json:"geral"`
	Componentes  []ComponenteStatus `json:"componentes"`
	Avisos       []AvisoStatus      `json:"avisos"`
	Recentes     []AvisoStatus      `json:"recentes"`
	AtualizadoEm time.Time          `json:"atualizado_em"`
}

var componentesAviso = map[string]bool{"geral": true, "site": true, "pagamentos": true, "emails": true, "checkin": true, "repasses": true}

// Status confere banco, Mercado Pago e fila de e-mails (com 30 s de cache
// para a página não virar um teste de carga) e junta os avisos do admin.
func (s *AjudaService) Status(agora time.Time) *StatusPlataforma {
	s.mu.Lock()
	if s.statusCache != nil && agora.Sub(s.statusCacheEm) < 30*time.Second {
		c := s.statusCache
		s.mu.Unlock()
		return c
	}
	s.mu.Unlock()

	st := &StatusPlataforma{AtualizadoEm: agora, Avisos: []AvisoStatus{}, Recentes: []AvisoStatus{}}
	st.Componentes = append(st.Componentes, ComponenteStatus{Chave: "site", Nome: "Site e API", Status: StatusOperacional})

	banco := ComponenteStatus{Chave: "banco", Nome: "Banco de dados", Status: StatusOperacional}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	inicio := time.Now()
	if err := s.db.WithContext(ctx).Exec("SELECT 1").Error; err != nil {
		banco.Status, banco.Detalhe = StatusFalha, "sem resposta"
	} else if time.Since(inicio) > time.Second {
		banco.Status, banco.Detalhe = StatusDegradado, "lento"
	}
	cancel()
	st.Componentes = append(st.Componentes, banco)
	st.Componentes = append(st.Componentes, s.statusPagamentos())

	emails := ComponenteStatus{Chave: "emails", Nome: "E-mails (ingressos e avisos)", Status: StatusOperacional}
	if banco.Status != StatusFalha {
		var fila struct{ Atrasados, Falhos int64 }
		s.db.Raw(`SELECT
			count(*) FILTER (WHERE status = 'pendente' AND criado_em < now() - interval '20 minutes') AS atrasados,
			count(*) FILTER (WHERE status = 'falhou' AND criado_em > now() - interval '1 hour') AS falhos
			FROM emails_fila`).Scan(&fila)
		switch {
		case fila.Atrasados > 50 || fila.Falhos > 20:
			emails.Status, emails.Detalhe = StatusFalha, "envio parado"
		case fila.Atrasados > 0 || fila.Falhos > 0:
			emails.Status, emails.Detalhe = StatusDegradado, "envio com atraso"
		}
		s.db.Where("fim_em IS NULL").Order("inicio_em DESC").Find(&st.Avisos)
		s.db.Where("fim_em IS NOT NULL AND fim_em > ?", agora.AddDate(0, 0, -14)).Order("inicio_em DESC").Limit(20).Find(&st.Recentes)
	}
	st.Componentes = append(st.Componentes, emails)

	// Aviso aberto do admin pesa no componente (e no geral).
	for _, a := range st.Avisos {
		nivel := StatusDegradado
		if a.Nivel == "falha" {
			nivel = StatusFalha
		}
		for i := range st.Componentes {
			c := &st.Componentes[i]
			if a.Componente == c.Chave || (a.Componente == "checkin" && c.Chave == "site") || (a.Componente == "repasses" && c.Chave == "pagamentos") {
				c.Status = pior(c.Status, nivel)
			}
		}
		if a.Componente == "geral" {
			st.Geral = pior(st.Geral, nivel)
		}
	}
	if st.Geral == "" {
		st.Geral = StatusOperacional
	}
	for _, c := range st.Componentes {
		st.Geral = pior(st.Geral, c.Status)
	}

	s.mu.Lock()
	s.statusCache, s.statusCacheEm = st, agora
	s.mu.Unlock()
	return st
}

func pior(a, b string) string {
	peso := map[string]int{"": 0, StatusOperacional: 1, StatusDegradado: 2, StatusFalha: 3}
	if peso[b] > peso[a] {
		return b
	}
	return a
}

func (s *AjudaService) statusPagamentos() ComponenteStatus {
	c := ComponenteStatus{Chave: "pagamentos", Nome: "Pagamentos (Pix e cartão)", Status: StatusOperacional}
	if s.mpToken == "" {
		c.Status, c.Detalhe = StatusDegradado, "não configurado"
		return c
	}
	req, err := http.NewRequest(http.MethodGet, s.mpBaseURL+"/v1/payment_methods", nil)
	if err != nil {
		c.Status = StatusFalha
		return c
	}
	req.Header.Set("Authorization", "Bearer "+s.mpToken)
	resp, err := s.httpCliente.Do(req)
	if err != nil {
		c.Status, c.Detalhe = StatusFalha, "Mercado Pago sem resposta"
		return c
	}
	resp.Body.Close()
	switch {
	case resp.StatusCode == http.StatusUnauthorized:
		c.Status, c.Detalhe = StatusFalha, "credencial recusada"
	case resp.StatusCode >= 500:
		c.Status, c.Detalhe = StatusDegradado, "Mercado Pago instável"
	}
	return c
}

// AvisosAbertos: só os avisos (faixa no site), sem as checagens.
func (s *AjudaService) AvisosAbertos() []AvisoStatus {
	lista := []AvisoStatus{}
	s.db.Where("fim_em IS NULL").Order("inicio_em DESC").Limit(5).Find(&lista)
	return lista
}

func (s *AjudaService) CriarAviso(adminID int64, titulo, mensagem, nivel, componente string) (*AvisoStatus, error) {
	titulo, mensagem = strings.TrimSpace(titulo), strings.TrimSpace(mensagem)
	if componente == "" {
		componente = "geral"
	}
	if titulo == "" || utf8.RuneCountInString(titulo) > 120 || utf8.RuneCountInString(mensagem) > 1000 ||
		(nivel != "aviso" && nivel != "falha" && nivel != "manutencao") || !componentesAviso[componente] {
		return nil, ErrDadosAviso
	}
	a := &AvisoStatus{Titulo: titulo, Mensagem: mensagem, Nivel: nivel, Componente: componente, CriadoPor: &adminID}
	if err := s.db.Create(a).Error; err != nil {
		return nil, err
	}
	s.limparCache()
	return a, nil
}

func (s *AjudaService) EncerrarAviso(id int64) error {
	res := s.db.Model(&AvisoStatus{}).Where("id = ? AND fim_em IS NULL", id).Update("fim_em", time.Now())
	if res.Error == nil && res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	s.limparCache()
	return res.Error
}

func (s *AjudaService) limparCache() {
	s.mu.Lock()
	s.statusCache = nil
	s.mu.Unlock()
}
