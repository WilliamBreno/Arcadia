package repository

import (
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
)

type CupomRepository struct {
	db *gorm.DB
}

func NovoCupomRepository(db *gorm.DB) *CupomRepository {
	return &CupomRepository{db: db}
}

func (r *CupomRepository) Criar(c *domain.Cupom) error {
	return r.db.Create(c).Error
}

func (r *CupomRepository) ListarPorEvento(eventoID int64) ([]domain.Cupom, error) {
	var lista []domain.Cupom
	err := r.db.Where("evento_id = ?", eventoID).Order("id DESC").Find(&lista).Error
	return lista, err
}

func (r *CupomRepository) BuscarPorID(id int64) (*domain.Cupom, error) {
	var c domain.Cupom
	if err := r.db.First(&c, id).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *CupomRepository) Salvar(c *domain.Cupom) error {
	return r.db.Save(c).Error
}

// BuscarPorCodigo é case-insensitive (códigos ficam em maiúsculas).
func (r *CupomRepository) BuscarPorCodigo(db *gorm.DB, eventoID int64, codigo string, travar bool) (*domain.Cupom, error) {
	if db == nil {
		db = r.db
	}
	if travar {
		db = db.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var c domain.Cupom
	if err := db.First(&c, "evento_id = ? AND codigo = ?", eventoID, strings.ToUpper(strings.TrimSpace(codigo))).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

// AjustarUsos soma delta (negativo devolve usos, ex.: reserva expirada).
func (r *CupomRepository) AjustarUsos(db *gorm.DB, id int64, delta int) error {
	if db == nil {
		db = r.db
	}
	return db.Model(&domain.Cupom{}).Where("id = ?", id).
		UpdateColumn("usos", gorm.Expr("GREATEST(usos + ?, 0)", delta)).Error
}
