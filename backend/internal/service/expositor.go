package service

// Item 8.6: artist alley e expositores. Decisões do dono (30/10/2026):
//   - candidatura com curadoria: o organizador aprova, escolhe a mesa e o
//     expositor tem 48 h para pagar (quem é recusado nunca paga);
//   - mesma taxa do ingresso, paga pelo expositor;
//   - reembolso igual ao do ingresso (garantia opcional; sem ela, só o
//     arrependimento de 7 dias); remoção pelo organizador ou evento
//     cancelado = tudo de volta;
//   - mesas numeradas por setor e credenciais com QR (expositor + ajudantes).
//
// O pagamento usa o mesmo pedido dos ingressos, com um "ingresso interno"
// inativo por evento (tipos_ingresso.espaco_expositor) e o item ligado ao
// expositor (itens_pedido.expositor_id).

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"html"
	"log/slog"
	"net/url"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/WilliamBreno/Arcadia/backend/internal/mail"
	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
)

const (
	StatusExpositorPendente            = "pendente"
	StatusExpositorAguardandoPagamento = "aguardando_pagamento"
	StatusExpositorAprovado            = "aprovado"
	StatusExpositorRejeitado           = "rejeitado"
	StatusExpositorDesistiu            = "desistiu"

	prefixoQRCredencial = "EVVEX"
	maxMesasPorLote     = 200
)

var (
	ErrExpositorNaoEncontrado = errors.New("expositor não encontrado")
	ErrExpositoresFechados    = errors.New("este evento não está recebendo expositores")
	ErrJaCandidatado          = errors.New("você já se candidatou neste evento")
	ErrDadosExpositor         = errors.New("dados do expositor inválidos")
	ErrEspacoInvalido         = errors.New("mesa ou espaço inválido")
	ErrEspacoOcupado          = errors.New("esta mesa já está com outro expositor")
	ErrTipoEspacoEmUso        = errors.New("este tipo de espaço já tem mesas ou candidaturas")
	ErrQRCredencial           = errors.New("credencial inválida")
	ErrEspacoSemPagamento     = errors.New("este espaço não tem pagamento pendente")
)

type TipoEspaco struct {
	ID            int64 `gorm:"primaryKey"`
	EventoID      int64
	Nome          string
	Descricao     string
	PrecoCentavos int64
	Credenciais   int
	Ativo         bool
	CriadoEm      time.Time
}

func (TipoEspaco) TableName() string { return "tipos_espaco" }

type Espaco struct {
	ID           int64 `gorm:"primaryKey"`
	EventoID     int64
	TipoEspacoID int64
	Setor        string
	Codigo       string
	Ordem        int
}

func (Espaco) TableName() string { return "espacos" }

type Expositor struct {
	ID               int64 `gorm:"primaryKey"`
	EventoID         int64
	UsuarioID        int64
	TipoEspacoID     int64
	EspacoID         *int64
	Marca            string
	Categoria        string
	Descricao        string
	PortfolioURL     string `gorm:"column:portfolio_url"`
	Instagram        string
	ResponsavelNome  string
	Telefone         string
	Status           string
	MotivoRejeicao   string
	PrazoPagamentoEm *time.Time
	CriadoEm         time.Time
}

func (Expositor) TableName() string { return "expositores" }

type CredencialExpositor struct {
	ID                int64 `gorm:"primaryKey"`
	ExpositorID       int64
	Nome              string
	PrimeiraEntradaEm *time.Time
	UltimaEntradaEm   *time.Time
	Ativa             bool
	CriadoEm          time.Time
}

func (CredencialExpositor) TableName() string { return "credenciais_expositor" }

// AvisoEspaco manda os e-mails do expositor (implementado pelo ExpositorService).
type AvisoEspaco interface {
	AvisarEspacoPago(expositorID int64)
	AvisarPrazoEspacoVencido(expositorID int64)
}

// DefinirAvisoEspaco liga os e-mails do espaço de expositor (8.6).
func (s *CheckoutService) DefinirAvisoEspaco(a AvisoEspaco) { s.avisoEspaco = a }

// --- QR da credencial (assinado, nada guardado) ---

func tokenCredencial(segredo string, id int64) string {
	m := hmac.New(sha256.New, []byte(segredo))
	fmt.Fprintf(m, "credencial-expositor:%d", id)
	return hex.EncodeToString(m.Sum(nil))[:24]
}

// PayloadCredencial: EVVEX:<credencial>:<assinatura>. Testada.
func PayloadCredencial(segredo string, id int64) string {
	return prefixoQRCredencial + ":" + strconv.FormatInt(id, 10) + ":" + tokenCredencial(segredo, id)
}

// LerPayloadCredencial confere a assinatura. Testada.
func LerPayloadCredencial(segredo, payload string) (int64, error) {
	partes := strings.Split(strings.TrimSpace(payload), ":")
	if len(partes) != 3 || partes[0] != prefixoQRCredencial {
		return 0, ErrQRCredencial
	}
	id, err := strconv.ParseInt(partes[1], 10, 64)
	if err != nil || !hmac.Equal([]byte(partes[2]), []byte(tokenCredencial(segredo, id))) {
		return 0, ErrQRCredencial
	}
	return id, nil
}

// LiberarExpositor tira o expositor da mesa (desistiu, recusado, prazo):
// a mesa volta a ficar livre e as credenciais deixam de valer.
func LiberarExpositor(db *gorm.DB, expositorID int64, status, motivo string) {
	db.Model(&Expositor{}).Where("id = ?", expositorID).
		Updates(map[string]any{"status": status, "motivo_rejeicao": motivo, "espaco_id": nil, "prazo_pagamento_em": nil})
	db.Model(&CredencialExpositor{}).Where("expositor_id = ?", expositorID).Update("ativa", false)
}

// expositoresDoPedido trava os expositores pagos por este pedido e diz se
// todos ainda esperam o pagamento (como fichasDoPedido, item 8.4).
func expositoresDoPedido(tx *gorm.DB, pedidoID int64) (ids []int64, esperando bool, err error) {
	if err = tx.Model(&domain.ItemPedido{}).Where("pedido_id = ? AND expositor_id IS NOT NULL", pedidoID).
		Distinct().Pluck("expositor_id", &ids).Error; err != nil || len(ids) == 0 {
		return nil, true, err
	}
	var es []Expositor
	if err = tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id IN ?", ids).Find(&es).Error; err != nil {
		return nil, false, err
	}
	if len(es) != len(ids) {
		return ids, false, nil
	}
	for _, e := range es {
		if e.Status != StatusExpositorAguardandoPagamento || e.EspacoID == nil {
			return ids, false, nil
		}
	}
	return ids, true, nil
}

// confirmarExpositoresPagos aprova e cria as credenciais.
func confirmarExpositoresPagos(tx *gorm.DB, ids []int64) error {
	for _, id := range ids {
		if err := aprovarExpositor(tx, id); err != nil {
			return err
		}
	}
	return nil
}

func aprovarExpositor(tx *gorm.DB, id int64) error {
	var e Expositor
	if err := tx.First(&e, id).Error; err != nil {
		return err
	}
	if err := tx.Model(&e).Updates(map[string]any{"status": StatusExpositorAprovado, "prazo_pagamento_em": nil, "motivo_rejeicao": ""}).Error; err != nil {
		return err
	}
	var existentes int64
	tx.Model(&CredencialExpositor{}).Where("expositor_id = ?", id).Count(&existentes)
	if existentes > 0 {
		return tx.Model(&CredencialExpositor{}).Where("expositor_id = ?", id).Update("ativa", true).Error
	}
	var t TipoEspaco
	if err := tx.First(&t, e.TipoEspacoID).Error; err != nil {
		return err
	}
	creds := make([]CredencialExpositor, 0, t.Credenciais)
	for i := 0; i < t.Credenciais; i++ {
		nome := ""
		if i == 0 {
			nome = e.ResponsavelNome
		}
		creds = append(creds, CredencialExpositor{ExpositorID: id, Nome: nome, Ativa: true, CriadoEm: time.Now()})
	}
	return tx.Create(&creds).Error
}

type ExpositorService struct {
	db           *gorm.DB
	eventos      *repository.EventoRepository
	usuarios     *repository.UsuarioRepository
	checkout     *CheckoutService
	cancelamento *CancelamentoService
	checkin      *CheckinService
	mailCliente  *mail.Cliente
	segredo      string
	frontendURL  string
}

func NovoExpositorService(db *gorm.DB, e *repository.EventoRepository, u *repository.UsuarioRepository, co *CheckoutService,
	ca *CancelamentoService, ch *CheckinService, m *mail.Cliente, segredo, frontendURL string) *ExpositorService {
	return &ExpositorService{db: db, eventos: e, usuarios: u, checkout: co, cancelamento: ca, checkin: ch, mailCliente: m,
		segredo: segredo, frontendURL: strings.TrimRight(frontendURL, "/")}
}

func (s *ExpositorService) doOrganizador(organizadorID, eventoID int64) (*domain.Evento, error) {
	e, err := s.eventos.BuscarPorID(eventoID)
	if err != nil || e.OrganizadorID != organizadorID {
		return nil, ErrEventoNaoPertenceAoOrganizador
	}
	return e, nil
}

// ---------- organizador: tipos e mesas ----------

type DadosTipoEspaco struct {
	Nome          string
	Descricao     string
	PrecoCentavos int64
	Credenciais   int
	Ativo         bool
}

func validarTipoEspaco(d DadosTipoEspaco) error {
	n := len([]rune(strings.TrimSpace(d.Nome)))
	if n < 2 || n > 80 || d.PrecoCentavos < 0 || (d.PrecoCentavos > 0 && d.PrecoCentavos < 100) || d.PrecoCentavos > 10000000 ||
		d.Credenciais < 1 || d.Credenciais > 10 {
		return ErrDadosExpositor
	}
	return nil
}

func (s *ExpositorService) CriarTipo(organizadorID, eventoID int64, d DadosTipoEspaco) (*TipoEspaco, error) {
	if _, err := s.doOrganizador(organizadorID, eventoID); err != nil {
		return nil, err
	}
	if err := validarTipoEspaco(d); err != nil {
		return nil, err
	}
	t := &TipoEspaco{EventoID: eventoID, Nome: strings.TrimSpace(d.Nome), Descricao: strings.TrimSpace(d.Descricao),
		PrecoCentavos: d.PrecoCentavos, Credenciais: d.Credenciais, Ativo: d.Ativo, CriadoEm: time.Now()}
	return t, s.db.Create(t).Error
}

// AtualizarTipo: o preço novo vale para quem ainda vai pagar.
func (s *ExpositorService) AtualizarTipo(organizadorID, eventoID, tipoID int64, d DadosTipoEspaco) (*TipoEspaco, error) {
	if _, err := s.doOrganizador(organizadorID, eventoID); err != nil {
		return nil, err
	}
	if err := validarTipoEspaco(d); err != nil {
		return nil, err
	}
	var t TipoEspaco
	if err := s.db.Where("id = ? AND evento_id = ?", tipoID, eventoID).First(&t).Error; err != nil {
		return nil, ErrEspacoInvalido
	}
	t.Nome, t.Descricao, t.PrecoCentavos, t.Credenciais, t.Ativo = strings.TrimSpace(d.Nome), strings.TrimSpace(d.Descricao), d.PrecoCentavos, d.Credenciais, d.Ativo
	return &t, s.db.Save(&t).Error
}

func (s *ExpositorService) ExcluirTipo(organizadorID, eventoID, tipoID int64) error {
	if _, err := s.doOrganizador(organizadorID, eventoID); err != nil {
		return err
	}
	var n int64
	s.db.Model(&Espaco{}).Where("tipo_espaco_id = ?", tipoID).Count(&n)
	var m int64
	s.db.Model(&Expositor{}).Where("tipo_espaco_id = ?", tipoID).Count(&m)
	if n+m > 0 {
		return ErrTipoEspacoEmUso
	}
	res := s.db.Where("id = ? AND evento_id = ?", tipoID, eventoID).Delete(&TipoEspaco{})
	if res.RowsAffected == 0 {
		return ErrEspacoInvalido
	}
	return res.Error
}

// CriarMesas cria em lote: setor "A", de 1 a 20 -> A1..A20 (pula as que já existem).
func (s *ExpositorService) CriarMesas(organizadorID, eventoID, tipoID int64, setor string, de, ate int) (int, error) {
	if _, err := s.doOrganizador(organizadorID, eventoID); err != nil {
		return 0, err
	}
	setor = strings.ToUpper(strings.TrimSpace(setor))
	if setor == "" || len(setor) > 10 || de < 1 || ate < de || ate-de+1 > maxMesasPorLote {
		return 0, fmt.Errorf("%w: setor de até 10 letras e até %d mesas por vez", ErrEspacoInvalido, maxMesasPorLote)
	}
	var t TipoEspaco
	if err := s.db.Where("id = ? AND evento_id = ?", tipoID, eventoID).First(&t).Error; err != nil {
		return 0, ErrEspacoInvalido
	}
	novas := 0
	for n := de; n <= ate; n++ {
		e := Espaco{EventoID: eventoID, TipoEspacoID: tipoID, Setor: setor, Codigo: setor + strconv.Itoa(n), Ordem: n}
		res := s.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&e)
		if res.Error != nil {
			return novas, res.Error
		}
		novas += int(res.RowsAffected)
	}
	return novas, nil
}

func (s *ExpositorService) ExcluirMesa(organizadorID, eventoID, espacoID int64) error {
	if _, err := s.doOrganizador(organizadorID, eventoID); err != nil {
		return err
	}
	var n int64
	s.db.Model(&Expositor{}).Where("espaco_id = ?", espacoID).Count(&n)
	if n > 0 {
		return ErrEspacoOcupado
	}
	res := s.db.Where("id = ? AND evento_id = ?", espacoID, eventoID).Delete(&Espaco{})
	if res.RowsAffected == 0 {
		return ErrEspacoInvalido
	}
	return res.Error
}

// Painel do organizador: tipos, mesas e candidaturas.
type PainelExpositores struct {
	Tipos       []TipoEspaco
	Mesas       []Espaco
	Expositores []Expositor
}

func (s *ExpositorService) Painel(organizadorID, eventoID int64) (*PainelExpositores, error) {
	if _, err := s.doOrganizador(organizadorID, eventoID); err != nil {
		return nil, err
	}
	p := &PainelExpositores{}
	if err := s.db.Where("evento_id = ?", eventoID).Order("id").Find(&p.Tipos).Error; err != nil {
		return nil, err
	}
	if err := s.db.Where("evento_id = ?", eventoID).Order("setor, ordem, codigo").Find(&p.Mesas).Error; err != nil {
		return nil, err
	}
	if err := s.db.Where("evento_id = ?", eventoID).Order("criado_em").Find(&p.Expositores).Error; err != nil {
		return nil, err
	}
	return p, nil
}

// ---------- organizador: curadoria ----------

func (s *ExpositorService) expositorDoEvento(eventoID, expositorID int64) (*Expositor, error) {
	var e Expositor
	if err := s.db.Where("id = ? AND evento_id = ?", expositorID, eventoID).First(&e).Error; err != nil {
		return nil, ErrExpositorNaoEncontrado
	}
	return &e, nil
}

func (s *ExpositorService) espacoPago(expositorID int64) bool {
	var n int64
	s.db.Model(&domain.ItemPedido{}).Where("expositor_id = ? AND status IN ?", expositorID,
		[]domain.StatusItemPedido{domain.StatusItemPago, domain.StatusItemUtilizado}).Count(&n)
	return n > 0
}

// Aprovar coloca o expositor numa mesa. Espaço gratuito: aprovado na hora;
// pago: 48 h para pagar. Já pago: só troca de mesa (do mesmo preço).
func (s *ExpositorService) Aprovar(organizadorID, eventoID, expositorID, espacoID int64) (*Expositor, error) {
	evento, err := s.doOrganizador(organizadorID, eventoID)
	if err != nil {
		return nil, err
	}
	e, err := s.expositorDoEvento(eventoID, expositorID)
	if err != nil {
		return nil, err
	}
	var mesa Espaco
	if err := s.db.Where("id = ? AND evento_id = ?", espacoID, eventoID).First(&mesa).Error; err != nil {
		return nil, ErrEspacoInvalido
	}
	var tipo TipoEspaco
	if err := s.db.First(&tipo, mesa.TipoEspacoID).Error; err != nil {
		return nil, ErrEspacoInvalido
	}
	pago := s.espacoPago(e.ID)
	if pago && tipo.ID != e.TipoEspacoID {
		var atual TipoEspaco
		if s.db.First(&atual, e.TipoEspacoID).Error != nil || atual.PrecoCentavos != tipo.PrecoCentavos {
			return nil, fmt.Errorf("%w: quem já pagou só troca para uma mesa do mesmo preço", ErrEspacoInvalido)
		}
	}
	mudancas := map[string]any{"espaco_id": mesa.ID, "tipo_espaco_id": tipo.ID, "motivo_rejeicao": ""}
	avisar := false
	switch {
	case pago:
		mudancas["status"] = StatusExpositorAprovado
	case tipo.PrecoCentavos == 0:
		mudancas["status"] = StatusExpositorAprovado
		avisar = true
	default:
		prazo := PrazoParaPagar(time.Now(), evento.InicioEm)
		mudancas["status"] = StatusExpositorAguardandoPagamento
		mudancas["prazo_pagamento_em"] = prazo
		avisar = true
	}
	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&Expositor{}).Where("id = ?", e.ID).Updates(mudancas).Error; err != nil {
			if strings.Contains(err.Error(), "expositores_espaco_un") {
				return ErrEspacoOcupado
			}
			return err
		}
		if mudancas["status"] == StatusExpositorAprovado && !pago {
			return aprovarExpositor(tx, e.ID)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	e, _ = s.expositorDoEvento(eventoID, e.ID)
	if avisar {
		s.avisarStatus(e)
	}
	return e, nil
}

// Rejeitar recusa (ou remove) o expositor. Se pagou, devolve tudo antes;
// sem o estorno confirmado, nada muda.
func (s *ExpositorService) Rejeitar(organizadorID, eventoID, expositorID int64, motivo string) (*Expositor, error) {
	if _, err := s.doOrganizador(organizadorID, eventoID); err != nil {
		return nil, err
	}
	e, err := s.expositorDoEvento(eventoID, expositorID)
	if err != nil {
		return nil, err
	}
	if s.espacoPago(e.ID) {
		if err := s.cancelamento.ReembolsarExpositor(e.ID, organizadorID, "expositor removido pelo organizador"); err != nil {
			return nil, err
		}
	}
	LiberarExpositor(s.db, e.ID, StatusExpositorRejeitado, strings.TrimSpace(motivo))
	e, _ = s.expositorDoEvento(eventoID, e.ID)
	s.avisarStatus(e)
	return e, nil
}

// ---------- expositor ----------

type DadosExpositor struct {
	TipoEspacoID    int64
	Marca           string
	Categoria       string
	Descricao       string
	PortfolioURL    string
	Instagram       string
	ResponsavelNome string
	Telefone        string
}

func normalizarExpositor(d DadosExpositor) (DadosExpositor, error) {
	d.Marca = strings.TrimSpace(d.Marca)
	d.ResponsavelNome = strings.TrimSpace(d.ResponsavelNome)
	d.Categoria = strings.TrimSpace(d.Categoria)
	d.Descricao = strings.TrimSpace(d.Descricao)
	d.Telefone = strings.TrimSpace(d.Telefone)
	d.Instagram = limparHandle(d.Instagram)
	d.PortfolioURL = strings.TrimSpace(d.PortfolioURL)
	if d.PortfolioURL != "" {
		u, err := url.Parse(d.PortfolioURL)
		if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" || len(d.PortfolioURL) > 300 {
			return d, fmt.Errorf("%w: link do portfólio precisa começar com https://", ErrDadosExpositor)
		}
	}
	if n := len([]rune(d.Marca)); n < 2 || n > 80 {
		return d, fmt.Errorf("%w: informe o nome da marca ou loja", ErrDadosExpositor)
	}
	if n := len([]rune(d.ResponsavelNome)); n < 3 || n > 120 {
		return d, fmt.Errorf("%w: informe o nome do responsável", ErrDadosExpositor)
	}
	if len([]rune(d.Categoria)) > 40 || len([]rune(d.Descricao)) > 1000 || len(d.Telefone) > 30 || len(d.Instagram) > 60 {
		return d, ErrDadosExpositor
	}
	return d, nil
}

// expositoresAbertos: evento publicado, ainda não começou e com espaço ativo.
func (s *ExpositorService) expositoresAbertos(e *domain.Evento) bool {
	if e.Status != domain.StatusEventoPublicado || (e.InicioEm != nil && !time.Now().Before(*e.InicioEm)) {
		return false
	}
	var n int64
	s.db.Model(&TipoEspaco{}).Where("evento_id = ? AND ativo", e.ID).Count(&n)
	return n > 0
}

// Candidatar: quem foi recusado ou desistiu pode se candidatar de novo.
func (s *ExpositorService) Candidatar(usuarioID int64, evento *domain.Evento, d DadosExpositor) (*Expositor, error) {
	if !s.expositoresAbertos(evento) {
		return nil, ErrExpositoresFechados
	}
	d, err := normalizarExpositor(d)
	if err != nil {
		return nil, err
	}
	var t TipoEspaco
	if err := s.db.Where("id = ? AND evento_id = ? AND ativo", d.TipoEspacoID, evento.ID).First(&t).Error; err != nil {
		return nil, fmt.Errorf("%w: escolha o tipo de espaço", ErrDadosExpositor)
	}
	var atual Expositor
	err = s.db.Where("evento_id = ? AND usuario_id = ?", evento.ID, usuarioID).First(&atual).Error
	campos := Expositor{EventoID: evento.ID, UsuarioID: usuarioID, TipoEspacoID: t.ID, Marca: d.Marca, Categoria: d.Categoria,
		Descricao: d.Descricao, PortfolioURL: d.PortfolioURL, Instagram: d.Instagram, ResponsavelNome: d.ResponsavelNome,
		Telefone: d.Telefone, Status: StatusExpositorPendente, CriadoEm: time.Now()}
	if err == nil {
		if atual.Status != StatusExpositorRejeitado && atual.Status != StatusExpositorDesistiu {
			return nil, ErrJaCandidatado
		}
		campos.ID = atual.ID
		return &campos, s.db.Save(&campos).Error
	}
	return &campos, s.db.Create(&campos).Error
}

// MeuEspaco é o que o expositor vê na página do evento.
type MeuEspaco struct {
	Expositor   Expositor
	Tipo        TipoEspaco
	Mesa        *Espaco
	Valores     *ValoresInscricao // preço, taxa, garantia, prazo, pedido aberto ou já pago
	Credenciais []CredencialComQR
}

type CredencialComQR struct {
	CredencialExpositor
	Payload string
}

func (s *ExpositorService) Meu(usuarioID int64, evento *domain.Evento) (*MeuEspaco, error) {
	var e Expositor
	if err := s.db.Where("evento_id = ? AND usuario_id = ?", evento.ID, usuarioID).First(&e).Error; err != nil {
		return nil, ErrExpositorNaoEncontrado
	}
	out := &MeuEspaco{Expositor: e}
	s.db.First(&out.Tipo, e.TipoEspacoID)
	if e.EspacoID != nil {
		var m Espaco
		if s.db.First(&m, *e.EspacoID).Error == nil {
			out.Mesa = &m
		}
	}
	var pago domain.ItemPedido
	if err := s.db.Where("expositor_id = ? AND status = ?", e.ID, domain.StatusItemPago).First(&pago).Error; err == nil {
		id := pago.ID
		out.Valores = &ValoresInscricao{Pago: true, ItemPagoID: &id, PrecoCentavos: pago.PrecoCentavos, TaxaCentavos: pago.TaxaPlataformaCentavos,
			GarantiaCentavos: pago.GarantiaCentavos, TotalPagoCentavos: pago.TotalCentavos, GarantiaContratada: pago.GarantiaContratada}
	} else if e.Status == StatusExpositorAguardandoPagamento {
		t, err := s.checkout.taxasInscricao(evento)
		if err != nil {
			return nil, err
		}
		sem, com := t.Calcular(out.Tipo.PrecoCentavos, false), t.Calcular(out.Tipo.PrecoCentavos, true)
		out.Valores = &ValoresInscricao{PrecoCentavos: out.Tipo.PrecoCentavos, TaxaCentavos: sem.TaxaCentavos, GarantiaCentavos: com.GarantiaCentavos,
			GarantiaDisponivel: garantiaInscricaoDisponivel(evento, time.Now()), PrazoEm: e.PrazoPagamentoEm}
		if p, err := pedidoAbertoDoExpositor(s.db, e.ID, time.Now()); err == nil {
			out.Valores.PedidoAbertoID = &p.ID
		}
	}
	if e.Status == StatusExpositorAprovado {
		var cs []CredencialExpositor
		s.db.Where("expositor_id = ? AND ativa", e.ID).Order("id").Find(&cs)
		for _, c := range cs {
			out.Credenciais = append(out.Credenciais, CredencialComQR{CredencialExpositor: c, Payload: PayloadCredencial(s.segredo, c.ID)})
		}
	}
	return out, nil
}

func pedidoAbertoDoExpositor(tx *gorm.DB, expositorID int64, agora time.Time) (*domain.Pedido, error) {
	var p domain.Pedido
	err := tx.Where(`status = ? AND expira_em > ? AND id IN
		(SELECT pedido_id FROM itens_pedido WHERE expositor_id = ? AND status = ?)`,
		domain.StatusPedidoAguardandoPagamento, agora, expositorID, domain.StatusItemReservado).Order("id DESC").First(&p).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// RenomearCredencial: o expositor põe o nome de cada ajudante.
func (s *ExpositorService) RenomearCredencial(usuarioID, credencialID int64, nome string) error {
	nome = strings.TrimSpace(nome)
	if len([]rune(nome)) > 120 {
		return ErrDadosExpositor
	}
	res := s.db.Model(&CredencialExpositor{}).
		Where("id = ? AND expositor_id IN (SELECT id FROM expositores WHERE usuario_id = ?)", credencialID, usuarioID).
		Update("nome", nome)
	if res.RowsAffected == 0 {
		return ErrQRCredencial
	}
	return res.Error
}

// tipoEspacoInterno devolve (ou cria) o ingresso interno dos espaços.
func tipoEspacoInterno(tx *gorm.DB, eventoID int64) (*domain.TipoIngresso, error) {
	err := tx.Exec(`INSERT INTO tipos_ingresso (evento_id, nome, descricao, preco_centavos, quantidade, min_por_pedido, max_por_pedido, ordem, ativo, espaco_expositor, criado_em)
		VALUES (?, 'Espaço de expositor', 'Uso interno: espaço de expositor (item 8.6).', 0, 1000000, 1, 1, 9999, false, true, now())
		ON CONFLICT (evento_id) WHERE espaco_expositor DO NOTHING`, eventoID).Error
	if err != nil {
		return nil, err
	}
	var t domain.TipoIngresso
	if err := tx.Where("evento_id = ? AND espaco_expositor", eventoID).First(&t).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

// ReservarEspaco abre o pedido do espaço (mesma lógica da inscrição paga).
func (s *ExpositorService) ReservarEspaco(usuarioID int64, evento *domain.Evento, garantia bool) (*domain.Pedido, []domain.ItemPedido, error) {
	c := s.checkout
	agora := time.Now()
	if garantia && !garantiaInscricaoDisponivel(evento, agora) {
		return nil, nil, ErrGarantiaNaoDisponivel
	}
	t, err := c.taxasInscricao(evento)
	if err != nil {
		return nil, nil, err
	}
	reservaMinutos, err := c.config.BuscarInt64(domain.ChaveReservaMinutos)
	if err != nil {
		return nil, nil, fmt.Errorf("config de reserva ausente: %w", err)
	}
	var email string
	if u, err := s.usuarios.BuscarPorID(usuarioID); err == nil {
		email = u.Email
	}
	var pedido *domain.Pedido
	var itens []domain.ItemPedido
	var trocado int64
	err = s.db.Transaction(func(tx *gorm.DB) error {
		var e Expositor
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("evento_id = ? AND usuario_id = ?", evento.ID, usuarioID).First(&e).Error; err != nil ||
			e.Status != StatusExpositorAguardandoPagamento || e.EspacoID == nil {
			return ErrEspacoSemPagamento
		}
		if e.PrazoPagamentoEm != nil && agora.After(*e.PrazoPagamentoEm) {
			return ErrPrazoInscricaoVencido
		}
		var tipo TipoEspaco
		if err := tx.First(&tipo, e.TipoEspacoID).Error; err != nil || tipo.PrecoCentavos <= 0 {
			return ErrEspacoSemPagamento
		}
		if aberto, err := pedidoAbertoDoExpositor(tx, e.ID, agora); err == nil {
			var item domain.ItemPedido
			if err := tx.Where("pedido_id = ? AND expositor_id = ?", aberto.ID, e.ID).First(&item).Error; err != nil {
				return err
			}
			if item.GarantiaContratada == garantia && item.PrecoCentavos == tipo.PrecoCentavos {
				pedido, itens = aberto, []domain.ItemPedido{item}
				return nil
			}
			if err := tx.Model(&domain.ItemPedido{}).Where("pedido_id = ? AND status = ?", aberto.ID, domain.StatusItemReservado).
				Update("status", domain.StatusItemExpirado).Error; err != nil {
				return err
			}
			if err := tx.Model(&domain.Pedido{}).Where("id = ?", aberto.ID).Update("status", domain.StatusPedidoExpirado).Error; err != nil {
				return err
			}
			trocado = aberto.ID
		}
		interno, err := tipoEspacoInterno(tx, evento.ID)
		if err != nil {
			return err
		}
		expira := agora.Add(time.Duration(reservaMinutos) * time.Minute)
		pedido = &domain.Pedido{UsuarioID: usuarioID, EventoID: evento.ID, Status: domain.StatusPedidoAguardandoPagamento, ExpiraEm: &expira, CriadoEm: agora}
		if err := c.pedidos.Criar(tx, pedido); err != nil {
			return err
		}
		valores := t.Calcular(tipo.PrecoCentavos, garantia)
		codigo, err := gerarCodigoItem()
		if err != nil {
			return err
		}
		expID := e.ID
		item := domain.ItemPedido{
			PedidoID: pedido.ID, TipoIngressoID: interno.ID, ExpositorID: &expID,
			TitularNome: e.Marca, TitularEmail: email,
			PrecoCentavos: tipo.PrecoCentavos, TaxaPlataformaCentavos: valores.TaxaCentavos,
			GarantiaContratada: valores.GarantiaCentavos > 0, GarantiaCentavos: valores.GarantiaCentavos,
			TotalCentavos: valores.TotalCentavos, Status: domain.StatusItemReservado,
			Codigo: codigo, QRToken: c.gerarQRToken(codigo), CriadoEm: agora,
		}
		if err := c.itensPedido.Criar(tx, &item); err != nil {
			return err
		}
		itens = []domain.ItemPedido{item}
		pedido.TotalCentavos = item.TotalCentavos
		return tx.Save(pedido).Error
	})
	if err != nil {
		return nil, nil, err
	}
	if trocado != 0 {
		c.cancelarPixPendente(trocado)
	}
	return pedido, itens, nil
}

// ExpirarEspacosNaoPagos: quem não pagou no prazo perde a mesa (menos quem
// está com pagamento em andamento dentro da reserva).
func (s *ExpositorService) ExpirarEspacosNaoPagos() (int, error) {
	agora := time.Now()
	var ids []int64
	err := s.db.Model(&Expositor{}).
		Where("status = ? AND prazo_pagamento_em < ?", StatusExpositorAguardandoPagamento, agora).
		Where(`NOT EXISTS (SELECT 1 FROM itens_pedido i JOIN pedidos p ON p.id = i.pedido_id
			WHERE i.expositor_id = expositores.id AND p.status = ? AND p.expira_em > ?)`, domain.StatusPedidoAguardandoPagamento, agora).
		Pluck("id", &ids).Error
	if err != nil {
		return 0, err
	}
	total := 0
	for _, id := range ids {
		res := s.db.Model(&Expositor{}).Where("id = ? AND status = ?", id, StatusExpositorAguardandoPagamento).
			Updates(map[string]any{"status": StatusExpositorRejeitado, "motivo_rejeicao": "O prazo para pagar o espaço terminou.", "espaco_id": nil, "prazo_pagamento_em": nil})
		if res.Error != nil {
			return total, res.Error
		}
		if res.RowsAffected == 1 {
			total++
			s.AvisarPrazoEspacoVencido(id)
		}
	}
	return total, nil
}

// ---------- público ----------

type MesaPublica struct {
	Codigo  string
	Setor   string
	Tipo    string
	Ocupada bool
	Marca   string // só de expositor aprovado
}

type ExpositorPublico struct {
	Marca     string
	Categoria string
	Descricao string
	Instagram string
	Mesa      string
}

type ExpositoresPublico struct {
	Aberto      bool
	Tipos       []TipoEspaco // ativos, para quem quer expor
	Mesas       []MesaPublica
	Expositores []ExpositorPublico
}

func (s *ExpositorService) Publico(evento *domain.Evento) (*ExpositoresPublico, error) {
	out := &ExpositoresPublico{Aberto: s.expositoresAbertos(evento)}
	if err := s.db.Where("evento_id = ? AND ativo", evento.ID).Order("preco_centavos, id").Find(&out.Tipos).Error; err != nil {
		return nil, err
	}
	var linhas []struct {
		Codigo, Setor, Tipo string
		Status              *string
		Marca, Categoria    *string
		Descricao           *string
		Instagram           *string
	}
	err := s.db.Table("espacos m").
		Select("m.codigo, m.setor, t.nome AS tipo, x.status, x.marca, x.categoria, x.descricao, x.instagram").
		Joins("JOIN tipos_espaco t ON t.id = m.tipo_espaco_id").
		Joins("LEFT JOIN expositores x ON x.espaco_id = m.id AND x.status IN ?", []string{StatusExpositorAguardandoPagamento, StatusExpositorAprovado}).
		Where("m.evento_id = ?", evento.ID).Order("m.setor, m.ordem, m.codigo").Scan(&linhas).Error
	if err != nil {
		return nil, err
	}
	for _, l := range linhas {
		m := MesaPublica{Codigo: l.Codigo, Setor: l.Setor, Tipo: l.Tipo, Ocupada: l.Status != nil}
		if l.Status != nil && *l.Status == StatusExpositorAprovado && l.Marca != nil {
			m.Marca = *l.Marca
			out.Expositores = append(out.Expositores, ExpositorPublico{Marca: *l.Marca, Categoria: deref(l.Categoria), Descricao: deref(l.Descricao),
				Instagram: deref(l.Instagram), Mesa: l.Codigo})
		}
		out.Mesas = append(out.Mesas, m)
	}
	return out, nil
}

func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// ---------- check-in da credencial ----------

type LeituraCredencial struct {
	Nome        string
	Marca       string
	Mesa        string
	PrimeiraVez bool
	EntrouEm    *time.Time
}

// LerCredencial: organizador ou equipe de check-in lê o QR do expositor.
// Expositor entra e sai várias vezes; a primeira entrada fica registrada.
func (s *ExpositorService) LerCredencial(usuarioID, eventoID int64, payload string) (*LeituraCredencial, error) {
	if s.checkin == nil || !s.checkin.TemAcesso(usuarioID, eventoID) {
		return nil, ErrEventoNaoPertenceAoOrganizador
	}
	id, err := LerPayloadCredencial(s.segredo, payload)
	if err != nil {
		return nil, err
	}
	var out LeituraCredencial
	err = s.db.Transaction(func(tx *gorm.DB) error {
		var c CredencialExpositor
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&c, id).Error; err != nil || !c.Ativa {
			return ErrQRCredencial
		}
		var e Expositor
		if err := tx.First(&e, c.ExpositorID).Error; err != nil || e.EventoID != eventoID || e.Status != StatusExpositorAprovado {
			return ErrQRCredencial
		}
		agora := time.Now()
		out.Nome, out.Marca, out.PrimeiraVez = c.Nome, e.Marca, c.PrimeiraEntradaEm == nil
		out.EntrouEm = c.PrimeiraEntradaEm
		if e.EspacoID != nil {
			var m Espaco
			if tx.First(&m, *e.EspacoID).Error == nil {
				out.Mesa = m.Codigo
			}
		}
		mud := map[string]any{"ultima_entrada_em": agora}
		if c.PrimeiraEntradaEm == nil {
			mud["primeira_entrada_em"] = agora
			out.EntrouEm = &agora
		}
		return tx.Model(&c).Updates(mud).Error
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ---------- e-mails ----------

func (s *ExpositorService) emailDe(usuarioID int64) string {
	if u, err := s.usuarios.BuscarPorID(usuarioID); err == nil {
		return u.Email
	}
	return ""
}

func (s *ExpositorService) linkEvento(e *domain.Evento) string {
	if s.frontendURL == "" {
		return ""
	}
	return fmt.Sprintf(`<p><a href="%s/e/%s#meu-espaco">Abrir meu espaço</a></p>`, s.frontendURL, e.Slug)
}

func (s *ExpositorService) avisarStatus(x *Expositor) {
	if x == nil {
		return
	}
	email := s.emailDe(x.UsuarioID)
	evento, err := s.eventos.BuscarPorID(x.EventoID)
	if email == "" || err != nil {
		return
	}
	mesa := ""
	if x.EspacoID != nil {
		var m Espaco
		if s.db.First(&m, *x.EspacoID).Error == nil {
			mesa = m.Codigo
		}
	}
	var assunto, corpo string
	switch x.Status {
	case StatusExpositorAguardandoPagamento:
		var t TipoEspaco
		s.db.First(&t, x.TipoEspacoID)
		prazo := ""
		if x.PrazoPagamentoEm != nil {
			prazo = " até " + x.PrazoPagamentoEm.In(FusoDoEvento(evento)).Format("02/01/2006 às 15:04")
		}
		assunto = "Espaço aprovado: falta pagar — " + evento.Titulo
		corpo = fmt.Sprintf(`<p>Olá, %s!</p><p><strong>%s</strong> foi aprovada para expor em <strong>%s</strong>, na mesa <strong>%s</strong>. Para garantir o espaço, pague %s (mais a taxa de serviço)%s. Sem o pagamento, a mesa é liberada.</p>`,
			html.EscapeString(x.ResponsavelNome), html.EscapeString(x.Marca), html.EscapeString(evento.Titulo), html.EscapeString(mesa), formatarReais(t.PrecoCentavos), prazo)
	case StatusExpositorAprovado:
		assunto = "Espaço confirmado — " + evento.Titulo
		corpo = fmt.Sprintf(`<p>Olá, %s!</p><p>O espaço de <strong>%s</strong> em <strong>%s</strong> está confirmado: mesa <strong>%s</strong>. As credenciais com QR (sua e dos ajudantes) ficam na página do evento, com você conectado. Coloque o nome de cada ajudante lá antes do dia.</p>`,
			html.EscapeString(x.ResponsavelNome), html.EscapeString(x.Marca), html.EscapeString(evento.Titulo), html.EscapeString(mesa))
	case StatusExpositorRejeitado:
		assunto = "Candidatura de expositor — " + evento.Titulo
		motivo := ""
		if x.MotivoRejeicao != "" {
			motivo = "<p>Motivo: " + html.EscapeString(x.MotivoRejeicao) + "</p>"
		}
		corpo = fmt.Sprintf(`<p>Olá, %s!</p><p>A candidatura de <strong>%s</strong> para expor em <strong>%s</strong> não foi aprovada (ou o espaço foi removido). Se você já tinha pago, o valor inteiro volta pelo mesmo meio de pagamento.</p>%s`,
			html.EscapeString(x.ResponsavelNome), html.EscapeString(x.Marca), html.EscapeString(evento.Titulo), motivo)
	default:
		return
	}
	_ = s.mailCliente.Enviar(email, assunto, corpo+s.linkEvento(evento))
}

// AvisarEspacoPago: pagamento confirmado = espaço confirmado.
func (s *ExpositorService) AvisarEspacoPago(expositorID int64) {
	var x Expositor
	if s.db.First(&x, expositorID).Error == nil {
		s.avisarStatus(&x)
	}
}

// AvisarPrazoEspacoVencido: o expositor aprovado não pagou a tempo.
func (s *ExpositorService) AvisarPrazoEspacoVencido(expositorID int64) {
	var x Expositor
	if s.db.First(&x, expositorID).Error != nil {
		return
	}
	email := s.emailDe(x.UsuarioID)
	evento, err := s.eventos.BuscarPorID(x.EventoID)
	if email == "" || err != nil {
		return
	}
	corpo := fmt.Sprintf(`<p>Olá, %s!</p><p>O prazo para pagar o espaço de <strong>%s</strong> em <strong>%s</strong> terminou, e a mesa foi liberada. Se ainda quiser expor, fale com o organizador.</p>`,
		html.EscapeString(x.ResponsavelNome), html.EscapeString(x.Marca), html.EscapeString(evento.Titulo))
	if err := s.mailCliente.Enviar(email, "Prazo do espaço encerrado — "+evento.Titulo, corpo); err != nil {
		slog.Warn("e-mail de prazo do espaço", "erro", err)
	}
}
