// Package ageing is stock ageing (E-07; ADR-029; LED-008, REP-005): the age of
// a stock record in whole days from its approved origin date, and the
// configurable ageing bands it falls into.
//
// Rules:
//   - Origin date. Only an approved origin counts: the received date of the
//     unit's goods receipt once that receipt is POSTED (source GRN), or — once
//     opening stock exists (G-06/G-07) — an approved opening-stock date (source
//     OPENING_STOCK). Never the registration time, PO date, PDI date, today or
//     any fallback. A record without a reliable origin has no age and no band;
//     it is reported as MISSING_ORIGIN with a reason.
//   - Business date. Ages are counted on the business calendar: today is the
//     current date in the configured business time zone (BUSINESS_TIME_ZONE,
//     default Asia/Kolkata). Origin dates are date-only. Age = business date −
//     origin date in whole calendar days (no partial days; received today = 0).
//     An origin after the business date is not used (FUTURE_ORIGIN_DATE).
//   - Bands. A band covers MinDays <= age <= MaxDays (both inclusive); the last
//     band is open-ended. A set starts at 0 with no gap or overlap. Bands are
//     applied when the age is read; nothing derived from them is stored, so a
//     new configuration never rewrites stock history.
package ageing

import (
	"regexp"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata" // the business time zone must load in minimal containers too
	"unicode/utf8"
)

// Origin sources (approved origins only).
const (
	SourceGRN          = "GRN"
	SourceOpeningStock = "OPENING_STOCK"
)

// Statuses of an ageing result.
const (
	// StatusAged: the origin is reliable and the age is known (the band may still be
	// null when no band configuration exists).
	StatusAged = "AGED"
	// StatusMissingOrigin: no reliable origin date; no age, no band.
	StatusMissingOrigin = "MISSING_ORIGIN"
)

// Reasons an origin date is missing or not reliable.
const (
	ReasonUnsupportedSource  = "UNSUPPORTED_SOURCE"   // the record did not come from an approved receipt or opening stock
	ReasonGRNNotPosted       = "GRN_NOT_POSTED"       // its goods receipt is missing or not posted (not accepted)
	ReasonGRNDateMismatch    = "GRN_DATE_MISMATCH"    // its received date is not its goods receipt's
	ReasonOpeningNotApproved = "OPENING_NOT_APPROVED" // opening-stock date not approved
	ReasonNoOriginDate       = "NO_ORIGIN_DATE"       // no date recorded
	ReasonFutureOriginDate   = "FUTURE_ORIGIN_DATE"   // the origin date is after the business date
)

// Reasons lists every reason, for filters and documentation.
var Reasons = []string{ReasonUnsupportedSource, ReasonGRNNotPosted, ReasonGRNDateMismatch, ReasonOpeningNotApproved,
	ReasonNoOriginDate, ReasonFutureOriginDate}

// ReasonText explains a reason to a user.
var ReasonText = map[string]string{
	ReasonUnsupportedSource:  "This record did not come from a posted goods receipt or approved opening stock, so it has no approved origin date.",
	ReasonGRNNotPosted:       "Its goods receipt is missing or not posted, so the received date is not an accepted origin date.",
	ReasonGRNDateMismatch:    "Its received date does not match its goods receipt's received date, so it is not reliable.",
	ReasonOpeningNotApproved: "Its opening-stock date has not been approved.",
	ReasonNoOriginDate:       "No origin date is recorded.",
	ReasonFutureOriginDate:   "Its origin date is after today's business date, so no age can be calculated.",
}

// DefaultTimeZone is the business time zone when none is configured.
const DefaultTimeZone = "Asia/Kolkata"

// Calendar gives the business date.
type Calendar struct {
	loc *time.Location
	now func() time.Time
}

// NewCalendar loads the business time zone (an IANA name). now is the clock;
// nil means time.Now.
func NewCalendar(zone string, now func() time.Time) (Calendar, error) {
	if zone == "" {
		zone = DefaultTimeZone
	}
	loc, err := time.LoadLocation(zone)
	if err != nil {
		return Calendar{}, err
	}
	if now == nil {
		now = time.Now
	}
	return Calendar{loc: loc, now: now}, nil
}

// Zone is the business time zone name.
func (c Calendar) Zone() string { return c.loc.String() }

// Today is the business date (date only, UTC midnight of that calendar day).
func (c Calendar) Today() time.Time {
	y, m, d := c.now().In(c.loc).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// ParseDate reads a YYYY-MM-DD date as UTC midnight.
func ParseDate(s string) (time.Time, error) { return time.Parse(time.DateOnly, s) }

// DaysBetween is to − from in whole calendar days (both date-only).
func DaysBetween(from, to time.Time) int { return int(to.Sub(from).Hours() / 24) }

// Band is one ageing band: MinDays <= age <= MaxDays; MaxDays nil = open-ended.
type Band struct {
	Key     string `json:"key" example:"0-30"`
	Label   string `json:"label" example:"0–30 days"`
	MinDays int    `json:"minDays" example:"0"`
	MaxDays *int   `json:"maxDays" extensions:"x-nullable" example:"30"`
} //@name StockAgeingBand

// Contains reports whether age falls in the band.
func (b Band) Contains(age int) bool {
	return age >= b.MinDays && (b.MaxDays == nil || age <= *b.MaxDays)
}

// Set statuses.
const (
	SetProposed = "PROPOSED"
	SetApproved = "APPROVED"
)

// BandSet is one version of the band configuration.
type BandSet struct {
	Version int    `json:"version" example:"1"`
	Status  string `json:"status" enums:"PROPOSED,APPROVED" example:"PROPOSED"`
	Source  string `json:"source"`
	// CreatedByID/Name: null when recorded by the migration.
	CreatedByID       *string    `json:"createdById" extensions:"x-nullable"`
	CreatedByName     *string    `json:"createdByName" extensions:"x-nullable"`
	CreatedAt         time.Time  `json:"createdAt"`
	ApprovedByID      *string    `json:"approvedById" extensions:"x-nullable"`
	ApprovedByName    *string    `json:"approvedByName" extensions:"x-nullable"`
	ApprovedAt        *time.Time `json:"approvedAt" extensions:"x-nullable"`
	ApprovalReference *string    `json:"approvalReference" extensions:"x-nullable"`
	Bands             []Band     `json:"bands"`
} //@name StockAgeingBandSet

// Find returns the band with key.
func (s *BandSet) Find(key string) (Band, bool) {
	if s == nil {
		return Band{}, false
	}
	for _, b := range s.Bands {
		if b.Key == key {
			return b, true
		}
	}
	return Band{}, false
}

// For returns the band age falls in.
func (s *BandSet) For(age int) (Band, bool) {
	if s == nil {
		return Band{}, false
	}
	for _, b := range s.Bands {
		if b.Contains(age) {
			return b, true
		}
	}
	return Band{}, false
}

// MaxBands and MaxDays bound a configuration.
const (
	MaxBands = 20
	MaxDays  = 36500
)

var keyPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,31}$`)

// Problem is one invalid input field.
type Problem struct{ Field, Message string }

// ValidateBands checks a configuration: 1–20 bands, unique keys
// (lower-case letters, digits, - and _), labels of 1–60 characters, whole-day
// boundaries from 0 to 36500, starting at 0, each band starting the day after
// the previous one ends (no gap, no overlap), only the last one open-ended.
func ValidateBands(bands []Band) []Problem {
	var out []Problem
	bad := func(i int, field, msg string) {
		out = append(out, Problem{"bands[" + strconv.Itoa(i) + "]." + field, msg})
	}
	if len(bands) == 0 || len(bands) > MaxBands {
		return []Problem{{"bands", "must have 1 to " + strconv.Itoa(MaxBands) + " bands"}}
	}
	seen := map[string]bool{}
	expected := 0
	for i, b := range bands {
		if !keyPattern.MatchString(b.Key) {
			bad(i, "key", "must be 1-32 lower-case letters, digits, '-' or '_', starting with a letter or digit")
		} else if seen[b.Key] {
			bad(i, "key", "is used by another band")
		}
		seen[b.Key] = true
		if l := strings.TrimSpace(b.Label); l == "" || utf8.RuneCountInString(b.Label) > 60 || l != b.Label {
			bad(i, "label", "must be 1-60 characters without leading or trailing spaces")
		}
		if b.MinDays < 0 || b.MinDays > MaxDays {
			bad(i, "minDays", "must be a whole number of days from 0 to "+strconv.Itoa(MaxDays))
		} else if b.MinDays != expected {
			if i == 0 {
				bad(i, "minDays", "must be 0: the first band starts on the day of receipt")
			} else {
				bad(i, "minDays", "must be "+strconv.Itoa(expected)+", the day after the previous band ends (no gap or overlap)")
			}
		}
		last := i == len(bands)-1
		switch {
		case b.MaxDays == nil && !last:
			bad(i, "maxDays", "is required: only the last band is open-ended")
		case b.MaxDays != nil && last:
			bad(i, "maxDays", "must be empty: the last band is open-ended, so every age has a band")
		case b.MaxDays != nil && (*b.MaxDays < b.MinDays || *b.MaxDays > MaxDays):
			bad(i, "maxDays", "must be from minDays to "+strconv.Itoa(MaxDays))
		}
		if b.MaxDays != nil {
			expected = *b.MaxDays + 1
		}
	}
	return out
}

// Origin is what a stock record says about where it came from.
type Origin struct {
	// SourceType is the record's source ("GOODS_RECEIPT", later "OPENING_STOCK").
	SourceType string
	// Date is the record's origin date (YYYY-MM-DD); empty when none.
	Date string
	// GRNStatus and GRNDate are its goods receipt's status and received date (GOODS_RECEIPT).
	GRNStatus, GRNDate string
	// OpeningApproved: the opening-stock date is approved (OPENING_STOCK).
	OpeningApproved bool
}

// Result is the ageing of one record.
type Result struct {
	Status       string  `json:"status" enums:"AGED,MISSING_ORIGIN" example:"AGED"`
	OriginDate   *string `json:"originDate" extensions:"x-nullable" example:"2026-10-23"`
	OriginSource *string `json:"originSource" extensions:"x-nullable" enums:"GRN,OPENING_STOCK" example:"GRN"`
	// AgeDays: whole days from the origin date to AsOf; null without a reliable origin.
	AgeDays *int `json:"ageDays" extensions:"x-nullable" example:"24"`
	// BandKey/BandLabel: the band of AgeDays in the current configuration; null without
	// an age or without a configuration.
	BandKey   *string `json:"bandKey" extensions:"x-nullable" example:"0-30"`
	BandLabel *string `json:"bandLabel" extensions:"x-nullable" example:"0–30 days"`
	// MissingReason/MissingExplanation: why there is no reliable origin; null when AGED.
	MissingReason      *string `json:"missingReason" extensions:"x-nullable" enums:"UNSUPPORTED_SOURCE,GRN_NOT_POSTED,GRN_DATE_MISMATCH,OPENING_NOT_APPROVED,NO_ORIGIN_DATE,FUTURE_ORIGIN_DATE"`
	MissingExplanation *string `json:"missingExplanation" extensions:"x-nullable"`
	// AsOf is the business date the age is counted to.
	AsOf string `json:"asOf" example:"2026-11-16"`
} //@name StockAgeing

// OriginReason returns why o has no reliable origin date as of asOf ("" when it has one).
func OriginReason(o Origin, asOf time.Time) string {
	switch o.SourceType {
	case "GOODS_RECEIPT":
		if o.Date == "" {
			return ReasonNoOriginDate
		}
		if o.GRNStatus != "POSTED" {
			return ReasonGRNNotPosted
		}
		if o.GRNDate != o.Date {
			return ReasonGRNDateMismatch
		}
	case "OPENING_STOCK":
		if o.Date == "" {
			return ReasonNoOriginDate
		}
		if !o.OpeningApproved {
			return ReasonOpeningNotApproved
		}
	default:
		return ReasonUnsupportedSource
	}
	d, err := ParseDate(o.Date)
	if err != nil {
		return ReasonNoOriginDate
	}
	if d.After(asOf) {
		return ReasonFutureOriginDate
	}
	return ""
}

// Evaluate ages o as of the business date asOf with the band configuration set
// (nil: no configuration, so no band).
func Evaluate(o Origin, asOf time.Time, set *BandSet) Result {
	r := Result{AsOf: asOf.Format(time.DateOnly)}
	if reason := OriginReason(o, asOf); reason != "" {
		text := ReasonText[reason]
		r.Status, r.MissingReason, r.MissingExplanation = StatusMissingOrigin, &reason, &text
		if o.Date != "" {
			date := o.Date
			r.OriginDate = &date
		}
		return r
	}
	date, source := o.Date, SourceGRN
	if o.SourceType == "OPENING_STOCK" {
		source = SourceOpeningStock
	}
	d, _ := ParseDate(o.Date)
	age := DaysBetween(d, asOf)
	r.Status, r.OriginDate, r.OriginSource, r.AgeDays = StatusAged, &date, &source, &age
	if b, ok := set.For(age); ok {
		key, label := b.Key, b.Label
		r.BandKey, r.BandLabel = &key, &label
	}
	return r
}
