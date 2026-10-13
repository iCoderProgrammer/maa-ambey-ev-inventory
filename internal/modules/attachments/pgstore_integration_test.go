package attachments

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/audit"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/database/dbtest"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/modules/access"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/modules/auth"
)

func pgSetup(t *testing.T) (*pgxpool.Pool, *PGStore, string) {
	t.Helper()
	ctx := context.Background()
	pool := dbtest.MigratedPool(t)
	hash, _ := auth.HashPassword(testPassword)
	u, err := auth.NewPGStore(pool).CreateUser(ctx, audit.CLIActor("cmd/user:test"), "asha", "Asha", nil, hash)
	if err != nil {
		t.Fatal(err)
	}
	return pool, NewPGStore(pool), u.ID
}

func newRow(uploader string) Attachment {
	return Attachment{
		ID: uuid.NewString(), ObjectKey: "2026/10/" + uuid.NewString(), StorageDriver: "local",
		ContentType: "application/pdf", SizeBytes: 42, SHA256: make([]byte, 32), OriginalName: "bill.pdf", UploadedBy: uploader,
	}
}

func uploadEvent(uploader, id string) audit.Event {
	return audit.Event{Actor: audit.UserActor(uploader, "asha"), Action: "attachment.uploaded", EntityType: "attachment", EntityID: id, RequestID: "req-1"}
}

func TestPGInsertGetAndList(t *testing.T) {
	ctx := context.Background()
	pool, store, uid := pgSetup(t)
	a := newRow(uid)
	saved, err := store.Insert(ctx, a, uploadEvent(uid, a.ID))
	if err != nil {
		t.Fatal(err)
	}
	if saved.ID != a.ID || saved.CreatedAt.IsZero() || saved.LocationID != "" {
		t.Errorf("saved = %+v", saved)
	}
	got, err := store.Get(ctx, a.ID)
	if err != nil || got.ObjectKey != a.ObjectKey || got.UploadedBy != uid {
		t.Fatalf("get = %+v, %v", got, err)
	}
	if _, err := store.Get(ctx, uuid.NewString()); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown id: %v", err)
	}
	list, err := store.ListUnlinked(ctx, uid, 10)
	if err != nil || len(list) != 1 {
		t.Errorf("list = %v, %v", list, err)
	}
	var n int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE action = 'attachment.uploaded' AND entity_id = $1 AND request_id = 'req-1'`, a.ID).Scan(&n)
	if n != 1 {
		t.Errorf("audit rows = %d", n)
	}
}

func TestPGUnknownLocationAndInvalidEventRollBack(t *testing.T) {
	ctx := context.Background()
	pool, store, uid := pgSetup(t)
	a := newRow(uid)
	a.LocationID = uuid.NewString()
	if _, err := store.Insert(ctx, a, uploadEvent(uid, a.ID)); !errors.Is(err, ErrInvalidLocation) {
		t.Errorf("unknown location: %v", err)
	}
	b := newRow(uid)
	bad := uploadEvent(uid, b.ID)
	bad.Summary = audit.Summary{"token": "x"}
	if _, err := store.Insert(ctx, b, bad); !errors.Is(err, audit.ErrInvalidEvent) {
		t.Errorf("invalid event: %v", err)
	}
	var n int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM attachments`).Scan(&n)
	if n != 0 {
		t.Errorf("metadata stored without its audit event: %d rows", n)
	}
}

func TestPGAttachmentConstraints(t *testing.T) {
	ctx := context.Background()
	pool, _, uid := pgSetup(t)
	for name, mutate := range map[string]func(*Attachment){
		"bad key":        func(a *Attachment) { a.ObjectKey = "../etc/passwd" },
		"bad driver":     func(a *Attachment) { a.StorageDriver = "public-cdn" },
		"bad type":       func(a *Attachment) { a.ContentType = "text/html" },
		"zero size":      func(a *Attachment) { a.SizeBytes = 0 },
		"short checksum": func(a *Attachment) { a.SHA256 = []byte{1} },
		"blank name":     func(a *Attachment) { a.OriginalName = " " },
	} {
		a := newRow(uid)
		mutate(&a)
		_, err := pool.Exec(ctx, `INSERT INTO attachments (id, object_key, storage_driver, content_type, size_bytes, sha256, original_name, uploaded_by)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`, a.ID, a.ObjectKey, a.StorageDriver, a.ContentType, a.SizeBytes, a.SHA256, a.OriginalName, a.UploadedBy)
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != "23514" {
			t.Errorf("%s: %v", name, err)
		}
	}
	a := newRow(uid)
	_, err := pool.Exec(ctx, `INSERT INTO attachments (id, object_key, storage_driver, content_type, size_bytes, sha256, original_name, uploaded_by, owner_type)
		VALUES ($1, $2, 'local', 'application/pdf', 1, $3, 'x.pdf', $4, 'goods_receipt')`, a.ID, a.ObjectKey, a.SHA256, uid)
	if err == nil {
		t.Error("owner type without owner id accepted")
	}
}

func TestPGAttachmentsAreImmutableAndNeverDeleted(t *testing.T) {
	ctx := context.Background()
	pool, store, uid := pgSetup(t)
	a := newRow(uid)
	if _, err := store.Insert(ctx, a, uploadEvent(uid, a.ID)); err != nil {
		t.Fatal(err)
	}
	for name, sql := range map[string]string{
		"change type":     `UPDATE attachments SET content_type = 'image/png' WHERE id = $1`,
		"change key":      `UPDATE attachments SET object_key = '2026/11/' || gen_random_uuid() WHERE id = $1`,
		"change uploader": `UPDATE attachments SET uploaded_by = gen_random_uuid() WHERE id = $1`,
		"delete":          `DELETE FROM attachments WHERE id = $1`,
	} {
		_, err := pool.Exec(ctx, sql, a.ID)
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != "42501" {
			t.Errorf("%s: %v", name, err)
		}
	}
	if _, err := pool.Exec(ctx, `TRUNCATE attachments`); err == nil || !strings.Contains(err.Error(), "cannot be deleted") {
		t.Errorf("truncate: %v", err)
	}
	// Linking once is the only allowed change; relinking is refused.
	if _, err := pool.Exec(ctx, `UPDATE attachments SET owner_type = 'goods_receipt', owner_id = 'GRN-1' WHERE id = $1`, a.ID); err != nil {
		t.Fatalf("first link: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE attachments SET owner_id = 'GRN-2' WHERE id = $1`, a.ID); err == nil {
		t.Error("relink accepted")
	}
	got, _ := store.Get(ctx, a.ID)
	if got.OwnerType != "goods_receipt" || got.ContentType != "application/pdf" {
		t.Errorf("row = %+v", got)
	}
}

// End to end over PostgreSQL and the local store: upload, then download, with
// both audit events in the database.
func TestPGServiceRoundTrip(t *testing.T) {
	ctx := context.Background()
	pool, store, uid := pgSetup(t)
	objects, err := NewLocalStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	signer, _ := NewSigner(strings.Repeat("k", 32), time.Minute)
	svc := NewService(store, objects, signer, 1<<20, slog.New(slog.NewTextHandler(io.Discard, nil)))
	a := access.NewAccess(uid, nil, []string{access.PermAttachmentsUpload, access.PermAttachmentsView}, nil, false)
	actor := Actor{UserID: uid, Username: "asha", RequestID: "req-up"}

	att, err := svc.Upload(ctx, actor, a, UploadInput{FileName: "bill.pdf", Body: strings.NewReader(string(pdfBytes))})
	if err != nil {
		t.Fatal(err)
	}
	_, link, err := svc.IssueLink(ctx, a, att.ID, DispositionAttachment)
	if err != nil {
		t.Fatal(err)
	}
	actor.RequestID = "req-down"
	_, f, err := svc.Open(ctx, actor, a, att.ID, DispositionAttachment, strconv.FormatInt(link.Expires.Unix(), 10), link.Signature)
	if err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	rows, _ := pool.Query(ctx, `SELECT action, request_id, summary::text FROM audit_events WHERE entity_type = 'attachment' ORDER BY id`)
	var got []string
	for rows.Next() {
		var action, req, summary string
		_ = rows.Scan(&action, &req, &summary)
		got = append(got, action+"/"+req)
		if strings.Contains(summary, "bill.pdf") {
			t.Errorf("file name in audit summary: %s", summary)
		}
	}
	rows.Close()
	if strings.Join(got, ",") != "attachment.uploaded/req-up,attachment.downloaded/req-down" {
		t.Errorf("audit = %v", got)
	}
}
