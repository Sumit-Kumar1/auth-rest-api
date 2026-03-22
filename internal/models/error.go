package models

import (
	"errors"
	"net/http"

	"gofr.dev/pkg/gofr/logging"
)

const (
	tokenRevoked = "token is revoked"
	unAuthorized = "unauthorized"
	invalidToken = "invalid token"
)

var (
	// Authentication
	ErrPasswordMismatch = errors.New("password does not match")
	ErrAccountLocked    = errors.New("account is temporarily locked")
)

type ErrTokenRevoked struct{}

func (ErrTokenRevoked) Error() string           { return tokenRevoked }
func (ErrTokenRevoked) StatusCode() int         { return http.StatusUnauthorized }
func (ErrTokenRevoked) LogLevel() logging.Level { return logging.ERROR }

type ErrUnAuthorized struct{}

func (ErrUnAuthorized) Error() string           { return unAuthorized }
func (ErrUnAuthorized) StatusCode() int         { return http.StatusUnauthorized }
func (ErrUnAuthorized) LogLevel() logging.Level { return logging.ERROR }

type ErrInvalidToken struct{}

func (ErrInvalidToken) Error() string           { return invalidToken }
func (ErrInvalidToken) StatusCode() int         { return http.StatusUnauthorized }
func (ErrInvalidToken) LogLevel() logging.Level { return logging.ERROR }
