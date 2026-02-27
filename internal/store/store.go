// Package store provides
package store

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"auth-rest-api/internal/models"

	"github.com/labstack/echo/v5"
	"github.com/redis/go-redis/v9"
)

const (
	userd               = "user:"
	emaild              = "email:"
	refTokend           = "refresh_tokens:"
	accTokend           = "access_tokens:"
	userRefTokend       = "user_refresh_tokens:"
	userAccTokend       = "user_access_tokens:"
	failedLoginAttempts = "failed_login_attempts:"
	accountLocked       = "account_locked:"
	failedLoginTTL      = 15 * time.Minute
	maxFailedAttempts   = 5
)

// Store represents the data storage layer.
// It implements the Storer interface and provides Redis-based persistence.
type Store struct {
	Redis *redis.Client
}

// New creates a new Store instance with the provided Redis client.
func New() (*Store, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	host := getEnvOrDefault("DB_HOST", "localhost")
	port := getEnvOrDefault("DB_PORT", "6379")
	dbIdx, err := strconv.Atoi(getEnvOrDefault("DB_NAME", "0"))
	if err != nil {
		return nil, fmt.Errorf("invalid DB_NAME: %w", err)
	}

	addr := net.JoinHostPort(host, port)
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: os.Getenv("DB_PASSWORD"),
		DB:       dbIdx,
	})

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, errors.Join(models.ErrDBNotConnected, err)
	}

	return &Store{Redis: rdb}, nil
}

// CreateUser stores a new user in the database.
func (s *Store) CreateUser(c *echo.Context, user *models.UserData) error {
	ctx := c.Request().Context()
	// Store email → user_id mapping, this helps in getEmail calls
	if err := s.Redis.Set(ctx, emaild+user.Email, user.ID, 0).Err(); err != nil {
		if errors.Is(err, redis.Nil) {
			return models.ErrUserAlreadyExists
		}

		return err
	}

	// store user hash data with uniqueness of userId
	if err := s.Redis.HSet(ctx, userd+user.ID, map[string]any{
		"id": user.ID, "email": user.Email, "password": user.Password}).Err(); err != nil {
		if errors.Is(err, redis.Nil) {
			return models.ErrUserAlreadyExists
		}

		return err
	}

	return nil
}

// GetUserByEmail retrieves a user from the database by their email.
func (s *Store) GetUserByEmail(c *echo.Context, userEmail string) (*models.UserData, error) {
	ctx := c.Request().Context()

	userID, err := s.Redis.Get(ctx, emaild+userEmail).Result()
	if errors.Is(err, redis.Nil) {
		return nil, models.ErrUserNotFound
	} else if err != nil {
		return nil, err
	}

	data, err := s.Redis.HGetAll(ctx, userd+userID).Result()
	if err != nil {
		return nil, err
	}

	if len(data) == 0 {
		return nil, models.ErrUserNotFound
	}

	return &models.UserData{ID: data["id"], Email: data["email"], Password: []byte(data["password"])}, nil
}

// CreateToken stores a new token in the database.
func (s *Store) CreateToken(c *echo.Context, email string, td *models.TokenData) error {
	ctx := c.Request().Context()
	accExp := time.Until(time.Unix(td.AccessExpiresAt, 0))
	refExp := time.Until(time.Unix(td.RefreshExpiresAt, 0))

	tx := s.Redis.TxPipeline()

	tx.Set(ctx, accTokend+td.AccessID, email, accExp)
	tx.SAdd(ctx, userAccTokend+email, td.AccessID)

	tx.Set(ctx, refTokend+td.RefreshID, email, refExp)
	tx.SAdd(ctx, userRefTokend+email, td.RefreshID)

	_, err := tx.Exec(ctx)

	return err
}

// DeleteToken removes one or more tokens from the database.
func (s *Store) DeleteToken(c *echo.Context, email, accTokenID, refTokenID string) error {
	ctx := c.Request().Context()

	tx := s.Redis.TxPipeline()

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
func (s *Store) IsTokenRevoked(c *echo.Context, tokenID string) (bool, error) {
	ctx := c.Request().Context()
	val, err := s.Redis.Exists(ctx, accTokend+tokenID).Result()
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

// IncrementFailedLogin increments the failed login counter for an email
func (s *Store) IncrementFailedLogin(c *echo.Context, email string) (int, error) {
	ctx := c.Request().Context()
	key := failedLoginAttempts + email

	count, err := s.Redis.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}

	// Set expiry on first failed attempt
	if count == 1 {
		s.Redis.Expire(ctx, key, failedLoginTTL)
	}

	// Lock account if max attempts reached
	if count >= maxFailedAttempts {
		s.Redis.Set(ctx, accountLocked+email, "locked", failedLoginTTL)
	}

	return int(count), nil
}

// ResetFailedLogin resets the failed login counter for an email
func (s *Store) ResetFailedLogin(c *echo.Context, email string) error {
	return s.Redis.Del(c.Request().Context(), failedLoginAttempts+email).Err()
}

// IsAccountLocked checks if an account is locked due to too many failed login attempts
func (s *Store) IsAccountLocked(c *echo.Context, email string) (bool, error) {
	ctx := c.Request().Context()

	val, err := s.Redis.Exists(ctx, accountLocked+email).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return false, nil
		}

		return false, err
	}

	return val > 0, nil
}

func getEnvAsInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	if intValue, err := strconv.Atoi(value); err == nil {
		return intValue
	}

	return defaultValue
}

func getEnvOrDefault(key, defaultVal string) string {
	val := strings.TrimSpace(os.Getenv(key))

	if val == "" {
		return defaultVal
	}

	return val
}
