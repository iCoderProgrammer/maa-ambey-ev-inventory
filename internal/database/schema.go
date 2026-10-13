package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// Schema state errors. They describe migration status only and never contain
// connection details, so they are safe to log and to print from the CLIs.
var (
	ErrSchemaNotMigrated = errors.New("no migrations applied")
	ErrSchemaDirty       = errors.New("last migration failed and left the schema dirty")
	ErrSchemaBehind      = errors.New("schema is behind this build (migrations pending)")
	ErrSchemaAhead       = errors.New("schema is newer than this build")
)

// Querier is satisfied by pgx.Tx, *pgx.Conn and *pgxpool.Pool.
type Querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// SchemaVersion reads the golang-migrate bookkeeping table. ok is false when
// no migration has ever been applied.
func SchemaVersion(ctx context.Context, q Querier) (version uint, dirty, ok bool, err error) {
	var exists bool
	if err := q.QueryRow(ctx, `SELECT to_regclass('schema_migrations') IS NOT NULL`).Scan(&exists); err != nil {
		return 0, false, false, fmt.Errorf("read schema version: %w", err)
	}
	if !exists {
		return 0, false, false, nil
	}
	var v int64
	err = q.QueryRow(ctx, `SELECT version, dirty FROM schema_migrations LIMIT 1`).Scan(&v, &dirty)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, false, false, nil
	}
	if err != nil {
		return 0, false, false, fmt.Errorf("read schema version: %w", err)
	}
	if v < 0 {
		return 0, false, false, nil
	}
	return uint(v), dirty, true, nil
}

// CheckSchema returns nil only when the database is migrated exactly to want
// and the last migration completed cleanly.
func CheckSchema(ctx context.Context, q Querier, want uint) error {
	v, dirty, ok, err := SchemaVersion(ctx, q)
	switch {
	case err != nil:
		return err
	case !ok:
		return fmt.Errorf("%w; run `make db-migrate-up`", ErrSchemaNotMigrated)
	case dirty:
		return fmt.Errorf("%w at version %d; repair it, then `go run ./cmd/migrate force <last good version>`", ErrSchemaDirty, v)
	case v < want:
		return fmt.Errorf("%w: database at %d, build expects %d; run `make db-migrate-up`", ErrSchemaBehind, v, want)
	case v > want:
		return fmt.Errorf("%w: database at %d, build expects %d; deploy the matching build", ErrSchemaAhead, v, want)
	}
	return nil
}

// SchemaChecker reports whether the pool's database matches the migration
// version this binary was built with (see migrations.Latest).
type SchemaChecker struct {
	DB   Querier
	Want uint
}

// CheckSchema implements health.SchemaChecker.
func (s SchemaChecker) CheckSchema(ctx context.Context) error {
	return CheckSchema(ctx, s.DB, s.Want)
}
