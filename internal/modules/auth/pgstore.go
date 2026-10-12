package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/audit"
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
func (s *PGStore) CreateSession(ctx context.Context, ns NewSession, ev audit.Event) (string, error) {
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
		return audit.Record(ctx, tx, withSummary(ev, "sessionId", id))
	})
	return id, err
}

// withSummary returns ev with one more summary field, leaving the caller's map untouched.
func withSummary(ev audit.Event, key string, value any) audit.Event {
	summary := make(audit.Summary, len(ev.Summary)+1)
	for k, v := range ev.Summary {
		summary[k] = v
	}
	summary[key] = value
	ev.Summary = summary
	return ev
}

// RecordAudit implements Store.
func (s *PGStore) RecordAudit(ctx context.Context, ev audit.Event) error {
	return audit.Record(ctx, s.pool, ev)
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
func (s *PGStore) RevokeSession(ctx context.Context, sessionID, reason string, at time.Time, ev audit.Event) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
			UPDATE user_sessions SET revoked_at = $2, revoked_reason = $3
			WHERE id = $1 AND revoked_at IS NULL`, sessionID, at, reason)
		if err != nil {
			return fmt.Errorf("revoke session: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return nil // already revoked: nothing happened, nothing to audit
		}
		return audit.Record(ctx, tx, ev)
	})
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

// CreateUser inserts an ACTIVE user and audits it. Inputs must already be validated.
func (s *PGStore) CreateUser(ctx context.Context, actor audit.Actor, username, displayName string, email *string, passwordHash string) (User, error) {
	var u User
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := scanUser(tx.QueryRow(ctx, `
			INSERT INTO users (username, display_name, email, password_hash)
			VALUES ($1, $2, $3, $4)
			RETURNING `+userColumns, username, displayName, email, passwordHash), &u); err != nil {
			return err
		}
		return audit.Record(ctx, tx, audit.Event{
			Actor: actor, Action: "user.created", EntityType: "user", EntityID: u.ID,
			Summary: audit.Summary{"username": u.Username, "status": u.Status, "emailProvided": email != nil},
		})
	})
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

// SetStatus changes a user's status and audits it. Disabling also revokes
// every active session in the same transaction. Returns the sessions revoked.
func (s *PGStore) SetStatus(ctx context.Context, actor audit.Actor, username, status string, at time.Time) (int64, error) {
	action := "user.enabled"
	if status == StatusDisabled {
		action = "user.disabled"
	}
	return s.updateUserAndRevoke(ctx, actor, action, username, RevokedUserDisabled, at, status == StatusDisabled,
		`UPDATE users u SET status = $2, version = u.version + 1
		 FROM users old WHERE u.id = old.id AND u.username = $1
		 RETURNING u.id::text, old.status`, status)
}

// SetPassword replaces the password hash, revokes every active session and
// audits the change (never the password or hash).
func (s *PGStore) SetPassword(ctx context.Context, actor audit.Actor, username, passwordHash string, at time.Time) (int64, error) {
	return s.updateUserAndRevoke(ctx, actor, "user.password_changed", username, RevokedPasswordChanged, at, true,
		`UPDATE users SET password_hash = $2, password_changed_at = $3, version = version + 1
		 WHERE username = $1 RETURNING id::text, status`,
		passwordHash, at)
}

// RevokeAllSessions revokes every active session of a user and audits it.
func (s *PGStore) RevokeAllSessions(ctx context.Context, actor audit.Actor, username string, at time.Time) (int64, error) {
	return s.updateUserAndRevoke(ctx, actor, "user.sessions_revoked", username, RevokedAdmin, at, true,
		`SELECT id::text, status FROM users WHERE username = $1`)
}

// updateUserAndRevoke runs sql (returning user id and previous status),
// optionally revokes the user's sessions and appends one audit event, all in
// one transaction.
func (s *PGStore) updateUserAndRevoke(ctx context.Context, actor audit.Actor, action, username, reason string, at time.Time,
	revoke bool, sql string, args ...any) (int64, error) {
	var revoked int64
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		var userID, previousStatus string
		err := tx.QueryRow(ctx, sql, append([]any{username}, args...)...).Scan(&userID, &previousStatus)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("update user: %w", err)
		}
		if revoke {
			tag, err := tx.Exec(ctx, `
				UPDATE user_sessions SET revoked_at = $2, revoked_reason = $3
				WHERE user_id = $1 AND revoked_at IS NULL`, userID, at, reason)
			if err != nil {
				return fmt.Errorf("revoke sessions: %w", err)
			}
			revoked = tag.RowsAffected()
		}
		summary := audit.Summary{"username": username, "sessionsEnded": revoked}
		if action == "user.enabled" || action == "user.disabled" {
			summary["status"] = audit.Change{From: previousStatus, To: args[0]}
		}
		return audit.Record(ctx, tx, audit.Event{
			Actor: actor, Action: action, EntityType: "user", EntityID: userID, Summary: summary,
		})
	})
	return revoked, err
}
