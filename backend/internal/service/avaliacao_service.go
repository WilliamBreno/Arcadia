package service

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
)

var (
	ErrCriterioInvalido      = errors.New("critério inválido: peso > 0, nota máxima > mínima e passo > 0")
	ErrCriterioNaoEncontrado = errors.New("critério não encontrado")
	ErrCriterioEmUso         = errors.New("critério já tem notas lançadas e não pode ser alterado ou excluído")
	ErrAvaliacaoFinalizada   = errors.New("avaliação já finalizada — não pode mais ser editada")
	ErrNotaInvalida          = errors.New("nota fora da escala do critério")
	ErrAvaliacaoIncompleta   = errors.New("para finalizar, dê nota a todos os critérios aplicáveis")
	ErrResultadoOculto       = errors.New("o resultado ainda não foi liberado")
)

type CriterioDados struct {
	Nome             string
	Peso             float64
	NotaMin          float64
	NotaMax          float64
	Passo            float64
	TipoApresentacao *domain.TipoApresentacao
	Ordem            int
}

type NotaInput struct {
	CriterioID int64
	Nota       float64
	Comentario string
}

type AvaliacaoService struct {
	eventos    *EventoService
	eventoRepo *repository.EventoRepository
	fichas     *repository.FichaParticipacaoRepository
	papeis     *repository.PapelEventoRepository
	avaliacoes *repository.AvaliacaoRepository
}

func NovoAvaliacaoService(
	eventos *EventoService, eventoRepo *repository.EventoRepository, fichas *repository.FichaParticipacaoRepository,
	papeis *repository.PapelEventoRepository, avaliacoes *repository.AvaliacaoRepository,
) *AvaliacaoService {
	return &AvaliacaoService{eventos: eventos, eventoRepo: eventoRepo, fichas: fichas, papeis: papeis, avaliacoes: avaliacoes}
}

// --- organizador: critérios ---

func validarCriterio(d CriterioDados) error {
	if d.Nome == "" || d.Peso <= 0 || d.NotaMax <= d.NotaMin || d.Passo <= 0 {
		return ErrCriterioInvalido
	}
	return nil
}

func (s *AvaliacaoService) ListarCriterios(organizadorID, eventoID int64) ([]domain.CriterioAvaliacao, error) {
	if _, err := s.eventos.BuscarDoOrganizador(organizadorID, eventoID); err != nil {
		return nil, err
	}
	return s.avaliacoes.CriteriosPorEvento(eventoID)
}

func (s *AvaliacaoService) CriarCriterio(organizadorID, eventoID int64, d CriterioDados) (*domain.CriterioAvaliacao, error) {
	if _, err := s.eventos.BuscarDoOrganizador(organizadorID, eventoID); err != nil {
		return nil, err
	}
	if err := validarCriterio(d); err != nil {
		return nil, err
	}
	c := &domain.CriterioAvaliacao{EventoID: eventoID, Nome: d.Nome, Peso: d.Peso, NotaMin: d.NotaMin, NotaMax: d.NotaMax,
		Passo: d.Passo, TipoApresentacao: d.TipoApresentacao, Ordem: d.Ordem, CriadoEm: time.Now()}
	return c, s.avaliacoes.CriarCriterio(c)
}

func (s *AvaliacaoService) criterioEditavel(organizadorID, eventoID, criterioID int64) (*domain.CriterioAvaliacao, error) {
	if _, err := s.eventos.BuscarDoOrganizador(organizadorID, eventoID); err != nil {
		return nil, err
	}
	c, err := s.avaliacoes.BuscarCriterio(criterioID)
	if err != nil || c.EventoID != eventoID {
		return nil, ErrCriterioNaoEncontrado
	}
	usos, err := s.avaliacoes.ContarAvaliacoesDoCriterio(criterioID)
	if err != nil {
		return nil, err
	}
	if usos > 0 {
		return nil, ErrCriterioEmUso
	}
	return c, nil
}

func (s *AvaliacaoService) AtualizarCriterio(organizadorID, eventoID, criterioID int64, d CriterioDados) (*domain.CriterioAvaliacao, error) {
	c, err := s.criterioEditavel(organizadorID, eventoID, criterioID)
	if err != nil {
		return nil, err
	}
	if err := validarCriterio(d); err != nil {
		return nil, err
	}
	c.Nome, c.Peso, c.NotaMin, c.NotaMax, c.Passo, c.TipoApresentacao, c.Ordem = d.Nome, d.Peso, d.NotaMin, d.NotaMax, d.Passo, d.TipoApresentacao, d.Ordem
	return c, s.avaliacoes.SalvarCriterio(c)
}

func (s *AvaliacaoService) ExcluirCriterio(organizadorID, eventoID, criterioID int64) error {
	c, err := s.criterioEditavel(organizadorID, eventoID, criterioID)
	if err != nil {
		return err
	}
	return s.avaliacoes.ExcluirCriterio(c)
}

// --- jurado ---

func (s *AvaliacaoService) exigirJurado(usuarioID, eventoID int64) error {
	papel, err := s.papeis.Buscar(eventoID, usuarioID, domain.PapelJurado)
	if err != nil || papel.Status != domain.StatusPapelConfirmado {
		return ErrNaoEhJuradoConfirmado
	}
	return nil
}

func (s *AvaliacaoService) fichaAvaliavel(eventoID, fichaID int64) (*domain.FichaParticipacao, error) {
	f, err := s.fichas.BuscarPorID(fichaID)
	if err != nil || f.EventoID != eventoID || f.Papel != domain.PapelParticipante || f.Status != domain.StatusFichaAprovado {
		return nil, ErrFichaNaoEncontrada
	}
	return f, nil
}

func tipoDaFicha(f *domain.FichaParticipacao) string {
	if f.TipoApresentacao == nil {
		return ""
	}
	return string(*f.TipoApresentacao)
}

func (s *AvaliacaoService) criteriosAplicaveis(eventoID int64, f *domain.FichaParticipacao) ([]domain.CriterioAvaliacao, error) {
	todos, err := s.avaliacoes.CriteriosPorEvento(eventoID)
	if err != nil {
		return nil, err
	}
	var aplicaveis []domain.CriterioAvaliacao
	for i := range todos {
		if criterioAplica(&todos[i], tipoDaFicha(f)) {
			aplicaveis = append(aplicaveis, todos[i])
		}
	}
	return aplicaveis, nil
}

// CriteriosParaJurado devolve os critérios que valem para a ficha.
func (s *AvaliacaoService) CriteriosParaJurado(usuarioID, eventoID, fichaID int64) ([]domain.CriterioAvaliacao, error) {
	if err := s.exigirJurado(usuarioID, eventoID); err != nil {
		return nil, err
	}
	f, err := s.fichaAvaliavel(eventoID, fichaID)
	if err != nil {
		return nil, err
	}
	return s.criteriosAplicaveis(eventoID, f)
}

// MinhaAvaliacao é só do próprio jurado: um jurado nunca vê a nota de outro.
func (s *AvaliacaoService) MinhaAvaliacao(usuarioID, eventoID, fichaID int64) ([]domain.Avaliacao, error) {
	if err := s.exigirJurado(usuarioID, eventoID); err != nil {
		return nil, err
	}
	if _, err := s.fichaAvaliavel(eventoID, fichaID); err != nil {
		return nil, err
	}
	return s.avaliacoes.DoJuradoNaFicha(fichaID, usuarioID)
}

// Avaliar grava/atualiza as notas do jurado numa ficha; com finalizar=true
// exige todos os critérios aplicáveis preenchidos e trava a avaliação.
func (s *AvaliacaoService) Avaliar(usuarioID, eventoID, fichaID int64, notas []NotaInput, finalizar bool) ([]domain.Avaliacao, error) {
	if err := s.exigirJurado(usuarioID, eventoID); err != nil {
		return nil, err
	}
	ficha, err := s.fichaAvaliavel(eventoID, fichaID)
	if err != nil {
		return nil, err
	}
	aplicaveis, err := s.criteriosAplicaveis(eventoID, ficha)
	if err != nil {
		return nil, err
	}
	porID := map[int64]*domain.CriterioAvaliacao{}
	for i := range aplicaveis {
		porID[aplicaveis[i].ID] = &aplicaveis[i]
	}

	existentes, err := s.avaliacoes.DoJuradoNaFicha(fichaID, usuarioID)
	if err != nil {
		return nil, err
	}
	preenchidos := map[int64]bool{}
	for _, e := range existentes {
		if e.Finalizada {
			return nil, ErrAvaliacaoFinalizada
		}
		preenchidos[e.CriterioID] = true
	}

	agora := time.Now()
	novas := make([]domain.Avaliacao, 0, len(notas))
	for _, n := range notas {
		c, ok := porID[n.CriterioID]
		if !ok {
			return nil, fmt.Errorf("%w: critério %d não se aplica a esta ficha", ErrCriterioNaoEncontrado, n.CriterioID)
		}
		if !notaValida(n.Nota, c) {
			return nil, fmt.Errorf("%w: %s (de %g a %g, passo %g)", ErrNotaInvalida, c.Nome, c.NotaMin, c.NotaMax, c.Passo)
		}
		preenchidos[n.CriterioID] = true
		novas = append(novas, domain.Avaliacao{FichaID: fichaID, JuradoUsuarioID: usuarioID, CriterioID: n.CriterioID,
			Nota: n.Nota, Comentario: n.Comentario, CriadoEm: agora})
	}
	if finalizar {
		for _, c := range aplicaveis {
			if !preenchidos[c.ID] {
				return nil, fmt.Errorf("%w: falta %s", ErrAvaliacaoIncompleta, c.Nome)
			}
		}
	}

	err = s.avaliacoes.DB().Transaction(func(tx *gorm.DB) error {
		if err := s.avaliacoes.SalvarNotas(tx, novas); err != nil {
			return err
		}
		if finalizar {
			return s.avaliacoes.Finalizar(tx, fichaID, usuarioID)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.avaliacoes.DoJuradoNaFicha(fichaID, usuarioID)
}

// --- ranking e resultado ---

func (s *AvaliacaoService) calcularRanking(eventoID int64) (map[string][]LinhaRanking, error) {
	criterios, err := s.avaliacoes.CriteriosPorEvento(eventoID)
	if err != nil {
		return nil, err
	}
	aprovado := domain.StatusFichaAprovado
	fichas, err := s.fichas.ListarPorEventoEPapel(eventoID, domain.PapelParticipante, &aprovado)
	if err != nil {
		return nil, err
	}
	entradas := make([]EntradaRanking, 0, len(fichas))
	for i := range fichas {
		entradas = append(entradas, EntradaRanking{FichaID: fichas[i].ID, Nome: fichas[i].Nome,
			NomeArtistico: fichas[i].NomeArtistico, TipoApresentacao: tipoDaFicha(&fichas[i]), Menor: fichas[i].MenorDeIdade()})
	}
	avs, err := s.avaliacoes.FinalizadasDoEvento(eventoID)
	if err != nil {
		return nil, err
	}
	return CalcularRanking(criterios, entradas, avs), nil
}

// RankingDoOrganizador mostra o ranking a qualquer momento (parcial).
func (s *AvaliacaoService) RankingDoOrganizador(organizadorID, eventoID int64) (map[string][]LinhaRanking, *time.Time, error) {
	evento, err := s.eventos.BuscarDoOrganizador(organizadorID, eventoID)
	if err != nil {
		return nil, nil, err
	}
	r, err := s.calcularRanking(eventoID)
	return r, evento.ResultadoLiberadoEm, err
}

func (s *AvaliacaoService) DefinirResultadoLiberado(organizadorID, eventoID int64, liberar bool) error {
	evento, err := s.eventos.BuscarDoOrganizador(organizadorID, eventoID)
	if err != nil {
		return err
	}
	if liberar {
		agora := time.Now()
		evento.ResultadoLiberadoEm = &agora
	} else {
		evento.ResultadoLiberadoEm = nil
	}
	return s.eventoRepo.Salvar(evento)
}

// ResultadoPublico só responde depois que o organizador liberou.
func (s *AvaliacaoService) ResultadoPublico(evento *domain.Evento) (map[string][]LinhaRanking, error) {
	if evento.ResultadoLiberadoEm == nil {
		return nil, ErrResultadoOculto
	}
	return s.calcularRanking(evento.ID)
}
