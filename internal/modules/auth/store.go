package auth

import (
	"context"
	"errors"
	"net/netip"
	"time"
)

// User statuses (users.status).
const (
	StatusActive   = "ACTIVE"
	StatusDisabled = "DISABLED"
)

// Session revocation reasons (user_sessions.revoked_reason).
const (
	RevokedLogout          = "LOGOUT"
	RevokedSuperseded      = "SUPERSEDED"
	RevokedUserDisabled    = "USER_DISABLED"
	RevokedPasswordChanged = "PASSWORD_CHANGED"
	RevokedAdmin           = "ADMIN_REVOKED"
)

// ErrNotFound is returned when a user or session does not exist.
var ErrNotFound = errors.New("not found")

// ErrUsernameTaken is returned when creating a user whose username or e-mail exists.
var ErrUsernameTaken = errors.New("username or e-mail already in use")

// User is an account as stored. PasswordHash never leaves the auth package's API.
type User struct {
	ID           string
	Username     string
	DisplayName  string
	Email        *string
	PasswordHash string
	Status       string
	LastLoginAt  *time.Time
	CreatedAt    time.Time
}

// NewSession is what Login asks the store to persist.
type NewSession struct {
	UserID    string
	TokenHash []byte
	CSRFToken string
	CreatedAt time.Time
	ExpiresAt time.Time
	IP        *netip.Addr
	UserAgent string
}

// SessionRecord is a stored session joined with its user.
type SessionRecord struct {
	ID         string
	CSRFToken  string
	CreatedAt  time.Time
	LastSeenAt time.Time
	ExpiresAt  time.Time
	RevokedAt  *time.Time
	User       User
}

// Store is the data-access boundary for authentication. The PostgreSQL
// implementation is PGStore; every query is parameterised.
type Store interface {
	FindUserByUsername(ctx context.Context, username string) (User, error)
	// CreateSession inserts the session and records the user's last login in one transaction.
	CreateSession(ctx context.Context, s NewSession) (string, error)
	FindSessionByTokenHash(ctx context.Context, tokenHash []byte) (SessionRecord, error)
	TouchSession(ctx context.Context, sessionID string, at time.Time) error
	// RevokeSession revokes one active session; revoking an already revoked session is a no-op.
	RevokeSession(ctx context.Context, sessionID, reason string, at time.Time) error
	RevokeSessionByTokenHash(ctx context.Context, tokenHash []byte, reason string, at time.Time) error
}
