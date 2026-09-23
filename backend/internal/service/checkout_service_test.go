package service

import "testing"

// TestCalcularTotalItem cobre os cenários de aceitação da seção 13 do
// plano: "Ingresso R$ 30 sem garantia → total R$ 30,99; com garantia →
// R$ 32,98. Cadastro gratuito → R$ 0,49; cadastro R$ 10 com garantia →
// R$ 12,48." (valores em centavos: taxa ingresso 99, taxa cadastro 49,
// garantia 199 — seção 2.1).
func TestCalcularTotalItem(t *testing.T) {
	casos := []struct {
		nome               string
		precoCentavos      int64
		taxaCentavos       int64
		garantiaCentavos   int64
		garantiaContratada bool
		esperado           int64
	}{
		{"ingresso R$30 sem garantia", 3000, 99, 199, false, 3099},
		{"ingresso R$30 com garantia", 3000, 99, 199, true, 3298},
		{"cadastro gratuito", 0, 49, 199, false, 49},
		{"cadastro R$10 com garantia", 1000, 49, 199, true, 1248},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			total := calcularTotalItem(c.precoCentavos, c.taxaCentavos, c.garantiaCentavos, c.garantiaContratada)
			if total != c.esperado {
				t.Errorf("total = %d, esperado %d", total, c.esperado)
			}
		})
	}
}

func TestGerarCodigoItemFormatoEUnicidade(t *testing.T) {
	a, err := gerarCodigoItem()
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	b, err := gerarCodigoItem()
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	if len(a) != 8 {
		t.Errorf("código com %d caracteres, esperado 8", len(a))
	}
	if a == b {
		t.Error("dois códigos gerados são iguais — gerador não está aleatório")
	}
	for _, ambiguo := range []rune{'O', '0', 'I', '1'} {
		for _, r := range a {
			if r == ambiguo {
				t.Errorf("código %q contém caractere ambíguo %q", a, ambiguo)
			}
		}
	}
}
