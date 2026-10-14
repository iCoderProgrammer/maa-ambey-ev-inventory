package suppliers

import (
	"net/mail"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
)

// OptString distinguishes an absent PATCH field from an explicit null.
type OptString struct {
	Set   bool
	Value *string
}

// UnmarshalJSON implements json.Unmarshaler.
func (o *OptString) UnmarshalJSON(b []byte) error {
	o.Set = true
	if string(b) == "null" {
		o.Value = nil
		return nil
	}
	s, err := strconv.Unquote(string(b))
	if err != nil {
		return err
	}
	o.Value = &s
	return nil
}

// SupplierInput creates a supplier (always PENDING_VERIFICATION).
type SupplierInput struct {
	LegalName    string  `json:"legalName"`
	TradeName    *string `json:"tradeName"`
	GSTIN        *string `json:"gstin"`
	AddressLine1 *string `json:"addressLine1"`
	AddressLine2 *string `json:"addressLine2"`
	City         *string `json:"city"`
	State        *string `json:"state"`
	PostalCode   *string `json:"postalCode"`
	Notes        *string `json:"notes"`
} //@name SupplierCreate

// SupplierPatch edits a supplier. Changing the legal name or GSTIN of a
// verified supplier returns it to PENDING_VERIFICATION (the verification
// covered the old identity). null clears an optional field.
type SupplierPatch struct {
	Version      int       `json:"version" example:"1"`
	LegalName    OptString `json:"legalName" swaggertype:"string"`
	TradeName    OptString `json:"tradeName" swaggertype:"string"`
	GSTIN        OptString `json:"gstin" swaggertype:"string"`
	AddressLine1 OptString `json:"addressLine1" swaggertype:"string"`
	AddressLine2 OptString `json:"addressLine2" swaggertype:"string"`
	City         OptString `json:"city" swaggertype:"string"`
	State        OptString `json:"state" swaggertype:"string"`
	PostalCode   OptString `json:"postalCode" swaggertype:"string"`
	Notes        OptString `json:"notes" swaggertype:"string"`
} //@name SupplierUpdate

// VerifyInput verifies a supplier with a reference and one of its documents.
type VerifyInput struct {
	Version    int    `json:"version" example:"1"`
	Reference  string `json:"reference" example:"GST certificate checked"`
	DocumentID string `json:"documentId"`
} //@name SupplierVerify

// ArchiveInput archives a supplier or ends a relationship.
type ArchiveInput struct {
	Version int     `json:"version" example:"1"`
	Reason  *string `json:"reason"`
} //@name SupplierArchive

// ContactInput adds a contact (phone or email required).
type ContactInput struct {
	Name      string  `json:"name"`
	Role      *string `json:"role"`
	Phone     *string `json:"phone"`
	Email     *string `json:"email"`
	IsPrimary bool    `json:"isPrimary"`
} //@name SupplierContactCreate

// ContactPatch edits a contact.
type ContactPatch struct {
	Version   int       `json:"version" example:"1"`
	Name      OptString `json:"name" swaggertype:"string"`
	Role      OptString `json:"role" swaggertype:"string"`
	Phone     OptString `json:"phone" swaggertype:"string"`
	Email     OptString `json:"email" swaggertype:"string"`
	IsPrimary *bool     `json:"isPrimary"`
} //@name SupplierContactUpdate

// VersionInput carries a version.
type VersionInput struct {
	Version int `json:"version" example:"1"`
} //@name SupplierVersion

// DocumentInput links one of the caller's private uploads to the supplier.
type DocumentInput struct {
	AttachmentID string  `json:"attachmentId"`
	DocumentType string  `json:"documentType" enums:"GST_REGISTRATION,AUTHORISATION,AGREEMENT,OTHER"`
	Note         *string `json:"note"`
} //@name SupplierDocumentAdd

// LinkInput proposes a relationship with a brand (unverified).
type LinkInput struct {
	BrandID string  `json:"brandId"`
	Note    *string `json:"note"`
} //@name SupplierBrandLinkCreate

// LinkVerifyInput verifies a relationship. The evidence must be a document of
// the same supplier; the verifier is the caller.
type LinkVerifyInput struct {
	Version                int     `json:"version" example:"1"`
	AuthorisationReference string  `json:"authorisationReference" example:"DLR-2026-001"`
	EvidenceDocumentID     string  `json:"evidenceDocumentId"`
	EffectiveFrom          string  `json:"effectiveFrom" example:"2026-10-01"`
	EffectiveTo            *string `json:"effectiveTo"`
} //@name SupplierBrandLinkVerify

// LinkEndInput ends a relationship (kept as history).
type LinkEndInput struct {
	Version     int     `json:"version" example:"1"`
	Reason      string  `json:"reason" example:"Agreement expired"`
	EffectiveTo *string `json:"effectiveTo"`
} //@name SupplierBrandLinkEnd

var (
	gstinPattern  = regexp.MustCompile(`^[0-9]{2}[A-Z]{5}[0-9]{4}[A-Z][1-9A-Z]Z[0-9A-Z]$`)
	postalPattern = regexp.MustCompile(`^[1-9][0-9]{5}$`)
	phonePattern  = regexp.MustCompile(`^\+?[0-9][0-9 -]{5,18}[0-9]$`)
)

type checker struct{ fields []FieldProblem }

func (c *checker) add(f, m string) { c.fields = append(c.fields, FieldProblem{f, m}) }

func (c *checker) err() error {
	if len(c.fields) == 0 {
		return nil
	}
	return &ValidationError{Fields: c.fields}
}

func hasControl(s string) bool { return strings.ContainsFunc(s, unicode.IsControl) }

func (c *checker) required(field string, s *string, max int) {
	*s = strings.TrimSpace(*s)
	switch {
	case *s == "":
		c.add(field, "is required")
	case utf8.RuneCountInString(*s) > max || hasControl(*s):
		c.add(field, "must be at most "+strconv.Itoa(max)+" characters")
	}
}

func (c *checker) optional(field string, s **string, max int) {
	if *s == nil {
		return
	}
	v := strings.TrimSpace(**s)
	if v == "" {
		*s = nil
		return
	}
	*s = &v
	if utf8.RuneCountInString(v) > max || hasControl(v) {
		c.add(field, "must be at most "+strconv.Itoa(max)+" characters")
	}
}

func (c *checker) gstin(s **string) {
	if *s == nil {
		return
	}
	v := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(**s), " ", ""))
	if v == "" {
		*s = nil
		return
	}
	*s = &v
	if !gstinPattern.MatchString(v) {
		c.add("gstin", "must be a 15-character GSTIN (e.g. 2 digits, 10-character PAN, entity number, Z, check character)")
	}
}

func (c *checker) postal(s **string) {
	c.optional("postalCode", s, 6)
	if *s != nil && !postalPattern.MatchString(**s) {
		c.add("postalCode", "must be a 6-digit PIN code")
	}
}

func (c *checker) phone(s **string) {
	c.optional("phone", s, 20)
	if *s != nil && !phonePattern.MatchString(**s) {
		c.add("phone", "must be a phone number of 7-20 digits (spaces, hyphens and a leading + allowed)")
	}
}

func (c *checker) email(s **string) {
	c.optional("email", s, 254)
	if *s != nil {
		a, err := mail.ParseAddress(**s)
		if err != nil || a.Address != **s {
			c.add("email", "must be an email address")
		}
	}
}

func (c *checker) version(v int) {
	if v < 1 {
		c.add("version", "is required: send the version you last read")
	}
}

func (c *checker) id(field string, s *string) {
	*s = strings.TrimSpace(*s)
	if *s == "" {
		c.add(field, "is required")
	} else if _, err := uuid.Parse(*s); err != nil {
		c.add(field, "must be a valid ID")
	}
}

func parseDate(c *checker, field, v string) (time.Time, bool) {
	d, err := time.Parse(time.DateOnly, strings.TrimSpace(v))
	if err != nil {
		c.add(field, "must be a date (YYYY-MM-DD)")
		return time.Time{}, false
	}
	return d, true
}

func validateSupplier(in *SupplierInput) error {
	var c checker
	c.required("legalName", &in.LegalName, 200)
	c.optional("tradeName", &in.TradeName, 200)
	c.gstin(&in.GSTIN)
	c.optional("addressLine1", &in.AddressLine1, 200)
	c.optional("addressLine2", &in.AddressLine2, 200)
	c.optional("city", &in.City, 100)
	c.optional("state", &in.State, 100)
	c.postal(&in.PostalCode)
	c.optional("notes", &in.Notes, 1000)
	return c.err()
}

func validateSupplierPatch(p *SupplierPatch) error {
	var c checker
	c.version(p.Version)
	if p.LegalName.Set {
		if p.LegalName.Value == nil {
			c.add("legalName", "cannot be cleared")
		} else {
			c.required("legalName", p.LegalName.Value, 200)
		}
	}
	c.optional("tradeName", &p.TradeName.Value, 200)
	c.gstin(&p.GSTIN.Value)
	c.optional("addressLine1", &p.AddressLine1.Value, 200)
	c.optional("addressLine2", &p.AddressLine2.Value, 200)
	c.optional("city", &p.City.Value, 100)
	c.optional("state", &p.State.Value, 100)
	c.postal(&p.PostalCode.Value)
	c.optional("notes", &p.Notes.Value, 1000)
	changed := p.LegalName.Set || p.TradeName.Set || p.GSTIN.Set || p.AddressLine1.Set || p.AddressLine2.Set ||
		p.City.Set || p.State.Set || p.PostalCode.Set || p.Notes.Set
	if !changed && len(c.fields) == 0 {
		c.add("legalName", "send at least one field to change")
	}
	return c.err()
}

func validateVerify(in *VerifyInput) error {
	var c checker
	c.version(in.Version)
	c.required("reference", &in.Reference, 200)
	c.id("documentId", &in.DocumentID)
	return c.err()
}

func validateArchive(in *ArchiveInput) error {
	var c checker
	c.version(in.Version)
	c.optional("reason", &in.Reason, 300)
	return c.err()
}

func validateContact(in *ContactInput) error {
	var c checker
	c.required("name", &in.Name, 120)
	c.optional("role", &in.Role, 80)
	c.phone(&in.Phone)
	c.email(&in.Email)
	if in.Phone == nil && in.Email == nil && len(c.fields) == 0 {
		c.add("phone", "give a phone number or an email address")
	}
	return c.err()
}

func validateContactPatch(p *ContactPatch) error {
	var c checker
	c.version(p.Version)
	if p.Name.Set {
		if p.Name.Value == nil {
			c.add("name", "cannot be cleared")
		} else {
			c.required("name", p.Name.Value, 120)
		}
	}
	c.optional("role", &p.Role.Value, 80)
	c.phone(&p.Phone.Value)
	c.email(&p.Email.Value)
	if !p.Name.Set && !p.Role.Set && !p.Phone.Set && !p.Email.Set && p.IsPrimary == nil && len(c.fields) == 0 {
		c.add("name", "send at least one field to change")
	}
	return c.err()
}

func validateDocument(in *DocumentInput) error {
	var c checker
	c.id("attachmentId", &in.AttachmentID)
	if !slices.Contains(DocumentTypes, in.DocumentType) {
		c.add("documentType", "must be one of "+strings.Join(DocumentTypes, ", "))
	}
	c.optional("note", &in.Note, 300)
	return c.err()
}

func validateLink(in *LinkInput) error {
	var c checker
	c.id("brandId", &in.BrandID)
	c.optional("note", &in.Note, 300)
	return c.err()
}

func validateLinkVerify(in *LinkVerifyInput) error {
	var c checker
	c.version(in.Version)
	c.required("authorisationReference", &in.AuthorisationReference, 200)
	c.id("evidenceDocumentId", &in.EvidenceDocumentID)
	from, okFrom := parseDate(&c, "effectiveFrom", in.EffectiveFrom)
	if in.EffectiveTo != nil && strings.TrimSpace(*in.EffectiveTo) == "" {
		in.EffectiveTo = nil
	}
	if in.EffectiveTo != nil {
		to, ok := parseDate(&c, "effectiveTo", *in.EffectiveTo)
		if ok && okFrom && to.Before(from) {
			c.add("effectiveTo", "must not be before effectiveFrom")
		}
	}
	return c.err()
}

func validateLinkEnd(in *LinkEndInput) error {
	var c checker
	c.version(in.Version)
	c.required("reason", &in.Reason, 300)
	if in.EffectiveTo != nil && strings.TrimSpace(*in.EffectiveTo) == "" {
		in.EffectiveTo = nil
	}
	if in.EffectiveTo != nil {
		parseDate(&c, "effectiveTo", *in.EffectiveTo)
	}
	return c.err()
}

// ListQuery is the raw list query.
type ListQuery struct{ Query, Status, BrandID, Page, Limit string }

// ListFilter is a validated list query.
type ListFilter struct {
	Query, Status, BrandID string
	Page, Limit            int
}

func validateList(q ListQuery) (ListFilter, error) {
	var c checker
	f := ListFilter{Query: strings.TrimSpace(q.Query), Status: q.Status, BrandID: strings.TrimSpace(q.BrandID), Page: 1, Limit: 25}
	if utf8.RuneCountInString(f.Query) > 100 || hasControl(f.Query) {
		c.add("q", "must be at most 100 characters")
	}
	switch f.Status {
	case "", StatusPending, StatusVerified, StatusArchived:
	default:
		c.add("status", "must be PENDING_VERIFICATION, VERIFIED or ARCHIVED")
	}
	if f.BrandID != "" {
		if _, err := uuid.Parse(f.BrandID); err != nil {
			c.add("brandId", "must be a valid ID")
		}
	}
	if q.Page != "" {
		if n, err := strconv.Atoi(q.Page); err != nil || n < 1 || n > 100000 {
			c.add("page", "must be a whole number from 1")
		} else {
			f.Page = n
		}
	}
	if q.Limit != "" {
		if n, err := strconv.Atoi(q.Limit); err != nil || n < 1 || n > 100 {
			c.add("limit", "must be a whole number from 1 to 100")
		} else {
			f.Limit = n
		}
	}
	return f, c.err()
}

// LinkQuery filters relationships.
type LinkQuery struct{ BrandID, SupplierID, Status string }

func validateLinkQuery(q *LinkQuery) error {
	var c checker
	for _, p := range []struct {
		field string
		v     *string
	}{{"brandId", &q.BrandID}, {"supplierId", &q.SupplierID}} {
		*p.v = strings.TrimSpace(*p.v)
		if *p.v != "" {
			if _, err := uuid.Parse(*p.v); err != nil {
				c.add(p.field, "must be a valid ID")
			}
		}
	}
	switch q.Status {
	case "", LinkUnverified, LinkVerified, LinkEnded:
	default:
		c.add("status", "must be UNVERIFIED, VERIFIED or ENDED")
	}
	if q.BrandID == "" && q.SupplierID == "" && len(c.fields) == 0 {
		c.add("brandId", "give brandId or supplierId")
	}
	return c.err()
}
