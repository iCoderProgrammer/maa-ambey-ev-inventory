package receiving

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/audit"
)

// PGStore is the PostgreSQL Repository. Each mutation is one transaction. A
// save locks the receipt (version, state), then its purchase order, so saves of
// different receipts of the same PO run one after the other and see each
// other's quantities. Nothing here touches purchase_order_lines.received_quantity,
// the PO status or any stock table: a draft changes no stock.
type PGStore struct{ pool *pgxpool.Pool }

// NewPGStore returns a Repository backed by pool.
func NewPGStore(pool *pgxpool.Pool) *PGStore { return &PGStore{pool: pool} }

// Totals are computed from the lines and units on every read.
const receiptColumns = `g.id::text, g.grn_number, g.status, g.purchase_order_id::text, po.po_number, po.status,
	po.supplier_id::text, s.legal_name, to_char(g.received_date, 'YYYY-MM-DD'), g.supplier_reference, g.notes,
	g.created_by::text, cu.display_name, g.cancelled_by::text, g.cancelled_at, g.cancel_reason, g.posted_by::text, g.posted_at,
	t.line_count, t.received, t.accepted, t.rejected,
	(SELECT count(*)::int FROM goods_receipt_units u WHERE u.goods_receipt_id = g.id),
	g.version, g.created_at, g.updated_at`

const receiptFrom = ` FROM goods_receipts g
	JOIN purchase_orders po ON po.id = g.purchase_order_id
	JOIN suppliers s ON s.id = po.supplier_id
	JOIN users cu ON cu.id = g.created_by
	CROSS JOIN LATERAL (
		SELECT count(*)::int AS line_count,
			coalesce(sum(gl.received_quantity), 0)::int AS received,
			coalesce(sum(gl.accepted_quantity), 0)::int AS accepted,
			coalesce(sum(gl.rejected_quantity), 0)::int AS rejected
		FROM goods_receipt_lines gl WHERE gl.goods_receipt_id = g.id) t`

func scanReceipt(row pgx.Row) (Receipt, error) {
	var r Receipt
	err := row.Scan(&r.ID, &r.Number, &r.Status, &r.PurchaseOrderID, &r.PurchaseOrderNumber, &r.PurchaseOrderStatus,
		&r.SupplierID, &r.SupplierName, &r.ReceivedDate, &r.SupplierReference, &r.Notes,
		&r.CreatedBy, &r.CreatedByName, &r.CancelledBy, &r.CancelledAt, &r.CancelReason, &r.PostedBy, &r.PostedAt,
		&r.LineCount, &r.ReceivedQuantity, &r.AcceptedQuantity, &r.RejectedQuantity, &r.UnitCount,
		&r.Version, &r.CreatedAt, &r.UpdatedAt)
	return r, err
}

// otherDrafts is what other open drafts of the PO hold on PO line l ($2: this receipt).
const otherDrafts = `coalesce((SELECT sum(ol.received_quantity) FROM goods_receipt_lines ol
		JOIN goods_receipts og ON og.id = ol.goods_receipt_id
		WHERE ol.purchase_order_line_id = l.id AND og.status = 'DRAFT' AND og.id <> $2), 0)::int`

const lineQuery = `SELECT gl.id::text, l.id::text, l.line_no, l.brand_id::text, b.name, m.name, v.name,
		l.quantity, l.received_quantity, ` + otherDrafts + `,
		coalesce(gl.received_quantity, 0), coalesce(gl.accepted_quantity, 0), coalesce(gl.rejected_quantity, 0), gl.discrepancy_note,
		gl.over_receipt_quantity, gl.over_receipt_reason, gl.over_receipt_by::text, ou.display_name, gl.over_receipt_at
	FROM purchase_order_lines l
	JOIN brands b ON b.id = l.brand_id
	JOIN models m ON m.id = l.model_id
	JOIN variants v ON v.id = l.variant_id
	LEFT JOIN goods_receipt_lines gl ON gl.purchase_order_line_id = l.id AND gl.goods_receipt_id = $2
	LEFT JOIN users ou ON ou.id = gl.over_receipt_by
	WHERE l.purchase_order_id = $1 ORDER BY l.line_no`

func scanLine(row pgx.Row) (Line, error) {
	var l Line
	var orQty *int
	var orReason, orBy, orByName *string
	var orAt *time.Time
	err := row.Scan(&l.ID, &l.PurchaseOrderLineID, &l.LineNo, &l.BrandID, &l.BrandName, &l.ModelName, &l.VariantName,
		&l.OrderedQuantity, &l.PostedQuantity, &l.OtherDraftsQuantity,
		&l.ReceivedQuantity, &l.AcceptedQuantity, &l.RejectedQuantity, &l.DiscrepancyNote,
		&orQty, &orReason, &orBy, &orByName, &orAt)
	if err != nil {
		return l, err
	}
	a := poLineAvailability{Ordered: l.OrderedQuantity, Posted: l.PostedQuantity, OtherDrafts: l.OtherDraftsQuantity}
	l.AvailableQuantity = a.available()
	l.ExcessQuantity = max(l.ReceivedQuantity-l.AvailableQuantity, 0)
	if orQty != nil {
		l.OverReceipt = &OverReceipt{Quantity: *orQty, Reason: *orReason, AuthorisedBy: *orBy, AuthorisedAt: *orAt}
		if orByName != nil {
			l.OverReceipt.AuthorisedByName = *orByName
		}
	}
	l.Units = []Unit{}
	return l, nil
}

type unitRow struct {
	LineID string
	Unit
}

func scanUnit(row pgx.Row) (unitRow, error) {
	var u unitRow
	err := row.Scan(&u.LineID, &u.ID, &u.UnitNo, &u.Condition, &u.ChassisNumber, &u.ChassisNormalised,
		&u.MotorSerial, &u.BatterySerial, &u.ChargerSerial, &u.Note)
	return u, err
}

const documentColumns = `d.id::text, d.attachment_id::text, d.document_type, d.note, a.original_name, a.content_type,
	d.added_by::text, u.display_name, d.created_at`

func scanDocument(row pgx.Row) (Document, error) {
	var d Document
	err := row.Scan(&d.ID, &d.AttachmentID, &d.DocumentType, &d.Note, &d.FileName, &d.ContentType, &d.AddedBy, &d.AddedByName, &d.CreatedAt)
	return d, err
}

const eventColumns = `e.id::text, e.action, e.from_status, e.to_status, e.actor_id::text, u.display_name, e.note, e.created_at`

func scanEvent(row pgx.Row) (Event, error) {
	var e Event
	err := row.Scan(&e.ID, &e.Action, &e.FromStatus, &e.ToStatus, &e.ActorID, &e.ActorName, &e.Note, &e.CreatedAt)
	return e, err
}

var duplicateFields = map[string]string{
	"goods_receipts_idempotency_key":         "Idempotency-Key",
	"goods_receipt_documents_attachment_key": "attachmentId",
	"goods_receipt_lines_po_line_key":        "purchaseOrderLineId",
	"goods_receipt_units_chassis_key":        "chassisNumber",
}

func mapPGError(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}
	switch pgErr.Code {
	case "23505":
		if f, ok := duplicateFields[pgErr.ConstraintName]; ok {
			return &DuplicateError{Fields: []FieldProblem{{f, "is already used"}}}
		}
	case "23503":
		if pgErr.ConstraintName == "goods_receipt_lines_po_line_fk" {
			return invalid("purchaseOrderLineId", "is not a line of this purchase order")
		}
	case "23514":
		return invalid("body", "violates goods receipt rule "+pgErr.ConstraintName)
	case "42501":
		return &StateError{Message: "This goods receipt record is history and cannot change."}
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

type querier interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

// queryAll runs a query and scans every row.
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

// ── Reads ────────────────────────────────────────────────────────────────────

// List implements Repository.
func (s *PGStore) List(ctx context.Context, f ListFilter) (Page[Receipt], error) {
	var conds []string
	var args []any
	add := func(cond string, v any) {
		args = append(args, v)
		conds = append(conds, strings.ReplaceAll(cond, "?", "$"+strconv.Itoa(len(args))))
	}
	if f.Status != "" {
		add("g.status = ?", f.Status)
	}
	if f.PurchaseOrderID != "" {
		add("g.purchase_order_id = ?", f.PurchaseOrderID)
	}
	if f.Query != "" {
		add(`(g.grn_number ILIKE ? ESCAPE '\' OR po.po_number ILIKE ? ESCAPE '\' OR s.legal_name ILIKE ? ESCAPE '\'
			OR g.supplier_reference ILIKE ? ESCAPE '\')`, likePattern(f.Query))
	}
	where := ""
	if len(conds) > 0 {
		where = " WHERE " + strings.Join(conds, " AND ")
	}
	var total int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM goods_receipts g JOIN purchase_orders po ON po.id = g.purchase_order_id
		JOIN suppliers s ON s.id = po.supplier_id`+where, args...).Scan(&total); err != nil {
		return Page[Receipt]{}, fmt.Errorf("count goods receipts: %w", err)
	}
	n := len(args)
	items, err := queryAll(ctx, s.pool, scanReceipt, `SELECT `+receiptColumns+receiptFrom+where+` ORDER BY g.created_at DESC, g.id LIMIT $`+
		strconv.Itoa(n+1)+` OFFSET $`+strconv.Itoa(n+2), append(args, f.Limit, (f.Page-1)*f.Limit)...)
	if err != nil {
		return Page[Receipt]{}, fmt.Errorf("list goods receipts: %w", err)
	}
	return Page[Receipt]{Items: items, Page: f.Page, Limit: f.Limit, Total: total}, nil
}

// ListReceivable implements Repository: APPROVED and PARTIALLY_RECEIVED POs.
func (s *PGStore) ListReceivable(ctx context.Context, f ListFilter) (Page[ReceivablePO], error) {
	where := ` WHERE po.status IN ('APPROVED', 'PARTIALLY_RECEIVED')`
	args := []any{}
	if f.Query != "" {
		args = append(args, likePattern(f.Query))
		where += ` AND (po.po_number ILIKE $1 ESCAPE '\' OR s.legal_name ILIKE $1 ESCAPE '\' OR s.trade_name ILIKE $1 ESCAPE '\')`
	}
	var total int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM purchase_orders po JOIN suppliers s ON s.id = po.supplier_id`+where, args...).
		Scan(&total); err != nil {
		return Page[ReceivablePO]{}, fmt.Errorf("count receivable purchase orders: %w", err)
	}
	n := len(args)
	items, err := queryAll(ctx, s.pool, func(r pgx.Row) (ReceivablePO, error) {
		var p ReceivablePO
		err := r.Scan(&p.ID, &p.Number, &p.Status, &p.SupplierID, &p.SupplierName, &p.ExpectedDate,
			&p.OrderedQuantity, &p.PostedQuantity, &p.DraftQuantity, &p.AvailableQuantity, &p.BrandNames, &p.OpenDraftCount)
		return p, err
	}, `SELECT po.id::text, po.po_number, po.status, po.supplier_id::text, s.legal_name, to_char(po.expected_date, 'YYYY-MM-DD'),
			t.ordered, t.posted, t.draft, t.available, t.brand_names,
			(SELECT count(*)::int FROM goods_receipts g WHERE g.purchase_order_id = po.id AND g.status = 'DRAFT')
		FROM purchase_orders po JOIN suppliers s ON s.id = po.supplier_id
		CROSS JOIN LATERAL (
			SELECT coalesce(sum(l.quantity), 0)::int AS ordered, coalesce(sum(l.received_quantity), 0)::int AS posted,
				coalesce(sum(x.draft), 0)::int AS draft,
				coalesce(sum(greatest(l.quantity - l.received_quantity - x.draft, 0)), 0)::int AS available,
				coalesce(array_agg(DISTINCT b.name ORDER BY b.name), '{}') AS brand_names
			FROM purchase_order_lines l JOIN brands b ON b.id = l.brand_id
			CROSS JOIN LATERAL (SELECT coalesce(sum(gl.received_quantity), 0)::int AS draft FROM goods_receipt_lines gl
				JOIN goods_receipts g ON g.id = gl.goods_receipt_id
				WHERE gl.purchase_order_line_id = l.id AND g.status = 'DRAFT') x
			WHERE l.purchase_order_id = po.id) t`+where+
		` ORDER BY po.approved_at DESC NULLS LAST, po.id LIMIT $`+strconv.Itoa(n+1)+` OFFSET $`+strconv.Itoa(n+2),
		append(args, f.Limit, (f.Page-1)*f.Limit)...)
	if err != nil {
		return Page[ReceivablePO]{}, fmt.Errorf("list receivable purchase orders: %w", err)
	}
	return Page[ReceivablePO]{Items: items, Page: f.Page, Limit: f.Limit, Total: total}, nil
}

// GetDetail implements Repository.
func (s *PGStore) GetDetail(ctx context.Context, id string) (Detail, error) {
	return loadDetail(ctx, s.pool, id)
}

func loadDetail(ctx context.Context, q querier, id string) (Detail, error) {
	var d Detail
	r, err := scanReceipt(q.QueryRow(ctx, `SELECT `+receiptColumns+receiptFrom+` WHERE g.id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return d, ErrNotFound
	}
	if err != nil {
		return d, fmt.Errorf("get goods receipt: %w", err)
	}
	d.Receipt = r
	d.PurchaseOrderReceivable = poReceivable(r.PurchaseOrderStatus)
	if d.Lines, err = queryAll(ctx, q, scanLine, lineQuery, r.PurchaseOrderID, id); err != nil {
		return d, fmt.Errorf("lines: %w", err)
	}
	units, err := queryAll(ctx, q, scanUnit, `SELECT u.goods_receipt_line_id::text, u.id::text, u.unit_no, u.condition, u.chassis_number,
			u.chassis_normalised, u.motor_serial, u.battery_serial, u.charger_serial, u.note
		FROM goods_receipt_units u WHERE u.goods_receipt_id = $1 ORDER BY u.goods_receipt_line_id, u.unit_no`, id)
	if err != nil {
		return d, fmt.Errorf("units: %w", err)
	}
	for i := range d.Lines {
		l := &d.Lines[i]
		for _, u := range units {
			if l.ID != nil && u.LineID == *l.ID {
				l.Units = append(l.Units, u.Unit)
			}
		}
	}
	if d.Documents, err = queryAll(ctx, q, scanDocument, `SELECT `+documentColumns+` FROM goods_receipt_documents d
		JOIN attachments a ON a.id = d.attachment_id JOIN users u ON u.id = d.added_by
		WHERE d.goods_receipt_id = $1 ORDER BY d.created_at, d.id`, id); err != nil {
		return d, fmt.Errorf("documents: %w", err)
	}
	if d.Events, err = queryAll(ctx, q, scanEvent, `SELECT `+eventColumns+` FROM goods_receipt_events e
		LEFT JOIN users u ON u.id = e.actor_id WHERE e.goods_receipt_id = $1 ORDER BY e.seq`, id); err != nil {
		return d, fmt.Errorf("events: %w", err)
	}
	return d, nil
}

// ── Writes ───────────────────────────────────────────────────────────────────

type receiptRow struct {
	ID, Number, Status, PurchaseOrderID string
	Version                             int
}

// lockReceipt locks the receipt and checks the version the caller last read.
func lockReceipt(ctx context.Context, tx pgx.Tx, id string, version int) (receiptRow, error) {
	var r receiptRow
	err := tx.QueryRow(ctx, `SELECT id::text, grn_number, status, purchase_order_id::text, version
		FROM goods_receipts WHERE id = $1 FOR UPDATE`, id).Scan(&r.ID, &r.Number, &r.Status, &r.PurchaseOrderID, &r.Version)
	if errors.Is(err, pgx.ErrNoRows) {
		return r, ErrNotFound
	}
	if err != nil {
		return r, err
	}
	if version > 0 && r.Version != version {
		return r, ErrVersionConflict
	}
	return r, nil
}

// lockPO locks the purchase order so receipt quantities against it are decided
// one save at a time (and the PO cannot be cancelled meanwhile).
func lockPO(ctx context.Context, tx pgx.Tx, id string) (number, status string, err error) {
	err = tx.QueryRow(ctx, `SELECT po_number, status FROM purchase_orders WHERE id = $1 FOR UPDATE`, id).Scan(&number, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", invalid("purchaseOrderId", "does not exist")
	}
	return number, status, err
}

func checkReceivedDate(ctx context.Context, tx pgx.Tx, date string) error {
	var future bool
	if err := tx.QueryRow(ctx, `SELECT $1::date > current_date`, date).Scan(&future); err != nil {
		return err
	}
	if future {
		return invalid("receivedDate", "cannot be in the future")
	}
	return nil
}

func addEvent(ctx context.Context, tx pgx.Tx, receiptID, action string, from *string, to, actorID string, note *string) error {
	_, err := tx.Exec(ctx, `INSERT INTO goods_receipt_events (goods_receipt_id, action, from_status, to_status, actor_id, note)
		VALUES ($1, $2, $3, $4, $5, $6)`, receiptID, action, str(from), to, actorID, str(note))
	return err
}

// Create implements Repository. With an idempotency key, a repeat by the same
// user returns the receipt created first (replayed = true).
func (s *PGStore) Create(ctx context.Context, in CreateInput, createdBy, idemKey string, ev audit.Event) (string, bool, error) {
	if idemKey != "" {
		if id, ok, err := s.findByKey(ctx, createdBy, idemKey); err != nil || ok {
			return id, ok, err
		}
	}
	var id string
	err := s.tx(ctx, func(tx pgx.Tx) error {
		poNumber, poStatus, err := lockPO(ctx, tx, in.PurchaseOrderID)
		if err != nil {
			return err
		}
		if !poReceivable(poStatus) {
			return invalid("purchaseOrderId", "must be an approved purchase order (this one is "+poStatus+")")
		}
		if err := checkReceivedDate(ctx, tx, in.ReceivedDate); err != nil {
			return err
		}
		var key any
		if idemKey != "" {
			key = idemKey
		}
		var number string
		if err := tx.QueryRow(ctx, `INSERT INTO goods_receipts (purchase_order_id, received_date, supplier_reference, notes, created_by, idempotency_key)
			VALUES ($1, $2::date, $3, $4, $5, $6) RETURNING id::text, grn_number`,
			in.PurchaseOrderID, in.ReceivedDate, str(in.SupplierReference), str(in.Notes), createdBy, key).Scan(&id, &number); err != nil {
			return err
		}
		if err := addEvent(ctx, tx, id, ActionCreated, nil, StatusDraft, createdBy, nil); err != nil {
			return err
		}
		ev.EntityID = id
		return audit.Record(ctx, tx, withSummary(ev, audit.Summary{"number": number, "purchaseOrderId": in.PurchaseOrderID,
			"purchaseOrderNumber": poNumber, "receivedDate": in.ReceivedDate, "supplierReference": str(in.SupplierReference)}))
	})
	var dup *DuplicateError
	if idemKey != "" && errors.As(err, &dup) && dup.Fields[0].Field == "Idempotency-Key" {
		// A concurrent request with the same key won.
		if id, ok, ferr := s.findByKey(ctx, createdBy, idemKey); ferr == nil && ok {
			return id, true, nil
		}
	}
	return id, false, err
}

func (s *PGStore) findByKey(ctx context.Context, createdBy, key string) (string, bool, error) {
	var id string
	err := s.pool.QueryRow(ctx, `SELECT id::text FROM goods_receipts WHERE created_by = $1 AND idempotency_key = $2`, createdBy, key).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	return id, err == nil, err
}

// availability reads every PO line's quantities as seen by receipt id (PO locked).
func availability(ctx context.Context, tx pgx.Tx, poID, receiptID string) (map[string]poLineAvailability, error) {
	list, err := queryAll(ctx, tx, func(r pgx.Row) (poLineAvailability, error) {
		var a poLineAvailability
		err := r.Scan(&a.ID, &a.LineNo, &a.Ordered, &a.Posted, &a.OtherDrafts)
		return a, err
	}, `SELECT l.id::text, l.line_no, l.quantity, l.received_quantity, `+otherDrafts+`
		FROM purchase_order_lines l WHERE l.purchase_order_id = $1`, poID, receiptID)
	if err != nil {
		return nil, err
	}
	out := make(map[string]poLineAvailability, len(list))
	for _, a := range list {
		out[a.ID] = a
	}
	return out, nil
}

func existingOverrides(ctx context.Context, tx pgx.Tx, receiptID string) (map[string]storedOverride, error) {
	type row struct {
		POLineID string
		storedOverride
	}
	list, err := queryAll(ctx, tx, func(r pgx.Row) (row, error) {
		var o row
		err := r.Scan(&o.POLineID, &o.Quantity, &o.Reason, &o.By, &o.At)
		return o, err
	}, `SELECT purchase_order_line_id::text, over_receipt_quantity, over_receipt_reason, over_receipt_by::text, over_receipt_at
		FROM goods_receipt_lines WHERE goods_receipt_id = $1 AND over_receipt_quantity IS NOT NULL`, receiptID)
	if err != nil {
		return nil, err
	}
	out := make(map[string]storedOverride, len(list))
	for _, o := range list {
		out[o.POLineID] = o.storedOverride
	}
	return out, nil
}

// checkChassisElsewhere refuses chassis/frame numbers already on another open
// or posted receipt. Each number is locked (transaction advisory lock, in a
// fixed order) first, so two receipts saving the same number at once cannot
// both pass. Registered vehicles (D-04) are checked at posting (D-07).
func checkChassisElsewhere(ctx context.Context, tx pgx.Tx, receiptID string, lines []lineDecision) error {
	paths := map[string]string{}
	for _, l := range lines {
		for j, u := range l.Units {
			if u.ChassisNumber != nil {
				paths[NormaliseChassis(*u.ChassisNumber)] = l.path("units[" + strconv.Itoa(j) + "].chassisNumber")
			}
		}
	}
	if len(paths) == 0 {
		return nil
	}
	numbers := make([]string, 0, len(paths))
	for n := range paths {
		numbers = append(numbers, n)
	}
	slices.Sort(numbers)
	for _, n := range numbers {
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('goods_receipt_chassis:' || $1, 0))`, n); err != nil {
			return err
		}
	}
	type hit struct{ Chassis, Number, Status string }
	hits, err := queryAll(ctx, tx, func(r pgx.Row) (hit, error) {
		var h hit
		err := r.Scan(&h.Chassis, &h.Number, &h.Status)
		return h, err
	}, `SELECT u.chassis_normalised, g.grn_number, g.status FROM goods_receipt_units u
		JOIN goods_receipts g ON g.id = u.goods_receipt_id
		WHERE g.id <> $1 AND g.status IN ('DRAFT', 'POSTED') AND u.chassis_normalised = ANY($2)
		ORDER BY g.grn_number`, receiptID, numbers)
	if err != nil {
		return err
	}
	if len(hits) == 0 {
		return nil
	}
	seen := map[string]bool{}
	var fields []FieldProblem
	for _, h := range hits {
		if seen[h.Chassis] {
			continue
		}
		seen[h.Chassis] = true
		where := "the draft goods receipt " + h.Number
		if h.Status == StatusPosted {
			where = "the posted goods receipt " + h.Number
		}
		fields = append(fields, FieldProblem{paths[h.Chassis], "is already on " + where})
	}
	slices.SortFunc(fields, func(a, b FieldProblem) int { return strings.Compare(a.Field, b.Field) })
	return &DuplicateError{Fields: fields}
}

// Save implements Repository: replaces the draft's header, lines and units in
// one transaction. Lines above availability need an authorisation
// (decideOverReceipt); identifiers must be free (checkChassisElsewhere).
func (s *PGStore) Save(ctx context.Context, id string, in SaveInput, lines []indexedLine, actorID string, canOverride bool, ev audit.Event) error {
	return s.tx(ctx, func(tx pgx.Tx) error {
		old, err := lockReceipt(ctx, tx, id, in.Version)
		if err != nil {
			return err
		}
		poNumber, poStatus, err := lockPO(ctx, tx, old.PurchaseOrderID)
		if err != nil {
			return err
		}
		if err := checkEditable(old.Status, poStatus); err != nil {
			return err
		}
		if err := checkReceivedDate(ctx, tx, in.ReceivedDate); err != nil {
			return err
		}
		avail, err := availability(ctx, tx, old.PurchaseOrderID, id)
		if err != nil {
			return err
		}
		prev, err := existingOverrides(ctx, tx, id)
		if err != nil {
			return err
		}
		decisions, err := decideOverReceipt(lines, avail, prev, canOverride, actorID)
		if err != nil {
			return err
		}
		if err := checkChassisElsewhere(ctx, tx, id, decisions); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `DELETE FROM goods_receipt_units WHERE goods_receipt_id = $1`, id); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `DELETE FROM goods_receipt_lines WHERE goods_receipt_id = $1`, id); err != nil {
			return err
		}
		var received, accepted, rejected, units int
		lineSums := make([]any, 0, len(decisions))
		var authorised []lineDecision
		for _, d := range decisions {
			var orQty, orReason, orBy, orAt any
			if o := d.Override; o != nil {
				orQty, orReason, orBy = o.Quantity, o.Reason, o.By
				if o.At != nil {
					orAt = *o.At
				}
			}
			var lineID string
			if err := tx.QueryRow(ctx, `INSERT INTO goods_receipt_lines (goods_receipt_id, purchase_order_id, purchase_order_line_id,
					received_quantity, accepted_quantity, rejected_quantity, discrepancy_note,
					over_receipt_quantity, over_receipt_reason, over_receipt_by, over_receipt_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, CASE WHEN $8::int IS NULL THEN NULL ELSE coalesce($11::timestamptz, now()) END)
				RETURNING id::text`,
				id, old.PurchaseOrderID, d.PurchaseOrderLineID, d.ReceivedQuantity, d.AcceptedQuantity, d.RejectedQuantity,
				str(d.DiscrepancyNote), orQty, orReason, orBy, orAt).Scan(&lineID); err != nil {
				return err
			}
			for j, u := range d.Units {
				var normalised any
				if u.ChassisNumber != nil {
					normalised = NormaliseChassis(*u.ChassisNumber)
				}
				if _, err := tx.Exec(ctx, `INSERT INTO goods_receipt_units (goods_receipt_id, goods_receipt_line_id, unit_no, condition,
						chassis_number, chassis_normalised, motor_serial, battery_serial, charger_serial, note)
					VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
					id, lineID, j+1, u.Condition, str(u.ChassisNumber), normalised, str(u.MotorSerial), str(u.BatterySerial),
					str(u.ChargerSerial), str(u.Note)); err != nil {
					return err
				}
			}
			received += d.ReceivedQuantity
			accepted += d.AcceptedQuantity
			rejected += d.RejectedQuantity
			units += len(d.Units)
			lineSums = append(lineSums, map[string]any{"purchaseOrderLineId": d.PurchaseOrderLineID, "received": d.ReceivedQuantity,
				"accepted": d.AcceptedQuantity, "rejected": d.RejectedQuantity, "units": len(d.Units), "overReceipt": d.Override != nil})
			if d.NewOverride {
				authorised = append(authorised, d)
			}
		}
		if _, err := tx.Exec(ctx, `UPDATE goods_receipts SET received_date = $2::date, supplier_reference = $3, notes = $4,
			version = version + 1 WHERE id = $1`, id, in.ReceivedDate, str(in.SupplierReference), str(in.Notes)); err != nil {
			return err
		}
		for _, d := range authorised {
			a := avail[d.PurchaseOrderLineID]
			oev := ev
			oev.Action = "goods_receipt.over_receipt_authorised"
			oev.Summary = nil
			if err := audit.Record(ctx, tx, withSummary(oev, audit.Summary{"number": old.Number, "purchaseOrderNumber": poNumber,
				"purchaseOrderLineId": d.PurchaseOrderLineID, "received": d.ReceivedQuantity, "available": a.available(),
				"excess": d.Excess, "reason": *d.OverReceiptReason})); err != nil {
				return err
			}
		}
		return audit.Record(ctx, tx, withSummary(ev, audit.Summary{"number": old.Number, "purchaseOrderNumber": poNumber,
			"receivedDate": in.ReceivedDate, "received": received, "accepted": accepted, "rejected": rejected, "units": units,
			"lines": lineSums, "stockChanged": false}))
	})
}

// Cancel implements Repository: DRAFT -> CANCELLED with a reason; everything
// stays as history and no longer holds PO quantity.
func (s *PGStore) Cancel(ctx context.Context, id string, in CancelInput, actorID string, ev audit.Event) error {
	return s.tx(ctx, func(tx pgx.Tx) error {
		old, err := lockReceipt(ctx, tx, id, in.Version)
		if err != nil {
			return err
		}
		if err := checkCancel(old.Status); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE goods_receipts SET status = 'CANCELLED', cancelled_by = $2, cancelled_at = now(),
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

// AddDocument implements Repository: links one of the actor's own unlinked
// uploads to a draft (the attachment's owner is set once and never changes).
func (s *PGStore) AddDocument(ctx context.Context, id string, in DocumentInput, actorID string, ev audit.Event) error {
	return s.tx(ctx, func(tx pgx.Tx) error {
		old, err := lockReceipt(ctx, tx, id, 0)
		if err != nil {
			return err
		}
		if old.Status != StatusDraft {
			return &StateError{Message: "Documents can be added to draft goods receipts only."}
		}
		var uploader string
		var owner *string
		err = tx.QueryRow(ctx, `SELECT uploaded_by::text, owner_type FROM attachments WHERE id = $1 FOR UPDATE`, in.AttachmentID).
			Scan(&uploader, &owner)
		if errors.Is(err, pgx.ErrNoRows) || (err == nil && (uploader != actorID || owner != nil)) {
			return invalid("attachmentId", "must be one of your own uploads that is not attached to anything yet")
		}
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE attachments SET owner_type = 'goods_receipt', owner_id = $2 WHERE id = $1`, in.AttachmentID, id); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO goods_receipt_documents (goods_receipt_id, attachment_id, document_type, note, added_by)
			VALUES ($1, $2, $3, $4, $5)`, id, in.AttachmentID, in.DocumentType, str(in.Note), actorID); err != nil {
			return err
		}
		return audit.Record(ctx, tx, withSummary(ev, audit.Summary{"number": old.Number, "attachmentId": in.AttachmentID,
			"documentType": in.DocumentType}))
	})
}

// RecordAudit implements Repository (refused over-receipts, outside a transaction).
func (s *PGStore) RecordAudit(ctx context.Context, ev audit.Event) error {
	return audit.Record(ctx, s.pool, ev)
}
