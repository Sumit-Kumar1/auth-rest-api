package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"auth-rest-api/internal/models"
	"auth-rest-api/internal/server"
)

type Servicer interface {
	SignUp(ctx context.Context, user *models.UserReq) error
	SignIn(ctx context.Context, user *models.UserReq) (string, string, error)
	RefreshToken(ctx context.Context, accToken, refToken string) (string, string, error)
	RevokeToken(ctx context.Context, accToken string) error
}

type Handler struct {
	Service Servicer
}

func New(s Servicer) *Handler {
	return &Handler{Service: s}
}

// SignUp lets you store user email and password in database
func (h *Handler) SignUp(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := ctx.Value(server.Logger).(*slog.Logger)

	var u models.UserReq

	if r.Body == nil {
		respondWithError(w, http.StatusBadRequest, "Request body missing")
		return
	}

	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		respondWithError(w, http.StatusBadRequest, fmt.Sprintf("failed to bind body - %s", err.Error()))
		return
	}

	if err := h.Service.SignUp(ctx, &u); err != nil {
		switch {
		case models.ErrUserAlreadyExists.Is(err):
			respondWithError(w, http.StatusConflict, fmt.Sprintf("failed to sign up - %s", err.Error()))
			logger.LogAttrs(ctx, slog.LevelError, err.Error())
			return

		default:
			respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("failed to sign up - %s", err.Error()))
			logger.LogAttrs(ctx, slog.LevelError, err.Error())
			return
		}
	}

	w.WriteHeader(http.StatusCreated)

	if _, err := w.Write([]byte("User created successfully")); err != nil {
		logger.LogAttrs(ctx, slog.LevelError, "failed to write response", slog.String("error", err.Error()))
	}
}

// SignIn lets you authenticate user with user details and JWT token
func (h *Handler) SignIn(w http.ResponseWriter, r *http.Request) {
	var u models.UserReq

	ctx := r.Context()
	logger := ctx.Value(server.Logger).(*slog.Logger)

	if r.Body == nil {
		respondWithError(w, http.StatusBadRequest, "Request body missing")
		return
	}

	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		respondWithError(w, http.StatusBadRequest, fmt.Sprintf("failed to bind body - %s", err.Error()))
		logger.LogAttrs(ctx, slog.LevelError, "failed to bind body", slog.String("error", err.Error()))
		return
	}

	token, refToken, err := h.Service.SignIn(ctx, &u)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, fmt.Sprintf("failed to signin - %s", err.Error()))
		return
	}

	logger.LogAttrs(ctx, slog.LevelInfo, "user signed in", slog.String("email", u.Email))

	resp := models.UserResp{Email: u.Email, AccessToken: token, RefreshToken: refToken}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "Failed to write response", http.StatusInternalServerError)
	}
}

func (h *Handler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var t = struct {
		Token string `json:"refreshToken"`
	}{}

	ctx := r.Context()
	logger := ctx.Value(server.Logger).(*slog.Logger)

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		logger.LogAttrs(ctx, slog.LevelError, "Missing Authorization header")
		respondWithError(w, http.StatusUnauthorized, "Missing Authorization header")
		return
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")

	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		logger.LogAttrs(ctx, slog.LevelError, "failed to bind body", slog.String("error", err.Error()))
		respondWithError(w, http.StatusBadRequest, fmt.Sprintf("failed to bind body - %s", err.Error()))
		return
	}

	newAccessToken, newRefreshToken, err := h.Service.RefreshToken(ctx, token, t.Token)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, fmt.Sprintf("failed to refresh token - %s", err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(models.UserResp{AccessToken: newAccessToken, RefreshToken: newRefreshToken}); err != nil {
		logger.LogAttrs(ctx, slog.LevelError, "failed to write response", slog.String("error", err.Error()))
		return
	}

	logger.LogAttrs(ctx, slog.LevelInfo, "user refreshed token", slog.String("token", newRefreshToken))
}

func (h *Handler) RevokeToken(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := ctx.Value(server.Logger).(*slog.Logger)

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		logger.LogAttrs(ctx, slog.LevelError, "Missing Authorization header")
		respondWithError(w, http.StatusUnauthorized, "Missing Authorization header")
		return
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")

	if err := h.Service.RevokeToken(ctx, token); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to revoke token")
		return
	}

	w.WriteHeader(http.StatusNoContent)

	logger.LogAttrs(ctx, slog.LevelInfo, "revoked token", slog.String("token", token))
}

func respondWithError(w http.ResponseWriter, code int, reason string) {
	cErr := models.CustomError{Message: reason, Code: code}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(cErr.Code)

	if err := json.NewEncoder(w).Encode(cErr); err != nil {
		http.Error(w, "Failed to write response", http.StatusInternalServerError)
	}
}
