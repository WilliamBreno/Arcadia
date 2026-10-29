package service

import (
	"testing"
	"time"
)

func TestPrazoParaComprar(t *testing.T) {
	agora := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	longe := agora.Add(10 * 24 * time.Hour)
	if got := PrazoParaComprar(agora, &longe, nil); !got.Equal(agora.Add(2 * time.Hour)) {
		t.Errorf("evento longe: 2 h, veio %v", got.Sub(agora))
	}
	perto := agora.Add(20 * time.Hour)
	if got := PrazoParaComprar(agora, &perto, nil); !got.Equal(agora.Add(30 * time.Minute)) {
		t.Errorf("menos de 24 h: 30 min, veio %v", got.Sub(agora))
	}
	quase := agora.Add(10 * time.Minute)
	if got := PrazoParaComprar(agora, &quase, nil); !got.Equal(quase) {
		t.Errorf("não passa do início do evento, veio %v", got.Sub(agora))
	}
	fimVendas := agora.Add(time.Hour)
	if got := PrazoParaComprar(agora, &longe, &fimVendas); !got.Equal(fimVendas) {
		t.Errorf("não passa do fim das vendas, veio %v", got.Sub(agora))
	}
	if got := PrazoParaComprar(agora, nil, nil); !got.Equal(agora.Add(2 * time.Hour)) {
		t.Errorf("sem data: 2 h, veio %v", got.Sub(agora))
	}
}

func TestLinkConvite(t *testing.T) {
	if got := LinkConvite("https://evve.app", "show", 12, 3); got != "https://evve.app/e/show?retomar=12%3A3" {
		t.Errorf("got %q", got)
	}
}

type promotorFalso struct{ chamado []int64 }

func (p *promotorFalso) PromoverListaEspera(id int64) { p.chamado = append(p.chamado, id) }

func TestPromotoresChamaTodos(t *testing.T) {
	a, b := &promotorFalso{}, &promotorFalso{}
	Promotores{a, nil, b}.PromoverListaEspera(7)
	if len(a.chamado) != 1 || len(b.chamado) != 1 || a.chamado[0] != 7 {
		t.Errorf("a=%v b=%v", a.chamado, b.chamado)
	}
}
