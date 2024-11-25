package service

import (
	"context"
	"errors"
	"log/slog"

	"auth-rest-api/internal/models"
	"auth-rest-api/internal/server"
)

type Storer interface {
}

type Service struct {
	Store Storer
}

func New(s Storer) *Service {
	return &Service{Store: s}
}

func (s *Service) SignIn(ctx context.Context, user *models.UserReq) (string, error) {
	logger := ctx.Value(server.Logger).(*slog.Logger)

	if user == nil {
		logger.LogAttrs(ctx, slog.LevelError, "empty user struct provided")
		return "", errors.New("empty user strut")
	}

	if err := user.Validate(); err != nil {
		return "", err
	}

	token, err := GenerateToken(user.Email)
	if err != nil {
		return "", err
	}

	return token, nil
}

func (s *Service) SignUp(ctx context.Context, user *models.UserReq) (string, error) {
	return "", nil
}

func (s *Service) RefreshToken(ctx context.Context, user *models.UserReq) (string, error) {
	return "", nil
}
