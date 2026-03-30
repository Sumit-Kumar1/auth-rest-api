package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"auth-rest-api/internal/models"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

const testFailStr = "TEST[%d] Failed - %s"

func generateTestToken(t *testing.T, secret string) string {
	t.Helper()

	claims := models.Claims{
		Email:    "test@example.com",
		ClaimUID: uuid.NewString(),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   uuid.NewString(),
			ID:        uuid.NewString(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(secret))
	assert.NoError(t, err)

	return tokenStr
}

func TestAuthMiddleware_ProtectedRoutes(t *testing.T) {
	t.Setenv("ACCESS_SECRET", "test_secret")

	validToken := generateTestToken(t, "test_secret")

	tests := []struct {
		name       string
		path       string
		authHeader string
		wantPass   bool
	}{
		{
			name:       "validate with valid token",
			path:       "/validate",
			authHeader: "Bearer " + validToken,
			wantPass:   true,
		},
		{
			name:       "refresh with valid token",
			path:       "/refresh",
			authHeader: "Bearer " + validToken,
			wantPass:   true,
		},
		{
			name:       "revoke with valid token",
			path:       "/revoke",
			authHeader: "Bearer " + validToken,
			wantPass:   true,
		},
		{
			name:       "validate without token",
			path:       "/validate",
			authHeader: "",
			wantPass:   false,
		},
		{
			name:       "validate with invalid token",
			path:       "/validate",
			authHeader: "Bearer invalid-token",
			wantPass:   false,
		},
		{
			name:       "validate with wrong secret token",
			path:       "/validate",
			authHeader: "Bearer " + generateTestToken(t, "wrong_secret"),
			wantPass:   false,
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlerCalled := false
			nextHandler := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
				handlerCalled = true
			})

			middleware := AuthMiddleware()
			handler := middleware(nextHandler)

			req := httptest.NewRequest(http.MethodPost, tt.path, http.NoBody)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			assert.Equalf(t, tt.wantPass, handlerCalled, testFailStr, i, tt.name)
		})
	}
}

func TestAuthMiddleware_PublicRoutes(t *testing.T) {
	paths := []string{"/signup", "/signin", "/health", "/other"}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			handlerCalled := false
			nextHandler := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
				handlerCalled = true
			})

			middleware := AuthMiddleware()
			handler := middleware(nextHandler)

			req := httptest.NewRequest(http.MethodPost, path, http.NoBody)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			assert.True(t, handlerCalled, "public route %s should pass through", path)
		})
	}
}

func TestAuthMiddleware_ClaimsInjected(t *testing.T) {
	t.Setenv("ACCESS_SECRET", "test_secret")

	validToken := generateTestToken(t, "test_secret")

	nextHandler := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		claims, ok := r.Context().Value(models.CtxClaimKey).(*models.Claims)
		assert.True(t, ok, "claims should be in context")
		assert.NotNil(t, claims)
		assert.Equal(t, "test@example.com", claims.Email)
	})

	middleware := AuthMiddleware()
	handler := middleware(nextHandler)

	req := httptest.NewRequest(http.MethodPost, "/validate", http.NoBody)
	req.Header.Set("Authorization", "Bearer "+validToken)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
}

func TestValidate(t *testing.T) {
	t.Setenv("ACCESS_SECRET", "test_secret")

	validToken := generateTestToken(t, "test_secret")

	tests := []struct {
		name       string
		authHeader string
		wantErr    bool
	}{
		{
			name:       "valid token",
			authHeader: "Bearer " + validToken,
		},
		{
			name:       "empty header",
			authHeader: "",
			wantErr:    true,
		},
		{
			name:       "whitespace header",
			authHeader: "   ",
			wantErr:    true,
		},
		{
			name:       "invalid token string",
			authHeader: "Bearer invalid.token.here",
			wantErr:    true,
		},
		{
			name:       "wrong signing key",
			authHeader: "Bearer " + generateTestToken(t, "wrong_secret"),
			wantErr:    true,
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := validate(tt.authHeader)
			if tt.wantErr {
				assert.Errorf(t, err, testFailStr, i, tt.name)
				assert.Nil(t, claims)
			} else {
				assert.NoErrorf(t, err, testFailStr, i, tt.name)
				assert.NotNil(t, claims)
				assert.Equal(t, "test@example.com", claims.Email)
			}
		})
	}
}

func TestGetJWTSecret(t *testing.T) {
	tests := []struct {
		name    string
		envVal  string
		wantErr bool
	}{
		{
			name:   "valid secret",
			envVal: "my_secret_key",
		},
		{
			name:    "missing secret",
			envVal:  "",
			wantErr: true,
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("ACCESS_SECRET", tt.envVal)

			secret, err := getJWTSecret()
			if tt.wantErr {
				assert.Errorf(t, err, testFailStr, i, tt.name)
				assert.Nil(t, secret)
			} else {
				assert.NoErrorf(t, err, testFailStr, i, tt.name)
				assert.Equal(t, []byte(tt.envVal), secret)
			}
		})
	}
}
