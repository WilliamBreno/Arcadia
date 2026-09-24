package service

import (
	"testing"
	"time"
)

func TestAlocarTaxaProcessador(t *testing.T) {
	// pagamento de R$ 100 com taxa R$ 5; item de R$ 40 leva R$ 2
	if got := alocarTaxaProcessador(500, 4000, 10000); got != 200 {
		t.Errorf("got %d, want 200", got)
	}
	if got := alocarTaxaProcessador(500, 4000, 0); got != 0 {
		t.Errorf("pagamento zerado deve dar 0, got %d", got)
	}
}

func TestAdicionarDiasUteis(t *testing.T) {
	sexta := time.Date(2026, 9, 25, 23, 59, 59, 0, time.UTC)
	if got := adicionarDiasUteis(sexta, 3); got.Weekday() != time.Wednesday {
		t.Errorf("sexta + 3 úteis deve ser quarta, got %s", got.Weekday())
	}
	quarta := time.Date(2026, 9, 23, 0, 0, 0, 0, time.UTC)
	if got := adicionarDiasUteis(quarta, 3); got.Weekday() != time.Monday {
		t.Errorf("quarta + 3 úteis deve ser segunda, got %s", got.Weekday())
	}
}
