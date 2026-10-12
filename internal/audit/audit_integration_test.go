package audit_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/audit"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/database/dbtest"
)

// Audit rows can be added but never changed or removed through the
// application's database role: UPDATE, DELETE and TRUNCATE are refused.
func TestAuditEventsAreAppendOnly(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.MigratedPool(t)

	ev := audit.Event{
		Actor: audit.CLIActor("cmd/user:test"), Action: "user.created", EntityType: "user", EntityID: "u-1",
		RequestID: "req-1", Summary: audit.Summary{"username": "asha"},
	}
	if err := audit.Record(ctx, pool, ev); err != nil {
		t.Fatal(err)
	}

	for name, sql := range map[string]string{
		"update":   `UPDATE audit_events SET action = 'user.deleted'`,
		"delete":   `DELETE FROM audit_events`,
		"truncate": `TRUNCATE audit_events`,
	} {
		_, err := pool.Exec(ctx, sql)
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != "42501" {
			t.Errorf("%s: want SQLSTATE 42501 (append-only), got %v", name, err)
		}
	}

	var n int
	var action, summary string
	if err := pool.QueryRow(ctx, `SELECT count(*), max(action), max(summary::text) FROM audit_events`).Scan(&n, &action, &summary); err != nil {
		t.Fatal(err)
	}
	if n != 1 || action != "user.created" || summary != `{"username": "asha"}` {
		t.Errorf("row changed: n=%d action=%s summary=%s", n, action, summary)
	}
}

// The database rejects malformed rows even if a caller bypasses audit.Record.
func TestAuditEventConstraints(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.MigratedPool(t)
	for name, sql := range map[string]string{
		"anonymous actor with user": `INSERT INTO audit_events (actor_type, actor_user_id, actor_label, action, entity_type) VALUES ('ANONYMOUS', gen_random_uuid(), 'x', 'auth.sign_in_failed', 'user')`,
		"user actor without user":   `INSERT INTO audit_events (actor_type, actor_label, action, entity_type) VALUES ('USER', 'x', 'user.created', 'user')`,
		"bad actor type":            `INSERT INTO audit_events (actor_type, actor_label, action, entity_type) VALUES ('ROBOT', 'x', 'user.created', 'user')`,
		"undotted action":           `INSERT INTO audit_events (actor_type, actor_label, action, entity_type) VALUES ('CLI', 'x', 'created', 'user')`,
		"bad outcome":               `INSERT INTO audit_events (actor_type, actor_label, action, entity_type, outcome) VALUES ('CLI', 'x', 'user.created', 'user', 'OK')`,
		"summary not an object":     `INSERT INTO audit_events (actor_type, actor_label, action, entity_type, summary) VALUES ('CLI', 'x', 'user.created', 'user', '[1]')`,
	} {
		_, err := pool.Exec(ctx, sql)
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != "23514" {
			t.Errorf("%s: want check violation, got %v", name, err)
		}
	}
}

// Record joins the caller's transaction: rolling back the change rolls back its event.
func TestRecordFollowsTheCallersTransaction(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.MigratedPool(t)
	err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		if err := audit.Record(ctx, tx, audit.Event{Actor: audit.CLIActor("test"), Action: "user.created", EntityType: "user"}); err != nil {
			return err
		}
		return errors.New("business change failed")
	})
	if err == nil {
		t.Fatal("want the transaction error")
	}
	var n int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM audit_events`).Scan(&n)
	if n != 0 {
		t.Errorf("event survived a rolled-back transaction (%d rows)", n)
	}
}
