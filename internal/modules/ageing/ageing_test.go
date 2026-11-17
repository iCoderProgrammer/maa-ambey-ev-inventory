package ageing

import (
	"slices"
	"testing"
	"time"
)

// Test-only band values (the LED-008 recommended set and a non-standard one);
// nothing here is a production threshold.

func days(n int) *int { return &n }

var led008 = &BandSet{Version: 1, Status: SetProposed, Bands: []Band{
	{Key: "0-30", Label: "0–30 days", MinDays: 0, MaxDays: days(30)},
	{Key: "31-60", Label: "31–60 days", MinDays: 31, MaxDays: days(60)},
	{Key: "61-90", Label: "61–90 days", MinDays: 61, MaxDays: days(90)},
	{Key: "over-90", Label: "Over 90 days", MinDays: 91},
}}

var asOf = time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)

func grn(daysAgo int) Origin {
	return Origin{SourceType: "GOODS_RECEIPT", Date: asOf.AddDate(0, 0, -daysAgo).Format(time.DateOnly), GRNStatus: "POSTED",
		GRNDate: asOf.AddDate(0, 0, -daysAgo).Format(time.DateOnly)}
}

// #6-#10: exact age, zero-day, every band at its lower and upper boundary and one day either side.
func TestEvaluateBoundaries(t *testing.T) {
	for age, want := range map[int]string{0: "0-30", 1: "0-30", 29: "0-30", 30: "0-30", 31: "31-60", 32: "31-60", 59: "31-60",
		60: "31-60", 61: "61-90", 62: "61-90", 89: "61-90", 90: "61-90", 91: "over-90", 92: "over-90", 36500: "over-90"} {
		r := Evaluate(grn(age), asOf, led008)
		if r.Status != StatusAged || *r.AgeDays != age || *r.BandKey != want || *r.OriginSource != SourceGRN ||
			r.MissingReason != nil || r.AsOf != "2026-06-30" || *r.OriginDate != grn(age).Date {
			t.Errorf("age %d: %+v", age, r)
		}
	}
	// A non-standard configuration (single-day band, wide band): same inclusive rule.
	custom := &BandSet{Bands: []Band{{Key: "today", Label: "Today", MinDays: 0, MaxDays: days(0)},
		{Key: "week", Label: "Week", MinDays: 1, MaxDays: days(7)}, {Key: "later", Label: "Later", MinDays: 8}}}
	for age, want := range map[int]string{0: "today", 1: "week", 7: "week", 8: "later"} {
		if r := Evaluate(grn(age), asOf, custom); *r.BandKey != want {
			t.Errorf("custom age %d: %v", age, *r.BandKey)
		}
	}
}

// #13: without a configuration the age is still known, with no band.
func TestEvaluateWithoutConfiguration(t *testing.T) {
	r := Evaluate(grn(12), asOf, nil)
	if r.Status != StatusAged || *r.AgeDays != 12 || r.BandKey != nil || r.BandLabel != nil {
		t.Fatalf("%+v", r)
	}
}

// #1-#5: the approved origins, and every unreliable or missing case (no age, no band, a reason).
func TestOriginRules(t *testing.T) {
	opening := Origin{SourceType: "OPENING_STOCK", Date: "2026-04-01", OpeningApproved: true}
	if r := Evaluate(opening, asOf, led008); r.Status != StatusAged || *r.OriginSource != SourceOpeningStock || *r.AgeDays != 90 ||
		*r.BandKey != "61-90" {
		t.Errorf("approved opening stock: %+v", r)
	}
	for name, c := range map[string]struct {
		o      Origin
		reason string
	}{
		"opening not approved": {Origin{SourceType: "OPENING_STOCK", Date: "2026-04-01"}, ReasonOpeningNotApproved},
		"opening without date": {Origin{SourceType: "OPENING_STOCK", OpeningApproved: true}, ReasonNoOriginDate},
		"receipt not posted":   {Origin{SourceType: "GOODS_RECEIPT", Date: "2026-06-01", GRNStatus: "DRAFT", GRNDate: "2026-06-01"}, ReasonGRNNotPosted},
		"receipt missing":      {Origin{SourceType: "GOODS_RECEIPT", Date: "2026-06-01"}, ReasonGRNNotPosted},
		"date mismatch":        {Origin{SourceType: "GOODS_RECEIPT", Date: "2026-06-01", GRNStatus: "POSTED", GRNDate: "2026-06-02"}, ReasonGRNDateMismatch},
		"no date":              {Origin{SourceType: "GOODS_RECEIPT", GRNStatus: "POSTED"}, ReasonNoOriginDate},
		"future":               {grn(-1), ReasonFutureOriginDate},
		"unknown source":       {Origin{SourceType: "MANUAL", Date: "2026-06-01"}, ReasonUnsupportedSource},
		"bad date":             {Origin{SourceType: "GOODS_RECEIPT", Date: "01/06/2026", GRNStatus: "POSTED", GRNDate: "01/06/2026"}, ReasonNoOriginDate},
	} {
		r := Evaluate(c.o, asOf, led008)
		if r.Status != StatusMissingOrigin || r.AgeDays != nil || r.BandKey != nil || r.OriginSource != nil ||
			r.MissingReason == nil || *r.MissingReason != c.reason || r.MissingExplanation == nil || *r.MissingExplanation == "" {
			t.Errorf("%s: %+v", name, r)
		}
		if !slices.Contains(Reasons, c.reason) || ReasonText[c.reason] == "" {
			t.Errorf("%s: undocumented reason", name)
		}
	}
	// Received on the business date: 0 days, never negative.
	if r := Evaluate(grn(0), asOf, led008); *r.AgeDays != 0 {
		t.Errorf("same day %+v", r)
	}
}

// #11, #12: invalid configurations are refused field by field; valid ones pass.
func TestValidateBands(t *testing.T) {
	if p := ValidateBands(led008.Bands); len(p) != 0 {
		t.Fatalf("LED-008 set refused: %+v", p)
	}
	if p := ValidateBands([]Band{{Key: "all", Label: "All", MinDays: 0}}); len(p) != 0 {
		t.Fatalf("single open band refused: %+v", p)
	}
	tooMany := make([]Band, MaxBands+1)
	for name, c := range map[string]struct {
		bands []Band
		field string
	}{
		"empty":          {nil, "bands"},
		"too many":       {tooMany, "bands"},
		"overlap":        {[]Band{{Key: "a", Label: "A", MaxDays: days(30)}, {Key: "b", Label: "B", MinDays: 30}}, "bands[1].minDays"},
		"gap":            {[]Band{{Key: "a", Label: "A", MaxDays: days(30)}, {Key: "b", Label: "B", MinDays: 32}}, "bands[1].minDays"},
		"not from zero":  {[]Band{{Key: "a", Label: "A", MinDays: 5}}, "bands[0].minDays"},
		"open not last":  {[]Band{{Key: "a", Label: "A"}, {Key: "b", Label: "B", MinDays: 1}}, "bands[0].maxDays"},
		"bounded last":   {[]Band{{Key: "a", Label: "A", MaxDays: days(10)}}, "bands[0].maxDays"},
		"max below min":  {[]Band{{Key: "a", Label: "A", MaxDays: days(5)}, {Key: "b", Label: "B", MinDays: 6, MaxDays: days(4)}, {Key: "c", Label: "C", MinDays: 5}}, "bands[1].maxDays"},
		"duplicate key":  {[]Band{{Key: "a", Label: "A", MaxDays: days(5)}, {Key: "a", Label: "B", MinDays: 6}}, "bands[1].key"},
		"bad key":        {[]Band{{Key: "Old Stock", Label: "A"}}, "bands[0].key"},
		"empty label":    {[]Band{{Key: "a", Label: " "}}, "bands[0].label"},
		"negative":       {[]Band{{Key: "a", Label: "A", MinDays: -1}}, "bands[0].minDays"},
		"beyond 100 yrs": {[]Band{{Key: "a", Label: "A", MaxDays: days(MaxDays + 1)}, {Key: "b", Label: "B", MinDays: MaxDays + 2}}, "bands[0].maxDays"},
	} {
		p := ValidateBands(c.bands)
		if !slices.ContainsFunc(p, func(x Problem) bool { return x.Field == c.field }) {
			t.Errorf("%s: %+v, want a problem on %s", name, p, c.field)
		}
	}
}

// Business date: the configured zone decides the calendar day (UTC evening = next day in India).
func TestCalendarUsesTheBusinessTimeZone(t *testing.T) {
	clock := time.Date(2026, 11, 16, 20, 0, 0, 0, time.UTC) // 01:30 on 17 Nov in Asia/Kolkata
	ist, err := NewCalendar("", func() time.Time { return clock })
	if err != nil {
		t.Fatal(err)
	}
	if ist.Zone() != DefaultTimeZone || ist.Today().Format(time.DateOnly) != "2026-11-17" {
		t.Fatalf("Asia/Kolkata today = %s (%s)", ist.Today().Format(time.DateOnly), ist.Zone())
	}
	utc, _ := NewCalendar("UTC", func() time.Time { return clock })
	if utc.Today().Format(time.DateOnly) != "2026-11-16" {
		t.Fatalf("UTC today = %s", utc.Today())
	}
	if _, err := NewCalendar("Mars/Olympus", nil); err == nil {
		t.Fatal("unknown zone accepted")
	}
	// Daylight-saving zones still count whole calendar days.
	ny, _ := NewCalendar("America/New_York", func() time.Time { return time.Date(2026, 3, 9, 15, 0, 0, 0, time.UTC) })
	if d := DaysBetween(time.Date(2026, 3, 7, 0, 0, 0, 0, time.UTC), ny.Today()); d != 2 {
		t.Fatalf("across DST: %d days", d)
	}
}
