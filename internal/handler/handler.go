package handler

import (
	"auth-rest-api/internal/models"
	"auth-rest-api/internal/server"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
)

type Servicer interface {
	SignIn(ctx context.Context, user *models.UserReq) (string, error)
	SignUp(ctx context.Context, user *models.UserReq) (string, error)
	RefreshToken(ctx context.Context, user *models.UserReq) (string, error)
}

type Handler struct {
	Service Servicer
}

func New(s Servicer) *Handler {
	return &Handler{Service: s}
}

func (h *Handler) SignIn(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := ctx.Value(server.Logger).(*slog.Logger)

	var u models.UserReq

	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		logger.LogAttrs(ctx, slog.LevelError, "failed to bind body", slog.String("error", err.Error()))
		return
	}

	token, err := h.Service.SignIn(ctx, &u)
	if err != nil {
		http.Error(w, "Failed to sign in", http.StatusUnauthorized)
		return
	}

	resp := models.UserResp{Email: u.Email, Token: token}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "Failed to write response", http.StatusInternalServerError)
	}
}

func (h *Handler) SignUp(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := ctx.Value(server.Logger).(*slog.Logger)

	var u models.UserReq

	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		logger.LogAttrs(ctx, slog.LevelError, "failed to bind body", slog.String("error", err.Error()))
		return
	}

	token, err := h.Service.SignUp(ctx, &u)
	if err != nil {
		http.Error(w, "Failed to sign in", http.StatusUnauthorized)
		return
	}

	resp := models.UserResp{Email: u.Email, Token: token}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "Failed to write response", http.StatusInternalServerError)
	}
}

func (h *Handler) RefreshToken(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
