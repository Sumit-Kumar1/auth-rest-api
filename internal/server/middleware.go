package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type ContextKey string

const (
	CorrelationID ContextKey = "correlationId"
	Logger        ContextKey = "logger"
)

type Middleware func(http.HandlerFunc) http.HandlerFunc

func Chain(f http.HandlerFunc, middlewares ...Middleware) http.HandlerFunc {
	for _, m := range middlewares {
		f = m(f)
	}

	return f
}

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

func getJWTSecret() []byte {
	secret := os.Getenv("ACCESS_SECRET")
	if secret == "" {
		return json.RawMessage("my_secret_key")
	}

	return json.RawMessage(secret)
}
