package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strconv"
	"sync"
	"time"

	"auth-rest-api/internal/models"

	"github.com/redis/go-redis/v9"
)

type Database struct {
	Client *redis.Client
}

var (
	dbInstance *Database
	dbOnce     sync.Once
)

func newDB(logger *slog.Logger) (*Database, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	host := getEnvOrDefault("DB_HOST", "localhost")
	port := getEnvOrDefault("DB_PORT", "6379")
	dbIdx, err := strconv.Atoi(getEnvOrDefault("DB_NAME", "0"))
	if err != nil {
		return nil, fmt.Errorf("invalid DB_NAME: %w", err)
	}

	addr := net.JoinHostPort(host, port)
	logger.DebugContext(ctx, "redis dial",
		slog.String("addr", addr),
		slog.Int("db", dbIdx))

	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: os.Getenv("DB_PASSWORD"),
		DB:       dbIdx,
	})

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, errors.Join(models.ErrDBNotConnected, err)
	}

	logger.InfoContext(ctx, "connected to redis", slog.String("addr", addr))
	return &Database{Client: rdb}, nil
}

func getDatabase(logger *slog.Logger) (*Database, error) {
	var initErr error
	dbOnce.Do(func() { dbInstance, initErr = newDB(logger) })
	return dbInstance, initErr
}
