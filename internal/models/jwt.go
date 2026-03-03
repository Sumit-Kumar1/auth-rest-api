package models

import "github.com/golang-jwt/jwt/v5"

// Claims represents the custom claims for JWT tokens.
// It extends the standard JWT claims with email and claim ID fields.
type Claims struct {
	Email    string `json:"email"`
	ClaimUID string `json:"claimID"`

	// registeredClaim's subject is userID from db
	jwt.RegisteredClaims
}
