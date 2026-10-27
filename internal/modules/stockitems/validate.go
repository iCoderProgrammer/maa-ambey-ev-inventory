package stockitems

import (
	"regexp"
	"strconv"
	"strings"
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

// CategoryInput creates a category (ACTIVE).
type CategoryInput struct {
	Name        string  `json:"name" example:"Brake parts"`
	Description *string `json:"description"`
} //@name StockItemCategoryCreate

// CategoryPatch edits a category. null clears the description.
type CategoryPatch struct {
	Version     int       `json:"version" example:"1"`
	Name        OptString `json:"name" swaggertype:"string"`
	Description OptString `json:"description" swaggertype:"string"`
} //@name StockItemCategoryUpdate

// ArchiveInput archives a category or SKU (kept as history; no reactivation yet).
type ArchiveInput struct {
	Version int    `json:"version" example:"1"`
	Reason  string `json:"reason" example:"No longer stocked"`
} //@name StockItemArchive

// SupplierLinkInput names a supplier of the item.
type SupplierLinkInput struct {
	SupplierID         string  `json:"supplierId"`
	SupplierPartNumber *string `json:"supplierPartNumber"`
} //@name StockItemSupplierInput

// CompatibilityInput is a brand (modelId omitted or null: every model) or one model of it.
type CompatibilityInput struct {
	BrandID string  `json:"brandId"`
	ModelID *string `json:"modelId"`
} //@name StockItemCompatibilityInput

// ItemInput creates a SKU. The SKU code and tracking mode cannot change later.
type ItemInput struct {
	SKU          string  `json:"sku" example:"BRK-PAD-01"`
	Name         string  `json:"name" example:"Brake pad set"`
	Description  *string `json:"description"`
	CategoryID   string  `json:"categoryId"`
	UOM          string  `json:"uom" example:"PCS"`
	TrackingMode string  `json:"trackingMode" enums:"QUANTITY,SERIAL"`
	// ReorderLevel is a decimal string (up to 3 decimals; whole for SERIAL).
	ReorderLevel  *string              `json:"reorderLevel" example:"5"`
	Suppliers     []SupplierLinkInput  `json:"suppliers"`
	Compatibility []CompatibilityInput `json:"compatibility"`
} //@name StockItemCreate

// ItemPatch edits a SKU. suppliers and compatibility, when sent, replace the
// current set (removed supplier links are ended and kept as history). The SKU
// code and tracking mode are not editable.
type ItemPatch struct {
	Version       int                   `json:"version" example:"1"`
	Name          OptString             `json:"name" swaggertype:"string"`
	Description   OptString             `json:"description" swaggertype:"string"`
	CategoryID    OptString             `json:"categoryId" swaggertype:"string"`
	UOM           OptString             `json:"uom" swaggertype:"string"`
	ReorderLevel  OptString             `json:"reorderLevel" swaggertype:"string"`
	Suppliers     *[]SupplierLinkInput  `json:"suppliers"`
	Compatibility *[]CompatibilityInput `json:"compatibility"`
} //@name StockItemUpdate

var (
	// Letters, digits and . _ / - (no spaces); compared upper-cased (ADR-021).
	skuPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._/-]{0,39}$`)
	// A unit of measure code; no approved list yet (OQ-34). Stored upper-cased.
	uomPattern = regexp.MustCompile(`^[A-Z][A-Z0-9]{0,9}$`)
	// Up to 9 whole digits and 3 decimals (numeric(12,3)).
	levelPattern = regexp.MustCompile(`^[0-9]{1,9}(\.[0-9]{1,3})?$`)
)

const (
	maxSuppliers     = 20
	maxCompatibility = 50
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

func (c *checker) uom(s *string) {
	*s = strings.ToUpper(strings.TrimSpace(*s))
	switch {
	case *s == "":
		c.add("uom", "is required")
	case !uomPattern.MatchString(*s):
		c.add("uom", "must be a unit code of 1-10 letters and digits starting with a letter (e.g. PCS)")
	}
}

// reorderLevel checks a decimal string; SERIAL items count whole pieces.
func (c *checker) reorderLevel(s **string, tracking string) {
	if *s == nil {
		return
	}
	v := strings.TrimSpace(**s)
	if v == "" {
		*s = nil
		return
	}
	*s = &v
	switch {
	case !levelPattern.MatchString(v):
		c.add("reorderLevel", "must be a number from 0 with up to 3 decimals (e.g. 5 or 2.5)")
	case tracking == TrackingSerial && !wholeNumber(v):
		c.add("reorderLevel", "must be a whole number for a serial-tracked item")
	}
}

// wholeNumber reports whether a decimal string has no fractional part.
func wholeNumber(v string) bool {
	i := strings.IndexByte(v, '.')
	return i < 0 || strings.Trim(v[i+1:], "0") == ""
}

// NormaliseSKU is the comparison form of a SKU code (trimmed, upper-cased),
// the same as the database's generated sku_normalised.
func NormaliseSKU(s string) string { return strings.ToUpper(strings.TrimSpace(s)) }

func (c *checker) suppliers(list []SupplierLinkInput) {
	if len(list) > maxSuppliers {
		c.add("suppliers", "must list at most "+strconv.Itoa(maxSuppliers)+" suppliers")
		return
	}
	seen := map[string]bool{}
	for i := range list {
		f := "suppliers[" + strconv.Itoa(i) + "]"
		c.id(f+".supplierId", &list[i].SupplierID)
		c.optional(f+".supplierPartNumber", &list[i].SupplierPartNumber, 60)
		if seen[list[i].SupplierID] {
			c.add(f+".supplierId", "is listed twice")
		}
		seen[list[i].SupplierID] = true
	}
}

func (c *checker) compatibility(list []CompatibilityInput) {
	if len(list) > maxCompatibility {
		c.add("compatibility", "must list at most "+strconv.Itoa(maxCompatibility)+" entries")
		return
	}
	wholeBrand := map[string]bool{}
	for i := range list {
		if list[i].ModelID != nil && strings.TrimSpace(*list[i].ModelID) == "" {
			list[i].ModelID = nil
		}
		if list[i].ModelID == nil {
			wholeBrand[strings.TrimSpace(list[i].BrandID)] = true
		}
	}
	seen := map[string]bool{}
	for i := range list {
		f := "compatibility[" + strconv.Itoa(i) + "]"
		c.id(f+".brandId", &list[i].BrandID)
		key := list[i].BrandID + "/"
		if list[i].ModelID != nil {
			c.id(f+".modelId", list[i].ModelID)
			key += *list[i].ModelID
			if wholeBrand[list[i].BrandID] {
				c.add(f+".modelId", "is already covered: the whole brand is listed")
			}
		}
		if seen[key] {
			c.add(f, "is listed twice")
		}
		seen[key] = true
	}
}

func validateCategory(in *CategoryInput) error {
	var c checker
	c.required("name", &in.Name, 100)
	c.optional("description", &in.Description, 500)
	return c.err()
}

func validateCategoryPatch(p *CategoryPatch) error {
	var c checker
	c.version(p.Version)
	if p.Name.Set {
		if p.Name.Value == nil {
			c.add("name", "cannot be cleared")
		} else {
			c.required("name", p.Name.Value, 100)
		}
	}
	if p.Description.Set {
		c.optional("description", &p.Description.Value, 500)
	}
	if !p.Name.Set && !p.Description.Set {
		c.add("body", "change at least one field")
	}
	return c.err()
}

func validateArchive(in *ArchiveInput) error {
	var c checker
	c.version(in.Version)
	c.required("reason", &in.Reason, 300)
	return c.err()
}

func validateItem(in *ItemInput) error {
	var c checker
	in.SKU = strings.TrimSpace(in.SKU)
	switch {
	case in.SKU == "":
		c.add("sku", "is required")
	case !skuPattern.MatchString(in.SKU):
		c.add("sku", "must be 1-40 letters, digits, '.', '_', '/' or '-' (no spaces), starting with a letter or digit")
	}
	c.required("name", &in.Name, 200)
	c.optional("description", &in.Description, 1000)
	c.id("categoryId", &in.CategoryID)
	c.uom(&in.UOM)
	switch in.TrackingMode {
	case TrackingQuantity, TrackingSerial:
	case "":
		c.add("trackingMode", "is required")
	default:
		c.add("trackingMode", "must be QUANTITY or SERIAL")
	}
	c.reorderLevel(&in.ReorderLevel, in.TrackingMode)
	c.suppliers(in.Suppliers)
	c.compatibility(in.Compatibility)
	return c.err()
}

// validateItemPatch checks the shape; the reorder level is checked against the
// stored tracking mode by the store.
func validateItemPatch(p *ItemPatch) error {
	var c checker
	c.version(p.Version)
	if p.Name.Set {
		if p.Name.Value == nil {
			c.add("name", "cannot be cleared")
		} else {
			c.required("name", p.Name.Value, 200)
		}
	}
	if p.Description.Set {
		c.optional("description", &p.Description.Value, 1000)
	}
	if p.CategoryID.Set {
		if p.CategoryID.Value == nil {
			c.add("categoryId", "cannot be cleared")
		} else {
			c.id("categoryId", p.CategoryID.Value)
		}
	}
	if p.UOM.Set {
		if p.UOM.Value == nil {
			c.add("uom", "cannot be cleared")
		} else {
			c.uom(p.UOM.Value)
		}
	}
	if p.ReorderLevel.Set {
		// Shape only here; whole numbers for SERIAL are checked with the stored mode.
		c.reorderLevel(&p.ReorderLevel.Value, "")
	}
	if p.Suppliers != nil {
		c.suppliers(*p.Suppliers)
	}
	if p.Compatibility != nil {
		c.compatibility(*p.Compatibility)
	}
	if !p.Name.Set && !p.Description.Set && !p.CategoryID.Set && !p.UOM.Set && !p.ReorderLevel.Set &&
		p.Suppliers == nil && p.Compatibility == nil {
		c.add("body", "change at least one field")
	}
	return c.err()
}

// ListQuery is the raw item list query.
type ListQuery struct{ Query, Status, CategoryID, TrackingMode, SupplierID, BrandID, Page, Limit string }

// ListFilter is a validated item list query.
type ListFilter struct {
	Query, Status, CategoryID, TrackingMode, SupplierID, BrandID string
	Page, Limit                                                  int
}

// CategoryQuery is the raw category list query.
type CategoryQuery struct{ Query, Status, Page, Limit string }

func (c *checker) query(q string) string {
	q = strings.TrimSpace(q)
	if utf8.RuneCountInString(q) > 100 || hasControl(q) {
		c.add("q", "must be at most 100 characters")
	}
	return q
}

func (c *checker) status(s string) {
	switch s {
	case "", StatusActive, StatusArchived:
	default:
		c.add("status", "must be ACTIVE or ARCHIVED")
	}
}

func (c *checker) optionalID(field string, s *string) {
	*s = strings.TrimSpace(*s)
	if *s != "" {
		if _, err := uuid.Parse(*s); err != nil {
			c.add(field, "must be a valid ID")
		}
	}
}

func (c *checker) paging(page, limit string) (int, int) {
	p, l := 1, 25
	if page != "" {
		if n, err := strconv.Atoi(page); err != nil || n < 1 || n > 100000 {
			c.add("page", "must be a whole number from 1")
		} else {
			p = n
		}
	}
	if limit != "" {
		if n, err := strconv.Atoi(limit); err != nil || n < 1 || n > 100 {
			c.add("limit", "must be a whole number from 1 to 100")
		} else {
			l = n
		}
	}
	return p, l
}

func validateList(q ListQuery) (ListFilter, error) {
	var c checker
	f := ListFilter{Query: c.query(q.Query), Status: q.Status, CategoryID: q.CategoryID, TrackingMode: q.TrackingMode,
		SupplierID: q.SupplierID, BrandID: q.BrandID}
	c.status(f.Status)
	switch f.TrackingMode {
	case "", TrackingQuantity, TrackingSerial:
	default:
		c.add("trackingMode", "must be QUANTITY or SERIAL")
	}
	c.optionalID("categoryId", &f.CategoryID)
	c.optionalID("supplierId", &f.SupplierID)
	c.optionalID("brandId", &f.BrandID)
	f.Page, f.Limit = c.paging(q.Page, q.Limit)
	return f, c.err()
}

func validateCategoryList(q CategoryQuery) (CategoryFilter, error) {
	var c checker
	f := CategoryFilter{Query: c.query(q.Query), Status: q.Status}
	c.status(f.Status)
	f.Page, f.Limit = c.paging(q.Page, q.Limit)
	return f, c.err()
}

// CategoryFilter is a validated category list query.
type CategoryFilter struct {
	Query, Status string
	Page, Limit   int
}
