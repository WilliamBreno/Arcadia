package service

// Item 9.8: divisão do repasse entre o organizador principal e os
// coprodutores, em pontos-base (10000 = 100%).

const BPTotal = 10000

// ParteRepasse: quem recebe e quanto (em pontos-base).
type ParteRepasse struct {
	OrganizadorID int64
	BP            int64
}

// ValoresParte: a fatia de cada organizador, com os mesmos campos do
// repasse. Líquido = bruto − taxa do processador − taxas absorvidas.
type ValoresParte struct {
	OrganizadorID           int64
	BP                      int64
	BrutoCentavos           int64
	TaxaProcessadorCentavos int64
	TaxaAbsorvidaCentavos   int64
	ReservaCentavos         int64
	LiquidoCentavos         int64
}

// PartesDoEvento monta a lista com o principal primeiro (fica com o que
// não foi dado aos coprodutores).
func PartesDoEvento(principalID int64, coprodutores []ParteRepasse) []ParteRepasse {
	resto := int64(BPTotal)
	partes := []ParteRepasse{{OrganizadorID: principalID}}
	for _, c := range coprodutores {
		if c.BP <= 0 || c.OrganizadorID == principalID {
			continue
		}
		partes = append(partes, c)
		resto -= c.BP
	}
	partes[0].BP = resto
	return partes
}

// DividirRepasse aplica o percentual de cada coprodutor a cada valor
// (divisão inteira); o principal fica com o resto, inclusive os centavos
// da divisão. Vale também para valores negativos (saldo devedor), que
// ficam divididos na mesma proporção. A soma das partes é sempre igual
// ao total. Testada.
func DividirRepasse(f Financeiro, reserva int64, partes []ParteRepasse) []ValoresParte {
	out := make([]ValoresParte, len(partes))
	var soma ValoresParte
	for i := 1; i < len(partes); i++ {
		p := partes[i]
		v := ValoresParte{
			OrganizadorID:           p.OrganizadorID,
			BP:                      p.BP,
			BrutoCentavos:           f.BrutoCentavos * p.BP / BPTotal,
			TaxaProcessadorCentavos: f.TaxaProcessadorCentavos * p.BP / BPTotal,
			TaxaAbsorvidaCentavos:   f.TaxaAbsorvidaCentavos * p.BP / BPTotal,
			ReservaCentavos:         reserva * p.BP / BPTotal,
		}
		v.LiquidoCentavos = v.BrutoCentavos - v.TaxaProcessadorCentavos - v.TaxaAbsorvidaCentavos
		out[i] = v
		soma.BrutoCentavos += v.BrutoCentavos
		soma.TaxaProcessadorCentavos += v.TaxaProcessadorCentavos
		soma.TaxaAbsorvidaCentavos += v.TaxaAbsorvidaCentavos
		soma.ReservaCentavos += v.ReservaCentavos
	}
	principal := ValoresParte{
		OrganizadorID:           partes[0].OrganizadorID,
		BP:                      partes[0].BP,
		BrutoCentavos:           f.BrutoCentavos - soma.BrutoCentavos,
		TaxaProcessadorCentavos: f.TaxaProcessadorCentavos - soma.TaxaProcessadorCentavos,
		TaxaAbsorvidaCentavos:   f.TaxaAbsorvidaCentavos - soma.TaxaAbsorvidaCentavos,
		ReservaCentavos:         reserva - soma.ReservaCentavos,
	}
	principal.LiquidoCentavos = principal.BrutoCentavos - principal.TaxaProcessadorCentavos - principal.TaxaAbsorvidaCentavos
	out[0] = principal
	return out
}
