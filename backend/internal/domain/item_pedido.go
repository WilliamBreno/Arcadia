package domain

import "time"

type StatusItemPedido string

const (
	StatusItemReservado   StatusItemPedido = "reservado"
	StatusItemPago        StatusItemPedido = "pago"
	StatusItemUtilizado   StatusItemPedido = "utilizado"
	StatusItemCancelado   StatusItemPedido = "cancelado"
	StatusItemReembolsado StatusItemPedido = "reembolsado"
	StatusItemExpirado    StatusItemPedido = "expirado"
)

// ItemPedido é um ingresso/cadastro individual e nominal (seção 7.2).
// Preço, taxa e garantia são sempre recalculados no servidor a partir do
// tipo_ingresso e do config_plataforma — nunca confiar em valor do cliente.
type ItemPedido struct {
	ID                     int64 `gorm:"primaryKey"`
	PedidoID               int64
	TipoIngressoID         int64
	TitularNome            string
	TitularEmail           string
	PrecoCentavos          int64
	TaxaPlataformaCentavos int64
	GarantiaContratada     bool
	GarantiaCentavos       int64
	TotalCentavos          int64
	Status                 StatusItemPedido
	Codigo                 string
	QRToken                string
	UtilizadoEm            *time.Time
	CanceladoEm            *time.Time
	MotivoCancelamento     string
	CriadoEm               time.Time
}

func (ItemPedido) TableName() string {
	return "itens_pedido"
}
