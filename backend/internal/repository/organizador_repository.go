package repository

import (
	"gorm.io/gorm"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
)

type OrganizadorRepository struct {
	db *gorm.DB
}

func NovoOrganizadorRepository(db *gorm.DB) *OrganizadorRepository {
	return &OrganizadorRepository{db: db}
}

func (r *OrganizadorRepository) Criar(o *domain.Organizador) error {
	return r.db.Create(o).Error
}

func (r *OrganizadorRepository) Salvar(o *domain.Organizador) error {
	return r.db.Save(o).Error
}

func (r *OrganizadorRepository) BuscarPorUsuarioID(usuarioID int64) (*domain.Organizador, error) {
	var o domain.Organizador
	if err := r.db.First(&o, "usuario_id = ?", usuarioID).Error; err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *OrganizadorRepository) BuscarPorID(id int64) (*domain.Organizador, error) {
	var o domain.Organizador
	if err := r.db.First(&o, id).Error; err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *OrganizadorRepository) BuscarPorSlug(slug string) (*domain.Organizador, error) {
	var o domain.Organizador
	if err := r.db.First(&o, "slug = ?", slug).Error; err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *OrganizadorRepository) SlugExiste(slug string) (bool, error) {
	var total int64
	if err := r.db.Model(&domain.Organizador{}).Where("slug = ?", slug).Count(&total).Error; err != nil {
		return false, err
	}
	return total > 0, nil
}
