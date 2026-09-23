package service

import (
	"testing"
	"time"
)

// TestDentroDoPrazoCDC cobre os cenários de aceitação 5 e 6 da seção 13:
// "Comprador sem garantia cancela no dia 3 (evento daqui a 20 dias):
// reembolso integral. No dia 10: negado." e "...evento começa em 24h:
// negado (regra das 48h)."
func TestDentroDoPrazoCDC(t *testing.T) {
	agora := time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC)

	casos := []struct {
		nome            string
		diasDaCompra    int
		horasParaEvento float64
		esperado        bool
	}{
		{"dia 3 da compra, evento em 20 dias — dentro do prazo", 3, 20 * 24, true},
		{"dia 10 da compra, evento em 20 dias — fora do prazo de 7 dias", 10, 20 * 24, false},
		{"dia 3 da compra, evento em 24h — fora da regra das 48h", 3, 24, false},
		{"dia 7 da compra (limite exato), evento longe — ainda dentro", 7, 20 * 24, true},
		{"dia 3 da compra, evento em exatas 48h — ainda dentro (limite inclusivo)", 3, 48, true},
		{"sem data de início definida — só checa prazo de compra", 3, 0, true},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			dataCompra := agora.Add(-time.Duration(c.diasDaCompra) * 24 * time.Hour)
			var inicioEvento *time.Time
			if c.horasParaEvento > 0 {
				t := agora.Add(time.Duration(c.horasParaEvento * float64(time.Hour)))
				inicioEvento = &t
			}

			resultado := dentroDoPrazoCDC(dataCompra, inicioEvento, agora)
			if resultado != c.esperado {
				t.Errorf("dentroDoPrazoCDC() = %v, esperado %v", resultado, c.esperado)
			}
		})
	}
}

func TestCalcularValorArrependimento(t *testing.T) {
	if v := calcularValorArrependimento(3000, 99, true); v != 3099 {
		t.Errorf("com reembolso de taxa: %d, esperado 3099", v)
	}
	if v := calcularValorArrependimento(3000, 99, false); v != 3000 {
		t.Errorf("sem reembolso de taxa: %d, esperado 3000", v)
	}
}
