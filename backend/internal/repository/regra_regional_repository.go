package repository

import (
	"gorm.io/gorm"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
)

type RegraRegionalRepository struct {
	db *gorm.DB
}

func NovoRegraRegionalRepository(db *gorm.DB) *RegraRegionalRepository {
	return &RegraRegionalRepository{db: db}
}

// Buscar tenta a regra específica do município antes da regra geral do
// estado (municipio nulo) — Fortaleza/CE tem restrição que o resto do
// Ceará não tem.
func (r *RegraRegionalRepository) Buscar(uf, municipio string) (*domain.RegraRegional, error) {
	var regra domain.RegraRegional

	if municipio != "" {
		err := r.db.First(&regra, "uf = ? AND municipio = ?", uf, municipio).Error
		if err == nil {
			return &regra, nil
		}
	}

	err := r.db.First(&regra, "uf = ? AND municipio IS NULL", uf).Error
	if err != nil {
		return nil, err
	}
	return &regra, nil
}
