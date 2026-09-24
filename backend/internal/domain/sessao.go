package domain

import "time"

type StatusSessao string

const (
	SessaoAtiva     StatusSessao = "ativa"
	SessaoCancelada StatusSessao = "cancelada"
)

// Sessao é uma data/turno de um evento com várias datas.
type Sessao struct {
	ID                 int64 `gorm:"primaryKey"`
	EventoID           int64
	Titulo             string
	InicioEm           time.Time
	FimEm              *time.Time
	Status             StatusSessao
	CanceladaEm        *time.Time
	MotivoCancelamento string
	CriadoEm           time.Time
}

func (Sessao) TableName() string { return "sessoes" }
