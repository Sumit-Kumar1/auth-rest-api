package models

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"gofr.dev/pkg/gofr/logging"
)

func TestErrTokenRevoked(t *testing.T) {
	err := ErrTokenRevoked{}

	assert.Equal(t, tokenRevoked, err.Error())
	assert.Equal(t, http.StatusUnauthorized, err.StatusCode())
	assert.Equal(t, logging.ERROR, err.LogLevel())
}

func TestErrUnAuthorized(t *testing.T) {
	err := ErrUnAuthorized{}

	assert.Equal(t, unAuthorized, err.Error())
	assert.Equal(t, http.StatusUnauthorized, err.StatusCode())
	assert.Equal(t, logging.ERROR, err.LogLevel())
}

func TestErrInvalidToken(t *testing.T) {
	err := ErrInvalidToken{}

	assert.Equal(t, invalidToken, err.Error())
	assert.Equal(t, http.StatusUnauthorized, err.StatusCode())
	assert.Equal(t, logging.ERROR, err.LogLevel())
}

func TestErrPasswordMismatch(t *testing.T) {
	err := ErrPasswordMismatch{}

	assert.Equal(t, passwordMismatch, err.Error())
	assert.Equal(t, http.StatusUnauthorized, err.StatusCode())
	assert.Equal(t, logging.ERROR, err.LogLevel())
}

func TestErrAccountLocked(t *testing.T) {
	err := ErrAccountLocked{}

	assert.Equal(t, accountLocked, err.Error())
	assert.Equal(t, http.StatusForbidden, err.StatusCode())
	assert.Equal(t, logging.ERROR, err.LogLevel())
}
