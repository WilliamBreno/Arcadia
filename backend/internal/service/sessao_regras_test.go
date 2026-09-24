package service

import (
	"testing"
	"time"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
)

func TestSessaoAtualEPeriodo(t *testing.T) {
	base := time.Date(2026, 12, 5, 20, 0, 0, 0, time.UTC)
	fim1 := base.Add(4 * time.Hour)
	dia2 := base.Add(24 * time.Hour)
	sessoes := []domain.Sessao{
		{ID: 1, InicioEm: base, FimEm: &fim1, Status: domain.SessaoAtiva},
		{ID: 2, InicioEm: dia2, Status: domain.SessaoAtiva},
		{ID: 3, InicioEm: dia2.Add(24 * time.Hour), Status: domain.SessaoCancelada},
	}

	if s := SessaoAtual(sessoes, base.Add(-2*time.Hour)); s == nil || s.ID != 1 {
		t.Errorf("2h antes da sessão 1 deve ser a sessão 1: %+v", s)
	}
	if s := SessaoAtual(sessoes, base.Add(-5*time.Hour)); s != nil {
		t.Errorf("5h antes ainda é cedo: %+v", s)
	}
	if s := SessaoAtual(sessoes, base.Add(10*time.Hour)); s != nil {
		t.Errorf("depois do fim e antes da sessão 2 não há sessão: %+v", s)
	}
	if s := SessaoAtual(sessoes, dia2.Add(time.Hour)); s == nil || s.ID != 2 {
		t.Errorf("durante a sessão 2: %+v", s)
	}
	if s := SessaoAtual(sessoes, dia2.Add(25*time.Hour)); s != nil {
		t.Errorf("sessão cancelada nunca é atual: %+v", s)
	}

	ini, fim, ok := PeriodoDoEvento(sessoes)
	if !ok || !ini.Equal(base) || !fim.Equal(dia2) {
		t.Errorf("período = %v..%v ok=%v (sessão cancelada não conta; sem fim usa o início)", ini, fim, ok)
	}
	if _, _, ok := PeriodoDoEvento(nil); ok {
		t.Error("sem sessões não há período")
	}
}

func TestTipoValeNaSessao(t *testing.T) {
	if !tipoValeNaSessao(nil, 7) {
		t.Error("sem lista vale em todas as sessões")
	}
	if !tipoValeNaSessao([]int64{1, 3}, 3) || tipoValeNaSessao([]int64{1, 3}, 2) {
		t.Error("com lista, só nas sessões listadas")
	}
}
