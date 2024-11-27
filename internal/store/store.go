package store

import (
	"context"
	"errors"
	"time"

	"auth-rest-api/internal/models"

	"github.com/redis/go-redis/v9"
)

const (
	userTable = "users"
)

type Store struct {
	DB *redis.Client
}

func New(db *redis.Client) *Store {
	return &Store{DB: db}
}

func (s *Store) CreateUser(ctx context.Context, user *models.UserData) error {
	if err := s.DB.HSet(ctx, userTable, user.Email, user.Password).Err(); err != nil {
		return err
	}

	return nil
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (*models.UserData, error) {
	passwd, err := s.DB.HGet(ctx, userTable, email).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, models.ErrNotFound("user")
		}

		return nil, err
	}

	if passwd == "" {
		return nil, models.ErrNotFound("user")
	}

	return &models.UserData{Email: email, Password: []byte(passwd)}, nil
}

func (s *Store) CreateToken(ctx context.Context, email string, td *models.TokenData) error {
	accExp := time.Unix(td.AccessExpiresAt, 0)
	refExp := time.Unix(td.RefreshExpiresAt, 0)

	if err := s.DB.Set(ctx, td.AccessID, email, time.Until(accExp)).Err(); err != nil {
		return err
	}

	if err := s.DB.Set(ctx, td.RefreshID, email, time.Until(refExp)).Err(); err != nil {
		return err
	}

	return nil
}

func (s *Store) DeleteToken(ctx context.Context, tokenID ...string) error {
	val, err := s.DB.Del(ctx, tokenID...).Result()
	if err != nil {
		return err
	}

	if val == 0 {
		return models.ErrNotFound("not able to delete the provided entry")
	}

	return nil
}

func (s *Store) IsTokenRevoked(ctx context.Context, tokenID string) (bool, error) {
	val, err := s.DB.Exists(ctx, tokenID).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return true, nil
		}

		return false, err
	}

	if val > 0 {
		return false, nil
	}

	return true, nil
}
