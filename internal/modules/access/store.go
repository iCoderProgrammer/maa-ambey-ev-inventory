package access

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/audit"
)

// Grant statuses (role_permissions.status).
const (
	GrantProposed = "PROPOSED"
	GrantApproved = "APPROVED"
)

// Matrix statuses reported by GET /access/roles.
const (
	MatrixPending           = "PENDING_OWNER_SIGN_OFF"
	MatrixPartiallyApproved = "PARTIALLY_APPROVED"
)

var (
	// ErrNotFound: unknown user, role, permission or location.
	ErrNotFound = errors.New("not found")
	// ErrAlreadyApproved: an approved grant cannot be downgraded to proposed; revoke it instead.
	ErrAlreadyApproved = errors.New("grant is already approved")
	// ErrNoChange: the requested state already holds; nothing was written or audited.
	ErrNoChange = errors.New("no change")
	// ErrDuplicate: a location with that code exists.
	ErrDuplicate = errors.New("already exists")
)

// Loader is what the HTTP layer needs; PGStore implements it.
type Loader interface {
	LoadAccess(ctx context.Context, userID string, allowProposed bool) (Access, error)
	RoleMatrix(ctx context.Context) (Matrix, error)
}

// Grant is one role → permission grant.
type Grant struct {
	Permission        string     `json:"permission" example:"access.roles.view"`
	Status            string     `json:"status" enums:"PROPOSED,APPROVED" example:"PROPOSED"`
	ApprovedBy        *string    `json:"approvedBy"`
	ApprovalReference *string    `json:"approvalReference"`
	ApprovedAt        *time.Time `json:"approvedAt"`
} //@name AccessGrant

// MatrixRole is a role and its grants.
type MatrixRole struct {
	Key         string  `json:"key" example:"owner"`
	Name        string  `json:"name" example:"Owner"`
	Description string  `json:"description" example:"Client-confirmed role. Permissions pending owner sign-off."`
	Grants      []Grant `json:"grants"`
} //@name AccessMatrixRole

// PermissionInfo describes a catalogue entry.
type PermissionInfo struct {
	Key         string `json:"key" example:"catalogue.cost_price.view"`
	Description string `json:"description" example:"View cost prices and stock valuation"`
	Sensitive   bool   `json:"sensitive" example:"true"`
} //@name AccessPermission

// Matrix is the payload of GET /access/roles.
type Matrix struct {
	MatrixStatus       string           `json:"matrixStatus" enums:"PENDING_OWNER_SIGN_OFF,PARTIALLY_APPROVED"`
	ApprovedGrantCount int              `json:"approvedGrantCount" example:"0"`
	Roles              []MatrixRole     `json:"roles"`
	Permissions        []PermissionInfo `json:"permissions"`
} //@name RoleMatrix

// Approval is the owner's sign-off evidence for a grant.
type Approval struct {
	ApprovedBy string
	Reference  string
}

// PGStore implements Loader and the administration used by cmd/access.
type PGStore struct{ pool *pgxpool.Pool }

// NewPGStore wraps a pgx pool.
func NewPGStore(pool *pgxpool.Pool) *PGStore { return &PGStore{pool: pool} }

// LoadAccess resolves a user's roles, effective permissions and location scope.
func (s *PGStore) LoadAccess(ctx context.Context, userID string, allowProposed bool) (Access, error) {
	roles, err := collect(ctx, s.pool, `
		SELECT r.key, r.name FROM user_roles ur JOIN roles r ON r.id = ur.role_id
		WHERE ur.user_id = $1 ORDER BY r.name`, []any{userID},
		func(row pgx.Rows) (Role, error) {
			var r Role
			return r, row.Scan(&r.Key, &r.Name)
		})
	if err != nil {
		return Access{}, fmt.Errorf("load roles: %w", err)
	}
	perms, err := collect(ctx, s.pool, `
		SELECT DISTINCT rp.permission_key
		FROM user_roles ur JOIN role_permissions rp ON rp.role_id = ur.role_id
		WHERE ur.user_id = $1 AND (rp.status = 'APPROVED' OR $2)`, []any{userID, allowProposed},
		func(row pgx.Rows) (string, error) {
			var p string
			return p, row.Scan(&p)
		})
	if err != nil {
		return Access{}, fmt.Errorf("load permissions: %w", err)
	}
	locations, err := collect(ctx, s.pool, `
		SELECT l.id::text, l.code, l.name
		FROM user_location_scopes ls JOIN locations l ON l.id = ls.location_id
		WHERE ls.user_id = $1 AND l.active ORDER BY l.code`, []any{userID},
		func(row pgx.Rows) (Location, error) {
			var l Location
			return l, row.Scan(&l.ID, &l.Code, &l.Name)
		})
	if err != nil {
		return Access{}, fmt.Errorf("load location scope: %w", err)
	}
	return NewAccess(userID, roles, perms, locations, allowProposed), nil
}

// RoleMatrix returns every role with its grants and the permission catalogue.
func (s *PGStore) RoleMatrix(ctx context.Context) (Matrix, error) {
	roles, err := collect(ctx, s.pool, `SELECT id::text, key, name, description FROM roles ORDER BY created_at, key`, nil,
		func(row pgx.Rows) (struct {
			id string
			MatrixRole
		}, error) {
			var r struct {
				id string
				MatrixRole
			}
			return r, row.Scan(&r.id, &r.Key, &r.Name, &r.Description)
		})
	if err != nil {
		return Matrix{}, fmt.Errorf("load roles: %w", err)
	}
	type grantRow struct {
		roleID string
		Grant
	}
	grants, err := collect(ctx, s.pool, `
		SELECT role_id::text, permission_key, status, approved_by, approval_reference, approved_at
		FROM role_permissions ORDER BY permission_key`, nil,
		func(row pgx.Rows) (grantRow, error) {
			var g grantRow
			return g, row.Scan(&g.roleID, &g.Permission, &g.Status, &g.ApprovedBy, &g.ApprovalReference, &g.ApprovedAt)
		})
	if err != nil {
		return Matrix{}, fmt.Errorf("load grants: %w", err)
	}
	perms, err := collect(ctx, s.pool, `SELECT key, description, sensitive FROM permissions ORDER BY key`, nil,
		func(row pgx.Rows) (PermissionInfo, error) {
			var p PermissionInfo
			return p, row.Scan(&p.Key, &p.Description, &p.Sensitive)
		})
	if err != nil {
		return Matrix{}, fmt.Errorf("load permissions: %w", err)
	}

	m := Matrix{MatrixStatus: MatrixPending, Roles: make([]MatrixRole, 0, len(roles)), Permissions: perms}
	for _, r := range roles {
		mr := r.MatrixRole
		mr.Grants = []Grant{}
		for _, g := range grants {
			if g.roleID == r.id {
				mr.Grants = append(mr.Grants, g.Grant)
				if g.Status == GrantApproved {
					m.ApprovedGrantCount++
				}
			}
		}
		m.Roles = append(m.Roles, mr)
	}
	if m.ApprovedGrantCount > 0 {
		m.MatrixStatus = MatrixPartiallyApproved
	}
	return m, nil
}

func collect[T any](ctx context.Context, pool *pgxpool.Pool, sql string, args []any, scan func(pgx.Rows) (T, error)) ([]T, error) {
	rows, err := pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []T{}
	for rows.Next() {
		v, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// ── Administration (cmd/access); every change is audited in its transaction ──

func lookup(ctx context.Context, tx pgx.Tx, sql, value string) (string, error) {
	var id string
	err := tx.QueryRow(ctx, sql, value).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return id, err
}

func roleID(ctx context.Context, tx pgx.Tx, key string) (string, error) {
	id, err := lookup(ctx, tx, `SELECT id::text FROM roles WHERE key = $1`, key)
	if errors.Is(err, ErrNotFound) {
		return "", fmt.Errorf("role %q: %w", key, ErrNotFound)
	}
	return id, err
}

func userID(ctx context.Context, tx pgx.Tx, username string) (string, error) {
	id, err := lookup(ctx, tx, `SELECT id::text FROM users WHERE username = $1`, username)
	if errors.Is(err, ErrNotFound) {
		return "", fmt.Errorf("user %q: %w", username, ErrNotFound)
	}
	return id, err
}

func locationID(ctx context.Context, tx pgx.Tx, code string) (string, error) {
	id, err := lookup(ctx, tx, `SELECT id::text FROM locations WHERE code = $1`, code)
	if errors.Is(err, ErrNotFound) {
		return "", fmt.Errorf("location %q: %w", code, ErrNotFound)
	}
	return id, err
}

// Grant records a role → permission grant. Without approval it is PROPOSED
// (does not count outside development); with the owner's approval evidence it
// is APPROVED. An approved grant is never silently downgraded.
func (s *PGStore) Grant(ctx context.Context, actor audit.Actor, roleKey, permission string, approval *Approval) (string, error) {
	status := GrantProposed
	if approval != nil {
		status = GrantApproved
	}
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		rid, err := roleID(ctx, tx, roleKey)
		if err != nil {
			return err
		}
		if _, err := lookup(ctx, tx, `SELECT key FROM permissions WHERE key = $1`, permission); err != nil {
			if errors.Is(err, ErrNotFound) {
				return fmt.Errorf("permission %q: %w", permission, ErrNotFound)
			}
			return err
		}
		var previous string
		err = tx.QueryRow(ctx, `SELECT status FROM role_permissions WHERE role_id = $1 AND permission_key = $2 FOR UPDATE`,
			rid, permission).Scan(&previous)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		switch {
		case previous == GrantApproved && status == GrantProposed:
			return ErrAlreadyApproved
		case previous == status && status == GrantProposed:
			return ErrNoChange
		}
		var by, ref any
		var at any
		if approval != nil {
			by, ref, at = approval.ApprovedBy, approval.Reference, time.Now().UTC()
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO role_permissions (role_id, permission_key, status, approved_by, approval_reference, approved_at)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (role_id, permission_key) DO UPDATE
			SET status = EXCLUDED.status, approved_by = EXCLUDED.approved_by,
			    approval_reference = EXCLUDED.approval_reference, approved_at = EXCLUDED.approved_at`,
			rid, permission, status, by, ref, at); err != nil {
			return fmt.Errorf("save grant: %w", err)
		}
		summary := audit.Summary{"role": roleKey, "permission": permission,
			"status": audit.Change{From: nullIfEmpty(previous), To: status}}
		if approval != nil {
			summary["approvedBy"] = approval.ApprovedBy
			summary["approvalReference"] = approval.Reference
		}
		action := "access.grant_proposed"
		if status == GrantApproved {
			action = "access.grant_approved"
		}
		return audit.Record(ctx, tx, audit.Event{Actor: actor, Action: action, EntityType: "role", EntityID: rid, Summary: summary})
	})
	return status, err
}

// Revoke removes a grant.
func (s *PGStore) Revoke(ctx context.Context, actor audit.Actor, roleKey, permission string) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		rid, err := roleID(ctx, tx, roleKey)
		if err != nil {
			return err
		}
		var previous string
		err = tx.QueryRow(ctx, `DELETE FROM role_permissions WHERE role_id = $1 AND permission_key = $2 RETURNING status`,
			rid, permission).Scan(&previous)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNoChange
		}
		if err != nil {
			return err
		}
		return audit.Record(ctx, tx, audit.Event{Actor: actor, Action: "access.grant_revoked", EntityType: "role", EntityID: rid,
			Summary: audit.Summary{"role": roleKey, "permission": permission, "status": audit.Change{From: previous, To: nil}}})
	})
}

// AssignRole gives a user a role; UnassignRole removes it.
func (s *PGStore) AssignRole(ctx context.Context, actor audit.Actor, username, roleKey string) error {
	return s.userRole(ctx, actor, username, roleKey, true)
}

// UnassignRole removes a role from a user.
func (s *PGStore) UnassignRole(ctx context.Context, actor audit.Actor, username, roleKey string) error {
	return s.userRole(ctx, actor, username, roleKey, false)
}

func (s *PGStore) userRole(ctx context.Context, actor audit.Actor, username, roleKey string, assign bool) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		uid, err := userID(ctx, tx, username)
		if err != nil {
			return err
		}
		rid, err := roleID(ctx, tx, roleKey)
		if err != nil {
			return err
		}
		sql, action := `INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, "access.role_assigned"
		if !assign {
			sql, action = `DELETE FROM user_roles WHERE user_id = $1 AND role_id = $2`, "access.role_unassigned"
		}
		tag, err := tx.Exec(ctx, sql, uid, rid)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return ErrNoChange
		}
		return audit.Record(ctx, tx, audit.Event{Actor: actor, Action: action, EntityType: "user", EntityID: uid,
			Summary: audit.Summary{"username": username, "role": roleKey}})
	})
}

// CreateLocation adds a location.
func (s *PGStore) CreateLocation(ctx context.Context, actor audit.Actor, code, name string) (Location, error) {
	l := Location{Code: code, Name: name}
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `INSERT INTO locations (code, name) VALUES ($1, $2) RETURNING id::text`, code, name).Scan(&l.ID); err != nil {
			return err
		}
		return audit.Record(ctx, tx, audit.Event{Actor: actor, Action: "location.created", EntityType: "location", EntityID: l.ID,
			Reference: code, Summary: audit.Summary{"code": code, "name": name}})
	})
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return Location{}, fmt.Errorf("location %q: %w", code, ErrDuplicate)
	}
	return l, err
}

// AddLocationScope lets a user work in a location; RemoveLocationScope takes it away.
func (s *PGStore) AddLocationScope(ctx context.Context, actor audit.Actor, username, code string) error {
	return s.locationScope(ctx, actor, username, code, true)
}

// RemoveLocationScope removes a location from a user's scope.
func (s *PGStore) RemoveLocationScope(ctx context.Context, actor audit.Actor, username, code string) error {
	return s.locationScope(ctx, actor, username, code, false)
}

func (s *PGStore) locationScope(ctx context.Context, actor audit.Actor, username, code string, add bool) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		uid, err := userID(ctx, tx, username)
		if err != nil {
			return err
		}
		lid, err := locationID(ctx, tx, code)
		if err != nil {
			return err
		}
		sql, action := `INSERT INTO user_location_scopes (user_id, location_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, "access.location_scope_added"
		if !add {
			sql, action = `DELETE FROM user_location_scopes WHERE user_id = $1 AND location_id = $2`, "access.location_scope_removed"
		}
		tag, err := tx.Exec(ctx, sql, uid, lid)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return ErrNoChange
		}
		return audit.Record(ctx, tx, audit.Event{Actor: actor, Action: action, EntityType: "user", EntityID: uid,
			Summary: audit.Summary{"username": username, "location": code}})
	})
}

// UserIDByName resolves a username (for cmd/access show).
func (s *PGStore) UserIDByName(ctx context.Context, username string) (string, error) {
	var id string
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		var err error
		id, err = userID(ctx, tx, username)
		return err
	})
	return id, err
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
