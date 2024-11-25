package store

import "auth-rest-api/internal/server"

type Store struct {
	DB *server.Database
}

func New(db *server.Database) *Store {
	return &Store{DB: db}
}
