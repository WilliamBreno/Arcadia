package service

import "testing"

func TestCalcularEconomiaPix(t *testing.T) {
	casos := []struct {
		nome                                      string
		pixValor, pixTaxa, cartaoValor, cartaoTax int64
		economia, bp                              int64
		estimada                                  bool
	}{
		{"sem venda no cartão usa a referência", 100000, 0, 0, 0, 3980, 398, true},
		{"com venda no cartão usa a média real", 100000, 500, 50000, 2000, 3500, 400, false},
		{"Pix mais caro que cartão não fica negativo", 1000, 500, 10000, 100, 0, 100, false},
		{"sem venda nenhuma", 0, 0, 0, 0, 0, 398, true},
	}
	for _, c := range casos {
		e, bp, est := calcularEconomiaPix(c.pixValor, c.pixTaxa, c.cartaoValor, c.cartaoTax, 398)
		if e != c.economia || bp != c.bp || est != c.estimada {
			t.Errorf("%s: (%d, %d, %v), esperado (%d, %d, %v)", c.nome, e, bp, est, c.economia, c.bp, c.estimada)
		}
	}
}
