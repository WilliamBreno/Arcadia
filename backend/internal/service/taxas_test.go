package service

import (
	"testing"
	"time"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
)

// regraPadrao é a tabela aprovada pelo dono (seção 2.1, item 5.1).
func regraPadrao(t *testing.T) RegraTaxas {
	t.Helper()
	faixas, err := ParseFaixas("3000:99,8000:199,20000:399,*:699")
	if err != nil {
		t.Fatalf("faixas padrão inválidas: %v", err)
	}
	return RegraTaxas{Faixas: faixas, GarantiaPercentualBP: 500, GarantiaMinCentavos: 199, GarantiaMaxCentavos: 1490}
}

func TestParseFaixasInvalidas(t *testing.T) {
	for _, s := range []string{"", "3000:99", "3000:99,*:699,8000:1", "8000:199,3000:99,*:699", "3000:x,*:1", "abc"} {
		if _, err := ParseFaixas(s); err == nil {
			t.Errorf("ParseFaixas(%q) deveria falhar", s)
		}
	}
}

func TestTaxaPorFaixa(t *testing.T) {
	r := regraPadrao(t)
	casos := []struct{ preco, taxa int64 }{
		{0, 0}, {1, 99}, {3000, 99}, {3001, 199}, {8000, 199}, {8001, 399}, {20000, 399}, {20001, 699}, {1000000, 699},
	}
	for _, c := range casos {
		if got := r.TaxaPorFaixa(c.preco); got != c.taxa {
			t.Errorf("TaxaPorFaixa(%d) = %d, esperado %d", c.preco, got, c.taxa)
		}
	}
}

func TestGarantia(t *testing.T) {
	r := regraPadrao(t)
	casos := []struct{ preco, garantia int64 }{
		{0, 0},      // gratuito não tem garantia
		{3000, 199}, // 1,50 → mínimo 1,99
		{3333, 199}, // 1,6665 → 1,67 → mínimo
		{5000, 250}, // 5%
		{4999, 250}, // 249,95 → arredonda para 250
		{4990, 250}, // 249,5 → arredonda para cima
		{4989, 249}, // 249,45 → para baixo
		{15000, 750},
		{29800, 1490}, // exatamente o teto
		{40000, 1490}, // 20,00 → teto 14,90
	}
	for _, c := range casos {
		if got := r.Garantia(c.preco); got != c.garantia {
			t.Errorf("Garantia(%d) = %d, esperado %d", c.preco, got, c.garantia)
		}
	}
}

// TestCalcularCenariosSecao13 cobre os cenários de aceitação do plano.
func TestCalcularCenariosSecao13(t *testing.T) {
	r := regraPadrao(t)
	comprador := TaxasEvento{Regra: r}
	organizador := TaxasEvento{Regra: r, Absorvida: true}
	limiteAL := int64(1000)
	alagoas := TaxasEvento{Regra: r, LimiteBP: &limiteAL}

	casos := []struct {
		nome      string
		t         TaxasEvento
		preco     int64
		garantia  bool
		taxa, gar int64
		absorvida bool
		total     int64
	}{
		{"R$30 sem garantia", comprador, 3000, false, 99, 0, false, 3099},
		{"R$30 com garantia", comprador, 3000, true, 99, 199, false, 3298},
		{"gratuito", comprador, 0, false, 0, 0, false, 0},
		{"gratuito pedindo garantia não cobra nada", comprador, 0, true, 0, 0, false, 0},
		{"R$10 com garantia (cadastro pago usa as faixas)", comprador, 1000, true, 99, 199, false, 1298},
		{"R$50 sem garantia", comprador, 5000, false, 199, 0, false, 5199},
		{"R$50 com garantia", comprador, 5000, true, 199, 250, false, 5449},
		{"R$150 com garantia", comprador, 15000, true, 399, 750, false, 16149},
		{"R$400 com garantia no teto", comprador, 40000, true, 699, 1490, false, 42189},
		{"organizador absorve R$30", organizador, 3000, false, 99, 0, true, 3000},
		{"organizador absorve R$30 com garantia", organizador, 3000, true, 99, 199, true, 3199},
		{"organizador absorve gratuito", organizador, 0, false, 0, 0, false, 0},
		{"AL: R$5 limitado a 10% (0,50)", alagoas, 500, false, 50, 0, false, 550},
		{"AL: R$30 dentro do limite", alagoas, 3000, false, 99, 0, false, 3099},
		{"AL: R$1 limitado a 0,10", alagoas, 100, false, 10, 0, false, 110},
		{"AL com organizador absorvendo: sem limite", TaxasEvento{Regra: r, LimiteBP: &limiteAL, Absorvida: true}, 500, false, 99, 0, true, 500},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			v := c.t.Calcular(c.preco, c.garantia)
			if v.TaxaCentavos != c.taxa || v.GarantiaCentavos != c.gar || v.TaxaAbsorvida != c.absorvida || v.TotalCentavos != c.total {
				t.Errorf("Calcular(%d, %v) = %+v, esperado taxa=%d garantia=%d absorvida=%v total=%d",
					c.preco, c.garantia, v, c.taxa, c.gar, c.absorvida, c.total)
			}
		})
	}
}

func TestDecomporReembolso(t *testing.T) {
	comum := &domain.ItemPedido{PrecoCentavos: 3000, TaxaPlataformaCentavos: 99, GarantiaContratada: true, GarantiaCentavos: 199}
	absorvido := &domain.ItemPedido{PrecoCentavos: 3000, TaxaPlataformaCentavos: 99, TaxaAbsorvida: true, GarantiaContratada: true, GarantiaCentavos: 199}

	casos := []struct {
		nome               string
		item               *domain.ItemPedido
		valor              int64
		preco, taxa, garan int64
	}{
		{"garantia: tudo de volta", comum, 3298, 3000, 99, 199},
		{"CDC com taxa", comum, 3099, 3000, 99, 0},
		{"CDC sem taxa", comum, 3000, 3000, 0, 0},
		{"absorvida com garantia: comprador não pagou taxa", absorvido, 3199, 3000, 0, 199},
		{"absorvida sem garantia", absorvido, 3000, 3000, 0, 0},
	}
	for _, c := range casos {
		p, tx, g := decomporReembolso(c.item, c.valor)
		if p != c.preco || tx != c.taxa || g != c.garan {
			t.Errorf("%s: (%d,%d,%d), esperado (%d,%d,%d)", c.nome, p, tx, g, c.preco, c.taxa, c.garan)
		}
	}
}

func TestLancamentosDoReembolsoTaxaAbsorvida(t *testing.T) {
	evento := &domain.Evento{ID: 1, OrganizadorID: 2}
	item := &domain.ItemPedido{ID: 3, PrecoCentavos: 3000, TaxaPlataformaCentavos: 99, TaxaAbsorvida: true}
	ls := lancamentosDoReembolso(item, evento, 4, 3000, time.Time{})
	soma := map[domain.TipoLancamento]int64{}
	for _, l := range ls {
		v := l.ValorCentavos
		if l.Sinal == "-" {
			v = -v
		}
		soma[l.Tipo] += v
	}
	if soma[domain.LancamentoReembolsoPreco] != -3000 || soma[domain.LancamentoReembolsoTaxa] != -99 || soma[domain.LancamentoEstornoTaxaAbsorvida] != 99 {
		t.Errorf("ledger inesperado: %+v", soma)
	}
}
