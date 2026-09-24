package service

import (
	"testing"
	"time"
)

func TestTokenRotativo(t *testing.T) {
	agora := time.Unix(1_800_000_000, 0)
	tok := tokenRotativo("segredo", "ABC12345", janelaQR(agora))

	if validarTokenRotativo("segredo", "ABC12345", tok, agora) != qrValido {
		t.Error("token da janela atual deve valer")
	}
	if validarTokenRotativo("segredo", "ABC12345", tok, agora.Add(40*time.Second)) != qrValido {
		t.Error("janela vizinha (tolerância de relógio) deve valer")
	}
	if validarTokenRotativo("segredo", "ABC12345", tok, agora.Add(3*time.Minute)) != qrExpirado {
		t.Error("token antigo deve expirar")
	}
	if validarTokenRotativo("outro", "ABC12345", tok, agora) != qrInvalido {
		t.Error("segredo diferente deve ser inválido")
	}
	if validarTokenRotativo("segredo", "OUTROCOD", tok, agora) != qrInvalido {
		t.Error("token de outro código deve ser inválido")
	}
	if validarTokenRotativo("segredo", "ABC12345", "R123.zzz", agora) != qrInvalido {
		t.Error("lixo deve ser inválido")
	}
}

func TestPayloadQR(t *testing.T) {
	agora := time.Unix(1_800_000_000, 0)
	if got := PayloadQR("s", "COD", "estatico", false, agora); got != "COD:estatico" {
		t.Errorf("estático: %s", got)
	}
	if got := PayloadQR("s", "COD", "estatico", true, agora); got == "COD:estatico" || got[:5] != "COD:R" {
		t.Errorf("rotativo: %s", got)
	}
}
