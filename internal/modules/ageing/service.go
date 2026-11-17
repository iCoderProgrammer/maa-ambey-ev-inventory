package ageing

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"unicode/utf8"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/audit"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/access"
)

// Service reads the business calendar and the band configuration, and lets
// authorised people propose and approve configurations.
type Service struct {
	repo Repository
	cal  Calendar
	log  *slog.Logger
}

// NewService builds the service.
func NewService(repo Repository, cal Calendar, log *slog.Logger) *Service {
	return &Service{repo: repo, cal: cal, log: log}
}

// Calendar is the business calendar.
func (s *Service) Calendar() Calendar { return s.cal }

// CurrentBands returns the current configuration, or nil when none exists.
func (s *Service) CurrentBands(ctx context.Context) (*BandSet, error) {
	set, err := s.repo.Current(ctx)
	if errors.Is(err, ErrNotConfigured) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &set, nil
}

// Configuration is the current band configuration as the API returns it.
type Configuration struct {
	// Configured: false when no band configuration exists (ages are still calculated, without bands).
	Configured bool `json:"configured"`
	// BandSet: the current version; null when not configured.
	BandSet *BandSet `json:"bandSet" extensions:"x-nullable"`
	// TimeZone and AsOf: the business calendar ages are counted on.
	TimeZone string `json:"timeZone" example:"Asia/Kolkata"`
	AsOf     string `json:"asOf" example:"2026-11-16"`
	// BoundaryRule states how bands are applied.
	BoundaryRule string `json:"boundaryRule"`
} //@name StockAgeingConfiguration

// BoundaryRule is the documented band convention.
const BoundaryRule = "A band covers minDays <= age <= maxDays in whole days (both inclusive); the last band is open-ended. Age = business date - origin date."

// Configuration returns the current configuration (vehicle_units.view or stock.ageing.configure).
func (s *Service) Configuration(ctx context.Context, a access.Access) (Configuration, error) {
	if !a.Has(access.PermVehicleUnitView) && !a.Has(access.PermAgeingConfigure) {
		return Configuration{}, ErrForbidden
	}
	set, err := s.CurrentBands(ctx)
	if err != nil {
		return Configuration{}, err
	}
	return Configuration{Configured: set != nil, BandSet: set, TimeZone: s.cal.Zone(),
		AsOf: s.cal.Today().Format("2006-01-02"), BoundaryRule: BoundaryRule}, nil
}

// ProposeInput is the body of a new configuration.
type ProposeInput struct {
	// ExpectedVersion is the current version (0 when none exists); a different one is 409 VERSION_CONFLICT.
	ExpectedVersion *int `json:"expectedVersion" example:"1"`
	// Source: where the thresholds come from (document or decision reference), 1-300 characters.
	Source string `json:"source" example:"Owner decision 2026-11-20"`
	Bands  []Band `json:"bands"`
} //@name StockAgeingProposeInput

// ApproveInput approves the current proposed version.
type ApproveInput struct {
	Version *int `json:"version" example:"2"`
	// Reference: optional approval reference (e.g. meeting minute), up to 200 characters.
	Reference *string `json:"reference" extensions:"x-nullable"`
} //@name StockAgeingApproveInput

// Propose records a new PROPOSED configuration (stock.ageing.configure).
func (s *Service) Propose(ctx context.Context, a access.Access, actor audit.Actor, requestID string, in ProposeInput) (BandSet, error) {
	if !a.Has(access.PermAgeingConfigure) {
		return BandSet{}, ErrForbidden
	}
	var probs []Problem
	if in.ExpectedVersion == nil || *in.ExpectedVersion < 0 {
		probs = append(probs, Problem{"expectedVersion", "is required (0 when no configuration exists)"})
	}
	src := strings.TrimSpace(in.Source)
	if src == "" || utf8.RuneCountInString(src) > 300 {
		probs = append(probs, Problem{"source", "must be 1-300 characters: name the document or decision the thresholds come from"})
	}
	probs = append(probs, ValidateBands(in.Bands)...)
	if len(probs) > 0 {
		return BandSet{}, &ValidationError{Fields: probs}
	}
	return s.repo.Propose(ctx, actor, requestID, *in.ExpectedVersion, src, in.Bands)
}

// Approve approves the current PROPOSED configuration (stock.ageing.configure; not by its proposer).
func (s *Service) Approve(ctx context.Context, a access.Access, actor audit.Actor, requestID string, in ApproveInput) (BandSet, error) {
	if !a.Has(access.PermAgeingConfigure) {
		return BandSet{}, ErrForbidden
	}
	var probs []Problem
	if in.Version == nil || *in.Version < 1 {
		probs = append(probs, Problem{"version", "is required"})
	}
	var ref *string
	if in.Reference != nil {
		r := strings.TrimSpace(*in.Reference)
		if r == "" || utf8.RuneCountInString(r) > 200 {
			probs = append(probs, Problem{"reference", "must be 1-200 characters when given"})
		}
		ref = &r
	}
	if len(probs) > 0 {
		return BandSet{}, &ValidationError{Fields: probs}
	}
	set, err := s.repo.Approve(ctx, actor, requestID, *in.Version, ref)
	if errors.Is(err, ErrSelfApproval) {
		if rerr := s.repo.RecordRefusal(ctx, actor, requestID, *in.Version, "self-approval"); rerr != nil && s.log != nil {
			s.log.ErrorContext(ctx, "record refused ageing approval", slog.Any("error", rerr))
		}
	}
	return set, err
}
