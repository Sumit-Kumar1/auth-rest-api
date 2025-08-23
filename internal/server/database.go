package server

import (
	"context"
	"log/slog"
	"os"
	"sync"

	"auth-rest-api/internal/models"

	"github.com/redis/go-redis/v9"
)

// Database represents the database connection and operations.
// It wraps the Redis client and provides methods for data persistence.
type Database struct {
	Client *redis.Client
}

//nolint:gochecknoglobals // had to define global for singleton pattern
var (
	dbInstance *Database
	dbOnce     sync.Once
)

// newDB creates a new Database instance with a Redis client.
// It initializes the connection using environment variables or defaults.
// Returns an error if the connection cannot be established.
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

// getDatabase returns a singleton instance of Database.
func getDatabase(logger *slog.Logger) (*Database, error) {
	var err error

	dbOnce.Do(func() {
		dbInstance, err = newDB(logger)
	})

	return dbInstance, err
}
