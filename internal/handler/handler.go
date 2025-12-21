package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"auth-rest-api/internal/models"
	"auth-rest-api/internal/server"

	"github.com/google/uuid"
)

// Servicer defines the interface for service layer operations.
// It provides methods for user authentication and token management.
//
//go:generate mockgen -source=handler.go -destination=mock_interface.go -package=handler
type Servicer interface {
	SignUp(ctx context.Context, user *models.UserReq) error
	SignIn(ctx context.Context, user *models.UserReq) (*models.TokenResponse, error)
	RefreshToken(ctx context.Context, accToken, refToken string) (*models.TokenResponse, error)
	RevokeToken(ctx context.Context, accToken string) error
	ValidateTokens(ctx context.Context, token string) (*uuid.UUID, error)
}

// Handler represents the HTTP request handler layer.
// It processes incoming HTTP requests and delegates business logic to the service layer.
type Handler struct {
	Service Servicer
}

// New creates a new instance of the Handler with the provided service implementation.
// It initializes the handler with the required dependencies.
func New(s Servicer) *Handler {
	return &Handler{Service: s}
}

func (h *Handler) Validate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := ctx.Value(server.Logger).(*slog.Logger)

	authHeader := r.Header.Get("Authorization")
	if strings.TrimSpace(authHeader) == "" {
		logger.LogAttrs(ctx, slog.LevelError, "Missing Authorization header")
		respondWithError(w, http.StatusUnauthorized, "Missing Authorization header")

		return
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")

	userID, err := h.Service.ValidateTokens(ctx, token)
	if err != nil {
		logger.LogAttrs(ctx, slog.LevelError, "error while validating tokens", slog.String("error", err.Error()))
		respondWithError(w, http.StatusBadRequest, fmt.Sprintf("error while validating tokens: %s", err.Error()))
		return
	}

	writeData(w, http.StatusOK, userID.String())
	logger.LogAttrs(ctx, slog.LevelInfo, "user validated", slog.String("userID", userID.String()))
}

// SignUp lets you store user email and password in database
func (h *Handler) SignUp(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := ctx.Value(server.Logger).(*slog.Logger)

	var u models.UserReq

	if r.Body == nil {
		respondWithError(w, http.StatusBadRequest, "missing request body")
		return
	}

	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		respondWithError(w, http.StatusBadRequest, fmt.Sprintf("failed to bind body - %s", err.Error()))
		return
	}

	defer func(body io.ReadCloser) { _ = body.Close() }(r.Body)

	if err := h.Service.SignUp(ctx, &u); err != nil {
		switch {
		case models.ErrUserAlreadyExists.Is(err):
			respondWithError(w, http.StatusConflict, fmt.Sprintf("failed to sign up - %s", err.Error()))
			logger.LogAttrs(ctx, slog.LevelError, err.Error())

			return

		case errors.Is(err, models.ErrBadRequest(err)):
			respondWithError(w, http.StatusBadRequest, err.Error())
			logger.LogAttrs(ctx, slog.LevelError, err.Error())

			return

		default:
			respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("failed to sign up - %s", err.Error()))
			logger.LogAttrs(ctx, slog.LevelError, err.Error())

			return
		}
	}

	writeData(w, http.StatusCreated, "user created successfully")
	logger.LogAttrs(ctx, slog.LevelInfo, "user signed up successfully", slog.String("email", u.Email))
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

	defer func(body io.ReadCloser) { _ = body.Close() }(r.Body)

	tokenResp, err := h.Service.SignIn(ctx, &u)
	if err != nil {
		switch {
		case errors.Is(err, models.ErrUserNotFound):
			respondWithError(w, http.StatusNotFound, err.Error())
			logger.LogAttrs(ctx, slog.LevelError, "user not found", slog.String("email", u.Email))

			return

		case errors.Is(err, models.ErrBadRequest(err)):
			respondWithError(w, http.StatusBadRequest, err.Error())
			logger.LogAttrs(ctx, slog.LevelError, err.Error())

			return

		default:
			respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("failed to sign up - %s", err.Error()))
			logger.LogAttrs(ctx, slog.LevelError, err.Error())

			return
		}
	}

	resp := models.UserResp{
		Email:        u.Email,
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
	}

	writeData(w, http.StatusCreated, resp)
	logger.LogAttrs(ctx, slog.LevelInfo, "user signed in", slog.String("email", u.Email))
}

func (h *Handler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	t := struct {
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

	defer func(body io.ReadCloser) { _ = body.Close() }(r.Body)

	tokenResp, err := h.Service.RefreshToken(ctx, token, t.Token)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, fmt.Sprintf("failed to refresh token - %s", err.Error()))
		logger.LogAttrs(ctx, slog.LevelError, err.Error())

		return
	}

	userResp := models.UserResp{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
	}

	writeData(w, http.StatusOK, userResp)
	logger.LogAttrs(ctx, slog.LevelInfo, "user refreshed token")
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
		logger.LogAttrs(ctx, slog.LevelError, "failed to revoke token", slog.String("error", err.Error()))
		respondWithError(w, http.StatusInternalServerError, "Failed to revoke token")

		return
	}

	writeData(w, http.StatusNoContent, "token revoked successfully")
	logger.LogAttrs(ctx, slog.LevelInfo, "revoked token")
}

func respondWithError(w http.ResponseWriter, code int, reason string) {
	errHTTP := models.NewHTTPError(code, reason, "")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(errHTTP.Code)

	if err := json.NewEncoder(w).Encode(errHTTP); err != nil {
		http.Error(w, "failed to write response", http.StatusInternalServerError)
	}
}

func writeData[T any](w http.ResponseWriter, code int, resp T) {
	w.WriteHeader(code)
	w.Header().Set("Content-Type", "application/json")

	data, err := json.Marshal(resp)
	if err != nil {
		respondWithError(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	_, _ = w.Write(json.RawMessage(`{"data":` + string(data) + `}`))
}
