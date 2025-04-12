package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"os/signal"

	"auth-rest-api/internal/handler"
	"auth-rest-api/internal/server"
	"auth-rest-api/internal/service"
	"auth-rest-api/internal/store"
)

// main is the entry point of the application.
// It initializes the server, sets up HTTP handlers, and starts the server.
// It also handles graceful shutdown when the application receives an interrupt signal.
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	app, err := server.ServerFromEnvs()
	if err != nil {
		slog.LogAttrs(ctx, slog.LevelError, "failed to create server", slog.Any("error", err))
		return
	}

	newHTTPHandler(app)

	srvErr := make(chan error, 1)
	go func() {
		app.Logger.LogAttrs(ctx, slog.LevelInfo, "application is running",
			slog.Group("server", slog.String("name", app.Name), slog.String("address", app.Addr),
				slog.Bool("DB Connected", true), slog.Group("timeouts (durations)", slog.Duration("read", app.ReadTimeout),
					slog.Duration("write", app.WriteTimeout), slog.Duration("idle", app.IdleTimeout))))

		app.Handler = app.Mux
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

// newHTTPHandler sets up the HTTP handlers for the application.
// It initializes the store, service, and handler layers, and registers the routes.
// The function configures the server's HTTP router with all necessary endpoints.
func newHTTPHandler(app *server.Server) {
	st := store.New(app.DB.Client)
	svc := service.New(st)
	h := handler.New(svc)

	app.Mux.HandleFunc("POST /signup", server.Chain(h.SignUp, server.AddCorrelation()))
	app.Mux.HandleFunc("POST /signin", server.Chain(h.SignIn, server.AddCorrelation()))
	app.Mux.HandleFunc("POST /refresh", server.Chain(h.RefreshToken, server.AddCorrelation(), server.AuthMiddleware()))
	app.Mux.HandleFunc("POST /revoke", server.Chain(h.RevokeToken, server.AddCorrelation(), server.AuthMiddleware()))

	app.Mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if err := app.DB.Client.Ping(context.Background()); err != nil {
			app.Health = &server.Health{
				Status:   "Down",
				DBStatus: "Down",
			}

			data, mErr := json.Marshal(app.Health)
			if mErr != nil {
				http.Error(w, "not able to marshal the health status", http.StatusInternalServerError)
				return
			}

			app.Logger.LogAttrs(ctx, slog.LevelDebug, "health status", slog.Any("status", app.Health))

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(data)

			return
		}

		app.Health = &server.Health{
			Status:   "Up",
			DBStatus: "Up",
		}

		app.Logger.LogAttrs(ctx, slog.LevelDebug, "health status", slog.Any("status", app.Health))

		data, mErr := json.Marshal(app.Health)
		if mErr != nil {
			http.Error(w, "not able to marshal the health status", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(data)
	})
}
