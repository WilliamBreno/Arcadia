package service_test

import (
	"testing"

	"github.com/WilliamBreno/Arcadia/backend/internal/service"
)

func TestGerarTokenOpacoEUnico(t *testing.T) {
	a, err := service.GerarTokenOpaco()
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	b, err := service.GerarTokenOpaco()
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if a == b {
		t.Fatal("dois tokens gerados são iguais — gerador não está aleatório")
	}
	if len(a) < 32 {
		t.Fatalf("token muito curto: %d caracteres", len(a))
	}
}

func TestHashTokenDeterministico(t *testing.T) {
	token := "token-de-teste"
	if service.HashToken(token) != service.HashToken(token) {
		t.Fatal("HashToken não é determinístico para a mesma entrada")
	}
	if service.HashToken(token) == service.HashToken(token+"x") {
		t.Fatal("HashToken produziu o mesmo hash para entradas diferentes")
	}
}
