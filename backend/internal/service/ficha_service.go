package service

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/datatypes"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/WilliamBreno/Arcadia/backend/internal/mail"
	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
)

var (
	ErrFichaNaoEncontrada     = errors.New("ficha não encontrada")
	ErrMenorSemAutorizacao    = errors.New("menor de 18 anos precisa de nome, contato e autorização do responsável")
	ErrInscricaoNaoPermitida  = errors.New("este evento não aceita inscrição aberta de participantes")
	ErrInscricaoForaDoPrazo   = errors.New("fora do prazo de inscrição")
	ErrJaInscrito             = errors.New("você já se inscreveu neste evento")
	ErrNaoEhJuradoConfirmado  = errors.New("você não é jurado confirmado deste evento")
	ErrFichaNaoPertenceEvento = errors.New("ficha não pertence a este evento")
)

type FichaDados struct {
	Nome                      string
	NomeArtistico             string
	Instagram                 string
	DataNascimento            *time.Time
	FotoURL                   string
	Telefone                  string
	TipoApresentacao          *domain.TipoApresentacao
	Dados                     datatypes.JSON
	ResponsavelNome           string
	ResponsavelContato        string
	AutorizacaoResponsavelURL string
}

func aplicarDadosFicha(f *domain.FichaParticipacao, dados FichaDados) {
	f.Nome = dados.Nome
	f.NomeArtistico = dados.NomeArtistico
	f.Instagram = dados.Instagram
	f.DataNascimento = dados.DataNascimento
	f.FotoURL = dados.FotoURL
	f.Telefone = dados.Telefone
	f.TipoApresentacao = dados.TipoApresentacao
	if len(dados.Dados) == 0 {
		f.Dados = datatypes.JSON("{}")
	} else {
		f.Dados = dados.Dados
	}
	f.ResponsavelNome = dados.ResponsavelNome
	f.ResponsavelContato = dados.ResponsavelContato
	f.AutorizacaoResponsavelURL = dados.AutorizacaoResponsavelURL
}

// validarMenorDeIdade aplica a regra da seção 7.12: menor de 18 (por
// data_nascimento) precisa de responsável e autorização.
func validarMenorDeIdade(f *domain.FichaParticipacao) error {
	if !f.MenorDeIdade() {
		return nil
	}
	if f.ResponsavelNome == "" || f.ResponsavelContato == "" || f.AutorizacaoResponsavelURL == "" {
		return ErrMenorSemAutorizacao
	}
	return nil
}

type FichaService struct {
	fichas      *repository.FichaParticipacaoRepository
	papeis      *repository.PapelEventoRepository
	eventos     *repository.EventoRepository
	usuarios    *repository.UsuarioRepository
	mailCliente *mail.Cliente
}

func NovoFichaService(
	fichas *repository.FichaParticipacaoRepository,
	papeis *repository.PapelEventoRepository,
	eventos *repository.EventoRepository,
	usuarios *repository.UsuarioRepository,
	mailCliente *mail.Cliente,
) *FichaService {
	return &FichaService{fichas: fichas, papeis: papeis, eventos: eventos, usuarios: usuarios, mailCliente: mailCliente}
}

// enviarEmailStatusFicha é o e-mail "a cada mudança de status" da seção
// 7.12 — best-effort (não falha a aprovação/rejeição se o e-mail não sair).
func (s *FichaService) enviarEmailStatusFicha(ficha *domain.FichaParticipacao) {
	usuario, err := s.usuarios.BuscarPorID(ficha.UsuarioID)
	if err != nil || usuario.Email == "" {
		return
	}
	evento, err := s.eventos.BuscarPorID(ficha.EventoID)
	if err != nil {
		return
	}

	var assunto, corpo string
	switch ficha.Status {
	case domain.StatusFichaAprovado:
		assunto = "Inscrição aprovada — " + evento.Titulo
		corpo = fmt.Sprintf(`<p>Olá, %s!</p><p>Sua inscrição como %s em <strong>%s</strong> foi aprovada.</p>`, ficha.Nome, ficha.Papel, evento.Titulo)
	case domain.StatusFichaRejeitado:
		assunto = "Inscrição não aprovada — " + evento.Titulo
		motivo := ""
		if ficha.MotivoRejeicao != "" {
			motivo = fmt.Sprintf("<p>Motivo: %s</p>", ficha.MotivoRejeicao)
		}
		corpo = fmt.Sprintf(`<p>Olá, %s!</p><p>Sua inscrição como %s em <strong>%s</strong> não foi aprovada.</p>%s`, ficha.Nome, ficha.Papel, evento.Titulo, motivo)
	default:
		return
	}
	_ = s.mailCliente.Enviar(usuario.Email, assunto, corpo)
}

func (s *FichaService) ObterMinha(eventoID, usuarioID int64, papel domain.Papel) (*domain.FichaParticipacao, error) {
	ficha, err := s.fichas.Buscar(eventoID, usuarioID, papel)
	if err != nil {
		return nil, ErrFichaNaoEncontrada
	}
	return ficha, nil
}

// AtualizarMinha só é permitida enquanto a ficha ainda pode ser editada
// (rascunho/pendente/aprovado) — depois de rejeitada/desistiu o
// organizador decide o próximo passo, não o próprio usuário editando
// livremente.
func (s *FichaService) AtualizarMinha(eventoID, usuarioID int64, papel domain.Papel, dados FichaDados) (*domain.FichaParticipacao, error) {
	ficha, err := s.fichas.Buscar(eventoID, usuarioID, papel)
	if err != nil {
		return nil, ErrFichaNaoEncontrada
	}

	aplicarDadosFicha(ficha, dados)
	if err := validarMenorDeIdade(ficha); err != nil {
		return nil, err
	}

	if err := s.fichas.Salvar(ficha); err != nil {
		return nil, err
	}
	return ficha, nil
}

// Inscrever é o fluxo de inscrição aberta (seção 2.4/7.12): o usuário se
// inscreve como participante, a ficha nasce "pendente" e só vira
// "aprovado" quando o organizador aprovar. Vagas por tipo/categoria não
// são checadas ainda (ver seção 14) — só prazo e modo_participantes.
func (s *FichaService) Inscrever(eventoID, usuarioID int64, dados FichaDados) (*domain.FichaParticipacao, error) {
	evento, err := s.eventos.BuscarPorID(eventoID)
	if err != nil {
		return nil, ErrFichaNaoEncontrada
	}

	if evento.ModoParticipantes != domain.ModoParticipantesInscricaoAberta && evento.ModoParticipantes != domain.ModoParticipantesAmbos {
		return nil, ErrInscricaoNaoPermitida
	}

	agora := time.Now()
	if evento.InscricaoTalentosInicio != nil && agora.Before(*evento.InscricaoTalentosInicio) {
		return nil, ErrInscricaoForaDoPrazo
	}
	if evento.InscricaoTalentosFim != nil && agora.After(*evento.InscricaoTalentosFim) {
		return nil, ErrInscricaoForaDoPrazo
	}

	if _, err := s.fichas.Buscar(eventoID, usuarioID, domain.PapelParticipante); err == nil {
		return nil, ErrJaInscrito
	}

	ficha := &domain.FichaParticipacao{
		EventoID:  eventoID,
		UsuarioID: usuarioID,
		Papel:     domain.PapelParticipante,
		Origem:    domain.OrigemInscricao,
		Status:    domain.StatusFichaPendente,
		CriadoEm:  agora,
	}
	aplicarDadosFicha(ficha, dados)
	if err := validarMenorDeIdade(ficha); err != nil {
		return nil, err
	}

	if err := s.fichas.Criar(ficha); err != nil {
		return nil, err
	}
	return ficha, nil
}

// Aprovar confirma a ficha e cria/confirma o papel_evento correspondente
// — só a partir daqui a pessoa aparece pro jurado e em "meus eventos".
func (s *FichaService) Aprovar(organizadorID, eventoID, fichaID int64) (*domain.FichaParticipacao, error) {
	ficha, err := s.buscarDoOrganizador(organizadorID, eventoID, fichaID)
	if err != nil {
		return nil, err
	}

	ficha.Status = domain.StatusFichaAprovado
	ficha.MotivoRejeicao = ""
	if err := s.fichas.Salvar(ficha); err != nil {
		return nil, err
	}

	papel, err := s.papeis.Buscar(eventoID, ficha.UsuarioID, ficha.Papel)
	if err != nil {
		papel = &domain.PapelEvento{
			EventoID:  eventoID,
			UsuarioID: ficha.UsuarioID,
			Papel:     ficha.Papel,
			Origem:    ficha.Origem,
			Status:    domain.StatusPapelConfirmado,
			CriadoEm:  time.Now(),
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

	s.enviarEmailStatusFicha(ficha)
	return ficha, nil
}

func (s *FichaService) Rejeitar(organizadorID, eventoID, fichaID int64, motivo string) (*domain.FichaParticipacao, error) {
	ficha, err := s.buscarDoOrganizador(organizadorID, eventoID, fichaID)
	if err != nil {
		return nil, err
	}

	ficha.Status = domain.StatusFichaRejeitado
	ficha.MotivoRejeicao = motivo
	if err := s.fichas.Salvar(ficha); err != nil {
		return nil, err
	}
	s.enviarEmailStatusFicha(ficha)
	return ficha, nil
}

// ListarDoOrganizador mostra todas as fichas (qualquer status) — usado
// no painel de aprovação.
func (s *FichaService) ListarDoOrganizador(organizadorID, eventoID int64) ([]domain.FichaParticipacao, error) {
	evento, err := s.eventos.BuscarPorID(eventoID)
	if err != nil {
		return nil, ErrFichaNaoEncontrada
	}
	if evento.OrganizadorID != organizadorID {
		return nil, ErrEventoNaoPertenceAoOrganizador
	}
	return s.fichas.ListarPorEvento(eventoID, nil)
}

// ListarParaJurado só mostra fichas aprovadas — e só se quem pede for
// jurado confirmado deste evento (seção 3: privacidade). A filtragem de
// campos privados (telefone, contato do responsável) acontece na
// camada de resposta do handler, não aqui.
func (s *FichaService) ListarParaJurado(usuarioID, eventoID int64) ([]domain.FichaParticipacao, error) {
	papel, err := s.papeis.Buscar(eventoID, usuarioID, domain.PapelJurado)
	if err != nil || papel.Status != domain.StatusPapelConfirmado {
		return nil, ErrNaoEhJuradoConfirmado
	}
	aprovado := domain.StatusFichaAprovado
	return s.fichas.ListarPorEventoEPapel(eventoID, domain.PapelParticipante, &aprovado)
}

func (s *FichaService) ObterParaJurado(usuarioID, eventoID, fichaID int64) (*domain.FichaParticipacao, error) {
	papel, err := s.papeis.Buscar(eventoID, usuarioID, domain.PapelJurado)
	if err != nil || papel.Status != domain.StatusPapelConfirmado {
		return nil, ErrNaoEhJuradoConfirmado
	}
	ficha, err := s.fichas.BuscarPorID(fichaID)
	if err != nil || ficha.EventoID != eventoID || ficha.Status != domain.StatusFichaAprovado || ficha.Papel != domain.PapelParticipante {
		return nil, ErrFichaNaoEncontrada
	}
	return ficha, nil
}

func (s *FichaService) buscarDoOrganizador(organizadorID, eventoID, fichaID int64) (*domain.FichaParticipacao, error) {
	evento, err := s.eventos.BuscarPorID(eventoID)
	if err != nil {
		return nil, ErrFichaNaoEncontrada
	}
	if evento.OrganizadorID != organizadorID {
		return nil, ErrEventoNaoPertenceAoOrganizador
	}
	ficha, err := s.fichas.BuscarPorID(fichaID)
	if err != nil {
		return nil, ErrFichaNaoEncontrada
	}
	if ficha.EventoID != eventoID {
		return nil, ErrFichaNaoPertenceEvento
	}
	return ficha, nil
}

// DefinirOrdem grava a ordem de apresentação (item 3.4): só participantes
// aprovados do evento, sem repetição; quem não estiver na lista fica sem ordem.
func (s *FichaService) DefinirOrdem(organizadorID, eventoID int64, fichaIDs []int64) error {
	evento, err := s.eventos.BuscarPorID(eventoID)
	if err != nil {
		return ErrFichaNaoEncontrada
	}
	if evento.OrganizadorID != organizadorID {
		return ErrEventoNaoPertenceAoOrganizador
	}
	vistos := map[int64]bool{}
	for _, id := range fichaIDs {
		if vistos[id] {
			return fmt.Errorf("ficha %d repetida na ordem", id)
		}
		vistos[id] = true
		f, err := s.fichas.BuscarPorID(id)
		if err != nil || f.EventoID != eventoID || f.Papel != domain.PapelParticipante || f.Status != domain.StatusFichaAprovado {
			return ErrFichaNaoEncontrada
		}
	}
	return s.fichas.DefinirOrdemApresentacao(eventoID, fichaIDs)
}
