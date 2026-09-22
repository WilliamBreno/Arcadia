package repository

import (
	"gorm.io/gorm"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
)

type MidiaFichaRepository struct {
	db *gorm.DB
}

func NovoMidiaFichaRepository(db *gorm.DB) *MidiaFichaRepository {
	return &MidiaFichaRepository{db: db}
}

func (r *MidiaFichaRepository) Criar(m *domain.MidiaFicha) error {
	return r.db.Create(m).Error
}

func (r *MidiaFichaRepository) Excluir(m *domain.MidiaFicha) error {
	return r.db.Delete(m).Error
}

func (r *MidiaFichaRepository) BuscarPorID(id int64) (*domain.MidiaFicha, error) {
	var m domain.MidiaFicha
	if err := r.db.First(&m, id).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *MidiaFichaRepository) ListarPorFicha(fichaID int64) ([]domain.MidiaFicha, error) {
	var midias []domain.MidiaFicha
	err := r.db.Where("ficha_id = ?", fichaID).Order("ordem").Find(&midias).Error
	return midias, err
}
