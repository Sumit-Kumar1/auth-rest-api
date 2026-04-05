// Package store provides
package store

import (
	"errors"
	"time"

	"auth-rest-api/internal/models"

	"github.com/redis/go-redis/v9"
	"gofr.dev/pkg/gofr"
	gofrHTTP "gofr.dev/pkg/gofr/http"
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

type Store struct {
}

func New() *Store {
	return &Store{}
}

// CreateUser stores a new user in the database
func (s *Store) CreateUser(c *gofr.Context, user *models.UserData) error {
	// Store email → user_id mapping, this helps in getEmail calls
	err := c.Redis.Set(c.Context, emaild+user.Email, user.ID, 0).Err()
	if errors.Is(err, redis.Nil) {
		return gofrHTTP.ErrorEntityAlreadyExist{}
	}

	if err != nil {
		return err
	}

	// store user hash data with uniqueness of userId
	err = c.Redis.HSet(c.Context, userd+user.ID, map[string]any{
		"id": user.ID, "email": user.Email, "password": user.Password}).Err()

	if errors.Is(err, redis.Nil) {
		return gofrHTTP.ErrorEntityAlreadyExist{}
	}

	if err != nil {
		return err
	}

	return nil
}

// GetUserByEmail retrieves a user from the database by their email.
func (s *Store) GetUserByEmail(c *gofr.Context, userEmail string) (*models.UserData, error) {
	userID, err := c.Redis.Get(c.Context, emaild+userEmail).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, gofrHTTP.ErrorEntityNotFound{Name: "user", Value: userEmail}
		}

		return nil, err
	}

	data, err := c.Redis.HGetAll(c.Context, userd+userID).Result()
	if err != nil {
		return nil, err
	}

	if len(data) == 0 {
		return nil, gofrHTTP.ErrorEntityNotFound{Name: "user", Value: userEmail}
	}

	return &models.UserData{ID: data["id"], Email: data["email"], Password: []byte(data["password"])}, nil
}

// CreateToken stores a new token in the database.
func (s *Store) CreateToken(c *gofr.Context, email string, td *models.TokenData) error {
	accExp := time.Until(time.Unix(td.AccessExpiresAt, 0))
	refExp := time.Until(time.Unix(td.RefreshExpiresAt, 0))

	tx := c.Redis.TxPipeline()

	tx.Set(c.Context, accTokend+td.AccessID, email, accExp)
	tx.SAdd(c.Context, userAccTokend+email, td.AccessID)

	tx.Set(c.Context, refTokend+td.RefreshID, email, refExp)
	tx.SAdd(c.Context, userRefTokend+email, td.RefreshID)

	_, err := tx.Exec(c)

	return err
}

// DeleteToken removes one or more tokens from the database.
func (s *Store) DeleteToken(c *gofr.Context, email, accTokenID, refTokenID string) error {
	tx := c.Redis.TxPipeline()

	tx.Del(c.Context, accTokend+accTokenID)
	tx.SRem(c.Context, userAccTokend+email, accTokenID)

	if refTokenID != "" {
		tx.Del(c.Context, refTokend+refTokenID)
		tx.SRem(c.Context, userRefTokend+email, refTokenID)
	}

	_, err := tx.Exec(c)

	return err
}

// IsTokenRevoked checks if a token has been revoked (token doesn't exist).
func (s *Store) IsTokenRevoked(c *gofr.Context, tokenID string) (bool, error) {
	val, err := c.Redis.Exists(c.Context, accTokend+tokenID).Result()
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
func (s *Store) IncrementFailedLogin(c *gofr.Context, email string) (int, error) {
	key := failedLoginAttempts + email

	count, err := c.Redis.Incr(c.Context, key).Result()
	if err != nil {
		return 0, err
	}

	// Set expiry on first failed attempt
	if count == 1 {
		c.Redis.Expire(c.Context, key, failedLoginTTL)
	}

	// Lock account if max attempts reached
	if count >= maxFailedAttempts {
		c.Redis.Set(c.Context, accountLocked+email, "locked", failedLoginTTL)
	}

	return int(count), nil
}

// ResetFailedLogin resets the failed login counter for an email
func (s *Store) ResetFailedLogin(c *gofr.Context, email string) error {
	return c.Redis.Del(c.Context, failedLoginAttempts+email).Err()
}

// IsAccountLocked checks if an account is locked due to too many failed login attempts
func (s *Store) IsAccountLocked(c *gofr.Context, email string) (bool, error) {
	val, err := c.Redis.Exists(c.Context, accountLocked+email).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return false, nil
		}

		return false, err
	}

	return val > 0, nil
}
