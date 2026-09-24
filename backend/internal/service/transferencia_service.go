package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/WilliamBreno/Arcadia/backend/internal/mail"
	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
)

var (
	ErrTransferenciaNaoPermitida = errors.New("transferência não permitida")
	ErrTransferenciaDados        = errors.New("informe nome e e-mail válidos do novo titular")
)

// TransferenciaService troca o titular NOMINAL de um ingresso (seção 2.6).
// Quem comprou continua dono do pedido (reembolso e "Meus ingressos" seguem
// com ele); o que muda é quem entra no evento. O código e o QR são
// regenerados: o QR antigo deixa de valer (quem transfere não pode
// continuar usando uma cópia).
type TransferenciaService struct {
	db          *gorm.DB
	itensPedido *repository.ItemPedidoRepository
	pedidos     *repository.PedidoRepository
	eventos     *repository.EventoRepository
	mailCliente *mail.Cliente
	qrSecret    string
	frontendURL string
	plataforma  string
}

func NovoTransferenciaService(
	db *gorm.DB, itensPedido *repository.ItemPedidoRepository, pedidos *repository.PedidoRepository,
	eventos *repository.EventoRepository, mailCliente *mail.Cliente, qrSecret, frontendURL, plataforma string,
) *TransferenciaService {
	return &TransferenciaService{db: db, itensPedido: itensPedido, pedidos: pedidos, eventos: eventos,
		mailCliente: mailCliente, qrSecret: qrSecret, frontendURL: frontendURL, plataforma: plataforma}
}

// podeTransferir: item pago (não utilizado/cancelado), evento não cancelado
// e ainda não iniciado. Função pura para testar isolada.
func podeTransferir(item *domain.ItemPedido, evento *domain.Evento, agora time.Time) error {
	if item.Status != domain.StatusItemPago {
		return fmt.Errorf("%w: só ingressos pagos e ainda não utilizados", ErrTransferenciaNaoPermitida)
	}
	if evento.Status == domain.StatusEventoCancelado {
		return fmt.Errorf("%w: evento cancelado", ErrTransferenciaNaoPermitida)
	}
	if evento.InicioEm != nil && !agora.Before(*evento.InicioEm) {
		return fmt.Errorf("%w: o evento já começou", ErrTransferenciaNaoPermitida)
	}
	return nil
}

func (s *TransferenciaService) Transferir(itemID, usuarioID int64, nome, email string) (*domain.ItemPedido, error) {
	nome, email = strings.TrimSpace(nome), strings.TrimSpace(email)
	if len(nome) < 2 || !strings.Contains(email, "@") {
		return nil, ErrTransferenciaDados
	}

	item, err := s.itensPedido.BuscarPorID(itemID)
	if err != nil {
		return nil, ErrItemNaoEncontrado
	}
	pedido, err := s.pedidos.BuscarPorID(item.PedidoID)
	if err != nil {
		return nil, ErrItemNaoEncontrado
	}
	if pedido.UsuarioID != usuarioID || item.Cortesia {
		return nil, ErrItemNaoPertenceAoUsuario
	}
	evento, err := s.eventos.BuscarPorID(pedido.EventoID)
	if err != nil {
		return nil, ErrItemNaoEncontrado
	}
	if err := podeTransferir(item, evento, time.Now()); err != nil {
		return nil, err
	}

	novoCodigo, err := gerarCodigoItem()
	if err != nil {
		return nil, err
	}
	anterior := *item

	err = s.db.Transaction(func(tx *gorm.DB) error {
		// UPDATE condicional: se o check-in ou um cancelamento mudou o status
		// no meio do caminho, não transfere.
		res := tx.Model(&domain.ItemPedido{}).
			Where("id = ? AND status = ? AND codigo = ?", item.ID, domain.StatusItemPago, anterior.Codigo).
			Updates(map[string]any{
				"titular_nome": nome, "titular_email": email,
				"codigo": novoCodigo, "qr_token": assinarQR(s.qrSecret, novoCodigo),
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return fmt.Errorf("%w: o ingresso mudou de estado, tente de novo", ErrTransferenciaNaoPermitida)
		}
		return tx.Exec(`INSERT INTO transferencias_ingresso
			(item_id, feito_por, de_nome, de_email, para_nome, para_email, codigo_anterior) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			item.ID, usuarioID, anterior.TitularNome, anterior.TitularEmail, nome, email, anterior.Codigo).Error
	})
	if err != nil {
		return nil, err
	}

	atualizado, err := s.itensPedido.BuscarPorID(item.ID)
	if err != nil {
		return nil, err
	}
	s.notificar(evento, &anterior, atualizado)
	return atualizado, nil
}

func (s *TransferenciaService) notificar(evento *domain.Evento, anterior, novo *domain.ItemPedido) {
	if novo.TitularEmail != "" {
		corpo, imagens := corpoEmailIngressos("Olá, "+novo.TitularNome+"!", "Um ingresso foi transferido para você.", evento, []domain.ItemPedido{*novo}, s.qrSecret, s.frontendURL)
		_ = s.mailCliente.EnviarComImagens(novo.TitularEmail, "Ingresso recebido — "+evento.Titulo+" — "+s.plataforma, corpo, imagens)
	}
	if anterior.TitularEmail != "" && anterior.TitularEmail != novo.TitularEmail {
		corpo := fmt.Sprintf(`<p>Olá, %s!</p><p>Seu ingresso (código %s) para <strong>%s</strong> foi transferido para outra pessoa. O QR antigo não vale mais.</p>`,
			anterior.TitularNome, anterior.Codigo, evento.Titulo)
		_ = s.mailCliente.Enviar(anterior.TitularEmail, "Ingresso transferido — "+s.plataforma, corpo)
	}
}
