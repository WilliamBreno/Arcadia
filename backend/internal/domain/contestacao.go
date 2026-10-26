package domain

import "time"

// Contestação (chargeback) de um pagamento no cartão (item 5.3).
const (
	ContestacaoEmDisputa = "em_disputa"
	ContestacaoPerdida   = "perdida"
	ContestacaoResolvida = "resolvida"
)

type Contestacao struct {
	ID                int64 `gorm:"primaryKey"`
	PagamentoID       int64
	PedidoID          int64
	EventoID          int64
	UsuarioID         int64
	ValorCentavos     int64
	Status            string
	StatusProcessador string
	CriadoEm          time.Time
	AtualizadoEm      time.Time
}

func (Contestacao) TableName() string { return "contestacoes" }
