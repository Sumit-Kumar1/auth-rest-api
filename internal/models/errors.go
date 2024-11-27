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

// CustomError error wrapper for sending in http response
type CustomError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type constError string

func NewConstError(message string) constError {
	return constError(message)
}

func (err constError) Error() string {
	return string(err)
}

func (err constError) Is(target error) bool {
	if targetErr, ok := target.(constError); ok {
		return string(err) == string(targetErr)
	}

	ts := target.Error()
	es := string(err)

	return ts == es || strings.HasPrefix(ts, es+": ")
}

func ErrNotFound(entity string) error {
	return NewConstError(fmt.Sprintf(notFoundFormat, entity))
}

func ErrInvalid(entity string) error {
	return NewConstError(fmt.Sprintf(invalidFormat, entity))
}

func ErrRequired(entity string) error {
	return NewConstError(fmt.Sprintf(requiredFormat, entity))
}
