package attachments

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strconv"
	"strings"
	"time"
)

// Download link errors.
var (
	ErrLinkInvalid = errors.New("download link is invalid")
	ErrLinkExpired = errors.New("download link has expired")
)

// Dispositions a download link may ask for.
const (
	DispositionAttachment = "attachment"
	DispositionInline     = "inline"
)

// Signer issues and checks short-lived download links. A link is bound to the
// attachment, the user it was issued to, the disposition and an expiry, so it
// cannot be reused by someone else or after it expires. Every download still
// needs the user's session and a fresh permission check.
type Signer struct {
	key []byte
	ttl time.Duration
	now func() time.Time
}

// NewSigner uses key (at least 32 bytes); an empty key makes a random one,
// which is only acceptable where config allows it (development/test).
func NewSigner(key string, ttl time.Duration) (*Signer, error) {
	k := []byte(key)
	if len(k) == 0 {
		k = make([]byte, 32)
		if _, err := rand.Read(k); err != nil {
			return nil, err
		}
	}
	if len(k) < 32 {
		return nil, errors.New("attachment signing key must be at least 32 bytes")
	}
	return &Signer{key: k, ttl: ttl, now: time.Now}, nil
}

// Link is a signed download link.
type Link struct {
	Expires   time.Time
	Signature string
}

// Sign issues a link valid for the configured TTL.
func (s *Signer) Sign(attachmentID, userID, disposition string) Link {
	exp := s.now().Add(s.ttl).Truncate(time.Second)
	return Link{Expires: exp, Signature: s.mac(attachmentID, userID, disposition, exp.Unix())}
}

// Verify checks a link presented by userID.
func (s *Signer) Verify(attachmentID, userID, disposition, expires, signature string) error {
	exp, err := strconv.ParseInt(expires, 10, 64)
	if err != nil || signature == "" {
		return ErrLinkInvalid
	}
	want := s.mac(attachmentID, userID, disposition, exp)
	if !hmac.Equal([]byte(want), []byte(signature)) {
		return ErrLinkInvalid
	}
	if !s.now().Before(time.Unix(exp, 0)) {
		return ErrLinkExpired
	}
	// A valid signature with an expiry beyond the TTL cannot come from Sign.
	if time.Unix(exp, 0).After(s.now().Add(s.ttl + time.Second)) {
		return ErrLinkInvalid
	}
	return nil
}

func (s *Signer) mac(attachmentID, userID, disposition string, exp int64) string {
	m := hmac.New(sha256.New, s.key)
	m.Write([]byte(strings.Join([]string{"attachment-download-v1", attachmentID, userID, disposition, strconv.FormatInt(exp, 10)}, "\n")))
	return base64.RawURLEncoding.EncodeToString(m.Sum(nil))
}
