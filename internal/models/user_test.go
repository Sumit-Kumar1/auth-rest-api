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
		{name: "missing email", user: &UserReq{Email: "", Password: "sumit@kumar"}, wantErr: errors.New("email is required")},
		{name: "missing password", user: &UserReq{Email: "sumit@kumar.com", Password: ""}, wantErr: errors.New("password is required")},
		{name: "passwd len < 8", user: &UserReq{Email: "sumit@kumar.com", Password: "sumit"}, wantErr: errors.New("password is too short")},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.user.Validate(); err != nil && errors.Is(err, tt.wantErr) {
				t.Errorf("Test[%d] Failed - %s\nExp:%v\nGot:%v", i, tt.name, err, tt.wantErr)
			}
		})
	}
}
