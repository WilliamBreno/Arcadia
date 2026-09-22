package repository

import (
	"time"

	"gorm.io/gorm"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
)

type RefreshTokenRepository struct {
	db *gorm.DB
}

func NovoRefreshTokenRepository(db *gorm.DB) *RefreshTokenRepository {
	return &RefreshTokenRepository{db: db}
}

func (r *RefreshTokenRepository) Criar(t *domain.RefreshToken) error {
	return r.db.Create(t).Error
}

func (r *RefreshTokenRepository) BuscarValidoPorHash(hash string) (*domain.RefreshToken, error) {
	var t domain.RefreshToken
	err := r.db.First(&t, "token_hash = ? AND revogado_em IS NULL AND expira_em > ?", hash, time.Now()).Error
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *RefreshTokenRepository) Revogar(t *domain.RefreshToken) error {
	agora := time.Now()
	t.RevogadoEm = &agora
	return r.db.Save(t).Error
}

func (r *RefreshTokenRepository) RevogarTodosDoUsuario(usuarioID int64) error {
	return r.db.Model(&domain.RefreshToken{}).
		Where("usuario_id = ? AND revogado_em IS NULL", usuarioID).
		Update("revogado_em", time.Now()).Error
}
