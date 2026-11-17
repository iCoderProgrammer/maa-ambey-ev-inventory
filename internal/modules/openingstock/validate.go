package openingstock

import (
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// Row statuses.
const (
	RowValid   = "VALID"
	RowWarning = "WARNING"
	RowError   = "ERROR"
	RowBlocked = "BLOCKED_UNVERIFIED_MAPPING"
)

// Issue severities. ERROR: the row cannot be imported as uploaded. BLOCKED: it
// maps to a catalogue entry that is not verified/active, so it cannot become
// stock until the catalogue is approved. WARNING: importable, but needs review.
const (
	SevError   = "ERROR"
	SevBlocked = "BLOCKED"
	SevWarning = "WARNING"
)

// Issue codes of a row.
const (
	CodeRowFieldCount         = "ROW_FIELD_COUNT"
	CodeRequired              = "REQUIRED"
	CodeTooLong               = "TOO_LONG"
	CodeInvalidCharacters     = "INVALID_CHARACTERS"
	CodeUnknownBrand          = "UNKNOWN_BRAND"
	CodeAmbiguousBrand        = "AMBIGUOUS_BRAND"
	CodeBrandNotActive        = "BRAND_NOT_ACTIVE"
	CodeBrandArchived         = "BRAND_ARCHIVED"
	CodeUnknownModel          = "UNKNOWN_MODEL"
	CodeModelNotInBrand       = "MODEL_NOT_IN_BRAND"
	CodeModelNotActive        = "MODEL_NOT_ACTIVE"
	CodeUnknownVariant        = "UNKNOWN_VARIANT"
	CodeVariantNotInModel     = "VARIANT_NOT_IN_MODEL"
	CodeVariantNotActive      = "VARIANT_NOT_ACTIVE"
	CodeUnknownColour         = "UNKNOWN_COLOUR"
	CodeColourNotActive       = "COLOUR_NOT_ACTIVE"
	CodeColourNotConfigured   = "COLOUR_NOT_CONFIGURED"
	CodeUnknownLocation       = "UNKNOWN_LOCATION"
	CodeLocationInactive      = "LOCATION_INACTIVE"
	CodeLocationOutOfScope    = "LOCATION_OUT_OF_SCOPE"
	CodeInvalidStatus         = "INVALID_STATUS"
	CodeInvalidDate           = "INVALID_DATE"
	CodeFutureDate            = "FUTURE_DATE"
	CodeOriginDateMissing     = "ORIGIN_DATE_MISSING"
	CodeInvalidCost           = "INVALID_COST"
	CodeCostMissing           = "COST_MISSING"
	CodeSupplierNotFound      = "SUPPLIER_NOT_FOUND"
	CodeSupplierArchived      = "SUPPLIER_ARCHIVED"
	CodeDuplicateChassisFile  = "DUPLICATE_CHASSIS_IN_FILE"
	CodeDuplicateChassisDB    = "DUPLICATE_CHASSIS_EXISTING"
	CodeDuplicateChassisGRN   = "DUPLICATE_CHASSIS_ON_RECEIPT"
	CodePossibleDupSerialFile = "POSSIBLE_DUPLICATE_SERIAL_IN_FILE"
	CodePossibleDupSerialDB   = "POSSIBLE_DUPLICATE_SERIAL_EXISTING"
)

var mappingCodes = []string{CodeUnknownBrand, CodeAmbiguousBrand, CodeBrandNotActive, CodeBrandArchived, CodeUnknownModel,
	CodeModelNotInBrand, CodeModelNotActive, CodeUnknownVariant, CodeVariantNotInModel, CodeVariantNotActive, CodeUnknownColour,
	CodeColourNotActive, CodeUnknownLocation, CodeLocationInactive, CodeLocationOutOfScope}

var duplicateCodes = []string{CodeDuplicateChassisFile, CodeDuplicateChassisDB, CodeDuplicateChassisGRN, CodePossibleDupSerialFile,
	CodePossibleDupSerialDB}

// Issue is one problem with one cell (or the row, when Field is empty).
type Issue struct {
	Field      string `json:"field" example:"chassis_number"`
	Code       string `json:"code" example:"DUPLICATE_CHASSIS_IN_FILE"`
	Severity   string `json:"severity" enums:"ERROR,BLOCKED,WARNING" example:"ERROR"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion,omitempty"`
} //@name OpeningStockRowIssue

// Catalogue, location, supplier and identifier facts the validation reads.
type (
	BrandRef struct {
		ID, Name, Lifecycle, Authorisation string
	}
	ModelRef struct {
		ID, BrandID, Code, Name, Lifecycle string
	}
	VariantRef struct {
		ID, BrandID, ModelID, Code, Name, Lifecycle string
	}
	ColourRef struct {
		ID, VariantID, Name, Lifecycle string
		Code                           *string
	}
	LocationRef struct {
		ID, Code, Name string
		Active         bool
	}
	SupplierRef struct {
		ID, LegalName, Status string
		TradeName             *string
	}
)

// Lookups is everything the rows reference, read in bulk by the repository.
type Lookups struct {
	Brands                                          []BrandRef
	Models                                          []ModelRef
	Variants                                        []VariantRef
	Colours                                         []ColourRef
	Locations                                       []LocationRef
	Suppliers                                       []SupplierRef
	ExistingChassis, OpenReceiptChassis             map[string]bool
	ExistingMotor, ExistingBattery, ExistingCharger map[string]bool
}

// Refs are the values to look up.
type Refs struct {
	BrandNames, ModelCodes, VariantCodes, LocationCodes, SupplierNames []string
	Chassis, Motors, Batteries, Chargers                               []string
}

// NormaliseIdentifier is the comparison form of a chassis/serial: trimmed and
// upper-cased, exactly as vehicle units (D-04/D-05). Which separators may also
// be ignored is pending (OQ-26).
func NormaliseIdentifier(s string) string { return strings.ToUpper(strings.TrimSpace(s)) }

func cell(r SourceRow, col string) string { return strings.TrimSpace(r.Values[col]) }

// CollectRefs lists the values Validate will need.
func CollectRefs(rows []SourceRow) Refs {
	var f Refs
	add := func(dst *[]string, v string) {
		if v != "" && !slices.Contains(*dst, v) {
			*dst = append(*dst, v)
		}
	}
	for _, r := range rows {
		add(&f.BrandNames, strings.ToLower(cell(r, ColBrand)))
		add(&f.ModelCodes, strings.ToUpper(cell(r, ColModelCode)))
		add(&f.VariantCodes, strings.ToUpper(cell(r, ColVariantCode)))
		add(&f.LocationCodes, strings.ToUpper(cell(r, ColLocation)))
		add(&f.SupplierNames, strings.ToLower(cell(r, ColSupplier)))
		add(&f.Chassis, NormaliseIdentifier(r.Values[ColChassis]))
		add(&f.Motors, NormaliseIdentifier(r.Values[ColMotor]))
		add(&f.Batteries, NormaliseIdentifier(r.Values[ColBattery]))
		add(&f.Chargers, NormaliseIdentifier(r.Values[ColCharger]))
	}
	return f
}

// RowResult is a validated row.
type RowResult struct {
	Line                                                  int
	Status                                                string
	CanProceed, Duplicate, Unresolved                     bool
	Source                                                map[string]string
	BrandID, ModelID, VariantID, ColourID, LocationID     *string
	SupplierID                                            *string
	ChassisNumber, ChassisNormalised                      *string
	MotorNormalised, BatteryNormalised, ChargerNormalised *string
	StockStatus, ReceivedDate, AcquisitionCost            *string
	Resolved                                              map[string]any
	Issues                                                []Issue
}

// Scope says which locations the uploader may stock.
type Scope interface{ Allows(locationID string) bool }

var (
	costPattern = regexp.MustCompile(`^\d{1,10}(\.\d{1,2})?$`)
	dmyPattern  = regexp.MustCompile(`^(\d{1,2})[-/.](\d{1,2})[-/.](\d{4})$`)
)

func ptr(s string) *string { return &s }

// Validate checks every row and returns them in file order. Nothing is
// corrected: values are only trimmed/normalised for comparison, and every
// problem is reported on its own cell.
func Validate(rows []SourceRow, l Lookups, scope Scope, businessDate time.Time) []RowResult {
	// Duplicates inside the file, by normalised identifier.
	lines := func(col string) map[string][]int {
		m := map[string][]int{}
		for _, r := range rows {
			if v := NormaliseIdentifier(r.Values[col]); v != "" {
				m[v] = append(m[v], r.Line)
			}
		}
		return m
	}
	inFile := map[string]map[string][]int{ColChassis: lines(ColChassis), ColMotor: lines(ColMotor), ColBattery: lines(ColBattery),
		ColCharger: lines(ColCharger)}
	existing := map[string]map[string]bool{ColMotor: l.ExistingMotor, ColBattery: l.ExistingBattery, ColCharger: l.ExistingCharger}

	out := make([]RowResult, 0, len(rows))
	for _, r := range rows {
		res := RowResult{Line: r.Line, Source: map[string]string{}, Resolved: map[string]any{}}
		for _, c := range Columns {
			res.Source[c.Name] = r.Values[c.Name]
		}
		add := func(field, code, sev, msg, suggestion string) {
			res.Issues = append(res.Issues, Issue{Field: field, Code: code, Severity: sev, Message: msg, Suggestion: suggestion})
		}
		if r.FieldCountMismatch {
			add("", CodeRowFieldCount, SevError, "This line has "+strconv.Itoa(r.Cells)+" cells but the header has "+
				strconv.Itoa(len(r.Values))+"; a value may contain an unquoted comma.", "Check the line in the spreadsheet and save the file again.")
		}
		// Cell format: required, length and control characters.
		ok := map[string]bool{}
		for _, c := range Columns {
			v := cell(r, c.Name)
			switch {
			case v == "" && c.Required:
				add(c.Name, CodeRequired, SevError, c.Name+" is required.", "")
			case v == "":
			case utf8.RuneCountInString(v) > c.MaxLength:
				add(c.Name, CodeTooLong, SevError, c.Name+" must be at most "+strconv.Itoa(c.MaxLength)+" characters.", "")
			case strings.ContainsFunc(v, unicode.IsControl):
				add(c.Name, CodeInvalidCharacters, SevError, c.Name+" contains line breaks or control characters.", "")
			default:
				ok[c.Name] = true
			}
		}

		// Brand → model → variant → colour, never guessed.
		var brand *BrandRef
		if ok[ColBrand] {
			name := cell(r, ColBrand)
			var found []BrandRef
			for _, b := range l.Brands {
				if strings.EqualFold(b.Name, name) {
					found = append(found, b)
				}
			}
			switch len(found) {
			case 0:
				add(ColBrand, CodeUnknownBrand, SevError, "No brand named \""+name+"\" exists in the catalogue.",
					"Use the brand name exactly as in the catalogue, or ask a catalogue administrator to add the brand.")
			case 1:
				brand = &found[0]
				res.BrandID = &brand.ID
				res.Resolved["brand"] = map[string]any{"id": brand.ID, "name": brand.Name, "lifecycleStatus": brand.Lifecycle,
					"authorisationStatus": brand.Authorisation}
				switch {
				case brand.Lifecycle == "ARCHIVED":
					add(ColBrand, CodeBrandArchived, SevBlocked, "Brand "+brand.Name+" is archived.", "")
				case brand.Lifecycle != "ACTIVE" || brand.Authorisation != "AUTHORISED":
					add(ColBrand, CodeBrandNotActive, SevBlocked, "Brand "+brand.Name+" exists but is not verified and activated ("+
						brand.Lifecycle+", "+brand.Authorisation+"): its stock cannot be imported until the brand is approved.", "")
				}
			default:
				add(ColBrand, CodeAmbiguousBrand, SevError, "More than one brand matches \""+name+"\".", "")
			}
		}
		var model *ModelRef
		if brand != nil && ok[ColModelCode] {
			code := cell(r, ColModelCode)
			var other *ModelRef
			for i, m := range l.Models {
				if !strings.EqualFold(m.Code, code) {
					continue
				}
				if m.BrandID == brand.ID {
					model = &l.Models[i]
				} else if other == nil {
					other = &l.Models[i]
				}
			}
			switch {
			case model != nil:
				res.ModelID = &model.ID
				res.Resolved["model"] = map[string]any{"id": model.ID, "name": model.Name, "code": model.Code, "lifecycleStatus": model.Lifecycle}
				if model.Lifecycle != "ACTIVE" {
					add(ColModelCode, CodeModelNotActive, SevBlocked, "Model "+model.Name+" exists but is not active ("+model.Lifecycle+").", "")
				}
			case other != nil:
				add(ColModelCode, CodeModelNotInBrand, SevError, "Model code \""+code+"\" belongs to another brand, not "+brand.Name+".",
					"Check the brand and model code of this vehicle.")
			default:
				add(ColModelCode, CodeUnknownModel, SevError, "Brand "+brand.Name+" has no model with code \""+code+"\".",
					"Use the model's internal code from the catalogue.")
			}
		}
		var variant *VariantRef
		if model != nil && ok[ColVariantCode] {
			code := cell(r, ColVariantCode)
			otherModel := false
			for i, v := range l.Variants {
				if !strings.EqualFold(v.Code, code) {
					continue
				}
				if v.ModelID == model.ID {
					variant = &l.Variants[i]
				} else if v.BrandID == model.BrandID {
					otherModel = true
				}
			}
			switch {
			case variant != nil:
				res.VariantID = &variant.ID
				res.Resolved["variant"] = map[string]any{"id": variant.ID, "name": variant.Name, "code": variant.Code,
					"lifecycleStatus": variant.Lifecycle}
				if variant.Lifecycle != "ACTIVE" {
					add(ColVariantCode, CodeVariantNotActive, SevBlocked, "Variant "+variant.Name+" exists but is not active ("+variant.Lifecycle+").", "")
				}
			case otherModel:
				add(ColVariantCode, CodeVariantNotInModel, SevError, "Variant code \""+code+"\" belongs to another model of this brand, not "+model.Name+".", "")
			default:
				add(ColVariantCode, CodeUnknownVariant, SevError, "Model "+model.Name+" has no variant with code \""+code+"\".", "")
			}
		}
		if variant != nil && ok[ColColour] {
			v := cell(r, ColColour)
			var configured []ColourRef
			for _, c := range l.Colours {
				if c.VariantID == variant.ID {
					configured = append(configured, c)
				}
			}
			var match *ColourRef
			for i, c := range configured {
				if strings.EqualFold(c.Name, v) || (c.Code != nil && strings.EqualFold(*c.Code, v)) {
					match = &configured[i]
				}
			}
			switch {
			case len(configured) == 0:
				add(ColColour, CodeColourNotConfigured, SevWarning, "Variant "+variant.Name+" has no colours in the catalogue, so \""+v+
					"\" cannot be checked.", "")
			case match == nil:
				add(ColColour, CodeUnknownColour, SevError, "Variant "+variant.Name+" has no colour \""+v+"\".", "")
			default:
				res.ColourID = &match.ID
				res.Resolved["colour"] = map[string]any{"id": match.ID, "name": match.Name, "lifecycleStatus": match.Lifecycle}
				if match.Lifecycle != "ACTIVE" {
					add(ColColour, CodeColourNotActive, SevBlocked, "Colour "+match.Name+" exists but is not active ("+match.Lifecycle+").", "")
				}
			}
		}

		// Location: known, active and in the uploader's scope.
		if ok[ColLocation] {
			code := cell(r, ColLocation)
			i := slices.IndexFunc(l.Locations, func(x LocationRef) bool { return strings.EqualFold(x.Code, code) })
			switch {
			case i < 0:
				add(ColLocation, CodeUnknownLocation, SevError, "No location has code \""+code+"\".", "")
			case !l.Locations[i].Active:
				add(ColLocation, CodeLocationInactive, SevError, "Location "+l.Locations[i].Code+" is not active.", "")
			case !scope.Allows(l.Locations[i].ID):
				add(ColLocation, CodeLocationOutOfScope, SevError, "Location "+l.Locations[i].Code+" is not one of your locations.", "")
			default:
				loc := l.Locations[i]
				res.LocationID = &loc.ID
				res.Resolved["location"] = map[string]any{"id": loc.ID, "code": loc.Code, "name": loc.Name}
			}
		}

		// Status.
		if ok[ColStatus] {
			s := strings.ToUpper(cell(r, ColStatus))
			if slices.Contains(StockStatuses, s) {
				res.StockStatus = ptr(s)
			} else {
				add(ColStatus, CodeInvalidStatus, SevError, "\""+cell(r, ColStatus)+"\" is not an opening stock status.",
					"Use one of "+strings.Join(StockStatuses, ", ")+".")
			}
		}

		// Received date: optional provenance; never invented, never in the future.
		if ok[ColReceivedDate] {
			v := cell(r, ColReceivedDate)
			d, err := time.Parse(time.DateOnly, v)
			switch {
			case err != nil:
				suggestion := "Write the date as YYYY-MM-DD."
				if m := dmyPattern.FindStringSubmatch(v); m != nil {
					suggestion = "Write the date as YYYY-MM-DD, e.g. " + m[3] + "-" + pad2(m[2]) + "-" + pad2(m[1]) + " if this is day-month-year."
				}
				add(ColReceivedDate, CodeInvalidDate, SevError, "\""+v+"\" is not a valid date.", suggestion)
			case d.After(businessDate):
				add(ColReceivedDate, CodeFutureDate, SevError, "The received date is after today's business date ("+
					businessDate.Format(time.DateOnly)+").", "")
			default:
				res.ReceivedDate = ptr(d.Format(time.DateOnly))
			}
		} else if cell(r, ColReceivedDate) == "" {
			add(ColReceivedDate, CodeOriginDateMissing, SevWarning, "No received date: the vehicle will have no stock age until an "+
				"opening-stock date is approved (E-07).", "")
		}

		// Cost: optional, never assumed.
		if ok[ColCost] {
			v := strings.ReplaceAll(cell(r, ColCost), ",", "")
			if costPattern.MatchString(v) {
				res.AcquisitionCost = ptr(v)
			} else {
				add(ColCost, CodeInvalidCost, SevError, "\""+cell(r, ColCost)+"\" is not an amount (digits, up to 2 decimals, no currency sign).", "")
			}
		} else if cell(r, ColCost) == "" {
			add(ColCost, CodeCostMissing, SevWarning, "No acquisition cost: an approved cost basis is needed before commit (OQ-29).", "")
		}

		// Supplier: optional provenance; reported, not required.
		if ok[ColSupplier] {
			v := cell(r, ColSupplier)
			i := slices.IndexFunc(l.Suppliers, func(s SupplierRef) bool {
				return strings.EqualFold(s.LegalName, v) || (s.TradeName != nil && strings.EqualFold(*s.TradeName, v))
			})
			switch {
			case i < 0:
				add(ColSupplier, CodeSupplierNotFound, SevWarning, "No supplier named \""+v+"\"; it is kept as text only.", "")
			default:
				s := l.Suppliers[i]
				res.SupplierID = &s.ID
				res.Resolved["supplier"] = map[string]any{"id": s.ID, "name": s.LegalName, "status": s.Status}
				if s.Status == "ARCHIVED" {
					add(ColSupplier, CodeSupplierArchived, SevWarning, "Supplier "+s.LegalName+" is archived.", "")
				}
			}
		}

		// Identifiers: chassis duplicates are fatal (IMP-002); motor/battery/charger
		// uniqueness is not approved (OQ-4), so their duplicates are warnings.
		if ok[ColChassis] {
			raw := cell(r, ColChassis)
			n := NormaliseIdentifier(raw)
			res.ChassisNumber, res.ChassisNormalised = ptr(raw), ptr(n)
			if others := without(inFile[ColChassis][n], r.Line); len(others) > 0 {
				add(ColChassis, CodeDuplicateChassisFile, SevError, "Chassis/frame "+n+" also appears on line(s) "+joinInts(others)+" of this file.", "")
			}
			if l.ExistingChassis[n] {
				add(ColChassis, CodeDuplicateChassisDB, SevError, "Chassis/frame "+n+" is already registered to a vehicle unit.", "")
			} else if l.OpenReceiptChassis[n] {
				add(ColChassis, CodeDuplicateChassisGRN, SevError, "Chassis/frame "+n+" is captured on an open goods receipt draft.", "")
			}
		}
		for _, col := range []string{ColMotor, ColBattery, ColCharger} {
			if !ok[col] {
				continue
			}
			n := NormaliseIdentifier(r.Values[col])
			switch col {
			case ColMotor:
				res.MotorNormalised = ptr(n)
			case ColBattery:
				res.BatteryNormalised = ptr(n)
			case ColCharger:
				res.ChargerNormalised = ptr(n)
			}
			if others := without(inFile[col][n], r.Line); len(others) > 0 {
				add(col, CodePossibleDupSerialFile, SevWarning, col+" "+n+" also appears on line(s) "+joinInts(others)+
					" (uniqueness rules for this serial are pending approval).", "")
			}
			if existing[col][n] {
				add(col, CodePossibleDupSerialDB, SevWarning, col+" "+n+" is already recorded on a vehicle unit (uniqueness rules pending approval).", "")
			}
		}

		res.Status = RowValid
		for _, is := range res.Issues {
			if slices.Contains(mappingCodes, is.Code) {
				res.Unresolved = true
			}
			if slices.Contains(duplicateCodes, is.Code) {
				res.Duplicate = true
			}
		}
		switch {
		case slices.ContainsFunc(res.Issues, func(i Issue) bool { return i.Severity == SevError }):
			res.Status = RowError
		case slices.ContainsFunc(res.Issues, func(i Issue) bool { return i.Severity == SevBlocked }):
			res.Status = RowBlocked
		case len(res.Issues) > 0:
			res.Status = RowWarning
		}
		res.CanProceed = res.Status == RowValid || res.Status == RowWarning
		if res.Issues == nil {
			res.Issues = []Issue{}
		}
		out = append(out, res)
	}
	return out
}

func pad2(s string) string {
	if len(s) == 1 {
		return "0" + s
	}
	return s
}

func without(lines []int, line int) []int {
	var out []int
	for _, l := range lines {
		if l != line {
			out = append(out, l)
		}
	}
	sort.Ints(out)
	return out
}

func joinInts(ns []int) string {
	s := make([]string, len(ns))
	for i, n := range ns {
		s[i] = strconv.Itoa(n)
	}
	return strings.Join(s, ", ")
}
