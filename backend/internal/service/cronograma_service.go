package service

import (
	"errors"
	"strings"
	"time"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
)

var (
	ErrCronogramaInvalido      = errors.New("informe título e horário de início; o fim não pode ser antes do início")
	ErrCronogramaNaoEncontrado = errors.New("item do cronograma não encontrado")
)

type CronogramaDados struct {
	Titulo    string
	Descricao string
	Local     string
	InicioEm  time.Time
	FimEm     *time.Time
}

type CronogramaService struct {
	eventos    *EventoService
	cronograma *repository.CronogramaRepository
}

func NovoCronogramaService(e *EventoService, c *repository.CronogramaRepository) *CronogramaService {
	return &CronogramaService{eventos: e, cronograma: c}
}

func validarCronograma(d CronogramaDados) error {
	if strings.TrimSpace(d.Titulo) == "" || d.InicioEm.IsZero() || (d.FimEm != nil && d.FimEm.Before(d.InicioEm)) {
		return ErrCronogramaInvalido
	}
	return nil
}

func (s *CronogramaService) Listar(organizadorID, eventoID int64) ([]domain.CronogramaItem, error) {
	if _, err := s.eventos.BuscarDoOrganizador(organizadorID, eventoID); err != nil {
		return nil, err
	}
	return s.cronograma.ListarPorEvento(eventoID)
}

func (s *CronogramaService) Criar(organizadorID, eventoID int64, d CronogramaDados) (*domain.CronogramaItem, error) {
	if _, err := s.eventos.BuscarDoOrganizador(organizadorID, eventoID); err != nil {
		return nil, err
	}
	if err := validarCronograma(d); err != nil {
		return nil, err
	}
	i := &domain.CronogramaItem{EventoID: eventoID, Titulo: strings.TrimSpace(d.Titulo), Descricao: d.Descricao,
		Local: d.Local, InicioEm: d.InicioEm, FimEm: d.FimEm, CriadoEm: time.Now()}
	return i, s.cronograma.Criar(i)
}

func (s *CronogramaService) itemDoEvento(organizadorID, eventoID, itemID int64) (*domain.CronogramaItem, error) {
	if _, err := s.eventos.BuscarDoOrganizador(organizadorID, eventoID); err != nil {
		return nil, err
	}
	i, err := s.cronograma.BuscarPorID(itemID)
	if err != nil || i.EventoID != eventoID {
		return nil, ErrCronogramaNaoEncontrado
	}
	return i, nil
}

func (s *CronogramaService) Atualizar(organizadorID, eventoID, itemID int64, d CronogramaDados) (*domain.CronogramaItem, error) {
	i, err := s.itemDoEvento(organizadorID, eventoID, itemID)
	if err != nil {
		return nil, err
	}
	if err := validarCronograma(d); err != nil {
		return nil, err
	}
	i.Titulo, i.Descricao, i.Local, i.InicioEm, i.FimEm = strings.TrimSpace(d.Titulo), d.Descricao, d.Local, d.InicioEm, d.FimEm
	return i, s.cronograma.Salvar(i)
}

func (s *CronogramaService) Excluir(organizadorID, eventoID, itemID int64) error {
	i, err := s.itemDoEvento(organizadorID, eventoID, itemID)
	if err != nil {
		return err
	}
	return s.cronograma.Excluir(i)
}
