package catalogue

import (
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func blockedKeys(t *testing.T, err error) []string {
	t.Helper()
	var b *ActivationBlockedError
	if !errors.As(err, &b) {
		t.Fatalf("want ActivationBlockedError, got %v", err)
	}
	keys := make([]string, len(b.Unmet))
	for i, c := range b.Unmet {
		keys[i] = c.Key
	}
	return keys
}

func isState(err error) bool {
	var s *StateError
	return errors.As(err, &s)
}

func TestItemActivationRules(t *testing.T) {
	verified := ConfidenceSupplierConfirmed
	// Pending, verified model under a pending brand: allowed (brand activation comes last).
	if err := checkItemActivation(kindModel, StatusPending, verified, StatusPending); err != nil {
		t.Fatal(err)
	}
	// Unverified: blocked.
	if k := blockedKeys(t, checkItemActivation(kindModel, StatusPending, ConfidenceUnverified, StatusActive)); k[0] != "verification" {
		t.Fatal(k)
	}
	// Archived brand: blocked.
	if k := blockedKeys(t, checkItemActivation(kindModel, StatusPending, verified, StatusArchived)); k[0] != "parent" {
		t.Fatal(k)
	}
	// Variant needs an ACTIVE model; colour an ACTIVE variant.
	for _, parent := range []string{StatusPending, StatusArchived} {
		blockedKeys(t, checkItemActivation(kindVariant, StatusPending, verified, parent))
		blockedKeys(t, checkItemActivation(kindColour, StatusPending, verified, parent))
	}
	if err := checkItemActivation(kindVariant, StatusPending, verified, StatusActive); err != nil {
		t.Fatal(err)
	}
	// Both problems are reported together.
	if k := blockedKeys(t, checkItemActivation(kindColour, StatusPending, ConfidenceUnverified, StatusPending)); len(k) != 2 {
		t.Fatal(k)
	}
	// Already active / archived items are state errors, not activation checks.
	if !isState(checkItemActivation(kindModel, StatusActive, verified, StatusActive)) ||
		!isState(checkItemActivation(kindModel, StatusArchived, verified, StatusActive)) {
		t.Fatal("expected state errors")
	}
}

func TestBrandReadinessFailsClosed(t *testing.T) {
	s, ref, ev, user := "Legal Name Pvt Ltd", "REF-1", someID, someID
	now := time.Now()
	b := Brand{ID: someID, LifecycleStatus: StatusPending, AuthorisationStatus: AuthAuthorised,
		ManufacturerLegalName: &s, AuthorisationReference: &ref, AuthorisationEvidenceID: &ev, VerifiedBy: &user, VerifiedAt: &now}
	r := brandReadiness(brandFacts{Brand: b, HasActiveVariant: true, HasApprovedPrice: true})
	if r.Ready {
		t.Fatal("brand must not be ready while prerequisites from later tasks are missing")
	}
	for _, c := range r.Checks {
		if c.Met == c.Pending {
			t.Errorf("check %s: met=%v pending=%v; with all recordable evidence only pending checks should fail", c.Key, c.Met, c.Pending)
		}
	}
	keys := blockedKeys(t, checkBrandActivation(brandFacts{Brand: b, HasActiveVariant: true, HasApprovedPrice: true}))
	if len(keys) != len(pendingChecks) {
		t.Fatalf("unmet %v", keys)
	}

	// A bare roster brand fails every recordable check too.
	bare := brandReadiness(brandFacts{Brand: Brand{LifecycleStatus: StatusPending, AuthorisationStatus: AuthUnverified}})
	for _, c := range bare.Checks {
		if c.Met && c.Key != "lifecycle" {
			t.Errorf("%s met for a bare brand", c.Key)
		}
	}
	if !isState(checkBrandActivation(brandFacts{Brand: Brand{LifecycleStatus: StatusArchived}})) {
		t.Fatal("archived brand must be a state error")
	}
}

func TestEditAndArchiveRules(t *testing.T) {
	if checkItemEditable(kindModel, StatusPending) != nil {
		t.Fatal("pending must be editable")
	}
	if !isState(checkItemEditable(kindModel, StatusActive)) || !isState(checkItemEditable(kindModel, StatusArchived)) {
		t.Fatal("active and archived items are not editable")
	}
	if !isState(checkBrandEditable(StatusArchived)) || checkBrandEditable(StatusActive) != nil {
		t.Fatal("brand edit rule")
	}
	if !isState(checkArchivable("model", StatusArchived)) || checkArchivable("model", StatusActive) != nil {
		t.Fatal("archive rule")
	}
	if !isState(checkCanAddChild(kindVariant, StatusArchived)) || checkCanAddChild(kindVariant, StatusPending) != nil {
		t.Fatal("add child rule")
	}
	if !isState(checkVerifiable(kindColour, StatusActive)) {
		t.Fatal("active items are not re-verified")
	}
}

func TestDeriveSlug(t *testing.T) {
	for in, want := range map[string]string{"Brand A": "brand-a", "  Ünï  EV!! ": "n-ev", "!!!": ""} {
		if got := deriveSlug(in); got != want {
			t.Errorf("deriveSlug(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestOptStringDistinguishesAbsentFromNull(t *testing.T) {
	var r ModelUpdateRequest
	if err := json.Unmarshal([]byte(`{"version":2,"manufacturerCode":null,"name":"X"}`), &r); err != nil {
		t.Fatal(err)
	}
	if !r.ManufacturerCode.Set || r.ManufacturerCode.Value != nil {
		t.Fatal("null must set and clear")
	}
	if !r.Name.Set || *r.Name.Value != "X" || r.VehicleType.Set {
		t.Fatalf("%+v", r)
	}
}

func TestValidationNormalises(t *testing.T) {
	in := ModelCreateRequest{BrandID: someID, Name: "  Model One ", InternalCode: " m1-a ", VehicleType: ptr(" scooter ")}
	if err := validateModelCreate(&in); err != nil {
		t.Fatal(err)
	}
	if in.Name != "Model One" || in.InternalCode != "M1-A" || *in.VehicleType != "SCOOTER" {
		t.Fatalf("%+v", in)
	}
	b := BrandCreateRequest{Name: "Brand A"}
	if err := validateBrandCreate(&b); err != nil || *b.Slug != "brand-a" {
		t.Fatalf("%v %v", err, b.Slug)
	}
}

func ptr(s string) *string { return &s }

func TestSourceDateTodayIsAcceptedInIndia(t *testing.T) {
	// 00:30 IST on 13 Oct is 19:00 UTC on 12 Oct.
	now := time.Date(2026, 10, 12, 19, 0, 0, 0, time.UTC)
	in := VerifyRequest{Version: 1, Confidence: ConfidenceClientApproved, SourceURL: ptr("https://example.com"), SourceDate: "2026-10-13"}
	if err := validateVerify(&in, now); err != nil {
		t.Fatalf("today in IST refused: %v", err)
	}
	in.SourceDate = "2026-10-15"
	if err := validateVerify(&in, now); err == nil {
		t.Fatal("future date accepted")
	}
}

func TestApprovedPricesIsARealCheck(t *testing.T) {
	r := brandReadiness(brandFacts{Brand: Brand{LifecycleStatus: StatusPending}, HasActiveVariant: true})
	for _, c := range r.Checks {
		if c.Key == "approved_prices" && (c.Met || c.Pending) {
			t.Fatalf("approved_prices must be an unmet, non-pending check without a price: %+v", c)
		}
	}
	r = brandReadiness(brandFacts{Brand: Brand{LifecycleStatus: StatusPending}, HasActiveVariant: true, HasApprovedPrice: true})
	for _, c := range r.Checks {
		if c.Key == "approved_prices" && !c.Met {
			t.Fatal("approved_prices not met with an approved price")
		}
	}
}
