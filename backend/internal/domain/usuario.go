package domain

import "time"

type PapelPlataforma string

const (
	PapelUsuario         PapelPlataforma = "usuario"
	PapelAdminPlataforma PapelPlataforma = "admin_plataforma"
)

// Usuario é a conta de login. O papel por evento (organizador, jurado,
// participante, staff) vive em papeis_evento, introduzido junto com
// eventos (item 1.3) — aqui só existe o papel global da plataforma.
type Usuario struct {
	ID                        int64 `gorm:"primaryKey"`
	Nome                      string
	Email                     string
	SenhaHash                 *string
	GoogleID                  *string
	Telefone                  string
	AvatarURL                 string
	EmailVerificadoEm         *time.Time
	EmailVerificacaoTokenHash *string
	EmailVerificacaoExpiraEm  *time.Time
	SenhaResetTokenHash       *string
	SenhaResetExpiraEm        *time.Time
	PapelPlataforma           PapelPlataforma `gorm:"default:usuario"`
	CriadoEm                  time.Time
}

func (Usuario) TableName() string {
	return "usuarios"
}
