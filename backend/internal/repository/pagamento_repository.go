package repository

import (
	"time"

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
	// Pagamento devolvido automaticamente (duplicado, atrasado) não conta.
	if err := r.db.First(&p, "pedido_id = ? AND status = ? AND devolucao_motivo IS NULL", pedidoID, "approved").Error; err != nil {
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
		Where("pedidos.evento_id = ? AND pagamentos.status = ? AND pagamentos.devolucao_motivo IS NULL", eventoID, "approved").
		Group("pagamentos.metodo").
		Scan(&linhas).Error
	return linhas, err
}

// AtualizarStatusSe muda o status do pagamento só se ele ainda estiver em
// `de` — é o que torna idempotente o processamento de notificações
// repetidas (item 5.3). Devolve true se esta chamada fez a mudança.
func (r *PagamentoRepository) AtualizarStatusSe(id int64, de, para string, valorCentavos, taxaCentavos int64) (bool, error) {
	res := r.db.Model(&domain.Pagamento{}).Where("id = ? AND status = ?", id, de).
		Updates(map[string]any{"status": para, "valor_centavos": valorCentavos, "taxa_processador_centavos": taxaCentavos})
	return res.RowsAffected == 1, res.Error
}

// MarcarDevolver grava o motivo uma única vez (o estorno sai depois).
func (r *PagamentoRepository) MarcarDevolver(tx *gorm.DB, id int64, motivo string) error {
	return tx.Model(&domain.Pagamento{}).Where("id = ? AND devolucao_motivo IS NULL", id).
		Update("devolucao_motivo", motivo).Error
}

// MarcarDevolvido registra o estorno confirmado pelo processador. Devolve
// false se outro processo já tinha registrado (o e-mail sai uma vez só).
func (r *PagamentoRepository) MarcarDevolvido(id int64, mpID string) (bool, error) {
	res := r.db.Model(&domain.Pagamento{}).Where("id = ? AND devolvido_em IS NULL", id).
		Updates(map[string]any{"devolvido_em": time.Now(), "devolucao_mp_id": mpID})
	return res.RowsAffected > 0, res.Error
}

// ListarDevolucoesPendentes: marcados para devolver e ainda sem estorno.
func (r *PagamentoRepository) ListarDevolucoesPendentes() ([]domain.Pagamento, error) {
	var ps []domain.Pagamento
	err := r.db.Where("devolucao_motivo IS NOT NULL AND devolvido_em IS NULL").Order("id").Limit(50).Find(&ps).Error
	return ps, err
}
