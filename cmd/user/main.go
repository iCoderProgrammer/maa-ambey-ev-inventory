// Command user administers staff accounts (B-06). There is no built-in or
// seeded admin account: the first user is created here by someone with
// database access.
//
// Usage:
//
//	go run ./cmd/user create --username asha.verma --name "Asha Verma" [--email asha@example.com]
//	go run ./cmd/user list
//	go run ./cmd/user disable --username asha.verma        (also ends all their sessions)
//	go run ./cmd/user enable --username asha.verma
//	go run ./cmd/user set-password --username asha.verma   (also ends all their sessions)
//	go run ./cmd/user revoke-sessions --username asha.verma
//
// Passwords are typed at a hidden prompt (twice). When stdin is not a terminal
// the first line of stdin is used, for scripted setups. Passwords are never
// accepted as flags, which would leak into shell history and process lists.
// Roles and permissions are assigned by B-07; a new user is granted nothing.
// Every change is written to audit_events with the operator's OS account.
package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	osuser "os/user"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/term"

	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/audit"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/config"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/database"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/modules/auth"
	"github.com/jarvisconsulting/maa-ambey-enterprises/migrations"
)

var (
	usernamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{2,63}$`)
	emailPattern    = regexp.MustCompile(`^[^@\s]+@[^@\s]+$`)
)

const usage = "usage: user create --username U --name NAME [--email E] | list | disable|enable|set-password|revoke-sessions --username U"

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "user: %v\n", err)
		os.Exit(1)
	}
}

// request is a parsed, validated command.
type request struct {
	command     string
	username    string
	displayName string
	email       *string
}

func parse(args []string) (request, error) {
	if len(args) == 0 {
		return request{}, errors.New(usage)
	}
	req := request{command: args[0]}
	fs := flag.NewFlagSet(req.command, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	username := fs.String("username", "", "login name")
	name := fs.String("name", "", "display name")
	email := fs.String("email", "", "optional e-mail")
	if err := fs.Parse(args[1:]); err != nil {
		return request{}, fmt.Errorf("%v; %s", err, usage)
	}
	if fs.NArg() > 0 {
		return request{}, fmt.Errorf("unexpected argument %q; %s", fs.Arg(0), usage)
	}

	switch req.command {
	case "list":
		return req, nil
	case "create", "disable", "enable", "set-password", "revoke-sessions":
	default:
		return request{}, fmt.Errorf("unknown command %q; %s", req.command, usage)
	}

	req.username = auth.NormaliseUsername(*username)
	if !usernamePattern.MatchString(req.username) {
		return request{}, errors.New("--username must be 3-64 characters: lower-case letters, digits, '.', '_' or '-', starting with a letter or digit")
	}
	if req.command != "create" {
		if *name != "" || *email != "" {
			return request{}, fmt.Errorf("--name and --email are only valid with create")
		}
		return req, nil
	}
	req.displayName = strings.TrimSpace(*name)
	if req.displayName == "" || utf8.RuneCountInString(req.displayName) > 120 {
		return request{}, errors.New("--name is required (at most 120 characters)")
	}
	if e := strings.TrimSpace(*email); e != "" {
		if !emailPattern.MatchString(e) || len(e) > 254 {
			return request{}, errors.New("--email is not a valid e-mail address")
		}
		req.email = &e
	}
	return req, nil
}

func run(args []string, stdin io.Reader, out io.Writer) error {
	req, err := parse(args)
	if err != nil {
		return err
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	pool, err := database.Connect(ctx, cfg)
	if err != nil {
		return err
	}
	defer pool.Close()
	want, err := migrations.Latest()
	if err != nil {
		return err
	}
	if err := database.CheckSchema(ctx, pool, want); err != nil {
		return fmt.Errorf("database not ready: %w", err)
	}
	store := auth.NewPGStore(pool)
	now := time.Now().UTC()
	actor := operator()

	switch req.command {
	case "list":
		users, err := store.ListUsers(ctx)
		if err != nil {
			return err
		}
		if len(users) == 0 {
			_, _ = fmt.Fprintln(out, "no users")
		}
		for _, u := range users {
			last := "never"
			if u.LastLoginAt != nil {
				last = u.LastLoginAt.Local().Format(time.DateTime)
			}
			_, _ = fmt.Fprintf(out, "%-24s %-8s %-30s last sign-in: %s\n", u.Username, u.Status, u.DisplayName, last)
		}
		return nil

	case "create":
		hash, err := newPasswordHash(req.username, stdin, out)
		if err != nil {
			return err
		}
		u, err := store.CreateUser(ctx, actor, req.username, req.displayName, req.email, hash)
		if errors.Is(err, auth.ErrUsernameTaken) {
			return fmt.Errorf("a user with username %q or that e-mail already exists", req.username)
		}
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintf(out, "created user %s (%s), status %s. No roles are assigned yet (B-07).\n", u.Username, u.ID, u.Status)
		return nil

	case "set-password":
		hash, err := newPasswordHash(req.username, stdin, out)
		if err != nil {
			return err
		}
		n, err := store.SetPassword(ctx, actor, req.username, hash, now)
		if err != nil {
			return notFound(err, req.username)
		}
		_, _ = fmt.Fprintf(out, "password changed for %s; %d session(s) ended\n", req.username, n)
		return nil

	case "disable", "enable":
		status := auth.StatusDisabled
		if req.command == "enable" {
			status = auth.StatusActive
		}
		n, err := store.SetStatus(ctx, actor, req.username, status, now)
		if err != nil {
			return notFound(err, req.username)
		}
		_, _ = fmt.Fprintf(out, "%s is now %s; %d session(s) ended\n", req.username, status, n)
		return nil

	case "revoke-sessions":
		n, err := store.RevokeAllSessions(ctx, actor, req.username, now)
		if err != nil {
			return notFound(err, req.username)
		}
		_, _ = fmt.Fprintf(out, "%d session(s) ended for %s\n", n, req.username)
		return nil
	}
	return nil
}

// operator identifies who ran the command, for the audit trail: the OS account
// (there is no signed-in user on the command line).
func operator() audit.Actor {
	name := "unknown"
	if u, err := osuser.Current(); err == nil && u.Username != "" {
		name = u.Username
	}
	return audit.CLIActor("cmd/user:" + name)
}

func notFound(err error, username string) error {
	if errors.Is(err, auth.ErrNotFound) {
		return fmt.Errorf("no user named %q", username)
	}
	return err
}

// newPasswordHash reads a new password (hidden prompt, or first stdin line),
// applies the password policy and returns its argon2id hash.
func newPasswordHash(username string, stdin io.Reader, out io.Writer) (string, error) {
	var password string
	if f, ok := stdin.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		_, _ = fmt.Fprintf(out, "New password for %s (min %d characters): ", username, auth.MinPasswordLength)
		first, err := term.ReadPassword(int(f.Fd()))
		_, _ = fmt.Fprintln(out)
		if err != nil {
			return "", fmt.Errorf("read password: %w", err)
		}
		_, _ = fmt.Fprint(out, "Repeat password: ")
		second, err := term.ReadPassword(int(f.Fd()))
		_, _ = fmt.Fprintln(out)
		if err != nil {
			return "", fmt.Errorf("read password: %w", err)
		}
		if string(first) != string(second) {
			return "", errors.New("passwords do not match")
		}
		password = string(first)
	} else {
		line, err := bufio.NewReader(stdin).ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return "", fmt.Errorf("read password: %w", err)
		}
		password = strings.TrimRight(line, "\r\n")
	}
	if err := auth.ValidateNewPassword(username, password); err != nil {
		return "", err
	}
	return auth.HashPassword(password)
}
