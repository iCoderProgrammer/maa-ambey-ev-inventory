package audit

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

type captureExec struct {
	sql  string
	args []any
}

func (c *captureExec) Exec(_ context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	c.sql, c.args = sql, args
	return pgconn.NewCommandTag("INSERT 0 1"), nil
}

func valid() Event {
	return Event{
		Actor:      UserActor("5b0f8a52-3c1e-4d6f-9a8b-0c1d2e3f4a5b", "asha"),
		Action:     "user.disabled",
		EntityType: "user",
		EntityID:   "u-1",
		RequestID:  "req-1",
		Summary:    Summary{"status": Change{From: "ACTIVE", To: "DISABLED"}, "sessionsEnded": 2},
	}
}

func TestRecordValidEvent(t *testing.T) {
	var db captureExec
	if err := Record(context.Background(), &db, valid()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(db.sql, "INSERT INTO audit_events") {
		t.Fatalf("sql = %s", db.sql)
	}
	if db.args[5] != OutcomeSuccess {
		t.Errorf("default outcome = %v", db.args[5])
	}
	if got := string(db.args[10].([]byte)); got != `{"sessionsEnded":2,"status":{"from":"ACTIVE","to":"DISABLED"}}` {
		t.Errorf("summary = %s", got)
	}
}

func TestRecordRejectsInvalidEvents(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Event)
		want   string
	}{
		{"user actor without id", func(e *Event) { e.Actor.UserID = "" }, "needs a user ID"},
		{"unknown actor type", func(e *Event) { e.Actor.Type = "ROBOT" }, "unknown actor type"},
		{"blank label", func(e *Event) { e.Actor.Label = " " }, "actor label"},
		{"undotted action", func(e *Event) { e.Action = "disable" }, "dotted"},
		{"upper-case action", func(e *Event) { e.Action = "User.Disabled" }, "dotted"},
		{"bad outcome", func(e *Event) { e.Outcome = "MAYBE" }, "unknown outcome"},
		{"bad entity type", func(e *Event) { e.EntityType = "User Account" }, "entity type"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := valid()
			tt.mutate(&e)
			var db captureExec
			err := Record(context.Background(), &db, e)
			if !errors.Is(err, ErrInvalidEvent) || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("err = %v, want %q", err, tt.want)
			}
			if db.sql != "" {
				t.Error("an invalid event must not be written")
			}
		})
	}
}

// Sensitive values must never reach the audit table.
func TestRecordRejectsSecretsInSummary(t *testing.T) {
	tests := map[string]Summary{
		"password key":       {"password": "hunter2hunter2"},
		"new password key":   {"newPassword": "x"},
		"password hash key":  {"passwordHash": "x"},
		"token key":          {"sessionToken": "abc"},
		"csrf key":           {"csrfToken": "abc"},
		"cookie key":         {"Cookie": "maa_session=abc"},
		"secret key":         {"clientSecret": "abc"},
		"api key":            {"apiKey": "abc"},
		"otp":                {"otp": "123456"},
		"nested secret":      {"changes": map[string]any{"password": Change{From: "a", To: "b"}}},
		"hash value":         {"note": "$argon2id$v=19$m=19456,t=2,p=1$c2FsdA$aGFzaA"},
		"hash inside change": {"value": Change{From: nil, To: "$argon2id$v=19$m=1$x$y"}},
		"oversized text":     {"note": strings.Repeat("x", 501)},
		"odd key":            {"not a key": 1},
		"unsupported value":  {"raw": struct{}{}},
		"deep nesting":       {"a": map[string]any{"b": map[string]any{"c": map[string]any{"d": 1}}}},
	}
	for name, summary := range tests {
		t.Run(name, func(t *testing.T) {
			e := valid()
			e.Summary = summary
			var db captureExec
			if err := Record(context.Background(), &db, e); !errors.Is(err, ErrInvalidEvent) {
				t.Fatalf("err = %v, want rejection", err)
			}
			if db.sql != "" {
				t.Error("an unsafe summary must not be written")
			}
		})
	}
}

func TestRecordRejectsOversizedSummary(t *testing.T) {
	big := Summary{}
	for i := range 40 {
		big[strings.Repeat("k", 10)+string(rune('a'+i%26))+string(rune('a'+i/26))] = strings.Repeat("v", 400)
	}
	e := valid()
	e.Summary = big
	if err := Record(context.Background(), &captureExec{}, e); !errors.Is(err, ErrInvalidEvent) {
		t.Fatalf("err = %v", err)
	}
}
