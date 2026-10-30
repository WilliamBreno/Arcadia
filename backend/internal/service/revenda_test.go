package service

import (
	"errors"
	"testing"
	"time"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
)

func TestQuantasDaRevenda(t *testing.T) {
	casos := []struct {
		qtd     int
		estoque int64
		quer    int
	}{
		{2, 5, 0},  // o organizador ainda tem: vende dele
		{2, 1, 1},  // um do organizador, um da revenda
		{2, 0, 2},  // tudo da revenda
		{1, -3, 1}, // estoque "negativo" (vagas guardadas) conta como zero
	}
	for _, c := range casos {
		if got := QuantasDaRevenda(c.qtd, c.estoque); got != c.quer {
			t.Errorf("QuantasDaRevenda(%d, %d) = %d, quer %d", c.qtd, c.estoque, got, c.quer)
		}
	}
}

func TestPodeRevender(t *testing.T) {
	agora := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	depois := agora.Add(48 * time.Hour)
	antes := agora.Add(-time.Hour)
	ev := &domain.Evento{Status: domain.StatusEventoPublicado, InicioEm: &depois}
	pago := &domain.ItemPedido{Status: domain.StatusItemPago, PrecoCentavos: 5000}
	if err := PodeRevender(pago, ev, agora); err != nil {
		t.Fatalf("ingresso pago antes do evento pode: %v", err)
	}
	casos := map[string]struct {
		item *domain.ItemPedido
		ev   *domain.Evento
	}{
		"já usado":         {&domain.ItemPedido{Status: domain.StatusItemUtilizado, PrecoCentavos: 5000}, ev},
		"cortesia":         {&domain.ItemPedido{Status: domain.StatusItemPago, PrecoCentavos: 5000, Cortesia: true}, ev},
		"gratuito":         {&domain.ItemPedido{Status: domain.StatusItemPago}, ev},
		"evento começou":   {pago, &domain.Evento{Status: domain.StatusEventoPublicado, InicioEm: &antes}},
		"evento cancelado": {pago, &domain.Evento{Status: domain.StatusEventoCancelado, InicioEm: &depois}},
	}
	for nome, c := range casos {
		if err := PodeRevender(c.item, c.ev, agora); !errors.Is(err, ErrRevendaNaoPermitida) {
			t.Errorf("%s: esperava recusa, veio %v", nome, err)
		}
	}
}

func TestVagasParaComprar(t *testing.T) {
	casos := []struct{ org, rev, guard, quer int64 }{
		{3, 0, 0, 3},
		{0, 1, 0, 1},  // só a revenda
		{0, 1, 1, 0},  // a vaga da revenda está guardada para a fila
		{1, 1, 1, 1},  // guardada consome uma, sobra uma
		{-2, 1, 0, 1}, // estoque do organizador nunca fica negativo
		{0, 0, 2, 0},
	}
	for _, c := range casos {
		if got := VagasParaComprar(c.org, c.rev, c.guard); got != c.quer {
			t.Errorf("VagasParaComprar(%d,%d,%d) = %d, quer %d", c.org, c.rev, c.guard, got, c.quer)
		}
	}
}
