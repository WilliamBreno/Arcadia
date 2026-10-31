package service

import (
	"errors"
	"fmt"
	"html"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/WilliamBreno/Arcadia/backend/internal/mail"
	"github.com/WilliamBreno/Arcadia/backend/internal/pagamento"
	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
)

// Item 10.4: Plano Pro do organizador e destaque pago na vitrine. São
// receitas da Evve: cobrança própria (referência "PLAT-<id>"), fora de
// pedido, repasse e ledger do evento.

var (
	ErrCobrancaNaoEncontrada = errors.New("cobrança não encontrada")
	ErrCobrancaPaga          = errors.New("esta cobrança já foi paga")
	ErrCobrancaExpirada      = errors.New("esta cobrança expirou; faça uma nova")
	ErrDadosDestaque         = errors.New("destaque inválido")
	ErrDestaqueLotado        = errors.New("não há mais vaga de destaque em um dos dias")
)

const (
	CobrancaProMensal = "pro_mensal"
	CobrancaProAnual  = "pro_anual"
	CobrancaDestaque  = "destaque"

	PrefixoRefPlataforma = "PLAT-"
	validadeCobranca     = time.Hour
	maxDiasDestaque      = 30
	CotaEmailsMesPro     = 50000
)

type CobrancaPlataforma struct {
	ID                      int64 `gorm:"primaryKey"`
	OrganizadorID           int64
	UsuarioID               int64
	Tipo                    string
	Descricao               string
	ValorCentavos           int64
	Status                  string  `gorm:"default:pendente"`
	MPPaymentID             *string `gorm:"column:mp_payment_id"`
	PixMPPaymentID          *string `gorm:"column:pix_mp_payment_id"`
	PixCopiaCola            *string
	PixExpiraEm             *time.Time
	Metodo                  string
	TaxaProcessadorCentavos int64
	ReembolsadoCentavos     int64
	ExpiraEm                time.Time
	PagoEm                  *time.Time
	CriadoEm                time.Time `gorm:"default:now()"`
}

func (CobrancaPlataforma) TableName() string { return "cobrancas_plataforma" }

type Destaque struct {
	ID            int64 `gorm:"primaryKey"`
	EventoID      int64
	OrganizadorID int64
	AlvoTipo      string
	AlvoValor     string
	Dia           time.Time
	ValorCentavos int64
	IncluidoPro   bool
	Status        string `gorm:"default:reservado"`
	CobrancaID    *int64
	CriadoEm      time.Time `gorm:"default:now()"`
}

func (Destaque) TableName() string { return "destaques" }

type PlataformaService struct {
	db          *gorm.DB
	mp          pagamento.Processador
	config      *repository.ConfigPlataformaRepository
	usuarios    *repository.UsuarioRepository
	mail        *mail.Cliente
	backendURL  string
	frontendURL string
}

func NovoPlataformaService(db *gorm.DB, mp pagamento.Processador, config *repository.ConfigPlataformaRepository, usuarios *repository.UsuarioRepository,
	m *mail.Cliente, backendURL, frontendURL string) *PlataformaService {
	return &PlataformaService{db: db, mp: mp, config: config, usuarios: usuarios, mail: m, backendURL: backendURL, frontendURL: strings.TrimSuffix(frontendURL, "/")}
}

func (s *PlataformaService) valor(chave string, padrao int64) int64 {
	if v, err := s.config.BuscarInt64(chave); err == nil && v >= 0 {
		return v
	}
	return padrao
}

type PrecosPlataforma struct {
	ProMensal   int64 `json:"pro_mensal_centavos"`
	ProAnual    int64 `json:"pro_anual_centavos"`
	DestaqueDia int64 `json:"destaque_dia_centavos"`
	VagasDia    int64 `json:"destaque_vagas_dia"`
}

func (s *PlataformaService) Precos() PrecosPlataforma {
	return PrecosPlataforma{ProMensal: s.valor("PRO_MENSAL_CENTAVOS", 4900), ProAnual: s.valor("PRO_ANUAL_CENTAVOS", 49000),
		DestaqueDia: s.valor("DESTAQUE_DIA_CENTAVOS", 1500), VagasDia: s.valor("DESTAQUE_VAGAS_DIA", 3)}
}

// ---------- Pro ----------

// ProAte: até quando o organizador é Pro (nil = não é).
func ProAte(db *gorm.DB, organizadorID int64) *time.Time {
	var fins []time.Time
	db.Raw(`SELECT fim_em FROM periodos_pro WHERE organizador_id = ? AND fim_em > now() ORDER BY fim_em DESC LIMIT 1`, organizadorID).Scan(&fins)
	if len(fins) == 0 {
		return nil
	}
	return &fins[0]
}

func EhPro(db *gorm.DB, organizadorID int64) bool { return ProAte(db, organizadorID) != nil }

// CotaEmailsDoMes: cota do e-mail para quem segue (10× no Pro).
func CotaEmailsDoMes(db *gorm.DB, organizadorID int64) int {
	if EhPro(db, organizadorID) {
		return CotaEmailsMesPro
	}
	return CotaEmailsMes
}

type SituacaoPro struct {
	Pro                   bool             `json:"pro"`
	Ate                   *time.Time       `json:"ate"`
	DestaqueIncluidoUsado bool             `json:"destaque_incluido_usado"`
	Precos                PrecosPlataforma `json:"precos"`
}

func (s *PlataformaService) Situacao(organizadorID int64) SituacaoPro {
	ate := ProAte(s.db, organizadorID)
	return SituacaoPro{Pro: ate != nil, Ate: ate, DestaqueIncluidoUsado: s.usouDestaqueIncluido(s.db, organizadorID), Precos: s.Precos()}
}

// usouDestaqueIncluido: o Pro dá 1 dia de destaque a cada 30 dias.
func (s *PlataformaService) usouDestaqueIncluido(db *gorm.DB, organizadorID int64) bool {
	var n int64
	db.Raw(`SELECT count(*) FROM destaques WHERE organizador_id = ? AND incluido_pro AND status IN ('reservado', 'pago')
		AND criado_em > now() - interval '30 days'`, organizadorID).Scan(&n)
	return n > 0
}

// CriarCobrancaPro: 30 dias ou 12 meses.
func (s *PlataformaService) CriarCobrancaPro(organizadorID, usuarioID int64, anual bool) (*CobrancaPlataforma, error) {
	c := &CobrancaPlataforma{OrganizadorID: organizadorID, UsuarioID: usuarioID, Tipo: CobrancaProMensal,
		Descricao: "Evve Pro · 30 dias", ValorCentavos: s.Precos().ProMensal, ExpiraEm: time.Now().Add(validadeCobranca)}
	if anual {
		c.Tipo, c.Descricao, c.ValorCentavos = CobrancaProAnual, "Evve Pro · 12 meses", s.Precos().ProAnual
	}
	if c.ValorCentavos <= 0 {
		return nil, fmt.Errorf("%w: preço do Pro não configurado", ErrDadosDestaque)
	}
	return c, s.db.Create(c).Error
}

// ---------- destaques ----------

type DadosDestaque struct {
	EventoID int64
	AlvoTipo string   // cidade ou categoria (sempre as do próprio evento)
	Dias     []string // AAAA-MM-DD
}

// alvoDoEvento: o lugar onde o evento pode aparecer (a cidade do local ou
// a categoria do evento), no mesmo formato das URLs da vitrine.
func alvoDoEvento(db *gorm.DB, e *domain.Evento, tipo string) (string, error) {
	switch tipo {
	case "cidade":
		var l struct{ Cidade, UF string }
		if e.LocalID == nil || db.Raw(`SELECT cidade, uf FROM locais WHERE id = ?`, *e.LocalID).Scan(&l).RowsAffected == 0 || l.Cidade == "" {
			return "", fmt.Errorf("%w: o evento precisa de um local com cidade", ErrDadosDestaque)
		}
		return Slugify(l.Cidade) + "-" + strings.ToLower(l.UF), nil
	case "categoria":
		if strings.TrimSpace(e.Categoria) == "" {
			return "", fmt.Errorf("%w: o evento precisa de uma categoria", ErrDadosDestaque)
		}
		return Slugify(e.Categoria), nil
	}
	return "", fmt.Errorf("%w: escolha cidade ou categoria", ErrDadosDestaque)
}

// vagasOcupadas: destaques pagos ou com cobrança ainda em aberto.
func vagasOcupadas(tx *gorm.DB, tipo, valor string, dia time.Time, excetoCobranca int64) int64 {
	var n int64
	tx.Raw(`SELECT count(*) FROM destaques d LEFT JOIN cobrancas_plataforma c ON c.id = d.cobranca_id
		WHERE d.alvo_tipo = ? AND d.alvo_valor = ? AND d.dia = ? AND COALESCE(d.cobranca_id, 0) <> ?
		AND (d.status = 'pago' OR (d.status = 'reservado' AND c.status = 'pendente' AND c.expira_em > now()))`,
		tipo, valor, dia.Format("2006-01-02"), excetoCobranca).Scan(&n)
	return n
}

// CriarCobrancaDestaque reserva os dias (vagas limitadas por lugar e dia)
// e cria a cobrança. Com o Pro, 1 dia a cada 30 sai de graça. Sem nada a
// pagar, confirma na hora.
func (s *PlataformaService) CriarCobrancaDestaque(organizadorID, usuarioID int64, d DadosDestaque) (*CobrancaPlataforma, error) {
	var e domain.Evento
	if s.db.First(&e, d.EventoID).Error != nil || e.OrganizadorID != organizadorID {
		return nil, ErrEventoNaoPertenceAoOrganizador
	}
	agora := time.Now()
	if e.Status != domain.StatusEventoPublicado || e.InicioEm == nil || !e.InicioEm.After(agora) {
		return nil, fmt.Errorf("%w: só evento publicado que ainda não começou", ErrDadosDestaque)
	}
	alvo, err := alvoDoEvento(s.db, &e, d.AlvoTipo)
	if err != nil {
		return nil, err
	}
	hoje := time.Date(agora.Year(), agora.Month(), agora.Day(), 0, 0, 0, 0, time.Local)
	vistos := map[string]bool{}
	var dias []time.Time
	for _, txt := range d.Dias {
		dia, err := time.ParseInLocation("2006-01-02", txt, time.Local)
		if err != nil || vistos[txt] || dia.Before(hoje) || !dia.Before(e.InicioEm.In(time.Local).Add(24*time.Hour)) {
			return nil, fmt.Errorf("%w: dias de hoje até o dia do evento, sem repetir", ErrDadosDestaque)
		}
		vistos[txt] = true
		dias = append(dias, dia)
	}
	if len(dias) == 0 || len(dias) > maxDiasDestaque {
		return nil, fmt.Errorf("%w: de 1 a %d dias", ErrDadosDestaque, maxDiasDestaque)
	}
	precos := s.Precos()
	var c *CobrancaPlataforma
	err = s.db.Transaction(func(tx *gorm.DB) error {
		// Uma compra de destaque por lugar de cada vez (vagas contadas certo).
		if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtext(?))", "destaque:"+d.AlvoTipo+":"+alvo).Error; err != nil {
			return err
		}
		// E uma por organizador (o dia incluído no Pro não sai duas vezes).
		if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtext(?))", "destaque-org:"+strconv.FormatInt(organizadorID, 10)).Error; err != nil {
			return err
		}
		incluido := EhPro(tx, organizadorID) && !s.usouDestaqueIncluido(tx, organizadorID)
		total := int64(0)
		for i := range dias {
			if vagasOcupadas(tx, d.AlvoTipo, alvo, dias[i], 0) >= precos.VagasDia {
				return fmt.Errorf("%w (%s)", ErrDestaqueLotado, dias[i].Format("02/01"))
			}
			var ja int64
			tx.Raw(`SELECT count(*) FROM destaques d LEFT JOIN cobrancas_plataforma c ON c.id = d.cobranca_id
				WHERE d.evento_id = ? AND d.alvo_tipo = ? AND d.alvo_valor = ? AND d.dia = ?
				AND (d.status = 'pago' OR (d.status = 'reservado' AND c.status = 'pendente' AND c.expira_em > now()))`,
				e.ID, d.AlvoTipo, alvo, dias[i].Format("2006-01-02")).Scan(&ja)
			if ja > 0 {
				return fmt.Errorf("%w: o evento já está em destaque em %s", ErrDadosDestaque, dias[i].Format("02/01"))
			}
			if !(incluido && i == 0) {
				total += precos.DestaqueDia
			}
		}
		// Reservas antigas que não foram pagas saem do caminho do índice único.
		if err := tx.Exec(`UPDATE destaques d SET status = 'cancelado' FROM cobrancas_plataforma c
			WHERE c.id = d.cobranca_id AND d.evento_id = ? AND d.status = 'reservado' AND (c.status <> 'pendente' OR c.expira_em <= now())`, e.ID).Error; err != nil {
			return err
		}
		lugar := map[string]string{"cidade": "na cidade", "categoria": "na categoria"}[d.AlvoTipo]
		c = &CobrancaPlataforma{OrganizadorID: organizadorID, UsuarioID: usuarioID, Tipo: CobrancaDestaque,
			Descricao:     limitarTexto(fmt.Sprintf("Destaque %s · %s · %d dia(s)", lugar, e.Titulo, len(dias)), 200),
			ValorCentavos: total, ExpiraEm: agora.Add(validadeCobranca)}
		if total == 0 {
			c.Status, c.PagoEm = "paga", &agora
		}
		if err := tx.Create(c).Error; err != nil {
			return err
		}
		for i, dia := range dias {
			v := precos.DestaqueDia
			inc := incluido && i == 0
			if inc {
				v = 0
			}
			status := "reservado"
			if total == 0 {
				status = "pago"
			}
			if err := tx.Create(&Destaque{EventoID: e.ID, OrganizadorID: organizadorID, AlvoTipo: d.AlvoTipo, AlvoValor: alvo, Dia: dia,
				ValorCentavos: v, IncluidoPro: inc, Status: status, CobrancaID: &c.ID}).Error; err != nil {
				return err
			}
		}
		return nil
	})
	return c, err
}

// Patrocinados: eventos com destaque pago hoje no lugar.
func (s *PlataformaService) Patrocinados(alvoTipo, alvoValor string, limite int) ([]domain.Evento, error) {
	var eventos []domain.Evento
	err := s.db.Raw(`SELECT e.* FROM eventos e JOIN destaques d ON d.evento_id = e.id
		WHERE d.alvo_tipo = ? AND d.alvo_valor = ? AND d.dia = CURRENT_DATE AND d.status = 'pago'
		AND e.status = 'publicado' AND e.visibilidade = 'publico' AND e.inicio_em > now()
		ORDER BY d.criado_em LIMIT ?`, alvoTipo, alvoValor, limite).Scan(&eventos).Error
	return eventos, err
}

type DestaqueView struct {
	ID           int64     `json:"id"`
	EventoID     int64     `json:"evento_id"`
	EventoTitulo string    `json:"evento_titulo"`
	AlvoTipo     string    `json:"alvo_tipo"`
	AlvoValor    string    `json:"alvo_valor"`
	Dia          time.Time `json:"dia"`
	Status       string    `json:"status"`
	IncluidoPro  bool      `json:"incluido_pro"`
}

func (s *PlataformaService) MeusDestaques(organizadorID int64) ([]DestaqueView, error) {
	lista := []DestaqueView{}
	err := s.db.Raw(`SELECT d.id, d.evento_id, e.titulo AS evento_titulo, d.alvo_tipo, d.alvo_valor, d.dia, d.status, d.incluido_pro
		FROM destaques d JOIN eventos e ON e.id = d.evento_id LEFT JOIN cobrancas_plataforma c ON c.id = d.cobranca_id
		WHERE d.organizador_id = ? AND (d.status IN ('pago', 'devolvido') OR (d.status = 'reservado' AND c.status = 'pendente' AND c.expira_em > now()))
		ORDER BY d.dia DESC LIMIT 200`, organizadorID).Scan(&lista).Error
	return lista, err
}

// Disponibilidade: vagas livres por dia para o lugar do evento.
func (s *PlataformaService) Disponibilidade(organizadorID, eventoID int64, alvoTipo string) (map[string]int64, error) {
	var e domain.Evento
	if s.db.First(&e, eventoID).Error != nil || e.OrganizadorID != organizadorID {
		return nil, ErrEventoNaoPertenceAoOrganizador
	}
	alvo, err := alvoDoEvento(s.db, &e, alvoTipo)
	if err != nil {
		return nil, err
	}
	out := map[string]int64{}
	if e.InicioEm == nil {
		return out, nil
	}
	vagas := s.Precos().VagasDia
	agora := time.Now()
	dia := time.Date(agora.Year(), agora.Month(), agora.Day(), 0, 0, 0, 0, time.Local)
	for i := 0; i < maxDiasDestaque && !dia.After(e.InicioEm.In(time.Local)); i++ {
		out[dia.Format("2006-01-02")] = max(vagas-vagasOcupadas(s.db, alvoTipo, alvo, dia, 0), 0)
		dia = dia.AddDate(0, 0, 1)
	}
	return out, nil
}

// ---------- pagamento ----------

func (s *PlataformaService) cobrancaDoOrg(organizadorID, id int64) (*CobrancaPlataforma, error) {
	var c CobrancaPlataforma
	if s.db.Where("id = ? AND organizador_id = ?", id, organizadorID).First(&c).Error != nil {
		return nil, ErrCobrancaNaoEncontrada
	}
	return &c, nil
}

func (s *PlataformaService) Cobranca(organizadorID, id int64) (*CobrancaPlataforma, error) {
	return s.cobrancaDoOrg(organizadorID, id)
}

func (s *PlataformaService) pagavel(organizadorID, id int64) (*CobrancaPlataforma, error) {
	c, err := s.cobrancaDoOrg(organizadorID, id)
	if err != nil {
		return nil, err
	}
	if c.Status != "pendente" {
		return nil, ErrCobrancaPaga
	}
	if !time.Now().Before(c.ExpiraEm) {
		return nil, ErrCobrancaExpirada
	}
	return c, nil
}

func (s *PlataformaService) email(usuarioID int64) string {
	if u, err := s.usuarios.BuscarPorID(usuarioID); err == nil {
		return u.Email
	}
	return ""
}

func refPlataforma(id int64) string { return PrefixoRefPlataforma + strconv.FormatInt(id, 10) }

func (s *PlataformaService) GerarPix(organizadorID, id int64) (*PixNaTela, error) {
	c, err := s.pagavel(organizadorID, id)
	if err != nil {
		return nil, err
	}
	agora := time.Now()
	if c.PixCopiaCola != nil && c.PixExpiraEm != nil && c.PixExpiraEm.After(agora.Add(time.Minute)) {
		return &PixNaTela{CopiaCola: *c.PixCopiaCola, ExpiraEm: *c.PixExpiraEm}, nil
	}
	expira := c.ExpiraEm
	if expira.Sub(agora) > ValidadePix {
		expira = agora.Add(ValidadePix)
	}
	criado, err := s.mp.CriarPagamento(pagamento.NovoPagamento{
		Referencia: refPlataforma(c.ID), Descricao: c.Descricao, ValorCentavos: c.ValorCentavos, Metodo: pagamento.MetodoPix,
		EmailPagador: s.email(c.UsuarioID), ExpiraEm: expira,
		ChaveIdempotencia: "plat-pix-" + strconv.FormatInt(c.ID, 10) + "-" + strconv.FormatInt(agora.Unix()/60, 10),
		URLNotificacao:    s.backendURL + "/api/v1/webhooks/mercadopago",
	})
	if err != nil {
		return nil, err
	}
	if criado.PixCopiaCola == "" {
		return nil, errors.New("o Mercado Pago não devolveu o código do Pix")
	}
	s.db.Model(c).Updates(map[string]any{"pix_mp_payment_id": criado.ID, "pix_copia_cola": criado.PixCopiaCola, "pix_expira_em": criado.PixExpiraEm})
	return &PixNaTela{CopiaCola: criado.PixCopiaCola, ExpiraEm: criado.PixExpiraEm}, nil
}

func (s *PlataformaService) PagarCartao(organizadorID, id int64, d DadosCartao) (*ResultadoCartao, error) {
	if strings.TrimSpace(d.Token) == "" || strings.TrimSpace(d.BandeiraID) == "" {
		return nil, ErrDadosCartaoIncompletos
	}
	c, err := s.pagavel(organizadorID, id)
	if err != nil {
		return nil, err
	}
	criado, err := s.mp.CriarPagamento(pagamento.NovoPagamento{
		Parcelas: 1, Referencia: refPlataforma(c.ID), Descricao: c.Descricao, ValorCentavos: c.ValorCentavos, Metodo: pagamento.MetodoCartao,
		EmailPagador: s.email(c.UsuarioID), CartaoToken: d.Token, BandeiraID: d.BandeiraID, EmissorID: d.EmissorID,
		DocumentoTipo: d.DocumentoTipo, DocumentoNumero: d.DocumentoNumero,
		ChaveIdempotencia: chaveCartao(-c.ID, d.Token), URLNotificacao: s.backendURL + "/api/v1/webhooks/mercadopago",
	})
	if err != nil {
		return &ResultadoCartao{Situacao: "recusado", Mensagem: "Não foi possível processar o cartão agora. Tente de novo ou pague com Pix."}, nil
	}
	switch {
	case criado.Status == pagamento.StatusAprovado:
		if err := s.ConfirmarPorID(criado.ID); err != nil {
			return nil, err
		}
		return &ResultadoCartao{Situacao: "aprovado", PagamentoID: criado.ID}, nil
	case criado.StatusDetalhe == pagamento.DetalheDesafio3DS:
		return &ResultadoCartao{Situacao: "desafio", PagamentoID: criado.ID, DesafioURL: criado.DesafioURL, DesafioCreq: criado.DesafioCreq}, nil
	case criado.Status == pagamento.StatusRecusado:
		return &ResultadoCartao{Situacao: "recusado", PagamentoID: criado.ID, Mensagem: MensagemRecusa(criado.StatusDetalhe)}, nil
	}
	return &ResultadoCartao{Situacao: "em_analise", PagamentoID: criado.ID, Mensagem: "Pagamento em análise pelo banco. Avisamos quando for confirmado."}, nil
}

// Verificar reconsulta o pagamento (Pix ou cartão depois do banco).
func (s *PlataformaService) Verificar(organizadorID, id int64, pagamentoID string) (*CobrancaPlataforma, error) {
	c, err := s.cobrancaDoOrg(organizadorID, id)
	if err != nil {
		return nil, err
	}
	if c.Status != "pendente" {
		return c, nil
	}
	if pagamentoID == "" {
		if c.PixMPPaymentID == nil {
			return c, nil
		}
		pagamentoID = *c.PixMPPaymentID
	}
	p, err := s.mp.BuscarPagamento(pagamentoID)
	if err != nil {
		return nil, err
	}
	if p.Referencia != refPlataforma(c.ID) {
		return nil, ErrCobrancaNaoEncontrada
	}
	if err := s.Confirmar(p); err != nil {
		return nil, err
	}
	return s.cobrancaDoOrg(organizadorID, id)
}

func (s *PlataformaService) ConfirmarPorID(mpID string) error {
	p, err := s.mp.BuscarPagamento(mpID)
	if err != nil {
		return err
	}
	return s.Confirmar(p)
}

// Confirmar aplica um pagamento aprovado (webhook, cartão ou verificação).
// Idempotente. Pago depois do prazo: o Pro vale igual; o destaque vale se
// os dias ainda estão livres, senão o valor é devolvido.
func (s *PlataformaService) Confirmar(p *pagamento.Pagamento) error {
	if p.Status != pagamento.StatusAprovado || !strings.HasPrefix(p.Referencia, PrefixoRefPlataforma) {
		return nil
	}
	id, err := strconv.ParseInt(strings.TrimPrefix(p.Referencia, PrefixoRefPlataforma), 10, 64)
	if err != nil {
		return fmt.Errorf("referência inválida: %q", p.Referencia)
	}
	devolver := false
	var c CobrancaPlataforma
	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&c, id).Error; err != nil {
			return ErrCobrancaNaoEncontrada
		}
		if c.MPPaymentID != nil && *c.MPPaymentID == p.ID {
			return nil // já aplicado
		}
		if c.Status != "pendente" || p.ValorCentavos != c.ValorCentavos {
			devolver = true // pagamento em dobro ou valor estranho
			return nil
		}
		agora := time.Now()
		if c.Tipo == CobrancaDestaque && !agora.Before(c.ExpiraEm) {
			var dias []Destaque
			tx.Where("cobranca_id = ?", c.ID).Find(&dias)
			for _, d := range dias {
				if d.Dia.Before(time.Date(agora.Year(), agora.Month(), agora.Day(), 0, 0, 0, 0, time.Local)) ||
					vagasOcupadas(tx, d.AlvoTipo, d.AlvoValor, d.Dia, c.ID) >= s.Precos().VagasDia {
					devolver = true
					return nil
				}
			}
		}
		if err := tx.Model(&c).Updates(map[string]any{"status": "paga", "mp_payment_id": p.ID, "metodo": p.Metodo,
			"taxa_processador_centavos": p.TaxaProcessadorCentavos, "pago_em": agora}).Error; err != nil {
			return err
		}
		switch c.Tipo {
		case CobrancaProMensal, CobrancaProAnual:
			inicio := agora
			if ate := ProAte(tx, c.OrganizadorID); ate != nil && ate.After(inicio) {
				inicio = *ate // renovação: emenda no fim do período atual
			}
			fim := inicio.AddDate(0, 0, 30)
			if c.Tipo == CobrancaProAnual {
				fim = inicio.AddDate(1, 0, 0)
			}
			return tx.Exec(`INSERT INTO periodos_pro (organizador_id, inicio_em, fim_em, cobranca_id) VALUES (?, ?, ?, ?)`,
				c.OrganizadorID, inicio, fim, c.ID).Error
		case CobrancaDestaque:
			return tx.Model(&Destaque{}).Where("cobranca_id = ?", c.ID).Update("status", "pago").Error
		}
		return nil
	})
	if err != nil {
		return err
	}
	if devolver {
		slog.Warn("plataforma: pagamento devolvido (cobrança já paga, valor diferente ou destaque sem vaga)", "cobranca_id", id, "mp", p.ID)
		if _, err := s.mp.Devolver(p.ID, "plat-devolve-"+p.ID); err != nil {
			slog.Error("plataforma: falha ao devolver; o admin precisa estornar", "cobranca_id", id, "mp", p.ID, "erro", err)
		}
	}
	return nil
}

// ReembolsarDestaquesDoEvento: evento cancelado devolve os dias de
// destaque depois de hoje (hoje já está no ar; os incluídos no Pro só saem).
func (s *PlataformaService) ReembolsarDestaquesDoEvento(eventoID int64) {
	var linhas []struct {
		CobrancaID  int64
		MPPaymentID string
		Valor       int64
	}
	s.db.Raw(`SELECT d.cobranca_id, c.mp_payment_id, SUM(d.valor_centavos) AS valor FROM destaques d
		JOIN cobrancas_plataforma c ON c.id = d.cobranca_id
		WHERE d.evento_id = ? AND d.status = 'pago' AND d.dia > CURRENT_DATE AND c.mp_payment_id IS NOT NULL
		GROUP BY d.cobranca_id, c.mp_payment_id`, eventoID).Scan(&linhas)
	for _, l := range linhas {
		if l.Valor > 0 {
			if _, err := s.mp.Reembolsar(l.MPPaymentID, l.Valor); err != nil {
				slog.Error("destaque: estorno do evento cancelado falhou; o admin precisa estornar", "cobranca_id", l.CobrancaID, "erro", err)
				continue
			}
			s.db.Exec(`UPDATE cobrancas_plataforma SET reembolsado_centavos = reembolsado_centavos + ? WHERE id = ?`, l.Valor, l.CobrancaID)
		}
		s.db.Exec(`UPDATE destaques SET status = 'devolvido' WHERE cobranca_id = ? AND status = 'pago' AND dia > CURRENT_DATE`, l.CobrancaID)
	}
	s.db.Exec(`UPDATE destaques SET status = 'cancelado' WHERE evento_id = ? AND status IN ('pago', 'reservado') AND dia >= CURRENT_DATE AND valor_centavos = 0`, eventoID)
}

// AvisarVencimentoPro: e-mail 5 dias antes de o Pro acabar (uma vez por
// período: marca o último período de cada organizador).
func (s *PlataformaService) AvisarVencimentoPro(agora time.Time) int {
	var linhas []struct {
		ID    int64
		FimEm time.Time
		Email string
		Nome  string
	}
	s.db.Raw(`SELECT p.id, p.fim_em, u.email, o.nome_publico AS nome FROM periodos_pro p
		JOIN organizadores o ON o.id = p.organizador_id JOIN usuarios u ON u.id = o.usuario_id
		WHERE p.aviso_vencimento_em IS NULL AND p.fim_em BETWEEN ? AND ?
		AND NOT EXISTS (SELECT 1 FROM periodos_pro q WHERE q.organizador_id = p.organizador_id AND q.fim_em > p.fim_em)`,
		agora, agora.Add(5*24*time.Hour)).Scan(&linhas)
	n := 0
	for _, l := range linhas {
		res := s.db.Exec(`UPDATE periodos_pro SET aviso_vencimento_em = now() WHERE id = ? AND aviso_vencimento_em IS NULL`, l.ID)
		if res.Error != nil || res.RowsAffected == 0 {
			continue
		}
		corpo := fmt.Sprintf(`<div style="font-family:Arial,sans-serif;max-width:560px;margin:auto;color:#111"><p>Olá, %s!</p>
			<p>Seu Evve Pro vale até <strong>%s</strong>. Para continuar com os benefícios, renove no painel: <a href="%s/organizador/pro">Evve Pro</a>.</p>
			<p style="color:#777;font-size:12px">Sem renovar, a conta volta ao plano grátis; nada do que você já tem é apagado.</p></div>`,
			html.EscapeString(l.Nome), l.FimEm.In(time.Local).Format("02/01/2006"), s.frontendURL)
		if s.mail.Enviar(l.Email, "Seu Evve Pro vence em breve", corpo) == nil {
			n++
		}
	}
	return n
}
