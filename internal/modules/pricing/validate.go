package pricing

import (
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
)

// Request bodies. Unknown fields are refused by the HTTP layer.

// PriceListCreateRequest creates a price list for a brand.
type PriceListCreateRequest struct {
	BrandID string `json:"brandId"`
	Name    string `json:"name" example:"Showroom prices"`
} //@name PriceListCreate

// PriceListUpdateRequest renames a price list.
type PriceListUpdateRequest struct {
	Version int    `json:"version" example:"1"`
	Name    string `json:"name"`
} //@name PriceListUpdate

// VersionRequest carries the version for archive/cancel.
type VersionRequest struct {
	Version int `json:"version" example:"1"`
} //@name PriceVersion

// ItemCreateRequest drafts a price. Brand and model come from the variant.
type ItemCreateRequest struct {
	PriceListID string `json:"priceListId"`
	VariantID   string `json:"variantId"`
	PriceType   string `json:"priceType" enums:"SELLING,COST"`
	// Decimal string, e.g. "125000.00"; at most 10 digits before and 2 after the point.
	Amount        string  `json:"amount" example:"125000.00"`
	EffectiveFrom string  `json:"effectiveFrom" example:"2026-11-01T00:00:00+05:30"`
	EffectiveTo   *string `json:"effectiveTo"`
	SourceNote    *string `json:"sourceNote"`
	// Must be null: no approved tax configuration exists yet (F-10).
	TaxConfigurationID *string `json:"taxConfigurationId"`
	// Approved item this draft will replace from its effectiveFrom.
	SupersedesID *string `json:"supersedesId"`
} //@name PriceItemCreate

// ItemUpdateRequest edits a draft. Fields left out are unchanged; send
// effectiveTo or sourceNote as null to clear them.
type ItemUpdateRequest struct {
	Version       int       `json:"version" example:"1"`
	Amount        *string   `json:"amount"`
	EffectiveFrom *string   `json:"effectiveFrom"`
	EffectiveTo   OptString `json:"effectiveTo" swaggertype:"string"`
	SourceNote    OptString `json:"sourceNote" swaggertype:"string"`
} //@name PriceItemUpdate

// ApproveRequest approves a draft. The approver is the caller (never the drafter).
type ApproveRequest struct {
	Version int `json:"version" example:"1"`
	// Optional reference to the owner's approval (e.g. a signed price circular number).
	ApprovalReference *string `json:"approvalReference"`
} //@name PriceItemApprove

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

var amountPattern = regexp.MustCompile(`^\d{1,10}(\.\d{1,2})?$`)

type checker struct{ fields []FieldProblem }

func (c *checker) add(f, m string) { c.fields = append(c.fields, FieldProblem{f, m}) }

func (c *checker) err() error {
	if len(c.fields) == 0 {
		return nil
	}
	return &ValidationError{Fields: c.fields}
}

func hasControl(s string) bool { return strings.ContainsFunc(s, unicode.IsControl) }

func (c *checker) id(field string, s *string, required bool) {
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
	if utf8.RuneCountInString(v) > max || hasControl(v) {
		c.add(field, "must be at most "+strconv.Itoa(max)+" characters without control characters")
	}
}

func (c *checker) version(v int) {
	if v < 1 {
		c.add("version", "is required: send the version you last read")
	}
}

// normaliseAmount checks a positive decimal amount and returns it with two decimals.
func normaliseAmount(c *checker, raw string) string {
	v := strings.TrimSpace(raw)
	if !amountPattern.MatchString(v) {
		c.add("amount", "must be a positive amount with up to 2 decimals, e.g. 125000.00")
		return ""
	}
	whole, frac, _ := strings.Cut(v, ".")
	whole = strings.TrimLeft(whole, "0")
	if whole == "" {
		whole = "0"
	}
	frac = (frac + "00")[:2]
	if whole == "0" && frac == "00" {
		c.add("amount", "must be greater than zero")
		return ""
	}
	return whole + "." + frac
}

func parseTime(c *checker, field, raw string) time.Time {
	t, err := time.Parse(time.RFC3339, strings.TrimSpace(raw))
	if err != nil {
		c.add(field, "must be a date-time with time zone (RFC 3339), e.g. 2026-11-01T00:00:00+05:30")
	}
	return t
}

// itemInput is a validated draft.
type itemInput struct {
	PriceListID, VariantID, PriceType, Amount string
	EffectiveFrom                             time.Time
	EffectiveTo                               *time.Time
	SourceNote, SupersedesID                  *string
}

func validateItemCreate(r ItemCreateRequest) (itemInput, error) {
	var c checker
	in := itemInput{PriceListID: r.PriceListID, VariantID: r.VariantID, PriceType: r.PriceType, SourceNote: r.SourceNote}
	c.id("priceListId", &in.PriceListID, true)
	c.id("variantId", &in.VariantID, true)
	switch r.PriceType {
	case TypeSelling, TypeCost:
	case "":
		c.add("priceType", "is required")
	default:
		c.add("priceType", "must be SELLING or COST")
	}
	in.Amount = normaliseAmount(&c, r.Amount)
	if strings.TrimSpace(r.EffectiveFrom) == "" {
		c.add("effectiveFrom", "is required")
	} else {
		in.EffectiveFrom = parseTime(&c, "effectiveFrom", r.EffectiveFrom)
	}
	if r.EffectiveTo != nil && strings.TrimSpace(*r.EffectiveTo) != "" {
		t := parseTime(&c, "effectiveTo", *r.EffectiveTo)
		in.EffectiveTo = &t
	}
	if in.EffectiveTo != nil && !in.EffectiveFrom.IsZero() && !in.EffectiveTo.After(in.EffectiveFrom) {
		c.add("effectiveTo", "must be after effectiveFrom")
	}
	c.optText("sourceNote", &in.SourceNote, 500)
	if r.TaxConfigurationID != nil && strings.TrimSpace(*r.TaxConfigurationID) != "" {
		c.add("taxConfigurationId", "no approved tax configuration exists yet (accountant sign-off, F-10); leave it empty")
	}
	if r.SupersedesID != nil {
		v := *r.SupersedesID
		c.id("supersedesId", &v, false)
		if v != "" {
			in.SupersedesID = &v
		}
	}
	return in, c.err()
}

// itemPatch is a validated draft edit.
type itemPatch struct {
	Version                 int
	Amount                  *string
	EffectiveFrom           *time.Time
	EffectiveToSet, NoteSet bool
	EffectiveTo             *time.Time
	SourceNote              *string
}

func validateItemUpdate(r ItemUpdateRequest) (itemPatch, error) {
	var c checker
	c.version(r.Version)
	p := itemPatch{Version: r.Version}
	if r.Amount != nil {
		a := normaliseAmount(&c, *r.Amount)
		p.Amount = &a
	}
	if r.EffectiveFrom != nil {
		t := parseTime(&c, "effectiveFrom", *r.EffectiveFrom)
		p.EffectiveFrom = &t
	}
	if r.EffectiveTo.Set {
		p.EffectiveToSet = true
		if r.EffectiveTo.Value != nil && strings.TrimSpace(*r.EffectiveTo.Value) != "" {
			t := parseTime(&c, "effectiveTo", *r.EffectiveTo.Value)
			p.EffectiveTo = &t
		}
	}
	if r.SourceNote.Set {
		p.NoteSet = true
		p.SourceNote = r.SourceNote.Value
		c.optText("sourceNote", &p.SourceNote, 500)
	}
	if p.Amount == nil && p.EffectiveFrom == nil && !p.EffectiveToSet && !p.NoteSet && len(c.fields) == 0 {
		c.add("amount", "send at least one field to change")
	}
	return p, c.err()
}

func validateListCreate(r *PriceListCreateRequest) error {
	var c checker
	c.id("brandId", &r.BrandID, true)
	r.Name = strings.TrimSpace(r.Name)
	if r.Name == "" {
		c.add("name", "is required")
	} else if utf8.RuneCountInString(r.Name) > 120 || hasControl(r.Name) {
		c.add("name", "must be at most 120 characters")
	}
	return c.err()
}

func validateListUpdate(r *PriceListUpdateRequest) error {
	var c checker
	c.version(r.Version)
	r.Name = strings.TrimSpace(r.Name)
	if r.Name == "" {
		c.add("name", "is required")
	} else if utf8.RuneCountInString(r.Name) > 120 || hasControl(r.Name) {
		c.add("name", "must be at most 120 characters")
	}
	return c.err()
}

func validateApprove(r *ApproveRequest) error {
	var c checker
	c.version(r.Version)
	c.optText("approvalReference", &r.ApprovalReference, 200)
	return c.err()
}

// ListQuery is the raw query string of a list.
type ListQuery struct {
	BrandID, PriceListID, VariantID, PriceType, Status, Query, Page, Limit string
}

// ListFilter is a validated list query.
type ListFilter struct {
	BrandID, PriceListID, VariantID, Status, Query string
	// Types the caller may see (and asked for).
	Types       []string
	Page, Limit int
}

// Offset of the page.
func (f ListFilter) Offset() int { return (f.Page - 1) * f.Limit }

func validateList(q ListQuery) (ListFilter, string, error) {
	var c checker
	f := ListFilter{BrandID: q.BrandID, PriceListID: q.PriceListID, VariantID: q.VariantID, Status: q.Status,
		Query: strings.TrimSpace(q.Query), Page: 1, Limit: 25}
	c.id("brandId", &f.BrandID, false)
	c.id("priceListId", &f.PriceListID, false)
	c.id("variantId", &f.VariantID, false)
	switch q.PriceType {
	case "", TypeSelling, TypeCost:
	default:
		c.add("priceType", "must be SELLING or COST")
	}
	if utf8.RuneCountInString(f.Query) > 100 || hasControl(f.Query) {
		c.add("q", "must be at most 100 characters")
	}
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
		if err != nil || n < 1 || n > 100 {
			c.add("limit", "must be a whole number from 1 to 100")
		} else {
			f.Limit = n
		}
	}
	return f, q.PriceType, c.err()
}
