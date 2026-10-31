package service

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
)

var (
	ErrConviteInvalido    = errors.New("convite inválido, expirado ou esgotado")
	ErrConviteNaoPertence = errors.New("convite não pertence a este organizador")
	ErrRemovidoDoEvento   = errors.New("você foi removido deste evento pelo organizador; fale com ele para voltar")
)

type ConviteService struct {
	convites *repository.ConviteRepository
	eventos  *EventoService
	papeis   *repository.PapelEventoRepository
	fichas   *repository.FichaParticipacaoRepository
	vagas    *VagasService
}

// DefinirVagas liga o limite de jurados e participantes (item 11.6).
func (s *ConviteService) DefinirVagas(v *VagasService) { s.vagas = v }

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
	if s.vagas == nil {
		return s.aceitar(nil, convite, usuarioID, dados)
	}
	// Item 11.6: papel, ficha e uso do convite na mesma transação, com a
	// vaga do evento travada (sem passar do limite com aceites simultâneos).
	var papel *domain.PapelEvento
	var ficha *domain.FichaParticipacao
	err = s.vagas.DB().Transaction(func(tx *gorm.DB) error {
		if err := TravarVagas(tx, convite.EventoID); err != nil {
			return err
		}
		// Revalida depois da trava: o convite pode ter sido revogado ou ter
		// vencido enquanto este aceite esperava.
		atual, err := repository.NovoConviteRepository(tx).BuscarPorID(convite.ID)
		if err != nil || !s.convites.Valido(atual) {
			return ErrConviteInvalido
		}
		var e error
		papel, ficha, e = s.aceitar(tx, convite, usuarioID, dados)
		return e
	})
	if err != nil {
		return nil, nil, err
	}
	return papel, ficha, nil
}

func (s *ConviteService) aceitar(tx *gorm.DB, convite *domain.Convite, usuarioID int64, dados FichaDados) (*domain.PapelEvento, *domain.FichaParticipacao, error) {
	papeis, fichas := s.papeis, s.fichas
	if tx != nil {
		papeis, fichas = repository.NovoPapelEventoRepository(tx), repository.NovoFichaParticipacaoRepository(tx)
	}

	papelTipo := domain.PapelJurado
	fichaPapel := domain.PapelJurado
	if convite.Tipo == domain.TipoConviteParticipanteEspecial {
		papelTipo = domain.PapelParticipante
		fichaPapel = domain.PapelParticipante
	}

	ficha, err := fichas.Buscar(convite.EventoID, usuarioID, fichaPapel)
	novaFicha := err != nil
	if novaFicha {
		ficha = &domain.FichaParticipacao{
			EventoID:  convite.EventoID,
			UsuarioID: usuarioID,
			Papel:     fichaPapel,
			Origem:    domain.OrigemConvite,
			Status:    domain.StatusFichaAprovado,
			CriadoEm:  time.Now(),
		}
	}
	jaOcupava := !novaFicha && OcupaVaga(ficha.Status)
	tipoAntes := ficha.TipoApresentacao
	aplicarDadosFicha(ficha, dados)
	if err := validarMenorDeIdade(ficha); err != nil {
		return nil, nil, err
	}

	papel, err := papeis.Buscar(convite.EventoID, usuarioID, papelTipo)
	novoPapel := err != nil
	if !novoPapel && papel.Status == domain.StatusPapelRemovido {
		return nil, nil, ErrRemovidoDoEvento
	}
	if tx != nil {
		// Jurado ocupa vaga pelo papel; participante, pela ficha (e pela
		// modalidade, quando ela muda).
		var errVaga error
		switch {
		case papelTipo == domain.PapelJurado && novoPapel,
			papelTipo == domain.PapelParticipante && !jaOcupava:
			errVaga = s.vagas.Conferir(tx, convite.EventoID, papelTipo, ficha.TipoApresentacao)
		case papelTipo == domain.PapelParticipante && !mesmoTipo(tipoAntes, ficha.TipoApresentacao):
			errVaga = s.vagas.ConferirTroca(tx, convite.EventoID, ficha.TipoApresentacao)
		}
		if errVaga != nil {
			return nil, nil, errVaga
		}
	}
	if novoPapel {
		papel = &domain.PapelEvento{
			EventoID:  convite.EventoID,
			UsuarioID: usuarioID,
			Papel:     papelTipo,
			Origem:    domain.OrigemConvite,
			Status:    domain.StatusPapelConfirmado,
			ConviteID: &convite.ID,
			CriadoEm:  time.Now(),
		}
		if err := papeis.Criar(papel); err != nil {
			return nil, nil, err
		}
		// Uso do convite somado de forma atômica: dois aceites ao mesmo
		// tempo não passam do max_usos.
		if tx != nil {
			res := tx.Exec(`UPDATE convites SET usos = usos + 1 WHERE id = ? AND revogado_em IS NULL
				AND (expira_em IS NULL OR expira_em > now()) AND (max_usos IS NULL OR usos < max_usos)`, convite.ID)
			if res.Error != nil {
				return nil, nil, res.Error
			}
			if res.RowsAffected == 0 {
				return nil, nil, ErrConviteInvalido
			}
		} else {
			convite.Usos++
			if err := s.convites.Salvar(convite); err != nil {
				return nil, nil, err
			}
		}
	}

	ficha.Status = domain.StatusFichaAprovado
	if ficha.ID == 0 {
		if err := fichas.Criar(ficha); err != nil {
			return nil, nil, err
		}
	} else {
		if err := fichas.Salvar(ficha); err != nil {
			return nil, nil, err
		}
	}

	return papel, ficha, nil
}

// mesmoTipo compara modalidades (nil = sem modalidade).
func mesmoTipo(a, b *domain.TipoApresentacao) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}
