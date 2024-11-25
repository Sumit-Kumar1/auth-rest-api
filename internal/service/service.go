package service

import (
	"auth-rest-api/internal/models"
	"context"
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
	return "", nil
}

func (s *Service) SignUp(ctx context.Context, user *models.UserReq) (string, error) {
	return "", nil
}

func (s *Service) RefreshToken(ctx context.Context, user *models.UserReq) (string, error) {
	return "", nil
}
