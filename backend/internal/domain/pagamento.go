package domain

import (
	"time"

	"gorm.io/datatypes"
)

type Pagamento struct {
	ID                      int64 `gorm:"primaryKey"`
	PedidoID                int64
	MPPaymentID             string
	Metodo                  string
	Status                  string
	ValorCentavos           int64
	TaxaProcessadorCentavos int64
	PayloadJSON             datatypes.JSON
	CriadoEm                time.Time
}

func (Pagamento) TableName() string {
	return "pagamentos"
}
