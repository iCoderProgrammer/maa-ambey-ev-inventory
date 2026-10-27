// Command access administers roles, permission grants and location scope (B-07).
// There is no HTTP API for these changes; every change made here is written to
// audit_events with the operator's OS account.
//
// Usage:
//
//	go run ./cmd/access roles                     roles and their grants
//	go run ./cmd/access permissions               permission catalogue
//	go run ./cmd/access show --username U         a user's effective access
//	go run ./cmd/access grant --role R --permission P [--approved-by NAME --approval-ref REF]
//	go run ./cmd/access revoke --role R --permission P
//	go run ./cmd/access assign|unassign --username U --role R
//	go run ./cmd/access location-create --code SR1 --name NAME
//	go run ./cmd/access scope-add|scope-remove --username U --location CODE
//
// A grant without --approved-by/--approval-ref is PROPOSED and does not count
// (except in local development with RBAC_ALLOW_PROPOSED_GRANTS=true). Record an
// APPROVED grant only with the owner's real sign-off: who approved it and a
// reference to the written decision. The permission matrix is pending (RB-5).
package main

import (
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

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/audit"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/config"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/database"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/access"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/auth"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/migrations"
)

const usage = "usage: access roles | permissions | show --username U | grant|revoke --role R --permission P [--approved-by NAME --approval-ref REF] | assign|unassign --username U --role R | location-create --code C --name N | scope-add|scope-remove --username U --location C"

var locationCode = regexp.MustCompile(`^[A-Z0-9][A-Z0-9-]{1,19}$`)

type request struct {
	command                    string
	username, role, permission string
	approvedBy, approvalRef    string
	location, name             string
}

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "access: %v\n", err)
		os.Exit(1)
	}
}

func parse(args []string) (request, error) {
	if len(args) == 0 {
		return request{}, errors.New(usage)
	}
	r := request{command: args[0]}
	fs := flag.NewFlagSet(r.command, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&r.username, "username", "", "")
	fs.StringVar(&r.role, "role", "", "")
	fs.StringVar(&r.permission, "permission", "", "")
	fs.StringVar(&r.approvedBy, "approved-by", "", "")
	fs.StringVar(&r.approvalRef, "approval-ref", "", "")
	fs.StringVar(&r.location, "location", "", "")
	fs.StringVar(&r.location, "code", "", "")
	fs.StringVar(&r.name, "name", "", "")
	if err := fs.Parse(args[1:]); err != nil {
		return request{}, fmt.Errorf("%v; %s", err, usage)
	}
	if fs.NArg() > 0 {
		return request{}, fmt.Errorf("unexpected argument %q; %s", fs.Arg(0), usage)
	}
	r.username = auth.NormaliseUsername(r.username)
	r.role = strings.TrimSpace(r.role)
	r.permission = strings.TrimSpace(r.permission)
	r.approvedBy = strings.TrimSpace(r.approvedBy)
	r.approvalRef = strings.TrimSpace(r.approvalRef)
	r.location = strings.ToUpper(strings.TrimSpace(r.location))
	r.name = strings.TrimSpace(r.name)

	need := func(fields map[string]string) error {
		for flagName, v := range fields {
			if v == "" {
				return fmt.Errorf("%s needs --%s", r.command, flagName)
			}
		}
		return nil
	}
	switch r.command {
	case "roles", "permissions":
		return r, nil
	case "show":
		return r, need(map[string]string{"username": r.username})
	case "grant":
		if err := need(map[string]string{"role": r.role, "permission": r.permission}); err != nil {
			return request{}, err
		}
		if (r.approvedBy == "") != (r.approvalRef == "") {
			return request{}, errors.New("an approved grant needs both --approved-by and --approval-ref (the owner's sign-off); omit both to record a proposal")
		}
		if len(r.approvedBy) > 120 || len(r.approvalRef) > 200 {
			return request{}, errors.New("--approved-by is limited to 120 and --approval-ref to 200 characters")
		}
		return r, nil
	case "revoke":
		return r, need(map[string]string{"role": r.role, "permission": r.permission})
	case "assign", "unassign":
		return r, need(map[string]string{"username": r.username, "role": r.role})
	case "location-create":
		if !locationCode.MatchString(r.location) {
			return request{}, errors.New("--code must be 2-20 characters: letters, digits and '-' (stored upper-case)")
		}
		if r.name == "" || len(r.name) > 120 {
			return request{}, errors.New("--name is required (at most 120 characters)")
		}
		return r, nil
	case "scope-add", "scope-remove":
		return r, need(map[string]string{"username": r.username, "location": r.location})
	}
	return request{}, fmt.Errorf("unknown command %q; %s", r.command, usage)
}

func operator() audit.Actor {
	name := "unknown"
	if u, err := osuser.Current(); err == nil && u.Username != "" {
		name = u.Username
	}
	return audit.CLIActor("cmd/access:" + name)
}

func run(args []string, out io.Writer) error {
	r, err := parse(args)
	if err != nil {
		return err
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
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
	store := access.NewPGStore(pool)
	actor := operator()
	say := func(format string, a ...any) { _, _ = fmt.Fprintf(out, format+"\n", a...) }

	switch r.command {
	case "roles":
		m, err := store.RoleMatrix(ctx)
		if err != nil {
			return err
		}
		say("matrix status: %s (%d approved grant(s))", m.MatrixStatus, m.ApprovedGrantCount)
		for _, role := range m.Roles {
			say("%-18s %s", role.Key, role.Name)
			if len(role.Grants) == 0 {
				say("    (no grants — pending owner sign-off)")
			}
			for _, g := range role.Grants {
				if g.Status == access.GrantApproved {
					say("    %-28s APPROVED by %s, ref %s", g.Permission, deref(g.ApprovedBy), deref(g.ApprovalReference))
				} else {
					say("    %-28s PROPOSED (does not count outside development)", g.Permission)
				}
			}
		}
	case "permissions":
		m, err := store.RoleMatrix(ctx)
		if err != nil {
			return err
		}
		for _, p := range m.Permissions {
			flag := ""
			if p.Sensitive {
				flag = " [sensitive]"
			}
			say("%-28s %s%s", p.Key, p.Description, flag)
		}
	case "show":
		uid, err := store.UserIDByName(ctx, r.username)
		if err != nil {
			return err
		}
		approved, err := store.LoadAccess(ctx, uid, false)
		if err != nil {
			return err
		}
		withProposed, err := store.LoadAccess(ctx, uid, true)
		if err != nil {
			return err
		}
		var roles, locs []string
		for _, ro := range approved.Roles {
			roles = append(roles, ro.Name)
		}
		for _, l := range approved.Scope.Locations {
			locs = append(locs, l.Code)
		}
		say("user:        %s", r.username)
		say("roles:       %s", orNone(roles))
		say("permissions: %s", orNone(approved.Permissions()))
		say("  + proposed (development only): %s", orNone(diff(withProposed.Permissions(), approved.Permissions())))
		if approved.Scope.All {
			say("locations:   all (locations.all)")
		} else {
			say("locations:   %s", orNone(locs))
		}
	case "grant":
		var approval *access.Approval
		if r.approvedBy != "" {
			approval = &access.Approval{ApprovedBy: r.approvedBy, Reference: r.approvalRef}
		}
		status, err := store.Grant(ctx, actor, r.role, r.permission, approval)
		if errors.Is(err, access.ErrNoChange) {
			say("no change: %s already has %s as PROPOSED", r.role, r.permission)
			return nil
		}
		if err != nil {
			return err
		}
		say("%s → %s recorded as %s", r.role, r.permission, status)
	case "revoke":
		if err := store.Revoke(ctx, actor, r.role, r.permission); errors.Is(err, access.ErrNoChange) {
			say("no change: %s does not have %s", r.role, r.permission)
		} else if err != nil {
			return err
		} else {
			say("revoked %s from %s", r.permission, r.role)
		}
	case "assign", "unassign":
		f := store.AssignRole
		if r.command == "unassign" {
			f = store.UnassignRole
		}
		if err := f(ctx, actor, r.username, r.role); errors.Is(err, access.ErrNoChange) {
			say("no change")
		} else if err != nil {
			return err
		} else {
			say("%sed role %s for %s; active sessions pick it up on their next request", r.command, r.role, r.username)
		}
	case "location-create":
		l, err := store.CreateLocation(ctx, actor, r.location, r.name)
		if err != nil {
			return err
		}
		say("created location %s (%s)", l.Code, l.ID)
	case "scope-add", "scope-remove":
		f := store.AddLocationScope
		if r.command == "scope-remove" {
			f = store.RemoveLocationScope
		}
		if err := f(ctx, actor, r.username, r.location); errors.Is(err, access.ErrNoChange) {
			say("no change")
		} else if err != nil {
			return err
		} else {
			say("%s: %s %s", r.command, r.username, r.location)
		}
	}
	return nil
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func orNone(items []string) string {
	if len(items) == 0 {
		return "none"
	}
	return strings.Join(items, ", ")
}

func diff(all, minus []string) []string {
	var out []string
	for _, a := range all {
		found := false
		for _, m := range minus {
			found = found || a == m
		}
		if !found {
			out = append(out, a)
		}
	}
	return out
}
