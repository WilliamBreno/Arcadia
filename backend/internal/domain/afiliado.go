package domain

import "time"

// Afiliado é um divulgador do evento: um link com ?ref=<codigo> que atribui
// os pedidos a ele. Item 9.2: comissão calculada pela Evve (% do preço ou
// valor fixo por ingresso) e paga pelo organizador por fora.
type Afiliado struct {
	ID                   int64 `gorm:"primaryKey"`
	EventoID             int64
	Nome                 string
	Codigo               string
	Ativo                bool
	ComissaoTipo         string // nenhuma, percentual, fixo
	ComissaoBP           int
	ComissaoFixaCentavos int64
	PainelTokenHash      *string
	CriadoEm             time.Time
}

func (Afiliado) TableName() string { return "afiliados" }

const (
	ComissaoNenhuma    = "nenhuma"
	ComissaoPercentual = "percentual"
	ComissaoFixo       = "fixo"
)

// ComissaoDoItem: quanto o divulgador ganha por um ingresso deste preço
// (item 9.2). Ingresso gratuito não gera comissão; o fixo nunca passa do
// preço; o percentual arredonda para baixo. Testada.
func (a *Afiliado) ComissaoDoItem(precoCentavos int64) int64 {
	if a == nil || precoCentavos <= 0 {
		return 0
	}
	switch a.ComissaoTipo {
	case ComissaoPercentual:
		return precoCentavos * int64(a.ComissaoBP) / 10000
	case ComissaoFixo:
		if a.ComissaoFixaCentavos > precoCentavos {
			return precoCentavos
		}
		return a.ComissaoFixaCentavos
	}
	return 0
}
