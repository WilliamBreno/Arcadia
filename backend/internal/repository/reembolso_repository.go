package repository

import (
	"gorm.io/gorm"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
)

type ReembolsoRepository struct {
	db *gorm.DB
}

func NovoReembolsoRepository(db *gorm.DB) *ReembolsoRepository {
	return &ReembolsoRepository{db: db}
}

func (r *ReembolsoRepository) Criar(reembolso *domain.Reembolso) error {
	return r.db.Create(reembolso).Error
}

func (r *ReembolsoRepository) Salvar(reembolso *domain.Reembolso) error {
	return r.db.Save(reembolso).Error
}

// SomaConcluidosPorPagamento é a base da invariante da seção 7.4: a soma
// dos reembolsos de um pagamento nunca pode exceder o valor pago.
func (r *ReembolsoRepository) SomaConcluidosPorPagamento(pagamentoID int64) (int64, error) {
	var soma int64
	err := r.db.Model(&domain.Reembolso{}).
		Where("pagamento_id = ? AND status = ?", pagamentoID, domain.StatusReembolsoConcluido).
		Select("COALESCE(SUM(valor_centavos), 0)").
		Scan(&soma).Error
	return soma, err
}
