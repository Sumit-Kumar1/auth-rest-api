package service

import (
	"encoding/json"
	"errors"
	"os"
	"time"

	"auth-rest-api/internal/models"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Claims represents the custom claims for JWT tokens.
// It extends the standard JWT claims with email and claim ID fields.
type Claims struct {
	Email    string `json:"email"`
	ClaimUID string `json:"claimID"`
	jwt.RegisteredClaims
}

// GenerateToken generates a new JWT token pair (access and refresh tokens).
// It creates tokens with appropriate expiration times and unique IDs.
// The access token expires in 15 minutes, while the refresh token lasts longer.
// Returns the token data or an error if token generation fails.
func GenerateToken(email string) (*models.TokenData, error) {
	accID := uuid.NewString()
	refID := uuid.NewString()
	claims := Claims{
		Email:    email,
		ClaimUID: accID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute * 15)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "sumit kumar",
			Subject:   email,
			ID:        "1",
		},
	}

	refClaims := jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 24)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		Subject:   email,
	}

	accessKey, refKey := getJWTSecrets()

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	refToken := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		Email:            email,
		ClaimUID:         refID,
		RegisteredClaims: refClaims,
	})

	accessTokenStr, err := accessToken.SignedString(accessKey)
	if err != nil {
		return nil, err
	}

	refTokenStr, err := refToken.SignedString(refKey)
	if err != nil {
		return nil, err
	}

	tkData := models.TokenData{
		AccessID:         accID,
		AccessExpiresAt:  claims.ExpiresAt.Unix(),
		AccessToken:      accessTokenStr,
		RefreshID:        refID,
		RefreshToken:     refTokenStr,
		RefreshExpiresAt: refClaims.ExpiresAt.Unix(),
	}

	return &tkData, nil
}

// ParseToken validates and parses a JWT token.
// It verifies the token signature and expiration time.
// Returns the token claims or an error if the token is invalid.
func ParseToken(tokenString, tokenType string) (*Claims, error) {
	var (
		token *jwt.Token
		err   error
	)

	accSecret, refSecret := getJWTSecrets()

	switch tokenType {
	case "access":
		token, err = jwt.ParseWithClaims(tokenString, &Claims{}, func(_ *jwt.Token) (any, error) {
			return accSecret, nil
		})
	case "refresh":
		token, err = jwt.ParseWithClaims(tokenString, &Claims{}, func(_ *jwt.Token) (any, error) {
			return refSecret, nil
		})
	default:
		return nil, errors.New("invalid token type")
	}

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	if !token.Valid {
		return nil, jwt.ErrSignatureInvalid
	}

	return nil, err
}

// getJWTSecrets retrieves the JWT signing secrets from environment variables.
// It returns the access and refresh token secrets.
// If environment variables are not set, it returns default values.
func getJWTSecrets() (accessSecret, refreshSecret []byte) {
	access := os.Getenv("ACCESS_SECRET")
	if access == "" {
		return json.RawMessage("my_secret_key"), json.RawMessage("my_refresh_secret_key")
	}

	refresh := os.Getenv("REFRESH_SECRET")
	if refresh == "" {
		return json.RawMessage("my_secret_key"), json.RawMessage("my_refresh_secret_key")
	}

	return json.RawMessage(access), json.RawMessage(refresh)
}
