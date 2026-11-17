package vehicles

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/access"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/ageing"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/attachments"
)

// Repository reads vehicle units. Registration is RegisterFromReceiptUnit,
// which runs inside the posting transaction rather than through this interface.
type Repository interface {
	List(ctx context.Context, f ListFilter) (Page[Unit], error)
	GetDetail(ctx context.Context, id string) (Detail, error)
	// UnitLocation returns the unit's location (nil = none) or ErrNotFound.
	UnitLocation(ctx context.Context, id string) (*string, error)
	Movements(ctx context.Context, f MovementFilter) (Page[Movement], error)
	// AgeingOrigins returns the origin of every unit the filter selects (E-07 summary).
	AgeingOrigins(ctx context.Context, f ListFilter) ([]ageing.Origin, error)
}

// AgeingSource is the business calendar and band configuration (package ageing).
type AgeingSource interface {
	Calendar() ageing.Calendar
	CurrentBands(ctx context.Context) (*ageing.BandSet, error)
}

// Service checks permissions (again, after the route guard), the caller's
// location scope, and shapes what each caller may see.
type Service struct {
	repo   Repository
	log    *slog.Logger
	ageing AgeingSource
}

// NewService builds the service. Without WithAgeing, ages are counted on the
// default business calendar with no band configuration (tests only).
func NewService(repo Repository, log *slog.Logger) *Service {
	return &Service{repo: repo, log: log}
}

// WithAgeing sets the business calendar and band configuration (E-07).
func (s *Service) WithAgeing(src AgeingSource) *Service {
	s.ageing = src
	return s
}

type defaultAgeing struct{ cal ageing.Calendar }

func (d defaultAgeing) Calendar() ageing.Calendar                           { return d.cal }
func (defaultAgeing) CurrentBands(context.Context) (*ageing.BandSet, error) { return nil, nil }

func (s *Service) ageingSource() AgeingSource {
	if s.ageing != nil {
		return s.ageing
	}
	cal, _ := ageing.NewCalendar(ageing.DefaultTimeZone, nil)
	return defaultAgeing{cal}
}

// prepareAgeing sets the business date, and resolves the band filter against the
// current configuration (422 for a key it does not have).
func (s *Service) prepareAgeing(ctx context.Context, f *ListFilter) (*ageing.BandSet, time.Time, error) {
	src := s.ageingSource()
	asOf := src.Calendar().Today()
	f.AsOf = asOf.Format(time.DateOnly)
	set, err := src.CurrentBands(ctx)
	if err != nil {
		return nil, asOf, err
	}
	if f.BandKey != "" {
		b, ok := set.Find(f.BandKey)
		if !ok {
			msg := "is not a band of the current ageing configuration"
			if set == nil {
				msg = "cannot be used: no ageing band configuration exists"
			}
			return nil, asOf, &ValidationError{Fields: []FieldProblem{{"ageingBand", msg}}}
		}
		f.Band = &b
	}
	return set, asOf, nil
}

// applyAgeing evaluates a unit's ageing (E-07) and keeps ageDays equal to it.
func applyAgeing(u *Unit, asOf time.Time, set *ageing.BandSet) {
	o := ageing.Origin{SourceType: u.SourceType, Date: u.ReceivedDate}
	if u.grnStatus != nil {
		o.GRNStatus = *u.grnStatus
	}
	if u.grnDate != nil {
		o.GRNDate = *u.grnDate
	}
	o.OpeningApproved = u.openingDateApproved != nil && *u.openingDateApproved
	u.Ageing = ageing.Evaluate(o, asOf, set)
	u.AgeDays = u.Ageing.AgeDays
}

// List lists units (vehicle_units.view) in the caller's location scope: units
// without a location, and units at a location the caller may work in. A
// location filter outside the scope is refused.
func (s *Service) List(ctx context.Context, a access.Access, q ListQuery) (Page[Unit], error) {
	if !a.Has(access.PermVehicleUnitView) {
		return Page[Unit]{}, ErrForbidden
	}
	f, err := scopedFilter(a, q)
	if err != nil {
		return Page[Unit]{}, err
	}
	set, asOf, err := s.prepareAgeing(ctx, &f)
	if err != nil {
		return Page[Unit]{}, err
	}
	p, err := s.repo.List(ctx, f)
	if err != nil {
		return Page[Unit]{}, err
	}
	for i := range p.Items {
		shapeCost(&p.Items[i], a)
		applyAgeing(&p.Items[i], asOf, set)
	}
	return p, nil
}

// scopedFilter validates a list query and applies the caller's location scope.
func scopedFilter(a access.Access, q ListQuery) (ListFilter, error) {
	f, err := validateList(q)
	if err != nil {
		return ListFilter{}, err
	}
	requested := f.LocationIDs
	ids, all, err := a.Scope.Restrict(requested)
	if errors.Is(err, access.ErrOutOfScope) {
		return ListFilter{}, &ValidationError{Fields: []FieldProblem{{"locationId", "is not one of your locations"}}}
	}
	if err != nil {
		return ListFilter{}, err
	}
	if len(requested) == 0 {
		f.LocationIDs, f.ScopeLocationIDs, f.AllLocations = nil, ids, all
	}
	return f, nil
}

// AgeingSummary is the stock ageing of the units a list filter selects (E-07).
type AgeingSummary struct {
	// AsOf/TimeZone: the business date ages are counted to.
	AsOf     string `json:"asOf" example:"2026-11-16"`
	TimeZone string `json:"timeZone" example:"Asia/Kolkata"`
	// Configured: a band configuration exists; BandSet identifies it (null when not configured).
	Configured bool                  `json:"configured"`
	BandSet    *AgeingSummaryBandSet `json:"bandSet" extensions:"x-nullable"`
	// Total units selected = WithOrigin + MissingOrigin.
	Total         int `json:"total" example:"12"`
	WithOrigin    int `json:"withOrigin" example:"11"`
	MissingOrigin int `json:"missingOrigin" example:"1"`
	// Bands: every band of the configuration, in order, with its count (0 included); empty when not configured.
	Bands []AgeingBandCount `json:"bands"`
	// Unbanded: units with an age but no band (only when no configuration exists).
	Unbanded int `json:"unbanded" example:"0"`
	// MissingByReason counts MissingOrigin by reason.
	MissingByReason map[string]int `json:"missingByReason"`
} //@name VehicleAgeingSummary

// AgeingSummaryBandSet identifies the band configuration a summary used.
type AgeingSummaryBandSet struct {
	Version int    `json:"version" example:"1"`
	Status  string `json:"status" enums:"PROPOSED,APPROVED" example:"PROPOSED"`
	Source  string `json:"source"`
} //@name VehicleAgeingSummaryBandSet

// AgeingBandCount is one band and how many selected units fall in it.
type AgeingBandCount struct {
	ageing.Band
	Count int `json:"count" example:"7"`
} //@name VehicleAgeingBandCount

// AgeingSummary counts the units the list filters select (vehicle_units.view, the
// caller's location scope) by ageing band and missing-origin reason, with the
// same rules as each unit's ageing.
func (s *Service) AgeingSummary(ctx context.Context, a access.Access, q ListQuery) (AgeingSummary, error) {
	if !a.Has(access.PermVehicleUnitView) {
		return AgeingSummary{}, ErrForbidden
	}
	f, err := scopedFilter(a, q)
	if err != nil {
		return AgeingSummary{}, err
	}
	set, asOf, err := s.prepareAgeing(ctx, &f)
	if err != nil {
		return AgeingSummary{}, err
	}
	origins, err := s.repo.AgeingOrigins(ctx, f)
	if err != nil {
		return AgeingSummary{}, err
	}
	out := AgeingSummary{AsOf: f.AsOf, TimeZone: s.ageingSource().Calendar().Zone(), Configured: set != nil,
		Bands: []AgeingBandCount{}, MissingByReason: map[string]int{}}
	index := map[string]int{}
	if set != nil {
		out.BandSet = &AgeingSummaryBandSet{Version: set.Version, Status: set.Status, Source: set.Source}
		for i, b := range set.Bands {
			out.Bands = append(out.Bands, AgeingBandCount{Band: b})
			index[b.Key] = i
		}
	}
	for _, o := range origins {
		r := ageing.Evaluate(o, asOf, set)
		out.Total++
		switch {
		case r.Status == ageing.StatusMissingOrigin:
			out.MissingOrigin++
			out.MissingByReason[*r.MissingReason]++
		case r.BandKey != nil:
			out.WithOrigin++
			out.Bands[index[*r.BandKey]].Count++
		default:
			out.WithOrigin++
			out.Unbanded++
		}
	}
	return out, nil
}

// GetDetail returns one unit with its source, history and the documents the
// caller may open (vehicle_units.view). A unit at a location outside the
// caller's scope is reported as not found.
func (s *Service) GetDetail(ctx context.Context, a access.Access, id string) (Detail, error) {
	if !a.Has(access.PermVehicleUnitView) {
		return Detail{}, ErrForbidden
	}
	if _, err := uuid.Parse(id); err != nil {
		return Detail{}, ErrNotFound
	}
	d, err := s.repo.GetDetail(ctx, id)
	if err != nil {
		return Detail{}, err
	}
	if d.LocationID != nil && !a.Scope.Allows(*d.LocationID) {
		return Detail{}, ErrNotFound
	}
	shapeCost(&d.Unit, a)
	src := s.ageingSource()
	set, err := src.CurrentBands(ctx)
	if err != nil {
		return Detail{}, err
	}
	applyAgeing(&d.Unit, src.Calendar().Today(), set)
	visible := make([]Document, 0, len(d.Documents))
	for _, doc := range d.Documents {
		if attachments.CanSee(a, attachments.Attachment{ID: doc.AttachmentID, OwnerType: doc.ownerType,
			UploadedBy: doc.uploadedBy, LocationID: doc.locationID}) {
			visible = append(visible, doc)
		}
	}
	d.Documents, d.DocumentsHidden = visible, !a.Has(access.PermAttachmentsView)
	return d, nil
}

// Movements returns one page of a unit's stock movements (vehicle_units.view),
// oldest first unless order=desc. A unit outside the caller's scope is not
// found; a location outside the scope is named by ID only.
func (s *Service) Movements(ctx context.Context, a access.Access, id string, q MovementQuery) (Page[Movement], error) {
	if !a.Has(access.PermVehicleUnitView) {
		return Page[Movement]{}, ErrForbidden
	}
	if _, err := uuid.Parse(id); err != nil {
		return Page[Movement]{}, ErrNotFound
	}
	f, err := validateMovements(id, q)
	if err != nil {
		return Page[Movement]{}, err
	}
	loc, err := s.repo.UnitLocation(ctx, id)
	if err != nil {
		return Page[Movement]{}, err
	}
	if loc != nil && !a.Scope.Allows(*loc) {
		return Page[Movement]{}, ErrNotFound
	}
	p, err := s.repo.Movements(ctx, f)
	if err != nil {
		return Page[Movement]{}, err
	}
	for i := range p.Items {
		m := &p.Items[i]
		if m.FromLocationID != nil && !a.Scope.Allows(*m.FromLocationID) {
			m.FromLocationCode, m.FromLocationName = nil, nil
		}
		if m.ToLocationID != nil && !a.Scope.Allows(*m.ToLocationID) {
			m.ToLocationCode, m.ToLocationName = nil, nil
		}
	}
	return p, nil
}

// shapeCost removes the acquisition cost for callers without vehicle_units.cost.view.
func shapeCost(u *Unit, a access.Access) {
	if a.Has(access.PermVehicleUnitCostView) {
		return
	}
	u.UnitCost, u.CostCurrency, u.CostHidden = nil, nil, true
}
