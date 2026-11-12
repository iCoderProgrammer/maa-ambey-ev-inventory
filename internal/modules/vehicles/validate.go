package vehicles

import (
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
)

// ListQuery is the raw list request.
type ListQuery struct {
	Query, Status, BrandID, ModelID, VariantID, SupplierID, PurchaseOrderID, GoodsReceiptID, LocationID, Page, Limit string
	// Saleable is "", "true" (AVAILABLE units only, INV-006) or "false" (every other status).
	Saleable string
	// ReceivedFrom/ReceivedTo are inclusive dates (YYYY-MM-DD); MinAgeDays/MaxAgeDays
	// are inclusive whole days since the received date.
	ReceivedFrom, ReceivedTo, MinAgeDays, MaxAgeDays string
	// Sort is one of SortOrders (default "-createdAt").
	Sort string
}

// SortOrders are the accepted list orders: a field, descending with a leading "-".
// Every order ends with the unit ID, so pages are stable.
var SortOrders = []string{"-createdAt", "createdAt", "-receivedDate", "receivedDate", "unitNumber", "-unitNumber"}

// maxAgeDays bounds the age filters (100 years).
const maxAgeDays = 36500

// ListFilter is a validated list request. Query is normalised like an
// identifier. LocationIDs is an explicit location filter (already checked
// against the scope); without one, a caller who does not have every location
// sees units without a location or at one of ScopeLocationIDs.
type ListFilter struct {
	Query, Status, Saleable                                                  string
	BrandID, ModelID, VariantID, SupplierID, PurchaseOrderID, GoodsReceiptID string
	ReceivedFrom, ReceivedTo                                                 string // YYYY-MM-DD or ""
	MinAgeDays, MaxAgeDays                                                   *int
	Sort                                                                     string
	LocationIDs, ScopeLocationIDs                                            []string
	AllLocations                                                             bool
	Page, Limit                                                              int
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
	f := ListFilter{Status: strings.TrimSpace(q.Status), BrandID: q.BrandID, ModelID: q.ModelID, VariantID: q.VariantID,
		SupplierID: q.SupplierID, PurchaseOrderID: q.PurchaseOrderID, GoodsReceiptID: q.GoodsReceiptID, Page: 1, Limit: 25,
		Sort: "-createdAt"}
	query := strings.TrimSpace(q.Query)
	if utf8.RuneCountInString(query) > 100 || strings.ContainsFunc(query, unicode.IsControl) {
		c.add("q", "must be at most 100 characters")
	}
	f.Query = NormaliseIdentifier(query)
	if f.Status != "" && !slices.Contains(Statuses, f.Status) {
		c.add("status", "must be one of "+strings.Join(Statuses, ", "))
	}
	switch f.Saleable = strings.TrimSpace(q.Saleable); f.Saleable {
	case "", "true", "false":
	default:
		c.add("saleable", "must be true or false")
	}
	c.optionalID("brandId", &f.BrandID)
	c.optionalID("modelId", &f.ModelID)
	c.optionalID("variantId", &f.VariantID)
	c.optionalID("supplierId", &f.SupplierID)
	c.optionalID("purchaseOrderId", &f.PurchaseOrderID)
	c.optionalID("goodsReceiptId", &f.GoodsReceiptID)
	location := q.LocationID
	c.optionalID("locationId", &location)
	if location != "" {
		f.LocationIDs = []string{location}
	}
	f.ReceivedFrom = c.optionalDate("receivedFrom", q.ReceivedFrom)
	f.ReceivedTo = c.optionalDate("receivedTo", q.ReceivedTo)
	if f.ReceivedFrom != "" && f.ReceivedTo != "" && f.ReceivedFrom > f.ReceivedTo {
		c.add("receivedTo", "must not be before receivedFrom")
	}
	f.MinAgeDays = c.optionalDays("minAgeDays", q.MinAgeDays)
	f.MaxAgeDays = c.optionalDays("maxAgeDays", q.MaxAgeDays)
	if f.MinAgeDays != nil && f.MaxAgeDays != nil && *f.MinAgeDays > *f.MaxAgeDays {
		c.add("maxAgeDays", "must not be less than minAgeDays")
	}
	if s := strings.TrimSpace(q.Sort); s != "" {
		if !slices.Contains(SortOrders, s) {
			c.add("sort", "must be one of "+strings.Join(SortOrders, ", "))
		} else {
			f.Sort = s
		}
	}
	f.Page, f.Limit = c.paging(q.Page, q.Limit, 25)
	return f, c.err()
}

func (c *checker) optionalDate(field, s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if _, err := time.Parse(time.DateOnly, s); err != nil {
		c.add(field, "must be a date (YYYY-MM-DD)")
		return ""
	}
	return s
}

func (c *checker) optionalDays(field, s string) *int {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 || n > maxAgeDays {
		c.add(field, "must be a whole number of days from 0 to "+strconv.Itoa(maxAgeDays))
		return nil
	}
	return &n
}

// paging validates page (from 1) and limit (1-100).
func (c *checker) paging(pageS, limitS string, defaultLimit int) (page, limit int) {
	page, limit = 1, defaultLimit
	if pageS != "" {
		if n, err := strconv.Atoi(pageS); err != nil || n < 1 || n > 100000 {
			c.add("page", "must be a whole number from 1")
		} else {
			page = n
		}
	}
	if limitS != "" {
		if n, err := strconv.Atoi(limitS); err != nil || n < 1 || n > 100 {
			c.add("limit", "must be a whole number from 1 to 100")
		} else {
			limit = n
		}
	}
	return page, limit
}

// MovementQuery is the raw request for a unit's stock movements.
type MovementQuery struct{ Order, Page, Limit string }

// MovementFilter is a validated MovementQuery. Newest is true for order=desc.
type MovementFilter struct {
	UnitID      string
	Newest      bool
	Page, Limit int
}

func validateMovements(unitID string, q MovementQuery) (MovementFilter, error) {
	var c checker
	f := MovementFilter{UnitID: unitID}
	switch strings.TrimSpace(q.Order) {
	case "", "asc":
	case "desc":
		f.Newest = true
	default:
		c.add("order", "must be asc or desc")
	}
	f.Page, f.Limit = c.paging(q.Page, q.Limit, 50)
	return f, c.err()
}
