package domain

import "time"

type Papel string

const (
	PapelOrganizador  Papel = "organizador"
	PapelJurado       Papel = "jurado"
	PapelParticipante Papel = "participante"
	PapelStaff        Papel = "staff"
)

type OrigemPapel string

const (
	OrigemConvite   OrigemPapel = "convite"
	OrigemInscricao OrigemPapel = "inscricao"
	OrigemManual    OrigemPapel = "manual"
)

type StatusPapel string

const (
	StatusPapelPendente   StatusPapel = "pendente"
	StatusPapelConfirmado StatusPapel = "confirmado"
	StatusPapelRemovido   StatusPapel = "removido"
)

// PapelEvento modela o papel de uma pessoa NUM evento específico (seção 3
// do plano: "o papel é por evento, não por conta"). Único por
// (evento, usuario, papel).
type PapelEvento struct {
	ID        int64 `gorm:"primaryKey"`
	EventoID  int64
	UsuarioID int64
	Papel     Papel
	Origem    OrigemPapel
	Status    StatusPapel
	ConviteID *int64
	CriadoEm  time.Time
}

func (PapelEvento) TableName() string {
	return "papeis_evento"
}
