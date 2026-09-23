package domain

import "time"

type TipoReembolso string

const (
	TipoReembolsoGarantia        TipoReembolso = "garantia"
	TipoReembolsoArrependimento  TipoReembolso = "arrependimento"
	TipoReembolsoEventoCancelado TipoReembolso = "evento_cancelado"
	TipoReembolsoManual          TipoReembolso = "manual"
)

type StatusReembolso string

const (
	StatusReembolsoPendente  StatusReembolso = "pendente"
	StatusReembolsoConcluido StatusReembolso = "concluido"
	StatusReembolsoFalhou    StatusReembolso = "falhou"
)

// Reembolso registra cada solicitação de estorno (seção 7.4/7.5) — o
// dinheiro em si é refletido no ledger (lancamentos), isto aqui é o
// histórico da operação (pra auditoria e pro painel de falhas do admin).
type Reembolso struct {
	ID            int64 `gorm:"primaryKey"`
	PagamentoID   int64
	ItemID        int64
	ValorCentavos int64
	Tipo          TipoReembolso
	Status        StatusReembolso
	MPRefundID    string
	Motivo        string
	SolicitadoPor int64
	CriadoEm      time.Time
}

func (Reembolso) TableName() string {
	return "reembolsos"
}
