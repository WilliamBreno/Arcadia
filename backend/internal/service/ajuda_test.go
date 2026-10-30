package service

import "testing"

func TestCodigoDoTexto(t *testing.T) {
	casos := map[string]string{
		"ABCD2345":          "ABCD2345",
		"  abcd2345 ":       "ABCD2345",
		"ABCD-2345":         "ABCD2345",
		"ABCD2345:0f9e8d7c": "ABCD2345",
		"https://evve.app/ingresso/ABCD2345/abc123":  "ABCD2345",
		"https://evve.app/ingresso/abcd2345/abc?x=1": "ABCD2345",
		"ABC":            "",
		"ABCD2345XYZ":    "",
		"<script>":       "",
		"ABCD 2345":      "ABCD2345",
		"/ingresso/ABCD": "",
	}
	for entrada, esperado := range casos {
		if got := CodigoDoTexto(entrada); got != esperado {
			t.Errorf("CodigoDoTexto(%q) = %q, esperado %q", entrada, got, esperado)
		}
	}
}

func TestPior(t *testing.T) {
	if pior(StatusOperacional, StatusFalha) != StatusFalha || pior(StatusFalha, StatusDegradado) != StatusFalha || pior("", StatusOperacional) != StatusOperacional {
		t.Fatal("ordem dos estados errada")
	}
}
