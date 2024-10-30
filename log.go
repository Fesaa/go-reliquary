package reliquary

import (
	"context"
	"log/slog"
	"os"
)

var (
	LevelTrace slog.Level = -8

	logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelWarn,
	})).With("module", "go-reliquary")
)

func SetLogger(l *slog.Logger) {
	logger = l.With("module", "go-reliquary")
}

func trace(msg string, args ...interface{}) {
	logger.Log(context.Background(), LevelTrace, msg, args...)
}
