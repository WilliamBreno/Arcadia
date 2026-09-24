package domain

import "time"

type TipoCupom string

const (
	CupomPercentual TipoCupom = "percentual"
	CupomValor      TipoCupom = "valor"
)

// Cupom dá desconto no PREÇO do ingresso (nunca na taxa da plataforma nem
// na garantia); o desconto é custo do organizador.
type Cupom struct {
	ID        int64 `gorm:"primaryKey"`
	EventoID  int64
	Codigo    string
	Tipo      TipoCupom
	Valor     int64 // percentual (1-100) ou centavos
	MaxUsos   *int
	Usos      int
	ValidoDe  *time.Time
	ValidoAte *time.Time
	Ativo     bool
	CriadoEm  time.Time
}

func (Cupom) TableName() string {
	return "cupons"
}
