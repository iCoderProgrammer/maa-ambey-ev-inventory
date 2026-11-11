package pdi

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/audit"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/access"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/stock"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/vehicles"
)

// PGStore is the PostgreSQL Repository.
type PGStore struct{ pool *pgxpool.Pool }

// NewPGStore returns a Repository backed by pool.
func NewPGStore(pool *pgxpool.Pool) *PGStore { return &PGStore{pool: pool} }

type querier interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

// ── Reads ───────────────────────────────────────────────────────────────────

func readUnit(ctx context.Context, q querier, id string, lock bool) (UnitState, bool, error) {
	sql := `SELECT id::text, unit_number, status, brand_id::text, location_id::text, version, chassis_normalised IS NOT NULL
		FROM vehicle_units WHERE id = $1`
	if lock {
		sql += ` FOR UPDATE`
	}
	var u UnitState
	var hasChassis bool
	err := q.QueryRow(ctx, sql, id).Scan(&u.ID, &u.UnitNumber, &u.Status, &u.BrandID, &u.LocationID, &u.Version, &hasChassis)
	if errors.Is(err, pgx.ErrNoRows) {
		return UnitState{}, false, ErrNotFound
	}
	if err != nil {
		return UnitState{}, false, fmt.Errorf("read vehicle unit: %w", err)
	}
	u.Saleable = vehicles.IsSaleable(u.Status)
	return u, hasChassis, nil
}

func inScope(scope access.Scope, location *string) bool {
	// Units without a location are in every scope (vehicles, E-01 convention).
	return location == nil || scope.Allows(*location)
}

const checklistColumns = `c.id::text, c.code, c.label, c.description, c.brand_id::text, b.name, c.sort_order, c.status, c.version,
	c.created_by::text, c.created_at, c.retired_by::text, c.retired_at, c.retire_reason`

const checklistFrom = ` FROM pdi_checklist_items c LEFT JOIN brands b ON b.id = c.brand_id`

func scanChecklist(r pgx.Row) (ChecklistItem, error) {
	var c ChecklistItem
	err := r.Scan(&c.ID, &c.Code, &c.Label, &c.Description, &c.BrandID, &c.BrandName, &c.SortOrder, &c.Status, &c.Version,
		&c.CreatedBy, &c.CreatedAt, &c.RetiredBy, &c.RetiredAt, &c.RetireReason)
	return c, err
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

// applicable is the active checklist for a brand: every-brand items and the
// brand's own, in order.
func applicable(ctx context.Context, q querier, brandID string, lock bool) ([]ChecklistItem, error) {
	sql := `SELECT ` + checklistColumns + checklistFrom + ` WHERE c.status = 'ACTIVE' AND (c.brand_id IS NULL OR c.brand_id = $1)
		ORDER BY c.sort_order, c.code, c.id`
	if lock {
		// A concurrent retirement waits for this inspection (and vice versa).
		sql += ` FOR SHARE OF c`
	}
	return queryAll(ctx, q, scanChecklist, sql, brandID)
}

const inspectionColumns = `i.id::text, i.inspection_number, i.vehicle_unit_id::text, v.unit_number, i.disposition, i.result,
	i.from_status, i.to_status, i.location_id::text, i.notes, i.movement_id::text, i.inspected_by::text, u.display_name,
	i.inspected_at,
	(SELECT count(*) FROM pdi_inspection_items x WHERE x.inspection_id = i.id)::int,
	(SELECT count(*) FROM pdi_inspection_items x WHERE x.inspection_id = i.id AND x.outcome = 'FAIL')::int,
	(SELECT count(*) FROM pdi_inspection_attachments x WHERE x.inspection_id = i.id)::int`

const inspectionFrom = ` FROM pdi_inspections i JOIN vehicle_units v ON v.id = i.vehicle_unit_id
	LEFT JOIN users u ON u.id = i.inspected_by`

func scanInspection(r pgx.Row) (Inspection, error) {
	var i Inspection
	err := r.Scan(&i.ID, &i.InspectionNumber, &i.VehicleUnitID, &i.UnitNumber, &i.Disposition, &i.Result, &i.FromStatus,
		&i.ToStatus, &i.LocationID, &i.Notes, &i.MovementID, &i.InspectedBy, &i.InspectedByName, &i.InspectedAt,
		&i.ItemCount, &i.FailedCount, &i.EvidenceCount)
	return i, err
}

// detail reads an inspection with outcomes and evidence, and the location of its unit.
func detail(ctx context.Context, q querier, id string) (InspectionDetail, *string, error) {
	var d InspectionDetail
	var unitLocation *string
	row := q.QueryRow(ctx, `SELECT `+inspectionColumns+`, v.location_id::text`+inspectionFrom+` WHERE i.id = $1`, id)
	err := row.Scan(&d.ID, &d.InspectionNumber, &d.VehicleUnitID, &d.UnitNumber, &d.Disposition, &d.Result, &d.FromStatus,
		&d.ToStatus, &d.LocationID, &d.Notes, &d.MovementID, &d.InspectedBy, &d.InspectedByName, &d.InspectedAt,
		&d.ItemCount, &d.FailedCount, &d.EvidenceCount, &unitLocation)
	if errors.Is(err, pgx.ErrNoRows) {
		return InspectionDetail{}, nil, ErrNotFound
	}
	if err != nil {
		return InspectionDetail{}, nil, fmt.Errorf("read inspection: %w", err)
	}
	d.Items, err = queryAll(ctx, q, func(r pgx.Row) (ItemOutcome, error) {
		var o ItemOutcome
		err := r.Scan(&o.LineNo, &o.ChecklistItemID, &o.Code, &o.Label, &o.Outcome, &o.Note)
		return o, err
	}, `SELECT x.line_no, x.checklist_item_id::text, c.code, c.label, x.outcome, x.note FROM pdi_inspection_items x
		JOIN pdi_checklist_items c ON c.id = x.checklist_item_id WHERE x.inspection_id = $1 ORDER BY x.line_no`, id)
	if err != nil {
		return InspectionDetail{}, nil, fmt.Errorf("read outcomes: %w", err)
	}
	d.Evidence, err = queryAll(ctx, q, func(r pgx.Row) (Evidence, error) {
		var e Evidence
		err := r.Scan(&e.AttachmentID, &e.OriginalName, &e.ContentType, &e.SizeBytes)
		return e, err
	}, `SELECT a.id::text, a.original_name, a.content_type, a.size_bytes FROM pdi_inspection_attachments l
		JOIN attachments a ON a.id = l.attachment_id WHERE l.inspection_id = $1 ORDER BY a.created_at, a.id`, id)
	if err != nil {
		return InspectionDetail{}, nil, fmt.Errorf("read evidence: %w", err)
	}
	return d, unitLocation, nil
}

// Overview implements Repository.
func (s *PGStore) Overview(ctx context.Context, scope access.Scope, unitID string) (UnitState, []ChecklistItem, []Inspection, error) {
	u, _, err := readUnit(ctx, s.pool, unitID, false)
	if err != nil {
		return UnitState{}, nil, nil, err
	}
	if !inScope(scope, u.LocationID) {
		return UnitState{}, nil, nil, ErrNotFound
	}
	items, err := applicable(ctx, s.pool, u.BrandID, false)
	if err != nil {
		return UnitState{}, nil, nil, fmt.Errorf("read checklist: %w", err)
	}
	history, err := queryAll(ctx, s.pool, scanInspection, `SELECT `+inspectionColumns+inspectionFrom+
		` WHERE i.vehicle_unit_id = $1 ORDER BY i.inspected_at DESC, i.inspection_number DESC`, unitID)
	if err != nil {
		return UnitState{}, nil, nil, fmt.Errorf("read inspections: %w", err)
	}
	return u, items, history, nil
}

// Inspection implements Repository.
func (s *PGStore) Inspection(ctx context.Context, scope access.Scope, id string) (InspectionDetail, error) {
	d, loc, err := detail(ctx, s.pool, id)
	if err != nil {
		return InspectionDetail{}, err
	}
	if !inScope(scope, loc) {
		return InspectionDetail{}, ErrNotFound
	}
	return d, nil
}

// ── Recording ───────────────────────────────────────────────────────────────

// stateError explains why the unit's status does not admit the disposition.
func stateError(status, disposition string) *StateError {
	msg := "A unit that is " + status + " cannot be inspected: PDI applies to units awaiting PDI (and rejecting units on quality hold)."
	switch {
	case status == "RECEIVED_PENDING_PDI" && disposition == DispositionReject:
		msg = "A unit awaiting PDI cannot be rejected directly: put it on hold first, then reject it. " +
			"Whether a failed PDI may go straight to REJECTED is not decided yet (OQ-31)."
	case status == "QUALITY_HOLD" && disposition == DispositionPass:
		msg = "A unit on quality hold cannot be passed back to available stock: re-inspection and release are not " +
			"approved yet (OQ-31). It can only be rejected."
	case status == "QUALITY_HOLD" && disposition == DispositionHold:
		msg = "This unit is already on quality hold. It can only be rejected."
	}
	return &StateError{Status: status, Disposition: disposition, Message: msg}
}

func lookupKey(ctx context.Context, q querier, actorID, key string) (id, hash string, found bool, err error) {
	err = q.QueryRow(ctx, `SELECT id::text, request_hash FROM pdi_inspections WHERE inspected_by = $1 AND idempotency_key = $2`,
		actorID, key).Scan(&id, &hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", false, nil
	}
	if err != nil {
		return "", "", false, fmt.Errorf("look up idempotency key: %w", err)
	}
	return id, hash, true, nil
}

func result(q querier, ctx context.Context, inspectionID, unitID string) (RecordResult, error) {
	d, _, err := detail(ctx, q, inspectionID)
	if err != nil {
		return RecordResult{}, err
	}
	u, _, err := readUnit(ctx, q, unitID, false)
	if err != nil {
		return RecordResult{}, err
	}
	return RecordResult{Inspection: d, Unit: u}, nil
}

// Record implements Repository: one transaction locks the unit, replays a
// repeated key, checks version, status, checklist and evidence, records the
// movement through stock.MoveUnit (source PDI_INSPECTION) and the inspection,
// links the evidence and audits. A concurrent repeat of the same key waits for
// the unit lock and is replayed.
func (s *PGStore) Record(ctx context.Context, actor audit.Actor, scope access.Scope, requestID, unitID string,
	r normalisedRecord, key, hash string) (RecordResult, bool, error) {
	var res RecordResult
	var replayed bool
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		var err error
		res, replayed, err = record(ctx, tx, actor, scope, requestID, unitID, r, key, hash)
		return err
	})
	if errors.Is(err, errKeyRace) {
		id, stored, found, lerr := lookupKey(ctx, s.pool, actor.UserID, key)
		switch {
		case lerr != nil:
			return RecordResult{}, false, lerr
		case !found:
			return RecordResult{}, false, errors.New("idempotency key conflict without a committed inspection")
		case stored != hash:
			return RecordResult{}, false, ErrIdempotencyKeyReused
		}
		res, err = result(s.pool, ctx, id, unitID)
		return res, err == nil, err
	}
	if err != nil {
		return RecordResult{}, false, err
	}
	return res, replayed, nil
}

func record(ctx context.Context, tx pgx.Tx, actor audit.Actor, scope access.Scope, requestID, unitID string,
	r normalisedRecord, key, hash string) (RecordResult, bool, error) {
	if actor.Type != audit.ActorUser || actor.UserID == "" {
		return RecordResult{}, false, errors.New("PDI inspections are recorded for a signed-in user")
	}
	u, hasChassis, err := readUnit(ctx, tx, unitID, true)
	if err != nil {
		return RecordResult{}, false, err
	}
	if !inScope(scope, u.LocationID) {
		return RecordResult{}, false, ErrNotFound
	}
	// After the lock: a concurrent request with the same key has committed by now.
	if id, stored, found, err := lookupKey(ctx, tx, actor.UserID, key); err != nil {
		return RecordResult{}, false, err
	} else if found {
		if stored != hash {
			return RecordResult{}, false, ErrIdempotencyKeyReused
		}
		res, err := result(tx, ctx, id, unitID)
		return res, err == nil, err
	}
	if r.Version != u.Version {
		return RecordResult{}, false, ErrVersionConflict
	}
	edge := Edges[r.Disposition]
	if u.Status != edge.From {
		return RecordResult{}, false, stateError(u.Status, r.Disposition)
	}
	if edge.From == "RECEIVED_PENDING_PDI" && !hasChassis {
		return RecordResult{}, false, ErrChassisRequired
	}

	checklist, err := applicable(ctx, tx, u.BrandID, true)
	if err != nil {
		return RecordResult{}, false, fmt.Errorf("read checklist: %w", err)
	}
	if r.Disposition == DispositionPass && len(checklist) == 0 {
		return RecordResult{}, false, ErrChecklistNotConfigured
	}
	if err := matchChecklist(checklist, r.Items); err != nil {
		return RecordResult{}, false, err
	}
	if err := checkEvidence(ctx, tx, actor.UserID, u.LocationID, r.AttachmentIDs); err != nil {
		return RecordResult{}, false, err
	}

	id := uuid.NewString()
	var seq int64
	if err := tx.QueryRow(ctx, `SELECT nextval('pdi_inspection_number_seq')`).Scan(&seq); err != nil {
		return RecordResult{}, false, fmt.Errorf("next inspection number: %w", err)
	}
	number := fmt.Sprintf("PDI-%06d", seq)
	version := r.Version
	moved, _, err := stock.MoveUnit(ctx, tx, actor, scope, stock.MoveRequest{UnitID: u.ID, Type: edge.MovementType,
		ExpectedVersion: &version, IdempotencyKey: "pdi-" + id, RequestID: requestID,
		Source: &stock.MoveSource{Type: stock.SourcePDIInspection, ID: id, Reference: number}})
	if err != nil {
		return RecordResult{}, false, moveError(err, u.Status, r.Disposition)
	}

	outcomes := map[string]ItemOutcomeInput{}
	failed := 0
	for _, it := range r.Items {
		outcomes[it.ChecklistItemID] = it
		if it.Outcome == OutcomeFail {
			failed++
		}
	}
	res := ResultAllPassed
	switch {
	case len(checklist) == 0:
		res = ResultNotChecked
	case failed > 0:
		res = ResultHasFailures
	}
	if _, err := tx.Exec(ctx, `INSERT INTO pdi_inspections (id, inspection_number, vehicle_unit_id, disposition, result,
			from_status, to_status, location_id, notes, movement_id, inspected_by, request_id, idempotency_key, request_hash)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
		id, number, u.ID, r.Disposition, res, edge.From, edge.To, u.LocationID, r.Notes, moved.Movement.ID, actor.UserID,
		nullable(requestID), key, hash); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "pdi_inspections_idempotency_key" {
			return RecordResult{}, false, errKeyRace
		}
		return RecordResult{}, false, fmt.Errorf("record inspection: %w", err)
	}
	for i, c := range checklist {
		o := outcomes[c.ID]
		if _, err := tx.Exec(ctx, `INSERT INTO pdi_inspection_items (inspection_id, checklist_item_id, line_no, outcome, note)
			VALUES ($1, $2, $3, $4, $5)`, id, c.ID, i+1, o.Outcome, o.Note); err != nil {
			return RecordResult{}, false, fmt.Errorf("record outcome: %w", err)
		}
	}
	for _, a := range r.AttachmentIDs {
		if _, err := tx.Exec(ctx, `UPDATE attachments SET owner_type = $2, owner_id = $3 WHERE id = $1`, a, AttachmentOwner, id); err != nil {
			return RecordResult{}, false, fmt.Errorf("link evidence: %w", err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO pdi_inspection_attachments (inspection_id, attachment_id) VALUES ($1, $2)`, id, a); err != nil {
			return RecordResult{}, false, fmt.Errorf("link evidence: %w", err)
		}
	}
	var location any
	if u.LocationID != nil {
		location = *u.LocationID
	}
	if err := audit.Record(ctx, tx, audit.Event{Actor: actor, Action: "pdi_inspection.recorded", EntityType: "pdi_inspection",
		EntityID: id, Reference: number, RequestID: requestID, Summary: audit.Summary{
			"vehicleUnitId": u.ID, "unitNumber": u.UnitNumber, "disposition": r.Disposition, "result": res,
			"status": audit.Change{From: edge.From, To: edge.To}, "movementId": moved.Movement.ID, "locationId": location,
			"itemCount": len(checklist), "failedCount": failed, "evidenceCount": len(r.AttachmentIDs),
		}}); err != nil {
		return RecordResult{}, false, err
	}
	out, err := result(tx, ctx, id, u.ID)
	return out, false, err
}

// matchChecklist requires exactly one outcome for every applicable item and
// nothing else (no unknown, retired or other-brand item).
func matchChecklist(checklist []ChecklistItem, items []ItemOutcomeInput) error {
	var c checker
	want := map[string]bool{}
	for _, it := range checklist {
		want[it.ID] = true
	}
	given := map[string]bool{}
	for i, it := range items {
		given[it.ChecklistItemID] = true
		if !want[it.ChecklistItemID] {
			c.add("items["+strconv.Itoa(i)+"].checklistItemId", "is not an active checklist item for this unit")
		}
	}
	for _, it := range checklist {
		if !given[it.ID] {
			c.add("items", "need an outcome for "+it.Code+" ("+it.Label+")")
		}
	}
	return c.err()
}

// checkEvidence requires each attachment to be one of the actor's own unlinked
// uploads at the unit's location (locks them until commit).
func checkEvidence(ctx context.Context, tx pgx.Tx, actorID string, unitLocation *string, ids []string) error {
	var c checker
	for i, id := range ids {
		var uploader string
		var owner, location *string
		err := tx.QueryRow(ctx, `SELECT uploaded_by::text, owner_type, location_id::text FROM attachments WHERE id = $1 FOR UPDATE`,
			id).Scan(&uploader, &owner, &location)
		f := "attachmentIds[" + strconv.Itoa(i) + "]"
		switch {
		case errors.Is(err, pgx.ErrNoRows) || (err == nil && (uploader != actorID || owner != nil)):
			c.add(f, "must be one of your own uploads that is not attached to anything yet")
		case err != nil:
			return fmt.Errorf("read attachment: %w", err)
		case !sameLocation(location, unitLocation):
			c.add(f, "must be uploaded for the vehicle unit's location")
		}
	}
	return c.err()
}

func sameLocation(a, b *string) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

// moveError maps errors of stock.MoveUnit (the unit was checked just before,
// under the same lock, so these are defensive).
func moveError(err error, status, disposition string) error {
	var inv *stock.InvalidStateError
	switch {
	case errors.Is(err, stock.ErrNotFound):
		return ErrNotFound
	case errors.Is(err, stock.ErrVersionConflict):
		return ErrVersionConflict
	case errors.Is(err, stock.ErrChassisRequired):
		return ErrChassisRequired
	case errors.As(err, &inv):
		return stateError(inv.Current, disposition)
	}
	return fmt.Errorf("record PDI movement (%s, %s): %w", status, disposition, err)
}

func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// RecordRefusal implements Repository.
func (s *PGStore) RecordRefusal(ctx context.Context, actor audit.Actor, requestID, unitID, disposition, reason string) error {
	return audit.Record(ctx, s.pool, audit.Event{Actor: actor, Action: "pdi_inspection.recorded", Outcome: audit.OutcomeDenied,
		EntityType: "vehicle_unit", EntityID: unitID, RequestID: requestID,
		Summary: audit.Summary{"disposition": disposition, "reason": reason}})
}

// ── Checklist configuration ────────────────────────────────────────────────

// ListChecklist implements Repository. brandId lists the items that apply to
// that brand (every-brand items and the brand's own).
func (s *PGStore) ListChecklist(ctx context.Context, f checklistFilter) (ChecklistPage, error) {
	where := ` WHERE ($1 = '' OR c.status = $1) AND ($2 = '' OR c.brand_id IS NULL OR c.brand_id::text = $2)`
	var total int
	if err := s.pool.QueryRow(ctx, `SELECT count(*)`+checklistFrom+where, f.Status, f.BrandID).Scan(&total); err != nil {
		return ChecklistPage{}, err
	}
	items, err := queryAll(ctx, s.pool, scanChecklist, `SELECT `+checklistColumns+checklistFrom+where+
		` ORDER BY c.status, c.sort_order, c.code, c.created_at, c.id LIMIT $3 OFFSET $4`,
		f.Status, f.BrandID, f.Limit, (f.Page-1)*f.Limit)
	if err != nil {
		return ChecklistPage{}, err
	}
	return ChecklistPage{Items: items, Page: f.Page, Limit: f.Limit, Total: total}, nil
}

func readChecklist(ctx context.Context, q querier, id string) (ChecklistItem, error) {
	c, err := scanChecklist(q.QueryRow(ctx, `SELECT `+checklistColumns+checklistFrom+` WHERE c.id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return ChecklistItem{}, ErrNotFound
	}
	return c, err
}

// CreateChecklistItem implements Repository.
func (s *PGStore) CreateChecklistItem(ctx context.Context, actor audit.Actor, requestID string, in normalisedItem) (ChecklistItem, error) {
	var out ChecklistItem
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if in.BrandID != nil {
			var ok bool
			if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM brands WHERE id = $1)`, *in.BrandID).Scan(&ok); err != nil {
				return err
			}
			if !ok {
				return &ValidationError{Fields: []FieldProblem{{"brandId", "is not a known brand"}}}
			}
		}
		var id string
		err := tx.QueryRow(ctx, `INSERT INTO pdi_checklist_items (code, label, description, brand_id, sort_order, created_by)
			VALUES ($1, $2, $3, $4, $5, $6) RETURNING id::text`,
			in.Code, in.Label, in.Description, in.BrandID, in.SortOrder, actor.UserID).Scan(&id)
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "pdi_checklist_items_active_code_key" {
			return ErrDuplicate
		}
		if err != nil {
			return fmt.Errorf("create checklist item: %w", err)
		}
		if out, err = readChecklist(ctx, tx, id); err != nil {
			return err
		}
		var brand any
		if in.BrandID != nil {
			brand = *in.BrandID
		}
		return audit.Record(ctx, tx, audit.Event{Actor: actor, Action: "pdi_checklist_item.created", EntityType: "pdi_checklist_item",
			EntityID: id, Reference: in.Code, RequestID: requestID, Summary: audit.Summary{"code": in.Code, "label": in.Label,
				"brandId": brand, "sortOrder": in.SortOrder}})
	})
	return out, err
}

// RetireChecklistItem implements Repository.
func (s *PGStore) RetireChecklistItem(ctx context.Context, actor audit.Actor, requestID, id string, version int,
	reason string) (ChecklistItem, error) {
	var out ChecklistItem
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		var status, code string
		var current int
		err := tx.QueryRow(ctx, `SELECT status, code, version FROM pdi_checklist_items WHERE id = $1 FOR UPDATE`, id).
			Scan(&status, &code, &current)
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return ErrNotFound
		case err != nil:
			return err
		case current != version:
			return ErrVersionConflict
		case status != ItemActive:
			return ErrItemRetired
		}
		if _, err := tx.Exec(ctx, `UPDATE pdi_checklist_items SET status = 'RETIRED', retired_by = $2, retired_at = now(),
			retire_reason = $3, version = version + 1 WHERE id = $1`, id, actor.UserID, reason); err != nil {
			return fmt.Errorf("retire checklist item: %w", err)
		}
		if out, err = readChecklist(ctx, tx, id); err != nil {
			return err
		}
		return audit.Record(ctx, tx, audit.Event{Actor: actor, Action: "pdi_checklist_item.retired", EntityType: "pdi_checklist_item",
			EntityID: id, Reference: code, RequestID: requestID, Summary: audit.Summary{"code": code, "reason": reason,
				"status": audit.Change{From: ItemActive, To: ItemRetired}}})
	})
	return out, err
}
