package service

import "testing"

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
