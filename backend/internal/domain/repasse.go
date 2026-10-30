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
	TaxaAbsorvidaCentavos   int64 // taxas que o organizador escolheu pagar (item 5.1)
	// Reserva de risco (item 5.4): parte do líquido segurada por um prazo.
	ReservaCentavos       int64
	ReservaLiberarEm      *time.Time
	ReservaStatus         string
	ReservaPagoEm         *time.Time
	ReservaComprovanteURL string
	ReservaObservacao     string
	ValorLiquidoCentavos  int64
	Status                StatusRepasse
	LiberarEm             time.Time
	PagoEm                *time.Time
	ComprovanteURL        string
	Observacao            string
	// Item 9.1: envio automático por Pix (Mercado Pago Payouts).
	EnvioStatus      string
	EnvioMotivo      string
	EnvioTentativa   int
	EnvioReferencia  *string
	MPPayoutID       *string `gorm:"column:mp_payout_id"`
	EnvioProximaEm   *time.Time
	EnvioAprovadoPor *int64
	// Item 9.8: parte deste organizador no evento (10000 = 100%).
	PercentualBP int64 `gorm:"column:percentual_bp;default:10000"`
	CriadoEm     time.Time
}

func (Repasse) TableName() string {
	return "repasses"
}

// Situação da reserva de risco (item 5.4).
const (
	ReservaNenhuma = "nenhuma"
	ReservaRetida  = "retida"
	ReservaPaga    = "paga"
	ReservaUsada   = "usada"
)
