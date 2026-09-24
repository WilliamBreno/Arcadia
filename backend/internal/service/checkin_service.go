package service

import (
	"crypto/subtle"
	"errors"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
)

var ErrSemAcessoCheckin = errors.New("você não tem acesso ao check-in deste evento")

type ResultadoValidacao string

const (
	ResultadoValido        ResultadoValidacao = "valido"
	ResultadoJaUtilizado   ResultadoValidacao = "ja_utilizado"
	ResultadoCancelado     ResultadoValidacao = "cancelado"
	ResultadoOutroEvento   ResultadoValidacao = "outro_evento"
	ResultadoNaoEncontrado ResultadoValidacao = "nao_encontrado"
)

type ResultadoCheckin struct {
	Resultado        ResultadoValidacao
	Item             *domain.ItemPedido
	TipoIngressoNome string
	MeiaEntrada      bool
}

type CheckinService struct {
	itensPedido   *repository.ItemPedidoRepository
	tiposIngresso *repository.TipoIngressoRepository
	eventos       *repository.EventoRepository
	organizadores *repository.OrganizadorRepository
	papeis        *repository.PapelEventoRepository
}

func NovoCheckinService(
	itensPedido *repository.ItemPedidoRepository,
	tiposIngresso *repository.TipoIngressoRepository,
	eventos *repository.EventoRepository,
	organizadores *repository.OrganizadorRepository,
	papeis *repository.PapelEventoRepository,
) *CheckinService {
	return &CheckinService{
		itensPedido: itensPedido, tiposIngresso: tiposIngresso, eventos: eventos,
		organizadores: organizadores, papeis: papeis,
	}
}

// TemAcesso é "operado por staff ou organizador" (seção 7.10): dono do
// evento OU papel_evento(staff, confirmado) para esse evento específico.
func (s *CheckinService) TemAcesso(usuarioID, eventoID int64) bool {
	evento, err := s.eventos.BuscarPorID(eventoID)
	if err != nil {
		return false
	}
	if organizador, err := s.organizadores.BuscarPorUsuarioID(usuarioID); err == nil && organizador.ID == evento.OrganizadorID {
		return true
	}
	papel, err := s.papeis.Buscar(eventoID, usuarioID, domain.PapelStaff)
	return err == nil && papel.Status == domain.StatusPapelConfirmado
}

// Validar é o coração do leitor de QR (seção 7.10). codigo+qrToken vêm
// do que a câmera leu; a checagem HMAC (tempo constante) impede alguém
// de simplesmente digitar um código curto sem ter o QR de verdade.
func (s *CheckinService) Validar(usuarioID, eventoID int64, codigo, qrToken string) (*ResultadoCheckin, error) {
	if !s.TemAcesso(usuarioID, eventoID) {
		return nil, ErrSemAcessoCheckin
	}

	item, err := s.itensPedido.BuscarPorCodigo(codigo)
	if err != nil {
		return &ResultadoCheckin{Resultado: ResultadoNaoEncontrado}, nil
	}
	if subtle.ConstantTimeCompare([]byte(item.QRToken), []byte(qrToken)) != 1 {
		return &ResultadoCheckin{Resultado: ResultadoNaoEncontrado}, nil
	}

	tipo, err := s.tiposIngresso.BuscarPorID(item.TipoIngressoID)
	if err != nil {
		return &ResultadoCheckin{Resultado: ResultadoNaoEncontrado}, nil
	}
	if tipo.EventoID != eventoID {
		return &ResultadoCheckin{Resultado: ResultadoOutroEvento, Item: item, TipoIngressoNome: tipo.Nome, MeiaEntrada: tipo.MeiaEntrada}, nil
	}

	switch item.Status {
	case domain.StatusItemUtilizado:
		return &ResultadoCheckin{Resultado: ResultadoJaUtilizado, Item: item, TipoIngressoNome: tipo.Nome, MeiaEntrada: tipo.MeiaEntrada}, nil
	case domain.StatusItemCancelado, domain.StatusItemReembolsado, domain.StatusItemExpirado, domain.StatusItemReservado:
		return &ResultadoCheckin{Resultado: ResultadoCancelado, Item: item, TipoIngressoNome: tipo.Nome, MeiaEntrada: tipo.MeiaEntrada}, nil
	}

	marcou, err := s.itensPedido.MarcarUtilizadoAtomico(item.ID)
	if err != nil {
		return nil, err
	}
	if !marcou {
		// alguém fez o check-in um instante antes — não é erro, é corrida.
		atualizado, err := s.itensPedido.BuscarPorID(item.ID)
		if err != nil {
			return nil, err
		}
		return &ResultadoCheckin{Resultado: ResultadoJaUtilizado, Item: atualizado, TipoIngressoNome: tipo.Nome, MeiaEntrada: tipo.MeiaEntrada}, nil
	}

	atualizado, err := s.itensPedido.BuscarPorID(item.ID)
	if err != nil {
		return nil, err
	}
	return &ResultadoCheckin{Resultado: ResultadoValido, Item: atualizado, TipoIngressoNome: tipo.Nome, MeiaEntrada: tipo.MeiaEntrada}, nil
}

func (s *CheckinService) Buscar(usuarioID, eventoID int64, texto string) ([]repository.ItemComTipo, error) {
	if !s.TemAcesso(usuarioID, eventoID) {
		return nil, ErrSemAcessoCheckin
	}
	return s.itensPedido.BuscarPorEventoEBusca(eventoID, texto)
}

func (s *CheckinService) Resumo(usuarioID, eventoID int64) (repository.ResumoCheckin, error) {
	if !s.TemAcesso(usuarioID, eventoID) {
		return repository.ResumoCheckin{}, ErrSemAcessoCheckin
	}
	return s.itensPedido.ResumoCheckin(eventoID)
}
