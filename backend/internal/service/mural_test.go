package service

import (
	"testing"
	"time"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
)

func TestJanelaMural(t *testing.T) {
	inicio := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	fim := inicio.Add(8 * time.Hour)
	e := &domain.Evento{Status: domain.StatusEventoPublicado, InicioEm: &inicio, FimEm: &fim}
	for _, c := range []struct {
		nome  string
		agora time.Time
		quer  bool
	}{{"antes", inicio.Add(-time.Second), false}, {"abertura", inicio, true}, {"durante", fim, true}, {"limite", fim.AddDate(0, 0, 7), true}, {"apos", fim.AddDate(0, 0, 7).Add(time.Second), false}} {
		t.Run(c.nome, func(t *testing.T) {
			ok, _ := janelaMural(e, c.agora)
			if ok != c.quer {
				t.Fatalf("janela=%v, esperado %v", ok, c.quer)
			}
		})
	}
	e.Status = domain.StatusEventoCancelado
	if ok, _ := janelaMural(e, inicio); ok {
		t.Fatal("evento cancelado abriu mural")
	}
	e.Status = domain.StatusEventoPublicado
	e.FimEm = nil
	if ok, _ := janelaMural(e, inicio.AddDate(0, 0, 7).Add(time.Second)); ok {
		t.Fatal("sem fim deve usar início como referência")
	}
}
