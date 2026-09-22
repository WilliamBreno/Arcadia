package domain

import "time"

type TipoConvite string

const (
	TipoConviteJurado               TipoConvite = "jurado"
	TipoConviteParticipanteEspecial TipoConvite = "participante_especial"
)

// Convite é um link reutilizável (por tipo) que o organizador gera para
// convidar jurados ou participantes especiais (seção 7.11 do plano).
// Só o hash do token é persistido.
type Convite struct {
	ID         int64 `gorm:"primaryKey"`
	EventoID   int64
	Tipo       TipoConvite
	TokenHash  string
	MaxUsos    *int
	Usos       int
	ExpiraEm   *time.Time
	RevogadoEm *time.Time
	CriadoPor  int64
	CriadoEm   time.Time
}

func (Convite) TableName() string {
	return "convites"
}
