package service

import (
	"strings"
	"testing"
	"time"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
)

func TestValidadePixRespeitaMinimoDoMP(t *testing.T) {
	if ValidadePix < 30*time.Minute {
		t.Fatalf("o Mercado Pago recusa Pix com menos de 30 minutos (atual: %v)", ValidadePix)
	}
}

func TestChaveCartao(t *testing.T) {
	a, b := chaveCartao(1, "tok-1"), chaveCartao(1, "tok-1")
	if a != b {
		t.Error("mesmo token precisa dar a mesma chave (reenvio não cobra duas vezes)")
	}
	if chaveCartao(1, "tok-2") == a || chaveCartao(2, "tok-1") == a {
		t.Error("token ou pedido diferente precisa dar outra chave")
	}
	if strings.Contains(a, "tok-1") || len(a) > 64 {
		t.Errorf("a chave não expõe o token e cabe no limite do MP: %q", a)
	}
}

func TestMensagemRecusa(t *testing.T) {
	if !strings.Contains(MensagemRecusa("cc_rejected_insufficient_amount"), "limite") {
		t.Error("saldo insuficiente")
	}
	if !strings.Contains(MensagemRecusa("cc_rejected_call_for_authorize"), "banco") {
		t.Error("autorizar com o banco")
	}
	if MensagemRecusa("qualquer_coisa") == "" || strings.Contains(MensagemRecusa("x"), "cc_") {
		t.Error("motivo desconhecido tem texto próprio, sem código técnico")
	}
}

func TestTextoDevolucao(t *testing.T) {
	for _, m := range []string{domain.DevolucaoAposReserva, domain.DevolucaoDuplicado, domain.DevolucaoValorDivergente} {
		if TextoDevolucao(m) == TextoDevolucao("") {
			t.Errorf("motivo %s sem texto próprio", m)
		}
	}
}
