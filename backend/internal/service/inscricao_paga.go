package service

// Item 8.4: inscrição paga de talento. Decisões do dono (30/10/2026):
//   - paga depois da aprovação, com 48 h de prazo (quem é rejeitado nunca paga);
//   - mesma taxa do ingresso, sempre paga pelo participante;
//   - mesma garantia opcional do ingresso (desistir até o início = tudo de volta);
//   - organizador remove quem pagou, ou cancela o evento: reembolso integral.
//
// O pagamento usa o mesmo pedido dos ingressos (Pix/cartão, devolução,
// repasse, ledger). O item fica num "ingresso interno" inativo do evento e
// aponta para a ficha (itens_pedido.ficha_id): nunca vira ingresso de
// plateia, não entra no check-in nem em "Meus ingressos".

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
)

var (
	ErrInscricaoSemPagamento = errors.New("esta inscrição não tem pagamento pendente")
	ErrPrazoInscricaoVencido = errors.New("o prazo para pagar a inscrição terminou")
)

// PrazoPagamentoInscricao é quanto tempo a pessoa aprovada tem para pagar.
const PrazoPagamentoInscricao = 48 * time.Hour

// AvisoInscricao manda os e-mails da ficha (implementado pelo FichaService).
type AvisoInscricao interface {
	AvisarInscricaoPaga(fichaID int64)
	AvisarPrazoVencido(fichaID int64)
}

// DefinirAvisoInscricao liga os e-mails da inscrição paga.
func (s *CheckoutService) DefinirAvisoInscricao(a AvisoInscricao) { s.avisoInscricao = a }

// PrazoParaPagar: 48 h a partir de agora, sem passar do início do evento.
func PrazoParaPagar(agora time.Time, inicio *time.Time) time.Time {
	prazo := agora.Add(PrazoPagamentoInscricao)
	if inicio != nil && inicio.Before(prazo) {
		return *inicio
	}
	return prazo
}

// ValoresInscricao é o resumo que a pessoa vê antes de pagar.
type ValoresInscricao struct {
	PrecoCentavos      int64
	TaxaCentavos       int64
	GarantiaCentavos   int64 // quanto custa a garantia, se ela marcar
	GarantiaDisponivel bool
	PrazoEm            *time.Time
	PedidoAbertoID     *int64 // pagamento já iniciado e ainda dentro da reserva
	// Já pago: o item serve para desistir (cancelamento com as regras da garantia).
	Pago               bool
	ItemPagoID         *int64
	TotalPagoCentavos  int64
	GarantiaContratada bool
}

func (s *CheckoutService) taxasInscricao(evento *domain.Evento) (TaxasEvento, error) {
	t, err := s.taxas.ParaEvento(evento)
	if err != nil {
		return t, err
	}
	t.Absorvida = false // decisão do dono: na inscrição, quem paga a taxa é o participante
	return t, nil
}

func garantiaInscricaoDisponivel(evento *domain.Evento, agora time.Time) bool {
	return evento.GarantiaHabilitada && (evento.InicioEm == nil || agora.Before(*evento.InicioEm))
}

// fichaParaPagar acha a ficha do participante que está esperando o pagamento.
func fichaParaPagar(tx *gorm.DB, eventoID, usuarioID int64, travar bool) (*domain.FichaParticipacao, error) {
	q := tx
	if travar {
		q = q.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var f domain.FichaParticipacao
	if err := q.Where("evento_id = ? AND usuario_id = ? AND papel = ?", eventoID, usuarioID, domain.PapelParticipante).
		First(&f).Error; err != nil {
		return nil, ErrInscricaoSemPagamento
	}
	if f.Status != domain.StatusFichaAguardandoPagamento {
		return nil, ErrInscricaoSemPagamento
	}
	return &f, nil
}

// pedidoAbertoDaFicha: pedido aguardando pagamento, dentro da reserva, com
// o item desta ficha.
func pedidoAbertoDaFicha(tx *gorm.DB, fichaID int64, agora time.Time) (*domain.Pedido, error) {
	var p domain.Pedido
	err := tx.Where(`status = ? AND expira_em > ? AND id IN
		(SELECT pedido_id FROM itens_pedido WHERE ficha_id = ? AND status = ?)`,
		domain.StatusPedidoAguardandoPagamento, agora, fichaID, domain.StatusItemReservado).
		Order("id DESC").First(&p).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// ValoresInscricao mostra preço, taxa e garantia antes de pagar.
func (s *CheckoutService) ValoresInscricao(usuarioID, eventoID int64) (*ValoresInscricao, error) {
	evento, err := s.eventos.BuscarPorID(eventoID)
	if err != nil {
		return nil, ErrInscricaoSemPagamento
	}
	// Já paga: mostra o que foi pago (e o item, para desistir).
	var pago domain.ItemPedido
	if err := s.db.Joins("JOIN fichas_participacao f ON f.id = itens_pedido.ficha_id").
		Where("f.evento_id = ? AND f.usuario_id = ? AND f.papel = ? AND itens_pedido.status = ?", eventoID, usuarioID, domain.PapelParticipante, domain.StatusItemPago).
		First(&pago).Error; err == nil {
		id := pago.ID
		return &ValoresInscricao{Pago: true, ItemPagoID: &id, PrecoCentavos: pago.PrecoCentavos, TaxaCentavos: pago.TaxaPlataformaCentavos,
			GarantiaCentavos: pago.GarantiaCentavos, TotalPagoCentavos: pago.TotalCentavos, GarantiaContratada: pago.GarantiaContratada}, nil
	}
	ficha, err := fichaParaPagar(s.db, eventoID, usuarioID, false)
	if err != nil {
		return nil, err
	}
	if evento.InscricaoTalentoCentavos <= 0 {
		return nil, ErrInscricaoSemPagamento
	}
	t, err := s.taxasInscricao(evento)
	if err != nil {
		return nil, err
	}
	agora := time.Now()
	semGarantia := t.Calcular(evento.InscricaoTalentoCentavos, false)
	comGarantia := t.Calcular(evento.InscricaoTalentoCentavos, true)
	v := &ValoresInscricao{
		PrecoCentavos:      evento.InscricaoTalentoCentavos,
		TaxaCentavos:       semGarantia.TaxaCentavos,
		GarantiaCentavos:   comGarantia.GarantiaCentavos,
		GarantiaDisponivel: garantiaInscricaoDisponivel(evento, agora),
		PrazoEm:            ficha.PrazoPagamentoEm,
	}
	if p, err := pedidoAbertoDaFicha(s.db, ficha.ID, agora); err == nil {
		v.PedidoAbertoID = &p.ID
	}
	return v, nil
}

// tipoInscricao devolve (ou cria) o ingresso interno da inscrição do evento.
func tipoInscricao(tx *gorm.DB, eventoID int64) (*domain.TipoIngresso, error) {
	err := tx.Exec(`INSERT INTO tipos_ingresso (evento_id, nome, descricao, preco_centavos, quantidade, min_por_pedido, max_por_pedido, ordem, ativo, inscricao_talento, criado_em)
		VALUES (?, 'Inscrição de talento', 'Uso interno: inscrição paga de participante (item 8.4).', 0, 1000000, 1, 1, 9999, false, true, now())
		ON CONFLICT (evento_id) WHERE inscricao_talento DO NOTHING`, eventoID).Error
	if err != nil {
		return nil, err
	}
	var t domain.TipoIngresso
	if err := tx.Where("evento_id = ? AND inscricao_talento", eventoID).First(&t).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

// ReservarInscricao abre o pedido da inscrição (mesma reserva dos
// ingressos). Se já existe um pagamento em andamento com a mesma escolha
// de garantia, devolve esse; com escolha diferente, troca por um novo.
func (s *CheckoutService) ReservarInscricao(usuarioID, eventoID int64, garantia bool) (*domain.Pedido, []domain.ItemPedido, error) {
	evento, err := s.eventos.BuscarPorID(eventoID)
	if err != nil {
		return nil, nil, ErrInscricaoSemPagamento
	}
	if evento.Status != domain.StatusEventoPublicado || evento.InscricaoTalentoCentavos <= 0 {
		return nil, nil, ErrInscricaoSemPagamento
	}
	agora := time.Now()
	if garantia && !garantiaInscricaoDisponivel(evento, agora) {
		return nil, nil, ErrGarantiaNaoDisponivel
	}
	t, err := s.taxasInscricao(evento)
	if err != nil {
		return nil, nil, err
	}
	reservaMinutos, err := s.config.BuscarInt64(domain.ChaveReservaMinutos)
	if err != nil {
		return nil, nil, fmt.Errorf("config de reserva ausente: %w", err)
	}
	var email string
	if u, err := s.usuarios.BuscarPorID(usuarioID); err == nil {
		email = u.Email
	}

	var pedido *domain.Pedido
	var itens []domain.ItemPedido
	var trocado int64
	err = s.db.Transaction(func(tx *gorm.DB) error {
		ficha, err := fichaParaPagar(tx, eventoID, usuarioID, true)
		if err != nil {
			return err
		}
		if ficha.PrazoPagamentoEm != nil && agora.After(*ficha.PrazoPagamentoEm) {
			return ErrPrazoInscricaoVencido
		}
		if aberto, err := pedidoAbertoDaFicha(tx, ficha.ID, agora); err == nil {
			var item domain.ItemPedido
			if err := tx.Where("pedido_id = ? AND ficha_id = ?", aberto.ID, ficha.ID).First(&item).Error; err != nil {
				return err
			}
			if item.GarantiaContratada == garantia {
				pedido, itens = aberto, []domain.ItemPedido{item}
				return nil
			}
			// Outra escolha de garantia: o pedido antigo sai de cena. Se o
			// Pix dele ainda for pago, a confirmação devolve (a ficha já
			// estará paga pelo novo, ou ele cai como pagamento a mais).
			if err := tx.Model(&domain.ItemPedido{}).Where("pedido_id = ? AND status = ?", aberto.ID, domain.StatusItemReservado).
				Update("status", domain.StatusItemExpirado).Error; err != nil {
				return err
			}
			if err := tx.Model(&domain.Pedido{}).Where("id = ?", aberto.ID).
				Update("status", domain.StatusPedidoExpirado).Error; err != nil {
				return err
			}
			trocado = aberto.ID
		}

		tipo, err := tipoInscricao(tx, eventoID)
		if err != nil {
			return err
		}
		expira := agora.Add(time.Duration(reservaMinutos) * time.Minute)
		pedido = &domain.Pedido{UsuarioID: usuarioID, EventoID: eventoID, Status: domain.StatusPedidoAguardandoPagamento, ExpiraEm: &expira, CriadoEm: agora}
		if err := s.pedidos.Criar(tx, pedido); err != nil {
			return err
		}
		valores := t.Calcular(evento.InscricaoTalentoCentavos, garantia)
		codigo, err := gerarCodigoItem()
		if err != nil {
			return err
		}
		nome := ficha.NomeArtistico
		if nome == "" {
			nome = ficha.Nome
		}
		fichaID := ficha.ID
		item := domain.ItemPedido{
			PedidoID: pedido.ID, TipoIngressoID: tipo.ID, FichaID: &fichaID,
			TitularNome: nome, TitularEmail: email,
			PrecoCentavos: evento.InscricaoTalentoCentavos, TaxaPlataformaCentavos: valores.TaxaCentavos,
			GarantiaContratada: valores.GarantiaCentavos > 0, GarantiaCentavos: valores.GarantiaCentavos,
			TotalCentavos: valores.TotalCentavos, Status: domain.StatusItemReservado,
			Codigo: codigo, QRToken: s.gerarQRToken(codigo), CriadoEm: agora,
		}
		if err := s.itensPedido.Criar(tx, &item); err != nil {
			return err
		}
		itens = []domain.ItemPedido{item}
		pedido.TotalCentavos = item.TotalCentavos
		return tx.Save(pedido).Error
	})
	if err != nil {
		return nil, nil, err
	}
	if trocado != 0 {
		s.cancelarPixPendente(trocado)
	}
	return pedido, itens, nil
}

// fichasDoPedido trava as fichas pagas por este pedido e diz se todas
// ainda estão esperando o pagamento.
func fichasDoPedido(tx *gorm.DB, pedidoID int64) (ids []int64, esperando bool, err error) {
	if err = tx.Model(&domain.ItemPedido{}).Where("pedido_id = ? AND ficha_id IS NOT NULL", pedidoID).
		Distinct().Pluck("ficha_id", &ids).Error; err != nil || len(ids) == 0 {
		return nil, true, err
	}
	var fichas []domain.FichaParticipacao
	if err = tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id IN ?", ids).Find(&fichas).Error; err != nil {
		return nil, false, err
	}
	if len(fichas) != len(ids) {
		return ids, false, nil
	}
	for _, f := range fichas {
		if f.Status != domain.StatusFichaAguardandoPagamento {
			return ids, false, nil
		}
	}
	return ids, true, nil
}

// confirmarFichasPagas aprova as fichas e confirma o papel de participante.
func confirmarFichasPagas(tx *gorm.DB, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	if err := tx.Model(&domain.FichaParticipacao{}).Where("id IN ?", ids).
		Updates(map[string]any{"status": domain.StatusFichaAprovado, "prazo_pagamento_em": nil, "motivo_rejeicao": ""}).Error; err != nil {
		return err
	}
	return tx.Exec(`INSERT INTO papeis_evento (evento_id, usuario_id, papel, origem, status, criado_em)
		SELECT evento_id, usuario_id, papel, origem, 'confirmado', now() FROM fichas_participacao WHERE id IN ?
		ON CONFLICT (evento_id, usuario_id, papel) DO UPDATE SET status = 'confirmado'`, ids).Error
}

// ExpirarInscricoesNaoPagas: quem não pagou no prazo perde a vaga. Não
// expira enquanto houver um pagamento em andamento dentro da reserva.
func (s *CheckoutService) ExpirarInscricoesNaoPagas() (int, error) {
	agora := time.Now()
	var ids []int64
	err := s.db.Model(&domain.FichaParticipacao{}).
		Where("status = ? AND prazo_pagamento_em < ?", domain.StatusFichaAguardandoPagamento, agora).
		Where(`NOT EXISTS (SELECT 1 FROM itens_pedido i JOIN pedidos p ON p.id = i.pedido_id
			WHERE i.ficha_id = fichas_participacao.id AND p.status = ? AND p.expira_em > ?)`, domain.StatusPedidoAguardandoPagamento, agora).
		Pluck("id", &ids).Error
	if err != nil {
		return 0, err
	}
	total := 0
	for _, id := range ids {
		res := s.db.Model(&domain.FichaParticipacao{}).
			Where("id = ? AND status = ?", id, domain.StatusFichaAguardandoPagamento).
			Updates(map[string]any{"status": domain.StatusFichaRejeitado, "motivo_rejeicao": "O prazo para pagar a inscrição terminou."})
		if res.Error != nil {
			return total, res.Error
		}
		if res.RowsAffected == 1 {
			total++
			if s.avisoInscricao != nil {
				s.avisoInscricao.AvisarPrazoVencido(id)
			}
		}
	}
	return total, nil
}
