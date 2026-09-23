package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/WilliamBreno/Arcadia/backend/internal/mail"
	"github.com/WilliamBreno/Arcadia/backend/internal/mercadopago"
	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
)

var (
	ErrItemNaoEncontrado        = errors.New("item não encontrado")
	ErrItemNaoPertenceAoUsuario = errors.New("item não pertence a este usuário")
	ErrCancelamentoNaoPermitido = errors.New("cancelamento não permitido")
	ErrPagamentoNaoEncontrado   = errors.New("pagamento do item não encontrado")
)

// DecisaoCancelamento é o resultado de avaliar as regras da seção 7.4 —
// usado tanto pela simulação (GET) quanto pela execução (POST), para as
// duas nunca divergirem.
type DecisaoCancelamento struct {
	Pode                   bool
	Motivo                 string
	Tipo                   domain.TipoReembolso
	ValorReembolsoCentavos int64
}

type CancelamentoService struct {
	itensPedido    *repository.ItemPedidoRepository
	pedidos        *repository.PedidoRepository
	pagamentos     *repository.PagamentoRepository
	reembolsos     *repository.ReembolsoRepository
	lancamentos    *repository.LancamentoRepository
	eventos        *repository.EventoRepository
	config         *repository.ConfigPlataformaRepository
	mp             *mercadopago.Cliente
	mailCliente    *mail.Cliente
	nomePlataforma string
}

func NovoCancelamentoService(
	itensPedido *repository.ItemPedidoRepository,
	pedidos *repository.PedidoRepository,
	pagamentos *repository.PagamentoRepository,
	reembolsos *repository.ReembolsoRepository,
	lancamentos *repository.LancamentoRepository,
	eventos *repository.EventoRepository,
	config *repository.ConfigPlataformaRepository,
	mp *mercadopago.Cliente,
	mailCliente *mail.Cliente,
	nomePlataforma string,
) *CancelamentoService {
	return &CancelamentoService{
		itensPedido: itensPedido, pedidos: pedidos, pagamentos: pagamentos, reembolsos: reembolsos,
		lancamentos: lancamentos, eventos: eventos, config: config, mp: mp, mailCliente: mailCliente,
		nomePlataforma: nomePlataforma,
	}
}

type contextoCancelamento struct {
	item      *domain.ItemPedido
	pedido    *domain.Pedido
	evento    *domain.Evento
	pagamento *domain.Pagamento
}

func (s *CancelamentoService) carregarContexto(itemID, usuarioID int64) (*contextoCancelamento, error) {
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
	evento, err := s.eventos.BuscarPorID(pedido.EventoID)
	if err != nil {
		return nil, ErrItemNaoEncontrado
	}

	ctx := &contextoCancelamento{item: item, pedido: pedido, evento: evento}

	if item.Status == domain.StatusItemPago {
		pagamento, err := s.pagamentos.BuscarAprovadoPorPedido(pedido.ID)
		if err != nil {
			return nil, ErrPagamentoNaoEncontrado
		}
		ctx.pagamento = pagamento
	}

	return ctx, nil
}

// dentroDoPrazoCDC implementa a janela do direito de arrependimento
// (seção 7.4, CDC art. 49): "≤ 7 dias corridos da compra E agora ≤
// inicio_em − 48h" (limites inclusivos, texto literal do plano). Função
// pura — sem I/O — pra poder testar isolada do resto.
func dentroDoPrazoCDC(dataCompra time.Time, inicioEvento *time.Time, agora time.Time) bool {
	dentroDoPrazoDeCompra := agora.Sub(dataCompra) <= 7*24*time.Hour
	faltamPeloMenos48h := inicioEvento == nil || !agora.After(inicioEvento.Add(-48*time.Hour))
	return dentroDoPrazoDeCompra && faltamPeloMenos48h
}

// calcularValorArrependimento é a fórmula da seção 7.4: reembolso do
// preço sempre; a taxa da plataforma só volta se CDC_REEMBOLSA_TAXA
// estiver ligado (padrão do plano, a confirmar com advogado).
func calcularValorArrependimento(precoCentavos, taxaPlataformaCentavos int64, reembolsaTaxa bool) int64 {
	if reembolsaTaxa {
		return precoCentavos + taxaPlataformaCentavos
	}
	return precoCentavos
}

// avaliar aplica as regras da seção 7.4, nesta ordem:
//  1. item utilizado -> negado
//  2. item não pago -> negado (nada a reembolsar)
//  3. garantia contratada e ainda antes do início do evento -> reembolso total
//  4. sem garantia, mas dentro de 7 dias da compra E ainda faltam mais de
//     48h para o evento -> reembolso total (CDC art. 49)
//  5. caso contrário -> negado
func (s *CancelamentoService) avaliar(ctx *contextoCancelamento) DecisaoCancelamento {
	item := ctx.item

	if item.Status == domain.StatusItemUtilizado {
		return DecisaoCancelamento{Pode: false, Motivo: "ingresso já utilizado, não pode ser cancelado"}
	}
	if item.Status != domain.StatusItemPago {
		return DecisaoCancelamento{Pode: false, Motivo: "item não está pago"}
	}

	agora := time.Now()

	if item.GarantiaContratada && ctx.evento.InicioEm != nil && agora.Before(*ctx.evento.InicioEm) {
		return DecisaoCancelamento{
			Pode: true, Motivo: "garantia de vaga contratada", Tipo: domain.TipoReembolsoGarantia,
			ValorReembolsoCentavos: item.TotalCentavos, // preço + taxa + garantia, tudo de volta
		}
	}

	// Data da compra = quando o pagamento foi aprovado (não a criação da
	// reserva, que é ~15 min antes na pior hipótese).
	dataCompra := item.CriadoEm
	if ctx.pagamento != nil {
		dataCompra = ctx.pagamento.CriadoEm
	}

	if dentroDoPrazoCDC(dataCompra, ctx.evento.InicioEm, agora) {
		reembolsaTaxa, err := s.config.BuscarBool(domain.ChaveCDCReembolsaTaxa)
		if err != nil {
			reembolsaTaxa = true // padrão do plano (seção 7.4) se a config sumir
		}
		return DecisaoCancelamento{
			Pode: true, Motivo: "direito de arrependimento (CDC art. 49)", Tipo: domain.TipoReembolsoArrependimento,
			ValorReembolsoCentavos: calcularValorArrependimento(item.PrecoCentavos, item.TaxaPlataformaCentavos, reembolsaTaxa),
		}
	}

	motivo := "fora do prazo de cancelamento"
	if !item.GarantiaContratada {
		motivo = "fora do prazo de cancelamento — sem garantia de vaga, só é possível cancelar em até 7 dias da compra e com pelo menos 48h de antecedência do evento"
	}
	return DecisaoCancelamento{Pode: false, Motivo: motivo}
}

// Simular é o GET /itens/:id/cancelamento — só avalia, não executa nada.
func (s *CancelamentoService) Simular(itemID, usuarioID int64) (DecisaoCancelamento, error) {
	ctx, err := s.carregarContexto(itemID, usuarioID)
	if err != nil {
		return DecisaoCancelamento{}, err
	}
	return s.avaliar(ctx), nil
}

// Cancelar é o POST /itens/:id/cancelar: reavalia (nunca confia numa
// decisão calculada antes), soltcita o estorno parcial no Mercado Pago
// e só muda o status do item depois do reembolso confirmado — não
// cobra o comprador sem devolver o dinheiro dele.
func (s *CancelamentoService) Cancelar(itemID, usuarioID int64) (*domain.ItemPedido, error) {
	ctx, err := s.carregarContexto(itemID, usuarioID)
	if err != nil {
		return nil, err
	}

	decisao := s.avaliar(ctx)
	if !decisao.Pode {
		return nil, fmt.Errorf("%w: %s", ErrCancelamentoNaoPermitido, decisao.Motivo)
	}

	return s.executarReembolso(ctx, decisao, usuarioID)
}

// executarReembolso é compartilhado entre o cancelamento pelo comprador
// (Cancelar) e pelo organizador (CancelarEvento) — sempre solicita o
// estorno no MP antes de mexer no status do item ou no ledger.
func (s *CancelamentoService) executarReembolso(ctx *contextoCancelamento, decisao DecisaoCancelamento, solicitadoPor int64) (*domain.ItemPedido, error) {
	item, evento, pedido, pagamento := ctx.item, ctx.evento, ctx.pedido, ctx.pagamento

	respostaMP, erroMP := s.mp.SolicitarReembolso(pagamento.MPPaymentID, &decisao.ValorReembolsoCentavos)

	reembolso := &domain.Reembolso{
		PagamentoID:   pagamento.ID,
		ItemID:        item.ID,
		ValorCentavos: decisao.ValorReembolsoCentavos,
		Tipo:          decisao.Tipo,
		Motivo:        decisao.Motivo,
		SolicitadoPor: solicitadoPor,
		CriadoEm:      time.Now(),
	}
	if erroMP != nil {
		reembolso.Status = domain.StatusReembolsoFalhou
		_ = s.reembolsos.Criar(reembolso)
		return nil, fmt.Errorf("erro ao processar reembolso no Mercado Pago: %w", erroMP)
	}
	reembolso.Status = domain.StatusReembolsoConcluido
	reembolso.MPRefundID = fmt.Sprintf("%d", respostaMP.ID)
	if err := s.reembolsos.Criar(reembolso); err != nil {
		return nil, err
	}

	agora := time.Now()
	item.Status = domain.StatusItemCancelado
	item.CanceladoEm = &agora
	item.MotivoCancelamento = decisao.Motivo
	if err := s.itensPedido.Salvar(item); err != nil {
		return nil, err
	}

	lancamentos := []domain.Lancamento{{
		Tipo: domain.LancamentoReembolsoPreco, ValorCentavos: item.PrecoCentavos, Sinal: "-",
		EventoID: &evento.ID, OrganizadorID: &evento.OrganizadorID, PedidoID: &pedido.ID, ItemID: &item.ID, CriadoEm: agora,
	}}
	if decisao.ValorReembolsoCentavos > item.PrecoCentavos {
		valorTaxa := decisao.ValorReembolsoCentavos - item.PrecoCentavos
		if decisao.Tipo == domain.TipoReembolsoGarantia {
			valorTaxa = item.TaxaPlataformaCentavos
		}
		lancamentos = append(lancamentos, domain.Lancamento{
			Tipo: domain.LancamentoReembolsoTaxa, ValorCentavos: valorTaxa, Sinal: "-",
			EventoID: &evento.ID, OrganizadorID: &evento.OrganizadorID, PedidoID: &pedido.ID, ItemID: &item.ID, CriadoEm: agora,
		})
		if decisao.Tipo == domain.TipoReembolsoGarantia && item.GarantiaCentavos > 0 {
			lancamentos = append(lancamentos, domain.Lancamento{
				Tipo: domain.LancamentoReembolsoGarantia, ValorCentavos: item.GarantiaCentavos, Sinal: "-",
				EventoID: &evento.ID, OrganizadorID: &evento.OrganizadorID, PedidoID: &pedido.ID, ItemID: &item.ID, CriadoEm: agora,
			})
		}
	}
	if err := s.lancamentos.CriarEmLote(lancamentos); err != nil {
		return nil, err
	}

	s.enviarEmailCancelamento(item, evento, decisao)

	return item, nil
}

func (s *CancelamentoService) enviarEmailCancelamento(item *domain.ItemPedido, evento *domain.Evento, decisao DecisaoCancelamento) {
	if item.TitularEmail == "" {
		return
	}
	corpo := fmt.Sprintf(
		`<p>Olá!</p><p>Seu ingresso (código <strong>%s</strong>) para <strong>%s</strong> foi cancelado.</p><p>Valor reembolsado: R$ %.2f — o dinheiro volta pelo mesmo meio de pagamento em alguns dias úteis.</p>`,
		item.Codigo, evento.Titulo, float64(decisao.ValorReembolsoCentavos)/100,
	)
	_ = s.mailCliente.Enviar(item.TitularEmail, "Cancelamento confirmado — "+s.nomePlataforma, corpo)
}

// CancelarEvento é o cancelamento pelo organizador (seção 7.5): reembolsa
// integralmente (preço + taxa + garantia) todos os itens pagos. Segue em
// frente mesmo se algum reembolso falhar — reúne as falhas para o painel
// de falhas do admin em vez de travar o cancelamento inteiro por causa
// de um item problemático.
func (s *CancelamentoService) CancelarEvento(organizadorID, eventoID int64, motivo string) (sucessos int, falhas []int64, err error) {
	evento, err := s.eventos.BuscarPorID(eventoID)
	if err != nil {
		return 0, nil, fmt.Errorf("evento não encontrado: %w", err)
	}
	if evento.OrganizadorID != organizadorID {
		return 0, nil, ErrEventoNaoPertenceAoOrganizador
	}

	itens, err := s.itensPedido.ListarPagosPorEvento(eventoID)
	if err != nil {
		return 0, nil, err
	}

	pedidosCache := map[int64]*domain.Pedido{}
	pagamentosCache := map[int64]*domain.Pagamento{}

	for i := range itens {
		item := &itens[i]

		pedido, ok := pedidosCache[item.PedidoID]
		if !ok {
			pedido, err = s.pedidos.BuscarPorID(item.PedidoID)
			if err != nil {
				falhas = append(falhas, item.ID)
				continue
			}
			pedidosCache[item.PedidoID] = pedido
		}

		pagamento, ok := pagamentosCache[pedido.ID]
		if !ok {
			pagamento, err = s.pagamentos.BuscarAprovadoPorPedido(pedido.ID)
			if err != nil {
				falhas = append(falhas, item.ID)
				continue
			}
			pagamentosCache[pedido.ID] = pagamento
		}

		ctx := &contextoCancelamento{item: item, pedido: pedido, evento: evento, pagamento: pagamento}
		decisao := DecisaoCancelamento{
			Pode: true, Motivo: "evento cancelado pelo organizador", Tipo: domain.TipoReembolsoEventoCancelado,
			ValorReembolsoCentavos: item.TotalCentavos,
		}

		if _, err := s.executarReembolso(ctx, decisao, organizadorID); err != nil {
			falhas = append(falhas, item.ID)
			continue
		}
		sucessos++
	}

	agora := time.Now()
	evento.Status = domain.StatusEventoCancelado
	evento.CanceladoEm = &agora
	evento.MotivoCancelamento = motivo
	if err := s.eventos.Salvar(evento); err != nil {
		return sucessos, falhas, err
	}

	return sucessos, falhas, nil
}
