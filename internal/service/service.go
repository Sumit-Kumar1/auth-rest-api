package service

import (
	"errors"

	"auth-rest-api/internal/models"

	"github.com/google/uuid"
	"gofr.dev/pkg/gofr"
	gofrHTTP "gofr.dev/pkg/gofr/http"
	"golang.org/x/crypto/bcrypt"
)

// Storer defines the interface for data storage operations.
// It provides methods for user and token management.
//
//go:generate mockgen -source=service.go -destination=mock_interface.go -package=service
type Storer interface {
	CreateUser(ctx *gofr.Context, u *models.UserData) error
	GetUserByEmail(ctx *gofr.Context, email string) (*models.UserData, error)

	IsTokenRevoked(ctx *gofr.Context, tokenID string) (bool, error)
	CreateToken(ctx *gofr.Context, email string, td *models.TokenData) error
	DeleteToken(ctx *gofr.Context, email, accTokenID, refTokenID string) error

	IncrementFailedLogin(ctx *gofr.Context, email string) (int, error)
	ResetFailedLogin(ctx *gofr.Context, email string) error
	IsAccountLocked(ctx *gofr.Context, email string) (bool, error)
}

// Service represents the core business logic layer.
// It handles user authentication and token management operations.
type Service struct {
	Store Storer
}

// New creates a new instance of the Service with the provided storage implementation.
// It initializes the service with the required dependencies.
func New(s Storer) *Service {
	return &Service{Store: s}
}

func (s *Service) SignUp(ctx *gofr.Context, user *models.UserReq) error {
	if user == nil {
		return models.ErrBadRequest(models.ErrInvalidInput)
	}

	if err := user.Validate(); err != nil {
		return models.ErrBadRequest(err)
	}

	exUser, err := s.Store.GetUserByEmail(ctx, user.Email)
	if err != nil && !errors.Is(err, models.ErrUserNotFound) {
		return err
	}

	if exUser != nil {
		return models.ErrUserAlreadyExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(user.Password), 10)
	if err != nil {
		return err
	}

	ud := models.UserData{
		ID:       uuid.NewString(),
		Email:    user.Email,
		Password: hash,
	}

	if err := s.Store.CreateUser(ctx, &ud); err != nil {
		return err
	}

	return nil
}

func (s *Service) SignIn(ctx *gofr.Context, user *models.UserReq) (*models.TokenResponse, error) {
	if user == nil {
		return nil, models.ErrBadRequest(models.ErrInvalidInput)
	}

	if valErr := user.Validate(); valErr != nil {
		return nil, models.ErrBadRequest(valErr)
	}

	// Check if account is locked due to too many failed login attempts
	isLocked, err := s.Store.IsAccountLocked(ctx, user.Email)
	if err != nil {
		return nil, err
	}

	if isLocked {
		return nil, models.ErrAccountLocked
	}

	exUser, err := s.Store.GetUserByEmail(ctx, user.Email)
	if err != nil {
		// Increment failed login counter on user not found
		s.Store.IncrementFailedLogin(ctx, user.Email)
		return nil, err
	}

	if err = bcrypt.CompareHashAndPassword(exUser.Password, []byte(user.Password)); err != nil {
		// Increment failed login counter on password mismatch
		s.Store.IncrementFailedLogin(ctx, user.Email)
		return nil, models.ErrPasswordMismatch
	}

	// Reset failed login counter on successful login
	s.Store.ResetFailedLogin(ctx, user.Email)

	tokenData, err := GenerateToken(exUser.ID, exUser.Email)
	if err != nil {
		return nil, err
	}

	if err := s.Store.CreateToken(ctx, exUser.Email, tokenData); err != nil {
		return nil, err
	}

	return &models.TokenResponse{
		AccessToken:  tokenData.AccessToken,
		RefreshToken: tokenData.RefreshToken,
	}, nil
}

func (s *Service) RefreshToken(ctx *gofr.Context, accessClaim *models.Claims, refreshToken string) (*models.TokenResponse, error) {
	if accessClaim == nil {
		return nil, gofrHTTP.ErrorMissingParam{Params: []string{"access token"}}
	}

	refreshClaim, err := s.tokenParsing(refreshToken, "refresh")
	if err != nil {
		return nil, err
	}

	// check if access token is revoked
	isRevoked, err := s.Store.IsTokenRevoked(ctx, accessClaim.ClaimUID)
	if err != nil {
		return nil, err
	}

	if isRevoked {
		return nil, models.ErrTokenRevoked
	}

	// Deleting old active tokens
	if delErr := s.Store.DeleteToken(ctx, accessClaim.Email, accessClaim.ClaimUID, refreshClaim.ClaimUID); delErr != nil {
		return nil, delErr
	}

	td, err := GenerateToken(accessClaim.Subject, accessClaim.Email)
	if err != nil {
		return nil, err
	}

	// store the newly generated tokens UIDs
	if err := s.Store.CreateToken(ctx, accessClaim.Email, td); err != nil {
		return nil, err
	}

	return &models.TokenResponse{
		AccessToken:  td.AccessToken,
		RefreshToken: td.RefreshToken,
	}, nil
}

// RevokeToken revokes the provided token, deletes stored token too
func (s *Service) RevokeToken(ctx *gofr.Context, accClaims *models.Claims) error {
	if accClaims == nil {
		return gofrHTTP.ErrorMissingParam{Params: []string{"access token"}}
	}

	// Check if already revoked for idempotency
	isRevoked, err := s.Store.IsTokenRevoked(ctx, accClaims.ClaimUID)
	if err != nil {
		return err
	}
	if isRevoked {
		return nil
	}

	if delErr := s.Store.DeleteToken(ctx, accClaims.Email, accClaims.ClaimUID, ""); delErr != nil {
		return delErr
	}

	return nil
}

func (s *Service) ValidateTokens(ctx *gofr.Context, accessClaim *models.Claims) (*uuid.UUID, error) {
	if accessClaim == nil {
		return nil, gofrHTTP.ErrorMissingParam{Params: []string{"access token"}}
	}

	// Check if token has been revoked
	isRevoked, err := s.Store.IsTokenRevoked(ctx, accessClaim.ClaimUID)
	if err != nil {
		return nil, err
	}

	if isRevoked {
		return nil, models.ErrTokenRevoked
	}

	sub := accessClaim.Subject
	if sub == "" {
		return nil, models.ErrInvalidTokenType
	}

	uid, err := uuid.Parse(sub)
	if err != nil {
		return nil, err
	}

	return &uid, nil
}

func (s *Service) tokenParsing(token, tokenType string) (*models.Claims, error) {
	claim, err := ParseToken(token, tokenType)
	if err != nil {
		return nil, err
	}

	return claim, nil
}
