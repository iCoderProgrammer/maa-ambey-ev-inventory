package openingstock

import (
	"slices"
	"strings"
	"testing"
	"time"
)

// Parsing and validation without a database. Every value is a test placeholder
// ("Brand A", "CH-0001"): no real brand, vehicle or serial.

const header = "brand,model_code,variant_code,colour,chassis_number,motor_serial,battery_serial,charger_serial,location_code," +
	"stock_status,received_date,provenance,supplier,purchase_order_reference,goods_receipt_reference,supplier_invoice_reference," +
	"acquisition_cost,notes"

func file(lines ...string) []byte {
	return []byte("#template:OS-IMPORT-1\n" + header + "\n" + strings.Join(lines, "\n") + "\n")
}

func codes(is []FileIssue) []string {
	out := []string{}
	for _, i := range is {
		out = append(out, i.Code)
	}
	return out
}

func TestTemplateHasNoDataAndParsesBack(t *testing.T) {
	tpl := string(TemplateCSV())
	lines := strings.Split(strings.TrimSpace(tpl), "\n")
	if len(lines) != 2 || lines[0] != "#template:OS-IMPORT-1" || lines[1] != header {
		t.Fatalf("template:\n%s", tpl)
	}
	// The empty template is a valid template with no rows: refused as NO_ROWS, nothing invented.
	if p := Parse(TemplateCSV(), 10); !slices.Equal(codes(p.Issues), []string{FileNoRows}) || len(p.Rows) != 0 {
		t.Fatalf("%+v", p)
	}
	for _, c := range Columns {
		if c.Description == "" || c.MaxLength == 0 {
			t.Errorf("column %s undocumented", c.Name)
		}
	}
}

func TestParseFileIssues(t *testing.T) {
	row := "Brand A,MA,STD,,CH-1,,,,L1,AVAILABLE,,Register,,,,,,"
	for name, c := range map[string]struct {
		data string
		want []string
	}{
		"not utf-8":         {"#template:OS-IMPORT-1\n" + header + "\nBrand \xff\n", []string{FileNotUTF8}},
		"no marker":         {header + "\n" + row + "\n", []string{FileMarkerMissing}},
		"old version":       {"#template:OS-IMPORT-0\n" + header + "\n" + row + "\n", []string{FileVersionUnsupported}},
		"no header":         {"#template:OS-IMPORT-1\n", []string{FileHeaderMissing}},
		"missing columns":   {"#template:OS-IMPORT-1\nbrand,model_code,variant_code,location_code,stock_status,provenance\nA,B,C,L1,AVAILABLE,x\n", []string{FileMissingColumn}},
		"duplicate column":  {"#template:OS-IMPORT-1\n" + header + ",brand\n" + row + ",A\n", []string{FileDuplicateColumn}},
		"unknown column":    {"#template:OS-IMPORT-1\n" + header + ",price\n" + row + ",1\n", []string{FileUnknownColumn}},
		"empty header cell": {"#template:OS-IMPORT-1\n" + header + ",\n" + row + ",\n", []string{FileUnknownColumn}},
		"stray quote":       {"#template:OS-IMPORT-1\n" + header + "\nBrand \"A,MA\n", []string{FileMalformed}},
		"no rows":           {"#template:OS-IMPORT-1\n" + header + "\n,,,,\n\n", []string{FileNoRows}},
		"too many rows":     {string(file(row, row, row)), []string{FileTooManyRows}},
		"empty":             {"", []string{FileMarkerMissing}},
	} {
		p := Parse([]byte(c.data), 2)
		got := codes(p.Issues)
		for _, w := range c.want {
			if !slices.Contains(got, w) {
				t.Errorf("%s: issues %v, want %s", name, got, w)
			}
		}
		if len(p.Rows) != 0 {
			t.Errorf("%s: rows staged from an unusable file", name)
		}
	}
	// Missing columns are named, one issue each.
	p := Parse([]byte("#template:OS-IMPORT-1\nbrand,model_code,variant_code,location_code,stock_status,provenance\nA,B,C,L1,AVAILABLE,x\n"), 10)
	if len(p.Issues) != 1 || p.Issues[0].Column != ColChassis || p.Issues[0].Line != 2 {
		t.Fatalf("%+v", p.Issues)
	}
}

func TestParseRowsKeepLinesAndValues(t *testing.T) {
	// BOM, CRLF, a quoted multi-line note, a blank line, column order free, optional columns left out.
	data := "\xef\xbb\xbf#template:OS-IMPORT-1,,\r\nstock_status,brand,model_code,variant_code,chassis_number,location_code,provenance,notes\r\n" +
		"AVAILABLE,Brand A,MA,STD,CH-1,L1,Register,\"two\nlines\"\r\n,,,,,,,\r\nDISPLAY,Brand A,MA,STD,CH-2,L1,Register,\r\n" +
		"DISPLAY,Brand A,MA,STD,CH-3,L1\r\n"
	p := Parse([]byte(data), 10)
	if len(p.Issues) != 0 || p.Version != TemplateVersion || len(p.Rows) != 3 || p.SkippedBlank != 1 {
		t.Fatalf("%+v", p)
	}
	if p.Rows[0].Line != 3 || p.Rows[0].Values[ColNotes] != "two\nlines" || p.Rows[1].Line != 6 || p.Rows[1].Values[ColChassis] != "CH-2" {
		t.Fatalf("rows %+v", p.Rows)
	}
	if !p.Rows[2].FieldCountMismatch || p.Rows[1].FieldCountMismatch {
		t.Fatalf("field counts %+v", p.Rows)
	}
}

type allScope struct{ allowed []string }

func (s allScope) Allows(id string) bool { return s.allowed == nil || slices.Contains(s.allowed, id) }

var bizDate = time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)

func lookups() Lookups {
	code := "RED"
	trade := "Example Trade"
	return Lookups{
		Brands: []BrandRef{{"b-ok", "Brand A", "ACTIVE", "AUTHORISED"}, {"b-pending", "Brand P", "PENDING_VERIFICATION", "UNVERIFIED"},
			{"b-arch", "Brand Z", "ARCHIVED", "AUTHORISED"}},
		Models: []ModelRef{{"m-ok", "b-ok", "MA", "Model A", "ACTIVE"}, {"m-pend", "b-ok", "MP", "Model P", "PENDING_VERIFICATION"},
			{"m-other", "b-pending", "MX", "Model X", "ACTIVE"}, {"m-p", "b-pending", "PA", "Model PA", "PENDING_VERIFICATION"}},
		Variants: []VariantRef{{"v-ok", "b-ok", "m-ok", "STD", "Standard", "ACTIVE"}, {"v-pend", "b-ok", "m-ok", "PRO", "Pro", "PENDING_VERIFICATION"},
			{"v-mp", "b-ok", "m-pend", "BASE", "Base", "ACTIVE"}, {"v-pa", "b-pending", "m-p", "STD", "Std", "ACTIVE"}},
		Colours:         []ColourRef{{ID: "c-red", VariantID: "v-ok", Name: "Red", Lifecycle: "ACTIVE", Code: &code}, {ID: "c-old", VariantID: "v-ok", Name: "Grey", Lifecycle: "ARCHIVED"}},
		Locations:       []LocationRef{{"l1", "L1", "One", true}, {"l2", "L2", "Two", true}, {"l9", "L9", "Closed", false}},
		Suppliers:       []SupplierRef{{ID: "s1", LegalName: "Example Supplier", Status: "VERIFIED", TradeName: &trade}, {ID: "s2", LegalName: "Old Supplier", Status: "ARCHIVED"}},
		ExistingChassis: map[string]bool{"CH-EXIST": true}, OpenReceiptChassis: map[string]bool{"CH-DRAFT": true},
		ExistingMotor: map[string]bool{"MOT-EXIST": true}, ExistingBattery: map[string]bool{}, ExistingCharger: map[string]bool{},
	}
}

func rowWith(line int, kv ...string) SourceRow {
	v := map[string]string{ColBrand: "Brand A", ColModelCode: "MA", ColVariantCode: "STD", ColChassis: "CH-" + string(rune('A'+line)),
		ColLocation: "L1", ColStatus: "AVAILABLE", ColProvenance: "Stock register", ColReceivedDate: "2026-01-15", ColCost: "95000.00"}
	for i := 0; i+1 < len(kv); i += 2 {
		v[kv[i]] = kv[i+1]
	}
	for _, c := range Columns {
		if _, ok := v[c.Name]; !ok {
			v[c.Name] = ""
		}
	}
	return SourceRow{Line: line, Values: v}
}

func issueCodes(r RowResult) []string {
	out := []string{}
	for _, i := range r.Issues {
		out = append(out, i.Field+":"+i.Code+":"+i.Severity)
	}
	return out
}

func TestValidateEveryRule(t *testing.T) {
	cases := []struct {
		name   string
		row    SourceRow
		status string
		want   []string // field:code:severity
	}{
		{"valid", rowWith(3, ColChassis, " ch-valid ", ColColour, "red", ColSupplier, "example trade"), RowValid, nil},
		{"code colour", rowWith(4, ColColour, "RED"), RowValid, nil},
		{"unknown brand", rowWith(5, ColBrand, "Brand Q"), RowError, []string{"brand:UNKNOWN_BRAND:ERROR"}},
		{"pending brand", rowWith(6, ColBrand, "brand p", ColModelCode, "PA"), RowBlocked, []string{"brand:BRAND_NOT_ACTIVE:BLOCKED", "model_code:MODEL_NOT_ACTIVE:BLOCKED"}},
		{"archived brand", rowWith(7, ColBrand, "Brand Z", ColModelCode, "ZZ"), RowError, []string{"brand:BRAND_ARCHIVED:BLOCKED", "model_code:UNKNOWN_MODEL:ERROR"}},
		{"unknown model", rowWith(8, ColModelCode, "NOPE"), RowError, []string{"model_code:UNKNOWN_MODEL:ERROR"}},
		{"model of other brand", rowWith(9, ColModelCode, "MX"), RowError, []string{"model_code:MODEL_NOT_IN_BRAND:ERROR"}},
		{"pending model", rowWith(10, ColModelCode, "MP", ColVariantCode, "BASE"), RowBlocked, []string{"model_code:MODEL_NOT_ACTIVE:BLOCKED"}},
		{"unknown variant", rowWith(11, ColVariantCode, "X"), RowError, []string{"variant_code:UNKNOWN_VARIANT:ERROR"}},
		{"variant of other model", rowWith(12, ColVariantCode, "BASE"), RowError, []string{"variant_code:VARIANT_NOT_IN_MODEL:ERROR"}},
		{"pending variant", rowWith(13, ColVariantCode, "PRO"), RowBlocked, []string{"variant_code:VARIANT_NOT_ACTIVE:BLOCKED"}},
		{"unknown colour", rowWith(14, ColColour, "Blue"), RowError, []string{"colour:UNKNOWN_COLOUR:ERROR"}},
		{"archived colour", rowWith(15, ColColour, "Grey"), RowBlocked, []string{"colour:COLOUR_NOT_ACTIVE:BLOCKED"}},
		{"colour not configured", rowWith(16, ColBrand, "Brand P", ColModelCode, "PA", ColColour, "Red"), RowBlocked,
			[]string{"brand:BRAND_NOT_ACTIVE:BLOCKED", "model_code:MODEL_NOT_ACTIVE:BLOCKED", "colour:COLOUR_NOT_CONFIGURED:WARNING"}},
		{"unknown location", rowWith(17, ColLocation, "L7"), RowError, []string{"location_code:UNKNOWN_LOCATION:ERROR"}},
		{"inactive location", rowWith(18, ColLocation, "l9"), RowError, []string{"location_code:LOCATION_INACTIVE:ERROR"}},
		{"out of scope", rowWith(19, ColLocation, "L2"), RowError, []string{"location_code:LOCATION_OUT_OF_SCOPE:ERROR"}},
		{"bad status", rowWith(20, ColStatus, "SOLD"), RowError, []string{"stock_status:INVALID_STATUS:ERROR"}},
		{"bad date", rowWith(21, ColReceivedDate, "15/01/2026"), RowError, []string{"received_date:INVALID_DATE:ERROR"}},
		{"future date", rowWith(22, ColReceivedDate, "2026-07-01"), RowError, []string{"received_date:FUTURE_DATE:ERROR"}},
		{"no date", rowWith(23, ColReceivedDate, ""), RowWarning, []string{"received_date:ORIGIN_DATE_MISSING:WARNING"}},
		{"bad cost", rowWith(24, ColCost, "₹95000"), RowError, []string{"acquisition_cost:INVALID_COST:ERROR"}},
		{"no cost", rowWith(25, ColCost, ""), RowWarning, []string{"acquisition_cost:COST_MISSING:WARNING"}},
		{"unknown supplier", rowWith(26, ColSupplier, "Nobody"), RowWarning, []string{"supplier:SUPPLIER_NOT_FOUND:WARNING"}},
		{"archived supplier", rowWith(27, ColSupplier, "old supplier"), RowWarning, []string{"supplier:SUPPLIER_ARCHIVED:WARNING"}},
		{"required missing", rowWith(28, ColProvenance, " ", ColChassis, ""), RowError,
			[]string{"chassis_number:REQUIRED:ERROR", "provenance:REQUIRED:ERROR"}},
		{"too long", rowWith(29, ColNotes, strings.Repeat("x", 301)), RowError, []string{"notes:TOO_LONG:ERROR"}},
		{"control chars", rowWith(30, ColNotes, "a\nb"), RowError, []string{"notes:INVALID_CHARACTERS:ERROR"}},
		{"existing chassis", rowWith(31, ColChassis, "ch-exist"), RowError, []string{"chassis_number:DUPLICATE_CHASSIS_EXISTING:ERROR"}},
		{"chassis on draft", rowWith(32, ColChassis, "CH-DRAFT"), RowError, []string{"chassis_number:DUPLICATE_CHASSIS_ON_RECEIPT:ERROR"}},
		{"existing motor", rowWith(33, ColMotor, "mot-exist"), RowWarning, []string{"motor_serial:POSSIBLE_DUPLICATE_SERIAL_EXISTING:WARNING"}},
		{"field count", SourceRow{Line: 34, Values: rowWith(34).Values, FieldCountMismatch: true, Cells: 19}, RowError, []string{":ROW_FIELD_COUNT:ERROR"}},
	}
	rows := make([]SourceRow, len(cases))
	for i, c := range cases {
		rows[i] = c.row
	}
	// In-file duplicates: two rows share a chassis (different case/spaces) and two a battery serial.
	rows = append(rows, rowWith(40, ColChassis, "CH-TWIN", ColBattery, "BAT-1"), rowWith(41, ColChassis, " ch-twin", ColBattery, "bat-1 "))
	res := Validate(rows, lookups(), allScope{allowed: []string{"l1", "l9"}}, bizDate)
	for i, c := range cases {
		r := res[i]
		got := issueCodes(r)
		if r.Status != c.status || !slices.Equal(got, nonNil(c.want)) || r.Line != c.row.Line {
			t.Errorf("%s: status %s issues %v, want %s %v", c.name, r.Status, got, c.status, c.want)
		}
		if r.CanProceed != (r.Status == RowValid || r.Status == RowWarning) {
			t.Errorf("%s: canProceed %v", c.name, r.CanProceed)
		}
	}
	v := res[0]
	if *v.ChassisNumber != "ch-valid" || *v.ChassisNormalised != "CH-VALID" || *v.BrandID != "b-ok" || *v.VariantID != "v-ok" ||
		*v.ColourID != "c-red" || *v.LocationID != "l1" || *v.SupplierID != "s1" || *v.ReceivedDate != "2026-01-15" ||
		*v.AcquisitionCost != "95000.00" || *v.StockStatus != "AVAILABLE" || v.Source[ColChassis] != " ch-valid " {
		t.Errorf("valid row normalisation %+v", v)
	}
	if res[0].Unresolved || !res[2].Unresolved || !res[5].Unresolved || res[22].Unresolved {
		t.Errorf("unresolved flags")
	}
	a, b := res[len(res)-2], res[len(res)-1]
	for _, r := range []RowResult{a, b} {
		if r.Status != RowError || !r.Duplicate || !slices.Contains(issueCodes(r), "chassis_number:DUPLICATE_CHASSIS_IN_FILE:ERROR") ||
			!slices.Contains(issueCodes(r), "battery_serial:POSSIBLE_DUPLICATE_SERIAL_IN_FILE:WARNING") {
			t.Errorf("in-file duplicate line %d: %v", r.Line, issueCodes(r))
		}
	}
	if !strings.Contains(a.Issues[0].Message, "41") || !strings.Contains(b.Issues[0].Message, "40") {
		t.Errorf("duplicate messages do not name the other line: %q / %q", a.Issues[0].Message, b.Issues[0].Message)
	}
	// A day-month-year date gets a safe suggestion, never an automatic correction.
	if s := res[18].Issues[0].Suggestion; !strings.Contains(s, "2026-01-15") || res[18].ReceivedDate != nil {
		t.Errorf("date suggestion %q", s)
	}
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
