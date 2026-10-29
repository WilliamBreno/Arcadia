package service

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
)

func TestNormalizarEmail(t *testing.T) {
	if got := NormalizarEmail("  Ana.Silva@Exemplo.COM "); got != "ana.silva@exemplo.com" {
		t.Errorf("got %q", got)
	}
}

func TestCodigoLoginTemSeisDigitos(t *testing.T) {
	re := regexp.MustCompile(`^\d{6}$`)
	vistos := map[string]bool{}
	for i := 0; i < 200; i++ {
		c, err := gerarCodigoLogin()
		if err != nil || !re.MatchString(c) {
			t.Fatalf("código %q, erro %v", c, err)
		}
		vistos[c] = true
	}
	if len(vistos) < 190 {
		t.Errorf("códigos repetindo demais: %d diferentes em 200", len(vistos))
	}
}

func TestHashCodigo(t *testing.T) {
	h := hashCodigo("s", "a@b.c", "123456")
	if len(h) != 64 || strings.Contains(h, "123456") {
		t.Fatalf("hash %q", h)
	}
	if h != hashCodigo("s", "a@b.c", "123456") {
		t.Error("mesmo código, mesmo hash")
	}
	if h == hashCodigo("s", "x@b.c", "123456") || h == hashCodigo("s", "a@b.c", "123457") || h == hashCodigo("t", "a@b.c", "123456") {
		t.Error("e-mail, código ou segredo diferente precisa mudar o hash")
	}
}

func TestParametroRetomar(t *testing.T) {
	got := ParametroRetomar([]ItemRetomar{{TipoID: 12, Quantidade: 2}, {TipoID: 13, Quantidade: 1}})
	if got != "12:2,13:1" {
		t.Errorf("got %q", got)
	}
}

func TestCorpoRecuperacaoEscapa(t *testing.T) {
	inicio := time.Date(2026, 11, 20, 23, 0, 0, 0, time.UTC)
	e := &domain.Evento{Titulo: "<b>Show</b>", InicioEm: &inicio}
	c := CorpoRecuperacao("Ana", e, []ItemRetomar{{Nome: "Pista <vip>", Quantidade: 2, Preco: 5000}}, "https://x/e/show?retomar=1%3A2", "https://x/descadastrar?u=1&t=a", "Evve")
	if strings.Contains(c, "<b>Show</b>") || strings.Contains(c, "<vip>") {
		t.Error("texto do organizador precisa ser escapado")
	}
	for _, esperado := range []string{"2 × Pista", "R$ 50,00", "Continuar a compra", "Não quero receber estes lembretes", "u=1&amp;t=a"} {
		if !strings.Contains(c, esperado) {
			t.Errorf("faltou %q", esperado)
		}
	}
}
