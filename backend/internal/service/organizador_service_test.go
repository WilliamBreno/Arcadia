package service

import "testing"

func TestSlugify(t *testing.T) {
	casos := map[string]string{
		"Festival de Cosplay & Cia": "festival-de-cosplay-cia",
		"  Ação Geek  ":             "acao-geek",
		"São João":                  "sao-joao",
		"":                          "organizador",
	}
	for entrada, esperado := range casos {
		if got := slugify(entrada); got != esperado {
			t.Errorf("slugify(%q) = %q, esperado %q", entrada, got, esperado)
		}
	}
}

func TestDocumentoValido(t *testing.T) {
	if !documentoValido("pf", "123.456.789-00") {
		t.Error("CPF com 11 dígitos deveria ser válido")
	}
	if documentoValido("pf", "123") {
		t.Error("CPF curto não deveria ser válido")
	}
	if !documentoValido("pj", "12.345.678/0001-90") {
		t.Error("CNPJ com 14 dígitos deveria ser válido")
	}
	if documentoValido("pj", "123.456.789-00") {
		t.Error("CPF não deveria ser válido como CNPJ")
	}
}
