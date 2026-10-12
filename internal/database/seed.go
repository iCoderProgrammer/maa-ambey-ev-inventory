package database

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"sort"

	"github.com/jackc/pgx/v5/pgconn"
)

// Execer is satisfied by pgx.Tx, *pgx.Conn and *pgxpool.Pool.
type Execer interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// ApplySeeds runs every *.sql file at the root of fsys in lexical order and
// returns the files applied. Seed files must be idempotent (e.g. INSERT ...
// ON CONFLICT DO NOTHING) so they can be re-run safely. Callers should pass a
// transaction so a failing file leaves no partial data. Finding no seed files
// is an error: it means the wrong directory or build, not "nothing to do".
func ApplySeeds(ctx context.Context, db Execer, fsys fs.FS) ([]string, error) {
	files, err := fs.Glob(fsys, "*.sql")
	if err != nil {
		return nil, fmt.Errorf("list seed files: %w", err)
	}
	if len(files) == 0 {
		return nil, errors.New("no seed files found")
	}
	sort.Strings(files)

	applied := make([]string, 0, len(files))
	for _, f := range files {
		sql, err := fs.ReadFile(fsys, f)
		if err != nil {
			return applied, fmt.Errorf("read %s: %w", f, err)
		}
		if _, err := db.Exec(ctx, string(sql)); err != nil {
			return applied, fmt.Errorf("apply %s: %w", f, err)
		}
		applied = append(applied, f)
	}
	return applied, nil
}
