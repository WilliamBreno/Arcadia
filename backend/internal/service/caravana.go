package service

// Item 8.7: caravanas. Decisões do dono (30/10/2026): cada membro compra o
// próprio ingresso (nada muda no dinheiro); a caravana é por link fechado
// (não aparece na página do evento); o responsável vê nome e telefone dos
// membros, que aceitam isso ao entrar; o organizador vê só as caravanas e
// quantas pessoas vêm de cada cidade.

import (
	"crypto/rand"
	"errors"
	"math/big"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
)

var (
	ErrCaravanaNaoEncontrada = errors.New("caravana não encontrada")
	ErrDadosCaravana         = errors.New("dados da caravana inválidos")
	ErrCaravanaLotada        = errors.New("a caravana já está com todas as vagas preenchidas")
	ErrSoResponsavel         = errors.New("só quem criou a caravana pode fazer isso")
	ErrAceiteCaravana        = errors.New("para entrar, aceite que o responsável veja seu nome e telefone")
)

type Caravana struct {
	ID            int64 `gorm:"primaryKey"`
	EventoID      int64
	ResponsavelID int64
	Codigo        string
	Nome          string
	Cidade        string
	UF            string `gorm:"column:uf"`
	PontoEmbarque string
	SaidaEm       *time.Time
	Vagas         *int
	Telefone      string
	Ativa         bool
	CriadoEm      time.Time
}

func (Caravana) TableName() string { return "caravanas" }

type MembroCaravana struct {
	ID         int64 `gorm:"primaryKey"`
	CaravanaID int64
	UsuarioID  int64
	Nome       string
	Telefone   string
	EmbarcouEm *time.Time
	CriadoEm   time.Time
}

func (MembroCaravana) TableName() string { return "caravana_membros" }

type CaravanaService struct {
	db      *gorm.DB
	eventos *repository.EventoRepository
}

func NovoCaravanaService(db *gorm.DB, e *repository.EventoRepository) *CaravanaService {
	return &CaravanaService{db: db, eventos: e}
}

const alfabetoCodigo = "abcdefghjkmnpqrstuvwxyz23456789"

func codigoCaravana() (string, error) {
	b := make([]byte, 10)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(alfabetoCodigo))))
		if err != nil {
			return "", err
		}
		b[i] = alfabetoCodigo[n.Int64()]
	}
	return string(b), nil
}

type DadosCaravana struct {
	Nome          string
	Cidade        string
	UF            string
	PontoEmbarque string
	SaidaEm       *time.Time
	Vagas         *int
	Telefone      string
}

func normalizarCaravana(d DadosCaravana) (DadosCaravana, error) {
	d.Nome, d.Cidade, d.PontoEmbarque, d.Telefone = strings.TrimSpace(d.Nome), strings.TrimSpace(d.Cidade), strings.TrimSpace(d.PontoEmbarque), strings.TrimSpace(d.Telefone)
	d.UF = strings.ToUpper(strings.TrimSpace(d.UF))
	if n := len([]rune(d.Nome)); n < 3 || n > 80 {
		return d, ErrDadosCaravana
	}
	if n := len([]rune(d.Cidade)); n < 2 || n > 80 {
		return d, ErrDadosCaravana
	}
	if n := len([]rune(d.PontoEmbarque)); n < 3 || n > 200 {
		return d, ErrDadosCaravana
	}
	if (d.UF != "" && !reUF.MatchString(d.UF)) || len(d.Telefone) > 30 || (d.Vagas != nil && (*d.Vagas < 1 || *d.Vagas > 500)) {
		return d, ErrDadosCaravana
	}
	return d, nil
}

// Criar: qualquer pessoa logada cria a caravana de um evento publicado.
func (s *CaravanaService) Criar(usuarioID int64, e *domain.Evento, d DadosCaravana) (*Caravana, error) {
	if e.Status != domain.StatusEventoPublicado || (e.InicioEm != nil && !time.Now().Before(*e.InicioEm)) {
		return nil, ErrDadosCaravana
	}
	d, err := normalizarCaravana(d)
	if err != nil {
		return nil, err
	}
	codigo, err := codigoCaravana()
	if err != nil {
		return nil, err
	}
	c := &Caravana{EventoID: e.ID, ResponsavelID: usuarioID, Codigo: codigo, Nome: d.Nome, Cidade: d.Cidade, UF: d.UF,
		PontoEmbarque: d.PontoEmbarque, SaidaEm: d.SaidaEm, Vagas: d.Vagas, Telefone: d.Telefone, Ativa: true, CriadoEm: time.Now()}
	return c, s.db.Create(c).Error
}

func (s *CaravanaService) porCodigo(codigo string) (*Caravana, error) {
	var c Caravana
	if err := s.db.Where("codigo = ? AND ativa", strings.ToLower(strings.TrimSpace(codigo))).First(&c).Error; err != nil {
		return nil, ErrCaravanaNaoEncontrada
	}
	return &c, nil
}

// Atualizar: o responsável muda ponto, horário, vagas...
func (s *CaravanaService) Atualizar(usuarioID int64, codigo string, d DadosCaravana) (*Caravana, error) {
	c, err := s.porCodigo(codigo)
	if err != nil {
		return nil, err
	}
	if c.ResponsavelID != usuarioID {
		return nil, ErrSoResponsavel
	}
	if d, err = normalizarCaravana(d); err != nil {
		return nil, err
	}
	c.Nome, c.Cidade, c.UF, c.PontoEmbarque, c.SaidaEm, c.Vagas, c.Telefone = d.Nome, d.Cidade, d.UF, d.PontoEmbarque, d.SaidaEm, d.Vagas, d.Telefone
	return c, s.db.Save(c).Error
}

// Encerrar desliga o link (os membros deixam de ver a caravana).
func (s *CaravanaService) Encerrar(usuarioID int64, codigo string) error {
	c, err := s.porCodigo(codigo)
	if err != nil {
		return err
	}
	if c.ResponsavelID != usuarioID {
		return ErrSoResponsavel
	}
	return s.db.Model(c).Update("ativa", false).Error
}

// VisaoCaravana é o que quem tem o link vê.
type VisaoCaravana struct {
	Caravana        Caravana
	EventoTitulo    string
	EventoSlug      string
	Membros         int
	Responsavel     bool
	Membro          bool
	ResponsavelNome string
}

func (s *CaravanaService) contarMembros(caravanaID int64) int {
	var n int64
	s.db.Model(&MembroCaravana{}).Where("caravana_id = ?", caravanaID).Count(&n)
	return int(n)
}

func (s *CaravanaService) Ver(usuarioID int64, codigo string) (*VisaoCaravana, error) {
	c, err := s.porCodigo(codigo)
	if err != nil {
		return nil, err
	}
	e, err := s.eventos.BuscarPorID(c.EventoID)
	if err != nil {
		return nil, ErrCaravanaNaoEncontrada
	}
	v := &VisaoCaravana{Caravana: *c, EventoTitulo: e.Titulo, EventoSlug: e.Slug, Membros: s.contarMembros(c.ID), Responsavel: c.ResponsavelID == usuarioID}
	var n int64
	s.db.Model(&MembroCaravana{}).Where("caravana_id = ? AND usuario_id = ?", c.ID, usuarioID).Count(&n)
	v.Membro = n > 0
	var nome string
	s.db.Table("usuarios").Where("id = ?", c.ResponsavelID).Pluck("nome", &nome)
	v.ResponsavelNome = primeiroNome(nome)
	return v, nil
}

// Entrar: o membro aceita mostrar nome e telefone ao responsável.
func (s *CaravanaService) Entrar(usuarioID int64, codigo, nome, telefone string, aceite bool) error {
	if !aceite {
		return ErrAceiteCaravana
	}
	nome, telefone = strings.TrimSpace(nome), strings.TrimSpace(telefone)
	if n := len([]rune(nome)); n < 2 || n > 120 || len(telefone) > 30 {
		return ErrDadosCaravana
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		var c Caravana
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("codigo = ? AND ativa", strings.ToLower(strings.TrimSpace(codigo))).First(&c).Error; err != nil {
			return ErrCaravanaNaoEncontrada
		}
		var atual MembroCaravana
		if tx.Where("caravana_id = ? AND usuario_id = ?", c.ID, usuarioID).First(&atual).Error == nil {
			return tx.Model(&atual).Updates(map[string]any{"nome": nome, "telefone": telefone}).Error
		}
		if c.Vagas != nil {
			var n int64
			tx.Model(&MembroCaravana{}).Where("caravana_id = ?", c.ID).Count(&n)
			if int(n) >= *c.Vagas {
				return ErrCaravanaLotada
			}
		}
		return tx.Create(&MembroCaravana{CaravanaID: c.ID, UsuarioID: usuarioID, Nome: nome, Telefone: telefone, CriadoEm: time.Now()}).Error
	})
}

// Sair: o membro sai (os dados dele deixam de aparecer para o responsável).
func (s *CaravanaService) Sair(usuarioID int64, codigo string) error {
	c, err := s.porCodigo(codigo)
	if err != nil {
		return err
	}
	return s.db.Where("caravana_id = ? AND usuario_id = ?", c.ID, usuarioID).Delete(&MembroCaravana{}).Error
}

// LinhaEmbarque é a lista do responsável.
type LinhaEmbarque struct {
	ID          int64
	Nome        string
	Telefone    string
	TemIngresso bool
	EmbarcouEm  *time.Time
}

func (s *CaravanaService) Lista(usuarioID int64, codigo string) ([]LinhaEmbarque, error) {
	c, err := s.porCodigo(codigo)
	if err != nil {
		return nil, err
	}
	if c.ResponsavelID != usuarioID {
		return nil, ErrSoResponsavel
	}
	var out []LinhaEmbarque
	err = s.db.Raw(`SELECT m.id, m.nome, m.telefone, m.embarcou_em,
		EXISTS (SELECT 1 FROM itens_pedido i JOIN pedidos p ON p.id = i.pedido_id
			WHERE p.usuario_id = m.usuario_id AND p.evento_id = ? AND i.status IN ('pago','utilizado')
			  AND i.ficha_id IS NULL AND i.expositor_id IS NULL AND NOT i.adicional) AS tem_ingresso
		FROM caravana_membros m WHERE m.caravana_id = ? ORDER BY m.nome`, c.EventoID, c.ID).Scan(&out).Error
	return out, err
}

// MarcarEmbarque: o responsável confere quem subiu no ônibus.
func (s *CaravanaService) MarcarEmbarque(usuarioID int64, codigo string, membroID int64, embarcou bool) error {
	c, err := s.porCodigo(codigo)
	if err != nil {
		return err
	}
	if c.ResponsavelID != usuarioID {
		return ErrSoResponsavel
	}
	var valor any
	if embarcou {
		valor = time.Now()
	}
	res := s.db.Model(&MembroCaravana{}).Where("id = ? AND caravana_id = ?", membroID, c.ID).Update("embarcou_em", valor)
	if res.RowsAffected == 0 {
		return ErrCaravanaNaoEncontrada
	}
	return res.Error
}

// MinhaCaravana aparece na página do evento para responsável e membros.
type MinhaCaravana struct {
	Codigo      string
	Nome        string
	Cidade      string
	Ponto       string
	SaidaEm     *time.Time
	Responsavel bool
}

func (s *CaravanaService) Minhas(usuarioID, eventoID int64) ([]MinhaCaravana, error) {
	var out []MinhaCaravana
	err := s.db.Raw(`SELECT c.codigo, c.nome, c.cidade, c.ponto_embarque AS ponto, c.saida_em, (c.responsavel_id = ?) AS responsavel
		FROM caravanas c WHERE c.evento_id = ? AND c.ativa AND (c.responsavel_id = ? OR EXISTS
			(SELECT 1 FROM caravana_membros m WHERE m.caravana_id = c.id AND m.usuario_id = ?))
		ORDER BY c.id`, usuarioID, eventoID, usuarioID, usuarioID).Scan(&out).Error
	return out, err
}

// ResumoOrganizador: caravanas do evento e quantos vêm de cada uma (sem
// dados pessoais dos membros).
type ResumoCaravana struct {
	Nome        string
	Cidade      string
	UF          string `gorm:"column:uf"`
	SaidaEm     *time.Time
	Membros     int
	ComIngresso int
	Responsavel string
}

func (s *CaravanaService) DoOrganizador(organizadorID, eventoID int64) ([]ResumoCaravana, error) {
	e, err := s.eventos.BuscarPorID(eventoID)
	if err != nil || e.OrganizadorID != organizadorID {
		return nil, ErrEventoNaoPertenceAoOrganizador
	}
	var out []ResumoCaravana
	err = s.db.Raw(`SELECT c.nome, c.cidade, c.uf, c.saida_em, u.nome AS responsavel,
		(SELECT count(*) FROM caravana_membros m WHERE m.caravana_id = c.id) AS membros,
		(SELECT count(*) FROM caravana_membros m WHERE m.caravana_id = c.id AND EXISTS (SELECT 1 FROM itens_pedido i JOIN pedidos p ON p.id = i.pedido_id
			WHERE p.usuario_id = m.usuario_id AND p.evento_id = c.evento_id AND i.status IN ('pago','utilizado')
			  AND i.ficha_id IS NULL AND i.expositor_id IS NULL AND NOT i.adicional)) AS com_ingresso
		FROM caravanas c JOIN usuarios u ON u.id = c.responsavel_id
		WHERE c.evento_id = ? AND c.ativa ORDER BY c.cidade, c.nome`, eventoID).Scan(&out).Error
	return out, err
}
