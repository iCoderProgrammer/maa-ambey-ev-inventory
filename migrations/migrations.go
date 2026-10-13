// Package migrations embeds the versioned SQL migrations so cmd/migrate is
// self-contained and the API knows which schema version it was built for.
//
// Policy (docs/DEVELOPMENT_SETUP.md, "Database migrations"): files are named
// NNNNNN_snake_case.up.sql / .down.sql, versions start at 1 and have no gaps,
// and every version has exactly one up and one down file.
package migrations

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"regexp"
	"strconv"
	"strings"
)

// FS holds every *.sql migration file in this directory.
//
//go:embed *.sql
var FS embed.FS

var fileName = regexp.MustCompile(`^(\d{6})_([a-z0-9]+(?:_[a-z0-9]+)*)\.(up|down)\.sql$`)

// Latest validates the embedded migrations and returns the highest version.
func Latest() (uint, error) {
	return Validate(FS)
}

// Validate checks the naming and pairing policy for the *.sql files at the
// root of fsys and returns the highest version (0 when there are none).
func Validate(fsys fs.FS) (uint, error) {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return 0, fmt.Errorf("read migrations: %w", err)
	}

	type pair struct {
		name     string
		up, down bool
	}
	versions := map[uint]*pair{}
	var errs []error
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		m := fileName.FindStringSubmatch(e.Name())
		if m == nil {
			errs = append(errs, fmt.Errorf("%s: name must look like 000001_create_things.up.sql", e.Name()))
			continue
		}
		n, _ := strconv.ParseUint(m[1], 10, 32)
		v := uint(n)
		if v == 0 {
			errs = append(errs, fmt.Errorf("%s: versions start at 000001", e.Name()))
			continue
		}
		p, ok := versions[v]
		if !ok {
			p = &pair{name: m[2]}
			versions[v] = p
		} else if p.name != m[2] {
			errs = append(errs, fmt.Errorf("version %06d is used by two migrations (%s, %s)", v, p.name, m[2]))
			continue
		}
		if m[3] == "up" {
			p.up = true
		} else {
			p.down = true
		}
	}

	var latest uint
	for v := range versions {
		latest = max(latest, v)
	}
	// Report in version order so the message is stable.
	for v := uint(1); v <= latest; v++ {
		p, ok := versions[v]
		switch {
		case !ok:
			errs = append(errs, fmt.Errorf("version %06d is missing; versions must be sequential", v))
		case !p.up:
			errs = append(errs, fmt.Errorf("version %06d (%s) has no .up.sql", v, p.name))
		case !p.down:
			errs = append(errs, fmt.Errorf("version %06d (%s) has no .down.sql", v, p.name))
		}
	}
	if len(errs) > 0 {
		return 0, fmt.Errorf("invalid migrations: %w", errors.Join(errs...))
	}
	return latest, nil
}
