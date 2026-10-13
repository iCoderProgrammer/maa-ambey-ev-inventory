package catalogue

// Lifecycle rules (BR-001…BR-008, ADR-012). They are pure functions so they can
// be tested without a database; the repository calls them inside the
// transaction after locking the rows they look at.

// kind names an item type in messages and audit events.
type kind struct {
	label      string // "model"
	entityType string // audit entity type
	parent     string // parent label ("brand", "model", "variant")
}

var (
	kindModel   = kind{label: "model", entityType: "vehicle_model", parent: "brand"}
	kindVariant = kind{label: "variant", entityType: "variant", parent: "model"}
	kindColour  = kind{label: "colour", entityType: "variant_colour", parent: "variant"}
)

// checkBrandEditable: archived brands are history and are not edited.
func checkBrandEditable(status string) error {
	if status == StatusArchived {
		return &StateError{Message: "Archived brands cannot be changed."}
	}
	return nil
}

// checkItemEditable: only pending items are edited. Active items carry
// verified, approved data that is not overwritten (BR-005); archived items are history.
func checkItemEditable(k kind, status string) error {
	switch status {
	case StatusPending:
		return nil
	case StatusActive:
		return &StateError{Message: "Active " + k.label + "s cannot be edited: approved catalogue data is not overwritten. Archive it instead."}
	default:
		return &StateError{Message: "Archived " + k.label + "s cannot be changed."}
	}
}

// checkCanAddChild: nothing new is added under an archived parent.
func checkCanAddChild(k kind, parentStatus string) error {
	if parentStatus == StatusArchived {
		return &StateError{Message: "A " + k.label + " cannot be added to an archived " + k.parent + "."}
	}
	return nil
}

// checkVerifiable: provenance is recorded while the item is pending.
func checkVerifiable(k kind, status string) error {
	if status != StatusPending {
		return &StateError{Message: "Only pending " + k.label + "s can be verified."}
	}
	return nil
}

// checkArchivable: anything not archived may be archived. Archive never
// deletes and never changes children, so history keeps its references (BR-006, BR-008).
func checkArchivable(label, status string) error {
	if status == StatusArchived {
		return &StateError{Message: "This " + label + " is already archived."}
	}
	return nil
}

// checkItemActivation: a model, variant or colour may become ACTIVE only when it
// is pending and verified (BR-004). A variant needs an ACTIVE model and a colour
// an ACTIVE variant. A model only needs a brand that is not archived: brand
// activation comes last because it requires an active catalogue.
func checkItemActivation(k kind, status, confidence, parentStatus string) error {
	switch status {
	case StatusActive:
		return &StateError{Message: "This " + k.label + " is already active."}
	case StatusArchived:
		return &StateError{Message: "Archived " + k.label + "s cannot be activated."}
	}
	var unmet []ReadinessCheck
	if confidence == ConfidenceUnverified {
		unmet = append(unmet, ReadinessCheck{Key: "verification",
			Message: "Record the " + k.label + "'s source and verification first."})
	}
	if k.parent == "brand" {
		if parentStatus == StatusArchived {
			unmet = append(unmet, ReadinessCheck{Key: "parent", Message: "The brand is archived."})
		}
	} else if parentStatus != StatusActive {
		unmet = append(unmet, ReadinessCheck{Key: "parent", Message: "Activate the " + k.parent + " first."})
	}
	if len(unmet) > 0 {
		return &ActivationBlockedError{Unmet: unmet}
	}
	return nil
}

// brandFacts is what brand readiness looks at.
type brandFacts struct {
	Brand Brand
	// HasActiveVariant: at least one ACTIVE variant under an ACTIVE model of this brand.
	HasActiveVariant bool
}

// pendingChecks are prerequisites from BR-002 whose data the system cannot
// record yet. They stay unmet until their task adds the check (fail closed).
var pendingChecks = []ReadinessCheck{
	{Key: "supplier_link", Pending: true, Message: "Verified supplier relationship for this brand (supplier master, D-01) is not available yet."},
	{Key: "approved_prices", Pending: true, Message: "Approved, effective-dated prices (C-04) are not available yet."},
	{Key: "warranty_terms", Pending: true, Message: "Approved warranty terms (RB-2) are not recorded yet."},
	{Key: "identifier_rules", Pending: true, Message: "Approved vehicle identifier rules (D-04, OQ-4) are not recorded yet."},
}

// brandReadiness lists every prerequisite for enabling purchase/sales (BR-002).
func brandReadiness(f brandFacts) Readiness {
	b := f.Brand
	checks := []ReadinessCheck{
		{Key: "lifecycle", Met: b.LifecycleStatus == StatusPending,
			Message: "The brand must be pending verification (not archived or already active)."},
		{Key: "authorisation", Met: b.AuthorisationStatus == AuthAuthorised,
			Message: "Record AUTHORISED status for the brand."},
		{Key: "legal_identity", Met: b.ManufacturerLegalName != nil,
			Message: "Record the verified supplier/manufacturer legal name."},
		{Key: "authorisation_evidence", Met: b.AuthorisationReference != nil && b.AuthorisationEvidenceID != nil && b.VerifiedBy != nil && b.VerifiedAt != nil,
			Message: "Record the authorisation reference, evidence document, verifier and date."},
		{Key: "verified_catalogue", Met: f.HasActiveVariant,
			Message: "Activate at least one verified model and variant of this brand."},
	}
	checks = append(checks, pendingChecks...)
	ready := true
	for _, c := range checks {
		ready = ready && c.Met
	}
	return Readiness{BrandID: b.ID, Ready: ready, Checks: checks}
}

// checkBrandActivation returns the blocking error for an activation attempt.
func checkBrandActivation(f brandFacts) error {
	switch f.Brand.LifecycleStatus {
	case StatusActive:
		return &StateError{Message: "This brand is already active."}
	case StatusArchived:
		return &StateError{Message: "Archived brands cannot be activated."}
	}
	r := brandReadiness(f)
	if r.Ready {
		return nil
	}
	var unmet []ReadinessCheck
	for _, c := range r.Checks {
		if !c.Met {
			unmet = append(unmet, c)
		}
	}
	return &ActivationBlockedError{Unmet: unmet}
}
