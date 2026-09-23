package repository

import (
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
