package service

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/WilliamBreno/Arcadia/backend/internal/mail"
	"github.com/WilliamBreno/Arcadia/backend/internal/pagamento"
	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
)

var (
	ErrEventoNaoDisponivelParaCompra    = errors.New("evento não está disponível para compra")
	ErrPedidoVazio                      = errors.New("pedido não pode ficar vazio")
	ErrExcedeuMaxItensPorPedido         = errors.New("quantidade excede o máximo por pedido")
	ErrTipoIndisponivel                 = errors.New("tipo de ingresso inativo")
	ErrVendasNaoAbertas                 = errors.New("vendas ainda não abertas")
	ErrVendasEncerradas                 = errors.New("vendas encerradas")
	ErrQuantidadeInvalida               = errors.New("quantidade fora do permitido")
	ErrEstoqueInsuficiente              = errors.New("estoque insuficiente")
	ErrPedidoNaoEncontrado              = errors.New("pedido não encontrado")
	ErrPedidoNaoPertenceAoUsuario       = errors.New("pedido não pertence a este usuário")
	ErrPedidoNaoDisponivelParaPagamento = errors.New("pedido não está aguardando pagamento")
	ErrReservaExpirada                  = errors.New("reserva expirada")
	ErrMercadoPagoNaoConfigurado        = errors.New("Mercado Pago não configurado")
	ErrGarantiaNaoDisponivel            = errors.New("este evento não oferece garantia de vaga")
	ErrGarantiaAposInicio               = errors.New("a garantia de vaga só pode ser contratada antes do início do evento")
	ErrLoteIndisponivel                 = errors.New("este lote não está disponível no momento")
)

type ItemRequisitado struct {
	TipoIngressoID     int64
	Quantidade         int
	GarantiaContratada bool
	// Item 7.6: respostas às perguntas do organizador, uma por ingresso.
	Respostas []map[int64]string
}

type CheckoutService struct {
	db             *gorm.DB
	eventos        *repository.EventoRepository
	itensPedido    *repository.ItemPedidoRepository
	pedidos        *repository.PedidoRepository
	pagamentos     *repository.PagamentoRepository
	lancamentos    *repository.LancamentoRepository
	config         *repository.ConfigPlataformaRepository
	cupons         *repository.CupomRepository
	tipos          *repository.TipoIngressoRepository
	plateia        *PlateiaService
	fila           *FilaEsperaService
	revenda        *RevendaService
	taxas          *TaxasService
	usuarios       *repository.UsuarioRepository
	sessoes        *repository.SessaoRepository
	contestacoes   *repository.ContestacaoRepository
	afiliados      *repository.AfiliadoRepository
	mp             pagamento.Processador
	mailCliente    *mail.Cliente
	qrSecret       string
	frontendURL    string
	backendURL     string
	nomePlataforma string
	avisoInscricao AvisoInscricao // item 8.4
	avisoEspaco    AvisoEspaco    // item 8.6
}

func NovoCheckoutService(
	db *gorm.DB,
	eventos *repository.EventoRepository,
	itensPedido *repository.ItemPedidoRepository,
	pedidos *repository.PedidoRepository,
	pagamentos *repository.PagamentoRepository,
	lancamentos *repository.LancamentoRepository,
	config *repository.ConfigPlataformaRepository,
	cupons *repository.CupomRepository,
	tipos *repository.TipoIngressoRepository,
	mp pagamento.Processador,
	mailCliente *mail.Cliente,
	qrSecret, frontendURL, backendURL, nomePlataforma string,
) *CheckoutService {
	return &CheckoutService{
		db: db, eventos: eventos, itensPedido: itensPedido, pedidos: pedidos,
		pagamentos: pagamentos, lancamentos: lancamentos, config: config, cupons: cupons, tipos: tipos,
		mp: mp, mailCliente: mailCliente, qrSecret: qrSecret,
		frontendURL: frontendURL, backendURL: backendURL, nomePlataforma: nomePlataforma,
	}
}

func gerarCodigoItem() (string, error) {
	const alfabeto = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // sem O/0/I/1, evita confusão na portaria
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	codigo := make([]byte, 8)
	for i, b := range buf {
		codigo[i] = alfabeto[int(b)%len(alfabeto)]
	}
	return string(codigo), nil
}

// gerarQRToken assina o código do item (seção 7.10) — o leitor de check-in
// (item 1.10) confere essa assinatura antes de aceitar o QR como válido.
func (s *CheckoutService) gerarQRToken(codigo string) string {
	return assinarQR(s.qrSecret, codigo)
}

func assinarQR(segredo, codigo string) string {
	h := hmac.New(sha256.New, []byte(segredo))
	h.Write([]byte(codigo))
	return hex.EncodeToString(h.Sum(nil))
}

// Reservar é a etapa "reserva + cálculo" da seção 8. Trava cada
// tipo_ingresso (SELECT ... FOR UPDATE) dentro de uma transação para
// impedir overselling sob concorrência (seção 7.2) — preço, taxa e
// garantia são sempre recalculados aqui, nunca aceitos do cliente.
func (s *CheckoutService) Reservar(usuarioID, eventoID int64, itensReq []ItemRequisitado, compradorNome, compradorEmail, cupomCodigo, afiliadoCodigo string) (*domain.Pedido, []domain.ItemPedido, error) {
	evento, err := s.eventos.BuscarPorID(eventoID)
	if err != nil {
		return nil, nil, fmt.Errorf("evento não encontrado: %w", err)
	}
	if evento.Status != domain.StatusEventoPublicado {
		return nil, nil, ErrEventoNaoDisponivelParaCompra
	}
	if s.plateia != nil && !s.plateia.PodeComprar(evento, usuarioID) {
		return nil, nil, ErrPrecisaAprovacao
	}

	totalUnidades := 0
	for _, ir := range itensReq {
		totalUnidades += ir.Quantidade
	}
	if totalUnidades == 0 {
		return nil, nil, ErrPedidoVazio
	}
	if evento.MaxItensPorPedido > 0 && totalUnidades > evento.MaxItensPorPedido {
		return nil, nil, ErrExcedeuMaxItensPorPedido
	}

	// Seção 7.1 / item 5.1: taxa por faixa do preço com desconto, limite
	// regional e garantia percentual — tudo recalculado aqui.
	taxasEvento, err := s.taxas.ParaEvento(evento)
	if err != nil {
		return nil, nil, err
	}
	reservaMinutos, err := s.config.BuscarInt64(domain.ChaveReservaMinutos)
	if err != nil {
		return nil, nil, fmt.Errorf("config de reserva ausente: %w", err)
	}
	agoraReserva := time.Now()
	for _, ir := range itensReq {
		if ir.GarantiaContratada && !evento.GarantiaHabilitada {
			return nil, nil, ErrGarantiaNaoDisponivel
		}
		// A garantia vale "até o início do evento": depois disso não há o
		// que garantir (item 5.5).
		if ir.GarantiaContratada && evento.InicioEm != nil && !agoraReserva.Before(*evento.InicioEm) {
			return nil, nil, ErrGarantiaAposInicio
		}
	}
	var sessoesEvento []domain.Sessao
	if s.sessoes != nil {
		if sessoesEvento, err = s.sessoes.ListarPorEvento(evento.ID); err != nil {
			return nil, nil, err
		}
	}

	var pedido *domain.Pedido
	var itensCriados []domain.ItemPedido

	err = s.db.Transaction(func(tx *gorm.DB) error {
		agora := time.Now()
		expiraEm := agora.Add(time.Duration(reservaMinutos) * time.Minute)
		pedido = &domain.Pedido{
			UsuarioID: usuarioID,
			EventoID:  eventoID,
			Status:    domain.StatusPedidoAguardandoPagamento,
			ExpiraEm:  &expiraEm,
			CriadoEm:  agora,
		}
		// ref inválido ou inativo é ignorado (não bloqueia a compra).
		var afiliado *domain.Afiliado
		if afiliadoCodigo != "" && s.afiliados != nil {
			if af, errAf := s.afiliados.BuscarAtivoPorCodigo(tx, eventoID, afiliadoCodigo); errAf == nil {
				pedido.AfiliadoID = &af.ID
				afiliado = af
			}
		}
		if err := s.checarLimitePorConta(tx, evento, usuarioID, totalUnidades); err != nil {
			return err
		}
		perguntas, err := PerguntasAtivas(tx, eventoID) // item 7.6
		if err != nil {
			return err
		}
		if err := s.pedidos.Criar(tx, pedido); err != nil {
			return err
		}

		// Cupom vale para todos os itens do pedido; a linha é travada para
		// max_usos não estourar sob concorrência.
		var cupom *domain.Cupom
		if cupomCodigo != "" {
			c, errCupom := s.cupons.BuscarPorCodigo(tx, eventoID, cupomCodigo, true)
			if errCupom != nil || !cupomUtilizavel(c, totalUnidades, agora) {
				return ErrCupomInvalido
			}
			cupom = c
		}

		var totalPedido int64

		for _, ir := range itensReq {
			tipo, err := s.itensPedido.BuscarTipoIngressoParaAtualizar(tx, ir.TipoIngressoID)
			if err != nil || tipo.EventoID != eventoID {
				return fmt.Errorf("tipo de ingresso não encontrado")
			}
			if !tipo.Ativo {
				return fmt.Errorf("%w: %s", ErrTipoIndisponivel, tipo.Nome)
			}
			if tipo.VendasInicio != nil && agora.Before(*tipo.VendasInicio) {
				return fmt.Errorf("%w: %s", ErrVendasNaoAbertas, tipo.Nome)
			}
			if tipo.VendasFim != nil && agora.After(*tipo.VendasFim) {
				return fmt.Errorf("%w: %s", ErrVendasEncerradas, tipo.Nome)
			}
			if s.sessoes != nil {
				comSessoes := []domain.TipoIngresso{*tipo}
				if err := s.tipos.PreencherSessoesCom(tx, comSessoes); err != nil {
					return err
				}
				if !VendaAberta(evento, comSessoes[0].SessaoIDs, sessoesEvento, agora) {
					return fmt.Errorf("%w: %s (o evento ou o dia deste ingresso já terminou)", ErrVendasEncerradas, tipo.Nome)
				}
			}
			if ir.Quantidade < tipo.MinPorPedido || (tipo.MaxPorPedido > 0 && ir.Quantidade > tipo.MaxPorPedido) {
				return fmt.Errorf("%w para %s", ErrQuantidadeInvalida, tipo.Nome)
			}

			if tipo.LoteGrupo != "" {
				grupo, err := s.tipos.ListarPorGrupoLote(tx, eventoID, tipo.LoteGrupo)
				if err != nil {
					return err
				}
				ids := make([]int64, len(grupo))
				for i := range grupo {
					ids[i] = grupo[i].ID
				}
				vendidos, err := s.itensPedido.ContarAtivosPorTipos(tx, ids)
				if err != nil {
					return err
				}
				if LoteAtual(grupo, vendidos, agora) != tipo.ID {
					return fmt.Errorf("%w: %s", ErrLoteIndisponivel, tipo.Nome)
				}
			}

			ativos, err := s.itensPedido.ContarAtivosPorTipo(tx, tipo.ID)
			if err != nil {
				return err
			}
			// Item 7.3: vagas guardadas para a lista de espera não estão à
			// venda (menos para a própria pessoa convidada).
			guardadas, err := VagasGuardadas(tx, tipo.ID, usuarioID)
			if err != nil {
				return err
			}
			// Item 7.4: ingressos na revenda oficial também estão à venda; o
			// estoque do organizador vende primeiro.
			aVenda, err := RevendasAVenda(tx, tipo.ID)
			if err != nil {
				return err
			}
			estoqueOrganizador := int64(tipo.Quantidade) - ativos - guardadas
			disponivel := VagasParaComprar(int64(tipo.Quantidade)-ativos, aVenda, guardadas)
			if int64(ir.Quantidade) > disponivel {
				return fmt.Errorf("%w: %s (restam %d)", ErrEstoqueInsuficiente, tipo.Nome, disponivel)
			}

			var descontoCentavos int64
			var cupomID *int64
			if cupom != nil {
				descontoCentavos = calcularDesconto(cupom.Tipo, cupom.Valor, tipo.PrecoCentavos)
				cupomID = &cupom.ID
			}
			precoFinal := tipo.PrecoCentavos - descontoCentavos
			// Item gratuito não tem garantia (não há o que devolver).
			valores := taxasEvento.Calcular(precoFinal, ir.GarantiaContratada && precoFinal > 0)

			for i := 0; i < ir.Quantidade; i++ {
				codigo, err := gerarCodigoItem()
				if err != nil {
					return err
				}
				item := domain.ItemPedido{
					PedidoID:               pedido.ID,
					TipoIngressoID:         tipo.ID,
					TitularNome:            compradorNome,
					TitularEmail:           compradorEmail,
					CupomID:                cupomID,
					DescontoCentavos:       descontoCentavos,
					PrecoCentavos:          precoFinal,
					TaxaPlataformaCentavos: valores.TaxaCentavos,
					TaxaAbsorvida:          valores.TaxaAbsorvida,
					GarantiaContratada:     valores.GarantiaCentavos > 0,
					GarantiaCentavos:       valores.GarantiaCentavos,
					TotalCentavos:          valores.TotalCentavos,
					// Item 9.2: comissão do divulgador pela regra de agora.
					ComissaoAfiliadoCentavos: afiliado.ComissaoDoItem(precoFinal),
					Status:                   domain.StatusItemReservado,
					Codigo:                   codigo,
					QRToken:                  s.gerarQRToken(codigo),
					CriadoEm:                 agora,
				}
				if err := s.itensPedido.Criar(tx, &item); err != nil {
					return err
				}
				totalPedido += item.TotalCentavos
				itensCriados = append(itensCriados, item)
			}
			{
				novos := itensCriados[len(itensCriados)-ir.Quantidade:]
				ids := make([]int64, len(novos))
				for k := range novos {
					ids[k] = novos[k].ID
				}
				if err := salvarRespostas(tx, perguntas, tipo.ID, ids, ir.Respostas); err != nil {
					return err
				}
			}
			if n := QuantasDaRevenda(ir.Quantidade, estoqueOrganizador); n > 0 {
				novos := itensCriados[len(itensCriados)-ir.Quantidade:]
				ids := make([]int64, 0, n)
				for k := 0; k < n; k++ {
					ids = append(ids, novos[k].ID)
				}
				if err := reivindicarRevendas(tx, tipo.ID, ids); err != nil {
					return err
				}
			}
			if err := consumirConvite(tx, usuarioID, tipo.ID, pedido.ID); err != nil {
				return err
			}
		}

		if cupom != nil {
			if err := s.cupons.AjustarUsos(tx, cupom.ID, totalUnidades); err != nil {
				return err
			}
		}

		pedido.TotalCentavos = totalPedido
		return tx.Save(pedido).Error
	})
	if err != nil {
		return nil, nil, err
	}

	// Pedido de valor zero (evento gratuito ou cupom de 100% sem garantia):
	// não passa pelo Mercado Pago — confirma na hora (item 5.1).
	if pedido.TotalCentavos == 0 {
		if err := s.aprovarPedido(pedido); err != nil {
			return nil, nil, err
		}
		itensCriados, err = s.itensPedido.ListarPorPedido(pedido.ID)
		if err != nil {
			return nil, nil, err
		}
	}

	return pedido, itensCriados, nil
}

func (s *CheckoutService) ObterPedido(usuarioID, pedidoID int64) (*domain.Pedido, []domain.ItemPedido, error) {
	pedido, err := s.pedidos.BuscarPorID(pedidoID)
	if err != nil {
		return nil, nil, ErrPedidoNaoEncontrado
	}
	if pedido.UsuarioID != usuarioID {
		return nil, nil, ErrPedidoNaoPertenceAoUsuario
	}
	itens, err := s.itensPedido.ListarPorPedido(pedido.ID)
	if err != nil {
		return nil, nil, err
	}
	return pedido, itens, nil
}

// IniciarPagamento cria a cobrança no processador (Checkout Pro na conta
// da plataforma — seção 2.3: sem split). metodo é "pix" (padrão,
// recomendado) ou "cartao": o checkout do processador mostra só o meio
// escolhido (item 5.2). Gerar a cobrança não move dinheiro.
func (s *CheckoutService) IniciarPagamento(usuarioID, pedidoID int64, metodo string) (string, error) {
	if !s.mp.Habilitado() {
		return "", ErrMercadoPagoNaoConfigurado
	}
	if metodo != pagamento.MetodoCartao {
		metodo = pagamento.MetodoPix
	}
	if metodo == pagamento.MetodoCartao && s.cartaoBloqueado(usuarioID) {
		return "", ErrCartaoBloqueado
	}

	pedido, itens, err := s.ObterPedido(usuarioID, pedidoID)
	if err != nil {
		return "", err
	}
	if pedido.Status != domain.StatusPedidoAguardandoPagamento {
		return "", ErrPedidoNaoDisponivelParaPagamento
	}
	if pedido.ExpiraEm != nil && pedido.ExpiraEm.Before(time.Now()) {
		return "", ErrReservaExpirada
	}

	tiposCache := map[int64]*domain.TipoIngresso{}
	cobrancaItens := make([]pagamento.Item, 0, len(itens))
	for _, item := range itens {
		if item.Status != domain.StatusItemReservado {
			continue
		}
		tipo, ok := tiposCache[item.TipoIngressoID]
		if !ok {
			tipo, err = s.itensPedido.BuscarTipoIngressoParaAtualizar(s.db, item.TipoIngressoID)
			if err != nil {
				return "", err
			}
			tiposCache[item.TipoIngressoID] = tipo
		}
		cobrancaItens = append(cobrancaItens, pagamento.Item{Titulo: tipo.Nome, ValorCentavos: item.TotalCentavos})
	}

	pedidoIDStr := strconv.FormatInt(pedido.ID, 10)
	nova := pagamento.NovaCobranca{
		Referencia:     pedidoIDStr,
		Itens:          cobrancaItens,
		Metodo:         metodo,
		URLSucesso:     s.frontendURL + "/pedidos/" + pedidoIDStr + "?status=sucesso",
		URLFalha:       s.frontendURL + "/pedidos/" + pedidoIDStr + "?status=falha",
		URLPendente:    s.frontendURL + "/pedidos/" + pedidoIDStr + "?status=pendente",
		URLNotificacao: s.backendURL + "/api/v1/webhooks/mercadopago",
		MaxParcelas:    MaxParcelas(pedido.TotalCentavos),
	}
	if metodo == pagamento.MetodoPix && pedido.ExpiraEm != nil && time.Until(*pedido.ExpiraEm) < ValidadePix {
		// O Pix vale no mínimo 30 minutos no MP: a reserva acompanha (7.1).
		novo := time.Now().Add(ValidadePix)
		pedido.ExpiraEm = &novo
	}
	if pedido.ExpiraEm != nil {
		nova.ExpiraEm = *pedido.ExpiraEm
	}
	cobranca, err := s.mp.CriarCobranca(nova)
	if err != nil {
		return "", err
	}

	pedido.MPPreferenceID = &cobranca.ID
	if err := s.pedidos.Salvar(pedido); err != nil {
		return "", err
	}
	return cobranca.URL, nil
}

// ProcessarWebhook é idempotente por mp_payment_id (seção 7.3) e SEMPRE
// reconsulta o pagamento na API do MP — nunca confia em valor/status do
// corpo da notificação recebida.
func (s *CheckoutService) ProcessarWebhook(mpPaymentID string) error {
	pagamentoMP, err := s.mp.BuscarPagamento(mpPaymentID)
	if err != nil {
		return err
	}

	// Pagamento já registrado: a notificação é uma mudança de status
	// (Pix pendente → aprovado, contestação aberta/perdida...). Antes do
	// item 5.3 essas notificações eram ignoradas.
	if existente, err := s.pagamentos.BuscarPorMPPaymentID(mpPaymentID); err == nil {
		return s.atualizarPagamento(existente, pagamentoMP)
	}

	pedidoID, err := strconv.ParseInt(pagamentoMP.Referencia, 10, 64)
	if err != nil {
		return fmt.Errorf("external_reference inválida: %q", pagamentoMP.Referencia)
	}

	pedido, err := s.pedidos.BuscarPorID(pedidoID)
	if err != nil {
		return fmt.Errorf("pedido %d do webhook não encontrado: %w", pedidoID, err)
	}

	payloadJSON, err := json.Marshal(pagamentoMP.Bruto)
	if err != nil {
		return err
	}

	registro := &domain.Pagamento{
		PedidoID:                pedido.ID,
		MPPaymentID:             mpPaymentID,
		Processador:             s.mp.Nome(),
		Metodo:                  pagamentoMP.Metodo,
		Status:                  pagamentoMP.Status,
		ValorCentavos:           pagamentoMP.ValorCentavos,
		TaxaProcessadorCentavos: pagamentoMP.TaxaProcessadorCentavos,
		PayloadJSON:             datatypes.JSON(payloadJSON),
		CriadoEm:                time.Now(),
	}
	if err := s.pagamentos.Criar(registro); err != nil {
		// Corrida rara (dois webhooks quase simultâneos): índice único em
		// mp_payment_id barra a duplicata — trata como já processado.
		return nil
	}

	if statusContestacao(pagamentoMP.Status) != "" {
		return s.registrarContestacao(registro, pedido, pagamentoMP.Status)
	}
	if pagamentoMP.Status != pagamento.StatusAprovado {
		return nil
	}

	return s.confirmarPagamento(pedido.ID, registro)
}

// aprovarPedido confirma o pedido de valor zero (sem processador).
func (s *CheckoutService) aprovarPedido(pedido *domain.Pedido) error {
	if err := s.confirmarPagamento(pedido.ID, nil); err != nil {
		return err
	}
	atual, err := s.pedidos.BuscarPorID(pedido.ID)
	if err != nil {
		return err
	}
	*pedido = *atual
	return nil
}

// lancamentosDaVenda é o ledger de um item aprovado (seção 7.3). Valores
// zero não geram linha (item gratuito). Com taxa absorvida, o organizador
// paga a taxa: ela entra como receita da plataforma e sai do saldo dele.
func lancamentosDaVenda(item *domain.ItemPedido, evento *domain.Evento, pedidoID int64, agora time.Time) []domain.Lancamento {
	var ls []domain.Lancamento
	add := func(tipo domain.TipoLancamento, valor int64, sinal string) {
		if valor <= 0 {
			return
		}
		ls = append(ls, domain.Lancamento{
			Tipo: tipo, ValorCentavos: valor, Sinal: sinal,
			EventoID: &evento.ID, OrganizadorID: &evento.OrganizadorID, PedidoID: &pedidoID, ItemID: &item.ID, CriadoEm: agora,
		})
	}
	add(domain.LancamentoVendaPreco, item.PrecoCentavos, "+")
	add(domain.LancamentoTaxaPlataforma, item.TaxaPlataformaCentavos, "+")
	if item.TaxaAbsorvida {
		add(domain.LancamentoTaxaAbsorvida, item.TaxaPlataformaCentavos, "-")
	}
	if item.GarantiaContratada {
		add(domain.LancamentoGarantia, item.GarantiaCentavos, "+")
	}
	return ls
}

func (s *CheckoutService) enviarEmailConfirmacao(pedido *domain.Pedido, itens []domain.ItemPedido, evento *domain.Evento) {
	if len(itens) == 0 {
		return
	}
	destinatario := itens[0].TitularEmail
	if destinatario == "" {
		return
	}

	pagos := make([]domain.ItemPedido, 0, len(itens))
	for _, it := range itens {
		if it.Status == domain.StatusItemPago && !it.Interno() { // 8.4/8.6: inscrição e espaço têm e-mail próprio
			pagos = append(pagos, it)
		}
	}
	if len(pagos) == 0 {
		return
	}
	intro := fmt.Sprintf("Seu pagamento (pedido #%d, total %s) foi aprovado. Seus ingressos estão abaixo — guarde este e-mail como garantia.", pedido.ID, formatarReais(pedido.TotalCentavos))
	corpo, imagens := corpoEmailIngressos("Olá, "+itens[0].TitularNome+"!", intro, evento, pagos, s.qrSecret, s.frontendURL)
	_ = s.mailCliente.EnviarComImagens(destinatario, "Ingresso confirmado — "+evento.Titulo+" — "+s.nomePlataforma, corpo, imagens)
}

// ExpirarReservas roda periodicamente (job protegido por X-Cron-Secret,
// seção 4) devolvendo ao estoque os itens cuja reserva venceu sem
// pagamento (seção 7.2).
func (s *CheckoutService) ExpirarReservas() (int, error) {
	pedidosExpirados, err := s.pedidos.ListarExpirados()
	if err != nil {
		return 0, err
	}

	total := 0
	eventosLiberados := map[int64]bool{}
	for i := range pedidosExpirados {
		expirou := false
		// Trava o pedido e confere de novo: um pagamento pode ter sido
		// aprovado entre a listagem e aqui (confirmarPagamento também trava).
		err := s.db.Transaction(func(tx *gorm.DB) error {
			var pedido domain.Pedido
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&pedido, pedidosExpirados[i].ID).Error; err != nil {
				return err
			}
			if pedido.Status != domain.StatusPedidoAguardandoPagamento || pedido.ExpiraEm == nil || pedido.ExpiraEm.After(time.Now()) {
				return nil
			}
			var itens []domain.ItemPedido
			if err := tx.Where("pedido_id = ? AND status = ?", pedido.ID, domain.StatusItemReservado).Find(&itens).Error; err != nil {
				return err
			}
			ids := make([]int64, 0, len(itens))
			for j := range itens {
				ids = append(ids, itens[j].ID)
			}
			if err := liberarRevendas(tx, ids); err != nil { // item 7.4
				return err
			}
			for j := range itens {
				itens[j].Status = domain.StatusItemExpirado
				if err := tx.Save(&itens[j]).Error; err != nil {
					return err
				}
				if itens[j].CupomID != nil {
					if err := s.cupons.AjustarUsos(tx, *itens[j].CupomID, -1); err != nil {
						return err
					}
				}
			}
			pedido.Status = domain.StatusPedidoExpirado
			expirou = true
			return tx.Save(&pedido).Error
		})
		if err != nil {
			return total, err
		}
		if expirou {
			eventosLiberados[pedidosExpirados[i].EventoID] = true
			total++
		}
	}
	for eventoID := range eventosLiberados {
		if s.plateia != nil {
			s.plateia.PromoverListaEspera(eventoID)
		}
		if s.fila != nil {
			s.fila.PromoverListaEspera(eventoID) // item 7.3
		}
	}
	return total, nil
}

// DefinirRevenda liga a revenda oficial (item 7.4).
func (s *CheckoutService) DefinirRevenda(r *RevendaService) { s.revenda = r }

// DefinirFila liga a lista de espera de esgotado (item 7.3).
func (s *CheckoutService) DefinirFila(f *FilaEsperaService) { s.fila = f }

// DefinirPlateia liga a exigência de aprovação e a promoção da lista de espera (item 3.2).
func (s *CheckoutService) DefinirPlateia(p *PlateiaService) { s.plateia = p }

// DefinirSessoes liga o bloqueio de venda depois do fim do evento ou das
// sessões do ingresso (item 5.5).
func (s *CheckoutService) DefinirSessoes(r *repository.SessaoRepository) { s.sessoes = r }

// DefinirTaxas liga o cálculo de taxas por faixa (item 5.1). Obrigatório.
func (s *CheckoutService) DefinirTaxas(t *TaxasService) { s.taxas = t }

// DefinirAfiliados liga a atribuição por link de divulgador.
func (s *CheckoutService) DefinirAfiliados(a *repository.AfiliadoRepository) { s.afiliados = a }
