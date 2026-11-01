package service

import (
	"testing"
	"time"
)

func TestDiaDestaquePreservaCalendarioDaCompra(t *testing.T) {
	br := time.FixedZone("BRT", -3*3600)
	viradaUTC := time.Date(2026, 11, 1, 1, 0, 0, 0, time.UTC)
	if got := diaDestaque(viradaUTC.In(br)); got != "2026-10-31" {
		t.Fatalf("dia no calendário da compra=%s; esperado 2026-10-31", got)
	}
	if got := diaDestaque(viradaUTC); got != "2026-11-01" {
		t.Fatalf("dia do calendário UTC=%s; esperado 2026-11-01", got)
	}
}
