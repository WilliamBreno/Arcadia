package service

import "testing"

func TestCelulaCSVNeutralizaFormulas(t *testing.T) {
	for _, c := range []string{"=1+1", "+cmd", "-2", "@SUM(A1)"} {
		if got := celulaCSV(c); got != "'"+c {
			t.Errorf("celulaCSV(%q) = %q", c, got)
		}
	}
	if got := celulaCSV("Maria"); got != "Maria" {
		t.Errorf("texto normal não deve mudar, got %q", got)
	}
}

func TestReais(t *testing.T) {
	if got := reais(3298); got != "32,98" {
		t.Errorf("got %s", got)
	}
	if got := reais(5); got != "0,05" {
		t.Errorf("got %s", got)
	}
}
