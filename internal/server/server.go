package server

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Configs struct {
	Name string `json:"name"`
	Env  string `json:"env"`
}

type Health struct {
	DBStatus string `json:"dbStatus"`
	Status   string `json:"status"`
}

type Server struct {
	DB          *Database
	Logger      *slog.Logger
	ShutDownFxn func(context.Context) error
	Health      *Health
	Mux         *http.ServeMux
	*http.Server
	*Configs
}

type Opts func(s *Server)

func NewServer(opts ...Opts) (*Server, error) {
	s := defaultServer()

	s.Logger = newLogger()

	db, err := newDB(s.Logger)
	if err != nil {
		return nil, err
	}

	s.DB = db

	for _, fn := range opts {
		fn(s)
	}

	return s, nil
}

func WithTimeouts(read, write, idle int) Opts {
	return func(s *Server) {
		s.ReadTimeout = time.Duration(read) * time.Second
		s.WriteTimeout = time.Duration(write) * time.Second
		s.IdleTimeout = time.Duration(idle) * time.Second
	}
}

func WithPort(port string) Opts {
	return func(s *Server) {
		s.Addr = ":" + port
	}
}

func WithAppName(name string) Opts {
	return func(s *Server) {
		s.Name = name
	}
}

func WithEnv(env string) Opts {
	return func(s *Server) {
		s.Env = env
	}
}

func ServerFromEnvs() (*Server, error) {
	if err := godotenv.Load(".env"); err != nil {
		log.Print("error while loading env file")

		return nil, err
	}

	opts := loadEnvVars()

	return NewServer(opts...)
}

func defaultServer() *Server {
	return &Server{
		Mux: http.NewServeMux(),
		Server: &http.Server{
			Addr:         ":9001",
			ReadTimeout:  time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  20 * time.Second,
		},
		Configs: &Configs{
			Name: "todoApp",
			Env:  "dev",
		},
	}
}

func loadEnvVars() []Opts {
	var opts []Opts

	appName := os.Getenv("APP_NAME")
	if appName != "" {
		opts = append(opts, WithAppName(appName))
	}

	port := os.Getenv("HTTP_PORT")
	if port != "" {
		opts = append(opts, WithPort(port))
	}

	env := os.Getenv("ENV")
	if env != "" {
		opts = append(opts, WithEnv(env))
	}

	readTimeout := getEnvAsInt("READ_TIMEOUT", 10)   // Default to 10 second
	writeTimeout := getEnvAsInt("WRITE_TIMEOUT", 20) // Default to 20 second
	idleTimeout := getEnvAsInt("IDLE_TIMEOUT", 30)   // Default to 30 second

	opts = append(opts, WithTimeouts(readTimeout, writeTimeout, idleTimeout))
	return opts
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
