package handler

import (
	"auth-rest-api/internal/models"
	"context"
	"encoding/json"
	"io"
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
	logger := ctx.Value("logger").(*slog.Logger)

	var u models.UserReq

	data, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		logger.LogAttrs(ctx, slog.LevelError, "Failed to read body", slog.Any("error", err.Error()))
		return
	}

	if err := json.Unmarshal(data, &u); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		logger.LogAttrs(ctx, slog.LevelError, "failed to unmarshal body", slog.Any("error", err.Error()))
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
	logger := ctx.Value("logger").(*slog.Logger)

	var u models.UserReq

	data, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		logger.LogAttrs(ctx, slog.LevelError, "Failed to read body", slog.Any("error", err.Error()))
		return
	}

	if err := json.Unmarshal(data, &u); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		logger.LogAttrs(ctx, slog.LevelError, "failed to unmarshal body", slog.Any("error", err.Error()))
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

func (h *Handler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
