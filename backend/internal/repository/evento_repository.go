package repository

import (
	"gorm.io/gorm"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
)

type EventoRepository struct {
	db *gorm.DB
}

func NovoEventoRepository(db *gorm.DB) *EventoRepository {
	return &EventoRepository{db: db}
}

func (r *EventoRepository) Criar(e *domain.Evento) error {
	return r.db.Create(e).Error
}

func (r *EventoRepository) Salvar(e *domain.Evento) error {
	return r.db.Save(e).Error
}

func (r *EventoRepository) BuscarPorID(id int64) (*domain.Evento, error) {
	var e domain.Evento
	if err := r.db.First(&e, id).Error; err != nil {
		return nil, err
	}
	return &e, nil
}

func (r *EventoRepository) BuscarPorSlug(slug string) (*domain.Evento, error) {
	var e domain.Evento
	if err := r.db.First(&e, "slug = ?", slug).Error; err != nil {
		return nil, err
	}
	return &e, nil
}

func (r *EventoRepository) SlugExiste(slug string) (bool, error) {
	var total int64
	if err := r.db.Model(&domain.Evento{}).Where("slug = ?", slug).Count(&total).Error; err != nil {
		return false, err
	}
	return total > 0, nil
}

func (r *EventoRepository) ListarPorOrganizador(organizadorID int64) ([]domain.Evento, error) {
	var eventos []domain.Evento
	if err := r.db.Where("organizador_id = ?", organizadorID).Order("criado_em desc").Find(&eventos).Error; err != nil {
		return nil, err
	}
	return eventos, nil
}
