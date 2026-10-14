package catalogue

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/audit"
)

// PGStore is the PostgreSQL Repository. Every mutation runs in one
// transaction: it locks the item (FOR UPDATE) and the parent it depends on
// (FOR SHARE), checks the version and the lifecycle rules, writes, and records
// the audit event, so the event exists if and only if the change does.
type PGStore struct {
	pool *pgxpool.Pool
}

// NewPGStore returns a Repository backed by pool.
func NewPGStore(pool *pgxpool.Pool) *PGStore { return &PGStore{pool: pool} }

// table describes where an item kind lives.
type table struct {
	name, parentTable, parentCol string
}

var tables = map[string]table{
	kindModel.entityType:   {name: "models", parentTable: "brands", parentCol: "brand_id"},
	kindVariant.entityType: {name: "variants", parentTable: "models", parentCol: "model_id"},
	kindColour.entityType:  {name: "variant_colours", parentTable: "variants", parentCol: "variant_id"},
}

const (
	brandColumns = `id::text, name, slug, lifecycle_status, authorisation_status, manufacturer_legal_name,
		authorisation_reference, authorisation_evidence_id::text, verified_by::text, verified_at, version, created_at, updated_at`
	provenanceColumns = `confidence, source_url, source_document_id::text, to_char(source_date, 'YYYY-MM-DD'),
		verified_by::text, verified_at`
	modelColumns = `id::text, brand_id::text, name, internal_code, manufacturer_code, vehicle_type, lifecycle_status, ` +
		provenanceColumns + `, version, created_at, updated_at`
	variantColumns = `id::text, brand_id::text, model_id::text, name, variant_code, manufacturer_code, lifecycle_status, ` +
		provenanceColumns + `, version, created_at, updated_at`
	colourColumns = `id::text, brand_id::text, variant_id::text, name, colour_code, lifecycle_status, ` +
		provenanceColumns + `, version, created_at, updated_at`
)

func scanBrand(row pgx.Row) (Brand, error) {
	var b Brand
	err := row.Scan(&b.ID, &b.Name, &b.Slug, &b.LifecycleStatus, &b.AuthorisationStatus, &b.ManufacturerLegalName,
		&b.AuthorisationReference, &b.AuthorisationEvidenceID, &b.VerifiedBy, &b.VerifiedAt, &b.Version, &b.CreatedAt, &b.UpdatedAt)
	b.Operational = b.LifecycleStatus == StatusActive && b.AuthorisationStatus == AuthAuthorised
	return b, err
}

func provenanceDest(p *Provenance) []any {
	return []any{&p.Confidence, &p.SourceURL, &p.SourceDocumentID, &p.SourceDate, &p.VerifiedBy, &p.VerifiedAt}
}

func scanModel(row pgx.Row) (Model, error) {
	var m Model
	dest := append([]any{&m.ID, &m.BrandID, &m.Name, &m.InternalCode, &m.ManufacturerCode, &m.VehicleType, &m.LifecycleStatus},
		provenanceDest(&m.Provenance)...)
	err := row.Scan(append(dest, &m.Version, &m.CreatedAt, &m.UpdatedAt)...)
	return m, err
}

func scanVariant(row pgx.Row) (Variant, error) {
	var v Variant
	dest := append([]any{&v.ID, &v.BrandID, &v.ModelID, &v.Name, &v.VariantCode, &v.ManufacturerCode, &v.LifecycleStatus},
		provenanceDest(&v.Provenance)...)
	err := row.Scan(append(dest, &v.Version, &v.CreatedAt, &v.UpdatedAt)...)
	return v, err
}

func scanColour(row pgx.Row) (Colour, error) {
	var c Colour
	dest := append([]any{&c.ID, &c.BrandID, &c.VariantID, &c.Name, &c.ColourCode, &c.LifecycleStatus},
		provenanceDest(&c.Provenance)...)
	err := row.Scan(append(dest, &c.Version, &c.CreatedAt, &c.UpdatedAt)...)
	return c, err
}

// duplicateFields maps unique indexes to the request field they protect.
var duplicateFields = map[string]string{
	"brands_slug_key":                      "slug",
	"brands_name_lower_key":                "name",
	"models_brand_internal_code_key":       "internalCode",
	"models_brand_manufacturer_code_key":   "manufacturerCode",
	"models_brand_name_key":                "name",
	"variants_model_code_key":              "variantCode",
	"variants_brand_manufacturer_code_key": "manufacturerCode",
	"variants_model_name_key":              "name",
	"variant_colours_variant_name_key":     "name",
	"variant_colours_variant_code_key":     "colourCode",
}

// mapPGError turns constraint violations into domain errors.
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
		switch {
		case strings.Contains(pgErr.ConstraintName, "source_document_id"):
			return &ValidationError{Fields: []FieldProblem{{"sourceDocumentId", "is not a known document"}}}
		case strings.Contains(pgErr.ConstraintName, "authorisation_evidence_id"):
			return &ValidationError{Fields: []FieldProblem{{"authorisationEvidenceId", "is not a known document"}}}
		}
	case "23514":
		return &ValidationError{Fields: []FieldProblem{{"body", "violates catalogue rule " + pgErr.ConstraintName}}}
	}
	return err
}

func str(p *string) any {
	if p == nil {
		return nil
	}
	return *p
}

func change(from, to *string) audit.Change { return audit.Change{From: str(from), To: str(to)} }

func withSummary(ev audit.Event, s audit.Summary) audit.Event {
	if ev.Summary == nil {
		ev.Summary = audit.Summary{}
	}
	for k, v := range s {
		ev.Summary[k] = v
	}
	return ev
}

// tx runs fn in a transaction and maps constraint errors.
func (s *PGStore) tx(ctx context.Context, fn func(pgx.Tx) error) error {
	return mapPGError(pgx.BeginFunc(ctx, s.pool, fn))
}

// ── List helpers ─────────────────────────────────────────────────────────────

type where struct {
	conds []string
	args  []any
}

func (w *where) add(cond string, arg any) {
	w.args = append(w.args, arg)
	w.conds = append(w.conds, strings.ReplaceAll(cond, "?", "$"+strconv.Itoa(len(w.args))))
}

func (w *where) sql() string {
	if len(w.conds) == 0 {
		return ""
	}
	return " WHERE " + strings.Join(w.conds, " AND ")
}

func likePattern(q string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return "%" + r.Replace(q) + "%"
}

func list[T any](ctx context.Context, s *PGStore, from, cols, order string, w where, f ListFilter, scan func(pgx.Row) (T, error)) (Page[T], error) {
	var total int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM `+from+w.sql(), w.args...).Scan(&total); err != nil {
		return Page[T]{}, fmt.Errorf("count %s: %w", from, err)
	}
	args := append(append([]any{}, w.args...), f.Limit, f.Offset())
	n := len(w.args)
	rows, err := s.pool.Query(ctx, `SELECT `+cols+` FROM `+from+w.sql()+` ORDER BY `+order+
		` LIMIT $`+strconv.Itoa(n+1)+` OFFSET $`+strconv.Itoa(n+2), args...)
	if err != nil {
		return Page[T]{}, fmt.Errorf("list %s: %w", from, err)
	}
	defer rows.Close()
	items := []T{}
	for rows.Next() {
		item, err := scan(rows)
		if err != nil {
			return Page[T]{}, fmt.Errorf("scan %s: %w", from, err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return Page[T]{}, err
	}
	return Page[T]{Items: items, Page: f.Page, Limit: f.Limit, Total: total}, nil
}

func itemFilters(w *where, f ListFilter, codeCols ...string) {
	if f.Status != "" {
		w.add("lifecycle_status = ?", f.Status)
	}
	if f.Confidence != "" {
		w.add("confidence = ?", f.Confidence)
	}
	if f.BrandID != "" {
		w.add("brand_id = ?", f.BrandID)
	}
	if f.Query != "" {
		conds := []string{`name ILIKE ? ESCAPE '\'`}
		for _, c := range codeCols {
			conds = append(conds, c+` ILIKE ? ESCAPE '\'`)
		}
		w.add("("+strings.Join(conds, " OR ")+")", likePattern(f.Query))
	}
}

// ── Brands ───────────────────────────────────────────────────────────────────

// ListBrands implements Repository.
func (s *PGStore) ListBrands(ctx context.Context, f ListFilter) (Page[Brand], error) {
	var w where
	if f.Status != "" {
		w.add("lifecycle_status = ?", f.Status)
	}
	if f.AuthStatus != "" {
		w.add("authorisation_status = ?", f.AuthStatus)
	}
	if f.Query != "" {
		w.add(`(name ILIKE ? ESCAPE '\' OR slug ILIKE ? ESCAPE '\')`, likePattern(f.Query))
	}
	return list(ctx, s, "brands", brandColumns, "lower(name), id", w, f, scanBrand)
}

// GetBrand implements Repository.
func (s *PGStore) GetBrand(ctx context.Context, id string) (Brand, error) {
	b, err := scanBrand(s.pool.QueryRow(ctx, `SELECT `+brandColumns+` FROM brands WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Brand{}, ErrNotFound
	}
	if err != nil {
		return Brand{}, fmt.Errorf("get brand: %w", err)
	}
	return b, nil
}

func lockBrand(ctx context.Context, tx pgx.Tx, id string, version int) (Brand, error) {
	b, err := scanBrand(tx.QueryRow(ctx, `SELECT `+brandColumns+` FROM brands WHERE id = $1 FOR UPDATE`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Brand{}, ErrNotFound
	}
	if err != nil {
		return Brand{}, fmt.Errorf("lock brand: %w", err)
	}
	if b.Version != version {
		return Brand{}, ErrVersionConflict
	}
	return b, nil
}

// CreateBrand implements Repository.
func (s *PGStore) CreateBrand(ctx context.Context, in BrandCreateRequest, ev audit.Event) (Brand, error) {
	var b Brand
	err := s.tx(ctx, func(tx pgx.Tx) error {
		var err error
		b, err = scanBrand(tx.QueryRow(ctx, `INSERT INTO brands (name, slug) VALUES ($1, $2) RETURNING `+brandColumns, in.Name, *in.Slug))
		if err != nil {
			return err
		}
		ev.EntityID = b.ID
		return audit.Record(ctx, tx, withSummary(ev, audit.Summary{"name": b.Name, "slug": b.Slug}))
	})
	return b, err
}

// UpdateBrand implements Repository.
func (s *PGStore) UpdateBrand(ctx context.Context, id string, in BrandUpdateRequest, ev audit.Event) (Brand, error) {
	var b Brand
	err := s.tx(ctx, func(tx pgx.Tx) error {
		old, err := lockBrand(ctx, tx, id, in.Version)
		if err != nil {
			return err
		}
		if err := checkBrandEditable(old.LifecycleStatus); err != nil {
			return err
		}
		name, slug := old.Name, old.Slug
		sum := audit.Summary{}
		if in.Name.Value != nil {
			name = *in.Name.Value
			sum["name"] = audit.Change{From: old.Name, To: name}
		}
		if in.Slug.Value != nil {
			slug = *in.Slug.Value
			sum["slug"] = audit.Change{From: old.Slug, To: slug}
		}
		b, err = scanBrand(tx.QueryRow(ctx, `UPDATE brands SET name = $2, slug = $3, version = version + 1
			WHERE id = $1 RETURNING `+brandColumns, id, name, slug))
		if err != nil {
			return err
		}
		return audit.Record(ctx, tx, withSummary(ev, sum))
	})
	return b, err
}

// RecordBrandAuthorisation implements Repository. The verifier is the actor;
// UNVERIFIED clears the verifier and date.
func (s *PGStore) RecordBrandAuthorisation(ctx context.Context, id string, in BrandAuthorisationRequest, verifierID string, ev audit.Event) (Brand, error) {
	var b Brand
	err := s.tx(ctx, func(tx pgx.Tx) error {
		old, err := lockBrand(ctx, tx, id, in.Version)
		if err != nil {
			return err
		}
		if err := checkBrandEditable(old.LifecycleStatus); err != nil {
			return err
		}
		var verifier any = verifierID
		if in.AuthorisationStatus == AuthUnverified {
			verifier = nil
		}
		b, err = scanBrand(tx.QueryRow(ctx, `UPDATE brands SET authorisation_status = $2, manufacturer_legal_name = $3,
				authorisation_reference = $4, authorisation_evidence_id = $5::uuid, verified_by = $6::uuid,
				verified_at = CASE WHEN $6::uuid IS NULL THEN NULL ELSE now() END, version = version + 1
			WHERE id = $1 RETURNING `+brandColumns,
			id, in.AuthorisationStatus, str(in.ManufacturerLegalName), str(in.AuthorisationReference), str(in.AuthorisationEvidenceID), verifier))
		if err != nil {
			return err
		}
		return audit.Record(ctx, tx, withSummary(ev, audit.Summary{
			"authorisationStatus":     audit.Change{From: old.AuthorisationStatus, To: b.AuthorisationStatus},
			"manufacturerLegalName":   change(old.ManufacturerLegalName, b.ManufacturerLegalName),
			"authorisationReference":  change(old.AuthorisationReference, b.AuthorisationReference),
			"authorisationEvidenceId": change(old.AuthorisationEvidenceID, b.AuthorisationEvidenceID),
			"lifecycleStatus":         b.LifecycleStatus,
		}))
	})
	return b, err
}

func hasActiveVariant(ctx context.Context, q pgx.Tx, brandID string, lock bool) (bool, error) {
	sql := `SELECT v.id FROM variants v JOIN models m ON m.id = v.model_id AND m.brand_id = v.brand_id
		WHERE v.brand_id = $1 AND v.lifecycle_status = 'ACTIVE' AND m.lifecycle_status = 'ACTIVE' LIMIT 1`
	if lock {
		sql += ` FOR SHARE OF v, m`
	}
	var id string
	err := q.QueryRow(ctx, sql, brandID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

// hasApprovedPrice: an approved SELLING price valid now for an ACTIVE variant
// under an ACTIVE model of the brand (price_items, C-04).
func hasApprovedPrice(ctx context.Context, q pgx.Tx, brandID string, lock bool) (bool, error) {
	sql := `SELECT p.id FROM price_items p
		JOIN variants v ON v.id = p.variant_id JOIN models m ON m.id = p.model_id
		WHERE p.brand_id = $1 AND p.status = 'APPROVED' AND p.price_type = 'SELLING'
		  AND p.effective_from <= now() AND (p.effective_to IS NULL OR p.effective_to > now())
		  AND v.lifecycle_status = 'ACTIVE' AND m.lifecycle_status = 'ACTIVE' LIMIT 1`
	if lock {
		sql += ` FOR SHARE OF p, v, m`
	}
	var id string
	err := q.QueryRow(ctx, sql, brandID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

// hasSupplierLink: a VERIFIED supplier relationship valid today with a VERIFIED supplier (D-01).
func hasSupplierLink(ctx context.Context, q pgx.Tx, brandID string, lock bool) (bool, error) {
	sql := `SELECT l.id FROM supplier_brand_links l JOIN suppliers s ON s.id = l.supplier_id
		WHERE l.brand_id = $1 AND l.status = 'VERIFIED' AND s.status = 'VERIFIED'
		  AND l.effective_from <= current_date AND (l.effective_to IS NULL OR l.effective_to >= current_date) LIMIT 1`
	if lock {
		sql += ` FOR SHARE OF l, s`
	}
	var id string
	err := q.QueryRow(ctx, sql, brandID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

// BrandReadiness implements Repository.
func (s *PGStore) BrandReadiness(ctx context.Context, id string) (Readiness, error) {
	var r Readiness
	err := pgx.BeginTxFunc(ctx, s.pool, pgx.TxOptions{AccessMode: pgx.ReadOnly}, func(tx pgx.Tx) error {
		b, err := scanBrand(tx.QueryRow(ctx, `SELECT `+brandColumns+` FROM brands WHERE id = $1`, id))
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		active, err := hasActiveVariant(ctx, tx, id, false)
		if err != nil {
			return err
		}
		priced, err := hasApprovedPrice(ctx, tx, id, false)
		if err != nil {
			return err
		}
		supplied, err := hasSupplierLink(ctx, tx, id, false)
		if err != nil {
			return err
		}
		r = brandReadiness(brandFacts{Brand: b, HasActiveVariant: active, HasApprovedPrice: priced, HasSupplierLink: supplied})
		return nil
	})
	return r, err
}

// ActivateBrand implements Repository.
func (s *PGStore) ActivateBrand(ctx context.Context, id string, version int, ev audit.Event) (Brand, error) {
	var b Brand
	err := s.tx(ctx, func(tx pgx.Tx) error {
		old, err := lockBrand(ctx, tx, id, version)
		if err != nil {
			return err
		}
		active, err := hasActiveVariant(ctx, tx, id, true)
		if err != nil {
			return err
		}
		priced, err := hasApprovedPrice(ctx, tx, id, true)
		if err != nil {
			return err
		}
		supplied, err := hasSupplierLink(ctx, tx, id, true)
		if err != nil {
			return err
		}
		if err := checkBrandActivation(brandFacts{Brand: old, HasActiveVariant: active, HasApprovedPrice: priced, HasSupplierLink: supplied}); err != nil {
			return err
		}
		b, err = scanBrand(tx.QueryRow(ctx, `UPDATE brands SET lifecycle_status = 'ACTIVE', version = version + 1
			WHERE id = $1 RETURNING `+brandColumns, id))
		if err != nil {
			return err
		}
		return audit.Record(ctx, tx, withSummary(ev, audit.Summary{
			"lifecycleStatus": audit.Change{From: old.LifecycleStatus, To: StatusActive}}))
	})
	return b, err
}

// ArchiveBrand implements Repository. Models and variants keep their own
// status; an archived brand is simply no longer operational.
func (s *PGStore) ArchiveBrand(ctx context.Context, id string, version int, ev audit.Event) (Brand, error) {
	var b Brand
	err := s.tx(ctx, func(tx pgx.Tx) error {
		old, err := lockBrand(ctx, tx, id, version)
		if err != nil {
			return err
		}
		if err := checkArchivable("brand", old.LifecycleStatus); err != nil {
			return err
		}
		b, err = scanBrand(tx.QueryRow(ctx, `UPDATE brands SET lifecycle_status = 'ARCHIVED', version = version + 1
			WHERE id = $1 RETURNING `+brandColumns, id))
		if err != nil {
			return err
		}
		return audit.Record(ctx, tx, withSummary(ev, audit.Summary{
			"lifecycleStatus": audit.Change{From: old.LifecycleStatus, To: StatusArchived}}))
	})
	return b, err
}

// ── Models ───────────────────────────────────────────────────────────────────

// ListModels implements Repository.
func (s *PGStore) ListModels(ctx context.Context, f ListFilter) (Page[Model], error) {
	var w where
	itemFilters(&w, f, "internal_code", "manufacturer_code")
	return list(ctx, s, "models", modelColumns, "lower(name), id", w, f, scanModel)
}

// GetModel implements Repository.
func (s *PGStore) GetModel(ctx context.Context, id string) (Model, error) {
	m, err := scanModel(s.pool.QueryRow(ctx, `SELECT `+modelColumns+` FROM models WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Model{}, ErrNotFound
	}
	if err != nil {
		return Model{}, fmt.Errorf("get model: %w", err)
	}
	return m, nil
}

// parentState locks a parent row against concurrent archive and returns its
// status and brand. A missing parent is a validation error on field.
func parentState(ctx context.Context, tx pgx.Tx, parentTable, id, field string) (status, brandID string, err error) {
	brandCol := "brand_id::text"
	if parentTable == "brands" {
		brandCol = "id::text"
	}
	err = tx.QueryRow(ctx, `SELECT lifecycle_status, `+brandCol+` FROM `+parentTable+` WHERE id = $1 FOR SHARE`, id).Scan(&status, &brandID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", &ValidationError{Fields: []FieldProblem{{field, "does not exist"}}}
	}
	return status, brandID, err
}

// CreateModel implements Repository.
func (s *PGStore) CreateModel(ctx context.Context, in ModelCreateRequest, ev audit.Event) (Model, error) {
	var m Model
	err := s.tx(ctx, func(tx pgx.Tx) error {
		status, _, err := parentState(ctx, tx, "brands", in.BrandID, "brandId")
		if err != nil {
			return err
		}
		if err := checkCanAddChild(kindModel, status); err != nil {
			return err
		}
		m, err = scanModel(tx.QueryRow(ctx, `INSERT INTO models (brand_id, name, internal_code, manufacturer_code, vehicle_type)
			VALUES ($1, $2, $3, $4, $5) RETURNING `+modelColumns,
			in.BrandID, in.Name, in.InternalCode, str(in.ManufacturerCode), str(in.VehicleType)))
		if err != nil {
			return err
		}
		ev.EntityID = m.ID
		return audit.Record(ctx, tx, withSummary(ev, audit.Summary{
			"brandId": m.BrandID, "name": m.Name, "internalCode": m.InternalCode,
			"manufacturerCode": str(m.ManufacturerCode), "vehicleType": str(m.VehicleType)}))
	})
	return m, err
}

// itemRow is the lifecycle state of a model, variant or colour.
type itemRow struct {
	status, confidence, parentID string
	version                      int
}

func lockItem(ctx context.Context, tx pgx.Tx, t table, id string, version int) (itemRow, error) {
	var r itemRow
	err := tx.QueryRow(ctx, `SELECT lifecycle_status, confidence, version, `+t.parentCol+`::text FROM `+t.name+
		` WHERE id = $1 FOR UPDATE`, id).Scan(&r.status, &r.confidence, &r.version, &r.parentID)
	if errors.Is(err, pgx.ErrNoRows) {
		return r, ErrNotFound
	}
	if err != nil {
		return r, fmt.Errorf("lock %s: %w", t.name, err)
	}
	if r.version != version {
		return r, ErrVersionConflict
	}
	return r, nil
}

// assignment is one column change of a PATCH.
type assignment struct {
	field, column string
	value         OptString
}

// updateItem applies a PATCH to a pending item. Editing a verified item resets
// it to UNVERIFIED: the verification covered the old values.
func updateItem[T any](ctx context.Context, s *PGStore, k kind, id string, version int, cols string, set []assignment,
	ev audit.Event, scan func(pgx.Row) (T, error)) (T, error) {
	var out T
	t := tables[k.entityType]
	err := s.tx(ctx, func(tx pgx.Tx) error {
		row, err := lockItem(ctx, tx, t, id, version)
		if err != nil {
			return err
		}
		if err := checkItemEditable(k, row.status); err != nil {
			return err
		}
		var olds []any
		args := []any{id}
		var sets []string
		var selects []string
		for _, a := range set {
			if !a.value.Set {
				continue
			}
			args = append(args, str(a.value.Value))
			sets = append(sets, a.column+" = $"+strconv.Itoa(len(args)))
			selects = append(selects, a.column)
		}
		if len(selects) > 0 {
			olds = make([]any, len(selects))
			dest := make([]any, len(selects))
			for i := range olds {
				dest[i] = &olds[i]
			}
			if err := tx.QueryRow(ctx, `SELECT `+strings.Join(selects, ", ")+` FROM `+t.name+` WHERE id = $1`, id).Scan(dest...); err != nil {
				return err
			}
		}
		sum := audit.Summary{}
		i := 0
		for _, a := range set {
			if a.value.Set {
				sum[a.field] = audit.Change{From: olds[i], To: str(a.value.Value)}
				i++
			}
		}
		if row.confidence != ConfidenceUnverified {
			sets = append(sets, `confidence = 'UNVERIFIED'`, `source_url = NULL`, `source_document_id = NULL`,
				`source_date = NULL`, `verified_by = NULL`, `verified_at = NULL`)
			sum["confidence"] = audit.Change{From: row.confidence, To: ConfidenceUnverified}
		}
		sets = append(sets, "version = version + 1")
		out, err = scan(tx.QueryRow(ctx, `UPDATE `+t.name+` SET `+strings.Join(sets, ", ")+` WHERE id = $1 RETURNING `+cols, args...))
		if err != nil {
			return err
		}
		return audit.Record(ctx, tx, withSummary(ev, sum))
	})
	return out, err
}

// UpdateModel implements Repository.
func (s *PGStore) UpdateModel(ctx context.Context, id string, in ModelUpdateRequest, ev audit.Event) (Model, error) {
	return updateItem(ctx, s, kindModel, id, in.Version, modelColumns, []assignment{
		{"name", "name", in.Name}, {"internalCode", "internal_code", in.InternalCode},
		{"manufacturerCode", "manufacturer_code", in.ManufacturerCode}, {"vehicleType", "vehicle_type", in.VehicleType},
	}, ev, scanModel)
}

// ── Variants ─────────────────────────────────────────────────────────────────

// filterBrandMatches refuses a list filter that pairs a brand with a parent of
// another brand (C-06): such a request is a cross-brand mismatch, not "no rows".
// An unknown parent is reported as a validation error too.
func (s *PGStore) filterBrandMatches(ctx context.Context, table, id, brandID, field, label string) error {
	if id == "" || brandID == "" {
		return nil
	}
	var actual string
	err := s.pool.QueryRow(ctx, `SELECT brand_id::text FROM `+table+` WHERE id = $1`, id).Scan(&actual)
	if errors.Is(err, pgx.ErrNoRows) {
		return &ValidationError{Fields: []FieldProblem{{field, "does not exist"}}}
	}
	if err != nil {
		return fmt.Errorf("check %s filter: %w", field, err)
	}
	if actual != brandID {
		return &ValidationError{Fields: []FieldProblem{{"brandId", "does not match the " + label + "'s brand"}}}
	}
	return nil
}

// ListVariants implements Repository.
func (s *PGStore) ListVariants(ctx context.Context, f ListFilter) (Page[Variant], error) {
	if err := s.filterBrandMatches(ctx, "models", f.ModelID, f.BrandID, "modelId", "model"); err != nil {
		return Page[Variant]{}, err
	}
	var w where
	itemFilters(&w, f, "variant_code", "manufacturer_code")
	if f.ModelID != "" {
		w.add("model_id = ?", f.ModelID)
	}
	return list(ctx, s, "variants", variantColumns, "lower(name), id", w, f, scanVariant)
}

// GetVariant implements Repository.
func (s *PGStore) GetVariant(ctx context.Context, id string) (Variant, error) {
	v, err := scanVariant(s.pool.QueryRow(ctx, `SELECT `+variantColumns+` FROM variants WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Variant{}, ErrNotFound
	}
	if err != nil {
		return Variant{}, fmt.Errorf("get variant: %w", err)
	}
	return v, nil
}

// brandMismatch rejects a client brandId that differs from the parent's brand.
func brandMismatch(claimed *string, actual, parent string) error {
	if claimed != nil && *claimed != actual {
		return &ValidationError{Fields: []FieldProblem{{"brandId", "does not match the " + parent + "'s brand"}}}
	}
	return nil
}

// CreateVariant implements Repository. The brand comes from the model, never the client.
func (s *PGStore) CreateVariant(ctx context.Context, in VariantCreateRequest, ev audit.Event) (Variant, error) {
	var v Variant
	err := s.tx(ctx, func(tx pgx.Tx) error {
		status, brandID, err := parentState(ctx, tx, "models", in.ModelID, "modelId")
		if err != nil {
			return err
		}
		if err := brandMismatch(in.BrandID, brandID, "model"); err != nil {
			return err
		}
		if err := checkCanAddChild(kindVariant, status); err != nil {
			return err
		}
		v, err = scanVariant(tx.QueryRow(ctx, `INSERT INTO variants (brand_id, model_id, name, variant_code, manufacturer_code)
			VALUES ($1, $2, $3, $4, $5) RETURNING `+variantColumns,
			brandID, in.ModelID, in.Name, in.VariantCode, str(in.ManufacturerCode)))
		if err != nil {
			return err
		}
		ev.EntityID = v.ID
		return audit.Record(ctx, tx, withSummary(ev, audit.Summary{
			"brandId": v.BrandID, "modelId": v.ModelID, "name": v.Name, "variantCode": v.VariantCode,
			"manufacturerCode": str(v.ManufacturerCode)}))
	})
	return v, err
}

// UpdateVariant implements Repository.
func (s *PGStore) UpdateVariant(ctx context.Context, id string, in VariantUpdateRequest, ev audit.Event) (Variant, error) {
	return updateItem(ctx, s, kindVariant, id, in.Version, variantColumns, []assignment{
		{"name", "name", in.Name}, {"variantCode", "variant_code", in.VariantCode},
		{"manufacturerCode", "manufacturer_code", in.ManufacturerCode},
	}, ev, scanVariant)
}

// ── Colours ──────────────────────────────────────────────────────────────────

// ListColours implements Repository.
func (s *PGStore) ListColours(ctx context.Context, f ListFilter) (Page[Colour], error) {
	if err := s.filterBrandMatches(ctx, "variants", f.VariantID, f.BrandID, "variantId", "variant"); err != nil {
		return Page[Colour]{}, err
	}
	var w where
	itemFilters(&w, f, "colour_code")
	if f.VariantID != "" {
		w.add("variant_id = ?", f.VariantID)
	}
	return list(ctx, s, "variant_colours", colourColumns, "lower(name), id", w, f, scanColour)
}

// GetColour implements Repository.
func (s *PGStore) GetColour(ctx context.Context, id string) (Colour, error) {
	c, err := scanColour(s.pool.QueryRow(ctx, `SELECT `+colourColumns+` FROM variant_colours WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Colour{}, ErrNotFound
	}
	if err != nil {
		return Colour{}, fmt.Errorf("get colour: %w", err)
	}
	return c, nil
}

// CreateColour implements Repository. The brand comes from the variant.
func (s *PGStore) CreateColour(ctx context.Context, in ColourCreateRequest, ev audit.Event) (Colour, error) {
	var c Colour
	err := s.tx(ctx, func(tx pgx.Tx) error {
		status, brandID, err := parentState(ctx, tx, "variants", in.VariantID, "variantId")
		if err != nil {
			return err
		}
		if err := brandMismatch(in.BrandID, brandID, "variant"); err != nil {
			return err
		}
		if err := checkCanAddChild(kindColour, status); err != nil {
			return err
		}
		c, err = scanColour(tx.QueryRow(ctx, `INSERT INTO variant_colours (brand_id, variant_id, name, colour_code)
			VALUES ($1, $2, $3, $4) RETURNING `+colourColumns, brandID, in.VariantID, in.Name, str(in.ColourCode)))
		if err != nil {
			return err
		}
		ev.EntityID = c.ID
		return audit.Record(ctx, tx, withSummary(ev, audit.Summary{
			"brandId": c.BrandID, "variantId": c.VariantID, "name": c.Name, "colourCode": str(c.ColourCode)}))
	})
	return c, err
}

// UpdateColour implements Repository.
func (s *PGStore) UpdateColour(ctx context.Context, id string, in ColourUpdateRequest, ev audit.Event) (Colour, error) {
	return updateItem(ctx, s, kindColour, id, in.Version, colourColumns, []assignment{
		{"name", "name", in.Name}, {"colourCode", "colour_code", in.ColourCode},
	}, ev, scanColour)
}

// ── Lifecycle actions shared by models, variants and colours ─────────────────

// VerifyItem implements Repository. The verifier is the actor.
func (s *PGStore) VerifyItem(ctx context.Context, k kind, id string, in VerifyRequest, verifierID string, ev audit.Event) error {
	t := tables[k.entityType]
	return s.tx(ctx, func(tx pgx.Tx) error {
		row, err := lockItem(ctx, tx, t, id, in.Version)
		if err != nil {
			return err
		}
		if err := checkVerifiable(k, row.status); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE `+t.name+` SET confidence = $2, source_url = $3, source_document_id = $4::uuid,
				source_date = $5::date, verified_by = $6, verified_at = now(), version = version + 1 WHERE id = $1`,
			id, in.Confidence, str(in.SourceURL), str(in.SourceDocumentID), in.SourceDate, verifierID); err != nil {
			return err
		}
		return audit.Record(ctx, tx, withSummary(ev, audit.Summary{
			"confidence": audit.Change{From: row.confidence, To: in.Confidence},
			"sourceUrl":  str(in.SourceURL), "sourceDocumentId": str(in.SourceDocumentID), "sourceDate": in.SourceDate}))
	})
}

// ActivateItem implements Repository.
func (s *PGStore) ActivateItem(ctx context.Context, k kind, id string, version int, ev audit.Event) error {
	t := tables[k.entityType]
	return s.tx(ctx, func(tx pgx.Tx) error {
		row, err := lockItem(ctx, tx, t, id, version)
		if err != nil {
			return err
		}
		var parentStatus string
		if err := tx.QueryRow(ctx, `SELECT lifecycle_status FROM `+t.parentTable+` WHERE id = $1 FOR SHARE`, row.parentID).Scan(&parentStatus); err != nil {
			return fmt.Errorf("lock parent: %w", err)
		}
		if err := checkItemActivation(k, row.status, row.confidence, parentStatus); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE `+t.name+` SET lifecycle_status = 'ACTIVE', version = version + 1 WHERE id = $1`, id); err != nil {
			return err
		}
		return audit.Record(ctx, tx, withSummary(ev, audit.Summary{
			"lifecycleStatus": audit.Change{From: row.status, To: StatusActive}, "confidence": row.confidence}))
	})
}

// ArchiveItem implements Repository. Children are left as they are.
func (s *PGStore) ArchiveItem(ctx context.Context, k kind, id string, version int, ev audit.Event) error {
	t := tables[k.entityType]
	return s.tx(ctx, func(tx pgx.Tx) error {
		row, err := lockItem(ctx, tx, t, id, version)
		if err != nil {
			return err
		}
		if err := checkArchivable(k.label, row.status); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE `+t.name+` SET lifecycle_status = 'ARCHIVED', version = version + 1 WHERE id = $1`, id); err != nil {
			return err
		}
		return audit.Record(ctx, tx, withSummary(ev, audit.Summary{
			"lifecycleStatus": audit.Change{From: row.status, To: StatusArchived}}))
	})
}

// RecordAudit implements Repository (used for denied activation attempts).
func (s *PGStore) RecordAudit(ctx context.Context, ev audit.Event) error {
	return audit.Record(ctx, s.pool, ev)
}
