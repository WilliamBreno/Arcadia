package domain

import "testing"

func TestComissaoDoItem(t *testing.T) {
	pct := &Afiliado{ComissaoTipo: ComissaoPercentual, ComissaoBP: 1000}     // 10%
	fixo := &Afiliado{ComissaoTipo: ComissaoFixo, ComissaoFixaCentavos: 300} // R$ 3
	nada := &Afiliado{ComissaoTipo: ComissaoNenhuma}
	casos := []struct {
		a     *Afiliado
		preco int64
		quer  int64
	}{
		{pct, 5000, 500},
		{pct, 1999, 199}, // arredonda para baixo
		{pct, 0, 0},      // gratuito não gera comissão
		{fixo, 5000, 300},
		{fixo, 200, 200}, // nunca passa do preço
		{fixo, 0, 0},
		{nada, 5000, 0},
		{nil, 5000, 0}, // compra sem divulgador
	}
	for i, c := range casos {
		if got := c.a.ComissaoDoItem(c.preco); got != c.quer {
			t.Errorf("caso %d: %d, esperado %d", i, got, c.quer)
		}
	}
}
