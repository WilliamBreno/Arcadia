package repository

import (
	"gorm.io/gorm"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
)

type PedidoRepository struct {
	db *gorm.DB
}

func NovoPedidoRepository(db *gorm.DB) *PedidoRepository {
	return &PedidoRepository{db: db}
}

// DB expõe a conexão para o service abrir transações que envolvem
// pedido + itens juntos (reserva precisa ser atômica).
func (r *PedidoRepository) DB() *gorm.DB {
	return r.db
}

func (r *PedidoRepository) Criar(tx *gorm.DB, p *domain.Pedido) error {
	return tx.Create(p).Error
}

func (r *PedidoRepository) Salvar(p *domain.Pedido) error {
	return r.db.Save(p).Error
}

func (r *PedidoRepository) BuscarPorID(id int64) (*domain.Pedido, error) {
	var p domain.Pedido
	if err := r.db.First(&p, id).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

// ListarExpirados retorna pedidos aguardando pagamento cujo prazo de
// reserva já passou — usado pelo job de expiração (seção 7.2).
func (r *PedidoRepository) ListarExpirados() ([]domain.Pedido, error) {
	var pedidos []domain.Pedido
	err := r.db.Where("status = ? AND expira_em < now()", domain.StatusPedidoAguardandoPagamento).Find(&pedidos).Error
	return pedidos, err
}
