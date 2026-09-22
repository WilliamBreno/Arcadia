package repository

import (
	"gorm.io/gorm"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
)

type LocalRepository struct {
	db *gorm.DB
}

func NovoLocalRepository(db *gorm.DB) *LocalRepository {
	return &LocalRepository{db: db}
}

func (r *LocalRepository) Criar(l *domain.Local) error {
	return r.db.Create(l).Error
}

func (r *LocalRepository) Salvar(l *domain.Local) error {
	return r.db.Save(l).Error
}

func (r *LocalRepository) Excluir(l *domain.Local) error {
	return r.db.Delete(l).Error
}

func (r *LocalRepository) BuscarPorID(id int64) (*domain.Local, error) {
	var l domain.Local
	if err := r.db.First(&l, id).Error; err != nil {
		return nil, err
	}
	return &l, nil
}

func (r *LocalRepository) ListarPorOrganizador(organizadorID int64) ([]domain.Local, error) {
	var locais []domain.Local
	if err := r.db.Where("organizador_id = ?", organizadorID).Order("nome").Find(&locais).Error; err != nil {
		return nil, err
	}
	return locais, nil
}
