// Package audit records append-only audit events (B-08, FR-017, SEC-004).
//
// Usage for a sensitive mutation: open a transaction, make the change, call
// Record with the same transaction, commit. The event then exists if and only
// if the change does. audit_events rejects UPDATE, DELETE and TRUNCATE at the
// database level, so history cannot be edited through the application.
//
//	err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
//	    if _, err := tx.Exec(ctx, `UPDATE users SET status = 'DISABLED' WHERE id = $1`, id); err != nil {
//	        return err
//	    }
//	    return audit.Record(ctx, tx, audit.Event{
//	        Actor: actor, Action: "user.disabled", EntityType: "user", EntityID: id,
//	        Summary: audit.Summary{"status": audit.Change{From: "ACTIVE", To: "DISABLED"}},
//	    })
//	})
//
// Summaries must never carry secrets: Record rejects keys such as "password",
// "token", "hash", "secret", "csrf" or "cookie" and values that look like
// password hashes, instead of silently storing them. Keep personal data out
// too: reference people by ID, not by phone, address or identity numbers.
package audit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgconn"
)

// Actor types (audit_events.actor_type).
const (
	ActorUser   = "USER"
	ActorCLI    = "CLI"
	ActorSystem = "SYSTEM"
	// ActorAnonymous is an unauthenticated requester, e.g. a failed sign-in: the
	// targeted account is the entity, not the actor.
	ActorAnonymous = "ANONYMOUS"
)

// Outcomes (audit_events.outcome).
const (
	OutcomeSuccess = "SUCCESS"
	OutcomeDenied  = "DENIED"
	OutcomeFailure = "FAILURE"
)

// ErrInvalidEvent wraps every validation failure, including unsafe summaries.
var ErrInvalidEvent = errors.New("invalid audit event")

// Actor is who performed the action.
type Actor struct {
	Type   string
	UserID string // required for ActorUser
	Label  string // username, or command name for ActorCLI/ActorSystem
}

// UserActor is a signed-in staff member.
func UserActor(userID, username string) Actor {
	return Actor{Type: ActorUser, UserID: userID, Label: username}
}

// CLIActor is an operator running a command such as cmd/user.
func CLIActor(label string) Actor { return Actor{Type: ActorCLI, Label: label} }

// AnonymousActor is a request without a signed-in user (label describes it,
// e.g. "sign-in attempt"). Never put typed usernames or other input here.
func AnonymousActor(label string) Actor { return Actor{Type: ActorAnonymous, Label: label} }

// Summary is the safe, structured description of what changed.
type Summary map[string]any

// Change is a before/after pair inside a Summary.
type Change struct {
	From any `json:"from"`
	To   any `json:"to"`
}

// Event is one audit record.
type Event struct {
	Actor      Actor
	Action     string // dotted past tense: "user.disabled", "auth.signed_in"
	Outcome    string // default OutcomeSuccess
	EntityType string // "user", "session", ...
	EntityID   string
	Reference  string
	RequestID  string
	Summary    Summary
	// OccurredAt defaults to the database clock.
	OccurredAt time.Time
}

// Execer is satisfied by pgx.Tx, *pgx.Conn and *pgxpool.Pool. Pass the
// transaction that makes the audited change.
type Execer interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// Record validates the event and appends it.
func Record(ctx context.Context, db Execer, e Event) error {
	summary, err := e.prepare()
	if err != nil {
		return err
	}
	var occurred any
	if !e.OccurredAt.IsZero() {
		occurred = e.OccurredAt.UTC()
	}
	_, err = db.Exec(ctx, `
		INSERT INTO audit_events (occurred_at, actor_type, actor_user_id, actor_label, action, outcome,
		                          entity_type, entity_id, reference, request_id, summary)
		VALUES (COALESCE($1, now()), $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		occurred, e.Actor.Type, nullable(e.Actor.UserID), e.Actor.Label, e.Action, e.Outcome,
		e.EntityType, nullable(e.EntityID), nullable(e.Reference), nullable(e.RequestID), summary)
	if err != nil {
		return fmt.Errorf("record audit event %s: %w", e.Action, err)
	}
	return nil
}

var (
	actionPattern     = regexp.MustCompile(`^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+$`)
	entityTypePattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	keyPattern        = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]{0,63}$`)
	// Key fragments that must never appear in a summary.
	sensitiveKeyParts = []string{"password", "passwd", "passphrase", "secret", "token", "hash", "csrf",
		"cookie", "authorization", "apikey", "api_key", "privatekey", "private_key", "otp", "credential"}
)

const (
	maxSummaryBytes = 8 << 10
	maxStringRunes  = 500
	maxDepth        = 3
)

// prepare validates the event, fills defaults and returns the JSON summary.
func (e *Event) prepare() ([]byte, error) {
	if e.Outcome == "" {
		e.Outcome = OutcomeSuccess
	}
	invalid := func(format string, args ...any) error {
		return fmt.Errorf("%w: %s", ErrInvalidEvent, fmt.Sprintf(format, args...))
	}
	switch e.Actor.Type {
	case ActorUser:
		if e.Actor.UserID == "" {
			return nil, invalid("user actor needs a user ID")
		}
	case ActorAnonymous:
		if e.Actor.UserID != "" {
			return nil, invalid("anonymous actor must not carry a user ID")
		}
	case ActorCLI, ActorSystem:
	default:
		return nil, invalid("unknown actor type %q", e.Actor.Type)
	}
	if l := strings.TrimSpace(e.Actor.Label); l == "" || utf8.RuneCountInString(l) > 120 {
		return nil, invalid("actor label must be 1-120 characters")
	}
	if !actionPattern.MatchString(e.Action) || len(e.Action) > 100 {
		return nil, invalid("action %q must be a dotted lower-case key such as user.disabled", e.Action)
	}
	switch e.Outcome {
	case OutcomeSuccess, OutcomeDenied, OutcomeFailure:
	default:
		return nil, invalid("unknown outcome %q", e.Outcome)
	}
	if !entityTypePattern.MatchString(e.EntityType) || len(e.EntityType) > 60 {
		return nil, invalid("entity type %q must be a lower-case identifier", e.EntityType)
	}
	if len(e.EntityID) > 100 || len(e.Reference) > 120 || len(e.RequestID) > 100 {
		return nil, invalid("entity ID, reference or request ID too long")
	}
	if e.Summary == nil {
		return []byte("{}"), nil
	}
	if err := checkValue("summary", map[string]any(e.Summary), 0); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidEvent, err)
	}
	b, err := json.Marshal(e.Summary)
	if err != nil {
		return nil, invalid("summary is not JSON-serialisable: %v", err)
	}
	if len(b) > maxSummaryBytes {
		return nil, invalid("summary is %d bytes; the limit is %d", len(b), maxSummaryBytes)
	}
	return b, nil
}

// checkValue walks the summary, rejecting secret-looking keys and values and
// anything that is not plain JSON data.
func checkValue(path string, v any, depth int) error {
	if depth > maxDepth {
		return fmt.Errorf("%s: nested too deeply", path)
	}
	switch x := v.(type) {
	case nil, bool, int, int32, int64, uint32, float64:
		return nil
	case string:
		if utf8.RuneCountInString(x) > maxStringRunes {
			return fmt.Errorf("%s: text longer than %d characters", path, maxStringRunes)
		}
		if strings.HasPrefix(x, "$argon2") || strings.HasPrefix(x, "$2a$") || strings.HasPrefix(x, "$2b$") {
			return fmt.Errorf("%s: looks like a password hash", path)
		}
		return nil
	case time.Time:
		return nil
	case Change:
		if err := checkValue(path+".from", x.From, depth+1); err != nil {
			return err
		}
		return checkValue(path+".to", x.To, depth+1)
	case Summary:
		return checkValue(path, map[string]any(x), depth)
	case map[string]any:
		for k, val := range x {
			if !keyPattern.MatchString(k) {
				return fmt.Errorf("%s: key %q must be a simple identifier", path, k)
			}
			lower := strings.ToLower(k)
			for _, part := range sensitiveKeyParts {
				if strings.Contains(lower, part) {
					return fmt.Errorf("%s: key %q may hold a secret and is not allowed in audit summaries", path, k)
				}
			}
			if err := checkValue(path+"."+k, val, depth+1); err != nil {
				return err
			}
		}
		return nil
	case []string:
		for i, s := range x {
			if err := checkValue(fmt.Sprintf("%s[%d]", path, i), s, depth+1); err != nil {
				return err
			}
		}
		return nil
	case []any:
		for i, item := range x {
			if err := checkValue(fmt.Sprintf("%s[%d]", path, i), item, depth+1); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("%s: unsupported value type %T", path, v)
	}
}

func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}
