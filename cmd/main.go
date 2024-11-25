package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"

	"auth-rest-api/internal/server"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	app, err := server.ServerFromEnvs()
	if err != nil {
		app.Logger.LogAttrs(ctx, slog.LevelError, "failed to create server", slog.Any("error", err))
		return
	}

	srvErr := make(chan error, 1)
	go func() {
		app.Logger.LogAttrs(ctx, slog.LevelInfo, "application is running",
			slog.Group("server", slog.String("name", app.Name), slog.String("address", app.Addr),
				slog.Bool("DB Connected", true), slog.Group("timeouts (durations)", slog.Duration("read", app.ReadTimeout),
					slog.Duration("write", app.WriteTimeout), slog.Duration("idle", app.IdleTimeout))))

		app.Server.Handler = app.Mux
		srvErr <- app.ListenAndServe()
	}()

	select {
	case err = <-srvErr:
		app.Logger.Error(err.Error())
		return
	case <-ctx.Done():
		stop()
	}

	err = app.Shutdown(context.Background())
	if err != nil {
		app.Logger.LogAttrs(ctx, slog.LevelError, "error while shutting down", slog.String("error", err.Error()))
	}

	app.Logger.LogAttrs(ctx, slog.LevelInfo, "application is shut down", slog.String("name", app.Name))
}
