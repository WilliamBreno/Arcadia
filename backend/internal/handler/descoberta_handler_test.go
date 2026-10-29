package handler

import (
	"testing"
	"time"

	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
)

func TestSlugCidade(t *testing.T) {
	casos := map[[2]string]string{
		{"Maceió", "AL"}:                   "maceio-al",
		{"São Paulo", "SP"}:                "sao-paulo-sp",
		{"Nossa Senhora do Socorro", "SE"}: "nossa-senhora-do-socorro-se",
	}
	for in, esperado := range casos {
		if got := slugCidade(in[0], in[1]); got != esperado {
			t.Errorf("slugCidade(%v) = %q, esperado %q", in, got, esperado)
		}
	}
}

func TestResolverCidade(t *testing.T) {
	cs := []repository.CidadeContagem{{Cidade: "Maceió", UF: "AL", Eventos: 3}, {Cidade: "Aracaju", UF: "SE", Eventos: 1}}
	if c := resolverCidade(cs, "maceio-al"); c == nil || c.Cidade != "Maceió" || c.Eventos != 3 {
		t.Errorf("resolverCidade(maceio-al) = %+v", c)
	}
	if c := resolverCidade(cs, "recife-pe"); c != nil {
		t.Errorf("cidade sem eventos deveria ser nil: %+v", c)
	}
}

func TestIntervaloDoAtalho(t *testing.T) {
	brt := fusoPadrao
	d := func(dia, hora int) time.Time { return time.Date(2026, 10, dia, hora, 0, 0, 0, brt) }
	casos := []struct {
		nome, atalho string
		agora        time.Time
		de, ate      time.Time
	}{
		// 23h de quarta em Maceió já é quinta em UTC: "hoje" tem que ser quarta.
		{"hoje à noite (UTC já virou)", "hoje", d(28, 23), d(28, 0), d(29, 0)},
		{"amanhã", "amanha", d(28, 10), d(29, 0), d(30, 0)},
		{"fim de semana numa quarta", "fim-de-semana", d(28, 10), d(31, 0), time.Date(2026, 11, 2, 0, 0, 0, 0, brt)},
		{"fim de semana no sábado", "fim-de-semana", d(31, 15), d(31, 0), time.Date(2026, 11, 2, 0, 0, 0, 0, brt)},
		{"fim de semana no domingo (não pula)", "fim-de-semana", time.Date(2026, 11, 1, 15, 0, 0, 0, brt), d(31, 0), time.Date(2026, 11, 2, 0, 0, 0, 0, brt)},
	}
	for _, c := range casos {
		de, ate := intervaloDoAtalho(c.atalho, c.agora.UTC())
		if de == nil || ate == nil || !de.Equal(c.de) || !ate.Equal(c.ate) {
			t.Errorf("%s: %v → %v, esperado %v → %v", c.nome, de, ate, c.de, c.ate)
		}
	}
	if de, ate := intervaloDoAtalho("qualquer", time.Now()); de != nil || ate != nil {
		t.Error("atalho desconhecido deveria ser nil")
	}
}
