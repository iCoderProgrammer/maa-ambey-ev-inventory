// Command migrate applies versioned SQL migrations from ./migrations.
//
// Usage:
//
//	go run ./cmd/migrate up          apply all pending migrations
//	go run ./cmd/migrate down [N]    roll back N migrations (default 1)
//	go run ./cmd/migrate version     print the current version
//	go run ./cmd/migrate force V     mark version V as clean after a failed migration (no SQL is run)
//
// It reads DATABASE_URL and MIGRATIONS_DIR (default ./migrations). Nothing is
// ever applied automatically by the API server.
package main

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "migrate: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return errors.New("missing command: up | down [N] | version | force V")
	}
	dbURL, err := driverURL(os.Getenv("DATABASE_URL"))
	if err != nil {
		return err
	}
	dir := os.Getenv("MIGRATIONS_DIR")
	if dir == "" {
		dir = "migrations"
	}

	m, err := migrate.New("file://"+dir, dbURL)
	if err != nil {
		return fmt.Errorf("open migrations: %w", err)
	}
	defer m.Close()

	switch args[0] {
	case "up":
		err = m.Up()
	case "down":
		n := 1
		if len(args) > 1 {
			if n, err = strconv.Atoi(args[1]); err != nil || n < 1 {
				return errors.New("down expects a positive number of steps")
			}
		}
		err = m.Steps(-n)
	case "version":
		v, dirty, verr := m.Version()
		if errors.Is(verr, migrate.ErrNilVersion) {
			fmt.Println("no migrations applied")
			return nil
		}
		if verr != nil {
			return verr
		}
		fmt.Printf("version %d (dirty=%t)\n", v, dirty)
		return nil
	case "force":
		if len(args) < 2 {
			return errors.New("force expects a version number")
		}
		v, perr := strconv.Atoi(args[1])
		if perr != nil {
			return errors.New("force expects a numeric version")
		}
		err = m.Force(v)
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}

	if errors.Is(err, migrate.ErrNoChange) {
		fmt.Println("no change")
		return nil
	}
	if err != nil {
		return err
	}
	v, dirty, _ := m.Version()
	fmt.Printf("ok: now at version %d (dirty=%t)\n", v, dirty)
	return nil
}

// driverURL converts a postgres:// DSN into the pgx5:// scheme golang-migrate expects.
func driverURL(raw string) (string, error) {
	if raw == "" {
		return "", errors.New("DATABASE_URL is required")
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") {
		return "", errors.New("DATABASE_URL must be a postgres:// URL")
	}
	u.Scheme = "pgx5"
	return u.String(), nil
}
