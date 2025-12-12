package server

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type Health struct {
	DBStatus   string `json:"dbStatus"`
	Status     string `json:"status"`
	StatusCode int
}

type Server struct {
	DB          *Database
	Logger      *slog.Logger
	ShutDownFxn func(context.Context) error
	Health      *Health
	Mux         *http.ServeMux
	*http.Server
}

type Builder struct {
	server *Server
}

type Opts func(s *Server)

func NewServerBuilder() *Builder {
	s := defaultServer()

	return s
}

func (sb *Builder) WithLogger() *Builder {
	sb.server.Logger = newLogger()
	return sb
}

func (sb *Builder) WithTimeouts() *Builder {
	read := getEnvAsInt("READ_TIMEOUT", 10)   // Default to 10 second
	write := getEnvAsInt("WRITE_TIMEOUT", 20) // Default to 20 second
	idle := getEnvAsInt("IDLE_TIMEOUT", 30)   // Default to 30 second

	sb.server.ReadTimeout = time.Duration(read) * time.Second
	sb.server.WriteTimeout = time.Duration(write) * time.Second
	sb.server.IdleTimeout = time.Duration(idle) * time.Second

	return sb
}

func (sb *Builder) WithHostPort() *Builder {
	port := os.Getenv("HTTP_PORT")
	host := os.Getenv("HTTP_HOST")

	sb.server.Addr = net.JoinHostPort(host, port)
	return sb
}

func (sb *Builder) Build() (*Server, error) {
	db, err := getDatabase(sb.server.Logger)
	if err != nil {
		return nil, err
	}

	sb.server.DB = db

	return sb.server, nil
}

func defaultServer() *Builder {
	return &Builder{
		server: &Server{
			Mux: http.NewServeMux(),
			Server: &http.Server{
				Addr:         "localhost:9001",
				ReadTimeout:  time.Second,
				WriteTimeout: 10 * time.Second,
				IdleTimeout:  20 * time.Second,
			},
		},
	}
}

func getEnvAsInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	if intValue, err := strconv.Atoi(value); err == nil {
		return intValue
	}

	return defaultValue
}

func getEnvOrDefault(key, defaultVal string) string {
	val := strings.TrimSpace(os.Getenv(key))

	if val == "" {
		return defaultVal
	}

	return val
}
