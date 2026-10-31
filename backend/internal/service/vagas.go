package service

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
)

// Item 11.6: limite opcional de jurados e participantes por evento. A vaga
// é ocupada por ordem de chegada; a contagem roda com uma trava por evento
// dentro da transação, então cadastros simultâneos não passam do limite.

var (
	ErrVagasJurados       = errors.New("as vagas de jurado deste evento já foram preenchidas")
	ErrVagasParticipantes = errors.New("as vagas de participante deste evento já foram preenchidas")
	ErrVagasTipo          = errors.New("as vagas desta modalidade já foram preenchidas")
	ErrVagasInvalidas     = errors.New("cada limite deve ficar entre 1 e 10.000, ou em branco para não limitar")
)

const maxVagas = 10000

// VagasEvento é a configuração do evento (nil = sem limite).
type VagasEvento struct {
	EventoID      int64 `gorm:"primaryKey"`
	Jurados       *int
	Participantes *int
	Cosplay       *int
	Danca         *int
	Canto         *int
	Atuacao       *int
	ListaEspera   bool
	AtualizadoEm  time.Time
}

func (VagasEvento) TableName() string { return "vagas_evento" }

var tiposApresentacao = []domain.TipoApresentacao{domain.TipoApresentacaoCosplay, domain.TipoApresentacaoDanca, domain.TipoApresentacaoCanto, domain.TipoApresentacaoAtuacao}

func (v *VagasEvento) limiteTipo(t domain.TipoApresentacao) *int {
	switch t {
	case domain.TipoApresentacaoCosplay:
		return v.Cosplay
	case domain.TipoApresentacaoDanca:
		return v.Danca
	case domain.TipoApresentacaoCanto:
		return v.Canto
	case domain.TipoApresentacaoAtuacao:
		return v.Atuacao
	}
	return nil
}

// OcupacaoVagas conta quem já ocupa vaga.
type OcupacaoVagas struct {
	Jurados       int
	Participantes int
	PorTipo       map[domain.TipoApresentacao]int
}

// Fichas que seguram vaga: em análise, aprovadas ou aprovadas esperando o
// pagamento da inscrição. Lista de espera, rejeitada e desistência não.
var statusQueOcupaVaga = []domain.StatusFicha{domain.StatusFichaPendente, domain.StatusFichaAprovado, domain.StatusFichaAguardandoPagamento}

func OcupaVaga(s domain.StatusFicha) bool {
	for _, x := range statusQueOcupaVaga {
		if x == s {
			return true
		}
	}
	return false
}

type VagasService struct {
	db      *gorm.DB
	eventos *EventoService
}

func NovoVagasService(db *gorm.DB, eventos *EventoService) *VagasService {
	return &VagasService{db: db, eventos: eventos}
}

func (s *VagasService) DB() *gorm.DB { return s.db }

func (s *VagasService) obter(db *gorm.DB, eventoID int64) VagasEvento {
	var lista []VagasEvento
	if db.Where("evento_id = ?", eventoID).Limit(1).Find(&lista).Error == nil && len(lista) == 1 {
		return lista[0]
	}
	return VagasEvento{EventoID: eventoID}
}

func (s *VagasService) Obter(eventoID int64) VagasEvento { return s.obter(s.db, eventoID) }

func (s *VagasService) ocupacao(db *gorm.DB, eventoID int64) (OcupacaoVagas, error) {
	o := OcupacaoVagas{PorTipo: map[domain.TipoApresentacao]int{}}
	var jurados int64
	if err := db.Model(&domain.PapelEvento{}).
		Where("evento_id = ? AND papel = ? AND status <> ?", eventoID, domain.PapelJurado, domain.StatusPapelRemovido).
		Count(&jurados).Error; err != nil {
		return o, err
	}
	o.Jurados = int(jurados)
	var linhas []struct {
		Tipo  *string
		Total int
	}
	if err := db.Model(&domain.FichaParticipacao{}).Select("tipo_apresentacao AS tipo, count(*) AS total").
		Where("evento_id = ? AND papel = ? AND status IN ?", eventoID, domain.PapelParticipante, statusQueOcupaVaga).
		Group("tipo_apresentacao").Scan(&linhas).Error; err != nil {
		return o, err
	}
	for _, l := range linhas {
		o.Participantes += l.Total
		if l.Tipo != nil {
			o.PorTipo[domain.TipoApresentacao(*l.Tipo)] += l.Total
		}
	}
	return o, nil
}

func (s *VagasService) Ocupacao(eventoID int64) (OcupacaoVagas, error) {
	return s.ocupacao(s.db, eventoID)
}

// TravarVagas segura a contagem de vagas do evento até o fim da transação.
func TravarVagas(tx *gorm.DB, eventoID int64) error {
	return tx.Exec("SELECT pg_advisory_xact_lock(hashtext('vagas-evento'), ?::int)", int32(eventoID)).Error
}

// Conferir diz se cabe mais um (jurado, ou participante da modalidade
// informada). Chame dentro da transação, depois de TravarVagas. Sem limite
// configurado, sempre cabe.
func (s *VagasService) Conferir(tx *gorm.DB, eventoID int64, papel domain.Papel, tipo *domain.TipoApresentacao) error {
	v := s.obter(tx, eventoID)
	o, err := s.ocupacao(tx, eventoID)
	if err != nil {
		return err
	}
	return conferirVagas(v, o, papel, tipo, false)
}

// ConferirTroca é para quem já ocupa vaga e só troca de modalidade: confere
// só o limite da nova modalidade (a pessoa já está no total).
func (s *VagasService) ConferirTroca(tx *gorm.DB, eventoID int64, tipo *domain.TipoApresentacao) error {
	v := s.obter(tx, eventoID)
	o, err := s.ocupacao(tx, eventoID)
	if err != nil {
		return err
	}
	return conferirVagas(v, o, domain.PapelParticipante, tipo, true)
}

// conferirVagas é a regra pura (testada).
func conferirVagas(v VagasEvento, o OcupacaoVagas, papel domain.Papel, tipo *domain.TipoApresentacao, soModalidade bool) error {
	if papel == domain.PapelJurado {
		if v.Jurados != nil && o.Jurados >= *v.Jurados {
			return ErrVagasJurados
		}
		return nil
	}
	if !soModalidade && v.Participantes != nil && o.Participantes >= *v.Participantes {
		return ErrVagasParticipantes
	}
	if tipo != nil {
		if lim := v.limiteTipo(*tipo); lim != nil && o.PorTipo[*tipo] >= *lim {
			return fmt.Errorf("%w (%s)", ErrVagasTipo, NomeTipoApresentacao(*tipo))
		}
	}
	return nil
}

// ConferirAgora faz a mesma conta numa transação própria (aprovação manual
// pelo organizador e troca de modalidade).
func (s *VagasService) ConferirAgora(eventoID int64, papel domain.Papel, tipo *domain.TipoApresentacao, soModalidade bool) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := TravarVagas(tx, eventoID); err != nil {
			return err
		}
		if soModalidade {
			return s.ConferirTroca(tx, eventoID, tipo)
		}
		return s.Conferir(tx, eventoID, papel, tipo)
	})
}

// EhErroVagas diz se o erro é de vaga preenchida.
func EhErroVagas(err error) bool {
	return errors.Is(err, ErrVagasJurados) || errors.Is(err, ErrVagasParticipantes) || errors.Is(err, ErrVagasTipo)
}

func NomeTipoApresentacao(t domain.TipoApresentacao) string {
	switch t {
	case domain.TipoApresentacaoCosplay:
		return "cosplay"
	case domain.TipoApresentacaoDanca:
		return "dança"
	case domain.TipoApresentacaoCanto:
		return "canto"
	case domain.TipoApresentacaoAtuacao:
		return "atuação"
	}
	return string(t)
}

// Definir grava a configuração do organizador. Baixar o limite abaixo de
// quem já está dentro não tira ninguém: só fecha novas entradas.
func (s *VagasService) Definir(organizadorID, eventoID int64, v VagasEvento) (VagasEvento, error) {
	if _, err := s.eventos.BuscarDoOrganizador(organizadorID, eventoID); err != nil {
		return v, err
	}
	for _, x := range []*int{v.Jurados, v.Participantes, v.Cosplay, v.Danca, v.Canto, v.Atuacao} {
		if x != nil && (*x < 1 || *x > maxVagas) {
			return v, ErrVagasInvalidas
		}
	}
	v.EventoID = eventoID
	v.AtualizadoEm = time.Now()
	err := s.db.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "evento_id"}}, UpdateAll: true}).Create(&v).Error
	return v, err
}

// SituacaoVagas é o que a tela mostra (painel e páginas públicas).
type SituacaoVagas struct {
	Limite   *int `json:"limite"`
	Ocupadas int  `json:"ocupadas"`
	Esgotado bool `json:"esgotado"`
}

func situacaoVagas(limite *int, ocupadas int) SituacaoVagas {
	return SituacaoVagas{Limite: limite, Ocupadas: ocupadas, Esgotado: limite != nil && ocupadas >= *limite}
}

type ResumoVagas struct {
	Jurados       SituacaoVagas            `json:"jurados"`
	Participantes SituacaoVagas            `json:"participantes"`
	PorTipo       map[string]SituacaoVagas `json:"por_tipo"`
	ListaEspera   bool                     `json:"lista_espera"`
}

func (s *VagasService) ResumoDoOrganizador(organizadorID, eventoID int64) (ResumoVagas, error) {
	if _, err := s.eventos.BuscarDoOrganizador(organizadorID, eventoID); err != nil {
		return ResumoVagas{}, err
	}
	return s.Resumo(eventoID)
}

func (s *VagasService) Resumo(eventoID int64) (ResumoVagas, error) {
	v := s.Obter(eventoID)
	o, err := s.Ocupacao(eventoID)
	if err != nil {
		return ResumoVagas{}, err
	}
	r := ResumoVagas{
		Jurados:       situacaoVagas(v.Jurados, o.Jurados),
		Participantes: situacaoVagas(v.Participantes, o.Participantes),
		PorTipo:       map[string]SituacaoVagas{},
		ListaEspera:   v.ListaEspera,
	}
	for _, t := range tiposApresentacao {
		r.PorTipo[string(t)] = situacaoVagas(v.limiteTipo(t), o.PorTipo[t])
	}
	return r, nil
}
