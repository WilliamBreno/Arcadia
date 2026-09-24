package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/WilliamBreno/Arcadia/backend/internal/mail"
	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
)

var (
	ErrEventoSemAprovacaoManual = errors.New("este evento não usa aprovação manual")
	ErrSolicitacaoJaExiste      = errors.New("você já solicitou participação neste evento")
	ErrSolicitacaoNaoEncontrada = errors.New("solicitação não encontrada")
	ErrDecisaoInvalida          = errors.New("decisão inválida (use aprovar, rejeitar ou lista_espera)")
	ErrPrecisaAprovacao         = errors.New("este evento exige aprovação do organizador antes da compra")
)

// PromotorListaEspera é o gancho chamado quando uma vaga volta ao estoque
// (reserva expirada, cancelamento, cortesia revogada).
type PromotorListaEspera interface {
	PromoverListaEspera(eventoID int64)
}

type PlateiaService struct {
	eventos      *repository.EventoRepository
	organizador  *EventoService
	solicitacoes *repository.SolicitacaoPlateiaRepository
	usuarios     *repository.UsuarioRepository
	mailCliente  *mail.Cliente
	frontendURL  string
	plataforma   string
}

func NovoPlateiaService(
	eventos *repository.EventoRepository, organizador *EventoService, solicitacoes *repository.SolicitacaoPlateiaRepository,
	usuarios *repository.UsuarioRepository, mailCliente *mail.Cliente, frontendURL, plataforma string,
) *PlateiaService {
	return &PlateiaService{eventos: eventos, organizador: organizador, solicitacoes: solicitacoes,
		usuarios: usuarios, mailCliente: mailCliente, frontendURL: frontendURL, plataforma: plataforma}
}

// PodeComprar: evento sem aprovação manual sempre pode; com ela, só quem
// tem solicitação aprovada.
func (s *PlateiaService) PodeComprar(evento *domain.Evento, usuarioID int64) bool {
	if !evento.AprovacaoManual {
		return true
	}
	sol, err := s.solicitacoes.BuscarPorEventoEUsuario(evento.ID, usuarioID)
	return err == nil && sol.Status == domain.SolicitacaoAprovada
}

func (s *PlateiaService) Solicitar(usuarioID int64, evento *domain.Evento) (*domain.SolicitacaoPlateia, error) {
	if !evento.AprovacaoManual {
		return nil, ErrEventoSemAprovacaoManual
	}
	if _, err := s.solicitacoes.BuscarPorEventoEUsuario(evento.ID, usuarioID); err == nil {
		return nil, ErrSolicitacaoJaExiste
	}
	sol := &domain.SolicitacaoPlateia{EventoID: evento.ID, UsuarioID: usuarioID, Status: domain.SolicitacaoPendente, CriadoEm: time.Now()}
	if err := s.solicitacoes.Criar(sol); err != nil {
		return nil, err
	}
	return sol, nil
}

func (s *PlateiaService) Minha(usuarioID, eventoID int64) (*domain.SolicitacaoPlateia, error) {
	sol, err := s.solicitacoes.BuscarPorEventoEUsuario(eventoID, usuarioID)
	if err != nil {
		return nil, ErrSolicitacaoNaoEncontrada
	}
	return sol, nil
}

func (s *PlateiaService) Listar(organizadorID, eventoID int64) ([]repository.SolicitacaoDetalhada, error) {
	if _, err := s.organizador.BuscarDoOrganizador(organizadorID, eventoID); err != nil {
		return nil, err
	}
	return s.solicitacoes.ListarPorEvento(eventoID)
}

// Decidir aplica a decisão do organizador. "aprovar" sem vaga livre vira
// lista_espera (aprovado, aguardando vaga); a promoção é automática.
func (s *PlateiaService) Decidir(organizadorID, eventoID, solicitacaoID int64, decisao string) (*domain.SolicitacaoPlateia, error) {
	evento, err := s.organizador.BuscarDoOrganizador(organizadorID, eventoID)
	if err != nil {
		return nil, err
	}
	sol, err := s.solicitacoes.BuscarPorID(solicitacaoID)
	if err != nil || sol.EventoID != eventoID {
		return nil, ErrSolicitacaoNaoEncontrada
	}

	agora := time.Now()
	switch decisao {
	case "aprovar":
		sol.Status = domain.SolicitacaoAprovada
		if vagas, err := s.vagasParaAprovar(eventoID); err == nil && vagas <= 0 {
			sol.Status = domain.SolicitacaoListaEspera
		}
	case "rejeitar":
		sol.Status = domain.SolicitacaoRejeitada
	case "lista_espera":
		sol.Status = domain.SolicitacaoListaEspera
	default:
		return nil, ErrDecisaoInvalida
	}
	sol.DecididoEm = &agora
	if err := s.solicitacoes.Salvar(sol); err != nil {
		return nil, err
	}
	s.notificar(evento, sol)
	if sol.Status != domain.SolicitacaoAprovada {
		s.PromoverListaEspera(eventoID) // a vaga prometida a este usuário pode ter sido liberada
	}
	return sol, nil
}

func (s *PlateiaService) vagasParaAprovar(eventoID int64) (int64, error) {
	livres, err := s.solicitacoes.VagasLivres(eventoID)
	if err != nil {
		return 0, err
	}
	prometidas, err := s.solicitacoes.ContarAprovadasSemCompra(eventoID)
	if err != nil {
		return 0, err
	}
	return livres - prometidas, nil
}

// PromoverListaEspera aprova, em ordem de chegada, quantos couberem nas
// vagas livres (menos as já prometidas a aprovados que ainda não compraram).
func (s *PlateiaService) PromoverListaEspera(eventoID int64) {
	evento, err := s.eventos.BuscarPorID(eventoID)
	if err != nil || !evento.AprovacaoManual {
		return
	}
	vagas, err := s.vagasParaAprovar(eventoID)
	if err != nil || vagas <= 0 {
		return
	}
	fila, err := s.solicitacoes.ListarListaEspera(eventoID)
	if err != nil {
		return
	}
	agora := time.Now()
	for i := range fila {
		if int64(i) >= vagas {
			break
		}
		fila[i].Status = domain.SolicitacaoAprovada
		fila[i].DecididoEm = &agora
		if err := s.solicitacoes.Salvar(&fila[i]); err != nil {
			continue
		}
		s.notificar(evento, &fila[i])
	}
}

func (s *PlateiaService) notificar(evento *domain.Evento, sol *domain.SolicitacaoPlateia) {
	usuario, err := s.usuarios.BuscarPorID(sol.UsuarioID)
	if err != nil || usuario.Email == "" {
		return
	}
	link := fmt.Sprintf("%s/e/%s", s.frontendURL, evento.Slug)
	var assunto, corpo string
	switch sol.Status {
	case domain.SolicitacaoAprovada:
		assunto = "Participação aprovada — " + evento.Titulo
		corpo = fmt.Sprintf(`<p>Olá, %s!</p><p>Sua participação em <strong>%s</strong> foi aprovada. <a href="%s">Garanta seu ingresso</a> — as vagas são limitadas.</p>`, usuario.Nome, evento.Titulo, link)
	case domain.SolicitacaoRejeitada:
		assunto = "Participação não aprovada — " + evento.Titulo
		corpo = fmt.Sprintf(`<p>Olá, %s!</p><p>Infelizmente sua participação em <strong>%s</strong> não foi aprovada.</p>`, usuario.Nome, evento.Titulo)
	case domain.SolicitacaoListaEspera:
		assunto = "Você está na lista de espera — " + evento.Titulo
		corpo = fmt.Sprintf(`<p>Olá, %s!</p><p>Sua participação em <strong>%s</strong> foi aprovada, mas as vagas acabaram. Você está na lista de espera e será avisado por e-mail se uma vaga abrir.</p>`, usuario.Nome, evento.Titulo)
	default:
		return
	}
	_ = s.mailCliente.Enviar(usuario.Email, assunto+" — "+s.plataforma, corpo)
}
