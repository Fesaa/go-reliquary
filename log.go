package reliquary

import (
	"context"
	"log/slog"
	"os"
)

var (
	// LevelTrace a level below slog.LevelDebug, will log every packet received
	LevelTrace slog.Level = -8

	logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	})).With("module", "go-reliquary")
)

// SetLogger sets the library's logger. The argument module with value 'go-reliquary' is always added
func SetLogger(l *slog.Logger) {
	logger = l.With(slog.String("module", "go-reliquary"))
}

// SetLevel sets the logger to the default one with the given slog.Level
// Use SetLogger for more configuration
func SetLevel(level slog.Level) {
	logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
		Level:     level,
	})).With("module", "go-reliquary")
}

func trace(msg string, args ...interface{}) {
	logger.Log(context.Background(), LevelTrace, msg, args...)
}

func traceL(l *slog.Logger, msg string, args ...interface{}) {
	l.Log(context.Background(), LevelTrace, msg, args...)
}
