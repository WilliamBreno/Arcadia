package service

import (
	"errors"
	"fmt"
	"html"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/WilliamBreno/Arcadia/backend/internal/mail"
)

// Item 9.8: coprodução. O organizador principal (dono do evento) convida
// outro organizador com um percentual do repasse líquido. Toda mudança de
// percentual precisa do aceite do coprodutor; depois da primeira venda
// ninguém sai sem esse aceite e, gerado o repasse, nada muda.

var (
	ErrCoproducao              = errors.New("coprodução inválida")
	ErrCoproducaoNaoEncontrada = errors.New("coprodução não encontrada")
	ErrCoproducaoTravada       = errors.New("o repasse deste evento já foi gerado: a divisão não muda mais")
)

const (
	CoproducaoConvidado = "convidado"
	CoproducaoAtiva     = "ativa"
	CoproducaoRecusada  = "recusada"
	CoproducaoEncerrada = "encerrada"
)

type Coproducao struct {
	ID            int64 `gorm:"primaryKey"`
	EventoID      int64
	OrganizadorID int64
	PercentualBP  int64
	PropostoBP    *int64
	Status        string
	ConvidadoEm   time.Time `gorm:"default:now()"`
	RespondidoEm  *time.Time
}

func (Coproducao) TableName() string { return "coproducoes" }

// PartesAtivas: coprodutores que valem na divisão (já aceitaram).
func PartesAtivas(db *gorm.DB, eventoID int64) ([]ParteRepasse, error) {
	var partes []ParteRepasse
	err := db.Raw(`SELECT organizador_id, percentual_bp AS bp FROM coproducoes
		WHERE evento_id = ? AND status = 'ativa' AND percentual_bp > 0 ORDER BY id`, eventoID).Scan(&partes).Error
	return partes, err
}

// EventoComVenda: já houve venda (ingresso com preço, reservado ou pago,
// ou venda na porta). A partir daí a divisão trava.
func EventoComVenda(db *gorm.DB, eventoID int64) bool {
	var n int64
	db.Raw(`SELECT count(*) FROM itens_pedido i JOIN pedidos p ON p.id = i.pedido_id
		WHERE p.evento_id = ? AND i.status IN ('reservado', 'pago', 'utilizado', 'reembolsado', 'cancelado')
		AND (i.preco_centavos > 0 OR i.taxa_plataforma_centavos > 0) AND i.ficha_id IS NULL AND i.expositor_id IS NULL`, eventoID).Scan(&n)
	return n > 0
}

func eventoComRepasse(tx *gorm.DB, eventoID int64) bool {
	var n int64
	tx.Raw(`SELECT count(*) FROM repasses WHERE evento_id = ? AND status <> 'cancelado'`, eventoID).Scan(&n)
	return n > 0
}

type CoproducaoService struct {
	db          *gorm.DB
	mail        *mail.Cliente
	frontendURL string
}

func NovoCoproducaoService(db *gorm.DB, m *mail.Cliente, frontendURL string) *CoproducaoService {
	return &CoproducaoService{db: db, mail: m, frontendURL: strings.TrimSuffix(frontendURL, "/")}
}

// travarEvento: trava a linha do evento (mesma ordem do job de repasse) e
// confere que o repasse ainda não saiu.
func travarEvento(tx *gorm.DB, eventoID int64) (*domain.Evento, error) {
	var e domain.Evento
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&e, eventoID).Error; err != nil {
		return nil, ErrCoproducaoNaoEncontrada
	}
	if eventoComRepasse(tx, eventoID) {
		return nil, ErrCoproducaoTravada
	}
	return &e, nil
}

// somaOutros: quanto os outros coprodutores ocupam (o maior entre o que
// vale e o que foi proposto), para o principal nunca ficar sem parte.
func somaOutros(tx *gorm.DB, eventoID, exceto int64) int64 {
	var soma int64
	tx.Raw(`SELECT COALESCE(SUM(GREATEST(percentual_bp, COALESCE(proposto_bp, 0))), 0) FROM coproducoes
		WHERE evento_id = ? AND id <> ? AND status IN ('convidado', 'ativa')`, eventoID, exceto).Scan(&soma)
	return soma
}

func validarBP(bp int64) error {
	if bp < 100 || bp > 9900 {
		return fmt.Errorf("%w: o percentual vai de 1%% a 99%%", ErrCoproducao)
	}
	return nil
}

// Convidar: o principal convida outro organizador (pelo e-mail da conta
// dele) com um percentual. Vale só depois do aceite.
func (s *CoproducaoService) Convidar(principalID, eventoID int64, email string, bp int64) (*Coproducao, error) {
	if err := validarBP(bp); err != nil {
		return nil, err
	}
	var convidado struct {
		ID          int64
		NomePublico string
		Email       string
	}
	s.db.Raw(`SELECT o.id, o.nome_publico, u.email FROM organizadores o JOIN usuarios u ON u.id = o.usuario_id
		WHERE lower(u.email) = lower(?) LIMIT 1`, strings.TrimSpace(email)).Scan(&convidado)
	if convidado.ID == 0 {
		return nil, fmt.Errorf("%w: não achamos um organizador com esse e-mail (a pessoa precisa ter perfil de organizador no Evve)", ErrCoproducao)
	}
	if convidado.ID == principalID {
		return nil, fmt.Errorf("%w: você já é o organizador principal", ErrCoproducao)
	}
	var c Coproducao
	var evento *domain.Evento
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var err error
		if evento, err = travarEvento(tx, eventoID); err != nil {
			return err
		}
		if evento.OrganizadorID != principalID {
			return ErrEventoNaoPertenceAoOrganizador
		}
		existente := tx.Where("evento_id = ? AND organizador_id = ?", eventoID, convidado.ID).First(&c).Error == nil
		if existente && (c.Status == CoproducaoAtiva || c.Status == CoproducaoConvidado) {
			return fmt.Errorf("%w: esse organizador já está na coprodução; mude o percentual dele", ErrCoproducao)
		}
		if somaOutros(tx, eventoID, c.ID)+bp >= BPTotal {
			return fmt.Errorf("%w: a soma dos coprodutores precisa deixar uma parte para você", ErrCoproducao)
		}
		c = Coproducao{ID: c.ID, EventoID: eventoID, OrganizadorID: convidado.ID, PercentualBP: 0, PropostoBP: &bp, Status: CoproducaoConvidado, ConvidadoEm: time.Now()}
		return tx.Save(&c).Error
	})
	if err != nil {
		return nil, err
	}
	s.avisar(convidado.Email, "Convite de coprodução: "+evento.Titulo,
		fmt.Sprintf(`<p>Você foi convidado para coproduzir <strong>%s</strong> recebendo <strong>%s</strong> do repasse líquido do evento.</p><p>Responda no seu painel: <a href="%s/organizador/coproducoes">Coproduções</a>.</p>`,
			html.EscapeString(evento.Titulo), FormatarBP(bp), s.frontendURL))
	return &c, nil
}

// Propor: o principal muda o percentual (0 = tirar da coprodução). Antes
// da primeira venda, tirar é na hora; o resto espera o aceite.
func (s *CoproducaoService) Propor(principalID, eventoID, coproducaoID, bp int64) error {
	if bp != 0 {
		if err := validarBP(bp); err != nil {
			return err
		}
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		e, err := travarEvento(tx, eventoID)
		if err != nil {
			return err
		}
		if e.OrganizadorID != principalID {
			return ErrEventoNaoPertenceAoOrganizador
		}
		var c Coproducao
		if tx.Where("id = ? AND evento_id = ? AND status IN ('convidado', 'ativa')", coproducaoID, eventoID).First(&c).Error != nil {
			return ErrCoproducaoNaoEncontrada
		}
		if bp == 0 && (c.Status == CoproducaoConvidado || !EventoComVenda(tx, eventoID)) {
			agora := time.Now()
			return tx.Model(&c).Updates(map[string]any{"status": CoproducaoEncerrada, "percentual_bp": 0, "proposto_bp": nil, "respondido_em": agora}).Error
		}
		if somaOutros(tx, eventoID, c.ID)+bp >= BPTotal {
			return fmt.Errorf("%w: a soma dos coprodutores precisa deixar uma parte para você", ErrCoproducao)
		}
		if bp == c.PercentualBP && c.Status == CoproducaoAtiva {
			return tx.Model(&c).Update("proposto_bp", nil).Error
		}
		return tx.Model(&c).Update("proposto_bp", bp).Error
	})
}

// Responder: o coprodutor aceita ou recusa o convite ou a mudança.
func (s *CoproducaoService) Responder(coprodutorID, coproducaoID int64, aceitar bool) error {
	var c Coproducao
	if s.db.Where("id = ? AND organizador_id = ?", coproducaoID, coprodutorID).First(&c).Error != nil {
		return ErrCoproducaoNaoEncontrada
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		if _, err := travarEvento(tx, c.EventoID); err != nil {
			return err
		}
		if tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&c, c.ID).Error != nil || c.PropostoBP == nil {
			return fmt.Errorf("%w: não há nada esperando a sua resposta", ErrCoproducao)
		}
		agora := time.Now()
		switch {
		case aceitar && *c.PropostoBP == 0:
			return tx.Model(&c).Updates(map[string]any{"status": CoproducaoEncerrada, "percentual_bp": 0, "proposto_bp": nil, "respondido_em": agora}).Error
		case aceitar:
			return tx.Model(&c).Updates(map[string]any{"status": CoproducaoAtiva, "percentual_bp": *c.PropostoBP, "proposto_bp": nil, "respondido_em": agora}).Error
		case c.Status == CoproducaoConvidado:
			return tx.Model(&c).Updates(map[string]any{"status": CoproducaoRecusada, "proposto_bp": nil, "respondido_em": agora}).Error
		default: // recusou a mudança: segue o percentual que já valia
			return tx.Model(&c).Updates(map[string]any{"proposto_bp": nil, "respondido_em": agora}).Error
		}
	})
}

// Sair: o coprodutor deixa a coprodução antes da primeira venda.
func (s *CoproducaoService) Sair(coprodutorID, coproducaoID int64) error {
	var c Coproducao
	if s.db.Where("id = ? AND organizador_id = ? AND status IN ('convidado', 'ativa')", coproducaoID, coprodutorID).First(&c).Error != nil {
		return ErrCoproducaoNaoEncontrada
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		if _, err := travarEvento(tx, c.EventoID); err != nil {
			return err
		}
		if c.Status == CoproducaoAtiva && EventoComVenda(tx, c.EventoID) {
			return fmt.Errorf("%w: o evento já tem vendas; para sair, peça ao organizador principal para propor 0%%", ErrCoproducao)
		}
		return tx.Model(&c).Updates(map[string]any{"status": CoproducaoEncerrada, "percentual_bp": 0, "proposto_bp": nil, "respondido_em": time.Now()}).Error
	})
}

type CoproducaoView struct {
	ID              int64      `json:"id"`
	EventoID        int64      `json:"evento_id"`
	EventoTitulo    string     `json:"evento_titulo"`
	EventoInicioEm  *time.Time `json:"evento_inicio_em"`
	OrganizadorID   int64      `json:"organizador_id"`
	OrganizadorNome string     `json:"organizador_nome"`
	PrincipalNome   string     `json:"principal_nome"`
	PercentualBP    int64      `json:"percentual_bp"`
	PropostoBP      *int64     `json:"proposto_bp"`
	Status          string     `json:"status"`
	PodeSair        bool       `json:"pode_sair" gorm:"-"`
}

const selectCoproducao = `SELECT c.id, c.evento_id, e.titulo AS evento_titulo, e.inicio_em AS evento_inicio_em,
	c.organizador_id, o.nome_publico AS organizador_nome, p.nome_publico AS principal_nome,
	c.percentual_bp, c.proposto_bp, c.status
	FROM coproducoes c JOIN eventos e ON e.id = c.evento_id
	JOIN organizadores o ON o.id = c.organizador_id JOIN organizadores p ON p.id = e.organizador_id `

type ResumoCoproducao struct {
	Coprodutores []CoproducaoView `json:"coprodutores"`
	PrincipalBP  int64            `json:"principal_bp"`
	TemVenda     bool             `json:"tem_venda"`
	Travada      bool             `json:"travada"`
}

// DoEvento: a coprodução vista pelo principal.
func (s *CoproducaoService) DoEvento(principalID, eventoID int64) (*ResumoCoproducao, error) {
	var e domain.Evento
	if s.db.First(&e, eventoID).Error != nil || e.OrganizadorID != principalID {
		return nil, ErrEventoNaoPertenceAoOrganizador
	}
	r := &ResumoCoproducao{Coprodutores: []CoproducaoView{}, TemVenda: EventoComVenda(s.db, eventoID), Travada: eventoComRepasse(s.db, eventoID)}
	if err := s.db.Raw(selectCoproducao+`WHERE c.evento_id = ? AND c.status IN ('convidado', 'ativa', 'recusada') ORDER BY c.id`, eventoID).
		Scan(&r.Coprodutores).Error; err != nil {
		return nil, err
	}
	r.PrincipalBP = BPTotal
	for _, c := range r.Coprodutores {
		if c.Status == CoproducaoAtiva {
			r.PrincipalBP -= c.PercentualBP
		}
	}
	return r, nil
}

// Minhas: coproduções em que o organizador é convidado.
func (s *CoproducaoService) Minhas(coprodutorID int64) ([]CoproducaoView, error) {
	lista := []CoproducaoView{}
	err := s.db.Raw(selectCoproducao+`WHERE c.organizador_id = ? AND c.status IN ('convidado', 'ativa') ORDER BY e.inicio_em DESC NULLS LAST`, coprodutorID).Scan(&lista).Error
	for i := range lista {
		lista[i].PodeSair = lista[i].Status == CoproducaoConvidado || !EventoComVenda(s.db, lista[i].EventoID)
	}
	return lista, err
}

// ParticipaComo: o percentual que vale para este organizador no evento
// (principal: o resto; coprodutor ativo: o dele; senão, false).
func ParticipaComo(db *gorm.DB, e *domain.Evento, organizadorID int64) (int64, bool) {
	partes, err := PartesAtivas(db, e.ID)
	if err != nil {
		return 0, false
	}
	for _, p := range PartesDoEvento(e.OrganizadorID, partes) {
		if p.OrganizadorID == organizadorID {
			return p.BP, true
		}
	}
	return 0, false
}

func (s *CoproducaoService) avisar(email, assunto, corpo string) {
	if s.mail != nil && email != "" {
		_ = s.mail.Enviar(email, assunto, `<div style="font-family:Arial,sans-serif;max-width:560px;margin:auto;color:#111">`+corpo+`</div>`)
	}
}

// FormatarBP: 2500 → "25%", 3333 → "33,33%".
func FormatarBP(bp int64) string {
	if bp%100 == 0 {
		return fmt.Sprintf("%d%%", bp/100)
	}
	return strings.Replace(fmt.Sprintf("%d.%02d%%", bp/100, bp%100), ".", ",", 1)
}
