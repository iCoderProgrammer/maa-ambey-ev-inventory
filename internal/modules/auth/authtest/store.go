// Package authtest provides an in-memory auth.Store for tests. It is never
// wired into the API; production uses auth.PGStore.
package authtest

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/audit"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/auth"
)

// Store is a concurrency-safe in-memory auth.Store.
type Store struct {
	mu       sync.Mutex
	users    map[string]auth.User // by username
	sessions []*session
	nextID   int
	// FailNext makes the next call return this error (then clears it).
	FailNext error
	// Events holds every audit event written, in order.
	Events []audit.Event
}

type session struct {
	auth.SessionRecord
	userID    string
	tokenHash []byte
	reason    string
}

// NewStore returns an empty store.
func NewStore() *Store { return &Store{users: map[string]auth.User{}} }

// AddUser stores a user with the given password, hashed like production.
func (s *Store) AddUser(username, displayName, password, status string) auth.User {
	hash, err := auth.HashPassword(password)
	if err != nil {
		panic(err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextID++
	u := auth.User{
		ID: fmt.Sprintf("user-%d", s.nextID), Username: username, DisplayName: displayName,
		PasswordHash: hash, Status: status, CreatedAt: time.Now(),
	}
	s.users[username] = u
	return u
}

// SetStatus changes a user's status without touching sessions.
func (s *Store) SetStatus(username, status string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u := s.users[username]
	u.Status = status
	s.users[username] = u
}

// RevokedReasons lists the revocation reason of every session, in creation order ("" = active).
func (s *Store) RevokedReasons() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, len(s.sessions))
	for i, ss := range s.sessions {
		out[i] = ss.reason
	}
	return out
}

func (s *Store) fail() error {
	err := s.FailNext
	s.FailNext = nil
	return err
}

// FindUserByUsername implements auth.Store.
func (s *Store) FindUserByUsername(_ context.Context, username string) (auth.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.fail(); err != nil {
		return auth.User{}, err
	}
	u, ok := s.users[username]
	if !ok {
		return auth.User{}, auth.ErrNotFound
	}
	return u, nil
}

// CreateSession implements auth.Store.
func (s *Store) CreateSession(_ context.Context, ns auth.NewSession, ev audit.Event) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.fail(); err != nil {
		return "", err
	}
	s.nextID++
	id := fmt.Sprintf("session-%d", s.nextID)
	s.sessions = append(s.sessions, &session{
		SessionRecord: auth.SessionRecord{
			ID: id, CSRFToken: ns.CSRFToken, CreatedAt: ns.CreatedAt, LastSeenAt: ns.CreatedAt, ExpiresAt: ns.ExpiresAt,
		},
		userID: ns.UserID, tokenHash: ns.TokenHash,
	})
	ev.Summary = audit.Summary{"sessionId": id}
	s.Events = append(s.Events, ev)
	return id, nil
}

// RecordAudit implements auth.Store.
func (s *Store) RecordAudit(_ context.Context, ev audit.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Events = append(s.Events, ev)
	return nil
}

// Actions lists the audited actions in order.
func (s *Store) Actions() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, len(s.Events))
	for i, e := range s.Events {
		out[i] = e.Action
	}
	return out
}

// FindSessionByTokenHash implements auth.Store.
func (s *Store) FindSessionByTokenHash(_ context.Context, tokenHash []byte) (auth.SessionRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.fail(); err != nil {
		return auth.SessionRecord{}, err
	}
	for _, ss := range s.sessions {
		if bytes.Equal(ss.tokenHash, tokenHash) {
			rec := ss.SessionRecord
			for _, u := range s.users {
				if u.ID == ss.userID {
					rec.User = u
				}
			}
			return rec, nil
		}
	}
	return auth.SessionRecord{}, auth.ErrNotFound
}

// TouchSession implements auth.Store.
func (s *Store) TouchSession(_ context.Context, id string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, ss := range s.sessions {
		if ss.ID == id && ss.RevokedAt == nil && ss.LastSeenAt.Before(at) {
			ss.LastSeenAt = at
		}
	}
	return nil
}

// RevokeSession implements auth.Store.
func (s *Store) RevokeSession(_ context.Context, id, reason string, at time.Time, ev audit.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.fail(); err != nil {
		return err
	}
	for _, ss := range s.sessions {
		if ss.ID == id && ss.RevokedAt == nil {
			revoke(ss, reason, at)
			s.Events = append(s.Events, ev)
		}
	}
	return nil
}

// RevokeSessionByTokenHash implements auth.Store.
func (s *Store) RevokeSessionByTokenHash(_ context.Context, tokenHash []byte, reason string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, ss := range s.sessions {
		if bytes.Equal(ss.tokenHash, tokenHash) {
			revoke(ss, reason, at)
		}
	}
	return nil
}

func revoke(ss *session, reason string, at time.Time) {
	if ss.RevokedAt == nil {
		ss.RevokedAt = &at
		ss.reason = reason
	}
}

var _ auth.Store = (*Store)(nil)
