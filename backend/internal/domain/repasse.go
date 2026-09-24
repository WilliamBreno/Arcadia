package domain

import "time"

type StatusRepasse string

const (
	StatusRepasseCalculado StatusRepasse = "calculado"
	StatusRepassePendente  StatusRepasse = "pendente"
	StatusRepassePago      StatusRepasse = "pago"
	StatusRepasseCancelado StatusRepasse = "cancelado"
)

type Repasse struct {
	ID                      int64 `gorm:"primaryKey"`
	EventoID                int64
	OrganizadorID           int64
	ValorBrutoCentavos      int64
	TaxaProcessadorCentavos int64
	ValorLiquidoCentavos    int64
	Status                  StatusRepasse
	LiberarEm               time.Time
	PagoEm                  *time.Time
	ComprovanteURL          string
	Observacao              string
	CriadoEm                time.Time
}

func (Repasse) TableName() string {
	return "repasses"
}
