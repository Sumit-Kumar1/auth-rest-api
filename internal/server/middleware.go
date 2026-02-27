package server

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"auth-rest-api/internal/models"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v5"
)

// SecurityHeadersMiddleware adds common security-related HTTP headers to every response.
func SecurityHeadersMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			c.Response().Header().Set("X-Content-Type-Options", "nosniff")
			c.Response().Header().Set("X-Frame-Options", "DENY")
			c.Response().Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains; preload")
			c.Response().Header().Set("Content-Security-Policy", "default-src 'self'")
			c.Response().Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			c.Response().Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

			return next(c)
		}
	}
}

// AuthMiddleware creates a middleware that validates JWT tokens.
// It checks for the presence of an Authorization header and validates the token.
// Returns an unauthorized error if the token is missing or invalid.
func AuthMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			token, err := validate(c.Request().Header.Get("Authorization"))
			if err != nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{"message": "Unauthorized"})
			}

			if token != "" {
				ctx := context.WithValue(c.Request().Context(), "userID", token)
				c.SetRequest(c.Request().WithContext(ctx))
			}

			return next(c)
		}
	}
}

func validate(authHeader string) (string, error) {
	if strings.TrimSpace(authHeader) == "" {
		return "", models.ErrUnauthorized
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")

	secret, err := getJWTSecret()
	if err != nil {
		return "", err
	}

	token, err := jwt.Parse(tokenString, func(_ *jwt.Token) (any, error) {
		return secret, nil
	})
	if err != nil || !token.Valid {
		return "", models.ErrUnauthorized
	}

	return token.Claims.GetSubject()
}

// getJWTSecret retrieves the JWT signing secret from environment variables.
func getJWTSecret() ([]byte, error) {
	secret := os.Getenv("ACCESS_SECRET")
	if secret == "" {
		return nil, errors.New("ACCESS_SECRET environment variable is required")
	}

	return []byte(secret), nil
}

// RateLimitStore stores rate limit information per IP address
type RateLimitStore struct {
	mu    sync.Mutex
	store map[string]*RateLimit
}

// RateLimit tracks requests for a specific IP
type RateLimit struct {
	requests []time.Time
	lastSeen time.Time
}

// NewRateLimitStore creates a new rate limit store
func NewRateLimitStore() *RateLimitStore {
	return &RateLimitStore{
		store: make(map[string]*RateLimit),
	}
}

// IsAllowed checks if a request from the given IP is allowed based on the limit
func (s *RateLimitStore) IsAllowed(ip string, requestsPerMinute int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	limit, exists := s.store[ip]

	if !exists {
		s.store[ip] = &RateLimit{
			requests: []time.Time{now},
			lastSeen: now,
		}
		return true
	}

	cutoff := now.Add(-time.Minute)
	validRequests := []time.Time{}
	for _, req := range limit.requests {
		if req.After(cutoff) {
			validRequests = append(validRequests, req)
		}
	}

	if len(validRequests) >= requestsPerMinute {
		return false
	}

	validRequests = append(validRequests, now)
	s.store[ip] = &RateLimit{
		requests: validRequests,
		lastSeen: now,
	}

	return true
}

// Cleanup removes old entries (call periodically)
func (s *RateLimitStore) Cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for ip, limit := range s.store {
		if now.Sub(limit.lastSeen) > 10*time.Minute {
			delete(s.store, ip)
		}
	}
}

// RateLimitMiddleware creates a middleware that rate limits requests by IP
func RateLimitMiddleware(store *RateLimitStore, requestsPerMinute int) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			ip := c.RealIP()

			if !store.IsAllowed(ip, requestsPerMinute) {
				return c.JSON(http.StatusTooManyRequests, map[string]string{"message": "Too many requests"})
			}

			return next(c)
		}
	}
}
