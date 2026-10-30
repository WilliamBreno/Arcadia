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
	AfiliadoID     *int64
	// Pix gerado na página (item 7.1): um QR por pedido.
	PixMPPaymentID *string
	PixCopiaCola   *string
	PixExpiraEm    *time.Time
	// Item 9.4: venda na porta (canal "porta", quem vendeu e como pagou).
	Canal       string `gorm:"default:online"`
	VendidoPor  *int64
	MetodoPorta *string
	CriadoEm    time.Time
}

const (
	CanalOnline = "online"
	CanalPorta  = "porta"
	// EmailBilheteria é o usuário de sistema dono dos pedidos da porta
	// (sem senha; o domínio .invalid nunca recebe e-mail).
	EmailBilheteria = "bilheteria@sistema.evve.invalid"
)

func (Pedido) TableName() string {
	return "pedidos"
}
