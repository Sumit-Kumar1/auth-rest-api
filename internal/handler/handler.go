package handler

import (
	"errors"
	"strings"

	"auth-rest-api/internal/models"

	"github.com/google/uuid"
	"gofr.dev/pkg/gofr"
	gofrHTTP "gofr.dev/pkg/gofr/http"
)

// Servicer defines the interface for service layer operations.
//
//go:generate mockgen -source=handler.go -destination=mock_interface.go -package=handler
type Servicer interface {
	SignUp(ctx *gofr.Context, user *models.UserReq) error
	SignIn(ctx *gofr.Context, user *models.UserReq) (*models.TokenResponse, error)
	RefreshToken(ctx *gofr.Context, accessClaim *models.Claims, refToken string) (*models.TokenResponse, error)
	RevokeToken(ctx *gofr.Context, accessClaim *models.Claims) error
	ValidateTokens(ctx *gofr.Context, accessClaim *models.Claims) (*uuid.UUID, error)
}

// Handler processes HTTP requests and delegates to the service layer.
type Handler struct {
	Service Servicer
}

// New creates a new Handler with the provided service implementation.
func New(s Servicer) *Handler {
	return &Handler{Service: s}
}

// SignUp registers a new user with email and password.
func (h *Handler) SignUp(c *gofr.Context) (any, error) {
	var u models.UserReq

	if err := c.Bind(&u); err != nil {
		return nil, err
	}

	if err := h.Service.SignUp(c, &u); err != nil {
		switch {
		case errors.Is(err, models.ErrUserAlreadyExists):
			return nil, err

		default:
			var httpErr *models.HTTPError
			if errors.As(err, &httpErr) {
				return nil, err
			}

			return nil, err
		}
	}

	return "user created successfully", nil
}

// SignIn authenticates a user and returns access + refresh tokens.
func (h *Handler) SignIn(c *gofr.Context) (any, error) {
	var u models.UserReq

	if err := c.Bind(&u); err != nil {
		return nil, err
	}

	tokenResp, err := h.Service.SignIn(c, &u)
	if err != nil {
		var httpErr *models.HTTPError
		if errors.As(err, &httpErr) {
			return nil, err
		}

		// All auth failures return generic 401 to prevent user enumeration
		return nil, err
	}

	return models.UserResp{
		Email:        u.Email,
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
	}, nil
}

// RefreshToken issues a new token pair using a valid access + refresh token.
func (h *Handler) RefreshToken(c *gofr.Context) (any, error) {
	accessClaim, err := extractClaimFromCtx(c)
	if err != nil {
		return nil, err
	}

	var body struct {
		Token string `json:"refreshToken"`
	}

	if err := c.Bind(&body); err != nil {
		return nil, err
	}

	if strings.TrimSpace(body.Token) == "" {
		return nil, err
	}

	tokenResp, err := h.Service.RefreshToken(c, accessClaim, body.Token)
	if err != nil {
		if errors.Is(err, models.ErrTokenRevoked) || errors.Is(err, models.ErrUnauthorized) {
			return nil, err
		}

		return nil, err
	}

	return models.UserResp{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
	}, nil
}

// RevokeToken invalidates the provided access token (logout).
func (h *Handler) RevokeToken(c *gofr.Context) (any, error) {
	claim, err := extractClaimFromCtx(c)
	if err != nil {
		return nil, err
	}

	if err := h.Service.RevokeToken(c, claim); err != nil {
		return nil, err
	}

	return "token revoked successfully", nil
}

// Validate checks a JWT token and returns the associated userID.
func (h *Handler) Validate(c *gofr.Context) (any, error) {
	claims, err := extractClaimFromCtx(c)
	if err != nil {
		return nil, err
	}

	userID, err := h.Service.ValidateTokens(c, claims)
	if err != nil {
		return nil, err
	}

	return userID.String(), nil
}

func extractClaimFromCtx(c *gofr.Context) (*models.Claims, error) {
	claims, ok := c.Value(models.CtxClaimKey).(*models.Claims)
	if !ok {
		return nil, gofrHTTP.ErrorEntityNotFound{Name: "context claim", Value: "nil"}
	}

	return claims, nil
}
