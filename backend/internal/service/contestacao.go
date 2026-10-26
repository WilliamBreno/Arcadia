package service

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/gorm"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/WilliamBreno/Arcadia/backend/internal/pagamento"
	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
)

// Antifraude e contestação (item 5.3, decisões do dono em 25/10/2026):
//   - limite de ingressos por conta em cada evento (padrão 10);
//   - quem contesta uma compra no cartão fica sem cartão (Pix liberado);
//   - contestação vira registro + lançamento + alerta ao admin, sem
//     desconto automático do organizador (por isso o status do pagamento
//     não muda: o repasse só olha pagamentos "approved").

var (
	ErrCartaoBloqueado = errors.New("pagamento com cartão indisponível para esta conta por causa de uma contestação anterior — pague com Pix")
	ErrLimitePorConta  = errors.New("limite de ingressos por conta neste evento")
)

// Status do Mercado Pago que indicam contestação.
const (
	statusMPEmMediacao = "in_mediation"
	statusMPEstornado  = "charged_back"
)

// statusContestacao traduz o status do processador. Vazio = não é contestação.
func statusContestacao(statusProcessador string) string {
	switch statusProcessador {
	case statusMPEmMediacao:
		return domain.ContestacaoEmDisputa
	case statusMPEstornado:
		return domain.ContestacaoPerdida
	}
	return ""
}

// DefinirAntifraude liga o limite por conta, o bloqueio de cartão e o
// registro de contestações.
func (s *CheckoutService) DefinirAntifraude(usuarios *repository.UsuarioRepository, contestacoes *repository.ContestacaoRepository) {
	s.usuarios, s.contestacoes = usuarios, contestacoes
}

// limitePorConta: erro se o pedido passar do máximo. Função pura, testada.
func limitePorConta(maximo int, jaTem int64, pedidos int) error {
	if maximo <= 0 || jaTem+int64(pedidos) <= int64(maximo) {
		return nil
	}
	restam := int64(maximo) - jaTem
	if restam < 0 {
		restam = 0
	}
	return fmt.Errorf("%w: são no máximo %d por conta e você ainda pode reservar %d", ErrLimitePorConta, maximo, restam)
}

// checarLimitePorConta roda dentro da transação da reserva, com a linha
// do usuário travada (duas abas comprando ao mesmo tempo não furam o limite).
func (s *CheckoutService) checarLimitePorConta(tx *gorm.DB, evento *domain.Evento, usuarioID int64, unidades int) error {
	if s.usuarios == nil || evento.MaxIngressosPorConta <= 0 {
		return nil
	}
	if err := s.usuarios.TravarParaCompra(tx, usuarioID); err != nil {
		return err
	}
	jaTem, err := s.itensPedido.ContarDoUsuarioNoEvento(tx, usuarioID, evento.ID)
	if err != nil {
		return err
	}
	return limitePorConta(evento.MaxIngressosPorConta, jaTem, unidades)
}

func (s *CheckoutService) cartaoBloqueado(usuarioID int64) bool {
	if s.usuarios == nil {
		return false
	}
	u, err := s.usuarios.BuscarPorID(usuarioID)
	return err == nil && u.CartaoBloqueadoEm != nil
}

// atualizarPagamento trata a notificação de um pagamento já registrado.
func (s *CheckoutService) atualizarPagamento(registro *domain.Pagamento, p *pagamento.Pagamento) error {
	pedido, err := s.pedidos.BuscarPorID(registro.PedidoID)
	if err != nil {
		return err
	}
	if statusContestacao(p.Status) != "" {
		return s.registrarContestacao(registro, pedido, p.Status)
	}
	if p.Status != pagamento.StatusAprovado {
		return nil
	}
	if registro.Status == pagamento.StatusAprovado {
		// Disputa resolvida a nosso favor: o MP volta o pagamento para approved.
		return s.resolverContestacao(registro)
	}
	// Pendente → aprovado (ex.: Pix compensado). O UPDATE condicional
	// garante que só uma notificação aprova o pedido.
	mudou, err := s.pagamentos.AtualizarStatusSe(registro.ID, registro.Status, pagamento.StatusAprovado, p.ValorCentavos, p.TaxaProcessadorCentavos)
	if err != nil || !mudou {
		return err
	}
	return s.aprovarPedido(pedido)
}

// registrarContestacao é idempotente por pagamento: cria o registro na
// primeira notificação e só avança o status (em disputa → perdida).
func (s *CheckoutService) registrarContestacao(registro *domain.Pagamento, pedido *domain.Pedido, statusProcessador string) error {
	if s.contestacoes == nil || s.usuarios == nil {
		return nil
	}
	novo := statusContestacao(statusProcessador)
	evento, err := s.eventos.BuscarPorID(pedido.EventoID)
	if err != nil {
		return err
	}

	alertar := false
	err = s.db.Transaction(func(tx *gorm.DB) error {
		c, err := s.contestacoes.BuscarPorPagamento(tx, registro.ID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c = &domain.Contestacao{
				PagamentoID: registro.ID, PedidoID: pedido.ID, EventoID: evento.ID, UsuarioID: pedido.UsuarioID,
				ValorCentavos: registro.ValorCentavos, CriadoEm: time.Now(),
			}
		} else if err != nil {
			return err
		}
		if c.Status == novo || c.Status == domain.ContestacaoPerdida {
			return nil // nada novo (perdida é final)
		}
		c.Status, c.StatusProcessador = novo, statusProcessador
		if err := s.contestacoes.Salvar(tx, c); err != nil {
			return err
		}
		if err := s.usuarios.BloquearCartao(tx, pedido.UsuarioID); err != nil {
			return err
		}
		if novo == domain.ContestacaoPerdida {
			if err := tx.Create(&domain.Lancamento{
				Tipo: domain.LancamentoChargeback, ValorCentavos: registro.ValorCentavos, Sinal: "-",
				EventoID: &evento.ID, OrganizadorID: &evento.OrganizadorID, PedidoID: &pedido.ID, CriadoEm: time.Now(),
			}).Error; err != nil {
				return err
			}
		}
		alertar = true
		return nil
	})
	if err != nil || !alertar {
		return err
	}
	s.alertarAdminsContestacao(evento, pedido, registro, novo)
	return nil
}

func (s *CheckoutService) resolverContestacao(registro *domain.Pagamento) error {
	if s.contestacoes == nil {
		return nil
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		c, err := s.contestacoes.BuscarPorPagamento(tx, registro.ID)
		if err != nil || c.Status != domain.ContestacaoEmDisputa {
			return nil
		}
		c.Status, c.StatusProcessador = domain.ContestacaoResolvida, pagamento.StatusAprovado
		return s.contestacoes.Salvar(tx, c)
	})
}

func (s *CheckoutService) alertarAdminsContestacao(evento *domain.Evento, pedido *domain.Pedido, registro *domain.Pagamento, status string) {
	emails, err := s.usuarios.EmailsAdmins()
	if err != nil || len(emails) == 0 {
		slog.Warn("contestação sem admin para alertar", "pedido_id", pedido.ID, "status", status)
		return
	}
	situacao := "Uma contestação foi aberta"
	if status == domain.ContestacaoPerdida {
		situacao = "Uma contestação foi perdida (o banco devolveu o valor ao comprador)"
	}
	corpo := fmt.Sprintf(
		`<p>%s no pagamento %s do pedido #%d, evento <strong>%s</strong>.</p><p>Valor: R$ %.2f. O comprador não pode mais pagar com cartão. Nada foi descontado do organizador automaticamente — decida no painel admin (Contestações).</p>`,
		situacao, registro.MPPaymentID, pedido.ID, evento.Titulo, float64(registro.ValorCentavos)/100,
	)
	for _, e := range emails {
		_ = s.mailCliente.Enviar(e, "Contestação de pagamento — "+s.nomePlataforma, corpo)
	}
}
