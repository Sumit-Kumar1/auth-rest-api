package service

import (
	"errors"
	"testing"
	"time"

	"auth-rest-api/internal/models"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

var (
	email = "test@example.com"
	accID = uuid.NewString()
	refID = uuid.NewString()
)

func Test_getJWTSecrets(t *testing.T) {
	tests := []struct {
		name              string
		accEnv            string
		refEnv            string
		wantAccessSecret  []byte
		wantRefreshSecret []byte
	}{
		{name: "missing access secret", accEnv: "", wantAccessSecret: []byte("my_secret_key"), wantRefreshSecret: []byte("my_refresh_secret_key")},
		{name: "missing refresh secret", accEnv: "ABCD", wantAccessSecret: []byte("my_secret_key"), wantRefreshSecret: []byte("my_refresh_secret_key")},
		{name: "valid secrets", accEnv: "ABCD", refEnv: "XYZ", wantAccessSecret: []byte("ABCD"), wantRefreshSecret: []byte("XYZ")},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("ACCESS_SECRET", tt.accEnv)
			t.Setenv("REFRESH_SECRET", tt.refEnv)

			gotAccessSecret, gotRefreshSecret := getJWTSecrets()
			assert.Equalf(t, tt.wantAccessSecret, gotAccessSecret, "TEST[%d] Failed - %s", i, tt.name)
			assert.Equalf(t, tt.wantRefreshSecret, gotRefreshSecret, "TEST[%d] Failed - %s", i, tt.name)
		})
	}
}

func TestParseToken(t *testing.T) {
	t.Setenv("ACCESS_SECRET", "ABCD")
	t.Setenv("REFRESH_SECRET", "XYZ")
	accessKey, refKey := getJWTSecrets()

	accClaims := Claims{
		Email:    email,
		ClaimUID: accID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Second * 1)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "sumit kumar",
			Subject:   email,
			ID:        "1",
		},
	}

	refClaims := Claims{
		Email:    email,
		ClaimUID: refID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Second * 1)),
			Subject:   email,
			ID:        "1",
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accClaims)
	valAccToken, err := accessToken.SignedString(accessKey)

	assert.NoError(t, err)

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refClaims)
	valRefToken, err := refreshToken.SignedString(refKey)

	assert.NoError(t, err)

	tests := []struct {
		name      string
		token     string
		tokenType string
		want      *Claims
		wantErr   error
	}{
		{
			name:  "valid case - access token",
			token: valAccToken, tokenType: "access",
			want: &accClaims,
		},
		{
			name:  "valid case - refresh token",
			token: valRefToken, tokenType: "refresh",
			want: &refClaims,
		},
		{
			name:  "invalid token type",
			token: valRefToken, tokenType: "ref",
			wantErr: errors.New("invalid token type"),
		},
		{
			name:  "invalid token type",
			token: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJlbWFpbCI6InN1bWl0QGt1bWFyLmNvbSIsImNsYWltSUQiOiJjNjVmMzNjYS1hNjZhLTQ1NTgtODdjNS01NTgxNGNjZWQ5ZmYiLCJzdWIiOiJzdW1pdEBrdW1hci5jb20iLCJleHAiOjE3MzI3OTg2MzEsImlhdCI6MTczMjcxMjIzMX0.leAyGmgmAqQEdkQexD8C5GzBXIZhR9HTib-tagNbbqw", tokenType: "refresh",
			wantErr: errors.New("invalid token type"),
		},
	}
	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseToken(tt.token, tt.tokenType)

			assert.Equalf(t, tt.wantErr, err, "TEST[%d] Failed - %s", i, tt.name)
			assert.Equalf(t, tt.want, got, "TEST[%d] Failed - %s", i, tt.name)
		})
	}
}

func TestGenerateToken(t *testing.T) {
	id := uuid.NewString()
	uuid.DisableRandPool()

	defer uuid.EnableRandPool()

	tests := []struct {
		name    string
		email   string
		want    *models.TokenData
		wantErr error
	}{
		{name: "valid case", email: email, want: &models.TokenData{
			AccessToken:      "access_token",
			AccessExpiresAt:  jwt.NewNumericDate(time.Now().Add(time.Minute * 15)).Unix(),
			RefreshExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 24)).Unix(),
			AccessID:         id,
			RefreshID:        id,
			RefreshToken:     "refresh_token",
		}},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := GenerateToken(tt.email)

			assert.Equalf(t, tt.wantErr, err, "TEST[%d] Failed - %s", i, tt.name)
		})
	}
}
