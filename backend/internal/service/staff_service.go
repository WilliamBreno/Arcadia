package service

import (
	"errors"
	"time"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
)

var (
	ErrUsuarioNaoEncontrado = errors.New("usuário não encontrado — peça pra essa pessoa criar uma conta primeiro")
	ErrStaffNaoEncontrado   = errors.New("staff não encontrado neste evento")
)

type StaffService struct {
	eventos  *repository.EventoRepository
	usuarios *repository.UsuarioRepository
	papeis   *repository.PapelEventoRepository
}

func NovoStaffService(eventos *repository.EventoRepository, usuarios *repository.UsuarioRepository, papeis *repository.PapelEventoRepository) *StaffService {
	return &StaffService{eventos: eventos, usuarios: usuarios, papeis: papeis}
}

type StaffComUsuario struct {
	Papel domain.PapelEvento
	Nome  string
	Email string
}

// Adicionar é a versão simplificada do "convite" pra staff (seção 3):
// em vez de um link público, o organizador atribui direto pelo e-mail
// de alguém que já tem conta — a pessoa não precisa preencher ficha
// nenhuma, só ganha acesso ao leitor de check-in.
func (s *StaffService) Adicionar(organizadorID, eventoID int64, email string) (*StaffComUsuario, error) {
	if _, err := s.buscarEventoDoOrganizador(organizadorID, eventoID); err != nil {
		return nil, err
	}

	usuario, err := s.usuarios.BuscarPorEmail(email)
	if err != nil {
		return nil, ErrUsuarioNaoEncontrado
	}

	papel, err := s.papeis.Buscar(eventoID, usuario.ID, domain.PapelStaff)
	if err != nil {
		papel = &domain.PapelEvento{
			EventoID: eventoID, UsuarioID: usuario.ID, Papel: domain.PapelStaff,
			Origem: domain.OrigemManual, Status: domain.StatusPapelConfirmado, CriadoEm: time.Now(),
		}
		if err := s.papeis.Criar(papel); err != nil {
			return nil, err
		}
	} else if papel.Status != domain.StatusPapelConfirmado {
		papel.Status = domain.StatusPapelConfirmado
		if err := s.papeis.Salvar(papel); err != nil {
			return nil, err
		}
	}

	return &StaffComUsuario{Papel: *papel, Nome: usuario.Nome, Email: usuario.Email}, nil
}

func (s *StaffService) Listar(organizadorID, eventoID int64) ([]StaffComUsuario, error) {
	if _, err := s.buscarEventoDoOrganizador(organizadorID, eventoID); err != nil {
		return nil, err
	}

	papeis, err := s.papeis.ListarPorEvento(eventoID)
	if err != nil {
		return nil, err
	}

	var resultado []StaffComUsuario
	for _, p := range papeis {
		if p.Papel != domain.PapelStaff || p.Status != domain.StatusPapelConfirmado {
			continue
		}
		usuario, err := s.usuarios.BuscarPorID(p.UsuarioID)
		if err != nil {
			continue
		}
		resultado = append(resultado, StaffComUsuario{Papel: p, Nome: usuario.Nome, Email: usuario.Email})
	}
	return resultado, nil
}

func (s *StaffService) Remover(organizadorID, eventoID, usuarioID int64) error {
	if _, err := s.buscarEventoDoOrganizador(organizadorID, eventoID); err != nil {
		return err
	}

	papel, err := s.papeis.Buscar(eventoID, usuarioID, domain.PapelStaff)
	if err != nil {
		return ErrStaffNaoEncontrado
	}
	papel.Status = domain.StatusPapelRemovido
	return s.papeis.Salvar(papel)
}

func (s *StaffService) buscarEventoDoOrganizador(organizadorID, eventoID int64) (*domain.Evento, error) {
	evento, err := s.eventos.BuscarPorID(eventoID)
	if err != nil {
		return nil, err
	}
	if evento.OrganizadorID != organizadorID {
		return nil, ErrEventoNaoPertenceAoOrganizador
	}
	return evento, nil
}
