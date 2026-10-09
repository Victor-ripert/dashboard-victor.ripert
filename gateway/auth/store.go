package auth

import (
	"context"
	"errors"
	"time"
)

var (
	ErrEmailTaken   = errors.New("email already registered")
	ErrUserNotFound = errors.New("user not found")
	ErrCodeNotFound = errors.New("verification code not found")
)

type User struct {
	ID            int64  `json:"id"`
	Email         string `json:"email"`
	PasswordHash  string `json:"-"`
	EmailVerified bool   `json:"-"`
}

type VerificationCode struct {
	UserID    int64
	CodeHash  string
	ExpiresAt time.Time
	Attempts  int // wrong or right guesses already made
	SentAt    time.Time
}

// AbandonedFilter selects the unverified accounts that DeleteAbandonedUsers removes:
//   - those created before CreatedBefore, whatever their code;
//   - those created before NoCodeCreatedBefore that have no usable code at Now
//     (no code, expired code, or MaxAttempts guesses already made).
type AbandonedFilter struct {
	Now                 time.Time
	CreatedBefore       time.Time
	NoCodeCreatedBefore time.Time
	MaxAttempts         int
}

type UserStore interface {
	// CreateUser creates an unverified user, or returns ErrEmailTaken if the
	// email is used by any account (verified or not).
	CreateUser(ctx context.Context, email, passwordHash string) (User, error)
	FindUserByEmail(ctx context.Context, email string) (User, error)
	DeleteUser(ctx context.Context, userID int64) error
	// MarkEmailVerified verifies the user and deletes their verification code.
	MarkEmailVerified(ctx context.Context, userID int64) error
	DeleteAbandonedUsers(ctx context.Context, filter AbandonedFilter) error
}

type CodeStore interface {
	// SaveVerificationCode stores the user's code, replacing any previous one.
	SaveVerificationCode(ctx context.Context, code VerificationCode) error
	FindVerificationCode(ctx context.Context, userID int64) (VerificationCode, error)
	// ConsumeVerificationAttempt counts one guess and returns the code, or
	// returns ErrCodeNotFound if there is no code, it expired at now, or it
	// already received maxAttempts guesses. Counting before comparing means
	// parallel requests can't make more than maxAttempts guesses.
	ConsumeVerificationAttempt(ctx context.Context, userID int64, maxAttempts int, now time.Time) (VerificationCode, error)
}

// Store is all the auth code knows about the database.
// Implement it to swap PostgreSQL for anything else.
type Store interface {
	UserStore
	CodeStore
}
