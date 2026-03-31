package main

import (
	"auth-rest-api/internal/handler"
	"auth-rest-api/internal/server"
	"auth-rest-api/internal/service"
	"auth-rest-api/internal/store"

	"gofr.dev/pkg/gofr"
)

func main() {
	app := gofr.New()

	st := store.New()
	svc := service.New(st)
	h := handler.New(svc)

	app.UseMiddleware(server.AuthMiddleware())

	// Public routes (rate limited)
	app.POST("/signup", h.SignUp)
	app.POST("/signin", h.SignIn)

	// Authenticated routes (auth middleware + rate limited)
	app.POST("/validate", h.Validate)
	app.POST("/refresh", h.RefreshToken)
	app.POST("/revoke", h.RevokeToken)

	app.Run()
}
