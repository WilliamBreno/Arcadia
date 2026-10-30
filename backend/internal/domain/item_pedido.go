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
	TaxaAbsorvida          bool // organizador pagou a taxa: não está em TotalCentavos
	GarantiaContratada     bool
	GarantiaCentavos       int64
	CupomID                *int64
	DescontoCentavos       int64
	Cortesia               bool
	// Item 8.4: inscrição paga de talento (não é ingresso de plateia).
	FichaID *int64
	// Item 8.6: espaço de expositor (também não é ingresso).
	ExpositorID *int64
	// Item 9.5: cadeira marcada (nil = sem lugar marcado) e o rótulo
	// gravado na compra ("Plateia · Fileira A · Lugar 5").
	LugarID     *int64
	LugarRotulo string
	// Item 10.3: adicional (entregue no evento, não é ingresso), pacote e
	// compra em grupo de onde o item veio.
	Adicional bool
	PacoteID  *int64
	GrupoID   *int64
	// Item 9.2: comissão do divulgador gravada na compra.
	ComissaoAfiliadoCentavos int64
	TotalCentavos            int64
	Status                   StatusItemPedido
	Codigo                   string
	QRToken                  string
	UtilizadoEm              *time.Time
	CanceladoEm              *time.Time
	MotivoCancelamento       string
	CriadoEm                 time.Time
}

func (ItemPedido) TableName() string {
	return "itens_pedido"
}

// Interno: item que não é ingresso de plateia (inscrição 8.4, espaço 8.6).
func (i *ItemPedido) Interno() bool { return i.FichaID != nil || i.ExpositorID != nil || i.Adicional }
