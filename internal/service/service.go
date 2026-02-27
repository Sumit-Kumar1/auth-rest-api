package service

import (
	"errors"
	"strings"

	"auth-rest-api/internal/models"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	"golang.org/x/crypto/bcrypt"
)

// Storer defines the interface for data storage operations.
// It provides methods for user and token management.
//
//go:generate mockgen -source=service.go -destination=mock_interface.go -package=service
type Storer interface {
	CreateUser(ctx *echo.Context, u *models.UserData) error
	GetUserByEmail(ctx *echo.Context, email string) (*models.UserData, error)

	IsTokenRevoked(ctx *echo.Context, tokenID string) (bool, error)
	CreateToken(ctx *echo.Context, email string, td *models.TokenData) error
	DeleteToken(ctx *echo.Context, email, accTokenID, refTokenID string) error

	IncrementFailedLogin(ctx *echo.Context, email string) (int, error)
	ResetFailedLogin(ctx *echo.Context, email string) error
	IsAccountLocked(ctx *echo.Context, email string) (bool, error)
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

func (s *Service) SignUp(ctx *echo.Context, user *models.UserReq) error {
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

func (s *Service) SignIn(ctx *echo.Context, user *models.UserReq) (*models.TokenResponse, error) {
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

func (s *Service) RefreshToken(ctx *echo.Context, accessToken, refreshToken string) (*models.TokenResponse, error) {
	accessClaim, refreshClaim, err := s.tokenParsing(accessToken, refreshToken)
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
func (s *Service) RevokeToken(ctx *echo.Context, token string) error {
	accClaims, err := ParseToken(token, "access")
	if err != nil {
		return err
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

func (s *Service) ValidateTokens(ctx *echo.Context, token string) (*uuid.UUID, error) {
	if strings.TrimSpace(token) == "" {
		return nil, errors.New("nil token found")
	}

	accClaim, err := ParseToken(token, "access")
	if err != nil {
		return nil, err
	}

	// Check if token has been revoked
	isRevoked, err := s.Store.IsTokenRevoked(ctx, accClaim.ClaimUID)
	if err != nil {
		return nil, err
	}
	if isRevoked {
		return nil, models.ErrTokenRevoked
	}

	sub := accClaim.Subject
	if sub == "" {
		return nil, errors.New("empty user id")
	}

	uid, err := uuid.Parse(sub)
	if err != nil {
		return nil, err
	}

	return &uid, nil
}

func (s *Service) tokenParsing(accToken, refToken string) (access, refresh *Claims, err error) {
	access, err = ParseToken(accToken, "access")
	if err != nil {
		return nil, nil, err
	}

	refresh, err = ParseToken(refToken, "refresh")
	if err != nil {
		return nil, nil, err
	}

	return
}
