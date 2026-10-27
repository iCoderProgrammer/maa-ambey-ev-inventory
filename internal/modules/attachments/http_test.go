package attachments

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/audit"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/httpx"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/access"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/auth"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/auth/authtest"
)

const testPassword = "correct horse battery staple"

// Minimal files whose content sniffs as the named type.
var (
	pngBytes  = append([]byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR"), bytes.Repeat([]byte{1}, 64)...)
	jpegBytes = append([]byte("\xff\xd8\xff\xe0\x00\x10JFIF\x00"), bytes.Repeat([]byte{2}, 64)...)
	webpBytes = append([]byte("RIFF\x24\x00\x00\x00WEBPVP8 "), bytes.Repeat([]byte{3}, 64)...)
	pdfBytes  = []byte("%PDF-1.4\n1 0 obj\n<< /Type /Catalog >>\nendobj\ntrailer\n%%EOF\n")
)

// memRepo is an in-memory Repository.
type memRepo struct {
	mu        sync.Mutex
	rows      map[string]Attachment
	events    []audit.Event
	insertErr error
}

func (m *memRepo) Insert(_ context.Context, a Attachment, ev audit.Event) (Attachment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.insertErr != nil {
		return Attachment{}, m.insertErr
	}
	a.CreatedAt = time.Now().UTC()
	m.rows[a.ID] = a
	m.events = append(m.events, ev)
	return a, nil
}

func (m *memRepo) Get(_ context.Context, id string) (Attachment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.rows[id]
	if !ok {
		return Attachment{}, ErrNotFound
	}
	return a, nil
}

func (m *memRepo) ListUnlinked(_ context.Context, uploaderID string, limit int) ([]Attachment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := []Attachment{}
	for _, a := range m.rows {
		if a.UploadedBy == uploaderID && a.OwnerType == "" && len(out) < limit {
			out = append(out, a)
		}
	}
	return out, nil
}

func (m *memRepo) RecordAudit(_ context.Context, ev audit.Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, ev)
	return nil
}

func (m *memRepo) actions() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]string, len(m.events))
	for i, e := range m.events {
		out[i] = e.Action
	}
	return out
}

// grants serves Access per user ID.
type grants struct {
	mu    sync.Mutex
	perms map[string][]string
	scope map[string][]access.Location
}

func (g *grants) LoadAccess(_ context.Context, userID string, _ bool) (access.Access, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	return access.NewAccess(userID, nil, g.perms[userID], g.scope[userID], false), nil
}

func (g *grants) RoleMatrix(context.Context) (access.Matrix, error) { return access.Matrix{}, nil }

func (g *grants) set(userID string, perms ...string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.perms[userID] = perms
}

type session struct {
	userID string
	cookie *http.Cookie
	csrf   string
}

type harness struct {
	t      *testing.T
	router *gin.Engine
	repo   *memRepo
	grants *grants
	svc    *Service
	dir    string
	asha   session
	ravi   session
}

const maxTestBytes = 4096

func newHarness(t *testing.T) *harness {
	t.Helper()
	gin.SetMode(gin.TestMode)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	store := authtest.NewStore()
	asha := store.AddUser("asha", "Asha Verma", testPassword, auth.StatusActive)
	ravi := store.AddUser("ravi", "Ravi Kumar", testPassword, auth.StatusActive)
	authSvc, err := auth.NewService(store, auth.Config{IdleTimeout: time.Hour, AbsoluteTimeout: 2 * time.Hour},
		auth.NewLoginThrottle(5, 20, time.Minute), log)
	if err != nil {
		t.Fatal(err)
	}
	authH := auth.NewHandler(authSvc, auth.CookieConfig{}, log)
	g := &grants{perms: map[string][]string{}, scope: map[string][]access.Location{}}
	accessH := access.NewHandler(g, false, log)

	dir := t.TempDir()
	objects, err := NewLocalStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := NewSigner(strings.Repeat("s", 32), time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	repo := &memRepo{rows: map[string]Attachment{}}
	svc := NewService(repo, objects, signer, maxTestBytes, log)

	r := gin.New()
	api := r.Group("/api/v1")
	api.Use(authH.RequireSession(auth.PublicRoutes("/api/v1")...))
	authH.Register(api)
	NewHandler(svc, accessH, "/api/v1", log).Register(api)

	h := &harness{t: t, router: r, repo: repo, grants: g, svc: svc, dir: dir}
	h.asha = h.login("asha", asha.ID)
	h.ravi = h.login("ravi", ravi.ID)
	return h
}

func (h *harness) login(username, id string) session {
	h.t.Helper()
	w := h.do(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"username":"`+username+`","password":"`+testPassword+`"}`), "application/json", nil)
	var body struct {
		Data auth.SessionResponse `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	s := session{userID: id, csrf: body.Data.CSRFToken}
	for _, c := range w.Result().Cookies() {
		if c.Name == "maa_session" {
			s.cookie = c
		}
	}
	if s.cookie == nil {
		h.t.Fatalf("login %s failed: %s", username, w.Body)
	}
	return s
}

func (h *harness) do(method, target string, body io.Reader, contentType string, s *session) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, body)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if s != nil {
		req.AddCookie(s.cookie)
		if method != http.MethodGet {
			req.Header.Set(auth.CSRFHeader, s.csrf)
		}
	}
	w := httptest.NewRecorder()
	h.router.ServeHTTP(w, req)
	return w
}

func multipartBody(t *testing.T, fileName, partType string, content []byte) (io.Reader, string) {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	hdr := textproto.MIMEHeader{}
	hdr.Set("Content-Disposition", `form-data; name="file"; filename="`+fileName+`"`)
	if partType != "" {
		hdr.Set("Content-Type", partType)
	}
	pw, err := mw.CreatePart(hdr)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = pw.Write(content)
	_ = mw.Close()
	return &buf, mw.FormDataContentType()
}

func (h *harness) upload(s *session, fileName, partType string, content []byte, query string) *httptest.ResponseRecorder {
	body, ct := multipartBody(h.t, fileName, partType, content)
	return h.do(http.MethodPost, "/api/v1/attachments"+query, body, ct, s)
}

func (h *harness) uploadOK(s *session, fileName string, content []byte) AttachmentResponse {
	h.t.Helper()
	w := h.upload(s, fileName, "", content, "")
	if w.Code != http.StatusCreated {
		h.t.Fatalf("upload %s: %d %s", fileName, w.Code, w.Body)
	}
	var body struct {
		Data AttachmentResponse `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		h.t.Fatal(err)
	}
	return body.Data
}

func (h *harness) link(s *session, id, disposition string) (*httptest.ResponseRecorder, LinkResponse) {
	w := h.do(http.MethodGet, "/api/v1/attachments/"+id+"/link?disposition="+disposition, nil, "", s)
	var body struct {
		Data LinkResponse `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	return w, body.Data
}

func errorCode(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	var body httpx.ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("not an error envelope: %d %q", w.Code, w.Body)
	}
	return body.Error.Code
}

func storedFiles(t *testing.T, dir string) []string {
	t.Helper()
	var files []string
	_ = filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			files = append(files, p)
		}
		return nil
	})
	return files
}

func TestUploadAcceptsAllowedTypes(t *testing.T) {
	h := newHarness(t)
	h.grants.set(h.asha.userID, access.PermAttachmentsUpload, access.PermAttachmentsView)
	for name, tc := range map[string]struct {
		file    string
		content []byte
		want    string
		preview bool
	}{
		"png":  {"pdi photo.PNG", pngBytes, "image/png", true},
		"jpeg": {"scooter.jpeg", jpegBytes, "image/jpeg", true},
		"webp": {"frame.webp", webpBytes, "image/webp", true},
		"pdf":  {"supplier-bill.pdf", pdfBytes, "application/pdf", false},
	} {
		t.Run(name, func(t *testing.T) {
			got := h.uploadOK(&h.asha, tc.file, tc.content)
			if got.ContentType != tc.want || got.SizeBytes != int64(len(tc.content)) || got.FileName != tc.file ||
				got.UploadedBy != h.asha.userID || got.Linked || got.Previewable != tc.preview || len(got.SHA256) != 64 {
				t.Errorf("metadata = %+v", got)
			}
		})
	}
	// Files are stored owner-only, under generated keys that do not contain the file name.
	for _, p := range storedFiles(t, h.dir) {
		info, _ := os.Stat(p)
		if info.Mode().Perm()&0o077 != 0 {
			t.Errorf("%s is readable by others: %v", p, info.Mode())
		}
		if strings.Contains(p, "supplier") || strings.Contains(p, "scooter") {
			t.Errorf("object key leaks the file name: %s", p)
		}
	}
	if got := h.repo.actions(); len(got) != 4 || got[0] != "attachment.uploaded" {
		t.Errorf("audit = %v", got)
	}
}

func TestUploadResponseNeverExposesStorage(t *testing.T) {
	h := newHarness(t)
	h.grants.set(h.asha.userID, access.PermAttachmentsUpload)
	w := h.upload(&h.asha, "a.pdf", "application/pdf", pdfBytes, "")
	if w.Code != http.StatusCreated {
		t.Fatalf("%d %s", w.Code, w.Body)
	}
	for _, leak := range []string{"objectKey", "object_key", "storage", h.dir, "http://", "https://"} {
		if strings.Contains(w.Body.String(), leak) {
			t.Errorf("response contains %q: %s", leak, w.Body)
		}
	}
}

func TestUploadRejectsInvalidFiles(t *testing.T) {
	h := newHarness(t)
	h.grants.set(h.asha.userID, access.PermAttachmentsUpload)
	for name, tc := range map[string]struct {
		file, partType string
		content        []byte
		status         int
		code           string
	}{
		"plain text":                {"notes.txt", "text/plain", []byte("hello there, this is text"), 415, httpx.CodeUnsupportedMediaType},
		"html renamed to pdf":       {"bill.pdf", "application/pdf", []byte("<html><script>alert(1)</script></html>"), 415, httpx.CodeUnsupportedMediaType},
		"svg image":                 {"logo.svg", "image/svg+xml", []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script/></svg>`), 415, httpx.CodeUnsupportedMediaType},
		"executable":                {"setup.pdf", "", append([]byte("MZ\x90\x00"), bytes.Repeat([]byte{0}, 64)...), 415, httpx.CodeUnsupportedMediaType},
		"pdf content, png name":     {"photo.png", "", pdfBytes, 415, httpx.CodeUnsupportedMediaType},
		"pdf content, png declared": {"bill.pdf", "image/png", pdfBytes, 415, httpx.CodeUnsupportedMediaType},
		"no extension":              {"bill", "", pdfBytes, 415, httpx.CodeUnsupportedMediaType},
		"empty":                     {"empty.pdf", "application/pdf", nil, 422, httpx.CodeValidationFailed},
		"oversized":                 {"big.pdf", "application/pdf", append(append([]byte{}, pdfBytes...), bytes.Repeat([]byte("x"), maxTestBytes)...), 413, httpx.CodeFileTooLarge},
		"far oversized":             {"huge.pdf", "application/pdf", append(append([]byte{}, pdfBytes...), bytes.Repeat([]byte("x"), 200<<10)...), 413, httpx.CodeFileTooLarge},
	} {
		t.Run(name, func(t *testing.T) {
			w := h.upload(&h.asha, tc.file, tc.partType, tc.content, "")
			if w.Code != tc.status || errorCode(t, w) != tc.code {
				t.Errorf("status %d %s, want %d %s", w.Code, w.Body, tc.status, tc.code)
			}
		})
	}
	if files := storedFiles(t, h.dir); len(files) != 0 {
		t.Errorf("rejected uploads left files behind: %v", files)
	}
	if len(h.repo.rows) != 0 || len(h.repo.actions()) != 0 {
		t.Errorf("rejected uploads stored metadata/audit: %d rows, %v", len(h.repo.rows), h.repo.actions())
	}
}

func TestUploadRequestShape(t *testing.T) {
	h := newHarness(t)
	h.grants.set(h.asha.userID, access.PermAttachmentsUpload)
	w := h.do(http.MethodPost, "/api/v1/attachments", strings.NewReader(`{"file":"x"}`), "application/json", &h.asha)
	if w.Code != http.StatusUnsupportedMediaType {
		t.Errorf("json body: %d %s", w.Code, w.Body)
	}
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	_ = mw.WriteField("note", "no file here")
	_ = mw.Close()
	w = h.do(http.MethodPost, "/api/v1/attachments", &buf, mw.FormDataContentType(), &h.asha)
	if w.Code != http.StatusBadRequest || errorCode(t, w) != httpx.CodeValidationFailed {
		t.Errorf("no file part: %d %s", w.Code, w.Body)
	}
}

func TestMetadataStoreFailureRemovesTheObject(t *testing.T) {
	h := newHarness(t)
	h.grants.set(h.asha.userID, access.PermAttachmentsUpload)
	h.repo.insertErr = errors.New("database down")
	if w := h.upload(&h.asha, "a.pdf", "", pdfBytes, ""); w.Code != http.StatusInternalServerError {
		t.Fatalf("%d %s", w.Code, w.Body)
	}
	if files := storedFiles(t, h.dir); len(files) != 0 {
		t.Errorf("orphaned object: %v", files)
	}
}

func TestEveryRouteNeedsSessionAndPermission(t *testing.T) {
	h := newHarness(t)
	h.grants.set(h.asha.userID, access.PermAttachmentsUpload, access.PermAttachmentsView)
	att := h.uploadOK(&h.asha, "a.pdf", pdfBytes)
	_, link := h.link(&h.asha, att.ID, DispositionAttachment)

	routes := []struct{ method, path string }{
		{http.MethodGet, "/api/v1/attachments/limits"},
		{http.MethodPost, "/api/v1/attachments"},
		{http.MethodGet, "/api/v1/attachments"},
		{http.MethodGet, "/api/v1/attachments/" + att.ID},
		{http.MethodGet, "/api/v1/attachments/" + att.ID + "/link"},
		{http.MethodGet, link.URL},
	}
	for _, r := range routes {
		if w := h.do(r.method, r.path, nil, "", nil); w.Code != http.StatusUnauthorized {
			t.Errorf("anonymous %s %s: %d", r.method, r.path, w.Code)
		}
	}
	// Ravi has a session but no permission (nothing is granted by default).
	for _, r := range routes {
		w := h.do(r.method, r.path, nil, "", &h.ravi)
		if w.Code != http.StatusForbidden || errorCode(t, w) != httpx.CodeForbidden {
			t.Errorf("no permission %s %s: %d %s", r.method, r.path, w.Code, w.Body)
		}
	}
	// Upload needs the CSRF token like every mutation.
	body, ct := multipartBody(t, "a.pdf", "", pdfBytes)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/attachments", body)
	req.Header.Set("Content-Type", ct)
	req.AddCookie(h.asha.cookie)
	w := httptest.NewRecorder()
	h.router.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden || errorCode(t, w) != httpx.CodeCSRFFailed {
		t.Errorf("upload without CSRF: %d %s", w.Code, w.Body)
	}
	// View alone cannot upload; upload alone cannot read.
	h.grants.set(h.ravi.userID, access.PermAttachmentsView)
	if w := h.upload(&h.ravi, "a.pdf", "", pdfBytes, ""); w.Code != http.StatusForbidden {
		t.Errorf("view-only upload: %d", w.Code)
	}
	h.grants.set(h.ravi.userID, access.PermAttachmentsUpload)
	if w := h.do(http.MethodGet, "/api/v1/attachments", nil, "", &h.ravi); w.Code != http.StatusForbidden {
		t.Errorf("upload-only list: %d", w.Code)
	}
}

func TestDownloadWithAValidLink(t *testing.T) {
	h := newHarness(t)
	h.grants.set(h.asha.userID, access.PermAttachmentsUpload, access.PermAttachmentsView)
	att := h.uploadOK(&h.asha, "बिल 1.pdf", pdfBytes)

	w, link := h.link(&h.asha, att.ID, DispositionAttachment)
	if w.Code != http.StatusOK || !strings.HasPrefix(link.URL, "/api/v1/attachments/"+att.ID+"/content?") {
		t.Fatalf("link: %d %s", w.Code, w.Body)
	}
	if ttl := time.Until(link.ExpiresAt); ttl <= 0 || ttl > time.Minute+time.Second {
		t.Errorf("link must be short-lived, expires in %v", ttl)
	}
	w = h.do(http.MethodGet, link.URL, nil, "", &h.asha)
	if w.Code != http.StatusOK || !bytes.Equal(w.Body.Bytes(), pdfBytes) {
		t.Fatalf("download: %d %q", w.Code, w.Body)
	}
	hdr := w.Result().Header
	for k, want := range map[string]string{
		"Content-Type":           "application/pdf",
		"X-Content-Type-Options": "nosniff",
		"Cache-Control":          "private, no-store",
	} {
		if hdr.Get(k) != want {
			t.Errorf("%s = %q, want %q", k, hdr.Get(k), want)
		}
	}
	if !strings.Contains(hdr.Get("Content-Security-Policy"), "sandbox") {
		t.Errorf("CSP = %q", hdr.Get("Content-Security-Policy"))
	}
	if cd := hdr.Get("Content-Disposition"); !strings.HasPrefix(cd, "attachment;") || !strings.Contains(cd, "filename*=utf-8''") {
		t.Errorf("Content-Disposition = %q", cd)
	}
	if got := h.repo.actions(); got[len(got)-1] != "attachment.downloaded" {
		t.Errorf("download not audited: %v", got)
	}

	// Images may be previewed inline; PDFs may not.
	img := h.uploadOK(&h.asha, "p.png", pngBytes)
	if w, l := h.link(&h.asha, img.ID, DispositionInline); w.Code != http.StatusOK {
		t.Errorf("inline image link: %d", w.Code)
	} else if w := h.do(http.MethodGet, l.URL, nil, "", &h.asha); w.Code != http.StatusOK || !strings.HasPrefix(w.Header().Get("Content-Disposition"), "inline") {
		t.Errorf("inline image: %d %q", w.Code, w.Header().Get("Content-Disposition"))
	}
	if w, _ := h.link(&h.asha, att.ID, DispositionInline); w.Code != http.StatusUnprocessableEntity {
		t.Errorf("inline pdf: %d", w.Code)
	}
	if w, _ := h.link(&h.asha, att.ID, "render"); w.Code != http.StatusUnprocessableEntity {
		t.Errorf("unknown disposition: %d", w.Code)
	}
}

func TestInvalidAndExpiredLinksAreRefused(t *testing.T) {
	h := newHarness(t)
	h.grants.set(h.asha.userID, access.PermAttachmentsUpload, access.PermAttachmentsView)
	att := h.uploadOK(&h.asha, "a.png", pngBytes)
	other := h.uploadOK(&h.asha, "b.png", pngBytes)
	_, link := h.link(&h.asha, att.ID, DispositionAttachment)
	u, _ := url.Parse(link.URL)
	q := u.Query()

	with := func(mutate func(url.Values), path string) string {
		v := url.Values{}
		for k, vals := range q {
			v[k] = append([]string(nil), vals...)
		}
		mutate(v)
		return path + "?" + v.Encode()
	}
	base := "/api/v1/attachments/" + att.ID + "/content"
	for name, target := range map[string]string{
		"no signature":        base + "?disposition=attachment&expires=" + q.Get("expires"),
		"tampered signature":  with(func(v url.Values) { v.Set("signature", "A"+v.Get("signature")[1:]) }, base),
		"changed disposition": with(func(v url.Values) { v.Set("disposition", "inline") }, base),
		"extended expiry":     with(func(v url.Values) { v.Set("expires", "4102444800") }, base),
		"garbage expiry":      with(func(v url.Values) { v.Set("expires", "soon") }, base),
		"other attachment":    with(func(url.Values) {}, "/api/v1/attachments/"+other.ID+"/content"),
		"unknown attachment":  with(func(url.Values) {}, "/api/v1/attachments/00000000-0000-4000-8000-000000000000/content"),
	} {
		w := h.do(http.MethodGet, target, nil, "", &h.asha)
		if w.Code != http.StatusForbidden || errorCode(t, w) != httpx.CodeDownloadLinkInvalid {
			t.Errorf("%s: %d %s", name, w.Code, w.Body)
		}
	}

	// Expired: move the signer's clock past the TTL.
	h.svc.signer.now = func() time.Time { return time.Now().Add(2 * time.Minute) }
	w := h.do(http.MethodGet, link.URL, nil, "", &h.asha)
	if w.Code != http.StatusForbidden || errorCode(t, w) != httpx.CodeDownloadLinkExpired {
		t.Errorf("expired: %d %s", w.Code, w.Body)
	}
	if got := h.repo.actions(); got[len(got)-1] == "attachment.downloaded" {
		t.Error("refused download was audited as a download")
	}
}

func TestOtherUsersCannotReachAnAttachment(t *testing.T) {
	h := newHarness(t)
	h.grants.set(h.asha.userID, access.PermAttachmentsUpload, access.PermAttachmentsView)
	h.grants.set(h.ravi.userID, access.PermAttachmentsUpload, access.PermAttachmentsView)
	att := h.uploadOK(&h.asha, "a.pdf", pdfBytes)
	_, ashaLink := h.link(&h.asha, att.ID, DispositionAttachment)

	// Ravi holds the permission, but an unlinked upload belongs to its uploader:
	// metadata and links are 404 (existence not revealed), Asha's link is not his.
	for _, p := range []string{"/api/v1/attachments/" + att.ID, "/api/v1/attachments/" + att.ID + "/link"} {
		if w := h.do(http.MethodGet, p, nil, "", &h.ravi); w.Code != http.StatusNotFound {
			t.Errorf("ravi %s: %d", p, w.Code)
		}
	}
	if w := h.do(http.MethodGet, ashaLink.URL, nil, "", &h.ravi); w.Code != http.StatusForbidden || errorCode(t, w) != httpx.CodeDownloadLinkInvalid {
		t.Errorf("ravi with asha's link: %d %s", w.Code, w.Body)
	}
	var list struct {
		Data []AttachmentResponse `json:"data"`
	}
	w := h.do(http.MethodGet, "/api/v1/attachments", nil, "", &h.ravi)
	_ = json.Unmarshal(w.Body.Bytes(), &list)
	if w.Code != http.StatusOK || len(list.Data) != 0 {
		t.Errorf("ravi's list: %d %s", w.Code, w.Body)
	}
	if w := h.do(http.MethodGet, "/api/v1/attachments/not-a-uuid", nil, "", &h.ravi); w.Code != http.StatusNotFound {
		t.Errorf("malformed id: %d", w.Code)
	}
}

func TestRevokedPermissionStopsAnIssuedLink(t *testing.T) {
	h := newHarness(t)
	h.grants.set(h.asha.userID, access.PermAttachmentsUpload, access.PermAttachmentsView)
	att := h.uploadOK(&h.asha, "a.pdf", pdfBytes)
	_, link := h.link(&h.asha, att.ID, DispositionAttachment)
	h.grants.set(h.asha.userID)
	if w := h.do(http.MethodGet, link.URL, nil, "", &h.asha); w.Code != http.StatusForbidden || errorCode(t, w) != httpx.CodeForbidden {
		t.Errorf("after revoke: %d %s", w.Code, w.Body)
	}
}

func TestLocationScopeIsEnforced(t *testing.T) {
	h := newHarness(t)
	srA := access.Location{ID: "11111111-1111-4111-8111-111111111111", Code: "SR-A", Name: "A"}
	srB := access.Location{ID: "22222222-2222-4222-8222-222222222222", Code: "SR-B", Name: "B"}
	h.grants.set(h.asha.userID, access.PermAttachmentsUpload, access.PermAttachmentsView)
	h.grants.scope[h.asha.userID] = []access.Location{srA}

	if w := h.upload(&h.asha, "a.pdf", "", pdfBytes, "?locationId="+srB.ID); w.Code != http.StatusForbidden {
		t.Errorf("upload to out-of-scope location: %d %s", w.Code, w.Body)
	}
	if w := h.upload(&h.asha, "a.pdf", "", pdfBytes, "?locationId=SR-A"); w.Code != http.StatusUnprocessableEntity {
		t.Errorf("malformed location: %d %s", w.Code, w.Body)
	}
	w := h.upload(&h.asha, "a.pdf", "", pdfBytes, "?locationId="+srA.ID)
	if w.Code != http.StatusCreated {
		t.Fatalf("upload in scope: %d %s", w.Code, w.Body)
	}
	var body struct {
		Data AttachmentResponse `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body.Data.LocationID == nil || *body.Data.LocationID != srA.ID {
		t.Fatalf("location = %v", body.Data.LocationID)
	}
	_, link := h.link(&h.asha, body.Data.ID, DispositionAttachment)

	// Moved to another location: the attachment and its issued link stop working.
	h.grants.scope[h.asha.userID] = []access.Location{srB}
	if w := h.do(http.MethodGet, "/api/v1/attachments/"+body.Data.ID, nil, "", &h.asha); w.Code != http.StatusNotFound {
		t.Errorf("out of scope metadata: %d", w.Code)
	}
	if w := h.do(http.MethodGet, link.URL, nil, "", &h.asha); w.Code != http.StatusNotFound {
		t.Errorf("out of scope download: %d", w.Code)
	}
}

func TestLinkedAttachmentsAreDeniedUntilTheirModuleChecksThem(t *testing.T) {
	h := newHarness(t)
	h.grants.set(h.asha.userID, access.PermAttachmentsUpload, access.PermAttachmentsView, access.PermAllLocations)
	att := h.uploadOK(&h.asha, "a.pdf", pdfBytes)
	row := h.repo.rows[att.ID]
	row.OwnerType, row.OwnerID = "pdi_inspection", "PDI-1"
	h.repo.rows[att.ID] = row
	if w := h.do(http.MethodGet, "/api/v1/attachments/"+att.ID, nil, "", &h.asha); w.Code != http.StatusNotFound {
		t.Errorf("linked attachment: %d", w.Code)
	}
}

func TestLimitsDescribeTheServerRules(t *testing.T) {
	h := newHarness(t)
	h.grants.set(h.asha.userID, access.PermAttachmentsUpload)
	w := h.do(http.MethodGet, "/api/v1/attachments/limits", nil, "", &h.asha)
	var body struct {
		Data LimitsResponse `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if w.Code != http.StatusOK || body.Data.MaxBytes != maxTestBytes || len(body.Data.AllowedTypes) != 4 {
		t.Errorf("limits: %d %s", w.Code, w.Body)
	}
}

func TestNoRouteChangesOrDeletesAttachments(t *testing.T) {
	h := newHarness(t)
	for _, r := range h.router.Routes() {
		if strings.Contains(r.Path, "attachments") && r.Method != http.MethodGet && !(r.Method == http.MethodPost && r.Path == "/api/v1/attachments") {
			t.Errorf("unexpected mutating route %s %s", r.Method, r.Path)
		}
	}
}

func TestSanitizeFileName(t *testing.T) {
	for in, want := range map[string]string{
		"../../etc/passwd.pdf":            "passwd.pdf",
		`C:\Users\x\bill.pdf`:             "bill.pdf",
		"a\x00b\r\n\".pdf":                "ab.pdf",
		"   ":                             "attachment",
		"..":                              "attachment",
		strings.Repeat("n", 300) + ".pdf": strings.Repeat("n", 196) + ".pdf",
	} {
		if got := sanitizeFileName(in); got != want {
			t.Errorf("sanitizeFileName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestLocalStoreRejectsUnsafeKeys(t *testing.T) {
	s, err := NewLocalStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"../x", "2026/10/../../etc", "/etc/passwd", "2026/10/not-a-uuid", ""} {
		if _, err := s.Put(context.Background(), key, strings.NewReader("x")); err == nil {
			t.Errorf("Put(%q) succeeded", key)
		}
		if _, err := s.Open(context.Background(), key); err == nil {
			t.Errorf("Open(%q) succeeded", key)
		}
	}
	key := "2026/10/0b8f5e9a-6d3c-4f21-9a7e-2c4d6e8f0a1b"
	if _, err := s.Put(context.Background(), key, strings.NewReader("x")); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Put(context.Background(), key, strings.NewReader("y")); err == nil {
		t.Error("Put overwrote an existing object")
	}
}
