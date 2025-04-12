package service

import (
	"context"
	"errors"
	"log/slog"

	"auth-rest-api/internal/models"
	"auth-rest-api/internal/server"

	"golang.org/x/crypto/bcrypt"
)

// Storer defines the interface for data storage operations.
// It provides methods for user and token management.
type Storer interface {
	// User operations
	CreateUser(ctx context.Context, u *models.UserData) error
	GetUserByEmail(ctx context.Context, email string) (*models.UserData, error)
	// Token operations
	IsTokenRevoked(ctx context.Context, tokenID string) (bool, error)
	CreateToken(ctx context.Context, email string, td *models.TokenData) error
	DeleteToken(ctx context.Context, tokenID ...string) error
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

func (s *Service) SignUp(ctx context.Context, user *models.UserReq) error {
	logger := ctx.Value(server.Logger).(*slog.Logger)

	if user == nil {
		logger.LogAttrs(ctx, slog.LevelError, "empty user struct provided")
		return models.ErrBadRequest(models.ErrInvalid("user input"))
	}

	if err := user.Validate(); err != nil {
		return models.ErrBadRequest(err)
	}

	exUser, err := s.Store.GetUserByEmail(ctx, user.Email)
	if err != nil && !errors.Is(models.ErrNotFound("user"), err) {
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
		Email:    user.Email,
		Password: hash,
	}

	if err := s.Store.CreateUser(ctx, &ud); err != nil {
		return err
	}

	return nil
}

func (s *Service) SignIn(ctx context.Context, user *models.UserReq) (access, refresh string, err error) {
	logger := ctx.Value(server.Logger).(*slog.Logger)

	if user == nil {
		logger.LogAttrs(ctx, slog.LevelError, "empty user struct provided")
		return "", "", models.ErrBadRequest(models.ErrInvalid("user"))
	}

	if valErr := user.Validate(); valErr != nil {
		return "", "", models.ErrBadRequest(valErr)
	}

	exUser, err := s.Store.GetUserByEmail(ctx, user.Email)
	if err != nil {
		return "", "", err
	}

	if err = bcrypt.CompareHashAndPassword(exUser.Password, []byte(user.Password)); err != nil {
		logger.LogAttrs(ctx, slog.LevelError, "wrong password")
		return "", "", models.ErrPsswdNotMatch
	}

	tokenData, err := GenerateToken(user.Email)
	if err != nil {
		return "", "", err
	}

	if err := s.Store.CreateToken(ctx, user.Email, tokenData); err != nil {
		return "", "", err
	}

	return tokenData.AccessToken, tokenData.RefreshToken, nil
}

func (s *Service) RefreshToken(ctx context.Context, accessToken, refreshToken string) (access, refresh string, err error) {
	logger := ctx.Value(server.Logger).(*slog.Logger)

	accClaims, err := ParseToken(accessToken, "access")
	if err != nil {
		logger.LogAttrs(ctx, slog.LevelError, "invalid access token", slog.String("token", accessToken), slog.String("error", err.Error()))
		return "", "", err
	}

	refClaims, err := ParseToken(refreshToken, "refresh")
	if err != nil {
		logger.LogAttrs(ctx, slog.LevelError, "invalid refresh token", slog.String("token", refreshToken), slog.String("error", err.Error()))
		return "", "", err
	}

	// check if token is revoked
	isRevoked, err := s.Store.IsTokenRevoked(ctx, accClaims.ClaimUID)
	if err != nil {
		return "", "", err
	}

	if isRevoked {
		return "", "", models.ErrTokenRevoked
	}

	// Deleting old active tokens
	if delErr := s.Store.DeleteToken(ctx, accClaims.ClaimUID, refClaims.ClaimUID); delErr != nil {
		return "", "", delErr
	}

	td, err := GenerateToken(accClaims.Email)
	if err != nil {
		return "", "", err
	}

	// store the newly generated tokens UIDs
	if err := s.Store.CreateToken(ctx, accClaims.Email, td); err != nil {
		return "", "", err
	}

	return td.AccessToken, td.RefreshToken, nil
}

// RevokeToken revokes the provided token, deletes stored token too
func (s *Service) RevokeToken(ctx context.Context, token string) error {
	logger := ctx.Value(server.Logger).(*slog.Logger)

	accClaims, err := ParseToken(token, "access")
	if err != nil {
		logger.LogAttrs(ctx, slog.LevelError, "invalid access token", slog.String("token", token), slog.String("error", err.Error()))
		return err
	}

	if delErr := s.Store.DeleteToken(ctx, accClaims.ClaimUID); delErr != nil {
		return delErr
	}

	return nil
}
