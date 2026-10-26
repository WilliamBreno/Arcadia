package service

import (
	"testing"
	"time"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
)

func TestVendaAberta(t *testing.T) {
	agora := time.Date(2026, 11, 10, 20, 0, 0, 0, time.UTC)
	h := func(d time.Duration) *time.Time { x := agora.Add(d); return &x }

	casos := []struct {
		nome    string
		evento  domain.Evento
		tipoIDs []int64
		sessoes []domain.Sessao
		aberta  bool
	}{
		{"evento futuro", domain.Evento{InicioEm: h(48 * time.Hour)}, nil, nil, true},
		{"em andamento, antes do fim (venda na porta)", domain.Evento{InicioEm: h(-time.Hour), FimEm: h(3 * time.Hour)}, nil, nil, true},
		{"já terminou", domain.Evento{InicioEm: h(-5 * time.Hour), FimEm: h(-time.Hour)}, nil, nil, false},
		{"sem fim: até 12h depois do início", domain.Evento{InicioEm: h(-11 * time.Hour)}, nil, nil, true},
		{"sem fim: mais de 12h depois", domain.Evento{InicioEm: h(-13 * time.Hour)}, nil, nil, false},
		{"sem datas", domain.Evento{}, nil, nil, true},
		{"sessões: ingresso geral com um dia ainda pela frente", domain.Evento{}, nil, []domain.Sessao{
			{ID: 1, Status: domain.SessaoAtiva, InicioEm: agora.Add(-30 * time.Hour), FimEm: h(-24 * time.Hour)},
			{ID: 2, Status: domain.SessaoAtiva, InicioEm: agora.Add(20 * time.Hour)},
		}, true},
		{"sessões: ingresso só do dia que já passou", domain.Evento{}, []int64{1}, []domain.Sessao{
			{ID: 1, Status: domain.SessaoAtiva, InicioEm: agora.Add(-30 * time.Hour), FimEm: h(-24 * time.Hour)},
			{ID: 2, Status: domain.SessaoAtiva, InicioEm: agora.Add(20 * time.Hour)},
		}, false},
		{"sessões: dia futuro foi cancelado", domain.Evento{}, []int64{2}, []domain.Sessao{
			{ID: 1, Status: domain.SessaoAtiva, InicioEm: agora.Add(-30 * time.Hour), FimEm: h(-24 * time.Hour)},
			{ID: 2, Status: domain.SessaoCancelada, InicioEm: agora.Add(20 * time.Hour)},
		}, false},
	}
	for _, c := range casos {
		if got := VendaAberta(&c.evento, c.tipoIDs, c.sessoes, agora); got != c.aberta {
			t.Errorf("%s: %v, esperado %v", c.nome, got, c.aberta)
		}
	}
}
