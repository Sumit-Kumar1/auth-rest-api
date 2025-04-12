package server

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// ContextKey is a type for context keys used in the application.
// It ensures type safety when using context values.
type ContextKey string

const (
	CorrelationID ContextKey = "correlationId"
	Logger        ContextKey = "logger"
)

// Middleware is a function type that wraps an HTTP handler.
// It can be used to add functionality before or after the handler execution.
type Middleware func(http.HandlerFunc) http.HandlerFunc

// Chain applies multiple middleware functions to an HTTP handler.
// It composes the middleware functions in the order they are provided.
// Returns a new handler that includes all the middleware functionality.
func Chain(f http.HandlerFunc, middlewares ...Middleware) http.HandlerFunc {
	for _, m := range middlewares {
		f = m(f)
	}

	return f
}

// AddCorrelation creates a middleware that adds a correlation ID to each request.
// It generates a unique ID for each request and adds it to the context.
// It also creates a structured logger with request information.
func AddCorrelation() Middleware {
	return func(f http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			corrID := uuid.NewString()

			logger := slog.With(slog.Group("request",
				slog.String(string(CorrelationID), corrID),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("host", r.Host),
				slog.String("remote-addr", r.RemoteAddr),
			))

			ctx := context.WithValue(r.Context(), Logger, logger)

			f(w, r.WithContext(context.WithValue(ctx, CorrelationID, corrID)))
		}
	}
}

// AuthMiddleware creates a middleware that validates JWT tokens.
// It checks for the presence of an Authorization header and validates the token.
// Returns an unauthorized error if the token is missing or invalid.
func AuthMiddleware() Middleware {
	return func(f http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			tokenString := strings.TrimPrefix(authHeader, "Bearer ")

			token, err := jwt.Parse(tokenString, func(_ *jwt.Token) (any, error) {
				return getJWTSecret(), nil
			})
			if err != nil || !token.Valid {
				slog.Log(context.Background(), slog.LevelError, "invalid token", slog.String("token", tokenString))
				http.Error(w, "Invalid token", http.StatusUnauthorized)

				return
			}

			f(w, r)
		}
	}
}

// getJWTSecret retrieves the JWT signing secret from environment variables.
// It returns the secret key for signing JWT tokens.
// If the environment variable is not set, it returns a default value.
func getJWTSecret() []byte {
	secret := os.Getenv("ACCESS_SECRET")
	if secret == "" {
		return []byte("my_secret_key")
	}

	return []byte(secret)
}
