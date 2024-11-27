package models

import (
	"regexp"
	"strings"
)

type UserReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type UserResp struct {
	Email        string `json:"email,omitempty"`
	AccessToken  string `json:"accessToken,omitempty"`
	RefreshToken string `json:"refreshToken,omitempty"`
}

type UserData struct {
	Email    string `json:"email"`
	Password []byte `json:"-"`
}

func (u *UserReq) Validate() error {
	if err := ValidateEmail(u.Email); err != nil {
		return err
	}

	if err := validatePassword(u.Password); err != nil {
		return err
	}

	return nil
}

func ValidateEmail(email string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	emailRegex := regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,4}$`)

	if email == "" {
		return ErrRequired("email")
	}

	if !emailRegex.MatchString(email) {
		return ErrInvalid("email")
	}

	return nil
}

func validatePassword(password string) error {
	passwd := strings.TrimSpace(password)

	if passwd == "" {
		return ErrRequired("password")
	}

	if len(passwd) < 8 {
		return ErrInvalid("password")
	}

	return nil
}
