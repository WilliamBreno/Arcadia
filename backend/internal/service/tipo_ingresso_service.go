package service

import (
	"errors"
	"strings"
	"time"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
)

type TipoIngressoService struct {
	eventos *EventoService
	tipos   *repository.TipoIngressoRepository
	sessoes *repository.SessaoRepository
}

func NovoTipoIngressoService(eventos *EventoService, tipos *repository.TipoIngressoRepository, sessoes *repository.SessaoRepository) *TipoIngressoService {
	return &TipoIngressoService{eventos: eventos, tipos: tipos, sessoes: sessoes}
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
	SessaoIDs     []int64 // vazio = todas as sessões
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
		SessaoIDs:     dados.SessaoIDs,
		Ativo:         dados.Ativo,
		CriadoEm:      time.Now(),
	}
	if err := s.checarSessao(eventoID, tipo); err != nil {
		return nil, err
	}
	if err := s.checarCotaMeia(eventoID, tipo); err != nil {
		return nil, err
	}
	if err := s.tipos.Criar(tipo); err != nil {
		return nil, err
	}
	if err := s.tipos.DefinirSessoes(tipo.ID, tipo.SessaoIDs); err != nil {
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
	tipo.SessaoIDs = dados.SessaoIDs
	tipo.Ativo = dados.Ativo

	if err := s.checarSessao(eventoID, tipo); err != nil {
		return nil, err
	}
	if err := s.checarCotaMeia(eventoID, tipo); err != nil {
		return nil, err
	}
	if err := s.tipos.Salvar(tipo); err != nil {
		return nil, err
	}
	if err := s.tipos.DefinirSessoes(tipo.ID, tipo.SessaoIDs); err != nil {
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

var ErrSessaoInvalidaParaTipo = errors.New("a sessão informada não existe neste evento ou está cancelada")

// checarSessao: tipo restrito só aceita sessões ATIVAS do mesmo evento.
func (s *TipoIngressoService) checarSessao(eventoID int64, t *domain.TipoIngresso) error {
	for _, id := range t.SessaoIDs {
		sessao, err := s.sessoes.BuscarPorID(id)
		if err != nil || sessao.EventoID != eventoID || sessao.Status != domain.SessaoAtiva {
			return ErrSessaoInvalidaParaTipo
		}
	}
	return nil
}

// PrecoPorSessao é uma linha do "um ingresso por dia": cada sessão com preço
// e quantidade próprios.
type PrecoPorSessao struct {
	SessaoID      int64
	PrecoCentavos int64
	Quantidade    int
}

// CriarPorSessao cria, de uma vez, um tipo de ingresso para cada sessão
// informada (válido só nela), com preço e quantidade próprios — preços
// diferentes por dia ou "cada dia o seu ingresso". Valida tudo antes de criar.
func (s *TipoIngressoService) CriarPorSessao(organizadorID, eventoID int64, base TipoIngressoDados, linhas []PrecoPorSessao) ([]domain.TipoIngresso, error) {
	if _, err := s.eventos.BuscarDoOrganizador(organizadorID, eventoID); err != nil {
		return nil, err
	}
	if len(linhas) == 0 {
		return nil, ErrSessaoInvalidaParaTipo
	}
	rotulos := map[int64]string{}
	for _, l := range linhas {
		sessao, err := s.sessoes.BuscarPorID(l.SessaoID)
		if err != nil || sessao.EventoID != eventoID || sessao.Status != domain.SessaoAtiva || l.Quantidade <= 0 || l.PrecoCentavos < 0 {
			return nil, ErrSessaoInvalidaParaTipo
		}
		rotulo := strings.TrimSpace(sessao.Titulo)
		if rotulo == "" {
			rotulo = sessao.InicioEm.In(time.Local).Format("02/01")
		}
		rotulos[l.SessaoID] = rotulo
	}

	var criados []domain.TipoIngresso
	for i, l := range linhas {
		d := base
		d.Nome = strings.TrimSpace(base.Nome) + " — " + rotulos[l.SessaoID]
		d.PrecoCentavos, d.Quantidade, d.SessaoIDs, d.Ordem = l.PrecoCentavos, l.Quantidade, []int64{l.SessaoID}, base.Ordem+i
		tipo, err := s.Criar(organizadorID, eventoID, d)
		if err != nil {
			return criados, err
		}
		criados = append(criados, *tipo)
	}
	return criados, nil
}
