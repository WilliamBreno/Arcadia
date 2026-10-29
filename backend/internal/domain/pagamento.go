package domain

import (
	"time"

	"gorm.io/datatypes"
)

type Pagamento struct {
	ID                      int64 `gorm:"primaryKey"`
	PedidoID                int64
	MPPaymentID             string // id do pagamento no processador (nome histórico)
	Processador             string // "mercadopago" (item 5.2)
	Metodo                  string
	Status                  string
	ValorCentavos           int64
	TaxaProcessadorCentavos int64
	PayloadJSON             datatypes.JSON
	// Devolução automática do pagamento inteiro (item 7.1): ver
	// migração 000042. DevolvidoEm vazio com motivo = estorno pendente.
	DevolucaoMotivo *string
	DevolucaoMPID   *string
	DevolvidoEm     *time.Time
	CriadoEm        time.Time
}

// Motivos de devolução automática.
const (
	DevolucaoAposReserva     = "apos_reserva"
	DevolucaoDuplicado       = "duplicado"
	DevolucaoValorDivergente = "valor_divergente"
)

func (Pagamento) TableName() string {
	return "pagamentos"
}
