package auth

import (
	"context"
	"errors"
)

var (
	ErrEmailTaken   = errors.New("email already registered")
	ErrUserNotFound = errors.New("user not found")
)

type User struct {
	ID           int64  `json:"id"`
	Email        string `json:"email"`
	PasswordHash string `json:"-"`
}

// Store is all the auth code knows about the database.
// Implement it to swap PostgreSQL for anything else.
type Store interface {
	CreateUser(ctx context.Context, email, passwordHash string) (User, error)
	FindUserByEmail(ctx context.Context, email string) (User, error)
}
