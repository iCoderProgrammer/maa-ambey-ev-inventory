package purchasing

import (
	"math/big"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
)

// MaxLines is the most lines one PO may have.
const MaxLines = 100

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

// LineInput is one line of a new PO, or a line added to a draft. The agreed
// unit cost comes from the supplier agreement; priceListId (optional) names a
// cost price list of the line's brand whose approved COST price is snapshotted
// as the reference (needs catalogue.cost_price.view).
type LineInput struct {
	BrandID     string  `json:"brandId"`
	ModelID     string  `json:"modelId"`
	VariantID   string  `json:"variantId"`
	Quantity    int     `json:"quantity" example:"1"`
	UnitCost    string  `json:"unitCost" example:"1000.00"`
	PriceListID *string `json:"priceListId"`
	Note        *string `json:"note"`
} //@name PurchaseOrderLineInput

// CreateInput creates a draft PO. Lines are optional here; submission needs one.
type CreateInput struct {
	SupplierID   string      `json:"supplierId"`
	ExpectedDate *string     `json:"expectedDate" example:"2026-11-15"`
	Notes        *string     `json:"notes"`
	Lines        []LineInput `json:"lines"`
} //@name PurchaseOrderCreate

// HeaderPatch edits a draft. null clears an optional field.
type HeaderPatch struct {
	Version      int       `json:"version" example:"1"`
	SupplierID   *string   `json:"supplierId"`
	ExpectedDate OptString `json:"expectedDate" swaggertype:"string"`
	Notes        OptString `json:"notes" swaggertype:"string"`
} //@name PurchaseOrderUpdate

// LineAddInput adds a line to a draft (version is the PO's).
type LineAddInput struct {
	Version     int     `json:"version" example:"1"`
	BrandID     string  `json:"brandId"`
	ModelID     string  `json:"modelId"`
	VariantID   string  `json:"variantId"`
	Quantity    int     `json:"quantity" example:"1"`
	UnitCost    string  `json:"unitCost" example:"1000.00"`
	PriceListID *string `json:"priceListId"`
	Note        *string `json:"note"`
} //@name PurchaseOrderLineAdd

// LinePatch edits a draft line (version is the PO's). Setting priceListId takes
// a fresh snapshot; null removes the reference.
type LinePatch struct {
	Version     int       `json:"version" example:"1"`
	Quantity    *int      `json:"quantity"`
	UnitCost    *string   `json:"unitCost"`
	PriceListID OptString `json:"priceListId" swaggertype:"string"`
	Note        OptString `json:"note" swaggertype:"string"`
} //@name PurchaseOrderLineUpdate

// VersionInput carries the PO version.
type VersionInput struct {
	Version int `json:"version" example:"1"`
} //@name PurchaseOrderVersion

// ApproveInput approves a submitted PO.
type ApproveInput struct {
	Version int     `json:"version" example:"2"`
	Note    *string `json:"note"`
} //@name PurchaseOrderApprove

// ReasonInput returns a PO to draft or cancels it.
type ReasonInput struct {
	Version int    `json:"version" example:"2"`
	Reason  string `json:"reason" example:"Quantities to be confirmed with the supplier"`
} //@name PurchaseOrderReason

// SettingsInput changes the approval settings. A threshold needs the owner's
// sign-off reference; null threshold means every PO needs an approver.
type SettingsInput struct {
	Version                        int     `json:"version" example:"1"`
	ApprovalThreshold              *string `json:"approvalThreshold" example:"100000.00"`
	FirstTimeSupplierNeedsApproval bool    `json:"firstTimeSupplierNeedsApproval"`
	ApprovalReference              *string `json:"approvalReference" example:"Owner sign-off 2026-10-20"`
} //@name PurchaseOrderSettingsUpdate

var (
	amountPattern      = regexp.MustCompile(`^\d{1,10}(\.\d{1,2})?$`)
	idempotencyPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{8,100}$`)
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
	if utf8.RuneCountInString(v) > max || hasControl(strings.ReplaceAll(v, "\n", "")) {
		c.add(field, "must be at most "+strconv.Itoa(max)+" characters")
	}
}

func (c *checker) required(field string, s *string, max int) {
	*s = strings.TrimSpace(*s)
	switch {
	case *s == "":
		c.add(field, "is required")
	case utf8.RuneCountInString(*s) > max || hasControl(strings.ReplaceAll(*s, "\n", "")):
		c.add(field, "must be at most "+strconv.Itoa(max)+" characters")
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

func (c *checker) optionalID(field string, s **string) {
	if *s == nil {
		return
	}
	v := strings.TrimSpace(**s)
	if v == "" {
		*s = nil
		return
	}
	*s = &v
	if _, err := uuid.Parse(v); err != nil {
		c.add(field, "must be a valid ID")
	}
}

func (c *checker) date(field string, s **string) {
	if *s == nil {
		return
	}
	v := strings.TrimSpace(**s)
	if v == "" {
		*s = nil
		return
	}
	*s = &v
	if _, err := time.Parse(time.DateOnly, v); err != nil {
		c.add(field, "must be a date (YYYY-MM-DD)")
	}
}

func (c *checker) quantity(field string, q int) {
	if q < 1 || q > 10000 {
		c.add(field, "must be a whole number from 1 to 10000")
	}
}

// amount checks a positive decimal amount and returns it with two decimals.
func (c *checker) amount(field, raw string) string {
	v := strings.TrimSpace(raw)
	if !amountPattern.MatchString(v) {
		c.add(field, "must be a positive amount with up to 2 decimals, e.g. 1000.00")
		return v
	}
	r, ok := new(big.Rat).SetString(v)
	if !ok || r.Sign() <= 0 {
		c.add(field, "must be greater than zero")
		return v
	}
	return r.FloatString(2)
}

func (c *checker) line(prefix string, in *LineInput) {
	c.id(prefix+"brandId", &in.BrandID)
	c.id(prefix+"modelId", &in.ModelID)
	c.id(prefix+"variantId", &in.VariantID)
	c.quantity(prefix+"quantity", in.Quantity)
	in.UnitCost = c.amount(prefix+"unitCost", in.UnitCost)
	c.optionalID(prefix+"priceListId", &in.PriceListID)
	c.optional(prefix+"note", &in.Note, 300)
}

func validateCreate(in *CreateInput) error {
	var c checker
	c.id("supplierId", &in.SupplierID)
	c.date("expectedDate", &in.ExpectedDate)
	c.optional("notes", &in.Notes, 1000)
	if len(in.Lines) > MaxLines {
		c.add("lines", "must have at most "+strconv.Itoa(MaxLines)+" lines")
		return c.err()
	}
	seen := map[string]bool{}
	for i := range in.Lines {
		p := "lines[" + strconv.Itoa(i) + "]."
		c.line(p, &in.Lines[i])
		if v := in.Lines[i].VariantID; v != "" {
			if seen[v] {
				c.add(p+"variantId", "is already on another line; change that line's quantity instead")
			}
			seen[v] = true
		}
	}
	return c.err()
}

func validateHeaderPatch(p *HeaderPatch) error {
	var c checker
	c.version(p.Version)
	if p.SupplierID != nil {
		c.id("supplierId", p.SupplierID)
	}
	c.date("expectedDate", &p.ExpectedDate.Value)
	c.optional("notes", &p.Notes.Value, 1000)
	if p.SupplierID == nil && !p.ExpectedDate.Set && !p.Notes.Set && len(c.fields) == 0 {
		c.add("supplierId", "send at least one field to change")
	}
	return c.err()
}

func validateLineAdd(in *LineAddInput) (LineInput, error) {
	var c checker
	c.version(in.Version)
	l := LineInput{BrandID: in.BrandID, ModelID: in.ModelID, VariantID: in.VariantID, Quantity: in.Quantity,
		UnitCost: in.UnitCost, PriceListID: in.PriceListID, Note: in.Note}
	c.line("", &l)
	return l, c.err()
}

func validateLinePatch(p *LinePatch) error {
	var c checker
	c.version(p.Version)
	if p.Quantity != nil {
		c.quantity("quantity", *p.Quantity)
	}
	if p.UnitCost != nil {
		v := c.amount("unitCost", *p.UnitCost)
		p.UnitCost = &v
	}
	c.optionalID("priceListId", &p.PriceListID.Value)
	c.optional("note", &p.Note.Value, 300)
	if p.Quantity == nil && p.UnitCost == nil && !p.PriceListID.Set && !p.Note.Set && len(c.fields) == 0 {
		c.add("quantity", "send at least one field to change")
	}
	return c.err()
}

func validateApprove(in *ApproveInput) error {
	var c checker
	c.version(in.Version)
	c.optional("note", &in.Note, 500)
	return c.err()
}

func validateReason(in *ReasonInput) error {
	var c checker
	c.version(in.Version)
	c.required("reason", &in.Reason, 500)
	return c.err()
}

func validateSettings(in *SettingsInput) error {
	var c checker
	c.version(in.Version)
	if in.ApprovalThreshold != nil && strings.TrimSpace(*in.ApprovalThreshold) == "" {
		in.ApprovalThreshold = nil
	}
	if in.ApprovalThreshold != nil {
		v := c.amount("approvalThreshold", *in.ApprovalThreshold)
		in.ApprovalThreshold = &v
	}
	c.optional("approvalReference", &in.ApprovalReference, 200)
	if in.ApprovalThreshold != nil && in.ApprovalReference == nil {
		c.add("approvalReference", "is required with a threshold: record the owner's sign-off")
	}
	return c.err()
}

func validateIdempotencyKey(k string) (string, error) {
	k = strings.TrimSpace(k)
	if k == "" {
		return "", nil
	}
	if !idempotencyPattern.MatchString(k) {
		return "", invalid("Idempotency-Key", "must be 8-100 letters, digits, '-' or '_'")
	}
	return k, nil
}

// ListQuery is the raw list query.
type ListQuery struct{ Query, Status, SupplierID, BrandID, Page, Limit string }

// ListFilter is a validated list query.
type ListFilter struct {
	Query, Status, SupplierID, BrandID string
	Page, Limit                        int
}

func validateList(q ListQuery) (ListFilter, error) {
	var c checker
	f := ListFilter{Query: strings.TrimSpace(q.Query), Status: q.Status, SupplierID: strings.TrimSpace(q.SupplierID),
		BrandID: strings.TrimSpace(q.BrandID), Page: 1, Limit: 25}
	if utf8.RuneCountInString(f.Query) > 100 || hasControl(f.Query) {
		c.add("q", "must be at most 100 characters")
	}
	if f.Status != "" && !slices.Contains(Statuses, f.Status) {
		c.add("status", "must be one of "+strings.Join(Statuses, ", "))
	}
	for _, p := range []struct{ field, v string }{{"supplierId", f.SupplierID}, {"brandId", f.BrandID}} {
		if p.v != "" {
			if _, err := uuid.Parse(p.v); err != nil {
				c.add(p.field, "must be a valid ID")
			}
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
