package models

import (
	"errors"
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
