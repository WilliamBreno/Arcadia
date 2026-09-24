package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/WilliamBreno/Arcadia/backend/internal/mail"
	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
)

var (
	ErrSessaoInvalida      = errors.New("informe o início da sessão; o fim não pode ser antes do início")
	ErrSessaoNaoEncontrada = errors.New("sessão não encontrada")
	ErrSessaoJaCancelada   = errors.New("sessão já cancelada")
	ErrSessaoEmUso         = errors.New("sessão já tem ingressos ou entradas e não pode ser excluída — cancele-a")
	ErrSessaoMotivo        = errors.New("informe o motivo do cancelamento")
)

type SessaoDados struct {
	Titulo   string
	InicioEm time.Time
	FimEm    *time.Time
}

// SessaoService cuida de eventos com várias sessões (datas). Estoque é por
// tipo de ingresso (compartilhado entre sessões); tipo com sessao_id só vale
// para aquela sessão. evento.inicio_em/fim_em passam a ser derivados das
// sessões ativas — assim o repasse sai depois da ÚLTIMA sessão e as regras
// de cancelamento contam a partir da PRIMEIRA.
type SessaoService struct {
	eventos      *EventoService
	eventoRepo   *repository.EventoRepository
	sessoes      *repository.SessaoRepository
	cancelamento *CancelamentoService
	mailCliente  *mail.Cliente
	plataforma   string
}

func NovoSessaoService(e *EventoService, er *repository.EventoRepository, s *repository.SessaoRepository,
	c *CancelamentoService, m *mail.Cliente, plataforma string) *SessaoService {
	return &SessaoService{eventos: e, eventoRepo: er, sessoes: s, cancelamento: c, mailCliente: m, plataforma: plataforma}
}

func validarSessao(d SessaoDados) error {
	if d.InicioEm.IsZero() || (d.FimEm != nil && d.FimEm.Before(d.InicioEm)) {
		return ErrSessaoInvalida
	}
	return nil
}

// recalcularPeriodo alinha o período do evento ao das sessões ativas.
func (s *SessaoService) recalcularPeriodo(eventoID int64) error {
	lista, err := s.sessoes.ListarPorEvento(eventoID)
	if err != nil {
		return err
	}
	ini, fim, ok := PeriodoDoEvento(lista)
	if !ok {
		return nil
	}
	evento, err := s.eventoRepo.BuscarPorID(eventoID)
	if err != nil {
		return err
	}
	evento.InicioEm, evento.FimEm = &ini, &fim
	return s.eventoRepo.Salvar(evento)
}

func (s *SessaoService) Listar(organizadorID, eventoID int64) ([]domain.Sessao, error) {
	if _, err := s.eventos.BuscarDoOrganizador(organizadorID, eventoID); err != nil {
		return nil, err
	}
	return s.sessoes.ListarPorEvento(eventoID)
}

func (s *SessaoService) Criar(organizadorID, eventoID int64, d SessaoDados) (*domain.Sessao, error) {
	if _, err := s.eventos.BuscarDoOrganizador(organizadorID, eventoID); err != nil {
		return nil, err
	}
	if err := validarSessao(d); err != nil {
		return nil, err
	}
	sessao := &domain.Sessao{EventoID: eventoID, Titulo: strings.TrimSpace(d.Titulo), InicioEm: d.InicioEm, FimEm: d.FimEm,
		Status: domain.SessaoAtiva, CriadoEm: time.Now()}
	if err := s.sessoes.Criar(sessao); err != nil {
		return nil, err
	}
	return sessao, s.recalcularPeriodo(eventoID)
}

func (s *SessaoService) doEvento(organizadorID, eventoID, sessaoID int64) (*domain.Sessao, error) {
	if _, err := s.eventos.BuscarDoOrganizador(organizadorID, eventoID); err != nil {
		return nil, err
	}
	sessao, err := s.sessoes.BuscarPorID(sessaoID)
	if err != nil || sessao.EventoID != eventoID {
		return nil, ErrSessaoNaoEncontrada
	}
	return sessao, nil
}

func (s *SessaoService) Atualizar(organizadorID, eventoID, sessaoID int64, d SessaoDados) (*domain.Sessao, error) {
	sessao, err := s.doEvento(organizadorID, eventoID, sessaoID)
	if err != nil {
		return nil, err
	}
	if sessao.Status != domain.SessaoAtiva {
		return nil, ErrSessaoJaCancelada
	}
	if err := validarSessao(d); err != nil {
		return nil, err
	}
	sessao.Titulo, sessao.InicioEm, sessao.FimEm = strings.TrimSpace(d.Titulo), d.InicioEm, d.FimEm
	if err := s.sessoes.Salvar(sessao); err != nil {
		return nil, err
	}
	return sessao, s.recalcularPeriodo(eventoID)
}

// Cancelar (só dono — mexe em dinheiro): a sessão fica cancelada, deixa de
// vender os tipos restritos a ela e reembolsa integralmente SÓ os ingressos
// restritos a ela. Ingresso do evento todo continua valendo (o titular é
// avisado por e-mail). Cancelar o evento inteiro é outro fluxo.
func (s *SessaoService) Cancelar(organizadorID, eventoID, sessaoID int64, motivo string) (sucessos int, falhas []int64, err error) {
	sessao, err := s.doEvento(organizadorID, eventoID, sessaoID)
	if err != nil {
		return 0, nil, err
	}
	if sessao.Status == domain.SessaoCancelada {
		return 0, nil, ErrSessaoJaCancelada
	}
	motivo = strings.TrimSpace(motivo)
	if len(motivo) < 3 {
		return 0, nil, ErrSessaoMotivo
	}
	evento, err := s.eventoRepo.BuscarPorID(eventoID)
	if err != nil {
		return 0, nil, err
	}

	agora := time.Now()
	sessao.Status, sessao.CanceladaEm, sessao.MotivoCancelamento = domain.SessaoCancelada, &agora, motivo
	if err := s.sessoes.Salvar(sessao); err != nil {
		return 0, nil, err
	}
	if err := s.sessoes.DesativarTiposSemSessaoAtiva(eventoID); err != nil {
		return 0, nil, err
	}

	itens, err := s.sessoes.ListarPagosSemSessaoAtiva(eventoID, sessaoID)
	if err != nil {
		return 0, nil, err
	}
	sucessos, falhas = s.cancelamento.reembolsarItens(evento, itens, organizadorID, "sessão cancelada pelo organizador: "+motivo)

	if err := s.recalcularPeriodo(eventoID); err != nil {
		return sucessos, falhas, err
	}
	if emails, errE := s.sessoes.TitularesAfetados(eventoID, sessaoID); errE == nil {
		titulo := sessao.Titulo
		if titulo == "" {
			titulo = sessao.InicioEm.In(time.Local).Format("02/01/2006 15:04")
		}
		for _, email := range emails {
			_ = s.mailCliente.Enviar(email, "Sessão cancelada — "+evento.Titulo+" — "+s.plataforma,
				fmt.Sprintf("<p>A sessão <strong>%s</strong> de <strong>%s</strong> foi cancelada pelo organizador. Seu ingresso continua válido para as demais sessões a que ele dá direito.</p><p>Motivo: %s</p>", titulo, evento.Titulo, motivo))
		}
	}
	return sucessos, falhas, nil
}

// CriarLote cria várias sessões de uma vez (um fim de semana, todas as sextas
// do mês...). Valida todas antes de gravar a primeira.
func (s *SessaoService) CriarLote(organizadorID, eventoID int64, itens []SessaoDados) ([]domain.Sessao, error) {
	if _, err := s.eventos.BuscarDoOrganizador(organizadorID, eventoID); err != nil {
		return nil, err
	}
	if len(itens) == 0 || len(itens) > 100 {
		return nil, ErrSessaoInvalida
	}
	for _, d := range itens {
		if err := validarSessao(d); err != nil {
			return nil, err
		}
	}
	criadas := make([]domain.Sessao, 0, len(itens))
	for _, d := range itens {
		sessao := domain.Sessao{EventoID: eventoID, Titulo: strings.TrimSpace(d.Titulo), InicioEm: d.InicioEm, FimEm: d.FimEm,
			Status: domain.SessaoAtiva, CriadoEm: time.Now()}
		if err := s.sessoes.Criar(&sessao); err != nil {
			return criadas, err
		}
		criadas = append(criadas, sessao)
	}
	return criadas, s.recalcularPeriodo(eventoID)
}

// Excluir só sessão sem tipos de ingresso nem entradas.
func (s *SessaoService) Excluir(organizadorID, eventoID, sessaoID int64) error {
	sessao, err := s.doEvento(organizadorID, eventoID, sessaoID)
	if err != nil {
		return err
	}
	usada, err := s.sessoes.EmUso(sessaoID)
	if err != nil {
		return err
	}
	if usada {
		return ErrSessaoEmUso
	}
	if err := s.sessoes.Excluir(sessao); err != nil {
		return err
	}
	return s.recalcularPeriodo(eventoID)
}
