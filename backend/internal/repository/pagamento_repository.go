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

// ResumoMetodo soma os pagamentos aprovados de um evento por método
// (pix/cartao) — base do quadro "economia com Pix" (item 5.2).
type ResumoMetodo struct {
	Metodo        string
	Quantidade    int64
	ValorCentavos int64
	TaxaCentavos  int64
}

func (r *PagamentoRepository) ResumoPorMetodo(eventoID int64) ([]ResumoMetodo, error) {
	var linhas []ResumoMetodo
	err := r.db.Table("pagamentos").
		Select("pagamentos.metodo, COUNT(*) as quantidade, COALESCE(SUM(pagamentos.valor_centavos),0) as valor_centavos, COALESCE(SUM(pagamentos.taxa_processador_centavos),0) as taxa_centavos").
		Joins("JOIN pedidos ON pedidos.id = pagamentos.pedido_id").
		Where("pedidos.evento_id = ? AND pagamentos.status = ?", eventoID, "approved").
		Group("pagamentos.metodo").
		Scan(&linhas).Error
	return linhas, err
}
