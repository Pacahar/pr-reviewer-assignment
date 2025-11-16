package postgres

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
	"github.com/pacahar/pr-reviewer-assignment/internal/storage"
)

func NewPostgresStorage(dsn string) (*storage.Storage, error) {
	const op = "storage.postgres.NewPostgresStorage"

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	teamStorage := &TeamPostgresStorage{db: db}
	userStorage := &UserPostgresStorage{db: db}
	prStorage := &PullRequestPostgresStorage{db: db}

	return &storage.Storage{
		UserStorage:        userStorage,
		TeamStorage:        teamStorage,
		PullRequestStorage: prStorage,
	}, nil
}
