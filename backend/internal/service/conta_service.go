package service

import (
	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
)

type ContaService struct {
	organizadores *repository.OrganizadorRepository
	eventos       *repository.EventoRepository
	papeis        *repository.PapelEventoRepository
	itensPedido   *repository.ItemPedidoRepository
	pedidos       *repository.PedidoRepository
}

func NovoContaService(
	organizadores *repository.OrganizadorRepository,
	eventos *repository.EventoRepository,
	papeis *repository.PapelEventoRepository,
	itensPedido *repository.ItemPedidoRepository,
	pedidos *repository.PedidoRepository,
) *ContaService {
	return &ContaService{organizadores: organizadores, eventos: eventos, papeis: papeis, itensPedido: itensPedido, pedidos: pedidos}
}

// Selo é o "crachá" que aparece em Meus eventos (seção 3 do plano):
// Organizador, Jurado, Participante, Participante especial ou Ingresso.
// Uma pessoa pode ter mais de um selo no mesmo evento (ex.: comprou
// ingresso E é jurada).
type Selo string

const (
	SeloOrganizador          Selo = "organizador"
	SeloJurado               Selo = "jurado"
	SeloParticipante         Selo = "participante"
	SeloParticipanteEspecial Selo = "participante_especial"
	SeloStaff                Selo = "staff"
	SeloIngresso             Selo = "ingresso"
)

type MeuEvento struct {
	Evento domain.Evento
	Selos  []Selo
}

// MeusEventos junta três fontes (seção 3): perfil de organizador (dono
// do evento), papeis_evento confirmados (jurado/participante/staff) e
// itens_pedido pagos (comprador). Um mesmo evento pode aparecer com
// vários selos.
func (s *ContaService) MeusEventos(usuarioID int64) ([]MeuEvento, error) {
	selosPorEvento := map[int64]map[Selo]bool{}
	adicionar := func(eventoID int64, selo Selo) {
		if selosPorEvento[eventoID] == nil {
			selosPorEvento[eventoID] = map[Selo]bool{}
		}
		selosPorEvento[eventoID][selo] = true
	}

	if organizador, err := s.organizadores.BuscarPorUsuarioID(usuarioID); err == nil {
		eventosOrganizados, err := s.eventos.ListarPorOrganizador(organizador.ID)
		if err != nil {
			return nil, err
		}
		for _, e := range eventosOrganizados {
			adicionar(e.ID, SeloOrganizador)
		}
	}

	papeis, err := s.papeis.ListarPorUsuario(usuarioID)
	if err != nil {
		return nil, err
	}
	for _, p := range papeis {
		switch p.Papel {
		case domain.PapelJurado:
			adicionar(p.EventoID, SeloJurado)
		case domain.PapelParticipante:
			if p.Origem == domain.OrigemConvite {
				adicionar(p.EventoID, SeloParticipanteEspecial)
			} else {
				adicionar(p.EventoID, SeloParticipante)
			}
		case domain.PapelStaff:
			adicionar(p.EventoID, SeloStaff)
		}
	}

	eventoIDsComIngresso, err := s.itensPedido.EventosComIngressoPago(usuarioID)
	if err != nil {
		return nil, err
	}
	for _, eventoID := range eventoIDsComIngresso {
		adicionar(eventoID, SeloIngresso)
	}

	resultado := make([]MeuEvento, 0, len(selosPorEvento))
	for eventoID, selos := range selosPorEvento {
		evento, err := s.eventos.BuscarPorID(eventoID)
		if err != nil {
			continue // evento pode ter sido removido — não quebra a lista toda
		}
		listaSelos := make([]Selo, 0, len(selos))
		for selo := range selos {
			listaSelos = append(listaSelos, selo)
		}
		resultado = append(resultado, MeuEvento{Evento: *evento, Selos: listaSelos})
	}

	return resultado, nil
}

func (s *ContaService) MeusIngressos(usuarioID int64) ([]repository.ItemComEvento, error) {
	return s.itensPedido.ListarPagosPorUsuario(usuarioID)
}

// MeuIngresso é o GET /me/ingressos/:id (detalhe com QR, seção 8) — só
// devolve o item se pertencer mesmo ao usuário logado.
func (s *ContaService) MeuIngresso(usuarioID, itemID int64) (*domain.ItemPedido, error) {
	item, err := s.itensPedido.BuscarPorID(itemID)
	if err != nil {
		return nil, ErrItemNaoEncontrado
	}
	pedido, err := s.pedidos.BuscarPorID(item.PedidoID)
	if err != nil {
		return nil, ErrItemNaoEncontrado
	}
	if pedido.UsuarioID != usuarioID {
		return nil, ErrItemNaoPertenceAoUsuario
	}
	return item, nil
}
