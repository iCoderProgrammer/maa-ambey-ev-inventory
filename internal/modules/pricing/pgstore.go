package pricing

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/audit"
)

// PGStore is the PostgreSQL Repository. Mutations run in one transaction that
// locks what they depend on, checks the version and the rules, writes and
// records the audit event.
type PGStore struct {
	pool *pgxpool.Pool
}

// NewPGStore returns a Repository backed by pool.
func NewPGStore(pool *pgxpool.Pool) *PGStore { return &PGStore{pool: pool} }

const listColumns = `id::text, brand_id::text, name, currency, lifecycle_status, created_by::text, version, created_at, updated_at`

const itemColumns = `id::text, price_list_id::text, brand_id::text, model_id::text, variant_id::text, price_type,
	amount::text, currency, effective_from, effective_to, status, tax_configuration_id::text, source_note,
	created_by::text, approved_by::text, approved_at, approval_reference, supersedes_id::text, superseded_by_id::text,
	version, created_at, updated_at`

func scanList(row pgx.Row) (PriceList, error) {
	var l PriceList
	err := row.Scan(&l.ID, &l.BrandID, &l.Name, &l.Currency, &l.LifecycleStatus, &l.CreatedBy, &l.Version, &l.CreatedAt, &l.UpdatedAt)
	return l, err
}

func scanItem(row pgx.Row) (Item, error) {
	var it Item
	err := row.Scan(&it.ID, &it.PriceListID, &it.BrandID, &it.ModelID, &it.VariantID, &it.PriceType,
		&it.Amount, &it.Currency, &it.EffectiveFrom, &it.EffectiveTo, &it.Status, &it.TaxConfigurationID, &it.SourceNote,
		&it.CreatedBy, &it.ApprovedBy, &it.ApprovedAt, &it.ApprovalReference, &it.SupersedesID, &it.SupersededByID,
		&it.Version, &it.CreatedAt, &it.UpdatedAt)
	return it, err
}

func mapPGError(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}
	switch pgErr.Code {
	case "23505":
		if pgErr.ConstraintName == "price_lists_brand_name_key" {
			return &DuplicateError{Field: "name"}
		}
	case "23P01":
		// Exclusion constraint: a concurrent approval won the period.
		return &OverlapError{}
	case "42501":
		return &StateError{Message: "Approved prices are immutable; draft a superseding price instead."}
	}
	return err
}

func (s *PGStore) tx(ctx context.Context, fn func(pgx.Tx) error) error {
	return mapPGError(pgx.BeginFunc(ctx, s.pool, fn))
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

func timeOrNil(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.UTC().Format(time.RFC3339)
}

func strOrNil(p *string) any {
	if p == nil {
		return nil
	}
	return *p
}

// ── Lists ────────────────────────────────────────────────────────────────────

type where struct {
	conds []string
	args  []any
}

func (w *where) add(cond string, args ...any) {
	for _, a := range args {
		w.args = append(w.args, a)
		cond = strings.Replace(cond, "?", "$"+strconv.Itoa(len(w.args)), 1)
	}
	w.conds = append(w.conds, cond)
}

func (w *where) sql() string {
	if len(w.conds) == 0 {
		return ""
	}
	return " WHERE " + strings.Join(w.conds, " AND ")
}

func page[T any](ctx context.Context, s *PGStore, from, cols, order string, w where, limit, offset, pageNo int, scan func(pgx.Row) (T, error)) (Page[T], error) {
	var total int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM `+from+w.sql(), w.args...).Scan(&total); err != nil {
		return Page[T]{}, fmt.Errorf("count %s: %w", from, err)
	}
	n := len(w.args)
	rows, err := s.pool.Query(ctx, `SELECT `+cols+` FROM `+from+w.sql()+` ORDER BY `+order+
		` LIMIT $`+strconv.Itoa(n+1)+` OFFSET $`+strconv.Itoa(n+2), append(append([]any{}, w.args...), limit, offset)...)
	if err != nil {
		return Page[T]{}, fmt.Errorf("list %s: %w", from, err)
	}
	defer rows.Close()
	items := []T{}
	for rows.Next() {
		it, err := scan(rows)
		if err != nil {
			return Page[T]{}, err
		}
		items = append(items, it)
	}
	return Page[T]{Items: items, Page: pageNo, Limit: limit, Total: total}, rows.Err()
}

// ListPriceLists implements Repository.
func (s *PGStore) ListPriceLists(ctx context.Context, f ListFilter) (Page[PriceList], error) {
	var w where
	if f.BrandID != "" {
		w.add("brand_id = ?", f.BrandID)
	}
	if f.Status != "" {
		w.add("lifecycle_status = ?", f.Status)
	}
	if f.Query != "" {
		r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
		w.add(`name ILIKE ? ESCAPE '\'`, "%"+r.Replace(f.Query)+"%")
	}
	return page(ctx, s, "price_lists", listColumns, "lower(name), id", w, f.Limit, f.Offset(), f.Page, scanList)
}

// GetPriceList implements Repository.
func (s *PGStore) GetPriceList(ctx context.Context, id string) (PriceList, error) {
	l, err := scanList(s.pool.QueryRow(ctx, `SELECT `+listColumns+` FROM price_lists WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return PriceList{}, ErrNotFound
	}
	return l, err
}

// CreatePriceList implements Repository.
func (s *PGStore) CreatePriceList(ctx context.Context, in PriceListCreateRequest, createdBy string, ev audit.Event) (PriceList, error) {
	var l PriceList
	err := s.tx(ctx, func(tx pgx.Tx) error {
		var brandStatus string
		err := tx.QueryRow(ctx, `SELECT lifecycle_status FROM brands WHERE id = $1 FOR SHARE`, in.BrandID).Scan(&brandStatus)
		if errors.Is(err, pgx.ErrNoRows) {
			return invalid("brandId", "does not exist")
		}
		if err != nil {
			return err
		}
		if brandStatus == "ARCHIVED" {
			return &StateError{Message: "A price list cannot be added to an archived brand."}
		}
		l, err = scanList(tx.QueryRow(ctx, `INSERT INTO price_lists (brand_id, name, created_by) VALUES ($1, $2, $3)
			RETURNING `+listColumns, in.BrandID, in.Name, createdBy))
		if err != nil {
			return err
		}
		ev.EntityID = l.ID
		return audit.Record(ctx, tx, withSummary(ev, audit.Summary{"brandId": l.BrandID, "name": l.Name}))
	})
	return l, err
}

func lockList(ctx context.Context, tx pgx.Tx, id string, version int) (PriceList, error) {
	l, err := scanList(tx.QueryRow(ctx, `SELECT `+listColumns+` FROM price_lists WHERE id = $1 FOR UPDATE`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return l, ErrNotFound
	}
	if err != nil {
		return l, err
	}
	if l.Version != version {
		return l, ErrVersionConflict
	}
	return l, nil
}

// RenamePriceList implements Repository.
func (s *PGStore) RenamePriceList(ctx context.Context, id string, in PriceListUpdateRequest, ev audit.Event) (PriceList, error) {
	var l PriceList
	err := s.tx(ctx, func(tx pgx.Tx) error {
		old, err := lockList(ctx, tx, id, in.Version)
		if err != nil {
			return err
		}
		if old.LifecycleStatus == ListArchived {
			return &StateError{Message: "Archived price lists cannot be changed."}
		}
		l, err = scanList(tx.QueryRow(ctx, `UPDATE price_lists SET name = $2, version = version + 1 WHERE id = $1 RETURNING `+listColumns, id, in.Name))
		if err != nil {
			return err
		}
		return audit.Record(ctx, tx, withSummary(ev, audit.Summary{"name": audit.Change{From: old.Name, To: l.Name}}))
	})
	return l, err
}

// ArchivePriceList implements Repository. Approved history stays; nothing new is added.
func (s *PGStore) ArchivePriceList(ctx context.Context, id string, version int, ev audit.Event) (PriceList, error) {
	var l PriceList
	err := s.tx(ctx, func(tx pgx.Tx) error {
		old, err := lockList(ctx, tx, id, version)
		if err != nil {
			return err
		}
		if old.LifecycleStatus == ListArchived {
			return &StateError{Message: "This price list is already archived."}
		}
		l, err = scanList(tx.QueryRow(ctx, `UPDATE price_lists SET lifecycle_status = 'ARCHIVED', version = version + 1 WHERE id = $1 RETURNING `+listColumns, id))
		if err != nil {
			return err
		}
		return audit.Record(ctx, tx, withSummary(ev, audit.Summary{"lifecycleStatus": audit.Change{From: ListActive, To: ListArchived}}))
	})
	return l, err
}

// ── Items ────────────────────────────────────────────────────────────────────

// brandOf returns the brand of a price list or variant; unknown IDs are a
// validation error on field.
func (s *PGStore) brandOf(ctx context.Context, table, id, field string) (string, error) {
	var brand string
	err := s.pool.QueryRow(ctx, `SELECT brand_id::text FROM `+table+` WHERE id = $1`, id).Scan(&brand)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", invalid(field, "does not exist")
	}
	if err != nil {
		return "", fmt.Errorf("brand of %s: %w", field, err)
	}
	return brand, nil
}

// checkFilterBrands refuses filters that name a brand, price list and variant of
// different brands (C-06): a cross-brand mismatch fails instead of returning nothing.
func (s *PGStore) checkFilterBrands(ctx context.Context, brandID, priceListID, variantID string) error {
	brand := brandID
	for _, ref := range []struct{ table, id, field string }{
		{"price_lists", priceListID, "priceListId"},
		{"variants", variantID, "variantId"},
	} {
		if ref.id == "" {
			continue
		}
		b, err := s.brandOf(ctx, ref.table, ref.id, ref.field)
		if err != nil {
			return err
		}
		if brand == "" {
			brand = b
			continue
		}
		if b != brand {
			return invalid(ref.field, "belongs to another brand than the other filters")
		}
	}
	return nil
}

// ListItems implements Repository. f.Types limits the visible price types.
func (s *PGStore) ListItems(ctx context.Context, f ListFilter) (Page[Item], error) {
	if err := s.checkFilterBrands(ctx, f.BrandID, f.PriceListID, f.VariantID); err != nil {
		return Page[Item]{}, err
	}
	var w where
	w.add("price_type = ANY(?)", f.Types)
	if f.BrandID != "" {
		w.add("brand_id = ?", f.BrandID)
	}
	if f.PriceListID != "" {
		w.add("price_list_id = ?", f.PriceListID)
	}
	if f.VariantID != "" {
		w.add("variant_id = ?", f.VariantID)
	}
	if f.Status != "" {
		w.add("status = ?", f.Status)
	}
	return page(ctx, s, "price_items", itemColumns, "effective_from DESC, created_at DESC, id", w, f.Limit, f.Offset(), f.Page, scanItem)
}

// GetItem implements Repository.
func (s *PGStore) GetItem(ctx context.Context, id string) (Item, error) {
	it, err := scanItem(s.pool.QueryRow(ctx, `SELECT `+itemColumns+` FROM price_items WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Item{}, ErrNotFound
	}
	return it, err
}

// CreateItem implements Repository. Brand and model come from the variant; the
// list must belong to the same brand and be active.
func (s *PGStore) CreateItem(ctx context.Context, in itemInput, createdBy string, ev audit.Event) (Item, error) {
	var it Item
	err := s.tx(ctx, func(tx pgx.Tx) error {
		var listBrand, listStatus string
		err := tx.QueryRow(ctx, `SELECT brand_id::text, lifecycle_status FROM price_lists WHERE id = $1 FOR SHARE`, in.PriceListID).Scan(&listBrand, &listStatus)
		if errors.Is(err, pgx.ErrNoRows) {
			return invalid("priceListId", "does not exist")
		}
		if err != nil {
			return err
		}
		var brandID, modelID, variantStatus string
		err = tx.QueryRow(ctx, `SELECT brand_id::text, model_id::text, lifecycle_status FROM variants WHERE id = $1 FOR SHARE`, in.VariantID).
			Scan(&brandID, &modelID, &variantStatus)
		if errors.Is(err, pgx.ErrNoRows) {
			return invalid("variantId", "does not exist")
		}
		if err != nil {
			return err
		}
		if brandID != listBrand {
			return invalid("variantId", "belongs to another brand than the price list")
		}
		if listStatus != ListActive {
			return &StateError{Message: "Prices cannot be added to an archived price list."}
		}
		if variantStatus == "ARCHIVED" {
			return &StateError{Message: "Prices cannot be added for an archived variant."}
		}
		if in.SupersedesID != nil {
			var sList, sVariant, sType, sStatus string
			var sBy *string
			err := tx.QueryRow(ctx, `SELECT price_list_id::text, variant_id::text, price_type, status, superseded_by_id::text
				FROM price_items WHERE id = $1`, *in.SupersedesID).Scan(&sList, &sVariant, &sType, &sStatus, &sBy)
			if errors.Is(err, pgx.ErrNoRows) {
				return invalid("supersedesId", "does not exist")
			}
			if err != nil {
				return err
			}
			if sStatus != StatusApproved || sBy != nil || sList != in.PriceListID || sVariant != in.VariantID || sType != in.PriceType {
				return invalid("supersedesId", "must be a current approved price of the same list, variant and type")
			}
		}
		it, err = scanItem(tx.QueryRow(ctx, `INSERT INTO price_items (price_list_id, brand_id, model_id, variant_id, price_type, amount,
				effective_from, effective_to, source_note, supersedes_id, created_by)
			VALUES ($1, $2, $3, $4, $5, $6::numeric, $7, $8, $9, $10, $11) RETURNING `+itemColumns,
			in.PriceListID, brandID, modelID, in.VariantID, in.PriceType, in.Amount, in.EffectiveFrom, in.EffectiveTo,
			strOrNil(in.SourceNote), strOrNil(in.SupersedesID), createdBy))
		if err != nil {
			return err
		}
		ev.EntityID = it.ID
		return audit.Record(ctx, tx, withSummary(ev, audit.Summary{
			"priceListId": it.PriceListID, "variantId": it.VariantID, "priceType": it.PriceType, "amount": it.Amount,
			"effectiveFrom": it.EffectiveFrom.UTC().Format(time.RFC3339), "effectiveTo": timeOrNil(it.EffectiveTo),
			"supersedesId": strOrNil(it.SupersedesID)}))
	})
	return it, err
}

func lockItem(ctx context.Context, tx pgx.Tx, id string, version int) (Item, error) {
	it, err := scanItem(tx.QueryRow(ctx, `SELECT `+itemColumns+` FROM price_items WHERE id = $1 FOR UPDATE`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return it, ErrNotFound
	}
	if err != nil {
		return it, err
	}
	if it.Version != version {
		return it, ErrVersionConflict
	}
	return it, nil
}

// UpdateItem implements Repository (drafts only).
func (s *PGStore) UpdateItem(ctx context.Context, id string, p itemPatch, ev audit.Event) (Item, error) {
	var it Item
	err := s.tx(ctx, func(tx pgx.Tx) error {
		old, err := lockItem(ctx, tx, id, p.Version)
		if err != nil {
			return err
		}
		if err := checkDraftEditable(old.Status); err != nil {
			return err
		}
		amount, from, to, note := old.Amount, old.EffectiveFrom, old.EffectiveTo, old.SourceNote
		if p.Amount != nil {
			amount = *p.Amount
		}
		if p.EffectiveFrom != nil {
			from = *p.EffectiveFrom
		}
		if p.EffectiveToSet {
			to = p.EffectiveTo
		}
		if p.NoteSet {
			note = p.SourceNote
		}
		if to != nil && !to.After(from) {
			return invalid("effectiveTo", "must be after effectiveFrom")
		}
		it, err = scanItem(tx.QueryRow(ctx, `UPDATE price_items SET amount = $2::numeric, effective_from = $3, effective_to = $4,
			source_note = $5, version = version + 1 WHERE id = $1 RETURNING `+itemColumns, id, amount, from, to, strOrNil(note)))
		if err != nil {
			return err
		}
		return audit.Record(ctx, tx, withSummary(ev, audit.Summary{
			"amount":        audit.Change{From: old.Amount, To: it.Amount},
			"effectiveFrom": audit.Change{From: old.EffectiveFrom.UTC().Format(time.RFC3339), To: it.EffectiveFrom.UTC().Format(time.RFC3339)},
			"effectiveTo":   audit.Change{From: timeOrNil(old.EffectiveTo), To: timeOrNil(it.EffectiveTo)},
		}))
	})
	return it, err
}

// CancelItem implements Repository (drafts only; kept as history).
func (s *PGStore) CancelItem(ctx context.Context, id string, version int, ev audit.Event) (Item, error) {
	var it Item
	err := s.tx(ctx, func(tx pgx.Tx) error {
		old, err := lockItem(ctx, tx, id, version)
		if err != nil {
			return err
		}
		if err := checkDraftEditable(old.Status); err != nil {
			return err
		}
		it, err = scanItem(tx.QueryRow(ctx, `UPDATE price_items SET status = 'CANCELLED', version = version + 1 WHERE id = $1 RETURNING `+itemColumns, id))
		if err != nil {
			return err
		}
		return audit.Record(ctx, tx, withSummary(ev, audit.Summary{"status": audit.Change{From: StatusDraft, To: StatusCancelled}}))
	})
	return it, err
}

// ApproveItem implements Repository. In one transaction: lock the draft, its
// list/brand/model/variant and any superseded price; check the rules and
// overlaps; end the superseded period at the new start; approve. The exclusion
// constraint makes concurrent approvals of overlapping periods fail.
func (s *PGStore) ApproveItem(ctx context.Context, id string, in ApproveRequest, approverID string, now time.Time,
	ev audit.Event, supersededEv audit.Event) (Item, error) {
	var it Item
	err := s.tx(ctx, func(tx pgx.Tx) error {
		draft, err := lockItem(ctx, tx, id, in.Version)
		if err != nil {
			return err
		}
		if draft.CreatedBy == approverID {
			return ErrSelfApproval
		}
		f := approvalFacts{Item: draft, Now: now}
		if err := tx.QueryRow(ctx, `SELECT pl.lifecycle_status, b.lifecycle_status, m.lifecycle_status, v.lifecycle_status, v.confidence
			FROM price_lists pl, brands b, models m, variants v
			WHERE pl.id = $1 AND b.id = $2 AND m.id = $3 AND v.id = $4
			FOR SHARE OF pl, b, m, v`, draft.PriceListID, draft.BrandID, draft.ModelID, draft.VariantID).
			Scan(&f.ListStatus, &f.BrandStatus, &f.ModelStatus, &f.VariantStatus, &f.VariantConfidence); err != nil {
			return fmt.Errorf("load approval facts: %w", err)
		}
		if draft.SupersedesID != nil {
			sup, err := scanItem(tx.QueryRow(ctx, `SELECT `+itemColumns+` FROM price_items WHERE id = $1 FOR UPDATE`, *draft.SupersedesID))
			if err != nil {
				return fmt.Errorf("lock superseded price: %w", err)
			}
			f.Superseded = &sup
		}
		if err := checkApproval(f); err != nil {
			return err
		}
		// Overlap with any other approved price of the same list/variant/type.
		var conflict string
		err = tx.QueryRow(ctx, `SELECT id::text FROM price_items
			WHERE status = 'APPROVED' AND price_list_id = $1 AND variant_id = $2 AND price_type = $3
			  AND id IS DISTINCT FROM $4::uuid
			  AND tstzrange(effective_from, effective_to, '[)') && tstzrange($5, $6, '[)')
			LIMIT 1`, draft.PriceListID, draft.VariantID, draft.PriceType, strOrNil(draft.SupersedesID),
			draft.EffectiveFrom, draft.EffectiveTo).Scan(&conflict)
		if err == nil {
			return &OverlapError{ConflictingID: conflict}
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		if sup := f.Superseded; sup != nil {
			if _, err := tx.Exec(ctx, `UPDATE price_items SET effective_to = $2, superseded_by_id = $3, version = version + 1 WHERE id = $1`,
				sup.ID, draft.EffectiveFrom, draft.ID); err != nil {
				return err
			}
			supersededEv.EntityID = sup.ID
			if err := audit.Record(ctx, tx, withSummary(supersededEv, audit.Summary{
				"supersededById": draft.ID,
				"effectiveTo":    audit.Change{From: timeOrNil(sup.EffectiveTo), To: draft.EffectiveFrom.UTC().Format(time.RFC3339)},
			})); err != nil {
				return err
			}
		}
		it, err = scanItem(tx.QueryRow(ctx, `UPDATE price_items SET status = 'APPROVED', approved_by = $2, approved_at = $3,
			approval_reference = $4, version = version + 1 WHERE id = $1 RETURNING `+itemColumns,
			id, approverID, now, strOrNil(in.ApprovalReference)))
		if err != nil {
			return err
		}
		return audit.Record(ctx, tx, withSummary(ev, audit.Summary{
			"status": audit.Change{From: StatusDraft, To: StatusApproved}, "priceType": it.PriceType, "amount": it.Amount,
			"effectiveFrom": it.EffectiveFrom.UTC().Format(time.RFC3339), "effectiveTo": timeOrNil(it.EffectiveTo),
			"supersedesId": strOrNil(it.SupersedesID), "approvalReference": strOrNil(it.ApprovalReference)}))
	})
	return it, err
}

// Lookup implements Repository: the approved price valid at the instant. A
// variant of another brand than the list is a cross-brand mismatch (422), not
// "no price".
func (s *PGStore) Lookup(ctx context.Context, priceListID, variantID, priceType string, at time.Time) (Item, error) {
	if err := s.checkFilterBrands(ctx, "", priceListID, variantID); err != nil {
		return Item{}, err
	}
	it, err := scanItem(s.pool.QueryRow(ctx, `SELECT `+itemColumns+` FROM price_items
		WHERE status = 'APPROVED' AND price_list_id = $1 AND variant_id = $2 AND price_type = $3
		  AND effective_from <= $4 AND (effective_to IS NULL OR effective_to > $4)`, priceListID, variantID, priceType, at))
	if errors.Is(err, pgx.ErrNoRows) {
		return Item{}, ErrNoPrice
	}
	return it, err
}

// RecordAudit implements Repository (refused approvals).
func (s *PGStore) RecordAudit(ctx context.Context, ev audit.Event) error {
	return audit.Record(ctx, s.pool, ev)
}
