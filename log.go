package reliquary

import (
	"fmt"
	"github.com/rs/zerolog"
	"os"
	"strings"
)

var (
	logger zerolog.Logger
)

func init() {
	output := zerolog.ConsoleWriter{Out: os.Stdout}
	output.FormatLevel = func(i interface{}) string {
		return strings.ToUpper(fmt.Sprintf("%-6s", i))
	}

	logger = zerolog.New(output).
		Level(zerolog.WarnLevel).
		With().
		Timestamp().
		Str("module", "go-reliquary").
		Logger()
}

// SetLogLevel sets the logger to the default one with the given slog.Level
func SetLogLevel(level zerolog.Level) {
	logger = logger.Level(level)
}

func isTraceEnabled(loggers ...zerolog.Logger) bool {
	l := func() zerolog.Logger {
		if len(loggers) > 0 {
			return loggers[0]
		}
		return logger
	}()
	return l.GetLevel() <= zerolog.TraceLevel
}

func bytesAsHex(bytes []byte) string {
	output := ""
	for _, b := range bytes {
		output += fmt.Sprintf("%02x", b)
	}
	return strings.TrimSpace(output)
}
