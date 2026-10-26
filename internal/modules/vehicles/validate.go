package vehicles

import (
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
)

// ListQuery is the raw list request.
type ListQuery struct {
	Query, Status, BrandID, VariantID, SupplierID, PurchaseOrderID, GoodsReceiptID, LocationID, Page, Limit string
}

// ListFilter is a validated list request. Query is normalised like an
// identifier. LocationIDs is an explicit location filter (already checked
// against the scope); without one, a caller who does not have every location
// sees units without a location or at one of ScopeLocationIDs.
type ListFilter struct {
	Query, Status                                                   string
	BrandID, VariantID, SupplierID, PurchaseOrderID, GoodsReceiptID string
	LocationIDs, ScopeLocationIDs                                   []string
	AllLocations                                                    bool
	Page, Limit                                                     int
}

type checker struct{ fields []FieldProblem }

func (c *checker) add(f, m string) { c.fields = append(c.fields, FieldProblem{f, m}) }

func (c *checker) err() error {
	if len(c.fields) == 0 {
		return nil
	}
	return &ValidationError{Fields: c.fields}
}

func (c *checker) optionalID(field string, s *string) {
	*s = strings.TrimSpace(*s)
	if *s == "" {
		return
	}
	if _, err := uuid.Parse(*s); err != nil {
		c.add(field, "must be a valid ID")
	}
}

func validateList(q ListQuery) (ListFilter, error) {
	var c checker
	f := ListFilter{Status: strings.TrimSpace(q.Status), BrandID: q.BrandID, VariantID: q.VariantID, SupplierID: q.SupplierID,
		PurchaseOrderID: q.PurchaseOrderID, GoodsReceiptID: q.GoodsReceiptID, Page: 1, Limit: 25}
	query := strings.TrimSpace(q.Query)
	if utf8.RuneCountInString(query) > 100 || strings.ContainsFunc(query, unicode.IsControl) {
		c.add("q", "must be at most 100 characters")
	}
	f.Query = NormaliseIdentifier(query)
	if f.Status != "" && !slices.Contains(Statuses, f.Status) {
		c.add("status", "must be one of "+strings.Join(Statuses, ", "))
	}
	c.optionalID("brandId", &f.BrandID)
	c.optionalID("variantId", &f.VariantID)
	c.optionalID("supplierId", &f.SupplierID)
	c.optionalID("purchaseOrderId", &f.PurchaseOrderID)
	c.optionalID("goodsReceiptId", &f.GoodsReceiptID)
	location := q.LocationID
	c.optionalID("locationId", &location)
	if location != "" {
		f.LocationIDs = []string{location}
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
