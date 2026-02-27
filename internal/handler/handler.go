package handler

import (
	"errors"
	"net/http"
	"strings"

	"auth-rest-api/internal/models"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

// Servicer defines the interface for service layer operations.
//
//go:generate mockgen -source=handler.go -destination=mock_interface.go -package=handler
type Servicer interface {
	SignUp(ctx *echo.Context, user *models.UserReq) error
	SignIn(ctx *echo.Context, user *models.UserReq) (*models.TokenResponse, error)
	RefreshToken(ctx *echo.Context, accToken, refToken string) (*models.TokenResponse, error)
	RevokeToken(ctx *echo.Context, accToken string) error
	ValidateTokens(ctx *echo.Context, token string) (*uuid.UUID, error)
}

// Handler processes HTTP requests and delegates to the service layer.
type Handler struct {
	Service Servicer
}

// New creates a new Handler with the provided service implementation.
func New(s Servicer) *Handler {
	return &Handler{Service: s}
}

// extractBearerToken extracts and validates a Bearer token from the Authorization header.
// Returns ErrUnauthorized if the header is missing, malformed, or not a Bearer token.
func extractBearerToken(c *echo.Context) (string, error) {
	header := c.Request().Header.Get("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		return "", models.ErrUnauthorized
	}

	token := strings.TrimPrefix(header, "Bearer ")
	if strings.TrimSpace(token) == "" {
		return "", models.ErrUnauthorized
	}

	return token, nil
}

// Validate checks a JWT token and returns the associated userID.
func (h *Handler) Validate(c *echo.Context) error {
	token, err := extractBearerToken(c)
	if err != nil {
		return respondWithError(c, http.StatusUnauthorized, "Unauthorized")
	}

	userID, err := h.Service.ValidateTokens(c, token)
	if err != nil {
		return respondWithError(c, http.StatusUnauthorized, "Unauthorized")
	}

	return writeData(c, http.StatusOK, map[string]string{"userID": userID.String()})
}

// SignUp registers a new user with email and password.
func (h *Handler) SignUp(c *echo.Context) error {
	var u models.UserReq

	if err := c.Bind(&u); err != nil {
		return respondWithError(c, http.StatusBadRequest, "invalid request body")
	}

	if err := h.Service.SignUp(c, &u); err != nil {
		switch {
		case errors.Is(err, models.ErrUserAlreadyExists):
			return respondWithError(c, http.StatusConflict, "user already exists")

		default:
			var httpErr *models.HTTPError
			if errors.As(err, &httpErr) {
				return respondWithError(c, httpErr.Code, httpErr.Message)
			}

			return respondWithError(c, http.StatusInternalServerError, "internal server error")
		}
	}

	return writeData(c, http.StatusCreated, map[string]string{"message": "user created successfully"})
}

// SignIn authenticates a user and returns access + refresh tokens.
func (h *Handler) SignIn(c *echo.Context) error {
	var u models.UserReq

	if err := c.Bind(&u); err != nil {
		return respondWithError(c, http.StatusBadRequest, "invalid request body")
	}

	tokenResp, err := h.Service.SignIn(c, &u)
	if err != nil {
		var httpErr *models.HTTPError
		if errors.As(err, &httpErr) {
			return respondWithError(c, httpErr.Code, httpErr.Message)
		}

		// All auth failures return generic 401 to prevent user enumeration
		return respondWithError(c, http.StatusUnauthorized, "Unauthorized")
	}

	return writeData(c, http.StatusOK, models.UserResp{
		Email:        u.Email,
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
	})
}

// RefreshToken issues a new token pair using a valid access + refresh token.
func (h *Handler) RefreshToken(c *echo.Context) error {
	token, err := extractBearerToken(c)
	if err != nil {
		return respondWithError(c, http.StatusUnauthorized, "Unauthorized")
	}

	var body struct {
		Token string `json:"refreshToken"`
	}

	if err := c.Bind(&body); err != nil {
		return respondWithError(c, http.StatusBadRequest, "invalid request body")
	}

	if strings.TrimSpace(body.Token) == "" {
		return respondWithError(c, http.StatusBadRequest, "refreshToken is required")
	}

	tokenResp, err := h.Service.RefreshToken(c, token, body.Token)
	if err != nil {
		if errors.Is(err, models.ErrTokenRevoked) || errors.Is(err, models.ErrUnauthorized) {
			return respondWithError(c, http.StatusUnauthorized, "Unauthorized")
		}

		return respondWithError(c, http.StatusInternalServerError, "internal server error")
	}

	return writeData(c, http.StatusOK, models.UserResp{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
	})
}

// RevokeToken invalidates the provided access token (logout).
func (h *Handler) RevokeToken(c *echo.Context) error {
	token, err := extractBearerToken(c)
	if err != nil {
		return respondWithError(c, http.StatusUnauthorized, "Unauthorized")
	}

	if err := h.Service.RevokeToken(c, token); err != nil {
		return respondWithError(c, http.StatusInternalServerError, "internal server error")
	}

	return c.NoContent(http.StatusNoContent)
}

func respondWithError(c *echo.Context, code int, message string) error {
	return c.JSON(code, models.NewHTTPError(code, message, ""))
}

func writeData[T any](c *echo.Context, code int, resp T) error {
	return c.JSON(code, map[string]T{"data": resp})
}
