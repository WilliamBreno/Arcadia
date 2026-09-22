package service

import (
	"errors"
	"strconv"
	"time"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
)

var (
	ErrEventoNaoPertenceAoOrganizador = errors.New("evento não pertence a este organizador")
	ErrEventoNaoEhRascunho            = errors.New("evento não está em rascunho")
	ErrPublicacaoInvalida             = errors.New("evento não atende aos requisitos para publicação")
)

type EventoService struct {
	eventos       *repository.EventoRepository
	tiposIngresso *repository.TipoIngressoRepository
	organizadores *repository.OrganizadorRepository
}

func NovoEventoService(eventos *repository.EventoRepository, tiposIngresso *repository.TipoIngressoRepository, organizadores *repository.OrganizadorRepository) *EventoService {
	return &EventoService{eventos: eventos, tiposIngresso: tiposIngresso, organizadores: organizadores}
}

type EventoDados struct {
	Titulo                    string
	LocalID                   *int64
	Descricao                 string
	Categoria                 string
	CapaURL                   string
	InicioEm                  *time.Time
	FimEm                     *time.Time
	Timezone                  string
	ClassificacaoEtaria       string
	Visibilidade              domain.Visibilidade
	TipoAcesso                domain.TipoAcesso
	ModoParticipantes         domain.ModoParticipantes
	InscricaoTalentosInicio   *time.Time
	InscricaoTalentosFim      *time.Time
	CapacidadeTotal           *int
	GarantiaHabilitada        bool
	PoliticaCancelamentoTexto string
	MaxItensPorPedido         int
}

func (s *EventoService) Criar(organizadorID int64, dados EventoDados) (*domain.Evento, error) {
	slug, err := s.gerarSlugUnico(dados.Titulo)
	if err != nil {
		return nil, err
	}

	evento := &domain.Evento{
		OrganizadorID:             organizadorID,
		LocalID:                   dados.LocalID,
		Titulo:                    dados.Titulo,
		Slug:                      slug,
		Descricao:                 dados.Descricao,
		Categoria:                 dados.Categoria,
		CapaURL:                   dados.CapaURL,
		InicioEm:                  dados.InicioEm,
		FimEm:                     dados.FimEm,
		Timezone:                  valorOuPadrao(dados.Timezone, "America/Maceio"),
		ClassificacaoEtaria:       dados.ClassificacaoEtaria,
		Visibilidade:              valorOuPadraoVisibilidade(dados.Visibilidade),
		TipoAcesso:                valorOuPadraoTipoAcesso(dados.TipoAcesso),
		Status:                    domain.StatusEventoRascunho,
		ModoParticipantes:         valorOuPadraoModoParticipantes(dados.ModoParticipantes),
		InscricaoTalentosInicio:   dados.InscricaoTalentosInicio,
		InscricaoTalentosFim:      dados.InscricaoTalentosFim,
		CapacidadeTotal:           dados.CapacidadeTotal,
		GarantiaHabilitada:        dados.GarantiaHabilitada,
		PoliticaCancelamentoTexto: dados.PoliticaCancelamentoTexto,
		MaxItensPorPedido:         valorIntOuPadrao(dados.MaxItensPorPedido, 10),
		CriadoEm:                  time.Now(),
	}
	if err := s.eventos.Criar(evento); err != nil {
		return nil, err
	}
	return evento, nil
}

func (s *EventoService) Atualizar(organizadorID, eventoID int64, dados EventoDados) (*domain.Evento, error) {
	evento, err := s.buscarDoOrganizador(organizadorID, eventoID)
	if err != nil {
		return nil, err
	}

	evento.Titulo = dados.Titulo
	evento.LocalID = dados.LocalID
	evento.Descricao = dados.Descricao
	evento.Categoria = dados.Categoria
	evento.CapaURL = dados.CapaURL
	evento.InicioEm = dados.InicioEm
	evento.FimEm = dados.FimEm
	evento.Timezone = valorOuPadrao(dados.Timezone, "America/Maceio")
	evento.ClassificacaoEtaria = dados.ClassificacaoEtaria
	evento.Visibilidade = valorOuPadraoVisibilidade(dados.Visibilidade)
	evento.TipoAcesso = valorOuPadraoTipoAcesso(dados.TipoAcesso)
	evento.ModoParticipantes = valorOuPadraoModoParticipantes(dados.ModoParticipantes)
	evento.InscricaoTalentosInicio = dados.InscricaoTalentosInicio
	evento.InscricaoTalentosFim = dados.InscricaoTalentosFim
	evento.CapacidadeTotal = dados.CapacidadeTotal
	evento.GarantiaHabilitada = dados.GarantiaHabilitada
	evento.PoliticaCancelamentoTexto = dados.PoliticaCancelamentoTexto
	evento.MaxItensPorPedido = valorIntOuPadrao(dados.MaxItensPorPedido, 10)

	if err := s.eventos.Salvar(evento); err != nil {
		return nil, err
	}
	return evento, nil
}

func (s *EventoService) BuscarDoOrganizador(organizadorID, eventoID int64) (*domain.Evento, error) {
	return s.buscarDoOrganizador(organizadorID, eventoID)
}

func (s *EventoService) Listar(organizadorID int64) ([]domain.Evento, error) {
	return s.eventos.ListarPorOrganizador(organizadorID)
}

// Publicar valida os requisitos da seção 6 do plano antes de virar o
// status para "publicado". Regras regionais (seção 7.8) ainda não são
// checadas aqui — a tabela regras_regionais entra no item 1.9.
func (s *EventoService) Publicar(organizadorID, eventoID int64) (*domain.Evento, []string, error) {
	evento, err := s.buscarDoOrganizador(organizadorID, eventoID)
	if err != nil {
		return nil, nil, err
	}
	if evento.Status != domain.StatusEventoRascunho {
		return nil, nil, ErrEventoNaoEhRascunho
	}

	var problemas []string

	if evento.InicioEm == nil || !evento.InicioEm.After(time.Now()) {
		problemas = append(problemas, "data de início precisa estar definida e no futuro")
	}
	if evento.FimEm != nil && evento.InicioEm != nil && evento.FimEm.Before(*evento.InicioEm) {
		problemas = append(problemas, "data de término não pode ser antes do início")
	}

	totalAtivos, err := s.tiposIngresso.ContarAtivosPorEvento(eventoID)
	if err != nil {
		return nil, nil, err
	}
	if totalAtivos == 0 {
		problemas = append(problemas, "é preciso ao menos um tipo de ingresso/cadastro ativo")
	}

	temValorPositivo, err := s.tiposIngresso.ExisteComPrecoMaiorQueZero(eventoID)
	if err != nil {
		return nil, nil, err
	}
	if temValorPositivo {
		organizador, err := s.organizadores.BuscarPorID(organizadorID)
		if err != nil {
			return nil, nil, err
		}
		if organizador.ChavePix == "" {
			problemas = append(problemas, "cadastre uma chave Pix no perfil do organizador antes de vender ingressos pagos")
		}
	}

	if len(problemas) > 0 {
		return nil, problemas, ErrPublicacaoInvalida
	}

	agora := time.Now()
	evento.Status = domain.StatusEventoPublicado
	evento.PublicadoEm = &agora
	if err := s.eventos.Salvar(evento); err != nil {
		return nil, nil, err
	}
	return evento, nil, nil
}

func (s *EventoService) buscarDoOrganizador(organizadorID, eventoID int64) (*domain.Evento, error) {
	evento, err := s.eventos.BuscarPorID(eventoID)
	if err != nil {
		return nil, err
	}
	if evento.OrganizadorID != organizadorID {
		return nil, ErrEventoNaoPertenceAoOrganizador
	}
	return evento, nil
}

func (s *EventoService) gerarSlugUnico(titulo string) (string, error) {
	base := slugify(titulo)
	slug := base
	for sufixo := 2; ; sufixo++ {
		existe, err := s.eventos.SlugExiste(slug)
		if err != nil {
			return "", err
		}
		if !existe {
			return slug, nil
		}
		slug = base + "-" + strconv.Itoa(sufixo)
	}
}

func valorOuPadrao(v, padrao string) string {
	if v == "" {
		return padrao
	}
	return v
}

func valorIntOuPadrao(v, padrao int) int {
	if v == 0 {
		return padrao
	}
	return v
}

func valorOuPadraoVisibilidade(v domain.Visibilidade) domain.Visibilidade {
	if v == "" {
		return domain.VisibilidadePublico
	}
	return v
}

func valorOuPadraoTipoAcesso(v domain.TipoAcesso) domain.TipoAcesso {
	if v == "" {
		return domain.TipoAcessoIngresso
	}
	return v
}

func valorOuPadraoModoParticipantes(v domain.ModoParticipantes) domain.ModoParticipantes {
	if v == "" {
		return domain.ModoParticipantesNenhum
	}
	return v
}
