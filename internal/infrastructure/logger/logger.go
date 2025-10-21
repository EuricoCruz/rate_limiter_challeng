package logger

import (
	"log/slog"
	"os"
)

// New cria um logger estruturado para produção
func New() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
}

// NewDevelopment cria um logger mais verboso para desenvolvimento
func NewDevelopment() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
}
