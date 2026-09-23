package repository

import (
	"gorm.io/gorm"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
)

type PagamentoRepository struct {
	db *gorm.DB
}

func NovoPagamentoRepository(db *gorm.DB) *PagamentoRepository {
	return &PagamentoRepository{db: db}
}

func (r *PagamentoRepository) Criar(p *domain.Pagamento) error {
	return r.db.Create(p).Error
}

// BuscarPorMPPaymentID é a base da idempotência do webhook (seção 7.3):
// se já existe, o evento já foi processado.
func (r *PagamentoRepository) BuscarPorMPPaymentID(mpPaymentID string) (*domain.Pagamento, error) {
	var p domain.Pagamento
	if err := r.db.First(&p, "mp_payment_id = ?", mpPaymentID).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

// BuscarAprovadoPorPedido busca o pagamento aprovado de um pedido — um
// pedido só deveria ter um pagamento aprovado (o resto seria rejeitado
// ou duplicado), usado para localizar o pagamento a estornar.
func (r *PagamentoRepository) BuscarAprovadoPorPedido(pedidoID int64) (*domain.Pagamento, error) {
	var p domain.Pagamento
	if err := r.db.First(&p, "pedido_id = ? AND status = ?", pedidoID, "approved").Error; err != nil {
		return nil, err
	}
	return &p, nil
}
