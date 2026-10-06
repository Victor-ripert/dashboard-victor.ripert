package auth

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// Run in order at every start-up; each statement does nothing if already applied.
var migrations = []string{
	`CREATE TABLE IF NOT EXISTS users (
		id            BIGSERIAL PRIMARY KEY,
		email         TEXT NOT NULL UNIQUE,
		password_hash TEXT NOT NULL,
		created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
	)`,
	// Accounts created before email verification existed count as verified (DEFAULT TRUE
	// fills the existing rows), then new accounts start unverified.
	`ALTER TABLE users ADD COLUMN IF NOT EXISTS email_verified BOOLEAN NOT NULL DEFAULT TRUE`,
	`ALTER TABLE users ALTER COLUMN email_verified SET DEFAULT FALSE`,
	// One code per user at most; deleted with its user
	`CREATE TABLE IF NOT EXISTS verification_codes (
		user_id    BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
		code_hash  TEXT NOT NULL,
		expires_at TIMESTAMPTZ NOT NULL,
		attempts   INT NOT NULL DEFAULT 0,
		sent_at    TIMESTAMPTZ NOT NULL
	)`,
}

const uniqueViolation = "23505"

type PostgresStore struct {
	db *sql.DB
}

// NewPostgresStore connects to the database and creates or updates the tables if needed.
func NewPostgresStore(ctx context.Context, databaseURL string) (*PostgresStore, error) {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, err
	}
	for _, statement := range migrations {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			db.Close()
			return nil, err
		}
	}
	return &PostgresStore{db: db}, nil
}

func (s *PostgresStore) CreateUser(ctx context.Context, email, passwordHash string) (User, error) {
	user := User{Email: email, PasswordHash: passwordHash}
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO users (email, password_hash, email_verified) VALUES ($1, $2, FALSE) RETURNING id`,
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
		`SELECT id, email, password_hash, email_verified FROM users WHERE email = $1`,
		email,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.EmailVerified)

	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrUserNotFound
	}
	if err != nil {
		return User{}, err
	}
	return user, nil
}

func (s *PostgresStore) DeleteUser(ctx context.Context, userID int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, userID)
	return err
}

func (s *PostgresStore) MarkEmailVerified(ctx context.Context, userID int64) error {
	// Both changes happen together or not at all
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() // does nothing once committed

	if _, err := tx.ExecContext(ctx, `UPDATE users SET email_verified = TRUE WHERE id = $1`, userID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM verification_codes WHERE user_id = $1`, userID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *PostgresStore) DeleteAbandonedUsers(ctx context.Context, filter AbandonedFilter) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM users u
		WHERE NOT u.email_verified
		  AND (u.created_at <= $1
		       OR (u.created_at <= $2 AND NOT EXISTS (
		             SELECT 1 FROM verification_codes c
		             WHERE c.user_id = u.id AND c.expires_at > $3 AND c.attempts < $4)))`,
		filter.CreatedBefore, filter.NoCodeCreatedBefore, filter.Now, filter.MaxAttempts,
	)
	return err
}

func (s *PostgresStore) SaveVerificationCode(ctx context.Context, code VerificationCode) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO verification_codes (user_id, code_hash, expires_at, attempts, sent_at)
		VALUES ($1, $2, $3, 0, $4)
		ON CONFLICT (user_id) DO UPDATE
		SET code_hash = EXCLUDED.code_hash, expires_at = EXCLUDED.expires_at,
		    attempts = 0, sent_at = EXCLUDED.sent_at`,
		code.UserID, code.CodeHash, code.ExpiresAt, code.SentAt,
	)
	return err
}

func (s *PostgresStore) FindVerificationCode(ctx context.Context, userID int64) (VerificationCode, error) {
	return scanCode(s.db.QueryRowContext(ctx,
		`SELECT user_id, code_hash, expires_at, attempts, sent_at FROM verification_codes WHERE user_id = $1`,
		userID,
	))
}

func (s *PostgresStore) ConsumeVerificationAttempt(ctx context.Context, userID int64, maxAttempts int, now time.Time) (VerificationCode, error) {
	// A single UPDATE: two parallel requests can't both use the last attempt
	return scanCode(s.db.QueryRowContext(ctx, `
		UPDATE verification_codes SET attempts = attempts + 1
		WHERE user_id = $1 AND attempts < $2 AND expires_at > $3
		RETURNING user_id, code_hash, expires_at, attempts, sent_at`,
		userID, maxAttempts, now,
	))
}

func scanCode(row *sql.Row) (VerificationCode, error) {
	var code VerificationCode
	err := row.Scan(&code.UserID, &code.CodeHash, &code.ExpiresAt, &code.Attempts, &code.SentAt)
	if errors.Is(err, sql.ErrNoRows) {
		return VerificationCode{}, ErrCodeNotFound
	}
	if err != nil {
		return VerificationCode{}, err
	}
	return code, nil
}
