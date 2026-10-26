package service

import (
	"testing"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
)

func TestOrganizadorNovo(t *testing.T) {
	casos := []struct {
		verificado bool
		pagos      int64
		novo       bool
	}{
		{false, 0, true},
		{false, 10, true}, // sem verificação é sempre novo
		{true, 0, true},
		{true, 2, true},
		{true, 3, false},
		{true, 8, false},
	}
	for _, c := range casos {
		if got := organizadorNovo(c.verificado, c.pagos, 3); got != c.novo {
			t.Errorf("organizadorNovo(%v, %d) = %v, esperado %v", c.verificado, c.pagos, got, c.novo)
		}
	}
}

func TestCalcularReserva(t *testing.T) {
	casos := []struct {
		nome                     string
		brutoCartao, liquido, bp int64
		novo                     bool
		reserva                  int64
	}{
		{"10% do cartão de organizador novo", 20000, 29000, 1000, true, 2000},
		{"organizador experiente não tem reserva", 20000, 29000, 1000, false, 0},
		{"só Pix: nada a reservar", 0, 29000, 1000, true, 0},
		{"nunca maior que o líquido", 20000, 1500, 1000, true, 1500},
		{"líquido zero", 20000, 0, 1000, true, 0},
		{"centavos arredondados para baixo", 999, 5000, 1000, true, 99},
	}
	for _, c := range casos {
		if got := calcularReserva(c.brutoCartao, c.liquido, c.bp, c.novo); got != c.reserva {
			t.Errorf("%s: %d, esperado %d", c.nome, got, c.reserva)
		}
	}
}

func TestValorAPagarAgora(t *testing.T) {
	r := &domain.Repasse{ValorLiquidoCentavos: 29000, ReservaCentavos: 2000}
	if v := ValorAPagarAgora(r); v != 27000 {
		t.Errorf("ValorAPagarAgora = %d, esperado 27000", v)
	}
}
