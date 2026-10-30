package service

import (
	"errors"
	"fmt"
	"html"
	"log/slog"
	"time"

	"gorm.io/gorm"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/WilliamBreno/Arcadia/backend/internal/mail"
	"github.com/WilliamBreno/Arcadia/backend/internal/pagamento"
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
	destaques      interface{ ReembolsarDestaquesDoEvento(int64) } // item 10.4
	itensPedido    *repository.ItemPedidoRepository
	pedidos        *repository.PedidoRepository
	pagamentos     *repository.PagamentoRepository
	reembolsos     *repository.ReembolsoRepository
	lancamentos    *repository.LancamentoRepository
	eventos        *repository.EventoRepository
	config         *repository.ConfigPlataformaRepository
	mp             pagamento.Processador
	mailCliente    *mail.Cliente
	nomePlataforma string
	promotor       PromotorListaEspera
	// Item 7.4: e-mail de quem revendeu e bloqueio de cancelar durante a revenda.
	db *gorm.DB
}

// DefinirRevenda liga a revenda oficial (item 7.4) ao cancelamento.
func (s *CancelamentoService) DefinirRevenda(db *gorm.DB) { s.db = db }

// DefinirDestaques: evento cancelado devolve os dias de destaque (item 10.4).
func (s *CancelamentoService) DefinirDestaques(d interface{ ReembolsarDestaquesDoEvento(int64) }) {
	s.destaques = d
}

func NovoCancelamentoService(
	itensPedido *repository.ItemPedidoRepository,
	pedidos *repository.PedidoRepository,
	pagamentos *repository.PagamentoRepository,
	reembolsos *repository.ReembolsoRepository,
	lancamentos *repository.LancamentoRepository,
	eventos *repository.EventoRepository,
	config *repository.ConfigPlataformaRepository,
	mp pagamento.Processador,
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

	// Item de valor zero (gratuito ou cortesia) não tem pagamento.
	if item.Status == domain.StatusItemPago && item.TotalCentavos > 0 {
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

// taxaPagaPeloComprador: com a taxa absorvida pelo organizador (item 5.1)
// o comprador não pagou taxa nenhuma, então não há taxa a devolver a ele.
func taxaPagaPeloComprador(item *domain.ItemPedido) int64 {
	if item.TaxaAbsorvida {
		return 0
	}
	return item.TaxaPlataformaCentavos
}

// decomporReembolso separa o valor devolvido ao comprador em preço, taxa
// e garantia, nessa ordem, para o ledger. Função pura, testada.
func decomporReembolso(item *domain.ItemPedido, valor int64) (preco, taxa, garantia int64) {
	preco = min(valor, item.PrecoCentavos)
	resto := valor - preco
	taxa = min(resto, taxaPagaPeloComprador(item))
	garantia = resto - taxa
	return preco, taxa, garantia
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

	// Item de valor zero (evento gratuito, item 5.1): cancela sem
	// reembolso até o início do evento e a vaga volta ao estoque.
	if item.TotalCentavos == 0 {
		if item.Cortesia {
			return DecisaoCancelamento{Pode: false, Motivo: "cortesia só pode ser revogada pelo organizador"}
		}
		if ctx.evento.InicioEm != nil && !agora.Before(*ctx.evento.InicioEm) {
			return DecisaoCancelamento{Pode: false, Motivo: "o evento já começou"}
		}
		return DecisaoCancelamento{Pode: true, Motivo: "inscrição gratuita cancelada pelo participante"}
	}

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
			ValorReembolsoCentavos: calcularValorArrependimento(item.PrecoCentavos, taxaPagaPeloComprador(item), reembolsaTaxa),
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
	// Item 7.4: cancelar tira o ingresso da revenda; se alguém está pagando
	// por ele agora, espera.
	if s.db != nil {
		if err := retirarSeLivre(s.db, itemID); err != nil {
			return nil, err
		}
	}

	var itemCancelado *domain.ItemPedido
	if decisao.ValorReembolsoCentavos == 0 {
		itemCancelado, err = s.cancelarSemReembolso(ctx.item, ctx.evento, decisao)
	} else {
		itemCancelado, err = s.executarReembolso(ctx, decisao, usuarioID, nil)
	}
	if err == nil && ctx.item.ExpositorID != nil {
		// Item 8.6: cancelar o espaço = desistir; a mesa fica livre.
		LiberarExpositor(s.itensPedido.DB(), *ctx.item.ExpositorID, StatusExpositorDesistiu, "")
		return itemCancelado, nil
	}
	if err == nil && ctx.item.FichaID != nil {
		// Item 8.4: cancelar a inscrição paga = desistir da participação.
		db := s.itensPedido.DB()
		db.Model(&domain.FichaParticipacao{}).Where("id = ?", *ctx.item.FichaID).
			Updates(map[string]any{"status": domain.StatusFichaDesistiu})
		db.Exec(`UPDATE papeis_evento SET status = 'removido' WHERE (evento_id, usuario_id, papel) IN
			(SELECT evento_id, usuario_id, papel FROM fichas_participacao WHERE id = ?)`, *ctx.item.FichaID)
		return itemCancelado, nil
	}
	if err == nil && !ctx.item.Adicional {
		s.cancelarAdicionaisSemIngresso(ctx.item.PedidoID, usuarioID)
	}
	if err == nil && s.promotor != nil {
		s.promotor.PromoverListaEspera(ctx.evento.ID)
	}
	return itemCancelado, err
}

// cancelarAdicionaisSemIngresso (item 10.3): se o pedido ficou sem
// ingresso ativo, os adicionais ainda não entregues são cancelados pelas
// mesmas regras (o entregue fica). Erro num adicional não desfaz o
// cancelamento do ingresso; fica no log para o admin.
func (s *CancelamentoService) cancelarAdicionaisSemIngresso(pedidoID, usuarioID int64) {
	db := s.itensPedido.DB()
	var ingressos int64
	db.Model(&domain.ItemPedido{}).Where("pedido_id = ? AND NOT adicional AND status IN ?", pedidoID,
		[]domain.StatusItemPedido{domain.StatusItemPago, domain.StatusItemUtilizado}).Count(&ingressos)
	if ingressos > 0 {
		return
	}
	var ids []int64
	db.Model(&domain.ItemPedido{}).Where("pedido_id = ? AND adicional AND status = ?", pedidoID, domain.StatusItemPago).Pluck("id", &ids)
	for _, id := range ids {
		if _, err := s.Cancelar(id, usuarioID); err != nil {
			slog.Error("cancelamento: adicional não cancelado junto com o ingresso", "item_id", id, "erro", err)
		}
	}
}

// DefinirPromotor liga a promoção da lista de espera (item 3.2).
func (s *CancelamentoService) DefinirPromotor(p PromotorListaEspera) { s.promotor = p }

// executarReembolso é compartilhado entre o cancelamento pelo comprador
// (Cancelar) e pelo organizador (CancelarEvento) — sempre solicita o
// estorno no MP antes de mexer no status do item ou no ledger.
// Se reembolsoExistente != nil, é um reprocessamento (seções 7.4/7.5): a
// mesma linha de reembolsos é atualizada em vez de criar outra.
func (s *CancelamentoService) executarReembolso(ctx *contextoCancelamento, decisao DecisaoCancelamento, solicitadoPor int64, reembolsoExistente *domain.Reembolso) (*domain.ItemPedido, error) {
	item, evento, pedido, pagamento := ctx.item, ctx.evento, ctx.pedido, ctx.pagamento

	// Invariante da seção 7.4: a soma dos reembolsos nunca excede o valor pago.
	jaReembolsado, err := s.reembolsos.SomaConcluidosPorPagamento(pagamento.ID)
	if err != nil {
		return nil, err
	}
	if jaReembolsado+decisao.ValorReembolsoCentavos > pagamento.ValorCentavos {
		return nil, fmt.Errorf("reembolso excederia o valor pago do pagamento %d", pagamento.ID)
	}

	respostaMP, erroMP := s.mp.Reembolsar(pagamento.MPPaymentID, decisao.ValorReembolsoCentavos)

	reembolso := reembolsoExistente
	salvarReembolso := s.reembolsos.Salvar
	if reembolso == nil {
		reembolso = &domain.Reembolso{
			PagamentoID:        pagamento.ID,
			ItemID:             item.ID,
			ValorCentavos:      decisao.ValorReembolsoCentavos,
			Tipo:               decisao.Tipo,
			Motivo:             decisao.Motivo,
			SolicitadoPor:      solicitadoPor,
			CriadoEm:           time.Now(),
			ProximaTentativaEm: time.Now(),
		}
		salvarReembolso = s.reembolsos.Criar
	}
	if erroMP != nil {
		reembolso.Status = domain.StatusReembolsoFalhou
		reembolso.Tentativas++
		reembolso.UltimoErro = erroMP.Error()
		reembolso.ProximaTentativaEm = time.Now().Add(intervaloReembolso(reembolso.Tentativas))
		_ = salvarReembolso(reembolso)
		return nil, fmt.Errorf("erro ao processar reembolso no Mercado Pago: %w", erroMP)
	}
	reembolso.Status = domain.StatusReembolsoConcluido
	reembolso.MPRefundID = respostaMP.ID
	if err := salvarReembolso(reembolso); err != nil {
		return nil, err
	}

	agora := time.Now()
	item.Status = domain.StatusItemCancelado
	item.CanceladoEm = &agora
	item.MotivoCancelamento = decisao.Motivo
	if err := s.itensPedido.Salvar(item); err != nil {
		return nil, err
	}

	if err := s.lancamentos.CriarEmLote(lancamentosDoReembolso(item, evento, pedido.ID, decisao.ValorReembolsoCentavos, agora)); err != nil {
		return nil, err
	}

	s.enviarEmailCancelamento(item, evento, decisao)

	return item, nil
}

// lancamentosDoReembolso é o ledger de um item cancelado com estorno.
// Com taxa absorvida, a taxa deixa de ser receita da plataforma e deixa
// de ser cobrada do organizador (decisão do dono, seção 2.1).
func lancamentosDoReembolso(item *domain.ItemPedido, evento *domain.Evento, pedidoID int64, valor int64, agora time.Time) []domain.Lancamento {
	var ls []domain.Lancamento
	add := func(tipo domain.TipoLancamento, v int64, sinal string) {
		if v <= 0 {
			return
		}
		ls = append(ls, domain.Lancamento{
			Tipo: tipo, ValorCentavos: v, Sinal: sinal,
			EventoID: &evento.ID, OrganizadorID: &evento.OrganizadorID, PedidoID: &pedidoID, ItemID: &item.ID, CriadoEm: agora,
		})
	}
	preco, taxa, garantia := decomporReembolso(item, valor)
	add(domain.LancamentoReembolsoPreco, preco, "-")
	add(domain.LancamentoReembolsoTaxa, taxa, "-")
	add(domain.LancamentoReembolsoGarantia, garantia, "-")
	if item.TaxaAbsorvida {
		add(domain.LancamentoReembolsoTaxa, item.TaxaPlataformaCentavos, "-")
		add(domain.LancamentoEstornoTaxaAbsorvida, item.TaxaPlataformaCentavos, "+")
	}
	return ls
}

// cancelarSemReembolso cancela item de valor zero (gratuito ou cortesia):
// não há dinheiro a devolver nem lançamento a fazer.
func (s *CancelamentoService) cancelarSemReembolso(item *domain.ItemPedido, evento *domain.Evento, decisao DecisaoCancelamento) (*domain.ItemPedido, error) {
	agora := time.Now()
	item.Status = domain.StatusItemCancelado
	item.CanceladoEm = &agora
	item.MotivoCancelamento = decisao.Motivo
	if err := s.itensPedido.Salvar(item); err != nil {
		return nil, err
	}
	s.enviarEmailCancelamento(item, evento, decisao)
	return item, nil
}

func (s *CancelamentoService) enviarEmailCancelamento(item *domain.ItemPedido, evento *domain.Evento, decisao DecisaoCancelamento) {
	if decisao.Tipo == domain.TipoReembolsoRevenda {
		return // a revenda manda o próprio e-mail, para quem vendeu (devolverRevenda)
	}
	if item.TitularEmail == "" {
		return
	}
	corpo := fmt.Sprintf(
		`<p>Olá!</p><p>Seu ingresso (código <strong>%s</strong>) para <strong>%s</strong> foi cancelado.</p><p>Valor reembolsado: R$ %.2f — o dinheiro volta pelo mesmo meio de pagamento em alguns dias úteis.</p>`,
		item.Codigo, evento.Titulo, float64(decisao.ValorReembolsoCentavos)/100,
	)
	if item.ExpositorID != nil { // item 8.6
		corpo = fmt.Sprintf(
			`<p>Olá!</p><p>Seu espaço de expositor em <strong>%s</strong> foi cancelado (%s).</p><p>Valor reembolsado: %s. O dinheiro volta pelo mesmo meio de pagamento em alguns dias úteis.</p>`,
			html.EscapeString(evento.Titulo), html.EscapeString(decisao.Motivo), formatarReais(decisao.ValorReembolsoCentavos),
		)
	}
	if item.FichaID != nil { // item 8.4
		corpo = fmt.Sprintf(
			`<p>Olá!</p><p>Sua inscrição de participante em <strong>%s</strong> foi cancelada (%s).</p><p>Valor reembolsado: %s. O dinheiro volta pelo mesmo meio de pagamento em alguns dias úteis.</p>`,
			html.EscapeString(evento.Titulo), html.EscapeString(decisao.Motivo), formatarReais(decisao.ValorReembolsoCentavos),
		)
	}
	if decisao.ValorReembolsoCentavos == 0 {
		corpo = fmt.Sprintf(
			`<p>Olá!</p><p>Sua inscrição gratuita (código <strong>%s</strong>) para <strong>%s</strong> foi cancelada.</p>`,
			item.Codigo, evento.Titulo,
		)
	}
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

	sucessos, falhas = s.reembolsarItens(evento, itens, organizadorID, "evento cancelado pelo organizador")
	if s.destaques != nil { // item 10.4
		s.destaques.ReembolsarDestaquesDoEvento(eventoID)
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

// reembolsarItens estorna integralmente (preço + taxa + garantia) cada item
// pago da lista — sem travar se algum falhar; devolve os ids que falharam.
func (s *CancelamentoService) reembolsarItens(evento *domain.Evento, itens []domain.ItemPedido, solicitadoPor int64, motivo string) (sucessos int, falhas []int64) {
	return s.reembolsarItensComo(evento, itens, solicitadoPor, motivo, domain.TipoReembolsoEventoCancelado)
}

// ReembolsarInscricao devolve tudo o que a pessoa pagou pela inscrição de
// talento (item 8.4: o organizador removeu a participação). Erro se algum
// estorno falhar; aí a ficha não muda e o organizador tenta de novo.
func (s *CancelamentoService) ReembolsarInscricao(fichaID, solicitadoPor int64, motivo string) error {
	return s.reembolsarPor("ficha_id", fichaID, solicitadoPor, motivo)
}

// ReembolsarExpositor devolve tudo o que o expositor pagou pelo espaço
// (item 8.6: o organizador removeu o expositor).
func (s *CancelamentoService) ReembolsarExpositor(expositorID, solicitadoPor int64, motivo string) error {
	return s.reembolsarPor("expositor_id", expositorID, solicitadoPor, motivo)
}

func (s *CancelamentoService) reembolsarPor(coluna string, id, solicitadoPor int64, motivo string) error {
	var itens []domain.ItemPedido
	if err := s.itensPedido.DB().Where(coluna+" = ? AND status = ?", id, domain.StatusItemPago).Find(&itens).Error; err != nil {
		return err
	}
	if len(itens) == 0 {
		return nil
	}
	pedido, err := s.pedidos.BuscarPorID(itens[0].PedidoID)
	if err != nil {
		return err
	}
	evento, err := s.eventos.BuscarPorID(pedido.EventoID)
	if err != nil {
		return err
	}
	if _, falhas := s.reembolsarItensComo(evento, itens, solicitadoPor, motivo, domain.TipoReembolsoManual); len(falhas) > 0 {
		return fmt.Errorf("não foi possível devolver o pagamento agora; tente de novo em instantes")
	}
	return nil
}

func (s *CancelamentoService) reembolsarItensComo(evento *domain.Evento, itens []domain.ItemPedido, solicitadoPor int64, motivo string, tipo domain.TipoReembolso) (sucessos int, falhas []int64) {
	pedidosCache := map[int64]*domain.Pedido{}
	pagamentosCache := map[int64]*domain.Pagamento{}

	for i := range itens {
		item := &itens[i]

		if item.TotalCentavos == 0 {
			decisao := DecisaoCancelamento{Pode: true, Motivo: motivo}
			if _, err := s.cancelarSemReembolso(item, evento, decisao); err != nil {
				falhas = append(falhas, item.ID)
				continue
			}
			sucessos++
			continue
		}

		pedido, ok := pedidosCache[item.PedidoID]
		if !ok {
			var err error
			pedido, err = s.pedidos.BuscarPorID(item.PedidoID)
			if err != nil {
				falhas = append(falhas, item.ID)
				continue
			}
			pedidosCache[item.PedidoID] = pedido
		}

		pagamento, ok := pagamentosCache[pedido.ID]
		if !ok {
			var err error
			pagamento, err = s.pagamentos.BuscarAprovadoPorPedido(pedido.ID)
			if err != nil {
				falhas = append(falhas, item.ID)
				continue
			}
			pagamentosCache[pedido.ID] = pagamento
		}

		ctx := &contextoCancelamento{item: item, pedido: pedido, evento: evento, pagamento: pagamento}
		decisao := DecisaoCancelamento{Pode: true, Motivo: motivo, Tipo: tipo, ValorReembolsoCentavos: item.TotalCentavos}

		if _, err := s.executarReembolso(ctx, decisao, solicitadoPor, nil); err != nil {
			falhas = append(falhas, item.ID)
			continue
		}
		sucessos++
	}
	return sucessos, falhas
}

// devolverRevenda estorna ao vendedor o preço do ingresso revendido (item
// 7.4). O ingresso já foi invalidado ao fechar a venda; aqui sai o dinheiro.
func (s *CancelamentoService) devolverRevenda(a *Revenda) error {
	item, err := s.itensPedido.BuscarPorID(a.ItemID)
	if err != nil {
		return err
	}
	pedido, err := s.pedidos.BuscarPorID(item.PedidoID)
	if err != nil {
		return err
	}
	evento, err := s.eventos.BuscarPorID(pedido.EventoID)
	if err != nil {
		return err
	}
	pag, err := s.pagamentos.BuscarAprovadoPorPedido(pedido.ID)
	if err != nil {
		return ErrPagamentoNaoEncontrado
	}
	ctx := &contextoCancelamento{item: item, pedido: pedido, evento: evento, pagamento: pag}
	decisao := DecisaoCancelamento{Pode: true, Motivo: MotivoRevendido, Tipo: domain.TipoReembolsoRevenda, ValorReembolsoCentavos: a.ValorDevolverCentavos}
	if _, err := s.executarReembolso(ctx, decisao, a.VendedorID, nil); err != nil {
		return err
	}
	s.avisarRevendido(a, item, evento, pag.Metodo)
	return nil
}

func (s *CancelamentoService) avisarRevendido(a *Revenda, item *domain.ItemPedido, evento *domain.Evento, metodo string) {
	if s.db == nil {
		return
	}
	var u struct{ Nome, Email string }
	s.db.Table("usuarios").Select("nome, email").Where("id = ?", a.VendedorID).Scan(&u)
	if u.Email == "" {
		return
	}
	prazo := "O Pix volta para a sua conta em instantes."
	if metodo == pagamento.MetodoCartao {
		prazo = "No cartão, o estorno aparece na fatura em até duas faturas, conforme o banco."
	}
	corpo := fmt.Sprintf(`<div style="font-family:Arial,sans-serif;max-width:560px;margin:auto;color:#111">`+
		`<p>Olá, %s! Seu ingresso (código <strong>%s</strong>) para <strong>%s</strong> foi vendido na revenda oficial.</p>`+
		`<p>Devolvemos <strong>%s</strong>, o preço que você pagou pelo ingresso. %s</p>`+
		`<p>Esse ingresso não vale mais para entrar: o QR Code foi trocado pelo do novo dono.</p></div>`,
		html.EscapeString(primeiroNome(u.Nome)), item.Codigo, html.EscapeString(evento.Titulo), formatarReais(a.ValorDevolverCentavos), prazo)
	_ = s.mailCliente.Enviar(u.Email, "Seu ingresso foi revendido — "+evento.Titulo, corpo)
}
