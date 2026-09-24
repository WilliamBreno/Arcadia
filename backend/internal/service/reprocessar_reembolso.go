package service

import (
	"errors"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
)

var ErrReembolsoNaoReprocessavel = errors.New("só reembolsos com status 'falhou' e item ainda pago podem ser reprocessados")

// ReprocessarReembolso repete o estorno de um reembolso que falhou (seção
// 7.5: "job com retry, e painel de falhas para o admin"). Reusa
// executarReembolso, então valem as mesmas garantias: status do item e
// ledger só mudam depois do MP confirmar.
func (s *CancelamentoService) ReprocessarReembolso(reembolsoID, adminID int64) error {
	reembolso, err := s.reembolsos.BuscarPorID(reembolsoID)
	if err != nil || reembolso.Status != domain.StatusReembolsoFalhou {
		return ErrReembolsoNaoReprocessavel
	}
	item, err := s.itensPedido.BuscarPorID(reembolso.ItemID)
	if err != nil || item.Status != domain.StatusItemPago {
		return ErrReembolsoNaoReprocessavel
	}
	pedido, err := s.pedidos.BuscarPorID(item.PedidoID)
	if err != nil {
		return err
	}
	evento, err := s.eventos.BuscarPorID(pedido.EventoID)
	if err != nil {
		return err
	}
	pagamento, err := s.pagamentos.BuscarAprovadoPorPedido(pedido.ID)
	if err != nil {
		return ErrPagamentoNaoEncontrado
	}

	ctx := &contextoCancelamento{item: item, pedido: pedido, evento: evento, pagamento: pagamento}
	decisao := DecisaoCancelamento{Pode: true, Motivo: reembolso.Motivo, Tipo: reembolso.Tipo, ValorReembolsoCentavos: reembolso.ValorCentavos}
	_, err = s.executarReembolso(ctx, decisao, adminID, reembolso)
	return err
}

// ReprocessarFalhas é o job /jobs/reprocessar-reembolsos.
func (s *CancelamentoService) ReprocessarFalhas() (sucessos, falhas int, err error) {
	lista, err := s.reembolsos.ListarDetalhados(string(domain.StatusReembolsoFalhou))
	if err != nil {
		return 0, 0, err
	}
	for _, r := range lista {
		if err := s.ReprocessarReembolso(r.ID, r.SolicitadoPor); err != nil {
			falhas++
			continue
		}
		sucessos++
	}
	return sucessos, falhas, nil
}
