package service

import "testing"

func TestCorTemaOuPadrao(t *testing.T) {
	ok, ruim, vazio := "#FF00AA", "red; background:url(x)", ""
	if got := corTemaOuPadrao(&ok, "#000000"); got != "#ff00aa" {
		t.Errorf("cor válida: %q", got)
	}
	if got := corTemaOuPadrao(&ruim, "#123456"); got != "#123456" {
		t.Errorf("cor inválida deve manter a atual: %q", got)
	}
	if got := corTemaOuPadrao(&vazio, "#123456"); got != "" {
		t.Errorf("vazio limpa o tema: %q", got)
	}
	if got := corTemaOuPadrao(nil, "#123456"); got != "#123456" {
		t.Errorf("nil mantém: %q", got)
	}
}
