package models

import (
	"regexp"
	"strings"
	"unicode"

	gofrHTTP "gofr.dev/pkg/gofr/http"
)

type CtxKey string

var CtxClaimKey CtxKey = "claims"

// UserReq represents the request payload for user-related operations.
// It contains the basic user information required for authentication.
type UserReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// UserResp represents the response payload for user-related operations.
// It contains the user information along with authentication tokens.
type UserResp struct {
	ID           string `json:"id,omitempty"`
	Email        string `json:"email,omitempty"`
	AccessToken  string `json:"accessToken,omitempty"`
	RefreshToken string `json:"refreshToken,omitempty"`
}

// UserData represents the internal user data structure.
// It contains the user's email and hashed password.
type UserData struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Password []byte `json:"-"`
}

// Validate performs validation on the UserReq fields.
// It checks both email and password meet the required criteria.
// Returns an error if validation fails.
func (u *UserReq) Validate() error {
	if err := ValidateEmail(u.Email); err != nil {
		return err
	}

	if err := validatePassword(u.Password); err != nil {
		return err
	}

	return nil
}

// ValidateEmail checks if the provided email address is valid.
// It performs the following checks:
// - Trims whitespace and converts to lowercase
// - Ensures the email is not empty
// - Validates against a regex pattern for email format
// Returns an error if the email is invalid.
func ValidateEmail(email string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	emailRegex := regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,4}$`)

	if email == "" {
		return gofrHTTP.ErrorMissingParam{Params: []string{"email"}}
	}

	if !emailRegex.MatchString(email) {
		return gofrHTTP.ErrorInvalidParam{Params: []string{"email"}}
	}

	return nil
}

func validatePassword(password string) error {
	var (
		hasUpper   = false
		hasLower   = false
		hasNumber  = false
		hasSpecial = false
		passwd     = strings.TrimSpace(password)
	)

	if passwd == "" {
		return gofrHTTP.ErrorMissingParam{Params: []string{"password"}}
	}

	if len(passwd) < 8 {
		return gofrHTTP.ErrorInvalidParam{Params: []string{"password must be at least 8 characters"}}
	}

	for _, char := range passwd {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsNumber(char):
			hasNumber = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	if !hasUpper {
		return gofrHTTP.ErrorInvalidParam{Params: []string{"password must contain at least one uppercase letter"}}
	}

	if !hasLower {
		return gofrHTTP.ErrorInvalidParam{Params: []string{"password must contain at least one lowercase letter"}}
	}

	if !hasNumber {
		return gofrHTTP.ErrorInvalidParam{Params: []string{"password must contain at least one number"}}
	}

	if !hasSpecial {
		return gofrHTTP.ErrorInvalidParam{Params: []string{"password must contain at least one special character"}}
	}

	return nil
}
