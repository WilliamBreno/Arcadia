package service

import (
	"strings"
	"time"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
)

type TipoIngressoService struct {
	eventos *EventoService
	tipos   *repository.TipoIngressoRepository
}

func NovoTipoIngressoService(eventos *EventoService, tipos *repository.TipoIngressoRepository) *TipoIngressoService {
	return &TipoIngressoService{eventos: eventos, tipos: tipos}
}

type TipoIngressoDados struct {
	Nome          string
	Descricao     string
	PrecoCentavos int64
	Quantidade    int
	VendasInicio  *time.Time
	VendasFim     *time.Time
	MinPorPedido  int
	MaxPorPedido  int
	Ordem         int
	LoteGrupo     string
	MeiaEntrada   bool
	Ativo         bool
}

func (s *TipoIngressoService) Listar(organizadorID, eventoID int64) ([]domain.TipoIngresso, error) {
	if _, err := s.eventos.BuscarDoOrganizador(organizadorID, eventoID); err != nil {
		return nil, err
	}
	return s.tipos.ListarPorEvento(eventoID)
}

func (s *TipoIngressoService) Criar(organizadorID, eventoID int64, dados TipoIngressoDados) (*domain.TipoIngresso, error) {
	if _, err := s.eventos.BuscarDoOrganizador(organizadorID, eventoID); err != nil {
		return nil, err
	}

	tipo := &domain.TipoIngresso{
		EventoID:      eventoID,
		Nome:          dados.Nome,
		Descricao:     dados.Descricao,
		PrecoCentavos: dados.PrecoCentavos,
		Quantidade:    dados.Quantidade,
		VendasInicio:  dados.VendasInicio,
		VendasFim:     dados.VendasFim,
		MinPorPedido:  valorIntOuPadrao(dados.MinPorPedido, 1),
		MaxPorPedido:  valorIntOuPadrao(dados.MaxPorPedido, 10),
		Ordem:         dados.Ordem,
		LoteGrupo:     strings.TrimSpace(dados.LoteGrupo),
		MeiaEntrada:   dados.MeiaEntrada,
		Ativo:         dados.Ativo,
		CriadoEm:      time.Now(),
	}
	if err := s.checarCotaMeia(eventoID, tipo); err != nil {
		return nil, err
	}
	if err := s.tipos.Criar(tipo); err != nil {
		return nil, err
	}
	return tipo, nil
}

func (s *TipoIngressoService) Atualizar(organizadorID, eventoID, tipoID int64, dados TipoIngressoDados) (*domain.TipoIngresso, error) {
	if _, err := s.eventos.BuscarDoOrganizador(organizadorID, eventoID); err != nil {
		return nil, err
	}

	tipo, err := s.buscarDoEvento(eventoID, tipoID)
	if err != nil {
		return nil, err
	}

	tipo.Nome = dados.Nome
	tipo.Descricao = dados.Descricao
	tipo.PrecoCentavos = dados.PrecoCentavos
	tipo.Quantidade = dados.Quantidade
	tipo.VendasInicio = dados.VendasInicio
	tipo.VendasFim = dados.VendasFim
	tipo.MinPorPedido = valorIntOuPadrao(dados.MinPorPedido, 1)
	tipo.MaxPorPedido = valorIntOuPadrao(dados.MaxPorPedido, 10)
	tipo.Ordem = dados.Ordem
	tipo.LoteGrupo = strings.TrimSpace(dados.LoteGrupo)
	tipo.MeiaEntrada = dados.MeiaEntrada
	tipo.Ativo = dados.Ativo

	if err := s.checarCotaMeia(eventoID, tipo); err != nil {
		return nil, err
	}
	if err := s.tipos.Salvar(tipo); err != nil {
		return nil, err
	}
	return tipo, nil
}

func (s *TipoIngressoService) Excluir(organizadorID, eventoID, tipoID int64) error {
	if _, err := s.eventos.BuscarDoOrganizador(organizadorID, eventoID); err != nil {
		return err
	}
	tipo, err := s.buscarDoEvento(eventoID, tipoID)
	if err != nil {
		return err
	}
	return s.tipos.Excluir(tipo)
}

func (s *TipoIngressoService) buscarDoEvento(eventoID, tipoID int64) (*domain.TipoIngresso, error) {
	tipo, err := s.tipos.BuscarPorID(tipoID)
	if err != nil {
		return nil, err
	}
	if tipo.EventoID != eventoID {
		return nil, ErrEventoNaoPertenceAoOrganizador
	}
	return tipo, nil
}
