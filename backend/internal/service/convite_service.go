package service

import (
	"errors"
	"time"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
)

var (
	ErrConviteInvalido    = errors.New("convite inválido, expirado ou esgotado")
	ErrConviteNaoPertence = errors.New("convite não pertence a este organizador")
)

type ConviteService struct {
	convites *repository.ConviteRepository
	eventos  *EventoService
	papeis   *repository.PapelEventoRepository
	fichas   *repository.FichaParticipacaoRepository
}

func NovoConviteService(convites *repository.ConviteRepository, eventos *EventoService, papeis *repository.PapelEventoRepository, fichas *repository.FichaParticipacaoRepository) *ConviteService {
	return &ConviteService{convites: convites, eventos: eventos, papeis: papeis, fichas: fichas}
}

func (s *ConviteService) Gerar(organizadorID, eventoID int64, tipo domain.TipoConvite, maxUsos *int, expiraEm *time.Time, criadoPor int64) (*domain.Convite, string, error) {
	if _, err := s.eventos.BuscarDoOrganizador(organizadorID, eventoID); err != nil {
		return nil, "", err
	}

	tokenPlano, err := GerarTokenOpaco()
	if err != nil {
		return nil, "", err
	}

	convite := &domain.Convite{
		EventoID:  eventoID,
		Tipo:      tipo,
		TokenHash: HashToken(tokenPlano),
		MaxUsos:   maxUsos,
		ExpiraEm:  expiraEm,
		CriadoPor: criadoPor,
		CriadoEm:  time.Now(),
	}
	if err := s.convites.Criar(convite); err != nil {
		return nil, "", err
	}
	return convite, tokenPlano, nil
}

func (s *ConviteService) Listar(organizadorID, eventoID int64) ([]domain.Convite, error) {
	if _, err := s.eventos.BuscarDoOrganizador(organizadorID, eventoID); err != nil {
		return nil, err
	}
	return s.convites.ListarPorEvento(eventoID)
}

func (s *ConviteService) Revogar(organizadorID, eventoID, conviteID int64) error {
	if _, err := s.eventos.BuscarDoOrganizador(organizadorID, eventoID); err != nil {
		return err
	}
	convite, err := s.convites.BuscarPorID(conviteID)
	if err != nil {
		return err
	}
	if convite.EventoID != eventoID {
		return ErrConviteNaoPertence
	}
	agora := time.Now()
	convite.RevogadoEm = &agora
	return s.convites.Salvar(convite)
}

// ConsultarPorToken é usado pela tela pública /convite/:token (mostra
// evento + tipo antes do login).
func (s *ConviteService) ConsultarPorToken(tokenPlano string) (*domain.Convite, error) {
	convite, err := s.convites.BuscarPorTokenHash(HashToken(tokenPlano))
	if err != nil {
		return nil, ErrConviteInvalido
	}
	if !s.convites.Valido(convite) {
		return nil, ErrConviteInvalido
	}
	return convite, nil
}

// Aceitar confirma o convite para o usuário logado: cria (ou reaproveita)
// o papel_evento como "confirmado" e a ficha já "aprovado" — seção 7.11
// do plano ("a ficha nasce já como aprovado ao ser confirmada"). Reuso
// pelo mesmo usuário não duplica papel (único por evento+usuario+papel).
func (s *ConviteService) Aceitar(tokenPlano string, usuarioID int64, dados FichaDados) (*domain.PapelEvento, *domain.FichaParticipacao, error) {
	convite, err := s.ConsultarPorToken(tokenPlano)
	if err != nil {
		return nil, nil, err
	}

	papelTipo := domain.PapelJurado
	fichaPapel := domain.PapelJurado
	if convite.Tipo == domain.TipoConviteParticipanteEspecial {
		papelTipo = domain.PapelParticipante
		fichaPapel = domain.PapelParticipante
	}

	papel, err := s.papeis.Buscar(convite.EventoID, usuarioID, papelTipo)
	if err != nil {
		papel = &domain.PapelEvento{
			EventoID:  convite.EventoID,
			UsuarioID: usuarioID,
			Papel:     papelTipo,
			Origem:    domain.OrigemConvite,
			Status:    domain.StatusPapelConfirmado,
			ConviteID: &convite.ID,
			CriadoEm:  time.Now(),
		}
		if err := s.papeis.Criar(papel); err != nil {
			return nil, nil, err
		}
		convite.Usos++
		if err := s.convites.Salvar(convite); err != nil {
			return nil, nil, err
		}
	}

	ficha, err := s.fichas.Buscar(convite.EventoID, usuarioID, fichaPapel)
	if err != nil {
		ficha = &domain.FichaParticipacao{
			EventoID:  convite.EventoID,
			UsuarioID: usuarioID,
			Papel:     fichaPapel,
			Origem:    domain.OrigemConvite,
			Status:    domain.StatusFichaAprovado,
			CriadoEm:  time.Now(),
		}
	}
	aplicarDadosFicha(ficha, dados)
	if err := validarMenorDeIdade(ficha); err != nil {
		return nil, nil, err
	}
	ficha.Status = domain.StatusFichaAprovado

	if ficha.ID == 0 {
		if err := s.fichas.Criar(ficha); err != nil {
			return nil, nil, err
		}
	} else {
		if err := s.fichas.Salvar(ficha); err != nil {
			return nil, nil, err
		}
	}

	return papel, ficha, nil
}
