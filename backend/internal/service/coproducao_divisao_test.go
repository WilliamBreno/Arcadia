package service

import "testing"

func somar(vs []ValoresParte) (b, p, a, r, l int64) {
	for _, v := range vs {
		b += v.BrutoCentavos
		p += v.TaxaProcessadorCentavos
		a += v.TaxaAbsorvidaCentavos
		r += v.ReservaCentavos
		l += v.LiquidoCentavos
	}
	return
}

func TestDividirRepasse(t *testing.T) {
	f := Financeiro{BrutoCentavos: 100001, TaxaProcessadorCentavos: 3333, TaxaAbsorvidaCentavos: 101}
	f.LiquidoCentavos = f.BrutoCentavos - f.TaxaProcessadorCentavos - f.TaxaAbsorvidaCentavos
	partes := PartesDoEvento(1, []ParteRepasse{{OrganizadorID: 2, BP: 4000}, {OrganizadorID: 3, BP: 1000}, {OrganizadorID: 9, BP: 0}})
	if len(partes) != 3 || partes[0].BP != 5000 {
		t.Fatalf("partes: %+v", partes)
	}
	vs := DividirRepasse(f, 999, partes)
	b, p, a, r, l := somar(vs)
	if b != f.BrutoCentavos || p != f.TaxaProcessadorCentavos || a != f.TaxaAbsorvidaCentavos || r != 999 || l != f.LiquidoCentavos {
		t.Fatalf("a soma não fecha: %d %d %d %d %d", b, p, a, r, l)
	}
	if vs[1].BrutoCentavos != 40000 || vs[2].BrutoCentavos != 10000 || vs[0].BrutoCentavos != 50001 {
		t.Fatalf("centavo da divisão deveria ficar com o principal: %+v", vs)
	}
	if vs[1].ReservaCentavos != 399 || vs[2].ReservaCentavos != 99 || vs[0].ReservaCentavos != 501 {
		t.Fatalf("reserva: %+v", vs)
	}
}

func TestDividirRepasseNegativo(t *testing.T) {
	// Só taxa de venda em dinheiro na porta: líquido negativo (saldo devedor).
	f := Financeiro{TaxaAbsorvidaCentavos: 1001}
	f.LiquidoCentavos = -1001
	vs := DividirRepasse(f, 0, PartesDoEvento(1, []ParteRepasse{{OrganizadorID: 2, BP: 3000}}))
	if vs[1].LiquidoCentavos != -300 || vs[0].LiquidoCentavos != -701 {
		t.Fatalf("devedor dividido na mesma proporção: %+v", vs)
	}
}

func TestSemCoprodutor(t *testing.T) {
	f := Financeiro{BrutoCentavos: 5000, TaxaProcessadorCentavos: 50, LiquidoCentavos: 4950}
	vs := DividirRepasse(f, 100, PartesDoEvento(7, nil))
	if len(vs) != 1 || vs[0].BP != BPTotal || vs[0].LiquidoCentavos != 4950 || vs[0].ReservaCentavos != 100 || vs[0].OrganizadorID != 7 {
		t.Fatalf("sem coprodutor o principal leva tudo: %+v", vs)
	}
}
