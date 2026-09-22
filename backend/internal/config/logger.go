package config

import (
	"log/slog"
	"os"
)

// ConfigurarLogger define o logger padrão da aplicação: texto legível em
// desenvolvimento e JSON estruturado em produção (mais fácil de agregar).
func ConfigurarLogger(cfg Config) {
	var handler slog.Handler
	opcoes := &slog.HandlerOptions{Level: slog.LevelInfo}

	if cfg.AmbienteApp == "production" {
		handler = slog.NewJSONHandler(os.Stdout, opcoes)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opcoes)
	}

	slog.SetDefault(slog.New(handler))
}
