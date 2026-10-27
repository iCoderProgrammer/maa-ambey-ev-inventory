package stockitems

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/audit"
)

// PGStore is the PostgreSQL Repository. Each mutation is one transaction that
// locks what it changes, checks the version and state, writes and audits.
type PGStore struct{ pool *pgxpool.Pool }

// NewPGStore returns a Repository backed by pool.
func NewPGStore(pool *pgxpool.Pool) *PGStore { return &PGStore{pool: pool} }

type querier interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

const categoryColumns = `c.id::text, c.name, c.description, c.status, c.archived_by::text, c.archived_at, c.archive_reason,
	(SELECT count(*) FROM stock_items i WHERE i.category_id = c.id AND i.status = 'ACTIVE')::int,
	c.created_by::text, c.version, c.created_at, c.updated_at`

func scanCategory(row pgx.Row) (Category, error) {
	var c Category
	err := row.Scan(&c.ID, &c.Name, &c.Description, &c.Status, &c.ArchivedBy, &c.ArchivedAt, &c.ArchiveReason,
		&c.ActiveItemCount, &c.CreatedBy, &c.Version, &c.CreatedAt, &c.UpdatedAt)
	return c, err
}

const itemColumns = `i.id::text, i.sku, i.sku_normalised, i.name, i.description, i.category_id::text, c.name, c.status, i.uom,
	i.tracking_mode, i.reorder_level::text, i.status, i.archived_by::text, i.archived_at, i.archive_reason,
	i.created_by::text, i.version, i.created_at, i.updated_at`

const itemFrom = ` FROM stock_items i JOIN stock_item_categories c ON c.id = i.category_id`

func scanItem(row pgx.Row) (Item, error) {
	var it Item
	err := row.Scan(&it.ID, &it.SKU, &it.SKUNormalised, &it.Name, &it.Description, &it.CategoryID, &it.CategoryName,
		&it.CategoryStatus, &it.UOM, &it.TrackingMode, &it.ReorderLevel, &it.Status, &it.ArchivedBy, &it.ArchivedAt,
		&it.ArchiveReason, &it.CreatedBy, &it.Version, &it.CreatedAt, &it.UpdatedAt)
	return it, err
}

func scanLink(row pgx.Row) (SupplierLink, error) {
	var l SupplierLink
	err := row.Scan(&l.ID, &l.SupplierID, &l.SupplierName, &l.SupplierStatus, &l.SupplierPartNumber, &l.CreatedBy,
		&l.CreatedAt, &l.EndedBy, &l.EndedAt)
	return l, err
}

func scanCompatibility(row pgx.Row) (Compatibility, error) {
	var c Compatibility
	err := row.Scan(&c.ID, &c.BrandID, &c.BrandName, &c.BrandStatus, &c.ModelID, &c.ModelName)
	return c, err
}

// Only these unique indexes mean "already used"; any other violation stays an internal error.
var duplicateFields = map[string]string{
	"stock_items_sku_key":            "sku",
	"stock_item_categories_name_key": "name",
}

// Named rule violations raised by the reference-check trigger (normally caught
// earlier with a friendlier message; these decide under concurrency).
var ruleErrors = map[string]error{
	"stock_items_category_active":             invalid("categoryId", "is archived"),
	"stock_items_active":                      &StateError{Message: "Archived SKUs cannot change."},
	"stock_item_suppliers_supplier_active":    invalid("suppliers", "includes an archived supplier"),
	"stock_item_compatibility_brand_active":   invalid("compatibility", "includes an archived brand"),
	"stock_item_compatibility_model_active":   invalid("compatibility", "includes an archived model"),
	"stock_item_categories_no_active_items":   &StateError{Message: "This category still has active SKUs. Move or archive them first."},
	"stock_item_compatibility_model_brand_fk": invalid("compatibility", "includes a model of another brand"),
}

func mapPGError(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}
	if e, ok := ruleErrors[pgErr.ConstraintName]; ok && (pgErr.Code == "23514" || pgErr.Code == "23503") {
		return e
	}
	switch pgErr.Code {
	case "23505":
		if f, ok := duplicateFields[pgErr.ConstraintName]; ok {
			return &DuplicateError{Field: f}
		}
	case "23514":
		return invalid("body", "violates stock item rule "+pgErr.ConstraintName)
	case "42501":
		return &StateError{Message: "This record is history and cannot change."}
	}
	return err
}

func (s *PGStore) tx(ctx context.Context, fn func(pgx.Tx) error) error {
	return mapPGError(pgx.BeginFunc(ctx, s.pool, fn))
}

func str(p *string) any {
	if p == nil {
		return nil
	}
	return *p
}

func withSummary(ev audit.Event, sum audit.Summary) audit.Event {
	if ev.Summary == nil {
		ev.Summary = audit.Summary{}
	}
	for k, v := range sum {
		ev.Summary[k] = v
	}
	return ev
}

func queryAll[T any](ctx context.Context, q querier, scan func(pgx.Row) (T, error), sql string, args ...any) ([]T, error) {
	rows, err := q.Query(ctx, sql, args...)
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

func likePattern(q string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return "%" + r.Replace(q) + "%"
}

type conds struct {
	where []string
	args  []any
}

func (c *conds) add(cond string, v any) {
	c.args = append(c.args, v)
	c.where = append(c.where, strings.ReplaceAll(cond, "?", "$"+strconv.Itoa(len(c.args))))
}

func (c *conds) sql() string {
	if len(c.where) == 0 {
		return ""
	}
	return " WHERE " + strings.Join(c.where, " AND ")
}

func (c *conds) paging(page, limit int) string {
	n := len(c.args)
	c.args = append(c.args, limit, (page-1)*limit)
	return ` LIMIT $` + strconv.Itoa(n+1) + ` OFFSET $` + strconv.Itoa(n+2)
}

// ── Categories ───────────────────────────────────────────────────────────────

// ListCategories implements Repository.
func (s *PGStore) ListCategories(ctx context.Context, f CategoryFilter) (Page[Category], error) {
	var c conds
	if f.Query != "" {
		c.add(`c.name ILIKE ? ESCAPE '\'`, likePattern(f.Query))
	}
	if f.Status != "" {
		c.add("c.status = ?", f.Status)
	}
	var total int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM stock_item_categories c`+c.sql(), c.args...).Scan(&total); err != nil {
		return Page[Category]{}, fmt.Errorf("count categories: %w", err)
	}
	where := c.sql()
	items, err := queryAll(ctx, s.pool, scanCategory, `SELECT `+categoryColumns+` FROM stock_item_categories c`+where+
		` ORDER BY lower(c.name), c.id`+c.paging(f.Page, f.Limit), c.args...)
	if err != nil {
		return Page[Category]{}, fmt.Errorf("list categories: %w", err)
	}
	return Page[Category]{Items: items, Page: f.Page, Limit: f.Limit, Total: total}, nil
}

func getCategory(ctx context.Context, q querier, id string) (Category, error) {
	return scanCategory(q.QueryRow(ctx, `SELECT `+categoryColumns+` FROM stock_item_categories c WHERE c.id = $1`, id))
}

func lockCategory(ctx context.Context, tx pgx.Tx, id string, version int) (Category, error) {
	c, err := scanCategory(tx.QueryRow(ctx, `SELECT `+categoryColumns+` FROM stock_item_categories c WHERE c.id = $1 FOR UPDATE OF c`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return c, ErrNotFound
	}
	if err != nil {
		return c, err
	}
	if c.Version != version {
		return c, ErrVersionConflict
	}
	if c.Status == StatusArchived {
		return c, &StateError{Message: "Archived categories cannot change."}
	}
	return c, nil
}

// CreateCategory implements Repository.
func (s *PGStore) CreateCategory(ctx context.Context, in CategoryInput, createdBy string, ev audit.Event) (Category, error) {
	var c Category
	err := s.tx(ctx, func(tx pgx.Tx) error {
		var id string
		if err := tx.QueryRow(ctx, `INSERT INTO stock_item_categories (name, description, created_by) VALUES ($1, $2, $3)
			RETURNING id::text`, in.Name, str(in.Description), createdBy).Scan(&id); err != nil {
			return err
		}
		var err error
		if c, err = getCategory(ctx, tx, id); err != nil {
			return err
		}
		ev.EntityID, ev.Reference = id, c.Name
		return audit.Record(ctx, tx, withSummary(ev, audit.Summary{"name": c.Name}))
	})
	return c, err
}

// UpdateCategory implements Repository.
func (s *PGStore) UpdateCategory(ctx context.Context, id string, p CategoryPatch, ev audit.Event) (Category, error) {
	var c Category
	err := s.tx(ctx, func(tx pgx.Tx) error {
		old, err := lockCategory(ctx, tx, id, p.Version)
		if err != nil {
			return err
		}
		args := []any{id}
		sets := []string{"version = version + 1"}
		sum := audit.Summary{}
		if p.Name.Set {
			args = append(args, *p.Name.Value)
			sets = append(sets, "name = $"+strconv.Itoa(len(args)))
			sum["name"] = audit.Change{From: old.Name, To: *p.Name.Value}
		}
		if p.Description.Set {
			args = append(args, str(p.Description.Value))
			sets = append(sets, "description = $"+strconv.Itoa(len(args)))
			sum["description"] = "changed"
		}
		if _, err := tx.Exec(ctx, `UPDATE stock_item_categories SET `+strings.Join(sets, ", ")+` WHERE id = $1`, args...); err != nil {
			return err
		}
		if c, err = getCategory(ctx, tx, id); err != nil {
			return err
		}
		ev.Reference = c.Name
		return audit.Record(ctx, tx, withSummary(ev, sum))
	})
	return c, err
}

// ArchiveCategory implements Repository. A category with ACTIVE SKUs stays
// active (the database re-checks under the category row lock).
func (s *PGStore) ArchiveCategory(ctx context.Context, id string, in ArchiveInput, actorID string, ev audit.Event) (Category, error) {
	var c Category
	err := s.tx(ctx, func(tx pgx.Tx) error {
		old, err := lockCategory(ctx, tx, id, in.Version)
		if err != nil {
			return err
		}
		if old.ActiveItemCount > 0 {
			return &StateError{Message: "This category still has active SKUs. Move or archive them first."}
		}
		if _, err := tx.Exec(ctx, `UPDATE stock_item_categories SET status = 'ARCHIVED', archived_by = $2, archived_at = now(),
			archive_reason = $3, version = version + 1 WHERE id = $1`, id, actorID, in.Reason); err != nil {
			return err
		}
		if c, err = getCategory(ctx, tx, id); err != nil {
			return err
		}
		ev.Reference = c.Name
		return audit.Record(ctx, tx, withSummary(ev, audit.Summary{"reason": in.Reason,
			"status": audit.Change{From: StatusActive, To: StatusArchived}}))
	})
	return c, err
}

// ── Items ────────────────────────────────────────────────────────────────────

// ListItems implements Repository. q matches the SKU code (normalised) and the name.
func (s *PGStore) ListItems(ctx context.Context, f ListFilter) (Page[Item], error) {
	var c conds
	if f.Query != "" {
		c.add(`(i.sku_normalised LIKE upper(?) ESCAPE '\' OR i.name ILIKE ? ESCAPE '\')`, likePattern(f.Query))
	}
	for _, x := range []struct{ cond, v string }{{"i.status = ?", f.Status}, {"i.category_id = ?", f.CategoryID},
		{"i.tracking_mode = ?", f.TrackingMode},
		{"EXISTS (SELECT 1 FROM stock_item_suppliers l WHERE l.stock_item_id = i.id AND l.ended_at IS NULL AND l.supplier_id = ?)", f.SupplierID},
		{"EXISTS (SELECT 1 FROM stock_item_compatibility m WHERE m.stock_item_id = i.id AND m.brand_id = ?)", f.BrandID}} {
		if x.v != "" {
			c.add(x.cond, x.v)
		}
	}
	where := c.sql()
	var total int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM stock_items i`+where, c.args...).Scan(&total); err != nil {
		return Page[Item]{}, fmt.Errorf("count stock items: %w", err)
	}
	items, err := queryAll(ctx, s.pool, scanItem, `SELECT `+itemColumns+itemFrom+where+` ORDER BY i.sku_normalised, i.id`+
		c.paging(f.Page, f.Limit), c.args...)
	if err != nil {
		return Page[Item]{}, fmt.Errorf("list stock items: %w", err)
	}
	return Page[Item]{Items: items, Page: f.Page, Limit: f.Limit, Total: total}, nil
}

func getDetail(ctx context.Context, q querier, id string) (Detail, error) {
	var d Detail
	it, err := scanItem(q.QueryRow(ctx, `SELECT `+itemColumns+itemFrom+` WHERE i.id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return d, ErrNotFound
	}
	if err != nil {
		return d, fmt.Errorf("get stock item: %w", err)
	}
	d.Item = it
	if d.Suppliers, err = queryAll(ctx, q, scanLink, `SELECT l.id::text, l.supplier_id::text, s.legal_name, s.status,
			l.supplier_part_number, l.created_by::text, l.created_at, l.ended_by::text, l.ended_at
		FROM stock_item_suppliers l JOIN suppliers s ON s.id = l.supplier_id WHERE l.stock_item_id = $1
		ORDER BY (l.ended_at IS NOT NULL), lower(s.legal_name), l.created_at DESC`, id); err != nil {
		return d, fmt.Errorf("stock item suppliers: %w", err)
	}
	if d.Compatibility, err = queryAll(ctx, q, scanCompatibility, `SELECT m.id::text, m.brand_id::text, b.name, b.lifecycle_status,
			m.model_id::text, mo.name
		FROM stock_item_compatibility m JOIN brands b ON b.id = m.brand_id LEFT JOIN models mo ON mo.id = m.model_id
		WHERE m.stock_item_id = $1 ORDER BY lower(b.name), (m.model_id IS NOT NULL), lower(mo.name)`, id); err != nil {
		return d, fmt.Errorf("stock item compatibility: %w", err)
	}
	return d, nil
}

// GetItem implements Repository.
func (s *PGStore) GetItem(ctx context.Context, id string) (Detail, error) {
	return getDetail(ctx, s.pool, id)
}

// checkCategory: the category exists and is ACTIVE.
func checkCategory(ctx context.Context, tx pgx.Tx, id string) error {
	var st string
	err := tx.QueryRow(ctx, `SELECT status FROM stock_item_categories WHERE id = $1`, id).Scan(&st)
	if errors.Is(err, pgx.ErrNoRows) {
		return invalid("categoryId", "does not exist")
	}
	if err != nil {
		return err
	}
	if st != StatusActive {
		return invalid("categoryId", "is archived")
	}
	return nil
}

// checkLinks validates every named supplier, brand and model, reporting each
// problem on its own entry. The model must belong to the named brand.
func checkLinks(ctx context.Context, tx pgx.Tx, sups []SupplierLinkInput, comp []CompatibilityInput) error {
	var c checker
	for i, l := range sups {
		f := "suppliers[" + strconv.Itoa(i) + "].supplierId"
		var st string
		err := tx.QueryRow(ctx, `SELECT status FROM suppliers WHERE id = $1`, l.SupplierID).Scan(&st)
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			c.add(f, "does not exist")
		case err != nil:
			return err
		case st == "ARCHIVED":
			c.add(f, "is archived")
		}
	}
	for i, m := range comp {
		f := "compatibility[" + strconv.Itoa(i) + "]"
		var st string
		err := tx.QueryRow(ctx, `SELECT lifecycle_status FROM brands WHERE id = $1`, m.BrandID).Scan(&st)
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			c.add(f+".brandId", "does not exist")
			continue
		case err != nil:
			return err
		case st == "ARCHIVED":
			c.add(f+".brandId", "is archived")
		}
		if m.ModelID == nil {
			continue
		}
		var brand string
		err = tx.QueryRow(ctx, `SELECT brand_id::text, lifecycle_status FROM models WHERE id = $1`, *m.ModelID).Scan(&brand, &st)
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			c.add(f+".modelId", "does not exist")
		case err != nil:
			return err
		case brand != m.BrandID:
			c.add(f+".modelId", "belongs to another brand")
		case st == "ARCHIVED":
			c.add(f+".modelId", "is archived")
		}
	}
	return c.err()
}

func insertLinks(ctx context.Context, tx pgx.Tx, itemID, actorID string, sups []SupplierLinkInput) error {
	for _, l := range sups {
		if _, err := tx.Exec(ctx, `INSERT INTO stock_item_suppliers (stock_item_id, supplier_id, supplier_part_number, created_by)
			VALUES ($1, $2, $3, $4)`, itemID, l.SupplierID, str(l.SupplierPartNumber), actorID); err != nil {
			return err
		}
	}
	return nil
}

func insertCompatibility(ctx context.Context, tx pgx.Tx, itemID, actorID string, comp []CompatibilityInput) error {
	for _, m := range comp {
		if _, err := tx.Exec(ctx, `INSERT INTO stock_item_compatibility (stock_item_id, brand_id, model_id, created_by)
			VALUES ($1, $2, $3, $4)`, itemID, m.BrandID, str(m.ModelID), actorID); err != nil {
			return err
		}
	}
	return nil
}

func supplierIDs(l []SupplierLinkInput) []string {
	out := make([]string, len(l))
	for i, x := range l {
		out[i] = x.SupplierID
	}
	slices.Sort(out)
	return out
}

func compatibilityKeys(l []CompatibilityInput) []string {
	out := make([]string, len(l))
	for i, x := range l {
		out[i] = x.BrandID + "/" + fmt.Sprint(str(x.ModelID))
	}
	slices.Sort(out)
	return out
}

// CreateItem implements Repository: the SKU with its supplier links and
// compatibility. No stock is recorded.
func (s *PGStore) CreateItem(ctx context.Context, in ItemInput, createdBy string, ev audit.Event) (Detail, error) {
	var d Detail
	err := s.tx(ctx, func(tx pgx.Tx) error {
		if err := checkCategory(ctx, tx, in.CategoryID); err != nil {
			return err
		}
		if err := checkLinks(ctx, tx, in.Suppliers, in.Compatibility); err != nil {
			return err
		}
		// Friendly early answer; the unique index decides when two requests race.
		var taken bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM stock_items WHERE sku_normalised = upper($1))`, in.SKU).Scan(&taken); err != nil {
			return err
		}
		if taken {
			return &DuplicateError{Field: "sku"}
		}
		var id string
		if err := tx.QueryRow(ctx, `INSERT INTO stock_items (sku, name, description, category_id, uom, tracking_mode, reorder_level, created_by)
			VALUES ($1, $2, $3, $4, $5, $6, $7::numeric, $8) RETURNING id::text`, in.SKU, in.Name, str(in.Description), in.CategoryID,
			in.UOM, in.TrackingMode, str(in.ReorderLevel), createdBy).Scan(&id); err != nil {
			return err
		}
		if err := insertLinks(ctx, tx, id, createdBy, in.Suppliers); err != nil {
			return err
		}
		if err := insertCompatibility(ctx, tx, id, createdBy, in.Compatibility); err != nil {
			return err
		}
		var err error
		if d, err = getDetail(ctx, tx, id); err != nil {
			return err
		}
		ev.EntityID, ev.Reference = id, d.SKU
		return audit.Record(ctx, tx, withSummary(ev, audit.Summary{"sku": d.SKU, "name": d.Name, "categoryId": d.CategoryID,
			"uom": d.UOM, "trackingMode": d.TrackingMode, "reorderLevel": str(d.ReorderLevel),
			"supplierIds": supplierIDs(in.Suppliers), "compatibility": compatibilityKeys(in.Compatibility)}))
	})
	return d, err
}

func lockItem(ctx context.Context, tx pgx.Tx, id string, version int) (Item, error) {
	it, err := scanItem(tx.QueryRow(ctx, `SELECT `+itemColumns+itemFrom+` WHERE i.id = $1 FOR UPDATE OF i`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return it, ErrNotFound
	}
	if err != nil {
		return it, err
	}
	if it.Version != version {
		return it, ErrVersionConflict
	}
	if it.Status == StatusArchived {
		return it, &StateError{Message: "Archived SKUs cannot change."}
	}
	return it, nil
}

type openLink struct{ id, supplierID string }

// replaceLinks ends the open links that are not in the new set (or whose part
// number changed) and adds the new ones. Ended links stay as history.
func replaceLinks(ctx context.Context, tx pgx.Tx, itemID, actorID string, next []SupplierLinkInput) (bool, error) {
	rows, err := tx.Query(ctx, `SELECT id::text, supplier_id::text, coalesce(supplier_part_number, '') FROM stock_item_suppliers
		WHERE stock_item_id = $1 AND ended_at IS NULL`, itemID)
	if err != nil {
		return false, err
	}
	current := map[string]openLink{}
	parts := map[string]string{}
	for rows.Next() {
		var l openLink
		var part string
		if err := rows.Scan(&l.id, &l.supplierID, &part); err != nil {
			rows.Close()
			return false, err
		}
		current[l.supplierID], parts[l.supplierID] = l, part
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return false, err
	}
	wanted := map[string]string{}
	var add []SupplierLinkInput
	for _, l := range next {
		part := ""
		if l.SupplierPartNumber != nil {
			part = *l.SupplierPartNumber
		}
		wanted[l.SupplierID] = part
		if old, ok := parts[l.SupplierID]; !ok || old != part {
			add = append(add, l)
		}
	}
	changed := len(add) > 0
	for sid, l := range current {
		if part, ok := wanted[sid]; !ok || part != parts[sid] {
			if _, err := tx.Exec(ctx, `UPDATE stock_item_suppliers SET ended_by = $2, ended_at = now() WHERE id = $1`, l.id, actorID); err != nil {
				return false, err
			}
			changed = true
		}
	}
	return changed, insertLinks(ctx, tx, itemID, actorID, add)
}

// replaceCompatibility swaps the set when it differs (audited by the caller).
func replaceCompatibility(ctx context.Context, tx pgx.Tx, itemID, actorID string, next []CompatibilityInput) (bool, error) {
	rows, err := tx.Query(ctx, `SELECT brand_id::text || '/' || coalesce(model_id::text, '<nil>') FROM stock_item_compatibility
		WHERE stock_item_id = $1`, itemID)
	if err != nil {
		return false, err
	}
	var cur []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			rows.Close()
			return false, err
		}
		cur = append(cur, k)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return false, err
	}
	slices.Sort(cur)
	if slices.Equal(cur, compatibilityKeys(next)) {
		return false, nil
	}
	if _, err := tx.Exec(ctx, `DELETE FROM stock_item_compatibility WHERE stock_item_id = $1`, itemID); err != nil {
		return false, err
	}
	return true, insertCompatibility(ctx, tx, itemID, actorID, next)
}

// UpdateItem implements Repository. The reorder level is checked against the
// stored tracking mode; the SKU code and tracking mode cannot change.
func (s *PGStore) UpdateItem(ctx context.Context, id string, p ItemPatch, actorID string, ev audit.Event) (Detail, error) {
	var d Detail
	err := s.tx(ctx, func(tx pgx.Tx) error {
		old, err := lockItem(ctx, tx, id, p.Version)
		if err != nil {
			return err
		}
		if p.ReorderLevel.Set && p.ReorderLevel.Value != nil && old.TrackingMode == TrackingSerial && !wholeNumber(*p.ReorderLevel.Value) {
			return invalid("reorderLevel", "must be a whole number for a serial-tracked item")
		}
		if p.CategoryID.Set && *p.CategoryID.Value != old.CategoryID {
			if err := checkCategory(ctx, tx, *p.CategoryID.Value); err != nil {
				return err
			}
		}
		var sups []SupplierLinkInput
		var comp []CompatibilityInput
		if p.Suppliers != nil {
			sups = *p.Suppliers
		}
		if p.Compatibility != nil {
			comp = *p.Compatibility
		}
		if err := checkLinks(ctx, tx, sups, comp); err != nil {
			return err
		}
		args := []any{id}
		sets := []string{"version = version + 1"}
		sum := audit.Summary{}
		set := func(field, column, cast string, o OptString, from *string) {
			if !o.Set {
				return
			}
			args = append(args, str(o.Value))
			sets = append(sets, column+" = $"+strconv.Itoa(len(args))+cast)
			if field == "description" {
				sum[field] = "changed"
			} else {
				sum[field] = audit.Change{From: str(from), To: str(o.Value)}
			}
		}
		name, category, uom := old.Name, old.CategoryID, old.UOM
		set("name", "name", "", p.Name, &name)
		set("description", "description", "", p.Description, old.Description)
		set("categoryId", "category_id", "::uuid", p.CategoryID, &category)
		set("uom", "uom", "", p.UOM, &uom)
		set("reorderLevel", "reorder_level", "::numeric", p.ReorderLevel, old.ReorderLevel)
		if _, err := tx.Exec(ctx, `UPDATE stock_items SET `+strings.Join(sets, ", ")+` WHERE id = $1`, args...); err != nil {
			return err
		}
		if p.Suppliers != nil {
			changed, err := replaceLinks(ctx, tx, id, actorID, sups)
			if err != nil {
				return err
			}
			if changed {
				sum["supplierIds"] = supplierIDs(sups)
			}
		}
		if p.Compatibility != nil {
			changed, err := replaceCompatibility(ctx, tx, id, actorID, comp)
			if err != nil {
				return err
			}
			if changed {
				sum["compatibility"] = compatibilityKeys(comp)
			}
		}
		if d, err = getDetail(ctx, tx, id); err != nil {
			return err
		}
		ev.Reference = d.SKU
		return audit.Record(ctx, tx, withSummary(ev, sum))
	})
	return d, err
}

// ArchiveItem implements Repository. The SKU keeps its code, supplier links,
// compatibility and references; it can no longer change.
func (s *PGStore) ArchiveItem(ctx context.Context, id string, in ArchiveInput, actorID string, ev audit.Event) (Detail, error) {
	var d Detail
	err := s.tx(ctx, func(tx pgx.Tx) error {
		if _, err := lockItem(ctx, tx, id, in.Version); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE stock_items SET status = 'ARCHIVED', archived_by = $2, archived_at = now(), archive_reason = $3,
			version = version + 1 WHERE id = $1`, id, actorID, in.Reason); err != nil {
			return err
		}
		var err error
		if d, err = getDetail(ctx, tx, id); err != nil {
			return err
		}
		ev.Reference = d.SKU
		return audit.Record(ctx, tx, withSummary(ev, audit.Summary{"reason": in.Reason,
			"status": audit.Change{From: StatusActive, To: StatusArchived}}))
	})
	return d, err
}
