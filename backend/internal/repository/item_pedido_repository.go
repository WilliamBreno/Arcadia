package repository

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
)

type ItemPedidoRepository struct {
	db *gorm.DB
}

func NovoItemPedidoRepository(db *gorm.DB) *ItemPedidoRepository {
	return &ItemPedidoRepository{db: db}
}

func (r *ItemPedidoRepository) Criar(tx *gorm.DB, item *domain.ItemPedido) error {
	return tx.Create(item).Error
}

func (r *ItemPedidoRepository) Salvar(item *domain.ItemPedido) error {
	return r.db.Save(item).Error
}

func (r *ItemPedidoRepository) BuscarPorID(id int64) (*domain.ItemPedido, error) {
	var item domain.ItemPedido
	if err := r.db.First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *ItemPedidoRepository) ListarPorPedido(pedidoID int64) ([]domain.ItemPedido, error) {
	var itens []domain.ItemPedido
	err := r.db.Where("pedido_id = ?", pedidoID).Find(&itens).Error
	return itens, err
}

// ContarAtivosPorTipo conta itens que ocupam vaga (reservados ainda
// dentro do prazo + pagos + utilizados) — usado para checar
// disponibilidade de estoque de forma atômica (seção 7.2).
// A trava do tipo_ingresso deve ser feita antes, via BuscarTipoParaAtualizar.
func (r *ItemPedidoRepository) ContarAtivosPorTipo(tx *gorm.DB, tipoIngressoID int64) (int64, error) {
	var total int64
	err := tx.Model(&domain.ItemPedido{}).
		Where("tipo_ingresso_id = ? AND status IN ?", tipoIngressoID, []domain.StatusItemPedido{
			domain.StatusItemReservado, domain.StatusItemPago, domain.StatusItemUtilizado,
		}).
		Count(&total).Error
	return total, err
}

// BuscarTipoIngressoParaAtualizar trava a linha do tipo_ingresso
// (SELECT ... FOR UPDATE) para impedir overselling sob concorrência.
func (r *ItemPedidoRepository) BuscarTipoIngressoParaAtualizar(tx *gorm.DB, id int64) (*domain.TipoIngresso, error) {
	var tipo domain.TipoIngresso
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&tipo, id).Error
	if err != nil {
		return nil, err
	}
	return &tipo, nil
}

func (r *ItemPedidoRepository) BuscarPorCodigo(codigo string) (*domain.ItemPedido, error) {
	var item domain.ItemPedido
	if err := r.db.First(&item, "codigo = ?", codigo).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

// MarcarUtilizadoAtomico é o check-in de verdade (seção 7.10):
// "UPDATE itens SET status='utilizado' WHERE id=? AND status='pago'" —
// só um scanner "vence" se dois lerem o mesmo QR ao mesmo tempo.
// RowsAffected=0 quer dizer que outra requisição já fez o check-in
// primeiro (ou o item não estava mais "pago").
func (r *ItemPedidoRepository) MarcarUtilizadoAtomico(itemID int64) (bool, error) {
	agora := time.Now()
	resultado := r.db.Model(&domain.ItemPedido{}).
		Where("id = ? AND status = ?", itemID, domain.StatusItemPago).
		Updates(map[string]any{"status": domain.StatusItemUtilizado, "utilizado_em": agora})
	if resultado.Error != nil {
		return false, resultado.Error
	}
	return resultado.RowsAffected > 0, nil
}

// ItemComTipo inclui o nome do tipo de ingresso — usado na busca manual
// do check-in (seção 7.10), pra portaria ver o que a pessoa comprou.
type ItemComTipo struct {
	domain.ItemPedido
	TipoIngressoNome string
}

// BuscarPorEventoEBusca é GET /checkin/eventos/:id/busca — por nome,
// e-mail ou código do titular.
func (r *ItemPedidoRepository) BuscarPorEventoEBusca(eventoID int64, busca string) ([]ItemComTipo, error) {
	var linhas []ItemComTipo
	like := "%" + busca + "%"
	err := r.db.Table("itens_pedido").
		Select("itens_pedido.*, tipos_ingresso.nome as tipo_ingresso_nome").
		Joins("JOIN tipos_ingresso ON tipos_ingresso.id = itens_pedido.tipo_ingresso_id").
		Where("tipos_ingresso.evento_id = ? AND itens_pedido.status IN ?", eventoID, []domain.StatusItemPedido{
			domain.StatusItemPago, domain.StatusItemUtilizado,
		}).
		Where("itens_pedido.titular_nome ILIKE ? OR itens_pedido.titular_email ILIKE ? OR itens_pedido.codigo ILIKE ?", like, like, like).
		Order("itens_pedido.titular_nome").
		Limit(50).
		Scan(&linhas).Error
	return linhas, err
}

// ResumoCheckin é o contador de GET /checkin/eventos/:id/resumo.
type ResumoCheckin struct {
	TotalPagos      int64
	TotalUtilizados int64
}

func (r *ItemPedidoRepository) ResumoCheckin(eventoID int64) (ResumoCheckin, error) {
	var resumo ResumoCheckin
	base := r.db.Table("itens_pedido").
		Joins("JOIN tipos_ingresso ON tipos_ingresso.id = itens_pedido.tipo_ingresso_id").
		Where("tipos_ingresso.evento_id = ?", eventoID)

	if err := base.Session(&gorm.Session{}).Where("itens_pedido.status IN ?", []domain.StatusItemPedido{
		domain.StatusItemPago, domain.StatusItemUtilizado,
	}).Count(&resumo.TotalPagos).Error; err != nil {
		return resumo, err
	}
	if err := base.Session(&gorm.Session{}).Where("itens_pedido.status = ?", domain.StatusItemUtilizado).Count(&resumo.TotalUtilizados).Error; err != nil {
		return resumo, err
	}
	return resumo, nil
}

// ItemVenda é uma linha do painel de vendas do organizador (seção 1.11):
// item vendido (pago ou já utilizado) com o nome do tipo de ingresso.
type ItemVenda struct {
	domain.ItemPedido
	TipoIngressoNome string
}

// ListarVendasPorEvento lista todos os ingressos vendidos (pago +
// utilizado) de um evento, mais recentes primeiro — a base do "vendas
// básico" do painel do organizador.
func (r *ItemPedidoRepository) ListarVendasPorEvento(eventoID int64) ([]ItemVenda, error) {
	var linhas []ItemVenda
	err := r.db.Table("itens_pedido").
		Select("itens_pedido.*, tipos_ingresso.nome as tipo_ingresso_nome").
		Joins("JOIN tipos_ingresso ON tipos_ingresso.id = itens_pedido.tipo_ingresso_id").
		Where("tipos_ingresso.evento_id = ? AND itens_pedido.status IN ?", eventoID, []domain.StatusItemPedido{
			domain.StatusItemPago, domain.StatusItemUtilizado,
		}).
		Order("itens_pedido.criado_em DESC").
		Scan(&linhas).Error
	return linhas, err
}

// ListarPagosPorEvento é usado no cancelamento de evento pelo organizador
// (seção 7.5) — todo item pago do evento precisa de reembolso integral.
func (r *ItemPedidoRepository) ListarPagosPorEvento(eventoID int64) ([]domain.ItemPedido, error) {
	var itens []domain.ItemPedido
	err := r.db.
		Select("itens_pedido.*").
		Joins("JOIN tipos_ingresso ON tipos_ingresso.id = itens_pedido.tipo_ingresso_id").
		Where("tipos_ingresso.evento_id = ? AND itens_pedido.status = ?", eventoID, domain.StatusItemPago).
		Find(&itens).Error
	return itens, err
}

// ItemComEvento é o item de "Meus ingressos" (seção 8) já com os dados
// do evento pra não precisar de N+1 no handler.
type ItemComEvento struct {
	domain.ItemPedido
	EventoTitulo   string
	EventoSlug     string
	EventoInicioEm *time.Time
}

// ListarPagosPorUsuario é "Meus ingressos" — pago ou já utilizado
// (check-in feito), mais recentes primeiro.
func (r *ItemPedidoRepository) ListarPagosPorUsuario(usuarioID int64) ([]ItemComEvento, error) {
	var linhas []ItemComEvento
	err := r.db.Table("itens_pedido").
		Select("itens_pedido.*, eventos.titulo as evento_titulo, eventos.slug as evento_slug, eventos.inicio_em as evento_inicio_em").
		Joins("JOIN pedidos ON pedidos.id = itens_pedido.pedido_id").
		Joins("JOIN tipos_ingresso ON tipos_ingresso.id = itens_pedido.tipo_ingresso_id").
		Joins("JOIN eventos ON eventos.id = tipos_ingresso.evento_id").
		Where("pedidos.usuario_id = ? AND itens_pedido.status IN ?", usuarioID, []domain.StatusItemPedido{
			domain.StatusItemPago, domain.StatusItemUtilizado,
		}).
		Order("eventos.inicio_em").
		Scan(&linhas).Error
	return linhas, err
}

// EventosComIngressoPago é usado em "Meus eventos" (seção 3) pro selo
// "Ingresso" — eventos onde o usuário tem pelo menos um item pago.
func (r *ItemPedidoRepository) EventosComIngressoPago(usuarioID int64) ([]int64, error) {
	var eventoIDs []int64
	err := r.db.Table("itens_pedido").
		Select("DISTINCT tipos_ingresso.evento_id").
		Joins("JOIN pedidos ON pedidos.id = itens_pedido.pedido_id").
		Joins("JOIN tipos_ingresso ON tipos_ingresso.id = itens_pedido.tipo_ingresso_id").
		Where("pedidos.usuario_id = ? AND itens_pedido.status IN ?", usuarioID, []domain.StatusItemPedido{
			domain.StatusItemPago, domain.StatusItemUtilizado,
		}).
		Scan(&eventoIDs).Error
	return eventoIDs, err
}
