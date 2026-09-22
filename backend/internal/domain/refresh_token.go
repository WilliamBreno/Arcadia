package domain

import "time"

// RefreshToken representa um refresh token emitido (guardado como hash).
// Renovado por rotação: cada uso revoga o antigo e cria um novo.
type RefreshToken struct {
	ID         int64 `gorm:"primaryKey"`
	UsuarioID  int64
	TokenHash  string
	ExpiraEm   time.Time
	RevogadoEm *time.Time
	CriadoEm   time.Time
}

func (RefreshToken) TableName() string {
	return "refresh_tokens"
}
