// Comando migrate aplica ou reverte as migrations em backend/migrations
// contra o Postgres apontado por DATABASE_URL.
//
// Uso:
//
//	go run ./cmd/migrate up
//	go run ./cmd/migrate down
package main

import (
	"errors"
	"log/slog"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/WilliamBreno/Arcadia/backend/internal/config"
)

func main() {
	cfg := config.Carregar()
	config.ConfigurarLogger(cfg)

	if len(os.Args) < 2 {
		slog.Error("uso: migrate up|down")
		os.Exit(1)
	}

	m, err := migrate.New("file://migrations", cfg.DatabaseURL)
	if err != nil {
		slog.Error("erro ao carregar migrations", "erro", err)
		os.Exit(1)
	}

	switch os.Args[1] {
	case "up":
		err = m.Up()
	case "down":
		err = m.Down()
	default:
		slog.Error("comando desconhecido", "comando", os.Args[1])
		os.Exit(1)
	}

	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		slog.Error("erro ao rodar migration", "erro", err)
		os.Exit(1)
	}

	slog.Info("migrations aplicadas com sucesso", "comando", os.Args[1])
}
