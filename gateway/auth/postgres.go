package auth

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
)

const schema = `
CREATE TABLE IF NOT EXISTS users (
	id            BIGSERIAL PRIMARY KEY,
	email         TEXT NOT NULL UNIQUE,
	password_hash TEXT NOT NULL,
	created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
)`

const uniqueViolation = "23505"

type PostgresStore struct {
	db *sql.DB
}

// NewPostgresStore connects to the database and creates the users table if needed.
func NewPostgresStore(ctx context.Context, databaseURL string) (*PostgresStore, error) {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, err
	}
	if _, err := db.ExecContext(ctx, schema); err != nil {
		db.Close()
		return nil, err
	}
	return &PostgresStore{db: db}, nil
}

func (s *PostgresStore) CreateUser(ctx context.Context, email, passwordHash string) (User, error) {
	user := User{Email: email, PasswordHash: passwordHash}
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING id`,
		email, passwordHash,
	).Scan(&user.ID)

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
		return User{}, ErrEmailTaken
	}
	if err != nil {
		return User{}, err
	}
	return user, nil
}

func (s *PostgresStore) FindUserByEmail(ctx context.Context, email string) (User, error) {
	var user User
	err := s.db.QueryRowContext(ctx,
		`SELECT id, email, password_hash FROM users WHERE email = $1`,
		email,
	).Scan(&user.ID, &user.Email, &user.PasswordHash)

	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrUserNotFound
	}
	if err != nil {
		return User{}, err
	}
	return user, nil
}
