package repository

import (
	"gorm.io/gorm"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
)

type CronogramaRepository struct {
	db *gorm.DB
}

func NovoCronogramaRepository(db *gorm.DB) *CronogramaRepository {
	return &CronogramaRepository{db: db}
}

func (r *CronogramaRepository) ListarPorEvento(eventoID int64) ([]domain.CronogramaItem, error) {
	var lista []domain.CronogramaItem
	err := r.db.Where("evento_id = ?", eventoID).Order("inicio_em, id").Find(&lista).Error
	return lista, err
}

func (r *CronogramaRepository) Criar(i *domain.CronogramaItem) error  { return r.db.Create(i).Error }
func (r *CronogramaRepository) Salvar(i *domain.CronogramaItem) error { return r.db.Save(i).Error }

func (r *CronogramaRepository) BuscarPorID(id int64) (*domain.CronogramaItem, error) {
	var i domain.CronogramaItem
	if err := r.db.First(&i, id).Error; err != nil {
		return nil, err
	}
	return &i, nil
}

func (r *CronogramaRepository) Excluir(i *domain.CronogramaItem) error { return r.db.Delete(i).Error }
