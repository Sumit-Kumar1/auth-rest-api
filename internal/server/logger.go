package server

import (
	"log/slog"
	"os"
	"time"
)

// newLogger creates a new structured logger with the specified configuration.
// It configures the log level based on the LOG_LEVEL environment variable.
// The default level is INFO if the environment variable is not set.
// Returns a configured slog.Logger instance.
func newLogger() *slog.Logger {
	var (
		leveler slog.Level
		level   = os.Getenv("LOG_LEVEL")
	)

	switch level {
	case "ERROR":
		leveler = slog.LevelError
	case "DEBUG":
		leveler = slog.LevelDebug
	case "WARN":
		leveler = slog.LevelWarn
	default:
		leveler = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: leveler,
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key != slog.TimeKey {
				return a
			}

			if t, ok := a.Value.Any().(time.Time); ok {
				a.Value = slog.StringValue(t.Format(time.RFC3339))
			}

			return a
		},
	}

	return slog.New(slog.NewJSONHandler(os.Stdout, opts))
}
