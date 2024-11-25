package server

import (
	"context"
	"log/slog"
	"net/http"

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
			))

			ctx := context.WithValue(r.Context(), Logger, logger)

			f(w, r.WithContext(context.WithValue(ctx, CorrelationID, corrID)))
		}
	}
}
