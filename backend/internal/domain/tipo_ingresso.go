package domain

import "time"

// TipoIngresso é um lote/categoria de ingresso (ou cadastro) de um evento.
// Preço sempre em centavos — nunca float (regra da seção 0 do plano).
type TipoIngresso struct {
	ID            int64 `gorm:"primaryKey"`
	EventoID      int64
	Nome          string
	Descricao     string
	PrecoCentavos int64
	Quantidade    int
	VendasInicio  *time.Time
	VendasFim     *time.Time
	MinPorPedido  int
	MaxPorPedido  int
	Ordem         int
	LoteGrupo     string
	MeiaEntrada   bool
	Ativo         bool
	CriadoEm      time.Time
}

func (TipoIngresso) TableName() string {
	return "tipos_ingresso"
}
