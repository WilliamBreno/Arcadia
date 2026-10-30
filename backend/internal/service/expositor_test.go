package service

import "testing"

func TestPayloadCredencial(t *testing.T) {
	p := PayloadCredencial("segredo", 42)
	id, err := LerPayloadCredencial("segredo", p)
	if err != nil || id != 42 {
		t.Fatalf("ida e volta: %v %d", err, id)
	}
	for _, ruim := range []string{p[:len(p)-1] + "0", "EVVEP:42:abc", "EVVEX:43:" + p[len("EVVEX:42:"):], "", "EVVEX:x:y"} {
		if _, err := LerPayloadCredencial("segredo", ruim); err == nil {
			t.Errorf("aceitou payload inválido %q", ruim)
		}
	}
	if _, err := LerPayloadCredencial("outro", p); err == nil {
		t.Error("aceitou assinatura de outro segredo")
	}
}
