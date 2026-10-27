// Command migrate applies the versioned SQL migrations embedded from ./migrations.
//
// Usage:
//
//	go run ./cmd/migrate up                    apply all pending migrations
//	go run ./cmd/migrate down [N] [--confirm]  roll back N migrations (default 1)
//	go run ./cmd/migrate version               print the current version
//	go run ./cmd/migrate force V [--confirm]   mark version V as clean after a failed migration (no SQL is run)
//
// It reads DATABASE_URL and APP_ENV. Policy (docs/DEVELOPMENT_SETUP.md):
// nothing is ever applied automatically by the API server; there is no
// "drop everything" command; in staging and production `down` and `force`
// are refused unless --confirm is passed after a verified backup.
package main

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/config"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/migrations"
)

func main() {
	if err := run(os.Args[1:], os.Getenv, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "migrate: %v\n", err)
		os.Exit(1)
	}
}

// command is a parsed, policy-checked invocation.
type command struct {
	name    string // up | down | version | force
	steps   int    // down
	version int    // force
}

func run(args []string, getenv func(string) string, out io.Writer) error {
	appEnv := getenv("APP_ENV")
	if appEnv == "" {
		appEnv = config.EnvDevelopment
	}
	latest, err := migrations.Latest()
	if err != nil {
		return err
	}
	cmd, err := parse(args, appEnv, latest)
	if err != nil {
		return err
	}
	dbURL, err := driverURL(getenv("DATABASE_URL"), appEnv)
	if err != nil {
		return err
	}

	src, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return fmt.Errorf("open embedded migrations: %w", err)
	}
	m, err := migrate.NewWithSourceInstance("iofs", src, dbURL)
	if err != nil {
		return fmt.Errorf("connect to PostgreSQL (is `make db-up` running?): %s", redact(err.Error(), getenv("DATABASE_URL")))
	}
	defer m.Close()

	switch cmd.name {
	case "up":
		err = m.Up()
	case "down":
		err = m.Steps(-cmd.steps)
	case "force":
		err = m.Force(cmd.version)
	case "version":
		v, dirty, verr := m.Version()
		if errors.Is(verr, migrate.ErrNilVersion) {
			_, _ = fmt.Fprintf(out, "no migrations applied (this build expects version %d)\n", latest)
			return nil
		}
		if verr != nil {
			return verr
		}
		_, _ = fmt.Fprintf(out, "version %d (dirty=%t); this build expects version %d\n", v, dirty, latest)
		return nil
	}

	if errors.Is(err, migrate.ErrNoChange) {
		_, _ = fmt.Fprintln(out, "no change")
		return nil
	}
	if err != nil {
		return err
	}
	v, dirty, verr := m.Version()
	if errors.Is(verr, migrate.ErrNilVersion) {
		_, _ = fmt.Fprintln(out, "ok: no migrations applied")
		return nil
	}
	_, _ = fmt.Fprintf(out, "ok: now at version %d (dirty=%t)\n", v, dirty)
	return nil
}

// parse validates the command line and enforces the rollback policy before
// any database connection is made.
func parse(args []string, appEnv string, latest uint) (command, error) {
	switch appEnv {
	case config.EnvDevelopment, config.EnvTest, config.EnvStaging, config.EnvProduction:
	default:
		return command{}, fmt.Errorf("APP_ENV must be one of development, test, staging, production (got %q)", appEnv)
	}
	confirmed := slices.Contains(args, "--confirm")
	args = slices.DeleteFunc(slices.Clone(args), func(a string) bool { return a == "--confirm" })
	if len(args) == 0 {
		return command{}, errors.New("missing command: up | down [N] [--confirm] | version | force V [--confirm]")
	}

	cmd := command{name: args[0]}
	switch cmd.name {
	case "up", "version":
		if len(args) > 1 {
			return command{}, fmt.Errorf("%s takes no arguments", cmd.name)
		}
		return cmd, nil
	case "down":
		cmd.steps = 1
		if len(args) > 2 {
			return command{}, errors.New("down takes at most one argument: the number of steps")
		}
		if len(args) == 2 {
			n, err := strconv.Atoi(args[1])
			if err != nil || n < 1 {
				return command{}, errors.New("down expects a positive number of steps")
			}
			cmd.steps = n
		}
	case "force":
		if len(args) != 2 {
			return command{}, errors.New("force expects exactly one version number")
		}
		v, err := strconv.Atoi(args[1])
		if err != nil || v < 0 || uint(v) > latest {
			return command{}, fmt.Errorf("force expects a version between 0 and %d", latest)
		}
		cmd.version = v
	default:
		return command{}, fmt.Errorf("unknown command %q (up | down | version | force)", cmd.name)
	}

	// down and force can lose data or hide a broken schema.
	if (appEnv == config.EnvStaging || appEnv == config.EnvProduction) && !confirmed {
		return command{}, fmt.Errorf("refusing %q with APP_ENV=%s: take and verify a backup, then re-run with --confirm", cmd.name, appEnv)
	}
	return cmd, nil
}

// driverURL validates DATABASE_URL like the API does and converts it to the
// pgx5:// scheme golang-migrate expects.
func driverURL(raw, appEnv string) (string, error) {
	if err := config.ValidateDatabaseURL(raw, appEnv); err != nil {
		return "", err
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", errors.New("DATABASE_URL is not a valid URL")
	}
	u.Scheme = "pgx5"
	return u.String(), nil
}

// redact removes the DATABASE_URL password from a driver error message.
func redact(msg, rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "invalid DATABASE_URL"
	}
	// Replace only the URL userinfo form (":password@"), so a short password
	// cannot mangle unrelated words in the message.
	if pw, ok := u.User.Password(); ok && pw != "" {
		for _, form := range []string{pw, url.QueryEscape(pw), url.PathEscape(pw)} {
			msg = strings.ReplaceAll(msg, ":"+form+"@", ":xxxxx@")
		}
	}
	return msg
}
