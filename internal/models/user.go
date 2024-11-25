package models

import (
	"errors"
	"regexp"
	"strings"
)

type UserReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type UserResp struct {
	Email string `json:"email"`
	Token string `json:"token"`
}

func (u *UserReq) Validate() error {
	email := strings.ToLower(strings.TrimSpace(u.Email))
	passwd := strings.TrimSpace(u.Password)

	emailRegex := regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,4}$`)

	if email == "" {
		return errors.New("email is required")
	}

	if !emailRegex.MatchString(email) {
		return errors.New("invalid email")
	}

	if passwd == "" {
		return errors.New("password is required")
	}

	if len(passwd) < 8 {
		return errors.New("password is too short")
	}

	return nil
}
