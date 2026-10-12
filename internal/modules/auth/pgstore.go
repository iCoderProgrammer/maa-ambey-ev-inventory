package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PGStore implements Store and the user administration used by cmd/user.
type PGStore struct {
	pool *pgxpool.Pool
}

// NewPGStore wraps a pgx pool.
func NewPGStore(pool *pgxpool.Pool) *PGStore { return &PGStore{pool: pool} }

const userColumns = `id::text, username, display_name, email, password_hash, status, last_login_at, created_at`

func scanUser(row pgx.Row, u *User) error {
	return row.Scan(&u.ID, &u.Username, &u.DisplayName, &u.Email, &u.PasswordHash, &u.Status, &u.LastLoginAt, &u.CreatedAt)
}

// FindUserByUsername looks up a user by normalised username.
func (s *PGStore) FindUserByUsername(ctx context.Context, username string) (User, error) {
	var u User
	err := scanUser(s.pool.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE username = $1`, username), &u)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("find user: %w", err)
	}
	return u, nil
}

// CreateSession implements Store.
func (s *PGStore) CreateSession(ctx context.Context, ns NewSession) (string, error) {
	var id string
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `
			INSERT INTO user_sessions (user_id, token_hash, csrf_token, created_at, last_seen_at, expires_at, ip_address, user_agent)
			VALUES ($1, $2, $3, $4, $4, $5, $6, NULLIF($7, ''))
			RETURNING id::text`,
			ns.UserID, ns.TokenHash, ns.CSRFToken, ns.CreatedAt, ns.ExpiresAt, ns.IP, ns.UserAgent,
		).Scan(&id)
		if err != nil {
			return fmt.Errorf("insert session: %w", err)
		}
		if _, err := tx.Exec(ctx, `UPDATE users SET last_login_at = $2 WHERE id = $1`, ns.UserID, ns.CreatedAt); err != nil {
			return fmt.Errorf("record last login: %w", err)
		}
		return nil
	})
	return id, err
}

// FindSessionByTokenHash implements Store.
func (s *PGStore) FindSessionByTokenHash(ctx context.Context, tokenHash []byte) (SessionRecord, error) {
	var r SessionRecord
	u := &r.User
	err := s.pool.QueryRow(ctx, `
		SELECT s.id::text, s.csrf_token, s.created_at, s.last_seen_at, s.expires_at, s.revoked_at,
		       u.id::text, u.username, u.display_name, u.email, u.status
		FROM user_sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.token_hash = $1`, tokenHash,
	).Scan(&r.ID, &r.CSRFToken, &r.CreatedAt, &r.LastSeenAt, &r.ExpiresAt, &r.RevokedAt,
		&u.ID, &u.Username, &u.DisplayName, &u.Email, &u.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return SessionRecord{}, ErrNotFound
	}
	if err != nil {
		return SessionRecord{}, fmt.Errorf("find session: %w", err)
	}
	return r, nil
}

// TouchSession implements Store. Only moves last_seen_at forward.
func (s *PGStore) TouchSession(ctx context.Context, sessionID string, at time.Time) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE user_sessions SET last_seen_at = $2
		WHERE id = $1 AND revoked_at IS NULL AND last_seen_at < $2`, sessionID, at)
	if err != nil {
		return fmt.Errorf("touch session: %w", err)
	}
	return nil
}

// RevokeSession implements Store.
func (s *PGStore) RevokeSession(ctx context.Context, sessionID, reason string, at time.Time) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE user_sessions SET revoked_at = $2, revoked_reason = $3
		WHERE id = $1 AND revoked_at IS NULL`, sessionID, at, reason)
	if err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	return nil
}

// RevokeSessionByTokenHash implements Store.
func (s *PGStore) RevokeSessionByTokenHash(ctx context.Context, tokenHash []byte, reason string, at time.Time) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE user_sessions SET revoked_at = $2, revoked_reason = $3
		WHERE token_hash = $1 AND revoked_at IS NULL`, tokenHash, at, reason)
	if err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	return nil
}

// ── User administration (cmd/user) ────────────────────────────────────────

// CreateUser inserts an ACTIVE user. Inputs must already be validated.
func (s *PGStore) CreateUser(ctx context.Context, username, displayName string, email *string, passwordHash string) (User, error) {
	var u User
	err := scanUser(s.pool.QueryRow(ctx, `
		INSERT INTO users (username, display_name, email, password_hash)
		VALUES ($1, $2, $3, $4)
		RETURNING `+userColumns, username, displayName, email, passwordHash), &u)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return User{}, ErrUsernameTaken
	}
	if err != nil {
		return User{}, fmt.Errorf("create user: %w", err)
	}
	return u, nil
}

// ListUsers returns all users ordered by username.
func (s *PGStore) ListUsers(ctx context.Context) ([]User, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+userColumns+` FROM users ORDER BY username`)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()
	var users []User
	for rows.Next() {
		var u User
		if err := scanUser(rows, &u); err != nil {
			return nil, fmt.Errorf("list users: %w", err)
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// SetStatus changes a user's status. Disabling also revokes every active
// session in the same transaction. Returns the number of sessions revoked.
func (s *PGStore) SetStatus(ctx context.Context, username, status string, at time.Time) (int64, error) {
	return s.updateUserAndRevoke(ctx, username, RevokedUserDisabled, at, status == StatusDisabled,
		`UPDATE users SET status = $2, version = version + 1 WHERE username = $1 RETURNING id::text`, status)
}

// SetPassword replaces the password hash and revokes every active session.
func (s *PGStore) SetPassword(ctx context.Context, username, passwordHash string, at time.Time) (int64, error) {
	return s.updateUserAndRevoke(ctx, username, RevokedPasswordChanged, at, true,
		`UPDATE users SET password_hash = $2, password_changed_at = $3, version = version + 1 WHERE username = $1 RETURNING id::text`,
		passwordHash, at)
}

// RevokeAllSessions revokes every active session of a user.
func (s *PGStore) RevokeAllSessions(ctx context.Context, username string, at time.Time) (int64, error) {
	return s.updateUserAndRevoke(ctx, username, RevokedAdmin, at, true,
		`SELECT id::text FROM users WHERE username = $1`)
}

func (s *PGStore) updateUserAndRevoke(ctx context.Context, username, reason string, at time.Time, revoke bool, sql string, args ...any) (int64, error) {
	var revoked int64
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		var userID string
		err := tx.QueryRow(ctx, sql, append([]any{username}, args...)...).Scan(&userID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("update user: %w", err)
		}
		if !revoke {
			return nil
		}
		tag, err := tx.Exec(ctx, `
			UPDATE user_sessions SET revoked_at = $2, revoked_reason = $3
			WHERE user_id = $1 AND revoked_at IS NULL`, userID, at, reason)
		if err != nil {
			return fmt.Errorf("revoke sessions: %w", err)
		}
		revoked = tag.RowsAffected()
		return nil
	})
	return revoked, err
}
