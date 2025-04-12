// Package store provides
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

// Store represents the data storage layer.
// It implements the Storer interface and provides Redis-based persistence.
type Store struct {
	DB *redis.Client
}

// New creates a new Store instance with the provided Redis client.
// It initializes the store with the required database connection.
func New(db *redis.Client) *Store {
	return &Store{DB: db}
}

// CreateUser stores a new user in the database.
// It uses Redis HSET to store the user data with email as the key.
// Returns an error if the user already exists or if the operation fails.
func (s *Store) CreateUser(ctx context.Context, user *models.UserData) error {
	val, err := s.DB.HSet(ctx, userTable, user.Email, user.Password).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return models.ErrUserAlreadyExists
		}

		return err
	}

	if val == 0 {
		return models.ErrUserAlreadyExists
	}

	return nil
}

// GetUserByEmail retrieves a user from the database by their email.
// It uses Redis HGET to fetch the user data.
// Returns nil and an error if the user is not found or if the operation fails.
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

// CreateToken stores a new token in the database.
// It uses Redis SET to store the token data with appropriate expiration.
// Returns an error if the operation fails.
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

// DeleteToken removes one or more tokens from the database.
// It uses Redis DEL to remove the specified tokens.
// Returns an error if the operation fails.
func (s *Store) DeleteToken(ctx context.Context, tokenID ...string) error {
	val, err := s.DB.Del(ctx, tokenID...).Result()
	if err != nil {
		return err
	}

	if val != int64(len(tokenID)) {
		return models.NewConstError("delete error")
	}

	return nil
}

// IsTokenRevoked checks if a token has been revoked.
// It uses Redis EXISTS to check if the token ID exists in the database.
// Returns true if the token is revoked, false otherwise.
// Returns an error if the operation fails.
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
