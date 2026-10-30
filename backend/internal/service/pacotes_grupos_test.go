package service

import "testing"

func somaRateio(ps []parteRateio) (total int64, unidades int) {
	for _, p := range ps {
		total += p.Preco * int64(p.Quantidade)
		unidades += p.Quantidade
	}
	return
}

func TestRatearPacote(t *testing.T) {
	// 4 ingressos de R$ 50 + camiseta de R$ 60 por R$ 200,01.
	itens := []PacoteItem{{PacoteID: 1, TipoIngressoID: 10, Quantidade: 4}, {PacoteID: 1, TipoIngressoID: 20, Quantidade: 1}}
	ps := RatearPacote(20001, itens, map[int64]int64{10: 5000, 20: 6000})
	total, unidades := somaRateio(ps)
	if total != 20001 || unidades != 5 {
		t.Fatalf("soma %d unidades %d: %+v", total, unidades, ps)
	}
	if ps[1].Preco != 4615 { // 20001 × 6000 / 26000
		t.Fatalf("camiseta proporcional: %+v", ps)
	}
	if len(ps) != 3 || ps[0].Quantidade != 3 || ps[2].Quantidade != 1 || ps[2].Preco <= ps[0].Preco {
		t.Fatalf("o centavo que sobra vai numa unidade do primeiro item: %+v", ps)
	}
}

func TestRatearPacoteSemSobra(t *testing.T) {
	ps := RatearPacote(8000, []PacoteItem{{TipoIngressoID: 1, Quantidade: 2}}, map[int64]int64{1: 5000})
	if len(ps) != 1 || ps[0].Preco != 4000 || ps[0].Quantidade != 2 {
		t.Fatalf("%+v", ps)
	}
}

func TestRatearPacoteGratuitos(t *testing.T) {
	ps := RatearPacote(1001, []PacoteItem{{TipoIngressoID: 1, Quantidade: 2}, {TipoIngressoID: 2, Quantidade: 1}}, map[int64]int64{})
	if total, _ := somaRateio(ps); total != 1001 {
		t.Fatalf("partes iguais quando a tabela é zero: %+v", ps)
	}
}
