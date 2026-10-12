package database

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/jackc/pgx/v5/pgconn"
)

// Execer is satisfied by pgx.Tx, *pgx.Conn and *pgxpool.Pool.
type Execer interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// ApplySeeds runs every *.sql file in dir in lexical order and returns the
// files applied. Seed files must be idempotent (e.g. INSERT ... ON CONFLICT
// DO NOTHING) so they can be re-run safely. Callers should pass a transaction
// so a failing file leaves no partial data.
func ApplySeeds(ctx context.Context, db Execer, dir string) ([]string, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*.sql"))
	if err != nil {
		return nil, fmt.Errorf("list seed files: %w", err)
	}
	sort.Strings(files)

	applied := make([]string, 0, len(files))
	for _, f := range files {
		sql, err := os.ReadFile(f)
		if err != nil {
			return applied, fmt.Errorf("read %s: %w", f, err)
		}
		if _, err := db.Exec(ctx, string(sql)); err != nil {
			return applied, fmt.Errorf("apply %s: %w", filepath.Base(f), err)
		}
		applied = append(applied, filepath.Base(f))
	}
	return applied, nil
}
