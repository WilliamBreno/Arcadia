package service

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"html"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/WilliamBreno/Arcadia/backend/internal/pagamento"
)

// Pagamento dentro da página do Evve (item 7.1): o QR do Pix aparece na
// tela e o cartão é pago pelo formulário seguro do Mercado Pago, sem sair
// do site. O Checkout Pro (redirecionar) continua como plano B.

// ValidadePix: o Mercado Pago não aceita Pix com menos de 30 minutos; um
// minuto de folga evita recusa por diferença de relógio. A reserva do
// pedido passa a terminar junto com o QR (decisão do dono, 29/10/2026).
const ValidadePix = 31 * time.Minute

var (
	ErrPagamentoNaoEncontradoNoPedido = errors.New("pagamento não pertence a este pedido")
	ErrDadosCartaoIncompletos         = errors.New("dados do cartão incompletos")
)

// PixNaTela é o que a página mostra: o QR (gerado a partir do "copia e
// cola") e até quando ele vale.
type PixNaTela struct {
	CopiaCola string    `json:"copia_cola"`
	ExpiraEm  time.Time `json:"expira_em"`
}

// DadosCartao vem do formulário do Mercado Pago: só o token (o número do
// cartão nunca chega ao Evve). O CPF do titular vai direto ao MP.
type DadosCartao struct {
	Token           string
	BandeiraID      string
	EmissorID       string
	DocumentoTipo   string
	DocumentoNumero string
}

// ResultadoCartao diz à página o que fazer depois de enviar o cartão.
type ResultadoCartao struct {
	Situacao    string `json:"situacao"` // aprovado, recusado, desafio, em_analise
	Mensagem    string `json:"mensagem,omitempty"`
	PagamentoID string `json:"pagamento_id,omitempty"`
	DesafioURL  string `json:"desafio_url,omitempty"`
	DesafioCreq string `json:"desafio_creq,omitempty"`
}

// pedidoPagavel confere dono, status e prazo antes de cobrar.
func (s *CheckoutService) pedidoPagavel(usuarioID, pedidoID int64) (*domain.Pedido, error) {
	if !s.mp.Habilitado() {
		return nil, ErrMercadoPagoNaoConfigurado
	}
	pedido, _, err := s.ObterPedido(usuarioID, pedidoID)
	if err != nil {
		return nil, err
	}
	if pedido.Status != domain.StatusPedidoAguardandoPagamento {
		return nil, ErrPedidoNaoDisponivelParaPagamento
	}
	if pedido.ExpiraEm != nil && pedido.ExpiraEm.Before(time.Now()) {
		return nil, ErrReservaExpirada
	}
	if pedido.TotalCentavos <= 0 {
		return nil, ErrPedidoNaoDisponivelParaPagamento
	}
	return pedido, nil
}

func (s *CheckoutService) emailPagador(usuarioID int64) string {
	if s.usuarios != nil {
		if u, err := s.usuarios.BuscarPorID(usuarioID); err == nil {
			return u.Email
		}
	}
	return ""
}

func (s *CheckoutService) descricaoCobranca(pedido *domain.Pedido) string {
	if e, err := s.eventos.BuscarPorID(pedido.EventoID); err == nil {
		return "Ingressos: " + e.Titulo
	}
	return fmt.Sprintf("Pedido #%d", pedido.ID)
}

// GerarPix cria (ou devolve, se ainda vale) o Pix do pedido. O pedido fica
// travado durante a criação: dois cliques não geram dois QR.
func (s *CheckoutService) GerarPix(usuarioID, pedidoID int64) (*PixNaTela, error) {
	if _, err := s.pedidoPagavel(usuarioID, pedidoID); err != nil {
		return nil, err
	}
	var resultado *PixNaTela
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var pedido domain.Pedido
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&pedido, pedidoID).Error; err != nil {
			return err
		}
		agora := time.Now()
		if pedido.Status != domain.StatusPedidoAguardandoPagamento {
			return ErrPedidoNaoDisponivelParaPagamento
		}
		if pedido.ExpiraEm != nil && pedido.ExpiraEm.Before(agora) {
			return ErrReservaExpirada
		}
		if pedido.PixCopiaCola != nil && pedido.PixExpiraEm != nil && pedido.PixExpiraEm.After(agora.Add(time.Minute)) {
			resultado = &PixNaTela{CopiaCola: *pedido.PixCopiaCola, ExpiraEm: *pedido.PixExpiraEm}
			return nil
		}
		ref := strconv.FormatInt(pedido.ID, 10)
		criado, err := s.mp.CriarPagamento(pagamento.NovoPagamento{
			Referencia:        ref,
			Descricao:         s.descricaoCobranca(&pedido),
			ValorCentavos:     pedido.TotalCentavos,
			Metodo:            pagamento.MetodoPix,
			EmailPagador:      s.emailPagador(pedido.UsuarioID),
			ExpiraEm:          agora.Add(ValidadePix),
			ChaveIdempotencia: "pix-" + ref + "-" + strconv.FormatInt(agora.Unix()/60, 10),
			URLNotificacao:    s.backendURL + "/api/v1/webhooks/mercadopago",
		})
		if err != nil {
			return err
		}
		if criado.PixCopiaCola == "" {
			return errors.New("o Mercado Pago não devolveu o código do Pix")
		}
		expira := criado.PixExpiraEm
		pedido.PixMPPaymentID = &criado.ID
		pedido.PixCopiaCola = &criado.PixCopiaCola
		pedido.PixExpiraEm = &expira
		// A reserva termina junto com o QR: assim ninguém paga um Pix de
		// ingresso que já voltou ao estoque.
		if pedido.ExpiraEm == nil || pedido.ExpiraEm.Before(expira) {
			pedido.ExpiraEm = &expira
		}
		if err := tx.Save(&pedido).Error; err != nil {
			return err
		}
		resultado = &PixNaTela{CopiaCola: criado.PixCopiaCola, ExpiraEm: expira}
		return nil
	})
	return resultado, err
}

// chaveCartao: o token do cartão é de uso único, então serve de base para
// a chave de idempotência (repetir o envio não cobra duas vezes).
func chaveCartao(pedidoID int64, token string) string {
	h := sha256.Sum256([]byte(token))
	return "cartao-" + strconv.FormatInt(pedidoID, 10) + "-" + hex.EncodeToString(h[:12])
}

// PagarCartao cobra o cartão com 3DS no modo "optional": se o banco pedir
// confirmação, a página abre a tela do banco e depois chama Verificar.
func (s *CheckoutService) PagarCartao(usuarioID, pedidoID int64, d DadosCartao) (*ResultadoCartao, error) {
	if s.cartaoBloqueado(usuarioID) {
		return nil, ErrCartaoBloqueado
	}
	if strings.TrimSpace(d.Token) == "" || strings.TrimSpace(d.BandeiraID) == "" {
		return nil, ErrDadosCartaoIncompletos
	}
	pedido, err := s.pedidoPagavel(usuarioID, pedidoID)
	if err != nil {
		return nil, err
	}
	ref := strconv.FormatInt(pedido.ID, 10)
	criado, err := s.mp.CriarPagamento(pagamento.NovoPagamento{
		Referencia:        ref,
		Descricao:         s.descricaoCobranca(pedido),
		ValorCentavos:     pedido.TotalCentavos,
		Metodo:            pagamento.MetodoCartao,
		EmailPagador:      s.emailPagador(usuarioID),
		CartaoToken:       d.Token,
		BandeiraID:        d.BandeiraID,
		EmissorID:         d.EmissorID,
		DocumentoTipo:     d.DocumentoTipo,
		DocumentoNumero:   d.DocumentoNumero,
		ChaveIdempotencia: chaveCartao(pedido.ID, d.Token),
		URLNotificacao:    s.backendURL + "/api/v1/webhooks/mercadopago",
	})
	if err != nil {
		slog.Warn("cartão: erro ao criar pagamento", "pedido_id", pedido.ID, "erro", err)
		return &ResultadoCartao{Situacao: "recusado", Mensagem: "Não foi possível processar o cartão agora. Tente de novo ou pague com Pix."}, nil
	}

	switch {
	case criado.Status == pagamento.StatusAprovado:
		if err := s.ProcessarWebhook(criado.ID); err != nil {
			return nil, err
		}
		s.cancelarPixPendente(pedido.ID)
		return &ResultadoCartao{Situacao: "aprovado", PagamentoID: criado.ID}, nil
	case criado.StatusDetalhe == pagamento.DetalheDesafio3DS:
		return &ResultadoCartao{Situacao: "desafio", PagamentoID: criado.ID, DesafioURL: criado.DesafioURL, DesafioCreq: criado.DesafioCreq}, nil
	case criado.Status == pagamento.StatusRecusado:
		_ = s.ProcessarWebhook(criado.ID) // registra a tentativa (histórico do antifraude)
		return &ResultadoCartao{Situacao: "recusado", PagamentoID: criado.ID, Mensagem: MensagemRecusa(criado.StatusDetalhe)}, nil
	default: // in_process / pending: o banco ainda vai responder; o webhook confirma
		_ = s.ProcessarWebhook(criado.ID)
		return &ResultadoCartao{Situacao: "em_analise", PagamentoID: criado.ID,
			Mensagem: "Pagamento em análise pelo banco. Avisamos por e-mail assim que for confirmado."}, nil
	}
}

// VerificarPagamento reconsulta um pagamento do pedido no Mercado Pago (depois
// do desafio do banco, ou quando o webhook atrasa). Nunca confia no que a
// página manda: só no que o MP responde, e só se o pagamento é deste pedido.
func (s *CheckoutService) VerificarPagamento(usuarioID, pedidoID int64, pagamentoID string) (*domain.Pedido, error) {
	pedido, _, err := s.ObterPedido(usuarioID, pedidoID)
	if err != nil {
		return nil, err
	}
	if pagamentoID == "" {
		if pedido.PixMPPaymentID == nil {
			return pedido, nil
		}
		pagamentoID = *pedido.PixMPPaymentID
	}
	p, err := s.mp.BuscarPagamento(pagamentoID)
	if err != nil {
		return nil, err
	}
	if p.Referencia != strconv.FormatInt(pedido.ID, 10) {
		return nil, ErrPagamentoNaoEncontradoNoPedido
	}
	if err := s.ProcessarWebhook(pagamentoID); err != nil {
		return nil, err
	}
	if p.Status == pagamento.StatusAprovado && p.Metodo == pagamento.MetodoCartao {
		s.cancelarPixPendente(pedido.ID)
	}
	return s.pedidos.BuscarPorID(pedido.ID)
}

// cancelarPixPendente invalida o QR depois que o cartão foi aprovado, para
// a pessoa não pagar duas vezes. Se ainda assim pagar, o segundo pagamento
// é devolvido sozinho (confirmarPagamento).
func (s *CheckoutService) cancelarPixPendente(pedidoID int64) {
	pedido, err := s.pedidos.BuscarPorID(pedidoID)
	if err != nil || pedido.PixMPPaymentID == nil {
		return
	}
	if err := s.mp.CancelarPagamento(*pedido.PixMPPaymentID); err != nil {
		slog.Info("pix: não foi possível cancelar (talvez já pago ou vencido)", "pedido_id", pedidoID, "erro", err)
	}
}

// MensagemRecusa traduz o motivo do Mercado Pago para o comprador. Testada.
func MensagemRecusa(detalhe string) string {
	switch detalhe {
	case "cc_rejected_insufficient_amount":
		return "Saldo ou limite insuficiente. Tente outro cartão ou pague com Pix."
	case "cc_rejected_bad_filled_security_code":
		return "Código de segurança incorreto. Confira o verso do cartão."
	case "cc_rejected_bad_filled_date":
		return "Data de validade incorreta. Confira o cartão."
	case "cc_rejected_bad_filled_card_number", "cc_rejected_bad_filled_other":
		return "Confira os dados do cartão e tente de novo."
	case "cc_rejected_call_for_authorize":
		return "Seu banco precisa autorizar este pagamento. Fale com o banco e tente de novo, ou pague com Pix."
	case "cc_rejected_card_disabled":
		return "Este cartão está desativado. Fale com o banco ou use outro cartão."
	case "cc_rejected_duplicated_payment":
		return "Você já fez um pagamento igual há pouco. Se precisar pagar de novo, use outro cartão ou o Pix."
	case "cc_rejected_max_attempts":
		return "Limite de tentativas atingido. Use outro cartão ou pague com Pix."
	case "cc_rejected_3ds_challenge", "cc_rejected_3ds_mandatory":
		return "O banco não confirmou o pagamento. Tente de novo ou pague com Pix."
	case "cc_rejected_high_risk", "cc_rejected_blacklist":
		return "Pagamento recusado por segurança. Use outro cartão ou pague com Pix."
	}
	return "O pagamento foi recusado pelo banco. Tente outro cartão ou pague com Pix."
}

// --- Confirmação com pagamento atrasado ou em dobro ---

// confirmarPagamento é o único caminho que transforma um pagamento aprovado
// em ingresso. Roda em transação com o pedido travado (o job que expira
// reservas também trava), então aprovação e expiração não se atropelam.
//
//   - pedido aguardando: emite os ingressos.
//   - pedido expirado (pagou depois da reserva): reemite se ainda há vaga
//     do mesmo ingresso; se não há, devolve tudo (decisão do dono).
//   - pedido que já não aguarda pagamento (pago por outro pagamento,
//     cancelado): devolve este pagamento a mais.
//   - valor pago menor que o pedido: devolve e não emite.
//
// registro é nil só no pedido de valor zero (não passa pelo processador).
func (s *CheckoutService) confirmarPagamento(pedidoID int64, registro *domain.Pagamento) error {
	var (
		pedido   domain.Pedido
		evento   *domain.Evento
		emitiu   bool
		devolver string
		vendidas []int64 // anúncios da revenda vendidos neste pagamento (7.4)
	)
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&pedido, pedidoID).Error; err != nil {
			return err
		}
		var err error
		if evento, err = s.eventos.BuscarPorID(pedido.EventoID); err != nil {
			return err
		}

		switch pedido.Status {
		case domain.StatusPedidoAguardandoPagamento:
		case domain.StatusPedidoExpirado:
			if registro == nil {
				return nil
			}
			if registro.ValorCentavos < pedido.TotalCentavos {
				devolver = domain.DevolucaoValorDivergente
				return s.pagamentos.MarcarDevolver(tx, registro.ID, devolver)
			}
			ok, err := s.reabrirReserva(tx, &pedido)
			if err != nil {
				return err
			}
			if !ok {
				devolver = domain.DevolucaoAposReserva
				return s.pagamentos.MarcarDevolver(tx, registro.ID, devolver)
			}
		default: // pago, cancelado, reembolsado...
			// Só chega aqui um pagamento aprovado pela primeira vez (as
			// notificações repetidas param antes, pelo índice único e pela
			// troca de status condicional). Então ele é um pagamento a mais.
			if registro == nil {
				return nil
			}
			devolver = domain.DevolucaoDuplicado
			return s.pagamentos.MarcarDevolver(tx, registro.ID, devolver)
		}

		if registro != nil && registro.ValorCentavos < pedido.TotalCentavos {
			devolver = domain.DevolucaoValorDivergente
			return s.pagamentos.MarcarDevolver(tx, registro.ID, devolver)
		}

		var itens []domain.ItemPedido
		if err := tx.Where("pedido_id = ?", pedido.ID).Find(&itens).Error; err != nil {
			return err
		}
		agora := time.Now()
		var lancamentos []domain.Lancamento
		for i := range itens {
			item := &itens[i]
			if item.Status != domain.StatusItemReservado {
				continue
			}
			item.Status = domain.StatusItemPago
			if err := tx.Save(item).Error; err != nil {
				return err
			}
			lancamentos = append(lancamentos, lancamentosDaVenda(item, evento, pedido.ID, agora)...)
		}
		if len(lancamentos) > 0 {
			if err := tx.Create(&lancamentos).Error; err != nil {
				return err
			}
		}
		pedido.Status = domain.StatusPedidoPago
		emitiu = true
		pagos := make([]int64, 0, len(itens))
		for _, it := range itens {
			if it.Status == domain.StatusItemPago {
				pagos = append(pagos, it.ID)
			}
		}
		var err2 error
		if vendidas, err2 = fecharVendas(tx, pagos); err2 != nil { // item 7.4
			return err2
		}
		return tx.Save(&pedido).Error
	})
	if err != nil {
		return err
	}
	if emitiu {
		if itens, err := s.itensPedido.ListarPorPedido(pedido.ID); err == nil {
			s.enviarEmailConfirmacao(&pedido, itens, evento)
		}
	}
	if len(vendidas) > 0 && s.revenda != nil {
		s.revenda.Liquidar(vendidas)
	}
	if devolver != "" {
		slog.Warn("pagamento será devolvido", "pedido_id", pedido.ID, "pagamento", registro.MPPaymentID, "motivo", devolver)
		s.devolver(registro.ID)
	}
	return nil
}

// reabrirReserva tenta pôr de volta os ingressos de um pedido expirado que
// acabou sendo pago: só se ainda há vaga de cada tipo (travado, como na
// compra). O preço é o da reserva original, que foi o que a pessoa pagou.
func (s *CheckoutService) reabrirReserva(tx *gorm.DB, pedido *domain.Pedido) (bool, error) {
	var itens []domain.ItemPedido
	if err := tx.Where("pedido_id = ? AND status = ?", pedido.ID, domain.StatusItemExpirado).Find(&itens).Error; err != nil {
		return false, err
	}
	if len(itens) == 0 {
		return false, nil
	}
	porTipo := map[int64]int64{}
	for _, it := range itens {
		porTipo[it.TipoIngressoID]++
	}
	daRevenda := map[int64]int{} // item 7.4: quantos vêm de anúncios da revenda
	for tipoID, precisa := range porTipo {
		tipo, err := s.itensPedido.BuscarTipoIngressoParaAtualizar(tx, tipoID)
		if err != nil {
			return false, err
		}
		ativos, err := s.itensPedido.ContarAtivosPorTipo(tx, tipoID)
		if err != nil {
			return false, err
		}
		guardadas, err := VagasGuardadas(tx, tipoID, pedido.UsuarioID)
		if err != nil {
			return false, err
		}
		aVenda, err := RevendasAVenda(tx, tipoID)
		if err != nil {
			return false, err
		}
		estoque := int64(tipo.Quantidade) - ativos - guardadas
		if VagasParaComprar(int64(tipo.Quantidade)-ativos, aVenda, guardadas) < precisa {
			return false, nil
		}
		daRevenda[tipoID] = QuantasDaRevenda(int(precisa), estoque)
	}
	for tipoID, n := range daRevenda {
		var ids []int64
		for _, it := range itens {
			if it.TipoIngressoID == tipoID && len(ids) < n {
				ids = append(ids, it.ID)
			}
		}
		if err := reivindicarRevendas(tx, tipoID, ids); err != nil {
			return false, nil
		}
	}
	for i := range itens {
		itens[i].Status = domain.StatusItemReservado
		if err := tx.Save(&itens[i]).Error; err != nil {
			return false, err
		}
		if itens[i].CupomID != nil {
			if err := s.cupons.AjustarUsos(tx, *itens[i].CupomID, 1); err != nil {
				return false, err
			}
		}
	}
	return true, nil
}

// devolver estorna o pagamento inteiro. A chave de idempotência impede
// estorno em dobro; se o MP falhar agora, o job tenta de novo
// (ReprocessarDevolucoes) até confirmar.
func (s *CheckoutService) devolver(pagamentoRegistroID int64) {
	var reg domain.Pagamento
	if err := s.db.First(&reg, pagamentoRegistroID).Error; err != nil || reg.DevolvidoEm != nil || reg.DevolucaoMotivo == nil {
		return
	}
	r, err := s.mp.Devolver(reg.MPPaymentID, "devolucao-"+reg.MPPaymentID)
	if err != nil {
		slog.Error("devolução falhou (o job tenta de novo)", "pagamento", reg.MPPaymentID, "erro", err)
		return
	}
	primeiro, err := s.pagamentos.MarcarDevolvido(reg.ID, r.ID)
	if err != nil || !primeiro {
		return
	}
	s.avisarDevolucao(&reg)
}

// ReprocessarDevolucoes roda no job de expirar reservas.
func (s *CheckoutService) ReprocessarDevolucoes() int {
	pendentes, err := s.pagamentos.ListarDevolucoesPendentes()
	if err != nil {
		return 0
	}
	for _, p := range pendentes {
		s.devolver(p.ID)
	}
	return len(pendentes)
}

// TextoDevolucao explica ao comprador por que o dinheiro voltou. Testada.
func TextoDevolucao(motivo string) string {
	switch motivo {
	case domain.DevolucaoAposReserva:
		return "O pagamento chegou depois do fim da reserva, e os ingressos já tinham sido vendidos para outra pessoa."
	case domain.DevolucaoDuplicado:
		return "Este pedido já estava pago quando este pagamento chegou. Os ingressos do primeiro pagamento continuam valendo, e o valor a mais foi devolvido."
	case domain.DevolucaoValorDivergente:
		return "O valor pago não correspondia ao valor do pedido, então nenhum ingresso foi emitido com ele."
	}
	return "O pagamento não pôde ser usado neste pedido."
}

func (s *CheckoutService) avisarDevolucao(reg *domain.Pagamento) {
	pedido, err := s.pedidos.BuscarPorID(reg.PedidoID)
	if err != nil {
		return
	}
	email := s.emailPagador(pedido.UsuarioID)
	if email == "" {
		return
	}
	titulo := ""
	if e, err := s.eventos.BuscarPorID(pedido.EventoID); err == nil {
		titulo = e.Titulo
	}
	motivo := ""
	if reg.DevolucaoMotivo != nil {
		motivo = *reg.DevolucaoMotivo
	}
	prazo := "O Pix volta para a sua conta em instantes."
	if reg.Metodo == pagamento.MetodoCartao {
		prazo = "No cartão, o estorno aparece na fatura em até duas faturas, conforme o banco."
	}
	corpo := fmt.Sprintf(`<div style="font-family:Arial,sans-serif;max-width:560px;margin:auto;color:#111"><p>Devolvemos <strong>%s</strong> do pedido #%d (%s).</p><p>%s</p><p>%s Você não precisa fazer nada.</p></div>`,
		formatarReais(reg.ValorCentavos), pedido.ID, html.EscapeString(titulo), html.EscapeString(TextoDevolucao(motivo)), prazo)
	_ = s.mailCliente.Enviar(email, "Pagamento devolvido — pedido #"+strconv.FormatInt(pedido.ID, 10)+" — "+s.nomePlataforma, corpo)
}
