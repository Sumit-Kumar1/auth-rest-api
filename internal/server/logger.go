package server

import (
	"log/slog"
	"os"
	"time"
)

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

	replaceFn := func(_ []string, a slog.Attr) slog.Attr {
		if a.Key == "time" {
			a.Value = slog.StringValue(time.Now().Format(time.RFC1123))
		}

		return a
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		AddSource:   false,
		Level:       leveler,
		ReplaceAttr: replaceFn,
	}))

	slog.SetLogLoggerLevel(leveler)
	slog.SetDefault(logger)

	return logger
}
