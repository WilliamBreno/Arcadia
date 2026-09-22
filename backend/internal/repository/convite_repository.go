package repository

import (
	"time"

	"gorm.io/gorm"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
)

type ConviteRepository struct {
	db *gorm.DB
}

func NovoConviteRepository(db *gorm.DB) *ConviteRepository {
	return &ConviteRepository{db: db}
}

func (r *ConviteRepository) Criar(c *domain.Convite) error {
	return r.db.Create(c).Error
}

func (r *ConviteRepository) Salvar(c *domain.Convite) error {
	return r.db.Save(c).Error
}

func (r *ConviteRepository) BuscarPorID(id int64) (*domain.Convite, error) {
	var c domain.Convite
	if err := r.db.First(&c, id).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *ConviteRepository) BuscarPorTokenHash(hash string) (*domain.Convite, error) {
	var c domain.Convite
	if err := r.db.First(&c, "token_hash = ?", hash).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *ConviteRepository) ListarPorEvento(eventoID int64) ([]domain.Convite, error) {
	var convites []domain.Convite
	if err := r.db.Where("evento_id = ?", eventoID).Order("criado_em desc").Find(&convites).Error; err != nil {
		return nil, err
	}
	return convites, nil
}

// Valido checa expiração, revogação e limite de usos — sem gravar nada.
func (r *ConviteRepository) Valido(convite *domain.Convite) bool {
	if convite.RevogadoEm != nil {
		return false
	}
	if convite.ExpiraEm != nil && convite.ExpiraEm.Before(time.Now()) {
		return false
	}
	if convite.MaxUsos != nil && convite.Usos >= *convite.MaxUsos {
		return false
	}
	return true
}
