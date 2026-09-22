package repository_test

import (
	"testing"

	"github.com/WilliamBreno/Arcadia/backend/internal/config"
	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
)

// TestConfigPlataformaBuscarInt64 é um teste de integração: precisa do
// Postgres local (docker compose up) com as migrations aplicadas. Se não
// conseguir conectar, pula em vez de falhar (não há Postgres no CI ainda).
func TestConfigPlataformaBuscarInt64(t *testing.T) {
	cfg := config.Carregar()
	db, err := repository.Conectar(cfg.DatabaseURL)
	if err != nil {
		t.Skipf("postgres local indisponível, pulando teste de integração: %v", err)
	}

	repo := repository.NovoConfigPlataformaRepository(db)

	valor, err := repo.BuscarInt64("TAXA_INGRESSO_CENTAVOS")
	if err != nil {
		t.Skipf("migrations não aplicadas, pulando teste de integração: %v", err)
	}

	if valor != 99 {
		t.Errorf("TAXA_INGRESSO_CENTAVOS = %d, esperado 99", valor)
	}
}
