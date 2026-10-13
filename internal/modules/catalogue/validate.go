package catalogue

import (
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
)

// Request bodies. Unknown fields are rejected by the HTTP layer, so a client
// cannot move an item to another brand or set server-owned fields (status,
// verifier, brandId of a child) through them.

// BrandCreateRequest creates a brand. It always starts PENDING_VERIFICATION / UNVERIFIED.
type BrandCreateRequest struct {
	Name string `json:"name" example:"Brand A"`
	// Optional; derived from the name when omitted.
	Slug *string `json:"slug" example:"brand-a"`
} //@name CatalogueBrandCreate

// BrandUpdateRequest edits a brand's name or slug. Fields left out are unchanged.
type BrandUpdateRequest struct {
	Version int       `json:"version" example:"1"`
	Name    OptString `json:"name" swaggertype:"string"`
	Slug    OptString `json:"slug" swaggertype:"string"`
} //@name CatalogueBrandUpdate

// BrandAuthorisationRequest records the brand's authorisation evidence (BR-002).
// The verifier and verification time are the caller and now, never client input.
type BrandAuthorisationRequest struct {
	Version                 int     `json:"version" example:"1"`
	AuthorisationStatus     string  `json:"authorisationStatus" enums:"UNVERIFIED,AUTHORISED,NOT_AUTHORISED,REVOKED"`
	ManufacturerLegalName   *string `json:"manufacturerLegalName"`
	AuthorisationReference  *string `json:"authorisationReference"`
	AuthorisationEvidenceID *string `json:"authorisationEvidenceId"`
} //@name CatalogueBrandAuthorisation

// ModelCreateRequest creates a model under a brand.
type ModelCreateRequest struct {
	BrandID          string  `json:"brandId"`
	Name             string  `json:"name" example:"Model One"`
	InternalCode     string  `json:"internalCode" example:"M1"`
	ManufacturerCode *string `json:"manufacturerCode"`
	VehicleType      *string `json:"vehicleType"`
} //@name CatalogueModelCreate

// ModelUpdateRequest edits a pending model. The brand cannot change.
type ModelUpdateRequest struct {
	Version          int       `json:"version" example:"1"`
	Name             OptString `json:"name" swaggertype:"string"`
	InternalCode     OptString `json:"internalCode" swaggertype:"string"`
	ManufacturerCode OptString `json:"manufacturerCode" swaggertype:"string"`
	VehicleType      OptString `json:"vehicleType" swaggertype:"string"`
} //@name CatalogueModelUpdate

// VariantCreateRequest creates a variant under a model. brandId is optional;
// when sent it must be the model's brand (the server derives it either way).
type VariantCreateRequest struct {
	ModelID          string  `json:"modelId"`
	BrandID          *string `json:"brandId"`
	Name             string  `json:"name" example:"Standard"`
	VariantCode      string  `json:"variantCode" example:"STD"`
	ManufacturerCode *string `json:"manufacturerCode"`
} //@name CatalogueVariantCreate

// VariantUpdateRequest edits a pending variant. The model and brand cannot change.
type VariantUpdateRequest struct {
	Version          int       `json:"version" example:"1"`
	Name             OptString `json:"name" swaggertype:"string"`
	VariantCode      OptString `json:"variantCode" swaggertype:"string"`
	ManufacturerCode OptString `json:"manufacturerCode" swaggertype:"string"`
} //@name CatalogueVariantUpdate

// ColourCreateRequest adds a colour to a variant. brandId is optional and must match.
type ColourCreateRequest struct {
	VariantID  string  `json:"variantId"`
	BrandID    *string `json:"brandId"`
	Name       string  `json:"name" example:"Colour A"`
	ColourCode *string `json:"colourCode"`
} //@name CatalogueColourCreate

// ColourUpdateRequest edits a pending colour.
type ColourUpdateRequest struct {
	Version    int       `json:"version" example:"1"`
	Name       OptString `json:"name" swaggertype:"string"`
	ColourCode OptString `json:"colourCode" swaggertype:"string"`
} //@name CatalogueColourUpdate

// VerifyRequest records the source and verification of a pending model,
// variant or colour. The verifier is the caller.
type VerifyRequest struct {
	Version          int     `json:"version" example:"1"`
	Confidence       string  `json:"confidence" enums:"SUPPLIER_CONFIRMED,CLIENT_APPROVED"`
	SourceURL        *string `json:"sourceUrl"`
	SourceDocumentID *string `json:"sourceDocumentId"`
	SourceDate       string  `json:"sourceDate" example:"2026-10-01"`
} //@name CatalogueVerify

// ActionRequest carries the version for activate.
type ActionRequest struct {
	Version int `json:"version" example:"1"`
} //@name CatalogueAction

// ArchiveRequest carries the version and an optional reason (kept in the audit log).
type ArchiveRequest struct {
	Version int     `json:"version" example:"1"`
	Reason  *string `json:"reason"`
} //@name CatalogueArchive

var (
	slugPattern        = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	codePattern        = regexp.MustCompile(`^[A-Z0-9][A-Z0-9._-]{0,39}$`)
	vehicleTypePattern = regexp.MustCompile(`^[A-Z][A-Z0-9_]{0,39}$`)
	slugJunk           = regexp.MustCompile(`[^a-z0-9]+`)
)

type checker struct{ fields []FieldProblem }

func (c *checker) add(field, msg string) { c.fields = append(c.fields, FieldProblem{field, msg}) }

func (c *checker) err() error {
	if len(c.fields) == 0 {
		return nil
	}
	return &ValidationError{Fields: c.fields}
}

func hasControl(s string) bool {
	return strings.ContainsFunc(s, unicode.IsControl)
}

// text trims and checks a required display text.
func (c *checker) text(field string, s *string, max int) {
	*s = strings.TrimSpace(*s)
	switch {
	case *s == "":
		c.add(field, "is required")
	case utf8.RuneCountInString(*s) > max:
		c.add(field, "must be at most "+strconv.Itoa(max)+" characters")
	case hasControl(*s):
		c.add(field, "must not contain control characters")
	}
}

// optText trims an optional text; blank becomes nil.
func (c *checker) optText(field string, s **string, max int) {
	if *s == nil {
		return
	}
	v := strings.TrimSpace(**s)
	if v == "" {
		*s = nil
		return
	}
	*s = &v
	if utf8.RuneCountInString(v) > max {
		c.add(field, "must be at most "+strconv.Itoa(max)+" characters")
	} else if hasControl(v) {
		c.add(field, "must not contain control characters")
	}
}

// code normalises an upper-case code (internal/variant code).
func (c *checker) code(field string, s *string) {
	*s = strings.ToUpper(strings.TrimSpace(*s))
	if *s == "" {
		c.add(field, "is required")
	} else if !codePattern.MatchString(*s) {
		c.add(field, "must be 1-40 characters: A-Z, 0-9, '.', '_' or '-', starting with a letter or digit")
	}
}

func (c *checker) vehicleType(field string, s **string) {
	if *s == nil {
		return
	}
	v := strings.ToUpper(strings.TrimSpace(**s))
	if v == "" {
		*s = nil
		return
	}
	*s = &v
	if !vehicleTypePattern.MatchString(v) {
		c.add(field, "must be an upper-case identifier such as SCOOTER (A-Z, 0-9, '_')")
	}
}

func (c *checker) slug(field string, s *string) {
	*s = strings.TrimSpace(*s)
	if *s == "" {
		c.add(field, "is required")
	} else if !slugPattern.MatchString(*s) || len(*s) > 80 {
		c.add(field, "must be lowercase letters, digits and single hyphens (at most 80)")
	}
}

func (c *checker) uuidField(field string, s *string, required bool) {
	*s = strings.TrimSpace(*s)
	if *s == "" {
		if required {
			c.add(field, "is required")
		}
		return
	}
	if _, err := uuid.Parse(*s); err != nil {
		c.add(field, "must be a valid ID")
	}
}

func (c *checker) version(v int) {
	if v < 1 {
		c.add("version", "is required: send the version you last read")
	}
}

// required rejects a PATCH field explicitly set to null.
func (c *checker) notNull(field string, o OptString) {
	if o.Set && o.Value == nil {
		c.add(field, "cannot be cleared")
	}
}

// deriveSlug turns a name into a slug ("Brand A" → "brand-a").
func deriveSlug(name string) string {
	s := slugJunk.ReplaceAllString(strings.ToLower(name), "-")
	s = strings.Trim(s, "-")
	if len(s) > 80 {
		s = strings.Trim(s[:80], "-")
	}
	return s
}

func validateBrandCreate(r *BrandCreateRequest) error {
	var c checker
	c.text("name", &r.Name, 120)
	if r.Slug == nil || strings.TrimSpace(*r.Slug) == "" {
		s := deriveSlug(r.Name)
		r.Slug = &s
		if s == "" && r.Name != "" {
			c.add("slug", "cannot be derived from the name; send a slug")
		}
	} else {
		c.slug("slug", r.Slug)
	}
	return c.err()
}

func validateBrandUpdate(r *BrandUpdateRequest) error {
	var c checker
	c.version(r.Version)
	c.notNull("name", r.Name)
	c.notNull("slug", r.Slug)
	if r.Name.Value != nil {
		c.text("name", r.Name.Value, 120)
	}
	if r.Slug.Value != nil {
		c.slug("slug", r.Slug.Value)
	}
	if r.Name.Value == nil && r.Slug.Value == nil && len(c.fields) == 0 {
		c.add("name", "send at least one field to change")
	}
	return c.err()
}

func validateAuthorisation(r *BrandAuthorisationRequest) error {
	var c checker
	c.version(r.Version)
	switch r.AuthorisationStatus {
	case AuthUnverified, AuthAuthorised, AuthNotAuthorised, AuthRevoked:
	case "":
		c.add("authorisationStatus", "is required")
	default:
		c.add("authorisationStatus", "must be UNVERIFIED, AUTHORISED, NOT_AUTHORISED or REVOKED")
	}
	c.optText("manufacturerLegalName", &r.ManufacturerLegalName, 200)
	c.optText("authorisationReference", &r.AuthorisationReference, 200)
	if r.AuthorisationEvidenceID != nil {
		c.uuidField("authorisationEvidenceId", r.AuthorisationEvidenceID, false)
		if *r.AuthorisationEvidenceID == "" {
			r.AuthorisationEvidenceID = nil
		}
	}
	// BR-002: authorisation needs the legal identity, a reference and the evidence document.
	if r.AuthorisationStatus == AuthAuthorised {
		if r.ManufacturerLegalName == nil {
			c.add("manufacturerLegalName", "is required to record AUTHORISED")
		}
		if r.AuthorisationReference == nil {
			c.add("authorisationReference", "is required to record AUTHORISED")
		}
		if r.AuthorisationEvidenceID == nil {
			c.add("authorisationEvidenceId", "is required to record AUTHORISED")
		}
	}
	return c.err()
}

func validateModelCreate(r *ModelCreateRequest) error {
	var c checker
	c.uuidField("brandId", &r.BrandID, true)
	c.text("name", &r.Name, 120)
	c.code("internalCode", &r.InternalCode)
	c.optText("manufacturerCode", &r.ManufacturerCode, 60)
	c.vehicleType("vehicleType", &r.VehicleType)
	return c.err()
}

func validateModelUpdate(r *ModelUpdateRequest) error {
	var c checker
	c.version(r.Version)
	c.notNull("name", r.Name)
	c.notNull("internalCode", r.InternalCode)
	if r.Name.Value != nil {
		c.text("name", r.Name.Value, 120)
	}
	if r.InternalCode.Value != nil {
		c.code("internalCode", r.InternalCode.Value)
	}
	c.optText("manufacturerCode", &r.ManufacturerCode.Value, 60)
	c.vehicleType("vehicleType", &r.VehicleType.Value)
	if !r.Name.Set && !r.InternalCode.Set && !r.ManufacturerCode.Set && !r.VehicleType.Set {
		c.add("name", "send at least one field to change")
	}
	return c.err()
}

func validateVariantCreate(r *VariantCreateRequest) error {
	var c checker
	c.uuidField("modelId", &r.ModelID, true)
	if r.BrandID != nil {
		c.uuidField("brandId", r.BrandID, false)
		if *r.BrandID == "" {
			r.BrandID = nil
		}
	}
	c.text("name", &r.Name, 120)
	c.code("variantCode", &r.VariantCode)
	c.optText("manufacturerCode", &r.ManufacturerCode, 60)
	return c.err()
}

func validateVariantUpdate(r *VariantUpdateRequest) error {
	var c checker
	c.version(r.Version)
	c.notNull("name", r.Name)
	c.notNull("variantCode", r.VariantCode)
	if r.Name.Value != nil {
		c.text("name", r.Name.Value, 120)
	}
	if r.VariantCode.Value != nil {
		c.code("variantCode", r.VariantCode.Value)
	}
	c.optText("manufacturerCode", &r.ManufacturerCode.Value, 60)
	if !r.Name.Set && !r.VariantCode.Set && !r.ManufacturerCode.Set {
		c.add("name", "send at least one field to change")
	}
	return c.err()
}

func validateColourCreate(r *ColourCreateRequest) error {
	var c checker
	c.uuidField("variantId", &r.VariantID, true)
	if r.BrandID != nil {
		c.uuidField("brandId", r.BrandID, false)
		if *r.BrandID == "" {
			r.BrandID = nil
		}
	}
	c.text("name", &r.Name, 80)
	c.optText("colourCode", &r.ColourCode, 40)
	return c.err()
}

func validateColourUpdate(r *ColourUpdateRequest) error {
	var c checker
	c.version(r.Version)
	c.notNull("name", r.Name)
	if r.Name.Value != nil {
		c.text("name", r.Name.Value, 80)
	}
	c.optText("colourCode", &r.ColourCode.Value, 40)
	if !r.Name.Set && !r.ColourCode.Set {
		c.add("name", "send at least one field to change")
	}
	return c.err()
}

// validateVerify checks a verification record. now is the server clock (the
// source date may not be in the future).
func validateVerify(r *VerifyRequest, now time.Time) error {
	var c checker
	c.version(r.Version)
	switch r.Confidence {
	case ConfidenceSupplierConfirmed, ConfidenceClientApproved:
	case "":
		c.add("confidence", "is required")
	default:
		c.add("confidence", "must be SUPPLIER_CONFIRMED or CLIENT_APPROVED")
	}
	if r.SourceURL != nil {
		v := strings.TrimSpace(*r.SourceURL)
		if v == "" {
			r.SourceURL = nil
		} else {
			r.SourceURL = &v
			u, err := url.Parse(v)
			if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || len(v) > 2000 || hasControl(v) {
				c.add("sourceUrl", "must be an http or https URL (at most 2000 characters)")
			}
		}
	}
	if r.SourceDocumentID != nil {
		c.uuidField("sourceDocumentId", r.SourceDocumentID, false)
		if *r.SourceDocumentID == "" {
			r.SourceDocumentID = nil
		}
	}
	if r.SourceURL == nil && r.SourceDocumentID == nil {
		c.add("sourceUrl", "send a source URL or a source document")
	}
	r.SourceDate = strings.TrimSpace(r.SourceDate)
	if r.SourceDate == "" {
		c.add("sourceDate", "is required")
	} else if d, err := time.Parse(time.DateOnly, r.SourceDate); err != nil {
		c.add("sourceDate", "must be a date (YYYY-MM-DD)")
	} else if d.After(now.UTC().Add(14 * time.Hour)) {
		// A date counts as future only once it has not begun anywhere yet
		// (UTC+14), so "today" in India is accepted just after midnight.
		c.add("sourceDate", "cannot be in the future")
	}
	return c.err()
}

func validateAction(version int) error {
	var c checker
	c.version(version)
	return c.err()
}

func validateArchive(r *ArchiveRequest) error {
	var c checker
	c.version(r.Version)
	c.optText("reason", &r.Reason, 500)
	return c.err()
}

// List query limits.
const (
	defaultLimit = 25
	maxLimit     = 100
)

// ListQuery is the raw list query string.
type ListQuery struct {
	Query, Status, Confidence, AuthStatus, BrandID, ModelID, VariantID, Page, Limit string
}

func validateList(q ListQuery) (ListFilter, error) {
	var c checker
	f := ListFilter{
		Query: strings.TrimSpace(q.Query), Status: q.Status, Confidence: q.Confidence, AuthStatus: q.AuthStatus,
		BrandID: q.BrandID, ModelID: q.ModelID, VariantID: q.VariantID, Page: 1, Limit: defaultLimit,
	}
	if utf8.RuneCountInString(f.Query) > 100 || hasControl(f.Query) {
		c.add("q", "must be at most 100 characters")
	}
	switch f.Status {
	case "", StatusPending, StatusActive, StatusArchived:
	default:
		c.add("status", "must be PENDING_VERIFICATION, ACTIVE or ARCHIVED")
	}
	switch f.Confidence {
	case "", ConfidenceUnverified, ConfidenceSupplierConfirmed, ConfidenceClientApproved:
	default:
		c.add("confidence", "must be UNVERIFIED, SUPPLIER_CONFIRMED or CLIENT_APPROVED")
	}
	switch f.AuthStatus {
	case "", AuthUnverified, AuthAuthorised, AuthNotAuthorised, AuthRevoked:
	default:
		c.add("authorisationStatus", "must be UNVERIFIED, AUTHORISED, NOT_AUTHORISED or REVOKED")
	}
	c.uuidField("brandId", &f.BrandID, false)
	c.uuidField("modelId", &f.ModelID, false)
	c.uuidField("variantId", &f.VariantID, false)
	if q.Page != "" {
		n, err := strconv.Atoi(q.Page)
		if err != nil || n < 1 || n > 100000 {
			c.add("page", "must be a whole number from 1")
		} else {
			f.Page = n
		}
	}
	if q.Limit != "" {
		n, err := strconv.Atoi(q.Limit)
		if err != nil || n < 1 || n > maxLimit {
			c.add("limit", "must be a whole number from 1 to "+strconv.Itoa(maxLimit))
		} else {
			f.Limit = n
		}
	}
	return f, c.err()
}
