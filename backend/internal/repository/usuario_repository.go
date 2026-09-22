package repository

import (
	"time"

	"gorm.io/gorm"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
)

type UsuarioRepository struct {
	db *gorm.DB
}

func NovoUsuarioRepository(db *gorm.DB) *UsuarioRepository {
	return &UsuarioRepository{db: db}
}

func (r *UsuarioRepository) Criar(u *domain.Usuario) error {
	return r.db.Create(u).Error
}

func (r *UsuarioRepository) Salvar(u *domain.Usuario) error {
	return r.db.Save(u).Error
}

func (r *UsuarioRepository) BuscarPorEmail(email string) (*domain.Usuario, error) {
	var u domain.Usuario
	if err := r.db.First(&u, "email = ?", email).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UsuarioRepository) BuscarPorID(id int64) (*domain.Usuario, error) {
	var u domain.Usuario
	if err := r.db.First(&u, id).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UsuarioRepository) BuscarPorGoogleID(googleID string) (*domain.Usuario, error) {
	var u domain.Usuario
	if err := r.db.First(&u, "google_id = ?", googleID).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UsuarioRepository) BuscarPorTokenVerificacao(hash string) (*domain.Usuario, error) {
	var u domain.Usuario
	err := r.db.First(&u, "email_verificacao_token_hash = ? AND email_verificacao_expira_em > ?", hash, time.Now()).Error
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UsuarioRepository) BuscarPorTokenReset(hash string) (*domain.Usuario, error) {
	var u domain.Usuario
	err := r.db.First(&u, "senha_reset_token_hash = ? AND senha_reset_expira_em > ?", hash, time.Now()).Error
	if err != nil {
		return nil, err
	}
	return &u, nil
}
