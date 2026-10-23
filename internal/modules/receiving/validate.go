package receiving

import (
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
)

// Limits of one receipt.
const (
	MaxLines        = 100
	MaxUnits        = 1000
	MaxLineQuantity = 10000
	// MaxIdentifierLength is the longest chassis/frame number or serial accepted.
	MaxIdentifierLength = 50
)

// CreateInput starts a draft receipt against an approved PO. Lines are entered
// with Save.
type CreateInput struct {
	PurchaseOrderID   string  `json:"purchaseOrderId"`
	ReceivedDate      string  `json:"receivedDate" example:"2026-10-23"`
	SupplierReference *string `json:"supplierReference" example:"DC-1234"`
	Notes             *string `json:"notes"`
} //@name GoodsReceiptCreate

// UnitInput is one physical unit. Every identifier is optional in a draft.
type UnitInput struct {
	Condition     string  `json:"condition" enums:"ACCEPTED,REJECTED" example:"ACCEPTED"`
	ChassisNumber *string `json:"chassisNumber"`
	MotorSerial   *string `json:"motorSerial"`
	BatterySerial *string `json:"batterySerial"`
	ChargerSerial *string `json:"chargerSerial"`
	Note          *string `json:"note"`
} //@name GoodsReceiptUnitInput

// LineInput is what arrived for one PO line. received = accepted + rejected; a
// line with all quantities zero, no units and no note is left out.
// overReceiptReason authorises receiving more than the line has left (needs
// goods_receipts.over_receipt); an earlier authorisation that still covers the
// received quantity is kept without sending it again.
type LineInput struct {
	PurchaseOrderLineID string      `json:"purchaseOrderLineId"`
	ReceivedQuantity    int         `json:"receivedQuantity" example:"2"`
	AcceptedQuantity    int         `json:"acceptedQuantity" example:"2"`
	RejectedQuantity    int         `json:"rejectedQuantity" example:"0"`
	DiscrepancyNote     *string     `json:"discrepancyNote"`
	OverReceiptReason   *string     `json:"overReceiptReason"`
	Units               []UnitInput `json:"units"`
} //@name GoodsReceiptLineInput

// SaveInput replaces a draft's header, lines and units (version is the receipt's).
type SaveInput struct {
	Version           int         `json:"version" example:"1"`
	ReceivedDate      string      `json:"receivedDate" example:"2026-10-23"`
	SupplierReference *string     `json:"supplierReference"`
	Notes             *string     `json:"notes"`
	Lines             []LineInput `json:"lines"`
} //@name GoodsReceiptSave

// CancelInput cancels a draft.
type CancelInput struct {
	Version int    `json:"version" example:"2"`
	Reason  string `json:"reason" example:"Entered against the wrong purchase order"`
} //@name GoodsReceiptCancel

// DocumentInput links one of the caller's unlinked uploads to the receipt.
type DocumentInput struct {
	AttachmentID string  `json:"attachmentId"`
	DocumentType string  `json:"documentType" enums:"DELIVERY_CHALLAN,SUPPLIER_INVOICE,PHOTO,OTHER" example:"DELIVERY_CHALLAN"`
	Note         *string `json:"note"`
} //@name GoodsReceiptDocumentInput

var idempotencyPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{8,100}$`)

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

// date checks a required YYYY-MM-DD; "not in the future" is checked against
// the database date in the transaction.
func (c *checker) date(field string, s *string) {
	*s = strings.TrimSpace(*s)
	if *s == "" {
		c.add(field, "is required")
	} else if _, err := time.Parse(time.DateOnly, *s); err != nil {
		c.add(field, "must be a date (YYYY-MM-DD)")
	}
}

// identifier trims an optional chassis/frame number or serial. Only printable
// ASCII is accepted so the normalised form is the same in Go and PostgreSQL;
// formats per brand are pending (OQ-4, D-04/D-05).
func (c *checker) identifier(field string, s **string) {
	if *s == nil {
		return
	}
	v := strings.TrimSpace(**s)
	if v == "" {
		*s = nil
		return
	}
	*s = &v
	if len(v) > MaxIdentifierLength {
		c.add(field, "must be at most "+strconv.Itoa(MaxIdentifierLength)+" characters")
		return
	}
	for _, r := range v {
		if r < 0x20 || r > 0x7e {
			c.add(field, "may contain only letters, digits, spaces and punctuation (no accents or symbols)")
			return
		}
	}
}

// NormaliseChassis is the comparison form of a chassis/frame number: trimmed
// and upper-cased (INV-002). Which separators may also be ignored is not yet
// approved (OQ-26), so separators are kept.
func NormaliseChassis(s string) string { return strings.ToUpper(strings.TrimSpace(s)) }

func validateCreate(in *CreateInput) error {
	var c checker
	c.id("purchaseOrderId", &in.PurchaseOrderID)
	c.date("receivedDate", &in.ReceivedDate)
	c.optional("supplierReference", &in.SupplierReference, 100)
	c.optional("notes", &in.Notes, 1000)
	return c.err()
}

func emptyLine(l LineInput) bool {
	return l.ReceivedQuantity == 0 && l.AcceptedQuantity == 0 && l.RejectedQuantity == 0 && len(l.Units) == 0 &&
		(l.DiscrepancyNote == nil || strings.TrimSpace(*l.DiscrepancyNote) == "") &&
		(l.OverReceiptReason == nil || strings.TrimSpace(*l.OverReceiptReason) == "")
}

// validateSave checks the shape of a save and drops empty lines. Field paths
// keep the client's line and unit positions so the screen can mark them.
// Quantities against the PO and duplicates across receipts are checked in the
// transaction.
func validateSave(in *SaveInput) ([]indexedLine, error) {
	var c checker
	c.version(in.Version)
	c.date("receivedDate", &in.ReceivedDate)
	c.optional("supplierReference", &in.SupplierReference, 100)
	c.optional("notes", &in.Notes, 1000)
	if len(in.Lines) > MaxLines {
		c.add("lines", "must have at most "+strconv.Itoa(MaxLines)+" lines")
		return nil, c.err()
	}
	var lines []indexedLine
	seenPOLine := map[string]bool{}
	chassisAt := map[string]string{}
	units := 0
	for i := range in.Lines {
		l := &in.Lines[i]
		if emptyLine(*l) {
			continue
		}
		p := "lines[" + strconv.Itoa(i) + "]."
		c.id(p+"purchaseOrderLineId", &l.PurchaseOrderLineID)
		if l.PurchaseOrderLineID != "" && seenPOLine[l.PurchaseOrderLineID] {
			c.add(p+"purchaseOrderLineId", "is already on another line of this receipt")
		}
		seenPOLine[l.PurchaseOrderLineID] = true
		qtyOK := true
		for _, q := range []struct {
			field string
			v     int
		}{{"receivedQuantity", l.ReceivedQuantity}, {"acceptedQuantity", l.AcceptedQuantity}, {"rejectedQuantity", l.RejectedQuantity}} {
			if q.v < 0 || q.v > MaxLineQuantity {
				c.add(p+q.field, "must be a whole number from 0 to "+strconv.Itoa(MaxLineQuantity))
				qtyOK = false
			}
		}
		if qtyOK && l.ReceivedQuantity < 1 {
			c.add(p+"receivedQuantity", "must be at least 1 when anything else is entered for this line")
			qtyOK = false
		}
		if qtyOK && l.AcceptedQuantity+l.RejectedQuantity != l.ReceivedQuantity {
			c.add(p+"acceptedQuantity", "accepted plus rejected must equal the received quantity")
		}
		c.optional(p+"discrepancyNote", &l.DiscrepancyNote, 500)
		c.optional(p+"overReceiptReason", &l.OverReceiptReason, 500)
		units += len(l.Units)
		if len(l.Units) > l.ReceivedQuantity && qtyOK {
			c.add(p+"units", "has more units than the received quantity ("+strconv.Itoa(l.ReceivedQuantity)+")")
		}
		var accepted, rejected int
		for j := range l.Units {
			u := &l.Units[j]
			up := p + "units[" + strconv.Itoa(j) + "]."
			switch u.Condition {
			case ConditionAccepted:
				accepted++
			case ConditionRejected:
				rejected++
			default:
				c.add(up+"condition", "must be one of "+strings.Join(Conditions, ", "))
			}
			c.identifier(up+"chassisNumber", &u.ChassisNumber)
			c.identifier(up+"motorSerial", &u.MotorSerial)
			c.identifier(up+"batterySerial", &u.BatterySerial)
			c.identifier(up+"chargerSerial", &u.ChargerSerial)
			c.optional(up+"note", &u.Note, 300)
			if u.ChassisNumber != nil {
				n := NormaliseChassis(*u.ChassisNumber)
				if prev, ok := chassisAt[n]; ok {
					c.add(up+"chassisNumber", "is the same as "+prev+" on this receipt")
				} else {
					chassisAt[n] = "unit " + strconv.Itoa(j+1) + " of line " + strconv.Itoa(i+1)
				}
			}
		}
		if qtyOK && accepted > l.AcceptedQuantity {
			c.add(p+"units", "has "+strconv.Itoa(accepted)+" accepted units but the accepted quantity is "+strconv.Itoa(l.AcceptedQuantity))
		}
		if qtyOK && rejected > l.RejectedQuantity {
			c.add(p+"units", "has "+strconv.Itoa(rejected)+" rejected units but the rejected quantity is "+strconv.Itoa(l.RejectedQuantity))
		}
		lines = append(lines, indexedLine{Index: i, LineInput: *l})
	}
	if units > MaxUnits {
		c.add("lines", "a receipt has at most "+strconv.Itoa(MaxUnits)+" units")
	}
	return lines, c.err()
}

// indexedLine is a non-empty input line with its position in the request.
type indexedLine struct {
	Index int
	LineInput
}

func (l indexedLine) path(field string) string {
	return "lines[" + strconv.Itoa(l.Index) + "]." + field
}

func validateCancel(in *CancelInput) error {
	var c checker
	c.version(in.Version)
	c.required("reason", &in.Reason, 500)
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
type ListQuery struct{ Query, Status, PurchaseOrderID, Page, Limit string }

// ListFilter is a validated list query.
type ListFilter struct {
	Query, Status, PurchaseOrderID string
	Page, Limit                    int
}

func validatePaging(c *checker, page, limit string, f *ListFilter) {
	if page != "" {
		if n, err := strconv.Atoi(page); err != nil || n < 1 || n > 100000 {
			c.add("page", "must be a whole number from 1")
		} else {
			f.Page = n
		}
	}
	if limit != "" {
		if n, err := strconv.Atoi(limit); err != nil || n < 1 || n > 100 {
			c.add("limit", "must be a whole number from 1 to 100")
		} else {
			f.Limit = n
		}
	}
}

func validateQuery(c *checker, q string) string {
	q = strings.TrimSpace(q)
	if utf8.RuneCountInString(q) > 100 || hasControl(q) {
		c.add("q", "must be at most 100 characters")
	}
	return q
}

func validateList(q ListQuery) (ListFilter, error) {
	var c checker
	f := ListFilter{Query: validateQuery(&c, q.Query), Status: q.Status, PurchaseOrderID: strings.TrimSpace(q.PurchaseOrderID), Page: 1, Limit: 25}
	if f.Status != "" && !slices.Contains(Statuses, f.Status) {
		c.add("status", "must be one of "+strings.Join(Statuses, ", "))
	}
	if f.PurchaseOrderID != "" {
		if _, err := uuid.Parse(f.PurchaseOrderID); err != nil {
			c.add("purchaseOrderId", "must be a valid ID")
		}
	}
	validatePaging(&c, q.Page, q.Limit, &f)
	return f, c.err()
}

func validateReceivable(query, page, limit string) (ListFilter, error) {
	var c checker
	f := ListFilter{Query: validateQuery(&c, query), Page: 1, Limit: 25}
	validatePaging(&c, page, limit, &f)
	return f, c.err()
}
