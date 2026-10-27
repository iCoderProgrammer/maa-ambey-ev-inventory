package access_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/audit"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/database"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/database/dbtest"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/access"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/auth"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/seeds"
)

var operator = audit.CLIActor("cmd/access:test")

// seededPool is a migrated, seeded throwaway database with one user "asha".
func seededPool(t *testing.T) (*pgxpool.Pool, *access.PGStore, string) {
	t.Helper()
	ctx := context.Background()
	pool := dbtest.MigratedPool(t)
	if _, err := database.ApplySeeds(ctx, pool, seeds.FS); err != nil {
		t.Fatal(err)
	}
	hash, _ := auth.HashPassword(password)
	u, err := auth.NewPGStore(pool).CreateUser(ctx, operator, "asha", "Asha", nil, hash)
	if err != nil {
		t.Fatal(err)
	}
	return pool, access.NewPGStore(pool), u.ID
}

func TestPermissionCatalogueMatchesDatabase(t *testing.T) {
	pool, store, _ := seededPool(t)
	m, err := store.RoleMatrix(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var inDB []string
	for _, p := range m.Permissions {
		inDB = append(inDB, p.Key)
	}
	code := slices.Clone(access.AllPermissions)
	slices.Sort(code)
	if !slices.Equal(inDB, code) {
		t.Errorf("permission catalogue drift:\n db   %v\n code %v", inDB, code)
	}
	for _, key := range []string{"catalogue.cost_price.view", "sales.refund.approve", "stock.adjustment.approve",
		"access.roles.manage", "settings.invoice.manage", "reports.export", "customers.export"} {
		if !slices.Contains(inDB, key) {
			t.Errorf("required permission key %s missing", key)
		}
	}
	var grants int
	_ = pool.QueryRow(context.Background(), `SELECT count(*) FROM role_permissions`).Scan(&grants)
	if grants != 0 {
		t.Errorf("a fresh database must grant nothing; found %d grants", grants)
	}
}

func TestConfirmedRolesAreRepresentedAndPending(t *testing.T) {
	_, store, _ := seededPool(t)
	m, err := store.RoleMatrix(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, r := range m.Roles {
		names = append(names, r.Name)
		if len(r.Grants) != 0 {
			t.Errorf("%s has grants on a fresh database", r.Name)
		}
	}
	want := []string{"Owner", "Inventory Manager", "Sales Executive", "Accountant", "Viewer"}
	slices.Sort(names)
	slices.Sort(want)
	if !slices.Equal(names, want) || m.MatrixStatus != access.MatrixPending || m.ApprovedGrantCount != 0 {
		t.Errorf("roles=%v status=%s approved=%d", names, m.MatrixStatus, m.ApprovedGrantCount)
	}
}

func TestEffectiveAccessFollowsApprovalAndAssignment(t *testing.T) {
	ctx := context.Background()
	_, store, uid := seededPool(t)
	load := func(allowProposed bool) access.Access {
		a, err := store.LoadAccess(ctx, uid, allowProposed)
		if err != nil {
			t.Fatal(err)
		}
		return a
	}
	if a := load(true); len(a.Permissions()) != 0 || len(a.Roles) != 0 {
		t.Fatalf("new user: %+v", a)
	}

	if _, err := store.Grant(ctx, operator, "viewer", access.PermRolesView, nil); err != nil {
		t.Fatal(err)
	}
	if a := load(true); a.Has(access.PermRolesView) {
		t.Error("a grant on a role the user does not hold must not count")
	}
	if err := store.AssignRole(ctx, operator, "asha", "viewer"); err != nil {
		t.Fatal(err)
	}
	if a := load(false); a.Has(access.PermRolesView) {
		t.Error("a PROPOSED grant must not count when proposed grants are off (staging/production)")
	}
	if a := load(true); !a.Has(access.PermRolesView) || len(a.Roles) != 1 || a.Roles[0].Name != "Viewer" {
		t.Errorf("proposed grant with the dev switch on: %+v", a)
	}

	if _, err := store.Grant(ctx, operator, "viewer", access.PermRolesView, &access.Approval{ApprovedBy: "Owner (test)", Reference: "TEST-ONLY-1"}); err != nil {
		t.Fatal(err)
	}
	if a := load(false); !a.Has(access.PermRolesView) {
		t.Error("an APPROVED grant must count")
	}
	if _, err := store.Grant(ctx, operator, "viewer", access.PermRolesView, nil); !errors.Is(err, access.ErrAlreadyApproved) {
		t.Errorf("downgrading an approved grant: %v", err)
	}
	m, _ := store.RoleMatrix(ctx)
	if m.MatrixStatus != access.MatrixPartiallyApproved || m.ApprovedGrantCount != 1 {
		t.Errorf("matrix = %s/%d", m.MatrixStatus, m.ApprovedGrantCount)
	}

	if err := store.UnassignRole(ctx, operator, "asha", "viewer"); err != nil {
		t.Fatal(err)
	}
	if a := load(true); a.Has(access.PermRolesView) {
		t.Error("removing the role must remove its permissions")
	}
	if err := store.UnassignRole(ctx, operator, "asha", "viewer"); !errors.Is(err, access.ErrNoChange) {
		t.Errorf("second unassign: %v", err)
	}
	if err := store.Revoke(ctx, operator, "viewer", access.PermRolesView); err != nil {
		t.Fatal(err)
	}
	for _, err := range []error{
		store.AssignRole(ctx, operator, "nobody", "viewer"),
		store.AssignRole(ctx, operator, "asha", "superuser"),
		func() error { _, err := store.Grant(ctx, operator, "viewer", "everything.allowed", nil); return err }(),
	} {
		if !errors.Is(err, access.ErrNotFound) {
			t.Errorf("unknown user/role/permission: %v", err)
		}
	}
}

func TestLocationScope(t *testing.T) {
	ctx := context.Background()
	pool, store, uid := seededPool(t)
	sr1, err := store.CreateLocation(ctx, operator, "SR1", "Test showroom 1")
	if err != nil {
		t.Fatal(err)
	}
	sr2, _ := store.CreateLocation(ctx, operator, "SR2", "Test showroom 2")
	if _, err := store.CreateLocation(ctx, operator, "SR1", "Duplicate"); !errors.Is(err, access.ErrDuplicate) {
		t.Errorf("duplicate code: %v", err)
	}
	if err := store.AddLocationScope(ctx, operator, "asha", "SR1"); err != nil {
		t.Fatal(err)
	}
	a, _ := store.LoadAccess(ctx, uid, false)
	if !a.Scope.Allows(sr1.ID) || a.Scope.Allows(sr2.ID) || a.Scope.All {
		t.Errorf("scope = %+v", a.Scope)
	}
	if _, _, err := a.Scope.Restrict([]string{sr2.ID}); !errors.Is(err, access.ErrOutOfScope) {
		t.Errorf("cross-location filter: %v", err)
	}

	// Deactivated locations drop out of scope.
	if _, err := pool.Exec(ctx, `UPDATE locations SET active = false WHERE code = 'SR1'`); err != nil {
		t.Fatal(err)
	}
	if a, _ := store.LoadAccess(ctx, uid, false); a.Scope.Allows(sr1.ID) {
		t.Error("inactive location still in scope")
	}

	// locations.all (approved) grants every location.
	_ = store.AssignRole(ctx, operator, "asha", "owner")
	_, _ = store.Grant(ctx, operator, "owner", access.PermAllLocations, &access.Approval{ApprovedBy: "Owner (test)", Reference: "TEST-ONLY-2"})
	if a, _ := store.LoadAccess(ctx, uid, false); !a.Scope.All || !a.Scope.Allows(sr2.ID) {
		t.Errorf("locations.all: %+v", a.Scope)
	}
}

func TestAccessChangesAreAudited(t *testing.T) {
	ctx := context.Background()
	pool, store, _ := seededPool(t)
	_, _ = store.Grant(ctx, operator, "viewer", access.PermRolesView, nil)
	_, _ = store.Grant(ctx, operator, "viewer", access.PermRolesView, &access.Approval{ApprovedBy: "Owner (test)", Reference: "TEST-ONLY-3"})
	_ = store.AssignRole(ctx, operator, "asha", "viewer")
	_, _ = store.CreateLocation(ctx, operator, "SR1", "Test showroom")
	_ = store.AddLocationScope(ctx, operator, "asha", "SR1")
	_ = store.RemoveLocationScope(ctx, operator, "asha", "SR1")
	_ = store.UnassignRole(ctx, operator, "asha", "viewer")
	_ = store.Revoke(ctx, operator, "viewer", access.PermRolesView)
	_ = store.AssignRole(ctx, operator, "asha", "viewer")
	_ = store.AssignRole(ctx, operator, "asha", "viewer") // no change → no event

	rows, err := pool.Query(ctx, `SELECT action FROM audit_events WHERE action LIKE 'access.%' OR action LIKE 'location.%' ORDER BY id`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var got []string
	for rows.Next() {
		var a string
		_ = rows.Scan(&a)
		got = append(got, a)
	}
	want := []string{"access.grant_proposed", "access.grant_approved", "access.role_assigned", "location.created",
		"access.location_scope_added", "access.location_scope_removed", "access.role_unassigned", "access.grant_revoked",
		"access.role_assigned"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("audit actions:\n got  %v\n want %v", got, want)
	}
	var summary string
	_ = pool.QueryRow(ctx, `SELECT summary::text FROM audit_events WHERE action = 'access.grant_approved'`).Scan(&summary)
	if !strings.Contains(summary, `"approvalReference": "TEST-ONLY-3"`) || !strings.Contains(summary, `"from": "PROPOSED"`) {
		t.Errorf("grant_approved summary = %s", summary)
	}
}

func TestApprovedGrantNeedsEvidenceInTheDatabase(t *testing.T) {
	ctx := context.Background()
	pool, _, _ := seededPool(t)
	_, err := pool.Exec(ctx, `
		INSERT INTO role_permissions (role_id, permission_key, status)
		SELECT id, 'access.roles.view', 'APPROVED' FROM roles WHERE key = 'viewer'`)
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23514" {
		t.Errorf("approved grant without approver/reference: %v", err)
	}
}
