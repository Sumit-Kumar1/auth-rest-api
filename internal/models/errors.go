package models

import (
	"fmt"
	"strings"
)

const (
	notFoundFormat = "%s not found"
	invalidFormat  = "invalid %s"
	requiredFormat = "%s is required"
)

var (
	ErrDBNotConnected    = constError("database not connected")
	ErrTokenRevoked      = constError("token is revoked")
	ErrUserAlreadyExists = constError("user already exists")
	ErrPsswdNotMatch     = constError("password does not match")
)

// CustomError represents an error that can be sent in HTTP responses.
// It includes an HTTP status code and an error message.
type CustomError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Error implements the error interface for CustomError.
// It returns the error message.
func (e *CustomError) Error() string {
	return e.Message
}

// constError is a type that implements the error interface.
// It's used for creating constant error values.
type constError string

// NewConstError creates a new constant error with the given message.
// It returns a constError that can be used as a constant error value.
func NewConstError(message string) constError {
	return constError(message)
}

// Error implements the error interface for constError.
// It returns the string representation of the error.
func (err constError) Error() string {
	return string(err)
}

// Is implements error comparison for constError.
// It allows checking if an error matches a specific constError value.
func (err constError) Is(target error) bool {
	t, ok := target.(constError)
	if !ok {
		return false
	}

	return strings.EqualFold(string(err), string(t))
}

// ErrNotFound creates an error for when an entity is not found.
// It formats the error message using the notFoundFormat constant.
func ErrNotFound(entity string) error {
	return fmt.Errorf(notFoundFormat, entity)
}

// ErrBadRequest creates an error for bad request scenarios.
// It wraps the provided error in a CustomError with a 400 status code.
func ErrBadRequest(err error) error {
	return &CustomError{
		Code:    400,
		Message: err.Error(),
	}
}

// ErrInvalid creates an error for invalid entity scenarios.
// It formats the error message using the invalidFormat constant.
func ErrInvalid(entity string) error {
	return fmt.Errorf(invalidFormat, entity)
}

// ErrRequired creates an error for required field scenarios.
// It formats the error message using the requiredFormat constant.
func ErrRequired(entity string) error {
	return fmt.Errorf(requiredFormat, entity)
}
