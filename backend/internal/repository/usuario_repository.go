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

// BloquearCartao marca que o usuário contestou uma compra no cartão
// (item 5.3). Idempotente: mantém a data do primeiro bloqueio.
func (r *UsuarioRepository) BloquearCartao(tx *gorm.DB, usuarioID int64) error {
	return tx.Model(&domain.Usuario{}).
		Where("id = ? AND cartao_bloqueado_em IS NULL", usuarioID).
		Update("cartao_bloqueado_em", time.Now()).Error
}

// TravarParaCompra trava a linha do usuário dentro da transação de
// reserva, para o limite de ingressos por conta valer mesmo com duas
// compras simultâneas da mesma pessoa.
func (r *UsuarioRepository) TravarParaCompra(tx *gorm.DB, usuarioID int64) error {
	var id int64
	return tx.Raw("SELECT id FROM usuarios WHERE id = ? FOR UPDATE", usuarioID).Scan(&id).Error
}

// EmailsAdmins devolve os e-mails dos admins da plataforma (alertas).
func (r *UsuarioRepository) EmailsAdmins() ([]string, error) {
	var emails []string
	err := r.db.Model(&domain.Usuario{}).Where("papel_plataforma = ?", domain.PapelAdminPlataforma).Pluck("email", &emails).Error
	return emails, err
}
