package service

import (
	"errors"
	"strings"
	"time"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
)

var (
	ErrCupomInvalido       = errors.New("cupom inválido, expirado ou esgotado")
	ErrCupomDadosInvalidos = errors.New("cupom inválido: percentual entre 1 e 100 ou valor em centavos maior que zero")
	ErrCupomJaExiste       = errors.New("já existe um cupom com esse código neste evento")
)

type CupomService struct {
	eventos *EventoService
	cupons  *repository.CupomRepository
}

func NovoCupomService(eventos *EventoService, cupons *repository.CupomRepository) *CupomService {
	return &CupomService{eventos: eventos, cupons: cupons}
}

type CupomDados struct {
	Codigo    string
	Tipo      domain.TipoCupom
	Valor     int64
	MaxUsos   *int
	ValidoDe  *time.Time
	ValidoAte *time.Time
}

func (s *CupomService) Listar(organizadorID, eventoID int64) ([]domain.Cupom, error) {
	if _, err := s.eventos.BuscarDoOrganizador(organizadorID, eventoID); err != nil {
		return nil, err
	}
	return s.cupons.ListarPorEvento(eventoID)
}

func (s *CupomService) Criar(organizadorID, eventoID int64, d CupomDados) (*domain.Cupom, error) {
	if _, err := s.eventos.BuscarDoOrganizador(organizadorID, eventoID); err != nil {
		return nil, err
	}
	codigo := strings.ToUpper(strings.TrimSpace(d.Codigo))
	if codigo == "" || d.Valor <= 0 || (d.Tipo != domain.CupomPercentual && d.Tipo != domain.CupomValor) ||
		(d.Tipo == domain.CupomPercentual && d.Valor > 100) {
		return nil, ErrCupomDadosInvalidos
	}
	if _, err := s.cupons.BuscarPorCodigo(nil, eventoID, codigo, false); err == nil {
		return nil, ErrCupomJaExiste
	}
	c := &domain.Cupom{
		EventoID: eventoID, Codigo: codigo, Tipo: d.Tipo, Valor: d.Valor, MaxUsos: d.MaxUsos,
		ValidoDe: d.ValidoDe, ValidoAte: d.ValidoAte, Ativo: true, CriadoEm: time.Now(),
	}
	if err := s.cupons.Criar(c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *CupomService) Desativar(organizadorID, eventoID, cupomID int64) error {
	if _, err := s.eventos.BuscarDoOrganizador(organizadorID, eventoID); err != nil {
		return err
	}
	c, err := s.cupons.BuscarPorID(cupomID)
	if err != nil || c.EventoID != eventoID {
		return ErrCupomInvalido
	}
	c.Ativo = false
	return s.cupons.Salvar(c)
}

// cupomUtilizavel: ativo, dentro da validade e com usos disponíveis para n itens.
func cupomUtilizavel(c *domain.Cupom, itens int, agora time.Time) bool {
	if !c.Ativo {
		return false
	}
	if c.ValidoDe != nil && agora.Before(*c.ValidoDe) {
		return false
	}
	if c.ValidoAte != nil && agora.After(*c.ValidoAte) {
		return false
	}
	return c.MaxUsos == nil || c.Usos+itens <= *c.MaxUsos
}

// Validar é a prévia do checkout (POST /eventos/:slug/cupom); o desconto
// definitivo é sempre recalculado em Reservar.
func (s *CupomService) Validar(eventoID int64, codigo string) (*domain.Cupom, error) {
	c, err := s.cupons.BuscarPorCodigo(nil, eventoID, codigo, false)
	if err != nil || !cupomUtilizavel(c, 1, time.Now()) {
		return nil, ErrCupomInvalido
	}
	return c, nil
}
