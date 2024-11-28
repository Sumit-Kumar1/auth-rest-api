package models

import (
	"errors"
	"testing"
)

func TestUserReq_Validate(t *testing.T) {
	tests := []struct {
		name    string
		user    *UserReq
		wantErr error
	}{
		{name: "valid case", user: &UserReq{Email: "sumit@kumar.com", Password: "sumit@kumar"}, wantErr: nil},
		{name: "missing email", user: &UserReq{Email: "", Password: "sumit@kumar"}, wantErr: ErrRequired("email")},
		{name: "missing password", user: &UserReq{Email: "sumit@kumar.com", Password: ""}, wantErr: ErrRequired("password")},
		{name: "passwd len < 8", user: &UserReq{Email: "sumit@kumar.com", Password: "sumit"}, wantErr: ErrInvalid("password")},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.user.Validate(); err != nil && !errors.Is(err, tt.wantErr) {
				t.Errorf("Test[%d] Failed - %s\nExp:%v\nGot:%v", i, tt.name, err, tt.wantErr)
			}
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
		{name: "invalid email", email: "sumit@kumar", wantErr: ErrInvalid("email")},
		{name: "invalid email", email: "", wantErr: ErrRequired("email")},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateEmail(tt.email); tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Errorf("Test[%d] Failed - %s\nExp:%v\nGot:%v", i, tt.name, err, tt.wantErr)
			}
		})
	}
}

func Test_validatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  error
	}{
		{name: "valid case", password: "sumit@kumar", wantErr: nil},
		{name: "passwd len < 8", password: "sumit", wantErr: ErrInvalid("password")},
		{name: "missing password", password: "", wantErr: ErrRequired("password")},
	}
	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validatePassword(tt.password); tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Errorf("Test[%d] Failed - %s\nExp:%v\nGot:%v", i, tt.name, err, tt.wantErr)
			}
		})
	}
}
