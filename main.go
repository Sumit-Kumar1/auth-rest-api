package main

import (
	"auth-rest-api/internal/handler"
	"auth-rest-api/internal/server"
	"auth-rest-api/internal/service"
	"auth-rest-api/internal/store"

	"github.com/labstack/echo/v5"
)

func main() {
	app := echo.New()

	st, err := store.New()
	if err != nil {
		app.Logger.Error(err.Error())
		return
	}

	svc := service.New(st)
	h := handler.New(svc)

	rateLimitStore := server.NewRateLimitStore()

	// Global middleware
	app.Use(server.SecurityHeadersMiddleware())

	// Public routes (rate limited)
	app.POST("/signup", h.SignUp, server.RateLimitMiddleware(rateLimitStore, 5))
	app.POST("/signin", h.SignIn, server.RateLimitMiddleware(rateLimitStore, 5))

	// Authenticated routes (auth middleware + rate limited)
	auth := app.Group("", server.AuthMiddleware(), server.RateLimitMiddleware(rateLimitStore, 10))
	auth.POST("/validate", h.Validate)
	auth.POST("/refresh", h.RefreshToken)
	auth.POST("/revoke", h.RevokeToken)

	app.Start(":9001")
}
