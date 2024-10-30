package reliquary

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
)

var (
	// LevelTrace a level below slog.LevelDebug, will log every packet received
	LevelTrace slog.Level = -8

	logLevel slog.LevelVar
	logger   *traceLogger
)

func init() {
	logLevel = slog.LevelVar{}
	logLevel.Set(slog.LevelWarn)

	logger = &traceLogger{slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level:     &logLevel,
		AddSource: true,
	})).With("module", "go-reliquary")}
}

// SetLogLevel sets the logger to the default one with the given slog.Level
func SetLogLevel(level slog.Level) {
	logLevel.Set(level)
}

// Not the best solution, I'd want to write a proper implementation at some point
// in my own package. But good enough
type traceLogger struct {
	*slog.Logger
}

func (tl *traceLogger) Trace(msg string, args ...any) {
	tl.Log(context.Background(), LevelTrace, msg, args...)
}

func (tl *traceLogger) IsTraceEnabled() bool {
	return tl.Enabled(context.Background(), LevelTrace)
}

func (tl *traceLogger) WithArgs(args ...any) *traceLogger {
	if len(args) == 0 {
		return tl
	}
	c := tl.With(args...)
	return &traceLogger{c}
}

func bytesAsHex(bytes []byte) string {
	output := ""
	for _, b := range bytes {
		output += fmt.Sprintf("%02x", b)
	}
	return strings.TrimSpace(output)
}
