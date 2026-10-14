// Package attachments stores private files such as PDI photos and supplier
// bills (B-09, FR-016, SEC-005).
//
// Files are kept in an ObjectStore under server-generated keys and are only
// ever read back through the API: the caller must hold a session, the
// attachments.view permission and pass the object check (location scope; an
// attachment not yet linked to a business record is visible to its uploader
// only). Downloads use short-lived links signed for one user; there is no
// public URL. Content type is detected from the bytes, not trusted from the
// client, and only an allow-list of types up to a size limit is accepted.
//
// Later modules (goods receipt, PDI …) link attachments to their records and
// must add their own owner check to canSee before linked attachments become
// visible; until then linked attachments are denied (fail closed). Supplier
// documents (owner "supplier", D-01) need suppliers.view.
package attachments

import (
	"bufio"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/audit"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/modules/access"
)

// Errors returned by the service. The HTTP layer maps them to status codes.
var (
	ErrNotFound         = errors.New("attachment not found")
	ErrForbidden        = errors.New("not allowed")
	ErrTooLarge         = errors.New("file is too large")
	ErrEmpty            = errors.New("file is empty")
	ErrTypeNotAllowed   = errors.New("file type is not allowed")
	ErrTypeMismatch     = errors.New("file content does not match its declared type or extension")
	ErrInvalidLocation  = errors.New("unknown location")
	ErrInlineNotAllowed = errors.New("only images can be shown inline")
	ErrBadDisposition   = errors.New("disposition must be attachment or inline")
)

// FileType is an accepted content type and its file-name extensions.
type FileType struct {
	ContentType string   `json:"contentType" example:"application/pdf"`
	Extensions  []string `json:"extensions" example:".pdf"`
	// Image types can be previewed inline; others are always downloaded.
	Previewable bool `json:"previewable" example:"false"`
}

// AllowedTypes is the accepted allow-list (proposal pending owner confirmation:
// photos and scanned/PDF documents for PDI and supplier bills). Keep it in step
// with the attachments_content_type_valid CHECK in migration 000006.
var AllowedTypes = []FileType{
	{ContentType: "image/jpeg", Extensions: []string{".jpg", ".jpeg"}, Previewable: true},
	{ContentType: "image/png", Extensions: []string{".png"}, Previewable: true},
	{ContentType: "image/webp", Extensions: []string{".webp"}, Previewable: true},
	{ContentType: "application/pdf", Extensions: []string{".pdf"}},
}

func fileType(contentType string) (FileType, bool) {
	i := slices.IndexFunc(AllowedTypes, func(t FileType) bool { return t.ContentType == contentType })
	if i < 0 {
		return FileType{}, false
	}
	return AllowedTypes[i], true
}

// Attachment is the stored metadata.
type Attachment struct {
	ID            string
	ObjectKey     string
	StorageDriver string
	ContentType   string
	SizeBytes     int64
	SHA256        []byte
	OriginalName  string
	UploadedBy    string
	LocationID    string
	OwnerType     string
	OwnerID       string
	CreatedAt     time.Time
}

// Repository persists attachment metadata.
type Repository interface {
	// Insert stores the metadata and its audit event in one transaction.
	// It returns ErrInvalidLocation when the location does not exist.
	Insert(ctx context.Context, a Attachment, ev audit.Event) (Attachment, error)
	// Get returns ErrNotFound for an unknown ID.
	Get(ctx context.Context, id string) (Attachment, error)
	// ListUnlinked returns the user's newest unlinked uploads.
	ListUnlinked(ctx context.Context, uploaderID string, limit int) ([]Attachment, error)
	RecordAudit(ctx context.Context, ev audit.Event) error
}

// Actor is the signed-in user making the request.
type Actor struct {
	UserID    string
	Username  string
	RequestID string
}

func (a Actor) audit() audit.Actor { return audit.UserActor(a.UserID, a.Username) }

// Service implements uploads, metadata and downloads.
type Service struct {
	repo     Repository
	store    ObjectStore
	signer   *Signer
	maxBytes int64
	log      *slog.Logger
	now      func() time.Time
}

// NewService builds the service.
func NewService(repo Repository, store ObjectStore, signer *Signer, maxBytes int64, log *slog.Logger) *Service {
	return &Service{repo: repo, store: store, signer: signer, maxBytes: maxBytes, log: log, now: time.Now}
}

// MaxBytes is the per-file size limit.
func (s *Service) MaxBytes() int64 { return s.maxBytes }

// canSee is the object-level check for metadata and downloads.
func canSee(a access.Access, att Attachment) bool {
	if !a.Has(access.PermAttachmentsView) {
		return false
	}
	if att.LocationID != "" && !a.Scope.Allows(att.LocationID) {
		return false
	}
	switch att.OwnerType {
	case "":
		return att.UploadedBy == a.UserID
	case "supplier":
		// Supplier documents (D-01): visible to whoever may view suppliers.
		return a.Has(access.PermSupplierView)
	default:
		// Other owners need their module's check, which none provides yet.
		// Deny rather than guess.
		return false
	}
}

// UploadInput is one file to store.
type UploadInput struct {
	FileName     string
	DeclaredType string // the multipart part's Content-Type, if any
	LocationID   string
	Body         io.Reader
}

// Upload validates and stores a file. The caller has checked attachments.upload.
func (s *Service) Upload(ctx context.Context, actor Actor, a access.Access, in UploadInput) (Attachment, error) {
	if !a.Has(access.PermAttachmentsUpload) {
		return Attachment{}, ErrForbidden
	}
	if in.LocationID != "" {
		if _, err := uuid.Parse(in.LocationID); err != nil {
			return Attachment{}, ErrInvalidLocation
		}
		if err := a.Scope.Check(in.LocationID); err != nil {
			return Attachment{}, ErrForbidden
		}
	}
	name := sanitizeFileName(in.FileName)

	br := bufio.NewReaderSize(in.Body, 512)
	head, err := br.Peek(512)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, bufio.ErrBufferFull) {
		return Attachment{}, err
	}
	if len(head) == 0 {
		return Attachment{}, ErrEmpty
	}
	detected, _, _ := strings.Cut(http.DetectContentType(head), ";")
	ft, ok := fileType(detected)
	if !ok {
		return Attachment{}, ErrTypeNotAllowed
	}
	declared := strings.ToLower(strings.TrimSpace(strings.SplitN(in.DeclaredType, ";", 2)[0]))
	if declared != "" && declared != "application/octet-stream" && declared != ft.ContentType {
		return Attachment{}, ErrTypeMismatch
	}
	if !slices.Contains(ft.Extensions, strings.ToLower(path.Ext(name))) {
		return Attachment{}, ErrTypeMismatch
	}

	now := s.now().UTC()
	att := Attachment{
		ID:            uuid.NewString(),
		StorageDriver: s.store.Driver(),
		ContentType:   ft.ContentType,
		OriginalName:  name,
		UploadedBy:    actor.UserID,
		LocationID:    in.LocationID,
	}
	att.ObjectKey = fmt.Sprintf("%04d/%02d/%s", now.Year(), int(now.Month()), uuid.NewString())

	sum := sha256.New()
	lr := &limitReader{r: io.TeeReader(br, sum), max: s.maxBytes}
	n, err := s.store.Put(ctx, att.ObjectKey, lr)
	if err != nil {
		if errors.Is(err, ErrTooLarge) {
			return Attachment{}, ErrTooLarge
		}
		return Attachment{}, fmt.Errorf("store attachment: %w", err)
	}
	att.SizeBytes = n
	att.SHA256 = sum.Sum(nil)

	summary := audit.Summary{"contentType": att.ContentType, "sizeBytes": att.SizeBytes}
	if att.LocationID != "" {
		summary["locationId"] = att.LocationID
	}
	saved, err := s.repo.Insert(ctx, att, audit.Event{
		Actor: actor.audit(), Action: "attachment.uploaded", EntityType: "attachment", EntityID: att.ID,
		RequestID: actor.RequestID, Summary: summary,
	})
	if err != nil {
		// No metadata means the object is unreachable; remove it.
		if derr := s.store.Delete(context.WithoutCancel(ctx), att.ObjectKey); derr != nil {
			s.log.ErrorContext(ctx, "orphaned attachment object", slog.String("object_key", att.ObjectKey), slog.Any("error", derr))
		}
		return Attachment{}, err
	}
	return saved, nil
}

// Get returns metadata the caller may see; anything else is ErrNotFound so
// the existence of other users' files is not revealed.
func (s *Service) Get(ctx context.Context, a access.Access, id string) (Attachment, error) {
	if _, err := uuid.Parse(id); err != nil {
		return Attachment{}, ErrNotFound
	}
	att, err := s.repo.Get(ctx, id)
	if err != nil {
		return Attachment{}, err
	}
	if !canSee(a, att) {
		return Attachment{}, ErrNotFound
	}
	return att, nil
}

// ListMine returns the caller's unlinked uploads that are still in scope.
func (s *Service) ListMine(ctx context.Context, a access.Access, limit int) ([]Attachment, error) {
	list, err := s.repo.ListUnlinked(ctx, a.UserID, limit)
	if err != nil {
		return nil, err
	}
	out := list[:0]
	for _, att := range list {
		if canSee(a, att) {
			out = append(out, att)
		}
	}
	return out, nil
}

// IssueLink authorises the caller and returns a signed, short-lived link.
func (s *Service) IssueLink(ctx context.Context, a access.Access, id, disposition string) (Attachment, Link, error) {
	att, err := s.Get(ctx, a, id)
	if err != nil {
		return Attachment{}, Link{}, err
	}
	if err := checkDisposition(att, disposition); err != nil {
		return Attachment{}, Link{}, err
	}
	return att, s.signer.Sign(att.ID, a.UserID, disposition), nil
}

// Open verifies the link, re-checks access (a permission revoked after the
// link was issued takes effect at once), audits the download and returns the
// file.
func (s *Service) Open(ctx context.Context, actor Actor, a access.Access, id, disposition, expires, signature string) (Attachment, io.ReadSeekCloser, error) {
	if err := s.signer.Verify(id, a.UserID, disposition, expires, signature); err != nil {
		return Attachment{}, nil, err
	}
	att, err := s.Get(ctx, a, id)
	if err != nil {
		return Attachment{}, nil, err
	}
	if err := checkDisposition(att, disposition); err != nil {
		return Attachment{}, nil, err
	}
	f, err := s.store.Open(ctx, att.ObjectKey)
	if err != nil {
		return Attachment{}, nil, fmt.Errorf("open attachment %s: %w", att.ID, err)
	}
	if err := s.repo.RecordAudit(ctx, audit.Event{
		Actor: actor.audit(), Action: "attachment.downloaded", EntityType: "attachment", EntityID: att.ID,
		RequestID: actor.RequestID, Summary: audit.Summary{"disposition": disposition},
	}); err != nil {
		_ = f.Close()
		return Attachment{}, nil, err
	}
	return att, f, nil
}

func checkDisposition(att Attachment, disposition string) error {
	switch disposition {
	case DispositionAttachment:
		return nil
	case DispositionInline:
		if ft, ok := fileType(att.ContentType); ok && ft.Previewable {
			return nil
		}
		return ErrInlineNotAllowed
	default:
		return ErrBadDisposition
	}
}

// sanitizeFileName keeps a display name only: no directories, no control
// characters, at most 200 characters.
func sanitizeFileName(name string) string {
	name = strings.ReplaceAll(name, "\\", "/")
	name = path.Base(name)
	name = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || r == '"' || r == utf8.RuneError {
			return -1
		}
		return r
	}, name)
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == ".." || name == "/" {
		name = "attachment"
	}
	if utf8.RuneCountInString(name) > 200 {
		ext := path.Ext(name)
		if utf8.RuneCountInString(ext) > 10 {
			ext = ""
		}
		r := []rune(strings.TrimSuffix(name, ext))
		name = string(r[:200-utf8.RuneCountInString(ext)]) + ext
	}
	return name
}

// limitReader fails with ErrTooLarge once more than max bytes are read.
type limitReader struct {
	r    io.Reader
	max  int64
	read int64
}

func (l *limitReader) Read(p []byte) (int, error) {
	n, err := l.r.Read(p)
	l.read += int64(n)
	if l.read > l.max {
		return n, ErrTooLarge
	}
	return n, err
}
