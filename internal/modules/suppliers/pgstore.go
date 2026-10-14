package suppliers

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
// locks what it changes, checks the version and state, writes and audits.
type PGStore struct{ pool *pgxpool.Pool }

// NewPGStore returns a Repository backed by pool.
func NewPGStore(pool *pgxpool.Pool) *PGStore { return &PGStore{pool: pool} }

const supplierColumns = `s.id::text, s.legal_name, s.trade_name, s.gstin, s.address_line1, s.address_line2, s.city, s.state,
	s.postal_code, s.status, s.verification_reference, s.verification_document_id::text, s.verified_by::text, s.verified_at,
	s.notes, s.created_by::text,
	(SELECT count(*) FROM supplier_brand_links l WHERE l.supplier_id = s.id AND l.status <> 'ENDED')::int,
	s.version, s.created_at, s.updated_at`

func scanSupplier(row pgx.Row) (Supplier, error) {
	var s Supplier
	err := row.Scan(&s.ID, &s.LegalName, &s.TradeName, &s.GSTIN, &s.AddressLine1, &s.AddressLine2, &s.City, &s.State,
		&s.PostalCode, &s.Status, &s.VerificationReference, &s.VerificationDocumentID, &s.VerifiedBy, &s.VerifiedAt,
		&s.Notes, &s.CreatedBy, &s.BrandCount, &s.Version, &s.CreatedAt, &s.UpdatedAt)
	return s, err
}

const contactColumns = `id::text, supplier_id::text, name, role, phone, email, is_primary, active, version, created_at, updated_at`

func scanContact(row pgx.Row) (Contact, error) {
	var c Contact
	err := row.Scan(&c.ID, &c.SupplierID, &c.Name, &c.Role, &c.Phone, &c.Email, &c.IsPrimary, &c.Active, &c.Version, &c.CreatedAt, &c.UpdatedAt)
	return c, err
}

const documentColumns = `d.id::text, d.supplier_id::text, d.attachment_id::text, d.document_type, d.note, a.original_name,
	a.content_type, d.added_by::text, d.created_at`

func scanDocument(row pgx.Row) (Document, error) {
	var d Document
	err := row.Scan(&d.ID, &d.SupplierID, &d.AttachmentID, &d.DocumentType, &d.Note, &d.FileName, &d.ContentType, &d.AddedBy, &d.CreatedAt)
	return d, err
}

const linkColumns = `l.id::text, l.supplier_id::text, s.legal_name, s.status, l.brand_id::text, b.name, l.status,
	l.authorisation_reference, l.evidence_document_id::text, to_char(l.effective_from, 'YYYY-MM-DD'),
	to_char(l.effective_to, 'YYYY-MM-DD'), l.verified_by::text, l.verified_at, l.note, l.ended_reason,
	(l.status = 'VERIFIED' AND s.status = 'VERIFIED' AND l.effective_from <= current_date
	 AND (l.effective_to IS NULL OR l.effective_to >= current_date)),
	l.version, l.created_at, l.updated_at`

const linkFrom = ` FROM supplier_brand_links l JOIN suppliers s ON s.id = l.supplier_id JOIN brands b ON b.id = l.brand_id`

func scanLink(row pgx.Row) (BrandLink, error) {
	var l BrandLink
	err := row.Scan(&l.ID, &l.SupplierID, &l.SupplierName, &l.SupplierStatus, &l.BrandID, &l.BrandName, &l.Status,
		&l.AuthorisationReference, &l.EvidenceDocumentID, &l.EffectiveFrom, &l.EffectiveTo, &l.VerifiedBy, &l.VerifiedAt,
		&l.Note, &l.EndedReason, &l.Current, &l.Version, &l.CreatedAt, &l.UpdatedAt)
	return l, err
}

var duplicateFields = map[string]string{
	"suppliers_legal_name_key":          "legalName",
	"suppliers_gstin_key":               "gstin",
	"supplier_brand_links_open_key":     "brandId",
	"supplier_documents_attachment_key": "attachmentId",
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
		case "suppliers_verification_document_fk":
			return invalid("documentId", "is not a document of this supplier")
		case "supplier_brand_links_evidence_fk":
			return invalid("evidenceDocumentId", "is not a document of this supplier")
		}
	case "23514":
		return invalid("body", "violates supplier rule "+pgErr.ConstraintName)
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

// ── Suppliers ────────────────────────────────────────────────────────────────

// ListSuppliers implements Repository.
func (s *PGStore) ListSuppliers(ctx context.Context, f ListFilter) (Page[Supplier], error) {
	var conds []string
	var args []any
	add := func(cond string, v any) {
		args = append(args, v)
		conds = append(conds, strings.ReplaceAll(cond, "?", "$"+strconv.Itoa(len(args))))
	}
	if f.Status != "" {
		add("s.status = ?", f.Status)
	}
	if f.Query != "" {
		r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
		add(`(s.legal_name ILIKE ? ESCAPE '\' OR s.trade_name ILIKE ? ESCAPE '\' OR s.gstin ILIKE ? ESCAPE '\')`, "%"+r.Replace(f.Query)+"%")
	}
	if f.BrandID != "" {
		add("EXISTS (SELECT 1 FROM supplier_brand_links l WHERE l.supplier_id = s.id AND l.brand_id = ? AND l.status <> 'ENDED')", f.BrandID)
	}
	where := ""
	if len(conds) > 0 {
		where = " WHERE " + strings.Join(conds, " AND ")
	}
	var total int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM suppliers s`+where, args...).Scan(&total); err != nil {
		return Page[Supplier]{}, fmt.Errorf("count suppliers: %w", err)
	}
	n := len(args)
	rows, err := s.pool.Query(ctx, `SELECT `+supplierColumns+` FROM suppliers s`+where+` ORDER BY lower(s.legal_name), s.id LIMIT $`+
		strconv.Itoa(n+1)+` OFFSET $`+strconv.Itoa(n+2), append(args, f.Limit, (f.Page-1)*f.Limit)...)
	if err != nil {
		return Page[Supplier]{}, fmt.Errorf("list suppliers: %w", err)
	}
	defer rows.Close()
	items := []Supplier{}
	for rows.Next() {
		sp, err := scanSupplier(rows)
		if err != nil {
			return Page[Supplier]{}, err
		}
		items = append(items, sp)
	}
	return Page[Supplier]{Items: items, Page: f.Page, Limit: f.Limit, Total: total}, rows.Err()
}

// queryAll runs a query and scans every row.
func queryAll[T any](ctx context.Context, pool *pgxpool.Pool, scan func(pgx.Row) (T, error), sql string, args ...any) ([]T, error) {
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

// GetDetail implements Repository.
func (s *PGStore) GetDetail(ctx context.Context, id string) (Detail, error) {
	var d Detail
	sp, err := scanSupplier(s.pool.QueryRow(ctx, `SELECT `+supplierColumns+` FROM suppliers s WHERE s.id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return d, ErrNotFound
	}
	if err != nil {
		return d, fmt.Errorf("get supplier: %w", err)
	}
	d.Supplier = sp
	if d.Contacts, err = queryAll(ctx, s.pool, scanContact, `SELECT `+contactColumns+` FROM supplier_contacts WHERE supplier_id = $1
		ORDER BY active DESC, is_primary DESC, lower(name), id`, id); err != nil {
		return d, fmt.Errorf("contacts: %w", err)
	}
	if d.Documents, err = queryAll(ctx, s.pool, scanDocument, `SELECT `+documentColumns+` FROM supplier_documents d
		JOIN attachments a ON a.id = d.attachment_id WHERE d.supplier_id = $1 ORDER BY d.created_at DESC, d.id`, id); err != nil {
		return d, fmt.Errorf("documents: %w", err)
	}
	if d.BrandLinks, err = queryAll(ctx, s.pool, scanLink, `SELECT `+linkColumns+linkFrom+` WHERE l.supplier_id = $1
		ORDER BY (l.status = 'ENDED'), lower(b.name), l.created_at DESC`, id); err != nil {
		return d, fmt.Errorf("links: %w", err)
	}
	return d, nil
}

func lockSupplier(ctx context.Context, tx pgx.Tx, id string, version int) (Supplier, error) {
	sp, err := scanSupplier(tx.QueryRow(ctx, `SELECT `+supplierColumns+` FROM suppliers s WHERE s.id = $1 FOR UPDATE OF s`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return sp, ErrNotFound
	}
	if err != nil {
		return sp, err
	}
	if version > 0 && sp.Version != version {
		return sp, ErrVersionConflict
	}
	return sp, nil
}

// CreateSupplier implements Repository.
func (s *PGStore) CreateSupplier(ctx context.Context, in SupplierInput, createdBy string, ev audit.Event) (Supplier, error) {
	var sp Supplier
	err := s.tx(ctx, func(tx pgx.Tx) error {
		var id string
		if err := tx.QueryRow(ctx, `INSERT INTO suppliers (legal_name, trade_name, gstin, address_line1, address_line2, city, state,
				postal_code, notes, created_by) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING id::text`,
			in.LegalName, str(in.TradeName), str(in.GSTIN), str(in.AddressLine1), str(in.AddressLine2), str(in.City), str(in.State),
			str(in.PostalCode), str(in.Notes), createdBy).Scan(&id); err != nil {
			return err
		}
		var err error
		if sp, err = scanSupplier(tx.QueryRow(ctx, `SELECT `+supplierColumns+` FROM suppliers s WHERE s.id = $1`, id)); err != nil {
			return err
		}
		ev.EntityID = id
		return audit.Record(ctx, tx, withSummary(ev, audit.Summary{"legalName": sp.LegalName, "gstin": str(sp.GSTIN)}))
	})
	return sp, err
}

// UpdateSupplier implements Repository. A new legal name or GSTIN on a verified
// supplier returns it to PENDING_VERIFICATION.
func (s *PGStore) UpdateSupplier(ctx context.Context, id string, p SupplierPatch, ev audit.Event) (Supplier, error) {
	var sp Supplier
	err := s.tx(ctx, func(tx pgx.Tx) error {
		old, err := lockSupplier(ctx, tx, id, p.Version)
		if err != nil {
			return err
		}
		if old.Status == StatusArchived {
			return &StateError{Message: "Archived suppliers cannot be changed."}
		}
		type col struct {
			field, column string
			o             OptString
			old           *string
		}
		legal := old.LegalName
		cols := []col{
			{"legalName", "legal_name", p.LegalName, &legal}, {"tradeName", "trade_name", p.TradeName, old.TradeName},
			{"gstin", "gstin", p.GSTIN, old.GSTIN}, {"addressLine1", "address_line1", p.AddressLine1, old.AddressLine1},
			{"addressLine2", "address_line2", p.AddressLine2, old.AddressLine2}, {"city", "city", p.City, old.City},
			{"state", "state", p.State, old.State}, {"postalCode", "postal_code", p.PostalCode, old.PostalCode},
			{"notes", "notes", p.Notes, old.Notes},
		}
		args := []any{id}
		var sets []string
		sum := audit.Summary{}
		identityChanged := false
		for _, c := range cols {
			if !c.o.Set {
				continue
			}
			args = append(args, str(c.o.Value))
			sets = append(sets, c.column+" = $"+strconv.Itoa(len(args)))
			if c.field == "notes" {
				sum[c.field] = "changed"
			} else {
				sum[c.field] = audit.Change{From: str(c.old), To: str(c.o.Value)}
			}
			if (c.field == "legalName" || c.field == "gstin") && fmt.Sprint(str(c.old)) != fmt.Sprint(str(c.o.Value)) {
				identityChanged = true
			}
		}
		if identityChanged && old.Status == StatusVerified {
			sets = append(sets, `status = 'PENDING_VERIFICATION'`, `verification_reference = NULL`, `verification_document_id = NULL`,
				`verified_by = NULL`, `verified_at = NULL`)
			sum["status"] = audit.Change{From: StatusVerified, To: StatusPending}
		}
		sets = append(sets, "version = version + 1")
		if _, err := tx.Exec(ctx, `UPDATE suppliers SET `+strings.Join(sets, ", ")+` WHERE id = $1`, args...); err != nil {
			return err
		}
		if sp, err = scanSupplier(tx.QueryRow(ctx, `SELECT `+supplierColumns+` FROM suppliers s WHERE s.id = $1`, id)); err != nil {
			return err
		}
		return audit.Record(ctx, tx, withSummary(ev, sum))
	})
	return sp, err
}

// VerifySupplier implements Repository.
func (s *PGStore) VerifySupplier(ctx context.Context, id string, in VerifyInput, verifier string, ev audit.Event) (Supplier, error) {
	var sp Supplier
	err := s.tx(ctx, func(tx pgx.Tx) error {
		old, err := lockSupplier(ctx, tx, id, in.Version)
		if err != nil {
			return err
		}
		if old.Status != StatusPending {
			return &StateError{Message: "Only suppliers pending verification can be verified."}
		}
		if _, err := tx.Exec(ctx, `UPDATE suppliers SET status = 'VERIFIED', verification_reference = $2, verification_document_id = $3,
			verified_by = $4, verified_at = now(), version = version + 1 WHERE id = $1`, id, in.Reference, in.DocumentID, verifier); err != nil {
			return err
		}
		if sp, err = scanSupplier(tx.QueryRow(ctx, `SELECT `+supplierColumns+` FROM suppliers s WHERE s.id = $1`, id)); err != nil {
			return err
		}
		return audit.Record(ctx, tx, withSummary(ev, audit.Summary{
			"status": audit.Change{From: StatusPending, To: StatusVerified}, "reference": in.Reference, "documentId": in.DocumentID}))
	})
	return sp, err
}

// ArchiveSupplier implements Repository. Contacts, documents and relationships stay as history.
func (s *PGStore) ArchiveSupplier(ctx context.Context, id string, in ArchiveInput, ev audit.Event) (Supplier, error) {
	var sp Supplier
	err := s.tx(ctx, func(tx pgx.Tx) error {
		old, err := lockSupplier(ctx, tx, id, in.Version)
		if err != nil {
			return err
		}
		if old.Status == StatusArchived {
			return &StateError{Message: "This supplier is already archived."}
		}
		if _, err := tx.Exec(ctx, `UPDATE suppliers SET status = 'ARCHIVED', version = version + 1 WHERE id = $1`, id); err != nil {
			return err
		}
		if sp, err = scanSupplier(tx.QueryRow(ctx, `SELECT `+supplierColumns+` FROM suppliers s WHERE s.id = $1`, id)); err != nil {
			return err
		}
		return audit.Record(ctx, tx, withSummary(ev, audit.Summary{
			"status": audit.Change{From: old.Status, To: StatusArchived}, "reason": str(in.Reason)}))
	})
	return sp, err
}

// ── Contacts ─────────────────────────────────────────────────────────────────

func requireOpenSupplier(ctx context.Context, tx pgx.Tx, supplierID string) error {
	var status string
	err := tx.QueryRow(ctx, `SELECT status FROM suppliers WHERE id = $1 FOR SHARE`, supplierID).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if status == StatusArchived {
		return &StateError{Message: "Archived suppliers cannot be changed."}
	}
	return nil
}

// AddContact implements Repository. A new primary contact replaces the old primary.
func (s *PGStore) AddContact(ctx context.Context, supplierID string, in ContactInput, ev audit.Event) (Contact, error) {
	var c Contact
	err := s.tx(ctx, func(tx pgx.Tx) error {
		if err := requireOpenSupplier(ctx, tx, supplierID); err != nil {
			return err
		}
		if in.IsPrimary {
			if _, err := tx.Exec(ctx, `UPDATE supplier_contacts SET is_primary = false, version = version + 1
				WHERE supplier_id = $1 AND is_primary AND active`, supplierID); err != nil {
				return err
			}
		}
		var err error
		c, err = scanContact(tx.QueryRow(ctx, `INSERT INTO supplier_contacts (supplier_id, name, role, phone, email, is_primary)
			VALUES ($1, $2, $3, $4, $5, $6) RETURNING `+contactColumns, supplierID, in.Name, str(in.Role), str(in.Phone), str(in.Email), in.IsPrimary))
		if err != nil {
			return err
		}
		ev.EntityID = supplierID
		// Contact details are personal data: the audit keeps the contact ID, not phone or email.
		return audit.Record(ctx, tx, withSummary(ev, audit.Summary{"contactId": c.ID, "isPrimary": c.IsPrimary}))
	})
	return c, err
}

func lockContact(ctx context.Context, tx pgx.Tx, supplierID, contactID string, version int) (Contact, error) {
	c, err := scanContact(tx.QueryRow(ctx, `SELECT `+contactColumns+` FROM supplier_contacts WHERE id = $1 AND supplier_id = $2 FOR UPDATE`,
		contactID, supplierID))
	if errors.Is(err, pgx.ErrNoRows) {
		return c, ErrNotFound
	}
	if err != nil {
		return c, err
	}
	if c.Version != version {
		return c, ErrVersionConflict
	}
	if !c.Active {
		return c, &StateError{Message: "Removed contacts cannot be changed."}
	}
	return c, nil
}

// UpdateContact implements Repository.
func (s *PGStore) UpdateContact(ctx context.Context, supplierID, contactID string, p ContactPatch, ev audit.Event) (Contact, error) {
	var c Contact
	err := s.tx(ctx, func(tx pgx.Tx) error {
		if err := requireOpenSupplier(ctx, tx, supplierID); err != nil {
			return err
		}
		old, err := lockContact(ctx, tx, supplierID, contactID, p.Version)
		if err != nil {
			return err
		}
		name, role, phone, email, primary := old.Name, old.Role, old.Phone, old.Email, old.IsPrimary
		if p.Name.Value != nil {
			name = *p.Name.Value
		}
		if p.Role.Set {
			role = p.Role.Value
		}
		if p.Phone.Set {
			phone = p.Phone.Value
		}
		if p.Email.Set {
			email = p.Email.Value
		}
		if p.IsPrimary != nil {
			primary = *p.IsPrimary
		}
		if phone == nil && email == nil {
			return invalid("phone", "give a phone number or an email address")
		}
		if primary && !old.IsPrimary {
			if _, err := tx.Exec(ctx, `UPDATE supplier_contacts SET is_primary = false, version = version + 1
				WHERE supplier_id = $1 AND is_primary AND active`, supplierID); err != nil {
				return err
			}
		}
		if c, err = scanContact(tx.QueryRow(ctx, `UPDATE supplier_contacts SET name = $2, role = $3, phone = $4, email = $5, is_primary = $6,
			version = version + 1 WHERE id = $1 RETURNING `+contactColumns, contactID, name, str(role), str(phone), str(email), primary)); err != nil {
			return err
		}
		ev.EntityID = supplierID
		return audit.Record(ctx, tx, withSummary(ev, audit.Summary{"contactId": c.ID, "isPrimary": c.IsPrimary}))
	})
	return c, err
}

// RemoveContact implements Repository (deactivates; kept as history).
func (s *PGStore) RemoveContact(ctx context.Context, supplierID, contactID string, version int, ev audit.Event) (Contact, error) {
	var c Contact
	err := s.tx(ctx, func(tx pgx.Tx) error {
		if err := requireOpenSupplier(ctx, tx, supplierID); err != nil {
			return err
		}
		if _, err := lockContact(ctx, tx, supplierID, contactID, version); err != nil {
			return err
		}
		var err error
		if c, err = scanContact(tx.QueryRow(ctx, `UPDATE supplier_contacts SET active = false, is_primary = false, version = version + 1
			WHERE id = $1 RETURNING `+contactColumns, contactID)); err != nil {
			return err
		}
		ev.EntityID = supplierID
		return audit.Record(ctx, tx, withSummary(ev, audit.Summary{"contactId": c.ID}))
	})
	return c, err
}

// ── Documents ────────────────────────────────────────────────────────────────

// AddDocument implements Repository: links one of the actor's own unlinked
// private uploads to the supplier (the attachment's owner is set once).
func (s *PGStore) AddDocument(ctx context.Context, supplierID string, in DocumentInput, actorID string, ev audit.Event) (Document, error) {
	var d Document
	err := s.tx(ctx, func(tx pgx.Tx) error {
		if err := requireOpenSupplier(ctx, tx, supplierID); err != nil {
			return err
		}
		var uploader string
		var owner *string
		err := tx.QueryRow(ctx, `SELECT uploaded_by::text, owner_type FROM attachments WHERE id = $1 FOR UPDATE`, in.AttachmentID).
			Scan(&uploader, &owner)
		if errors.Is(err, pgx.ErrNoRows) || (err == nil && (uploader != actorID || owner != nil)) {
			return invalid("attachmentId", "must be one of your own uploads that is not attached to anything yet")
		}
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE attachments SET owner_type = 'supplier', owner_id = $2 WHERE id = $1`, in.AttachmentID, supplierID); err != nil {
			return err
		}
		var id string
		if err := tx.QueryRow(ctx, `INSERT INTO supplier_documents (supplier_id, attachment_id, document_type, note, added_by)
			VALUES ($1, $2, $3, $4, $5) RETURNING id::text`, supplierID, in.AttachmentID, in.DocumentType, str(in.Note), actorID).Scan(&id); err != nil {
			return err
		}
		if d, err = scanDocument(tx.QueryRow(ctx, `SELECT `+documentColumns+` FROM supplier_documents d JOIN attachments a ON a.id = d.attachment_id
			WHERE d.id = $1`, id)); err != nil {
			return err
		}
		ev.EntityID = supplierID
		return audit.Record(ctx, tx, withSummary(ev, audit.Summary{"attachmentId": d.AttachmentID, "documentType": d.DocumentType}))
	})
	return d, err
}

// ── Brand relationships ──────────────────────────────────────────────────────

// ListLinks implements Repository.
func (s *PGStore) ListLinks(ctx context.Context, q LinkQuery) ([]BrandLink, error) {
	conds := []string{"TRUE"}
	var args []any
	add := func(cond string, v any) {
		args = append(args, v)
		conds = append(conds, strings.ReplaceAll(cond, "?", "$"+strconv.Itoa(len(args))))
	}
	if q.BrandID != "" {
		add("l.brand_id = ?", q.BrandID)
	}
	if q.SupplierID != "" {
		add("l.supplier_id = ?", q.SupplierID)
	}
	if q.Status != "" {
		add("l.status = ?", q.Status)
	}
	return queryAll(ctx, s.pool, scanLink, `SELECT `+linkColumns+linkFrom+` WHERE `+strings.Join(conds, " AND ")+
		` ORDER BY (l.status = 'ENDED'), lower(s.legal_name), lower(b.name), l.created_at DESC LIMIT 200`, args...)
}

func getLink(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, id string) (BrandLink, error) {
	l, err := scanLink(q.QueryRow(ctx, `SELECT `+linkColumns+linkFrom+` WHERE l.id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return l, ErrNotFound
	}
	return l, err
}

// CreateLink implements Repository: a proposed (unverified) relationship.
func (s *PGStore) CreateLink(ctx context.Context, supplierID string, in LinkInput, actorID string, ev audit.Event) (BrandLink, error) {
	var l BrandLink
	err := s.tx(ctx, func(tx pgx.Tx) error {
		if err := requireOpenSupplier(ctx, tx, supplierID); err != nil {
			return err
		}
		var brandStatus string
		err := tx.QueryRow(ctx, `SELECT lifecycle_status FROM brands WHERE id = $1 FOR SHARE`, in.BrandID).Scan(&brandStatus)
		if errors.Is(err, pgx.ErrNoRows) {
			return invalid("brandId", "does not exist")
		}
		if err != nil {
			return err
		}
		if brandStatus == "ARCHIVED" {
			return &StateError{Message: "A relationship cannot be added for an archived brand."}
		}
		var id string
		if err := tx.QueryRow(ctx, `INSERT INTO supplier_brand_links (supplier_id, brand_id, note, created_by) VALUES ($1, $2, $3, $4)
			RETURNING id::text`, supplierID, in.BrandID, str(in.Note), actorID).Scan(&id); err != nil {
			return err
		}
		if l, err = getLink(ctx, tx, id); err != nil {
			return err
		}
		ev.EntityID = id
		return audit.Record(ctx, tx, withSummary(ev, audit.Summary{"supplierId": supplierID, "brandId": in.BrandID}))
	})
	return l, err
}

func lockLink(ctx context.Context, tx pgx.Tx, id string, version int) (BrandLink, error) {
	var lockedVersion int
	err := tx.QueryRow(ctx, `SELECT version FROM supplier_brand_links WHERE id = $1 FOR UPDATE`, id).Scan(&lockedVersion)
	if errors.Is(err, pgx.ErrNoRows) {
		return BrandLink{}, ErrNotFound
	}
	if err != nil {
		return BrandLink{}, err
	}
	if lockedVersion != version {
		return BrandLink{}, ErrVersionConflict
	}
	return getLink(ctx, tx, id)
}

// VerifyLink implements Repository. The supplier must be verified and the
// evidence must be one of its documents (composite FK).
func (s *PGStore) VerifyLink(ctx context.Context, id string, in LinkVerifyInput, verifier string, ev audit.Event) (BrandLink, error) {
	var l BrandLink
	err := s.tx(ctx, func(tx pgx.Tx) error {
		old, err := lockLink(ctx, tx, id, in.Version)
		if err != nil {
			return err
		}
		if old.Status != LinkUnverified {
			return &StateError{Message: "Only unverified relationships can be verified."}
		}
		var supplierStatus, brandStatus string
		if err := tx.QueryRow(ctx, `SELECT s.status, b.lifecycle_status FROM suppliers s, brands b WHERE s.id = $1 AND b.id = $2
			FOR SHARE OF s, b`, old.SupplierID, old.BrandID).Scan(&supplierStatus, &brandStatus); err != nil {
			return err
		}
		if supplierStatus != StatusVerified {
			return &StateError{Message: "Verify the supplier itself before verifying its brand relationships."}
		}
		if brandStatus == "ARCHIVED" {
			return &StateError{Message: "The brand is archived."}
		}
		if _, err := tx.Exec(ctx, `UPDATE supplier_brand_links SET status = 'VERIFIED', authorisation_reference = $2,
			evidence_document_id = $3, effective_from = $4::date, effective_to = $5::date, verified_by = $6, verified_at = now(),
			version = version + 1 WHERE id = $1`, id, in.AuthorisationReference, in.EvidenceDocumentID, in.EffectiveFrom,
			str(in.EffectiveTo), verifier); err != nil {
			return err
		}
		if l, err = getLink(ctx, tx, id); err != nil {
			return err
		}
		return audit.Record(ctx, tx, withSummary(ev, audit.Summary{"status": audit.Change{From: LinkUnverified, To: LinkVerified},
			"authorisationReference": in.AuthorisationReference, "evidenceDocumentId": in.EvidenceDocumentID,
			"effectiveFrom": in.EffectiveFrom, "effectiveTo": str(in.EffectiveTo)}))
	})
	return l, err
}

// EndLink implements Repository. The relationship stays as history.
func (s *PGStore) EndLink(ctx context.Context, id string, in LinkEndInput, today time.Time, ev audit.Event) (BrandLink, error) {
	var l BrandLink
	err := s.tx(ctx, func(tx pgx.Tx) error {
		old, err := lockLink(ctx, tx, id, in.Version)
		if err != nil {
			return err
		}
		if old.Status == LinkEnded {
			return &StateError{Message: "This relationship has already ended."}
		}
		end := today.Format(time.DateOnly)
		if in.EffectiveTo != nil {
			end = strings.TrimSpace(*in.EffectiveTo)
		}
		if old.EffectiveFrom != nil && end < *old.EffectiveFrom {
			return invalid("effectiveTo", "must not be before the relationship's start")
		}
		if _, err := tx.Exec(ctx, `UPDATE supplier_brand_links SET status = 'ENDED', effective_to = $2::date, ended_reason = $3,
			version = version + 1 WHERE id = $1`, id, end, in.Reason); err != nil {
			return err
		}
		if l, err = getLink(ctx, tx, id); err != nil {
			return err
		}
		return audit.Record(ctx, tx, withSummary(ev, audit.Summary{"status": audit.Change{From: old.Status, To: LinkEnded},
			"effectiveTo": end, "reason": in.Reason}))
	})
	return l, err
}

// RecordAudit implements Repository.
func (s *PGStore) RecordAudit(ctx context.Context, ev audit.Event) error {
	return audit.Record(ctx, s.pool, ev)
}
