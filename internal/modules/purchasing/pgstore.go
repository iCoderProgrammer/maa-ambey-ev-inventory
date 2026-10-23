package purchasing

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

	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/audit"
)

// PGStore is the PostgreSQL Repository. Each mutation is one transaction that
// locks the PO, checks its version and state, writes, records the status
// history and audits.
type PGStore struct{ pool *pgxpool.Pool }

// NewPGStore returns a Repository backed by pool.
func NewPGStore(pool *pgxpool.Pool) *PGStore { return &PGStore{pool: pool} }

// Totals are computed from the lines on every read; nothing stores them.
const poColumns = `po.id::text, po.po_number, po.supplier_id::text, s.legal_name, s.status, po.status,
	to_char(po.expected_date, 'YYYY-MM-DD'), po.notes, po.currency, po.created_by::text, cu.display_name,
	po.submitted_by::text, po.submitted_at, po.approval_basis, po.approved_by::text, po.approved_at, po.approval_note,
	po.threshold_applied::text, po.cancelled_by::text, po.cancelled_at, po.cancel_reason,
	t.line_count, t.total_quantity, t.received_quantity, t.remaining_quantity, t.total_amount, t.brand_names,
	po.version, po.created_at, po.updated_at`

const poFrom = ` FROM purchase_orders po
	JOIN suppliers s ON s.id = po.supplier_id
	JOIN users cu ON cu.id = po.created_by
	CROSS JOIN LATERAL (
		SELECT count(*)::int AS line_count,
			coalesce(sum(l.quantity), 0)::int AS total_quantity,
			coalesce(sum(l.received_quantity), 0)::int AS received_quantity,
			coalesce(sum(greatest(l.quantity - l.received_quantity, 0)), 0)::int AS remaining_quantity,
			coalesce(sum(l.quantity * l.unit_cost), 0)::numeric(18,2)::text AS total_amount,
			coalesce(array_agg(DISTINCT b.name ORDER BY b.name) FILTER (WHERE b.name IS NOT NULL), '{}') AS brand_names
		FROM purchase_order_lines l JOIN brands b ON b.id = l.brand_id
		WHERE l.purchase_order_id = po.id) t`

func scanPO(row pgx.Row) (PurchaseOrder, error) {
	var p PurchaseOrder
	err := row.Scan(&p.ID, &p.Number, &p.SupplierID, &p.SupplierName, &p.SupplierStatus, &p.Status,
		&p.ExpectedDate, &p.Notes, &p.Currency, &p.CreatedBy, &p.CreatedByName,
		&p.SubmittedBy, &p.SubmittedAt, &p.ApprovalBasis, &p.ApprovedBy, &p.ApprovedAt, &p.ApprovalNote,
		&p.ThresholdApplied, &p.CancelledBy, &p.CancelledAt, &p.CancelReason,
		&p.LineCount, &p.TotalQuantity, &p.ReceivedQuantity, &p.RemainingQuantity, &p.TotalAmount, &p.BrandNames,
		&p.Version, &p.CreatedAt, &p.UpdatedAt)
	return p, err
}

const lineColumns = `l.id::text, l.line_no, l.brand_id::text, b.name, b.lifecycle_status, l.model_id::text, m.name,
	m.lifecycle_status, l.variant_id::text, v.name, v.lifecycle_status, l.quantity, l.unit_cost::text,
	(l.quantity * l.unit_cost)::numeric(18,2)::text, l.received_quantity, greatest(l.quantity - l.received_quantity, 0),
	l.reference_price_list_id::text, l.reference_price_item_id::text, l.reference_unit_cost::text, l.reference_captured_at,
	(l.unit_cost - l.reference_unit_cost)::text, l.note, l.created_at, l.updated_at`

const lineFrom = ` FROM purchase_order_lines l
	JOIN brands b ON b.id = l.brand_id
	JOIN models m ON m.id = l.model_id
	JOIN variants v ON v.id = l.variant_id`

func scanLine(row pgx.Row) (Line, error) {
	var l Line
	err := row.Scan(&l.ID, &l.LineNo, &l.BrandID, &l.BrandName, &l.BrandStatus, &l.ModelID, &l.ModelName,
		&l.ModelStatus, &l.VariantID, &l.VariantName, &l.VariantStatus, &l.Quantity, &l.UnitCost,
		&l.LineTotal, &l.ReceivedQuantity, &l.RemainingQuantity,
		&l.ReferencePriceListID, &l.ReferencePriceItemID, &l.ReferenceUnitCost, &l.ReferenceCapturedAt,
		&l.CostVariance, &l.Note, &l.CreatedAt, &l.UpdatedAt)
	return l, err
}

const eventColumns = `e.id::text, e.action, e.from_status, e.to_status, e.actor_id::text, u.display_name, e.note, e.created_at`

func scanEvent(row pgx.Row) (Event, error) {
	var e Event
	err := row.Scan(&e.ID, &e.Action, &e.FromStatus, &e.ToStatus, &e.ActorID, &e.ActorName, &e.Note, &e.CreatedAt)
	return e, err
}

const settingsColumns = `approval_threshold::text, first_time_supplier_needs_approval, approval_reference, updated_by::text,
	version, CASE WHEN version > 1 THEN updated_at END`

func scanSettings(row pgx.Row) (Settings, error) {
	var s Settings
	err := row.Scan(&s.ApprovalThreshold, &s.FirstTimeSupplierNeedsApproval, &s.ApprovalReference, &s.UpdatedBy,
		&s.Version, &s.UpdatedAt)
	return s, err
}

var duplicateFields = map[string]string{
	"purchase_order_lines_variant_key": "variantId",
	"purchase_orders_idempotency_key":  "Idempotency-Key",
}

func mapPGError(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}
	switch pgErr.Code {
	case "23505":
		if f, ok := duplicateFields[pgErr.ConstraintName]; ok {
			return &DuplicateError{Field: f}
		}
	case "23503":
		switch pgErr.ConstraintName {
		case "purchase_order_lines_model_fk":
			return invalid("modelId", "is not a model of that brand")
		case "purchase_order_lines_variant_brand_fk", "purchase_order_lines_variant_model_fk":
			return invalid("variantId", "is not a variant of that model")
		case "purchase_order_lines_price_list_fk":
			return invalid("priceListId", "is not a price list of the line's brand")
		}
	case "23514":
		return invalid("body", "violates purchase order rule "+pgErr.ConstraintName)
	case "42501":
		return &StateError{Message: "This purchase order record is history and cannot change."}
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

// queryAll runs a query and scans every row.
func queryAll[T any](ctx context.Context, q interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}, scan func(pgx.Row) (T, error), sql string, args ...any) ([]T, error) {
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

// ── Reads ────────────────────────────────────────────────────────────────────

// List implements Repository.
func (s *PGStore) List(ctx context.Context, f ListFilter) (Page[PurchaseOrder], error) {
	var conds []string
	var args []any
	add := func(cond string, v any) {
		args = append(args, v)
		conds = append(conds, strings.ReplaceAll(cond, "?", "$"+strconv.Itoa(len(args))))
	}
	if f.Status != "" {
		add("po.status = ?", f.Status)
	}
	if f.SupplierID != "" {
		add("po.supplier_id = ?", f.SupplierID)
	}
	if f.BrandID != "" {
		add("EXISTS (SELECT 1 FROM purchase_order_lines bl WHERE bl.purchase_order_id = po.id AND bl.brand_id = ?)", f.BrandID)
	}
	if f.Query != "" {
		r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
		add(`(po.po_number ILIKE ? ESCAPE '\' OR s.legal_name ILIKE ? ESCAPE '\' OR s.trade_name ILIKE ? ESCAPE '\')`, "%"+r.Replace(f.Query)+"%")
	}
	where := ""
	if len(conds) > 0 {
		where = " WHERE " + strings.Join(conds, " AND ")
	}
	var total int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM purchase_orders po JOIN suppliers s ON s.id = po.supplier_id`+where, args...).Scan(&total); err != nil {
		return Page[PurchaseOrder]{}, fmt.Errorf("count purchase orders: %w", err)
	}
	n := len(args)
	items, err := queryAll(ctx, s.pool, scanPO, `SELECT `+poColumns+poFrom+where+` ORDER BY po.created_at DESC, po.id LIMIT $`+
		strconv.Itoa(n+1)+` OFFSET $`+strconv.Itoa(n+2), append(args, f.Limit, (f.Page-1)*f.Limit)...)
	if err != nil {
		return Page[PurchaseOrder]{}, fmt.Errorf("list purchase orders: %w", err)
	}
	return Page[PurchaseOrder]{Items: items, Page: f.Page, Limit: f.Limit, Total: total}, nil
}

// snapshot is a PO as stored plus the facts its next step depends on.
type snapshot struct {
	Detail
	Facts                 poFacts
	Settings              Settings
	SupplierHadApprovedPO bool
}

type querier interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

func loadSnapshot(ctx context.Context, q querier, id string, lock bool) (snapshot, error) {
	var sn snapshot
	po, err := scanPO(q.QueryRow(ctx, `SELECT `+poColumns+poFrom+` WHERE po.id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return sn, ErrNotFound
	}
	if err != nil {
		return sn, fmt.Errorf("get purchase order: %w", err)
	}
	sn.PurchaseOrder = po
	if sn.Lines, err = queryAll(ctx, q, scanLine, `SELECT `+lineColumns+lineFrom+` WHERE l.purchase_order_id = $1 ORDER BY l.line_no`, id); err != nil {
		return sn, fmt.Errorf("lines: %w", err)
	}
	if sn.Events, err = queryAll(ctx, q, scanEvent, `SELECT `+eventColumns+` FROM purchase_order_events e
		LEFT JOIN users u ON u.id = e.actor_id WHERE e.purchase_order_id = $1 ORDER BY e.seq`, id); err != nil {
		return sn, fmt.Errorf("events: %w", err)
	}
	if sn.Facts, err = loadFacts(ctx, q, po, lock); err != nil {
		return sn, fmt.Errorf("facts: %w", err)
	}
	if sn.Settings, err = scanSettings(q.QueryRow(ctx, `SELECT `+settingsColumns+` FROM purchase_order_settings`)); err != nil {
		return sn, fmt.Errorf("settings: %w", err)
	}
	if err = q.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM purchase_orders WHERE supplier_id = $1 AND id <> $2 AND approved_at IS NOT NULL)`,
		po.SupplierID, po.ID).Scan(&sn.SupplierHadApprovedPO); err != nil {
		return sn, fmt.Errorf("supplier history: %w", err)
	}
	return sn, nil
}

// loadFacts reads what the prerequisites look at. With lock (inside a
// transition's transaction, PO already locked) brands, models and variants are
// share-locked so they cannot change until the transition commits.
func loadFacts(ctx context.Context, q querier, po PurchaseOrder, lock bool) (poFacts, error) {
	f := poFacts{SupplierStatus: po.SupplierStatus}
	if po.ExpectedDate != nil {
		d, err := time.Parse(time.DateOnly, *po.ExpectedDate)
		if err != nil {
			return f, err
		}
		f.ExpectedDate = &d
	}
	var today string
	if err := q.QueryRow(ctx, `SELECT to_char(current_date, 'YYYY-MM-DD')`).Scan(&today); err != nil {
		return f, err
	}
	f.Today, _ = time.Parse(time.DateOnly, today)
	lockClause := ""
	if lock {
		lockClause = " FOR SHARE OF b, m, v"
	}
	lines, err := queryAll(ctx, q, func(r pgx.Row) (lineFacts, error) {
		var l lineFacts
		err := r.Scan(&l.LineID, &l.BrandID, &l.BrandName, &l.BrandStatus, &l.BrandAuthorisation, &l.ModelName, &l.ModelStatus,
			&l.VariantName, &l.VariantStatus, &l.SupplierLinkCurrent)
		return l, err
	}, `SELECT l.id::text, l.brand_id::text, b.name, b.lifecycle_status, b.authorisation_status, m.name, m.lifecycle_status,
			v.name, v.lifecycle_status,
			EXISTS (SELECT 1 FROM supplier_brand_links sl WHERE sl.supplier_id = $2 AND sl.brand_id = l.brand_id
				AND sl.status = 'VERIFIED' AND sl.effective_from <= current_date
				AND (sl.effective_to IS NULL OR sl.effective_to >= current_date))
		FROM purchase_order_lines l JOIN brands b ON b.id = l.brand_id JOIN models m ON m.id = l.model_id
		JOIN variants v ON v.id = l.variant_id WHERE l.purchase_order_id = $1 ORDER BY l.line_no`+lockClause, po.ID, po.SupplierID)
	if err != nil {
		return f, err
	}
	f.Lines = lines
	return f, nil
}

// GetSnapshot implements Repository.
func (s *PGStore) GetSnapshot(ctx context.Context, id string) (snapshot, error) {
	return loadSnapshot(ctx, s.pool, id, false)
}

// GetSettings implements Repository.
func (s *PGStore) GetSettings(ctx context.Context) (Settings, error) {
	return scanSettings(s.pool.QueryRow(ctx, `SELECT `+settingsColumns+` FROM purchase_order_settings`))
}

// ── Writes ───────────────────────────────────────────────────────────────────

type poRow struct {
	ID, Number, Status, SupplierID, CreatedBy string
	SubmittedBy                               *string
	Version, Received                         int
}

func lockPO(ctx context.Context, tx pgx.Tx, id string, version int) (poRow, error) {
	var p poRow
	err := tx.QueryRow(ctx, `SELECT id::text, po_number, status, supplier_id::text, created_by::text, submitted_by::text, version,
			(SELECT coalesce(sum(received_quantity), 0)::int FROM purchase_order_lines WHERE purchase_order_id = po.id)
		FROM purchase_orders po WHERE id = $1 FOR UPDATE`, id).
		Scan(&p.ID, &p.Number, &p.Status, &p.SupplierID, &p.CreatedBy, &p.SubmittedBy, &p.Version, &p.Received)
	if errors.Is(err, pgx.ErrNoRows) {
		return p, ErrNotFound
	}
	if err != nil {
		return p, err
	}
	if p.Version != version {
		return p, ErrVersionConflict
	}
	return p, nil
}

func bump(ctx context.Context, tx pgx.Tx, id string) error {
	_, err := tx.Exec(ctx, `UPDATE purchase_orders SET version = version + 1 WHERE id = $1`, id)
	return err
}

func addEvent(ctx context.Context, tx pgx.Tx, poID, action string, from *string, to, actorID string, note *string) error {
	_, err := tx.Exec(ctx, `INSERT INTO purchase_order_events (purchase_order_id, action, from_status, to_status, actor_id, note)
		VALUES ($1, $2, $3, $4, $5, $6)`, poID, action, str(from), to, actorID, str(note))
	return err
}

// checkSupplier: the supplier exists and is not archived (drafts may use a
// supplier still pending verification; submission needs it verified).
func checkSupplier(ctx context.Context, tx pgx.Tx, id string) error {
	var status string
	err := tx.QueryRow(ctx, `SELECT status FROM suppliers WHERE id = $1 FOR SHARE`, id).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return invalid("supplierId", "does not exist")
	}
	if err != nil {
		return err
	}
	if status == "ARCHIVED" {
		return invalid("supplierId", "is archived; choose another supplier")
	}
	return nil
}

// resolvedLine is a validated line with its catalogue snapshot.
type resolvedLine struct {
	LineInput
	RefItemID, RefAmount *string
}

// resolveLine checks the brand → model → variant chain (none archived) and
// takes the reference cost snapshot when a price list is named.
func resolveLine(ctx context.Context, tx pgx.Tx, prefix string, in LineInput) (resolvedLine, error) {
	out := resolvedLine{LineInput: in}
	var vBrand, vModel, mBrand, bStatus, mStatus, vStatus string
	err := tx.QueryRow(ctx, `SELECT v.brand_id::text, v.model_id::text, m.brand_id::text, b.lifecycle_status, m.lifecycle_status,
			v.lifecycle_status
		FROM variants v JOIN models m ON m.id = v.model_id JOIN brands b ON b.id = v.brand_id
		WHERE v.id = $1 FOR SHARE OF v, m, b`, in.VariantID).Scan(&vBrand, &vModel, &mBrand, &bStatus, &mStatus, &vStatus)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, invalid(prefix+"variantId", "does not exist")
	}
	if err != nil {
		return out, err
	}
	switch {
	case vModel != in.ModelID:
		return out, invalid(prefix+"variantId", "is not a variant of that model")
	case mBrand != in.BrandID || vBrand != in.BrandID:
		return out, invalid(prefix+"modelId", "is not a model of that brand")
	case bStatus == "ARCHIVED":
		return out, invalid(prefix+"brandId", "is archived")
	case mStatus == "ARCHIVED":
		return out, invalid(prefix+"modelId", "is archived")
	case vStatus == "ARCHIVED":
		return out, invalid(prefix+"variantId", "is archived")
	}
	if in.PriceListID == nil {
		return out, nil
	}
	var listBrand, listStatus string
	err = tx.QueryRow(ctx, `SELECT brand_id::text, lifecycle_status FROM price_lists WHERE id = $1`, *in.PriceListID).Scan(&listBrand, &listStatus)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, invalid(prefix+"priceListId", "does not exist")
	}
	if err != nil {
		return out, err
	}
	if listBrand != in.BrandID {
		return out, invalid(prefix+"priceListId", "is not a price list of the line's brand")
	}
	if listStatus != "ACTIVE" {
		return out, invalid(prefix+"priceListId", "is archived")
	}
	var itemID, amount string
	err = tx.QueryRow(ctx, `SELECT id::text, amount::text FROM price_items
		WHERE price_list_id = $1 AND variant_id = $2 AND price_type = 'COST' AND status = 'APPROVED'
		  AND effective_from <= now() AND (effective_to IS NULL OR effective_to > now())`, *in.PriceListID, in.VariantID).Scan(&itemID, &amount)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, invalid(prefix+"priceListId", "has no approved cost price valid now for this variant")
	}
	if err != nil {
		return out, err
	}
	out.RefItemID, out.RefAmount = &itemID, &amount
	return out, nil
}

func insertLine(ctx context.Context, tx pgx.Tx, poID string, lineNo int, l resolvedLine) (string, error) {
	var id string
	err := tx.QueryRow(ctx, `INSERT INTO purchase_order_lines (purchase_order_id, line_no, brand_id, model_id, variant_id, quantity,
			unit_cost, reference_price_list_id, reference_price_item_id, reference_unit_cost, reference_captured_at, note)
		VALUES ($1, $2, $3, $4, $5, $6, $7::numeric, $8, $9, $10::numeric, CASE WHEN $9::uuid IS NULL THEN NULL ELSE now() END, $11)
		RETURNING id::text`,
		poID, lineNo, l.BrandID, l.ModelID, l.VariantID, l.Quantity, l.UnitCost,
		refList(l), str(l.RefItemID), str(l.RefAmount), str(l.Note)).Scan(&id)
	return id, err
}

func refList(l resolvedLine) any {
	if l.RefItemID == nil {
		return nil
	}
	return str(l.PriceListID)
}

func lineSummary(l resolvedLine) audit.Summary {
	return audit.Summary{"variantId": l.VariantID, "quantity": l.Quantity, "unitCost": l.UnitCost,
		"referencePriceItemId": str(l.RefItemID)}
}

// Create implements Repository. With an idempotency key, a repeat by the same
// user returns the PO created first (replayed = true).
func (s *PGStore) Create(ctx context.Context, in CreateInput, createdBy, idemKey string, ev audit.Event) (string, bool, error) {
	if idemKey != "" {
		if id, ok, err := s.findByKey(ctx, createdBy, idemKey); err != nil || ok {
			return id, ok, err
		}
	}
	var id string
	err := s.tx(ctx, func(tx pgx.Tx) error {
		if err := checkSupplier(ctx, tx, in.SupplierID); err != nil {
			return err
		}
		lines := make([]resolvedLine, len(in.Lines))
		for i, l := range in.Lines {
			r, err := resolveLine(ctx, tx, "lines["+strconv.Itoa(i)+"].", l)
			if err != nil {
				return err
			}
			lines[i] = r
		}
		var number string
		var key any
		if idemKey != "" {
			key = idemKey
		}
		if err := tx.QueryRow(ctx, `INSERT INTO purchase_orders (supplier_id, expected_date, notes, created_by, idempotency_key)
			VALUES ($1, $2::date, $3, $4, $5) RETURNING id::text, po_number`,
			in.SupplierID, str(in.ExpectedDate), str(in.Notes), createdBy, key).Scan(&id, &number); err != nil {
			return err
		}
		lineSums := make([]any, len(lines))
		for i, l := range lines {
			if _, err := insertLine(ctx, tx, id, i+1, l); err != nil {
				return err
			}
			lineSums[i] = map[string]any(lineSummary(l))
		}
		if err := addEvent(ctx, tx, id, ActionCreated, nil, StatusDraft, createdBy, nil); err != nil {
			return err
		}
		ev.EntityID = id
		return audit.Record(ctx, tx, withSummary(ev, audit.Summary{"number": number, "supplierId": in.SupplierID,
			"expectedDate": str(in.ExpectedDate), "lines": lineSums}))
	})
	var dup *DuplicateError
	if idemKey != "" && errors.As(err, &dup) && dup.Field == "Idempotency-Key" {
		// A concurrent request with the same key won.
		id, ok, ferr := s.findByKey(ctx, createdBy, idemKey)
		if ferr == nil && ok {
			return id, true, nil
		}
	}
	return id, false, err
}

func (s *PGStore) findByKey(ctx context.Context, createdBy, key string) (string, bool, error) {
	var id string
	err := s.pool.QueryRow(ctx, `SELECT id::text FROM purchase_orders WHERE created_by = $1 AND idempotency_key = $2`, createdBy, key).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	return id, err == nil, err
}

// UpdateHeader implements Repository.
func (s *PGStore) UpdateHeader(ctx context.Context, id string, p HeaderPatch, ev audit.Event) error {
	return s.tx(ctx, func(tx pgx.Tx) error {
		old, err := lockPO(ctx, tx, id, p.Version)
		if err != nil {
			return err
		}
		if err := checkEditable(old.Status); err != nil {
			return err
		}
		args := []any{id}
		var sets []string
		sum := audit.Summary{}
		if p.SupplierID != nil {
			if err := checkSupplier(ctx, tx, *p.SupplierID); err != nil {
				return err
			}
			args = append(args, *p.SupplierID)
			sets = append(sets, "supplier_id = $"+strconv.Itoa(len(args)))
			sum["supplierId"] = audit.Change{From: old.SupplierID, To: *p.SupplierID}
		}
		if p.ExpectedDate.Set {
			args = append(args, str(p.ExpectedDate.Value))
			sets = append(sets, "expected_date = $"+strconv.Itoa(len(args))+"::date")
			sum["expectedDate"] = str(p.ExpectedDate.Value)
		}
		if p.Notes.Set {
			args = append(args, str(p.Notes.Value))
			sets = append(sets, "notes = $"+strconv.Itoa(len(args)))
			sum["notes"] = "changed"
		}
		sets = append(sets, "version = version + 1")
		if _, err := tx.Exec(ctx, `UPDATE purchase_orders SET `+strings.Join(sets, ", ")+` WHERE id = $1`, args...); err != nil {
			return err
		}
		return audit.Record(ctx, tx, withSummary(ev, sum))
	})
}

// AddLine implements Repository.
func (s *PGStore) AddLine(ctx context.Context, id string, version int, in LineInput, ev audit.Event) error {
	return s.tx(ctx, func(tx pgx.Tx) error {
		old, err := lockPO(ctx, tx, id, version)
		if err != nil {
			return err
		}
		if err := checkEditable(old.Status); err != nil {
			return err
		}
		var count, next int
		if err := tx.QueryRow(ctx, `SELECT count(*)::int, coalesce(max(line_no), 0) + 1 FROM purchase_order_lines WHERE purchase_order_id = $1`,
			id).Scan(&count, &next); err != nil {
			return err
		}
		if count >= MaxLines {
			return invalid("variantId", "a purchase order has at most "+strconv.Itoa(MaxLines)+" lines")
		}
		r, err := resolveLine(ctx, tx, "", in)
		if err != nil {
			return err
		}
		lineID, err := insertLine(ctx, tx, id, next, r)
		if err != nil {
			return err
		}
		if err := bump(ctx, tx, id); err != nil {
			return err
		}
		sum := lineSummary(r)
		sum["lineId"] = lineID
		return audit.Record(ctx, tx, withSummary(ev, sum))
	})
}

// UpdateLine implements Repository.
func (s *PGStore) UpdateLine(ctx context.Context, id, lineID string, p LinePatch, ev audit.Event) error {
	return s.tx(ctx, func(tx pgx.Tx) error {
		old, err := lockPO(ctx, tx, id, p.Version)
		if err != nil {
			return err
		}
		if err := checkEditable(old.Status); err != nil {
			return err
		}
		var cur LineInput
		err = tx.QueryRow(ctx, `SELECT brand_id::text, model_id::text, variant_id::text, quantity, unit_cost::text,
				reference_price_list_id::text, note
			FROM purchase_order_lines WHERE id = $1 AND purchase_order_id = $2 FOR UPDATE`, lineID, id).
			Scan(&cur.BrandID, &cur.ModelID, &cur.VariantID, &cur.Quantity, &cur.UnitCost, &cur.PriceListID, &cur.Note)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		sum := audit.Summary{"lineId": lineID}
		next := cur
		if p.Quantity != nil {
			next.Quantity = *p.Quantity
			sum["quantity"] = audit.Change{From: cur.Quantity, To: next.Quantity}
		}
		if p.UnitCost != nil {
			next.UnitCost = *p.UnitCost
			sum["unitCost"] = audit.Change{From: cur.UnitCost, To: next.UnitCost}
		}
		if p.Note.Set {
			next.Note = p.Note.Value
			sum["note"] = "changed"
		}
		// The snapshot is retaken only when a price list is named again; otherwise
		// the existing snapshot (or its absence) is kept as it was.
		retake := p.PriceListID.Set
		if retake {
			next.PriceListID = p.PriceListID.Value
		}
		sets := `quantity = $3, unit_cost = $4::numeric, note = $5`
		args := []any{lineID, id, next.Quantity, next.UnitCost, str(next.Note)}
		if retake {
			r := resolvedLine{LineInput: next}
			if next.PriceListID != nil {
				if r, err = resolveLine(ctx, tx, "", next); err != nil {
					return err
				}
			}
			sets += `, reference_price_list_id = $6, reference_price_item_id = $7, reference_unit_cost = $8::numeric,
				reference_captured_at = CASE WHEN $7::uuid IS NULL THEN NULL ELSE now() END`
			args = append(args, refList(r), str(r.RefItemID), str(r.RefAmount))
			sum["referencePriceItemId"] = str(r.RefItemID)
		}
		if _, err := tx.Exec(ctx, `UPDATE purchase_order_lines SET `+sets+` WHERE id = $1 AND purchase_order_id = $2`, args...); err != nil {
			return err
		}
		if err := bump(ctx, tx, id); err != nil {
			return err
		}
		return audit.Record(ctx, tx, withSummary(ev, sum))
	})
}

// RemoveLine implements Repository. Only draft lines are removed (never history).
func (s *PGStore) RemoveLine(ctx context.Context, id, lineID string, version int, ev audit.Event) error {
	return s.tx(ctx, func(tx pgx.Tx) error {
		old, err := lockPO(ctx, tx, id, version)
		if err != nil {
			return err
		}
		if err := checkEditable(old.Status); err != nil {
			return err
		}
		var variantID string
		var qty int
		err = tx.QueryRow(ctx, `DELETE FROM purchase_order_lines WHERE id = $1 AND purchase_order_id = $2 RETURNING variant_id::text, quantity`,
			lineID, id).Scan(&variantID, &qty)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if err := bump(ctx, tx, id); err != nil {
			return err
		}
		return audit.Record(ctx, tx, withSummary(ev, audit.Summary{"lineId": lineID, "variantId": variantID, "quantity": qty}))
	})
}

// Submit implements Repository: DRAFT -> SUBMITTED, or straight on to APPROVED
// when the signed-off threshold says no approval is needed.
func (s *PGStore) Submit(ctx context.Context, id string, version int, actorID string, ev audit.Event) error {
	return s.tx(ctx, func(tx pgx.Tx) error {
		old, err := lockPO(ctx, tx, id, version)
		if err != nil {
			return err
		}
		if err := checkTransition(ActionSubmitted, old.Status, old.Received); err != nil {
			return err
		}
		// Settings cannot change while this submission decides on them.
		if _, err := tx.Exec(ctx, `SELECT 1 FROM purchase_order_settings FOR SHARE`); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `SELECT 1 FROM suppliers WHERE id = $1 FOR SHARE`, old.SupplierID); err != nil {
			return err
		}
		sn, err := loadSnapshot(ctx, tx, id, true)
		if err != nil {
			return err
		}
		if unmet := prerequisites(sn.Facts); len(unmet) > 0 {
			return &BlockedError{Unmet: unmet}
		}
		required, reason := approvalRequirement(sn.TotalAmount, sn.Settings, sn.SupplierHadApprovedPO)
		draft := StatusDraft
		if required {
			_, err = tx.Exec(ctx, `UPDATE purchase_orders SET status = 'SUBMITTED', submitted_by = $2, submitted_at = now(),
				threshold_applied = $3::numeric, version = version + 1 WHERE id = $1`, id, actorID, str(sn.Settings.ApprovalThreshold))
		} else {
			_, err = tx.Exec(ctx, `UPDATE purchase_orders SET status = 'APPROVED', submitted_by = $2, submitted_at = now(),
				threshold_applied = $3::numeric, approval_basis = 'BELOW_THRESHOLD', approved_at = now(), version = version + 1
				WHERE id = $1`, id, actorID, str(sn.Settings.ApprovalThreshold))
		}
		if err != nil {
			return err
		}
		if err := addEvent(ctx, tx, id, ActionSubmitted, &draft, StatusSubmitted, actorID, nil); err != nil {
			return err
		}
		to := StatusSubmitted
		if !required {
			submitted := StatusSubmitted
			to = StatusApproved
			if err := addEvent(ctx, tx, id, ActionApprovalNotRequired, &submitted, StatusApproved, actorID, &reason); err != nil {
				return err
			}
		}
		return audit.Record(ctx, tx, withSummary(ev, audit.Summary{"number": old.Number,
			"status": audit.Change{From: StatusDraft, To: to}, "total": sn.TotalAmount,
			"approvalRequired": required, "threshold": str(sn.Settings.ApprovalThreshold)}))
	})
}

// Approve implements Repository. The creator and the submitter never approve
// (ErrSelfApproval); prerequisites are checked again at approval time.
func (s *PGStore) Approve(ctx context.Context, id string, in ApproveInput, approverID string, ev audit.Event) error {
	return s.tx(ctx, func(tx pgx.Tx) error {
		old, err := lockPO(ctx, tx, id, in.Version)
		if err != nil {
			return err
		}
		if err := checkTransition(ActionApproved, old.Status, old.Received); err != nil {
			return err
		}
		if old.CreatedBy == approverID || (old.SubmittedBy != nil && *old.SubmittedBy == approverID) {
			return ErrSelfApproval
		}
		if _, err := tx.Exec(ctx, `SELECT 1 FROM suppliers WHERE id = $1 FOR SHARE`, old.SupplierID); err != nil {
			return err
		}
		sn, err := loadSnapshot(ctx, tx, id, true)
		if err != nil {
			return err
		}
		if unmet := prerequisites(sn.Facts); len(unmet) > 0 {
			return &BlockedError{Approval: true, Unmet: unmet}
		}
		if _, err := tx.Exec(ctx, `UPDATE purchase_orders SET status = 'APPROVED', approval_basis = 'APPROVER', approved_by = $2,
			approved_at = now(), approval_note = $3, version = version + 1 WHERE id = $1`, id, approverID, str(in.Note)); err != nil {
			return err
		}
		from := StatusSubmitted
		if err := addEvent(ctx, tx, id, ActionApproved, &from, StatusApproved, approverID, in.Note); err != nil {
			return err
		}
		return audit.Record(ctx, tx, withSummary(ev, audit.Summary{"number": old.Number,
			"status": audit.Change{From: StatusSubmitted, To: StatusApproved}, "total": sn.TotalAmount}))
	})
}

// Return implements Repository: SUBMITTED -> DRAFT with a reason.
func (s *PGStore) Return(ctx context.Context, id string, in ReasonInput, actorID string, ev audit.Event) error {
	return s.tx(ctx, func(tx pgx.Tx) error {
		old, err := lockPO(ctx, tx, id, in.Version)
		if err != nil {
			return err
		}
		if err := checkTransition(ActionReturned, old.Status, old.Received); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE purchase_orders SET status = 'DRAFT', submitted_by = NULL, submitted_at = NULL,
			threshold_applied = NULL, version = version + 1 WHERE id = $1`, id); err != nil {
			return err
		}
		from := StatusSubmitted
		if err := addEvent(ctx, tx, id, ActionReturned, &from, StatusDraft, actorID, &in.Reason); err != nil {
			return err
		}
		return audit.Record(ctx, tx, withSummary(ev, audit.Summary{"number": old.Number,
			"status": audit.Change{From: StatusSubmitted, To: StatusDraft}, "reason": in.Reason}))
	})
}

// Cancel implements Repository: nothing received yet; the PO stays as history.
func (s *PGStore) Cancel(ctx context.Context, id string, in ReasonInput, actorID string, ev audit.Event) error {
	return s.tx(ctx, func(tx pgx.Tx) error {
		old, err := lockPO(ctx, tx, id, in.Version)
		if err != nil {
			return err
		}
		if err := checkTransition(ActionCancelled, old.Status, old.Received); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE purchase_orders SET status = 'CANCELLED', cancelled_by = $2, cancelled_at = now(),
			cancel_reason = $3, version = version + 1 WHERE id = $1`, id, actorID, in.Reason); err != nil {
			return err
		}
		from := old.Status
		if err := addEvent(ctx, tx, id, ActionCancelled, &from, StatusCancelled, actorID, &in.Reason); err != nil {
			return err
		}
		return audit.Record(ctx, tx, withSummary(ev, audit.Summary{"number": old.Number,
			"status": audit.Change{From: old.Status, To: StatusCancelled}, "reason": in.Reason}))
	})
}

// UpdateSettings implements Repository.
func (s *PGStore) UpdateSettings(ctx context.Context, in SettingsInput, actorID string, ev audit.Event) (Settings, error) {
	var out Settings
	err := s.tx(ctx, func(tx pgx.Tx) error {
		old, err := scanSettings(tx.QueryRow(ctx, `SELECT `+settingsColumns+` FROM purchase_order_settings FOR UPDATE`))
		if err != nil {
			return err
		}
		if old.Version != in.Version {
			return ErrVersionConflict
		}
		if out, err = scanSettings(tx.QueryRow(ctx, `UPDATE purchase_order_settings SET approval_threshold = $1::numeric,
				first_time_supplier_needs_approval = $2, approval_reference = $3, updated_by = $4, version = version + 1
			RETURNING `+settingsColumns, str(in.ApprovalThreshold), in.FirstTimeSupplierNeedsApproval, str(in.ApprovalReference), actorID)); err != nil {
			return err
		}
		return audit.Record(ctx, tx, withSummary(ev, audit.Summary{
			"approvalThreshold":              audit.Change{From: str(old.ApprovalThreshold), To: str(out.ApprovalThreshold)},
			"firstTimeSupplierNeedsApproval": audit.Change{From: old.FirstTimeSupplierNeedsApproval, To: out.FirstTimeSupplierNeedsApproval},
			"approvalReference":              str(out.ApprovalReference)}))
	})
	return out, err
}

// RecordAudit implements Repository (refused approvals, outside a transaction).
func (s *PGStore) RecordAudit(ctx context.Context, ev audit.Event) error {
	return audit.Record(ctx, s.pool, ev)
}
