package logging

import (
	"io"
	"log/slog"
	"os"
)

var handler slog.Handler

func init() {
	level := slog.LevelInfo
	if os.Getenv("DEBUG") != "" {
		level = slog.LevelDebug
	}

	handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})
}

type Logger = slog.Logger

func New() *Logger {
	return slog.New(handler)
}

func Noop() *Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
