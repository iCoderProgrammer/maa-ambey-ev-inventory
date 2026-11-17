// Package openingstock is the opening stock import PREVIEW (G-06; ADR-030;
// IMP-001…IMP-005, FR-014): an authorised user downloads the template, uploads
// a filled CSV, and gets every row validated against the catalogue, locations,
// suppliers and existing vehicle identifiers. The file and the validation
// result are staged as an immutable batch. Nothing else changes: no vehicle
// unit, receipt, stock movement, balance or catalogue record is created or
// modified here. Committing an approved batch is G-07.
package openingstock

import (
	"bytes"
	"encoding/csv"
	"errors"
	"io"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
)

// TemplateVersion identifies the column set below. The file's first line must
// be TemplateMarker + TemplateVersion so a stale or foreign sheet is refused.
const (
	TemplateVersion = "OS-IMPORT-1"
	TemplateMarker  = "#template:"
)

// Column is one template column.
type Column struct {
	Name        string `json:"name" example:"chassis_number"`
	Required    bool   `json:"required"`
	MaxLength   int    `json:"maxLength" example:"100"`
	Format      string `json:"format" example:"text"`
	Description string `json:"description"`
} //@name OpeningStockTemplateColumn

// Column names.
const (
	ColBrand        = "brand"
	ColModelCode    = "model_code"
	ColVariantCode  = "variant_code"
	ColColour       = "colour"
	ColChassis      = "chassis_number"
	ColMotor        = "motor_serial"
	ColBattery      = "battery_serial"
	ColCharger      = "charger_serial"
	ColLocation     = "location_code"
	ColStatus       = "stock_status"
	ColReceivedDate = "received_date"
	ColProvenance   = "provenance"
	ColSupplier     = "supplier"
	ColPORef        = "purchase_order_reference"
	ColGRNRef       = "goods_receipt_reference"
	ColInvoiceRef   = "supplier_invoice_reference"
	ColCost         = "acquisition_cost"
	ColNotes        = "notes"
)

// Columns is the OS-IMPORT-1 template: the A-04 fields that the approved
// documents and the existing schema support (brand/model/variant, identifiers,
// colour, location, status/condition, received/opening date, provenance and
// supplier/PO/GRN/cost references where available). Nothing here is a value.
var Columns = []Column{
	{ColBrand, true, 120, "text", "Brand name exactly as in the catalogue (case does not matter)."},
	{ColModelCode, true, 60, "text", "The model's internal code in the catalogue, within that brand."},
	{ColVariantCode, true, 60, "text", "The variant's code in the catalogue, within that model."},
	{ColColour, false, 60, "text", "Colour name or colour code of the variant, if colours are configured."},
	{ColChassis, true, 100, "identifier", "Chassis/frame number as on the vehicle (compared trimmed and upper-cased)."},
	{ColMotor, false, 100, "identifier", "Motor serial, where the paperwork or label has one."},
	{ColBattery, false, 100, "identifier", "Battery serial, where applicable."},
	{ColCharger, false, 100, "identifier", "Charger serial, where applicable."},
	{ColLocation, true, 20, "code", "Location code where the vehicle physically is (must be one of your locations)."},
	{ColStatus, true, 30, "status", "Current stock status (see stockStatuses)."},
	{ColReceivedDate, false, 10, "date YYYY-MM-DD", "Date the vehicle was originally received, if known; not after today. Becomes an origin date only once approved in G-07."},
	{ColProvenance, true, 200, "text", "Where this record comes from (e.g. stock register, supplier invoice). Rows are opening balances, never invented purchase history (IMP-005)."},
	{ColSupplier, false, 200, "text", "Supplier legal or trade name, if known."},
	{ColPORef, false, 100, "text", "Original purchase order reference, if available."},
	{ColGRNRef, false, 100, "text", "Original goods receipt / delivery challan reference, if available."},
	{ColInvoiceRef, false, 100, "text", "Supplier invoice reference, if available."},
	{ColCost, false, 15, "decimal INR, up to 2 decimals", "Acquisition cost excluding tax, if available and approved (cost basis OQ-29)."},
	{ColNotes, false, 300, "text", "Anything else the reviewer should know."},
}

// StockStatuses are the statuses an opening unit may be in: the in-stock
// statuses the stock ledger can hold today (E-01/D-06). Reserved, sold,
// delivered, service and return statuses have no movement types yet and are
// refused (owner decision OQ-47).
var StockStatuses = []string{"RECEIVED_PENDING_PDI", "QUALITY_HOLD", "AVAILABLE", "DISPLAY", "REJECTED"}

// TemplateCSV is the downloadable template: the marker line and the header,
// with no data rows (nothing is invented).
func TemplateCSV() []byte {
	var b bytes.Buffer
	w := csv.NewWriter(&b)
	_ = w.Write([]string{TemplateMarker + TemplateVersion})
	names := make([]string, len(Columns))
	for i, c := range Columns {
		names[i] = c.Name
	}
	_ = w.Write(names)
	w.Flush()
	return b.Bytes()
}

func column(name string) (Column, bool) {
	i := slices.IndexFunc(Columns, func(c Column) bool { return c.Name == name })
	if i < 0 {
		return Column{}, false
	}
	return Columns[i], true
}

// Issue codes of the file itself (the batch is REJECTED and no row is staged).
const (
	FileNotUTF8            = "FILE_NOT_UTF8"
	FileMalformed          = "CSV_MALFORMED"
	FileMarkerMissing      = "TEMPLATE_MARKER_MISSING"
	FileVersionUnsupported = "TEMPLATE_VERSION_UNSUPPORTED"
	FileHeaderMissing      = "HEADER_MISSING"
	FileMissingColumn      = "MISSING_COLUMN"
	FileDuplicateColumn    = "DUPLICATE_COLUMN"
	FileUnknownColumn      = "UNKNOWN_COLUMN"
	FileNoRows             = "NO_ROWS"
	FileTooManyRows        = "TOO_MANY_ROWS"
)

// FileIssue is a problem with the file as a whole.
type FileIssue struct {
	Code    string `json:"code" example:"MISSING_COLUMN"`
	Line    int    `json:"line" example:"2"`
	Column  string `json:"column" example:"chassis_number"`
	Message string `json:"message"`
} //@name OpeningStockFileIssue

// SourceRow is one data line: its line number in the file and its values by
// column name (exactly as uploaded). FieldCountMismatch: the line has a
// different number of cells than the header.
type SourceRow struct {
	Line               int
	Values             map[string]string
	FieldCountMismatch bool
	Cells              int
}

// Parsed is a parsed file: the template version, the header's columns and the
// data rows, or the issues that make the file unusable.
type Parsed struct {
	Version      string
	Columns      []string
	Rows         []SourceRow
	SkippedBlank int
	Issues       []FileIssue
}

// Parse reads a template CSV (UTF-8, optional BOM, comma-separated). maxRows
// bounds the non-blank data rows.
func Parse(data []byte, maxRows int) Parsed {
	var p Parsed
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	if !utf8.Valid(data) {
		p.Issues = append(p.Issues, FileIssue{Code: FileNotUTF8, Message: "The file is not UTF-8 text. Save it as \"CSV UTF-8 (comma delimited)\"."})
		return p
	}
	r := csv.NewReader(bytes.NewReader(data))
	r.FieldsPerRecord = -1
	r.ReuseRecord = false
	read := func() ([]string, int, error) {
		rec, err := r.Read()
		if err != nil {
			return nil, 0, err
		}
		line, _ := r.FieldPos(0)
		return rec, line, nil
	}
	malformed := func(err error) Parsed {
		var pe *csv.ParseError
		line := 0
		if errors.As(err, &pe) {
			line = pe.Line
		}
		p.Issues = append(p.Issues, FileIssue{Code: FileMalformed, Line: line,
			Message: "The file is not valid CSV at line " + strconv.Itoa(line) + " (for example a stray quote). Fix it in the spreadsheet and save again."})
		return p
	}

	marker, line, err := read()
	switch {
	case errors.Is(err, io.EOF):
		p.Issues = append(p.Issues, FileIssue{Code: FileMarkerMissing, Line: 1, Message: "The file has no content."})
		return p
	case err != nil:
		return malformed(err)
	}
	first := strings.TrimSpace(marker[0])
	if !strings.HasPrefix(strings.ToLower(first), TemplateMarker) {
		p.Issues = append(p.Issues, FileIssue{Code: FileMarkerMissing, Line: line,
			Message: "The first line must be \"" + TemplateMarker + TemplateVersion + "\". Download the template and fill it in."})
		return p
	}
	p.Version = strings.TrimSpace(first[len(TemplateMarker):])
	if p.Version != TemplateVersion {
		p.Issues = append(p.Issues, FileIssue{Code: FileVersionUnsupported, Line: line,
			Message: "Template version \"" + p.Version + "\" is not supported; download the current template (" + TemplateVersion + ")."})
		return p
	}

	header, line, err := read()
	switch {
	case errors.Is(err, io.EOF):
		p.Issues = append(p.Issues, FileIssue{Code: FileHeaderMissing, Line: 2, Message: "The column header line is missing."})
		return p
	case err != nil:
		return malformed(err)
	}
	seen := map[string]bool{}
	for _, h := range header {
		name := strings.ToLower(strings.TrimSpace(h))
		switch {
		case name == "":
			p.Issues = append(p.Issues, FileIssue{Code: FileUnknownColumn, Line: line, Message: "A header cell is empty; remove the empty column."})
		case seen[name]:
			p.Issues = append(p.Issues, FileIssue{Code: FileDuplicateColumn, Line: line, Column: name,
				Message: "Column \"" + name + "\" appears more than once."})
		default:
			if _, ok := column(name); !ok {
				p.Issues = append(p.Issues, FileIssue{Code: FileUnknownColumn, Line: line, Column: name,
					Message: "Column \"" + name + "\" is not part of the template; remove or rename it."})
			}
		}
		seen[name] = true
		p.Columns = append(p.Columns, name)
	}
	for _, c := range Columns {
		if c.Required && !seen[c.Name] {
			p.Issues = append(p.Issues, FileIssue{Code: FileMissingColumn, Line: line, Column: c.Name,
				Message: "Required column \"" + c.Name + "\" is missing."})
		}
	}
	if len(p.Issues) > 0 {
		return p
	}

	for {
		rec, line, err := read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			p.Rows = nil
			return malformed(err)
		}
		blank := true
		for _, v := range rec {
			if strings.TrimSpace(v) != "" {
				blank = false
				break
			}
		}
		if blank {
			p.SkippedBlank++
			continue
		}
		if len(p.Rows) >= maxRows {
			p.Rows = nil
			p.Issues = append(p.Issues, FileIssue{Code: FileTooManyRows, Line: line,
				Message: "The file has more than " + strconv.Itoa(maxRows) + " rows; split it into smaller files."})
			return p
		}
		row := SourceRow{Line: line, Values: map[string]string{}, Cells: len(rec), FieldCountMismatch: len(rec) != len(p.Columns)}
		for i, name := range p.Columns {
			if i < len(rec) {
				row.Values[name] = rec[i]
			} else {
				row.Values[name] = ""
			}
		}
		p.Rows = append(p.Rows, row)
	}
	if len(p.Rows) == 0 {
		p.Issues = append(p.Issues, FileIssue{Code: FileNoRows, Line: 3, Message: "The file has no data rows below the header."})
	}
	return p
}
