package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	gofrHTTP "gofr.dev/pkg/gofr/http"
)

func TestUserReq_Validate(t *testing.T) {
	tests := []struct {
		name    string
		user    *UserReq
		wantErr error
	}{
		{name: "valid case", user: &UserReq{Email: "sumit@kumar.com", Password: "Sumit@Kumar123"}, wantErr: nil},
		{name: "missing email", user: &UserReq{Email: "", Password: "Sumit@Kumar123"}, wantErr: gofrHTTP.ErrorInvalidParam{Params: []string{emailStr}}},
		{name: "missing password", user: &UserReq{Email: "sumit@kumar.com", Password: ""}, wantErr: gofrHTTP.ErrorMissingParam{Params: []string{passwd}}},
		{name: "passwd len < 8", user: &UserReq{Email: "sumit@kumar.com", Password: "Sum1@K"}, wantErr: gofrHTTP.ErrorInvalidParam{Params: []string{passwdLenErr}}},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.user.Validate()
			assert.Equalf(t, tt.wantErr, err, "Test[%d] failed - %s", i, tt.name)
		})
	}
}

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		wantErr error
	}{
		{name: "valid csae", email: "sumit@kumar.com", wantErr: nil},
		{name: "invalid email", email: "sumit@kumar", wantErr: gofrHTTP.ErrorInvalidParam{Params: []string{emailStr}}},
		{name: "empty email", email: "", wantErr: gofrHTTP.ErrorMissingParam{Params: []string{emailStr}}},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEmail(tt.email)
			assert.Equalf(t, tt.wantErr, err, "Test[%d] failed - %s", i, tt.name)
		})
	}
}

func Test_validatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  error
	}{
		{name: "valid case", password: "Sumit@Kumar123", wantErr: nil},
		{name: "passwd len < 8", password: "sumit", wantErr: gofrHTTP.ErrorInvalidParam{Params: []string{passwdLenErr}}},
		{name: "missing password", password: "", wantErr: gofrHTTP.ErrorMissingParam{Params: []string{passwd}}},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePassword(tt.password)
			assert.Equalf(t, tt.wantErr, err, "Test[%d] failed - %s", i, tt.name)
		})
	}
}
