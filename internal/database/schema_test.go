package database

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

// fakeRow scans fixed values (or returns err) like a pgx.Row.
type fakeRow struct {
	vals []any
	err  error
}

func (r fakeRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	for i, d := range dest {
		switch p := d.(type) {
		case *bool:
			*p = r.vals[i].(bool)
		case *int64:
			*p = r.vals[i].(int64)
		}
	}
	return nil
}

// fakeQuerier answers the existence query first, then the version query.
type fakeQuerier struct{ rows []fakeRow }

func (q *fakeQuerier) QueryRow(context.Context, string, ...any) pgx.Row {
	r := q.rows[0]
	q.rows = q.rows[1:]
	return r
}

func TestCheckSchema(t *testing.T) {
	exists := fakeRow{vals: []any{true}}
	tests := []struct {
		name    string
		rows    []fakeRow
		want    error
		wantMsg string
	}{
		{"current", []fakeRow{exists, {vals: []any{int64(3), false}}}, nil, ""},
		{"no table", []fakeRow{{vals: []any{false}}}, ErrSchemaNotMigrated, "make db-migrate-up"},
		{"empty table", []fakeRow{exists, {err: pgx.ErrNoRows}}, ErrSchemaNotMigrated, ""},
		{"nil version", []fakeRow{exists, {vals: []any{int64(-1), false}}}, ErrSchemaNotMigrated, ""},
		{"dirty", []fakeRow{exists, {vals: []any{int64(3), true}}}, ErrSchemaDirty, "force"},
		{"behind", []fakeRow{exists, {vals: []any{int64(2), false}}}, ErrSchemaBehind, "database at 2, build expects 3"},
		{"ahead", []fakeRow{exists, {vals: []any{int64(4), false}}}, ErrSchemaAhead, "deploy the matching build"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckSchema(context.Background(), &fakeQuerier{rows: tt.rows}, 3)
			if tt.want == nil {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if !errors.Is(err, tt.want) || !strings.Contains(err.Error(), tt.wantMsg) {
				t.Fatalf("err = %v, want %v containing %q", err, tt.want, tt.wantMsg)
			}
		})
	}
}

func TestCheckSchemaPropagatesQueryErrors(t *testing.T) {
	boom := errors.New("connection reset")
	err := CheckSchema(context.Background(), &fakeQuerier{rows: []fakeRow{{err: boom}}}, 1)
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v", err)
	}
}
