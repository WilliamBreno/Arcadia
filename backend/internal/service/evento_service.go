package service

import (
	"errors"
	"fmt"
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
	eventos         *repository.EventoRepository
	tiposIngresso   *repository.TipoIngressoRepository
	organizadores   *repository.OrganizadorRepository
	locais          *repository.LocalRepository
	regrasRegionais *repository.RegraRegionalRepository
	config          *repository.ConfigPlataformaRepository
}

func NovoEventoService(
	eventos *repository.EventoRepository,
	tiposIngresso *repository.TipoIngressoRepository,
	organizadores *repository.OrganizadorRepository,
	locais *repository.LocalRepository,
	regrasRegionais *repository.RegraRegionalRepository,
	config *repository.ConfigPlataformaRepository,
) *EventoService {
	return &EventoService{
		eventos: eventos, tiposIngresso: tiposIngresso, organizadores: organizadores,
		locais: locais, regrasRegionais: regrasRegionais, config: config,
	}
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
	GarantiaHabilitada        *bool // nil = padrão (true) ao criar, manter ao atualizar
	AprovacaoManual           *bool // nil = padrão (false) ao criar, manter ao atualizar
	QRRotativo                *bool // idem
	CorTema                   *string
	TaxaPagaPor               *string // nil = padrão (comprador) ao criar, manter ao atualizar
	PoliticaCancelamentoTexto string
	MaxItensPorPedido         int
	MaxIngressosPorConta      *int // nil = padrão (10) ao criar, manter ao atualizar; 0 = sem limite
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
		GarantiaHabilitada:        boolOuPadrao(dados.GarantiaHabilitada, true),
		AprovacaoManual:           boolOuPadrao(dados.AprovacaoManual, false),
		QRRotativo:                boolOuPadrao(dados.QRRotativo, false),
		CorTema:                   corTemaOuPadrao(dados.CorTema, ""),
		TaxaPagaPor:               taxaPagaPorOuPadrao(dados.TaxaPagaPor, domain.TaxaPagaPorComprador),
		PoliticaCancelamentoTexto: dados.PoliticaCancelamentoTexto,
		MaxItensPorPedido:         valorIntOuPadrao(dados.MaxItensPorPedido, 10),
		MaxIngressosPorConta:      limiteContaOuPadrao(dados.MaxIngressosPorConta, 10),
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
	evento.GarantiaHabilitada = boolOuPadrao(dados.GarantiaHabilitada, evento.GarantiaHabilitada)
	evento.AprovacaoManual = boolOuPadrao(dados.AprovacaoManual, evento.AprovacaoManual)
	evento.QRRotativo = boolOuPadrao(dados.QRRotativo, evento.QRRotativo)
	evento.CorTema = corTemaOuPadrao(dados.CorTema, evento.CorTema)
	evento.TaxaPagaPor = taxaPagaPorOuPadrao(dados.TaxaPagaPor, evento.TaxaPagaPor)
	evento.PoliticaCancelamentoTexto = dados.PoliticaCancelamentoTexto
	evento.MaxItensPorPedido = valorIntOuPadrao(dados.MaxItensPorPedido, 10)
	evento.MaxIngressosPorConta = limiteContaOuPadrao(dados.MaxIngressosPorConta, evento.MaxIngressosPorConta)

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

	tiposDoEvento, err := s.tiposIngresso.ListarPorEvento(eventoID)
	if err != nil {
		return nil, nil, err
	}
	if err := validarCotaMeia(tiposDoEvento); err != nil {
		problemas = append(problemas, err.Error())
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

		problemasRegionais, err := s.checarRegraRegional(evento)
		if err != nil {
			return nil, nil, err
		}
		problemas = append(problemas, problemasRegionais...)
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

// checarRegraRegional aplica o levantamento da seção 7.8 (a validar com
// advogado). Desde o item 5.1: o limite percentual (ex.: AL, 10%) não
// bloqueia mais — a taxa é reduzida automaticamente no cálculo. Só
// bloqueia UF/cidade que proíbe a taxa ou exige canal sem taxa, e só
// quando o comprador paga a taxa e há ingresso pago (gratuito não tem
// taxa). Se o organizador absorver a taxa, o comprador não paga taxa de
// conveniência e não há bloqueio.
func (s *EventoService) checarRegraRegional(evento *domain.Evento) ([]string, error) {
	if evento.TaxaPagaPor == domain.TaxaPagaPorOrganizador {
		return nil, nil
	}
	regra := NovoTaxasService(s.config, s.locais, s.regrasRegionais).RegraRegionalDoEvento(evento)
	if regra == nil || (regra.PermiteTaxa && !regra.ExigeCanalSemTaxa) {
		return nil, nil
	}

	tipos, err := s.tiposIngresso.ListarAtivosPublicoPorEvento(evento.ID)
	if err != nil {
		return nil, err
	}
	temPago := false
	for _, tipo := range tipos {
		if tipo.PrecoCentavos > 0 {
			temPago = true
			break
		}
	}
	if !temPago {
		return nil, nil
	}

	lugar := regra.UF
	if regra.Municipio != nil {
		lugar = *regra.Municipio + "/" + regra.UF
	}
	return []string{fmt.Sprintf(
		"publicação bloqueada: %s tem restrição à taxa de conveniência (%s). Para publicar, escolha que o organizador paga a taxa (opção \"Quem paga a taxa\") ou peça liberação ao admin da plataforma.",
		lugar, regra.Observacao,
	)}, nil
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

func boolOuPadrao(v *bool, padrao bool) bool {
	if v == nil {
		return padrao
	}
	return *v
}

// taxaPagaPorOuPadrao aceita só os dois valores conhecidos; qualquer outro
// mantém o atual. A mudança vale para vendas futuras: cada item já vendido
// guarda se a taxa dele foi absorvida.
func taxaPagaPorOuPadrao(v *string, atual string) string {
	if v != nil && (*v == domain.TaxaPagaPorComprador || *v == domain.TaxaPagaPorOrganizador) {
		return *v
	}
	if atual == "" {
		return domain.TaxaPagaPorComprador
	}
	return atual
}

// limiteContaOuPadrao: limite de ingressos por conta (item 5.3). Aceita 0
// (sem limite) até 1000; fora disso mantém o atual.
func limiteContaOuPadrao(v *int, atual int) int {
	if v != nil && *v >= 0 && *v <= 1000 {
		return *v
	}
	return atual
}
