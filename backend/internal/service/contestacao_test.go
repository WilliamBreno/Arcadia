package service

import (
	"errors"
	"testing"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
)

func TestLimitePorConta(t *testing.T) {
	casos := []struct {
		nome    string
		maximo  int
		jaTem   int64
		pedidos int
		passa   bool
	}{
		{"sem limite", 0, 500, 50, true},
		{"cabe exatamente", 10, 6, 4, true},
		{"passa de um", 10, 6, 5, false},
		{"já no limite", 10, 10, 1, false},
		{"primeira compra grande demais", 10, 0, 11, false},
	}
	for _, c := range casos {
		err := limitePorConta(c.maximo, c.jaTem, c.pedidos)
		if (err == nil) != c.passa {
			t.Errorf("%s: err = %v", c.nome, err)
		}
		if err != nil && !errors.Is(err, ErrLimitePorConta) {
			t.Errorf("%s: erro deveria ser ErrLimitePorConta: %v", c.nome, err)
		}
	}
	if err := limitePorConta(10, 7, 5); err == nil || err.Error() != "limite de ingressos por conta neste evento: são no máximo 10 por conta e você ainda pode reservar 3" {
		t.Errorf("mensagem inesperada: %v", err)
	}
}

func TestStatusContestacao(t *testing.T) {
	casos := map[string]string{
		"in_mediation": domain.ContestacaoEmDisputa,
		"charged_back": domain.ContestacaoPerdida,
		"approved":     "",
		"pending":      "",
		"refunded":     "",
	}
	for mp, esperado := range casos {
		if got := statusContestacao(mp); got != esperado {
			t.Errorf("statusContestacao(%q) = %q, esperado %q", mp, got, esperado)
		}
	}
}
