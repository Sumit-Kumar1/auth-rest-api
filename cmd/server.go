package cmd

import (
	"context"
	"encoding/json"
	"log"
	"log/slog"
	"net/http"

	"auth-rest-api/internal/handler"
	"auth-rest-api/internal/server"
	"auth-rest-api/internal/service"
	"auth-rest-api/internal/store"

	"github.com/joho/godotenv"
)

const (
	up   = "UP"
	down = "DOWN"
)

// Run initializes the server, sets up HTTP handlers, and starts the server.
// It handles graceful shutdown when the application receives an interrupt signal.
func Run(ctx context.Context) error {
	if err := godotenv.Load(".env"); err != nil {
		log.Printf(".env file not found: %v", err)
		log.Printf("continuing loading system / docker env variables")
	}

	app, err := server.NewServerBuilder().WithLogger().WithHostPort().WithTimeouts().Build()
	if err != nil {
		return err
	}

	setupHandlers(app)

	srvErr := make(chan error, 1)
	go startServer(ctx, app, srvErr)

	select {
	case err = <-srvErr:
		return err
	case <-ctx.Done():
	}

	return gracefulShutdown(ctx, app)
}

// setupHandlers initializes the application layers and registers all HTTP routes
func setupHandlers(app *server.Server) {
	st := store.New(app.DB.Client)
	svc := service.New(st)
	h := handler.New(svc)

	registerAuthRoutes(app, h)
	registerHealthRoute(app)
}

// registerAuthRoutes registers all authentication-related endpoints
func registerAuthRoutes(app *server.Server, h *handler.Handler) {
	authMiddleware := server.AuthMiddleware()
	correlationMiddleware := server.AddCorrelation()

	authEndpoints := []struct {
		methodPath string
		handler    http.HandlerFunc
		middleware []server.Middleware
	}{
		{"POST /signup", h.SignUp, []server.Middleware{correlationMiddleware}},
		{"POST /signin", h.SignIn, []server.Middleware{correlationMiddleware}},
		{"POST /refresh", h.RefreshToken, []server.Middleware{correlationMiddleware, authMiddleware}},
		{"POST /revoke", h.RevokeToken, []server.Middleware{correlationMiddleware, authMiddleware}},
	}

	for _, endpoint := range authEndpoints {
		app.Mux.HandleFunc(endpoint.methodPath, server.Chain(endpoint.handler, endpoint.middleware...))
	}
}

// registerHealthRoute registers the health check endpoint
func registerHealthRoute(app *server.Server) {
	app.Mux.HandleFunc("GET /health", healthHandler(app))
}

// healthHandler returns the HTTP handler for health checks
func healthHandler(app *server.Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		healthStatus := checkSystemHealth(ctx, app)
		logHealthStatus(ctx, app, healthStatus)
		sendHealthResponse(w, app, healthStatus)
	}
}

// checkSystemHealth performs all health checks and returns the overall status
func checkSystemHealth(ctx context.Context, app *server.Server) *server.Health {
	dbHealthy := checkDatabaseHealth(ctx, app)
	status := up
	dbStatus := up
	httpStatus := http.StatusOK

	if !dbHealthy {
		status = down
		dbStatus = down
		httpStatus = http.StatusServiceUnavailable
	}

	return &server.Health{
		Status:     status,
		DBStatus:   dbStatus,
		StatusCode: httpStatus,
	}
}

// checkDatabaseHealth verifies database connectivity
func checkDatabaseHealth(ctx context.Context, app *server.Server) bool {
	const pong = "PONG"

	if app.DB == nil {
		app.Logger.LogAttrs(ctx, slog.LevelWarn, "database health check failed",
			slog.String("error", "nil db object"))
		return false
	}

	statusCmd := app.DB.Client.Ping(ctx)
	res, err := statusCmd.Result()
	if err != nil {
		app.Logger.LogAttrs(ctx, slog.LevelWarn, "database health check failed",
			slog.String("error", err.Error()))
		return false
	}

	if res != pong {
		app.Logger.LogAttrs(ctx, slog.LevelInfo, "database health check failed")
		return false
	}

	return true
}

// logHealthStatus logs the health check results with appropriate log level
func logHealthStatus(ctx context.Context, app *server.Server, health *server.Health) {
	if health.Status == up {
		app.Logger.LogAttrs(ctx, slog.LevelInfo, "health check passed",
			slog.String("overall_status", health.Status),
			slog.String("database_status", health.DBStatus))
	} else {
		app.Logger.LogAttrs(ctx, slog.LevelError, "health check failed",
			slog.String("overall_status", health.Status),
			slog.String("database_status", health.DBStatus))
	}
}

// sendHealthResponse writes the health status as a JSON response
func sendHealthResponse(w http.ResponseWriter, app *server.Server, health *server.Health) {
	data, err := json.Marshal(health)
	if err != nil {
		app.Logger.LogAttrs(context.Background(), slog.LevelError,
			"failed to marshal health status", slog.String("error", err.Error()))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(health.StatusCode)
	if _, err := w.Write(data); err != nil {
		app.Logger.LogAttrs(context.Background(), slog.LevelError,
			"failed to write health response", slog.String("error", err.Error()))
	}
}

// startServer begins listening for HTTP requests
func startServer(ctx context.Context, app *server.Server, srvErr chan<- error) {
	app.Logger.LogAttrs(ctx, slog.LevelInfo, "application is running",
		slog.Group("auth-rest-api server",
			slog.String("address", app.Addr),
			slog.Bool("DB Connected", true),
			slog.Group("timeouts (durations)",
				slog.Duration("read", app.ReadTimeout),
				slog.Duration("write", app.WriteTimeout),
				slog.Duration("idle", app.IdleTimeout))))

	app.Handler = app.Mux
	srvErr <- app.ListenAndServe()
}

// gracefulShutdown performs a graceful shutdown of the server
func gracefulShutdown(ctx context.Context, app *server.Server) error {
	var err error
	if err = app.Shutdown(context.Background()); err != nil {
		app.Logger.LogAttrs(ctx, slog.LevelError, "error while shutting down",
			slog.String("error", err.Error()))
		return err
	}

	defer app.Logger.Info("application is shut down")

	//close db
	if app.DB == nil {
		app.Logger.Warn("closing app with nil DB")
		return nil
	}

	if app.DB.Client == nil {
		app.Logger.Warn("closing app with nil DB client")
		return nil
	}

	if err = app.DB.Client.Close(); err != nil {
		app.Logger.Error("closing DB client", slog.String("error", err.Error()))
	}

	return nil
}
