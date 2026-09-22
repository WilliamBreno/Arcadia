package repository

import (
	"gorm.io/gorm"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
)

type PapelEventoRepository struct {
	db *gorm.DB
}

func NovoPapelEventoRepository(db *gorm.DB) *PapelEventoRepository {
	return &PapelEventoRepository{db: db}
}

func (r *PapelEventoRepository) Criar(p *domain.PapelEvento) error {
	return r.db.Create(p).Error
}

func (r *PapelEventoRepository) Salvar(p *domain.PapelEvento) error {
	return r.db.Save(p).Error
}

func (r *PapelEventoRepository) Buscar(eventoID, usuarioID int64, papel domain.Papel) (*domain.PapelEvento, error) {
	var p domain.PapelEvento
	err := r.db.First(&p, "evento_id = ? AND usuario_id = ? AND papel = ?", eventoID, usuarioID, papel).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *PapelEventoRepository) ListarPorUsuario(usuarioID int64) ([]domain.PapelEvento, error) {
	var papeis []domain.PapelEvento
	err := r.db.Where("usuario_id = ? AND status = ?", usuarioID, domain.StatusPapelConfirmado).Find(&papeis).Error
	return papeis, err
}

func (r *PapelEventoRepository) ListarPorEvento(eventoID int64) ([]domain.PapelEvento, error) {
	var papeis []domain.PapelEvento
	err := r.db.Where("evento_id = ?", eventoID).Find(&papeis).Error
	return papeis, err
}
