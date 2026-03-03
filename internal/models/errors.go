package models

import (
	"errors"
	"net/http"
)

var (
	// Authentication
	ErrUnauthorized     = errors.New("unauthorized")
	ErrPasswordMismatch = errors.New("password does not match")
	ErrAccountLocked    = errors.New("account is temporarily locked")

	// Token
	ErrTokenRevoked     = errors.New("token is revoked")
	ErrInvalidTokenType = errors.New("invalid token type")

	// User
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrUserNotFound      = errors.New("user does not exist")

	// Validation: email
	ErrEmailRequired = errors.New("email is required")
	ErrEmailInvalid  = errors.New("invalid email")

	// Validation: password
	ErrPasswordRequired  = errors.New("password is required")
	ErrPasswordTooShort  = errors.New("password must be at least 8 characters")
	ErrPasswordNoUpper   = errors.New("password must contain at least one uppercase letter")
	ErrPasswordNoLower   = errors.New("password must contain at least one lowercase letter")
	ErrPasswordNoNumber  = errors.New("password must contain at least one number")
	ErrPasswordNoSpecial = errors.New("password must contain at least one special character")

	// Validation: general
	ErrInvalidInput = errors.New("invalid input")
)

// HTTPError is an error carrying an HTTP status code for API responses.
// Use Unwrap() to access the underlying domain error for errors.Is checks.
type HTTPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
	Err     error  `json:"-"`
}

func (e *HTTPError) Error() string {
	return e.Message
}

func (e *HTTPError) Unwrap() error {
	return e.Err
}

// NewHTTPError creates an HTTPError for handler-layer JSON responses.
func NewHTTPError(code int, msg, details string) *HTTPError {
	return &HTTPError{Code: code, Message: msg, Details: details}
}

// ErrBadRequest wraps err as a 400 Bad Request.
// The original error is preserved for errors.Is / errors.As checks.
func ErrBadRequest(err error) *HTTPError {
	return &HTTPError{
		Code:    http.StatusBadRequest,
		Message: err.Error(),
		Err:     err,
	}
}
