package domain

import "time"

type StatusPedido string

const (
	StatusPedidoAberto              StatusPedido = "aberto"
	StatusPedidoAguardandoPagamento StatusPedido = "aguardando_pagamento"
	StatusPedidoPago                StatusPedido = "pago"
	StatusPedidoExpirado            StatusPedido = "expirado"
	StatusPedidoCancelado           StatusPedido = "cancelado"
	StatusPedidoReembolsadoParcial  StatusPedido = "reembolsado_parcial"
	StatusPedidoReembolsado         StatusPedido = "reembolsado"
)

// Pedido segue a máquina de estados da seção 6: aberto -> aguardando_pagamento
// -> pago; expirado/cancelado/reembolsado(_parcial) são terminais.
type Pedido struct {
	ID             int64 `gorm:"primaryKey"`
	UsuarioID      int64
	EventoID       int64
	Status         StatusPedido
	TotalCentavos  int64
	ExpiraEm       *time.Time
	MPPreferenceID *string
	CriadoEm       time.Time
}

func (Pedido) TableName() string {
	return "pedidos"
}
