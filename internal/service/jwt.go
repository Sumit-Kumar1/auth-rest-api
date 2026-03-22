package service

import (
	"errors"
	"os"
	"time"

	"auth-rest-api/internal/models"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// GenerateToken generates a new JWT token pair (access and refresh tokens).
// It creates tokens with appropriate expiration times and unique IDs.
// The access token expires in 15 minutes, while the refresh token expires in 24 hours.
// Returns the token data or an error if token generation fails.
func GenerateToken(idSub, email string) (*models.TokenData, error) {
	accID := uuid.NewString()
	refID := uuid.NewString()
	accessJTI := uuid.NewString()
	refreshJTI := uuid.NewString()

	claims := models.Claims{
		Email:    email,
		ClaimUID: accID,
		RegisteredClaims: jwt.RegisteredClaims{
			Audience:  jwt.ClaimStrings{"todo-app"},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute * 15)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "auth-api",
			Subject:   idSub,
			ID:        accessJTI,
		},
	}

	refClaims := jwt.RegisteredClaims{
		Audience:  jwt.ClaimStrings{"todo-app"},
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 24)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		Issuer:    "auth-api",
		Subject:   idSub,
		ID:        refreshJTI,
	}

	accessKey, refKey, err := getJWTSecrets()
	if err != nil {
		return nil, err
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	refToken := jwt.NewWithClaims(jwt.SigningMethodHS256, models.Claims{
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
func ParseToken(tokenString, tokenType string) (*models.Claims, error) {
	var (
		token *jwt.Token
		err   error
	)

	accSecret, refSecret, err := getJWTSecrets()
	if err != nil {
		return nil, err
	}

	switch tokenType {
	case "access":
		token, err = jwt.ParseWithClaims(tokenString, &models.Claims{}, func(_ *jwt.Token) (any, error) {
			return accSecret, nil
		}, jwt.WithExpirationRequired(), jwt.WithStrictDecoding())
	case "refresh":
		token, err = jwt.ParseWithClaims(tokenString, &models.Claims{}, func(_ *jwt.Token) (any, error) {
			return refSecret, nil
		}, jwt.WithExpirationRequired(), jwt.WithStrictDecoding())
	default:
		return nil, models.ErrInvalidToken{}
	}

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*models.Claims); ok && token.Valid {
		return claims, nil
	}

	if !token.Valid {
		return nil, models.ErrInvalidToken{}
	}

	return nil, err
}

// getJWTSecrets retrieves the JWT signing secrets from environment variables.
// It returns the access and refresh token secrets, or an error if not configured.
// Both ACCESS_SECRET and REFRESH_SECRET environment variables must be set.
func getJWTSecrets() (accessSecret, refreshSecret []byte, err error) {
	access := os.Getenv("ACCESS_SECRET")
	if access == "" {
		return nil, nil, errors.New("ACCESS_SECRET environment variable is required")
	}

	refresh := os.Getenv("REFRESH_SECRET")
	if refresh == "" {
		return nil, nil, errors.New("REFRESH_SECRET environment variable is required")
	}

	return []byte(access), []byte(refresh), nil
}
