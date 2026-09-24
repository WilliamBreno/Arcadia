package service

import (
	"testing"
	"time"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
)

func TestPodeTransferir(t *testing.T) {
	agora := time.Now()
	futuro := agora.Add(48 * time.Hour)
	passado := agora.Add(-time.Hour)
	pago := &domain.ItemPedido{Status: domain.StatusItemPago}

	if err := podeTransferir(pago, &domain.Evento{InicioEm: &futuro}, agora); err != nil {
		t.Errorf("pago e evento futuro deve poder: %v", err)
	}
	if podeTransferir(&domain.ItemPedido{Status: domain.StatusItemUtilizado}, &domain.Evento{InicioEm: &futuro}, agora) == nil {
		t.Error("utilizado não pode transferir")
	}
	if podeTransferir(pago, &domain.Evento{InicioEm: &passado}, agora) == nil {
		t.Error("evento já iniciado não pode transferir")
	}
	if podeTransferir(pago, &domain.Evento{InicioEm: &futuro, Status: domain.StatusEventoCancelado}, agora) == nil {
		t.Error("evento cancelado não pode transferir")
	}
}
