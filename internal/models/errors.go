package models

import (
	"errors"
	"fmt"
)

const (
	invalidFormat  = "invalid %s"
	requiredFormat = "%s is required"
)

var (
	ErrDBNotConnected    = NewConstError("database not connected")
	ErrTokenRevoked      = NewConstError("token is revoked")
	ErrUserAlreadyExists = NewConstError("user already exists")
	ErrPasswordMismatch  = NewConstError("password does not match")
	ErrUserNotFound      = NewConstError("user does not exist")
)

// CustomError represents an error that can be sent in HTTP responses.
// It includes an HTTP status code and an error message.
type CustomError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// Error implements the error interface for CustomError.
// It returns the error message.
func (e *CustomError) Error() string {
	return e.Message
}

// NewHTTPError created a custom error based on code, msg and detail of error,
// use only at handler layer
func NewHTTPError(code int, msg, details string) *CustomError {
	return &CustomError{
		Code:    code,
		Message: msg,
		Details: details,
	}
}

// ErrBadRequest creates an error for bad request scenarios.
// It wraps the provided error in a CustomError with a 400 status code.
func ErrBadRequest(err error) *CustomError {
	return NewHTTPError(400, err.Error(), "")
}

// ConstError is a type that implements the error interface.
// It's used for creating constant error values for internal error use.
type ConstError string

// NewConstError creates a new constant error with the given message.
// It returns a constError that can be used as a constant error value.
func NewConstError(message string) ConstError {
	return ConstError(message)
}

// Error implements the error interface for constError.
// It returns the string representation of the error.
func (err ConstError) Error() string {
	return string(err)
}

// Is implements error comparison for constError.
// It allows checking if an error matches a specific constError value.
func (err ConstError) Is(target error) bool {
	var t ConstError

	ok := errors.As(target, &t)
	if !ok {
		return false
	}

	return err == t
}

// ErrInvalid creates an error for invalid entity scenarios.
// It formats the error message using the invalidFormat constant.
func ErrInvalid(entity string) error {
	return NewConstError(fmt.Sprintf(invalidFormat, entity))
}

// ErrRequired creates an error for required field scenarios.
// It formats the error message using the requiredFormat constant.
func ErrRequired(entity string) error {
	return NewConstError(fmt.Sprintf(requiredFormat, entity))
}
