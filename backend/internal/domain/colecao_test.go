package domain

import (
	"testing"
	"time"
)

func data(s string) *time.Time {
	t, _ := time.Parse("2006-01-02", s)
	return &t
}

func TestColecaoVisivelEm(t *testing.T) {
	c := Colecao{Ativa: true, ExibirDe: data("2026-10-01"), ExibirAte: data("2026-10-31"), AbertaOrganizador: true, Regra: RegraColecaoManual}
	brt := time.FixedZone("BRT", -3*3600)
	casos := []struct {
		nome   string
		agora  time.Time
		espera bool
	}{
		{"antes", time.Date(2026, 9, 30, 23, 59, 0, 0, brt), false},
		{"primeiro dia", time.Date(2026, 10, 1, 0, 1, 0, 0, brt), true},
		{"último dia à noite (já é dia 1 em UTC)", time.Date(2026, 10, 31, 23, 0, 0, 0, brt), true},
		{"depois", time.Date(2026, 11, 1, 0, 1, 0, 0, brt), false},
	}
	for _, x := range casos {
		if got := c.VisivelEm(x.agora); got != x.espera {
			t.Errorf("%s: obtido %v, esperado %v", x.nome, got, x.espera)
		}
	}
	if (&Colecao{Ativa: false}).VisivelEm(time.Now()) {
		t.Error("inativa não aparece")
	}
	if !(&Colecao{Ativa: true}).VisivelEm(time.Now()) {
		t.Error("sem período aparece sempre")
	}
	antes := time.Date(2026, 9, 1, 12, 0, 0, 0, brt)
	if !c.AceitaMarcacaoEm(antes) {
		t.Error("tema pode ser marcado antes da época")
	}
	if c.AceitaMarcacaoEm(time.Date(2026, 11, 1, 12, 0, 0, 0, brt)) {
		t.Error("tema encerrado não aceita marcação")
	}
	auto := c
	auto.Regra = RegraColecaoGratuitos
	if auto.AceitaMarcacaoEm(antes) {
		t.Error("coleção automática não aceita marcação")
	}
}
