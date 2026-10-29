package service

import (
	"testing"
	"time"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
)

func TestNomeAvaliador(t *testing.T) {
	casos := map[string]string{
		"":                     "Participante",
		"  ":                   "Participante",
		"Maria":                "Maria",
		"Maria da Silva":       "Maria S.",
		"joão  évora":          "joão É.",
		"Ana Clara Souza Lima": "Ana L.",
	}
	for entrada, esperado := range casos {
		if got := NomeAvaliador(entrada); got != esperado {
			t.Errorf("NomeAvaliador(%q) = %q, esperado %q", entrada, got, esperado)
		}
	}
}

func TestJanelaAvaliacao(t *testing.T) {
	inicio := time.Date(2026, 10, 10, 20, 0, 0, 0, time.UTC)
	fim := inicio.Add(4 * time.Hour)
	e := &domain.Evento{InicioEm: &inicio, FimEm: &fim}
	casos := []struct {
		agora time.Time
		ok    bool
	}{
		{inicio.Add(-time.Minute), false},
		{inicio, true},
		{fim.AddDate(0, 0, 30), true},
		{fim.AddDate(0, 0, 30).Add(time.Second), false},
	}
	for _, c := range casos {
		if ok, _ := janelaAvaliacao(e, c.agora); ok != c.ok {
			t.Errorf("janela em %v = %v, esperado %v", c.agora, ok, c.ok)
		}
	}
	if ok, _ := janelaAvaliacao(&domain.Evento{}, inicio); ok {
		t.Error("evento sem data não pode abrir avaliação")
	}
	// Sem fim: conta 30 dias a partir do início.
	semFim := &domain.Evento{InicioEm: &inicio}
	if ok, _ := janelaAvaliacao(semFim, inicio.AddDate(0, 0, 31)); ok {
		t.Error("sem fim, a janela fecha 30 dias depois do início")
	}
}

func TestResumir(t *testing.T) {
	if r := resumir(2, 10); r.MostrarMedia || r.Media != 0 {
		t.Errorf("com 2 avaliações não mostra média: %+v", r)
	}
	if r := resumir(3, 14); !r.MostrarMedia || r.Media != 4.7 {
		t.Errorf("3 avaliações somando 14 = 4.7: %+v", r)
	}
	if r := resumir(4, 17); r.Media != 4.3 {
		t.Errorf("17/4 = 4.25 arredonda para 4.3: %+v", r)
	}
	if r := resumir(3, 15); r.Media != 5 {
		t.Errorf("tudo 5 = 5.0: %+v", r)
	}
	if r := resumir(0, 0); r.Total != 0 || r.MostrarMedia {
		t.Errorf("vazio: %+v", r)
	}
}

func TestDeveConvidar(t *testing.T) {
	inicio := time.Date(2026, 10, 10, 20, 0, 0, 0, time.UTC)
	fim := inicio.Add(4 * time.Hour)
	e := &domain.Evento{InicioEm: &inicio, FimEm: &fim}
	casos := []struct {
		depois time.Duration
		ok     bool
	}{
		{2 * time.Hour, false},
		{3 * time.Hour, true},
		{48 * time.Hour, true},
		{49 * time.Hour, false},
	}
	for _, c := range casos {
		if got := deveConvidar(e, fim.Add(c.depois)); got != c.ok {
			t.Errorf("convite %v depois do fim = %v, esperado %v", c.depois, got, c.ok)
		}
	}
	semFim := &domain.Evento{InicioEm: &inicio}
	if !deveConvidar(semFim, inicio.Add(6*time.Hour)) {
		t.Error("sem fim, conta o fim como início + 3 h")
	}
}
