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

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/WilliamBreno/Arcadia/backend/internal/mail"
	"github.com/WilliamBreno/Arcadia/backend/internal/mercadopago"
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
)

type ItemRequisitado struct {
	TipoIngressoID     int64
	Quantidade         int
	GarantiaContratada bool
}

type CheckoutService struct {
	db             *gorm.DB
	eventos        *repository.EventoRepository
	itensPedido    *repository.ItemPedidoRepository
	pedidos        *repository.PedidoRepository
	pagamentos     *repository.PagamentoRepository
	lancamentos    *repository.LancamentoRepository
	config         *repository.ConfigPlataformaRepository
	mp             *mercadopago.Cliente
	mailCliente    *mail.Cliente
	qrSecret       string
	frontendURL    string
	backendURL     string
	nomePlataforma string
}

func NovoCheckoutService(
	db *gorm.DB,
	eventos *repository.EventoRepository,
	itensPedido *repository.ItemPedidoRepository,
	pedidos *repository.PedidoRepository,
	pagamentos *repository.PagamentoRepository,
	lancamentos *repository.LancamentoRepository,
	config *repository.ConfigPlataformaRepository,
	mp *mercadopago.Cliente,
	mailCliente *mail.Cliente,
	qrSecret, frontendURL, backendURL, nomePlataforma string,
) *CheckoutService {
	return &CheckoutService{
		db: db, eventos: eventos, itensPedido: itensPedido, pedidos: pedidos,
		pagamentos: pagamentos, lancamentos: lancamentos, config: config,
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
	h := hmac.New(sha256.New, []byte(s.qrSecret))
	h.Write([]byte(codigo))
	return hex.EncodeToString(h.Sum(nil))
}

// calcularTotalItem é a fórmula da seção 7.1:
// total_item = preco_organizador + taxa_plataforma + garantia (se contratada).
// Função pura (sem I/O) para poder testar o cálculo isolado do resto do
// fluxo de reserva.
func calcularTotalItem(precoCentavos, taxaPlataformaCentavos, garantiaCentavos int64, garantiaContratada bool) int64 {
	total := precoCentavos + taxaPlataformaCentavos
	if garantiaContratada {
		total += garantiaCentavos
	}
	return total
}

// Reservar é a etapa "reserva + cálculo" da seção 8. Trava cada
// tipo_ingresso (SELECT ... FOR UPDATE) dentro de uma transação para
// impedir overselling sob concorrência (seção 7.2) — preço, taxa e
// garantia são sempre recalculados aqui, nunca aceitos do cliente.
func (s *CheckoutService) Reservar(usuarioID, eventoID int64, itensReq []ItemRequisitado, compradorNome, compradorEmail string) (*domain.Pedido, []domain.ItemPedido, error) {
	evento, err := s.eventos.BuscarPorID(eventoID)
	if err != nil {
		return nil, nil, fmt.Errorf("evento não encontrado: %w", err)
	}
	if evento.Status != domain.StatusEventoPublicado {
		return nil, nil, ErrEventoNaoDisponivelParaCompra
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

	chaveTaxa := domain.ChaveTaxaIngressoCentavos
	if evento.TipoAcesso == domain.TipoAcessoCadastro {
		chaveTaxa = domain.ChaveTaxaCadastroCentavos
	}
	taxaPlataforma, err := s.config.BuscarInt64(chaveTaxa)
	if err != nil {
		return nil, nil, fmt.Errorf("config de taxa ausente: %w", err)
	}
	reservaMinutos, err := s.config.BuscarInt64(domain.ChaveReservaMinutos)
	if err != nil {
		return nil, nil, fmt.Errorf("config de reserva ausente: %w", err)
	}
	garantiaCentavos, err := s.config.BuscarInt64(domain.ChaveGarantiaCentavos)
	if err != nil {
		return nil, nil, fmt.Errorf("config de garantia ausente: %w", err)
	}
	for _, ir := range itensReq {
		if ir.GarantiaContratada && !evento.GarantiaHabilitada {
			return nil, nil, ErrGarantiaNaoDisponivel
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
		if err := s.pedidos.Criar(tx, pedido); err != nil {
			return err
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
			if ir.Quantidade < tipo.MinPorPedido || (tipo.MaxPorPedido > 0 && ir.Quantidade > tipo.MaxPorPedido) {
				return fmt.Errorf("%w para %s", ErrQuantidadeInvalida, tipo.Nome)
			}

			ativos, err := s.itensPedido.ContarAtivosPorTipo(tx, tipo.ID)
			if err != nil {
				return err
			}
			disponivel := int64(tipo.Quantidade) - ativos
			if int64(ir.Quantidade) > disponivel {
				return fmt.Errorf("%w: %s (restam %d)", ErrEstoqueInsuficiente, tipo.Nome, disponivel)
			}

			itemGarantiaCentavos := int64(0)
			if ir.GarantiaContratada {
				itemGarantiaCentavos = garantiaCentavos
			}

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
					PrecoCentavos:          tipo.PrecoCentavos,
					TaxaPlataformaCentavos: taxaPlataforma,
					GarantiaContratada:     ir.GarantiaContratada,
					GarantiaCentavos:       itemGarantiaCentavos,
					TotalCentavos:          calcularTotalItem(tipo.PrecoCentavos, taxaPlataforma, itemGarantiaCentavos, ir.GarantiaContratada),
					Status:                 domain.StatusItemReservado,
					Codigo:                 codigo,
					QRToken:                s.gerarQRToken(codigo),
					CriadoEm:               agora,
				}
				if err := s.itensPedido.Criar(tx, &item); err != nil {
					return err
				}
				totalPedido += item.TotalCentavos
				itensCriados = append(itensCriados, item)
			}
		}

		pedido.TotalCentavos = totalPedido
		return tx.Save(pedido).Error
	})
	if err != nil {
		return nil, nil, err
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

// IniciarPagamento cria a preference no Mercado Pago (Checkout Pro, conta
// da plataforma — seção 2.3: sem split, sem marketplace_fee). Gerar a
// preference não move dinheiro; só o pagamento em si, feito pelo
// comprador na página do MP, move.
func (s *CheckoutService) IniciarPagamento(usuarioID, pedidoID int64) (string, error) {
	if !s.mp.Habilitado() {
		return "", ErrMercadoPagoNaoConfigurado
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
	mpItens := make([]mercadopago.ItemPreference, 0, len(itens))
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
		mpItens = append(mpItens, mercadopago.ItemPreference{
			Title:      tipo.Nome,
			Quantity:   1,
			UnitPrice:  float64(item.TotalCentavos) / 100,
			CurrencyID: "BRL",
		})
	}

	pedidoIDStr := strconv.FormatInt(pedido.ID, 10)
	pref, err := s.mp.CriarPreference(
		pedidoIDStr,
		mpItens,
		s.frontendURL+"/pedidos/"+pedidoIDStr+"?status=sucesso",
		s.frontendURL+"/pedidos/"+pedidoIDStr+"?status=falha",
		s.frontendURL+"/pedidos/"+pedidoIDStr+"?status=pendente",
		s.backendURL+"/api/v1/webhooks/mercadopago",
	)
	if err != nil {
		return "", err
	}

	pedido.MPPreferenceID = &pref.ID
	if err := s.pedidos.Salvar(pedido); err != nil {
		return "", err
	}

	if pref.InitPoint != "" {
		return pref.InitPoint, nil
	}
	return pref.SandboxInitPoint, nil
}

// ProcessarWebhook é idempotente por mp_payment_id (seção 7.3) e SEMPRE
// reconsulta o pagamento na API do MP — nunca confia em valor/status do
// corpo da notificação recebida.
func (s *CheckoutService) ProcessarWebhook(mpPaymentID string) error {
	if _, err := s.pagamentos.BuscarPorMPPaymentID(mpPaymentID); err == nil {
		return nil // já processado
	}

	pagamentoMP, err := s.mp.BuscarPagamento(mpPaymentID)
	if err != nil {
		return err
	}

	pedidoID, err := strconv.ParseInt(pagamentoMP.ExternalReference, 10, 64)
	if err != nil {
		return fmt.Errorf("external_reference inválida: %q", pagamentoMP.ExternalReference)
	}

	pedido, err := s.pedidos.BuscarPorID(pedidoID)
	if err != nil {
		return fmt.Errorf("pedido %d do webhook não encontrado: %w", pedidoID, err)
	}

	metodo := "cartao"
	if pagamentoMP.PaymentTypeID == "bank_transfer" || pagamentoMP.PaymentTypeID == "pix" {
		metodo = "pix"
	}

	payloadJSON, err := json.Marshal(pagamentoMP)
	if err != nil {
		return err
	}

	registro := &domain.Pagamento{
		PedidoID:                pedido.ID,
		MPPaymentID:             mpPaymentID,
		Metodo:                  metodo,
		Status:                  pagamentoMP.Status,
		ValorCentavos:           int64(pagamentoMP.TransactionAmount*100 + 0.5),
		TaxaProcessadorCentavos: pagamentoMP.TaxaProcessadorCentavos(),
		PayloadJSON:             datatypes.JSON(payloadJSON),
		CriadoEm:                time.Now(),
	}
	if err := s.pagamentos.Criar(registro); err != nil {
		// Corrida rara (dois webhooks quase simultâneos): índice único em
		// mp_payment_id barra a duplicata — trata como já processado.
		return nil
	}

	if pagamentoMP.Status != "approved" {
		return nil
	}

	return s.aprovarPedido(pedido)
}

func (s *CheckoutService) aprovarPedido(pedido *domain.Pedido) error {
	itens, err := s.itensPedido.ListarPorPedido(pedido.ID)
	if err != nil {
		return err
	}
	evento, err := s.eventos.BuscarPorID(pedido.EventoID)
	if err != nil {
		return err
	}

	agora := time.Now()
	var lancamentos []domain.Lancamento
	for i := range itens {
		item := &itens[i]
		if item.Status != domain.StatusItemReservado {
			continue // idempotência extra: já processado antes
		}
		item.Status = domain.StatusItemPago
		if err := s.itensPedido.Salvar(item); err != nil {
			return err
		}

		lancamentos = append(lancamentos,
			domain.Lancamento{
				Tipo: domain.LancamentoVendaPreco, ValorCentavos: item.PrecoCentavos, Sinal: "+",
				EventoID: &evento.ID, OrganizadorID: &evento.OrganizadorID, PedidoID: &pedido.ID, ItemID: &item.ID, CriadoEm: agora,
			},
			domain.Lancamento{
				Tipo: domain.LancamentoTaxaPlataforma, ValorCentavos: item.TaxaPlataformaCentavos, Sinal: "+",
				EventoID: &evento.ID, OrganizadorID: &evento.OrganizadorID, PedidoID: &pedido.ID, ItemID: &item.ID, CriadoEm: agora,
			},
		)
		if item.GarantiaContratada && item.GarantiaCentavos > 0 {
			lancamentos = append(lancamentos, domain.Lancamento{
				Tipo: domain.LancamentoGarantia, ValorCentavos: item.GarantiaCentavos, Sinal: "+",
				EventoID: &evento.ID, OrganizadorID: &evento.OrganizadorID, PedidoID: &pedido.ID, ItemID: &item.ID, CriadoEm: agora,
			})
		}
	}
	if err := s.lancamentos.CriarEmLote(lancamentos); err != nil {
		return err
	}

	pedido.Status = domain.StatusPedidoPago
	if err := s.pedidos.Salvar(pedido); err != nil {
		return err
	}

	s.enviarEmailConfirmacao(pedido, itens, evento)
	return nil
}

func (s *CheckoutService) enviarEmailConfirmacao(pedido *domain.Pedido, itens []domain.ItemPedido, evento *domain.Evento) {
	if len(itens) == 0 {
		return
	}
	destinatario := itens[0].TitularEmail
	if destinatario == "" {
		return
	}

	linkIngressos := fmt.Sprintf("%s/meus-ingressos", s.frontendURL)
	corpo := fmt.Sprintf(`<p>Olá!</p><p>Seu pagamento para <strong>%s</strong> foi aprovado.</p><ul>`, evento.Titulo)
	for _, item := range itens {
		corpo += fmt.Sprintf("<li>Código: <strong>%s</strong></li>", item.Codigo)
	}
	corpo += fmt.Sprintf(`</ul><p>Veja seus ingressos com QR code em <a href="%s">%s</a>.</p>`, linkIngressos, linkIngressos)

	_ = s.mailCliente.Enviar(destinatario, "Ingresso confirmado — "+s.nomePlataforma, corpo)
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
	for i := range pedidosExpirados {
		pedido := &pedidosExpirados[i]
		itens, err := s.itensPedido.ListarPorPedido(pedido.ID)
		if err != nil {
			return total, err
		}
		for j := range itens {
			item := &itens[j]
			if item.Status == domain.StatusItemReservado {
				item.Status = domain.StatusItemExpirado
				if err := s.itensPedido.Salvar(item); err != nil {
					return total, err
				}
			}
		}
		pedido.Status = domain.StatusPedidoExpirado
		if err := s.pedidos.Salvar(pedido); err != nil {
			return total, err
		}
		total++
	}
	return total, nil
}
