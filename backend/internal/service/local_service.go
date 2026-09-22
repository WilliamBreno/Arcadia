package service

import (
	"errors"
	"time"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
)

var ErrLocalNaoPertenceAoOrganizador = errors.New("local não pertence a este organizador")

type LocalService struct {
	locais *repository.LocalRepository
}

func NovoLocalService(locais *repository.LocalRepository) *LocalService {
	return &LocalService{locais: locais}
}

type LocalDados struct {
	Nome        string
	Logradouro  string
	Numero      string
	Bairro      string
	Cidade      string
	UF          string
	CEP         string
	Latitude    *float64
	Longitude   *float64
	Capacidade  *int
	Observacoes string
}

func (s *LocalService) Criar(organizadorID int64, dados LocalDados) (*domain.Local, error) {
	local := &domain.Local{
		OrganizadorID: organizadorID,
		Nome:          dados.Nome,
		Logradouro:    dados.Logradouro,
		Numero:        dados.Numero,
		Bairro:        dados.Bairro,
		Cidade:        dados.Cidade,
		UF:            dados.UF,
		CEP:           dados.CEP,
		Latitude:      dados.Latitude,
		Longitude:     dados.Longitude,
		Capacidade:    dados.Capacidade,
		Observacoes:   dados.Observacoes,
		CriadoEm:      time.Now(),
	}
	if err := s.locais.Criar(local); err != nil {
		return nil, err
	}
	return local, nil
}

func (s *LocalService) Atualizar(organizadorID, localID int64, dados LocalDados) (*domain.Local, error) {
	local, err := s.buscarDoOrganizador(organizadorID, localID)
	if err != nil {
		return nil, err
	}

	local.Nome = dados.Nome
	local.Logradouro = dados.Logradouro
	local.Numero = dados.Numero
	local.Bairro = dados.Bairro
	local.Cidade = dados.Cidade
	local.UF = dados.UF
	local.CEP = dados.CEP
	local.Latitude = dados.Latitude
	local.Longitude = dados.Longitude
	local.Capacidade = dados.Capacidade
	local.Observacoes = dados.Observacoes

	if err := s.locais.Salvar(local); err != nil {
		return nil, err
	}
	return local, nil
}

func (s *LocalService) Excluir(organizadorID, localID int64) error {
	local, err := s.buscarDoOrganizador(organizadorID, localID)
	if err != nil {
		return err
	}
	return s.locais.Excluir(local)
}

func (s *LocalService) Listar(organizadorID int64) ([]domain.Local, error) {
	return s.locais.ListarPorOrganizador(organizadorID)
}

func (s *LocalService) buscarDoOrganizador(organizadorID, localID int64) (*domain.Local, error) {
	local, err := s.locais.BuscarPorID(localID)
	if err != nil {
		return nil, err
	}
	if local.OrganizadorID != organizadorID {
		return nil, ErrLocalNaoPertenceAoOrganizador
	}
	return local, nil
}
