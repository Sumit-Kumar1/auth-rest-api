package server

import (
	"auth-rest-api/internal/models"
	"context"
	"log/slog"
	"os"

	"github.com/redis/go-redis/v9"
)

type Database struct {
	Client *redis.Client
}

func newDB(logger *slog.Logger) (*Database, error) {
	addr := os.Getenv("DB_ADDRESS")
	if addr == "" {
		addr = "localhost:6379"
	}

	psswd := os.Getenv("DB_PASSWORD")
	name := getEnvAsInt("DB_NAME", 0)

	rClient := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: psswd,
		DB:       name,
	})

	if rClient == nil {
		return nil, models.ErrDBNotConnected
	}

	rcmd := rClient.Ping(context.Background())
	if err := rcmd.Err(); err != nil {
		return nil, err
	}

	logger.LogAttrs(context.Background(), slog.LevelInfo, "Connected to Redis success!")

	return &Database{Client: rClient}, nil
}
