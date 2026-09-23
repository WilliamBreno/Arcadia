package domain

import "time"

// RegraRegional é o levantamento (seção 7.8, a validar com advogado) de
// restrições estaduais/municipais à cobrança de taxa de conveniência.
type RegraRegional struct {
	ID                   int64 `gorm:"primaryKey"`
	UF                   string
	Municipio            *string
	PermiteTaxa          bool
	TaxaMaximaPercentual *float64
	ExigeCanalSemTaxa    bool
	ExcecaoPublicoAte    *int
	Observacao           string
	Fonte                string
	VigenteDesde         *time.Time
	CriadoEm             time.Time
}

func (RegraRegional) TableName() string {
	return "regras_regionais"
}
