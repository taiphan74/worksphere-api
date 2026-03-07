package logger

import (
	"log/slog"
	"os"
	"strings"
)

func New(env string) *slog.Logger {
	level := new(slog.LevelVar)
	level.Set(resolveLevel(env))

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})

	return slog.New(handler)
}

func resolveLevel(env string) slog.Level {
	switch strings.ToLower(env) {
	case "production":
		return slog.LevelInfo
	case "staging":
		return slog.LevelInfo
	default:
		return slog.LevelDebug
	}
}
