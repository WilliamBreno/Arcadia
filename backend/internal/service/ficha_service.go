package service

import (
	"errors"
	"fmt"
	"gorm.io/gorm"
	"html"
	"strings"
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
	frontendURL string // item 8.3: link do QR de participante no e-mail
	// Item 8.4: inscrição paga (consulta do item pago e reembolso na remoção).
	db           *gorm.DB
	reembolsador ReembolsadorInscricao
	vagas        *VagasService
}

// DefinirVagas liga o limite de jurados e participantes (item 11.6).
func (s *FichaService) DefinirVagas(v *VagasService) { s.vagas = v }

// ReembolsadorInscricao devolve o que a pessoa pagou pela inscrição.
type ReembolsadorInscricao interface {
	ReembolsarInscricao(fichaID, solicitadoPor int64, motivo string) error
}

// DefinirInscricaoPaga liga a inscrição paga de talento (item 8.4).
func (s *FichaService) DefinirInscricaoPaga(db *gorm.DB, r ReembolsadorInscricao) {
	s.db, s.reembolsador = db, r
}

// inscricaoPaga: a ficha tem a inscrição paga e ainda valendo.
func (s *FichaService) inscricaoPaga(fichaID int64) bool {
	if s.db == nil {
		return false
	}
	var n int64
	s.db.Model(&domain.ItemPedido{}).Where("ficha_id = ? AND status IN ?", fichaID,
		[]domain.StatusItemPedido{domain.StatusItemPago, domain.StatusItemUtilizado}).Count(&n)
	return n > 0
}

// AvisarInscricaoPaga: pagamento confirmado = inscrição confirmada (8.4).
func (s *FichaService) AvisarInscricaoPaga(fichaID int64) {
	if f, err := s.fichas.BuscarPorID(fichaID); err == nil {
		s.enviarEmailStatusFicha(f)
	}
}

// AvisarPrazoVencido: a pessoa aprovada não pagou a tempo (8.4).
func (s *FichaService) AvisarPrazoVencido(fichaID int64) {
	f, err := s.fichas.BuscarPorID(fichaID)
	if err != nil {
		return
	}
	usuario, err := s.usuarios.BuscarPorID(f.UsuarioID)
	if err != nil || usuario.Email == "" {
		return
	}
	evento, err := s.eventos.BuscarPorID(f.EventoID)
	if err != nil {
		return
	}
	corpo := fmt.Sprintf(`<p>Olá, %s!</p><p>O prazo para pagar sua inscrição em <strong>%s</strong> terminou, e a vaga foi liberada. Se ainda quiser participar, fale com o organizador.</p>`,
		html.EscapeString(f.Nome), html.EscapeString(evento.Titulo))
	_ = s.mailCliente.Enviar(usuario.Email, "Prazo da inscrição encerrado — "+evento.Titulo, corpo)
}

// DefinirFrontendURL liga o link do QR de participante no e-mail de aprovação (8.3).
func (s *FichaService) DefinirFrontendURL(u string) { s.frontendURL = strings.TrimRight(u, "/") }

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
		corpo = fmt.Sprintf(`<p>Olá, %s!</p><p>Sua inscrição como %s em <strong>%s</strong> foi aprovada.</p>`, html.EscapeString(ficha.Nome), ficha.Papel, html.EscapeString(evento.Titulo))
		if ficha.Papel == domain.PapelParticipante && s.frontendURL != "" {
			// Item 8.3: o QR do camarim fica na página do evento, para a pessoa logada.
			corpo += fmt.Sprintf(`<p>No dia, mostre seu <strong>QR de participante</strong> na entrada do camarim. Ele fica na página do evento, com você conectado: <a href="%s/e/%s#meu-qr">abrir meu QR</a>.</p>`, s.frontendURL, evento.Slug)
		}
	case domain.StatusFichaAguardandoPagamento:
		assunto = "Inscrição aprovada: falta pagar — " + evento.Titulo
		prazo := ""
		if ficha.PrazoPagamentoEm != nil {
			prazo = " até " + ficha.PrazoPagamentoEm.In(FusoDoEvento(evento)).Format("02/01/2006 às 15:04")
		}
		corpo = fmt.Sprintf(`<p>Olá, %s!</p><p>Sua inscrição em <strong>%s</strong> foi aprovada. Para confirmar a vaga, pague a inscrição de <strong>%s</strong> (mais a taxa de serviço)%s. Sem o pagamento, a vaga é liberada.</p>`,
			html.EscapeString(ficha.Nome), html.EscapeString(evento.Titulo), formatarReais(evento.InscricaoTalentoCentavos), prazo)
		if s.frontendURL != "" {
			corpo += fmt.Sprintf(`<p><a href="%s/e/%s/inscricao">Pagar a inscrição</a></p>`, s.frontendURL, evento.Slug)
		}
	case domain.StatusFichaRejeitado:
		assunto = "Inscrição não aprovada — " + evento.Titulo
		motivo := ""
		if ficha.MotivoRejeicao != "" {
			motivo = fmt.Sprintf("<p>Motivo: %s</p>", html.EscapeString(ficha.MotivoRejeicao))
		}
		corpo = fmt.Sprintf(`<p>Olá, %s!</p><p>Sua inscrição como %s em <strong>%s</strong> não foi aprovada.</p>%s`, html.EscapeString(ficha.Nome), ficha.Papel, html.EscapeString(evento.Titulo), motivo)
	default:
		return
	}
	// Item 8.5: classificado na seletiva.
	if ficha.ClassificadaPorEventoID != nil && ficha.Status != domain.StatusFichaRejeitado {
		if sel, err := s.eventos.BuscarPorID(*ficha.ClassificadaPorEventoID); err == nil {
			corpo = fmt.Sprintf(`<p>Parabéns! Você se classificou em <strong>%s</strong> e entrou direto na final, sem passar pela análise.</p>`, html.EscapeString(sel.Titulo)) + corpo
		}
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

	tipoAntes := ficha.TipoApresentacao
	aplicarDadosFicha(ficha, dados)
	if err := validarMenorDeIdade(ficha); err != nil {
		return nil, err
	}
	troca := s.vagas != nil && ficha.Papel == domain.PapelParticipante && !mesmoTipo(tipoAntes, ficha.TipoApresentacao)
	if !troca {
		if err := s.fichas.SalvarDadosEditaveis(ficha); err != nil {
			return nil, err
		}
		return s.fichas.BuscarPorID(ficha.ID)
	}
	// Item 11.6: trocar de modalidade só se a nova ainda tiver vaga, com a
	// conferência e a gravação sob a mesma trava.
	err = s.vagas.DB().Transaction(func(tx *gorm.DB) error {
		if err := TravarVagas(tx, eventoID); err != nil {
			return err
		}
		fichas := repository.NovoFichaParticipacaoRepository(tx)
		atual, err := fichas.BuscarPorID(ficha.ID)
		if err != nil {
			return err
		}
		if OcupaVaga(atual.Status) {
			if err := s.vagas.ConferirTroca(tx, eventoID, ficha.TipoApresentacao); err != nil {
				return err
			}
		}
		return fichas.SalvarDadosEditaveis(ficha)
	})
	if err != nil {
		return nil, err
	}
	return s.fichas.BuscarPorID(ficha.ID)
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

	if s.vagas == nil {
		if _, err := s.fichas.Buscar(eventoID, usuarioID, domain.PapelParticipante); err == nil {
			return nil, ErrJaInscrito
		}
		if err := s.fichas.Criar(ficha); err != nil {
			return nil, err
		}
		return ficha, nil
	}
	// Item 11.6: ordem de chegada. Com a vaga preenchida, entra na lista de
	// espera (se o organizador ligou) ou a inscrição é recusada.
	err = s.vagas.DB().Transaction(func(tx *gorm.DB) error {
		if err := TravarVagas(tx, eventoID); err != nil {
			return err
		}
		fichas := repository.NovoFichaParticipacaoRepository(tx)
		if _, err := fichas.Buscar(eventoID, usuarioID, domain.PapelParticipante); err == nil {
			return ErrJaInscrito
		}
		if err := s.vagas.Conferir(tx, eventoID, domain.PapelParticipante, ficha.TipoApresentacao); err != nil {
			if !EhErroVagas(err) || !s.vagas.obter(tx, eventoID).ListaEspera {
				return err
			}
			ficha.Status = domain.StatusFichaListaEspera
		}
		return fichas.Criar(ficha)
	})
	if err != nil {
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
	if s.vagas != nil {
		if err := s.reservarVagaNaAprovacao(eventoID, ficha.ID); err != nil {
			return nil, err
		}
		if ficha, err = s.fichas.BuscarPorID(ficha.ID); err != nil {
			return nil, err
		}
	}
	return s.aprovar(ficha, eventoID)
}

// reservarVagaNaAprovacao (item 11.6): quem estava fora das vagas (lista de
// espera, rejeitado, desistiu; jurado removido) só entra se ainda houver
// vaga. A vaga é reservada sob a trava (ficha volta a "pendente", papel de
// jurado a "pendente") antes de aprovar, então duas aprovações ao mesmo
// tempo não ficam com a mesma última vaga.
func (s *FichaService) reservarVagaNaAprovacao(eventoID, fichaID int64) error {
	return s.vagas.DB().Transaction(func(tx *gorm.DB) error {
		if err := TravarVagas(tx, eventoID); err != nil {
			return err
		}
		atual, err := repository.NovoFichaParticipacaoRepository(tx).BuscarPorID(fichaID)
		if err != nil {
			return err
		}
		if atual.Papel == domain.PapelJurado {
			papeis := repository.NovoPapelEventoRepository(tx)
			papel, err := papeis.Buscar(eventoID, atual.UsuarioID, domain.PapelJurado)
			if err == nil && papel.Status != domain.StatusPapelRemovido {
				return nil
			}
			if err := s.vagas.Conferir(tx, eventoID, domain.PapelJurado, nil); err != nil {
				return err
			}
			if err != nil {
				return papeis.Criar(&domain.PapelEvento{EventoID: eventoID, UsuarioID: atual.UsuarioID, Papel: domain.PapelJurado,
					Origem: atual.Origem, Status: domain.StatusPapelPendente, CriadoEm: time.Now()})
			}
			papel.Status = domain.StatusPapelPendente
			return papeis.Salvar(papel)
		}
		if OcupaVaga(atual.Status) {
			return nil
		}
		if err := s.vagas.Conferir(tx, eventoID, atual.Papel, atual.TipoApresentacao); err != nil {
			return err
		}
		return tx.Model(&domain.FichaParticipacao{}).Where("id = ?", atual.ID).Update("status", domain.StatusFichaPendente).Error
	})
}

// AprovarPeloSistema aprova sem o organizador (item 8.5: classificado na
// seletiva entra na final já aprovado; paga a inscrição, se houver).
func (s *FichaService) AprovarPeloSistema(fichaID int64) (*domain.FichaParticipacao, error) {
	ficha, err := s.fichas.BuscarPorID(fichaID)
	if err != nil {
		return nil, ErrFichaNaoEncontrada
	}
	return s.aprovar(ficha, ficha.EventoID)
}

func (s *FichaService) aprovar(ficha *domain.FichaParticipacao, eventoID int64) (*domain.FichaParticipacao, error) {

	// Item 8.4: inscrição paga. Aprovada, a pessoa tem 48 h para pagar; a
	// vaga (e o papel de participante) só é confirmada com o pagamento.
	// Convidado pelo organizador não paga.
	if ficha.Papel == domain.PapelParticipante && ficha.Origem == domain.OrigemInscricao && !s.inscricaoPaga(ficha.ID) {
		if evento, err := s.eventos.BuscarPorID(eventoID); err == nil && evento.InscricaoTalentoCentavos > 0 {
			prazo := PrazoParaPagar(time.Now(), evento.InicioEm)
			ficha.Status = domain.StatusFichaAguardandoPagamento
			ficha.PrazoPagamentoEm = &prazo
			ficha.MotivoRejeicao = ""
			if err := s.fichas.Salvar(ficha); err != nil {
				return nil, err
			}
			s.enviarEmailStatusFicha(ficha)
			return ficha, nil
		}
	}

	ficha.Status = domain.StatusFichaAprovado
	ficha.MotivoRejeicao = ""
	ficha.PrazoPagamentoEm = nil
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

	// Item 8.4: removeu quem já pagou a inscrição = reembolso integral.
	// Sem o estorno confirmado, a ficha não muda.
	if s.reembolsador != nil && s.inscricaoPaga(ficha.ID) {
		if err := s.reembolsador.ReembolsarInscricao(ficha.ID, organizadorID, "participação removida pelo organizador"); err != nil {
			return nil, err
		}
	}

	ficha.Status = domain.StatusFichaRejeitado
	ficha.MotivoRejeicao = motivo
	ficha.PrazoPagamentoEm = nil
	if err := s.fichas.Salvar(ficha); err != nil {
		return nil, err
	}
	if papel, err := s.papeis.Buscar(eventoID, ficha.UsuarioID, ficha.Papel); err == nil && papel.Status != domain.StatusPapelRemovido {
		papel.Status = domain.StatusPapelRemovido
		_ = s.papeis.Salvar(papel)
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
