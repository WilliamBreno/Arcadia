package repository

import (
	"gorm.io/gorm"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
)

type LancamentoRepository struct {
	db *gorm.DB
}

func NovoLancamentoRepository(db *gorm.DB) *LancamentoRepository {
	return &LancamentoRepository{db: db}
}

func (r *LancamentoRepository) Criar(l *domain.Lancamento) error {
	return r.db.Create(l).Error
}

func (r *LancamentoRepository) CriarEmLote(lancamentos []domain.Lancamento) error {
	if len(lancamentos) == 0 {
		return nil
	}
	return r.db.Create(&lancamentos).Error
}
