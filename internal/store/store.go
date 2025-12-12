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
	userd         = "user:"
	emaild        = "email:"
	refTokend     = "refresh_tokens:"
	accTokend     = "access_tokens:"
	userRefTokend = "user_refresh_tokens:"
	userAccTokend = "user_access_tokens:"
)

// Store represents the data storage layer.
// It implements the Storer interface and provides Redis-based persistence.
type Store struct {
	DB *redis.Client
}

// New creates a new Store instance with the provided Redis client.
func New(db *redis.Client) *Store {
	return &Store{DB: db}
}

// CreateUser stores a new user in the database.
func (s *Store) CreateUser(ctx context.Context, user *models.UserData) error {
	// Store email → user_id mapping, this helps in getEmail calls
	if err := s.DB.Set(ctx, emaild+user.Email, user.ID, 0).Err(); err != nil {
		if errors.Is(err, redis.Nil) {
			return models.ErrUserAlreadyExists
		}

		return err
	}

	// store user hash data with uniqueness of userId
	if err := s.DB.HSet(ctx, userd+user.ID, map[string]any{
		"id": user.ID, "email": user.Email, "password": user.Password}).Err(); err != nil {
		if errors.Is(err, redis.Nil) {
			return models.ErrUserAlreadyExists
		}

		return err
	}

	return nil
}

// GetUserByEmail retrieves a user from the database by their email.
func (s *Store) GetUserByEmail(ctx context.Context, userEmail string) (*models.UserData, error) {
	userID, err := s.DB.Get(ctx, emaild+userEmail).Result()
	if errors.Is(err, redis.Nil) {
		return nil, models.ErrUserNotFound
	} else if err != nil {
		return nil, err
	}

	data, err := s.DB.HGetAll(ctx, userd+userID).Result()
	if err != nil {
		return nil, err
	}

	if len(data) == 0 {
		return nil, models.ErrUserNotFound
	}

	return &models.UserData{ID: data["id"], Email: data["email"], Password: []byte(data["password"])}, nil
}

// CreateToken stores a new token in the database.
func (s *Store) CreateToken(ctx context.Context, email string, td *models.TokenData) error {
	accExp := time.Until(time.Unix(td.AccessExpiresAt, 0))
	refExp := time.Until(time.Unix(td.RefreshExpiresAt, 0))

	tx := s.DB.TxPipeline()

	tx.Set(ctx, accTokend+td.AccessID, email, accExp)
	tx.SAdd(ctx, userAccTokend+email, td.AccessID)

	tx.Set(ctx, refTokend+td.RefreshID, email, refExp)
	tx.SAdd(ctx, userRefTokend+email, td.RefreshID)

	_, err := tx.Exec(ctx)

	return err
}

// DeleteToken removes one or more tokens from the database.
func (s *Store) DeleteToken(ctx context.Context, email, accTokenID, refTokenID string) error {
	tx := s.DB.TxPipeline()

	tx.Del(ctx, accTokend+accTokenID)
	tx.SRem(ctx, userAccTokend+email, accTokenID)

	if refTokenID != "" {
		tx.Del(ctx, refTokend+refTokenID)
		tx.SRem(ctx, userRefTokend+email, refTokenID)
	}

	_, err := tx.Exec(ctx)

	return err
}

// IsTokenRevoked checks if a token has been revoked (token doesn't exist).
func (s *Store) IsTokenRevoked(ctx context.Context, tokenID string) (bool, error) {
	val, err := s.DB.Exists(ctx, accTokend+tokenID).Result()
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
