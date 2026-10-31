package service

import "testing"

func TestDiaDisponibilidade(t *testing.T) {
	for _, d := range []string{"2026-11-01", "2028-02-29", "2026-12-31"} {
		if !diaDisponibilidadeValido(d) {
			t.Errorf("data válida recusada: %s", d)
		}
	}
	for _, d := range []string{"", "2026-02-29", "0000-01-01", "2026-13-01", "2026-11-31", "2026-1-1", "2026-11-01T00:00:00Z", "2026-11-01' OR true--"} {
		if diaDisponibilidadeValido(d) {
			t.Errorf("data inválida aceita: %s", d)
		}
	}
}
