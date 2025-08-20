package main

import (
	"auth-rest-api/cmd"
	"context"
	"log/slog"
	"os"
	"os/signal"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := cmd.Run(ctx); err != nil {
		slog.LogAttrs(ctx, slog.LevelError, "error while running server", slog.String("error", err.Error()))
		return
	}
}
