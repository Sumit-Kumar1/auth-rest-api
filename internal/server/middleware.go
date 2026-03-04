package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"

	"auth-rest-api/internal/models"

	"github.com/golang-jwt/jwt/v5"
	gofrHTTP "gofr.dev/pkg/gofr/http"
)

func AuthMiddleware() gofrHTTP.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/validate", "/refresh", "/revoke":
				claims, err := validate(r.Header.Get("Authorization"))
				if err != nil {
					w.WriteHeader(http.StatusUnauthorized)
					w.Write(json.RawMessage(`unauthorized access, ` + err.Error()))
					return
				}

				if claims == nil {
					w.WriteHeader(http.StatusUnauthorized)
					w.Write(json.RawMessage(`unauthorized access`))
					return
				}

				keyRequest := r.WithContext(context.WithValue(r.Context(), models.CtxClaimKey, claims))

				next.ServeHTTP(w, keyRequest)

			default:
				next.ServeHTTP(w, r)
			}
		})
	}
}

func validate(authHeader string) (*models.Claims, error) {
	if strings.TrimSpace(authHeader) == "" {
		return nil, models.ErrUnauthorized
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")

	secret, err := getJWTSecret()
	if err != nil {
		return nil, err
	}

	token, err := jwt.Parse(tokenString, func(_ *jwt.Token) (any, error) {
		return secret, nil
	})
	if err != nil {
		return nil, models.ErrUnauthorized
	}

	if claims, ok := token.Claims.(*models.Claims); ok && token.Valid {
		return claims, nil
	}

	if !token.Valid {
		return nil, jwt.ErrSignatureInvalid
	}

	return nil, err
}

// getJWTSecret retrieves the JWT signing secret from environment variables.
func getJWTSecret() ([]byte, error) {
	secret := os.Getenv("ACCESS_SECRET")
	if secret == "" {
		return nil, errors.New("ACCESS_SECRET environment variable is required")
	}

	return []byte(secret), nil
}
