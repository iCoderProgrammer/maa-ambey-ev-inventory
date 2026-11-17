package openingstock

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

// SetNow replaces the service clock (tests only).
func SetNow(s *Service, now func() time.Time) { s.now = now }

// SetCommitFault installs a hook run inside the commit transaction after the
// units are created (tests only; nil removes it).
func SetCommitFault(fn func(ctx context.Context, tx pgx.Tx) error) { commitFault = fn }

// SetPreflightProbe installs a hook run inside the preflight transaction (tests only; nil removes it).
func SetPreflightProbe(fn func(ctx context.Context, tx pgx.Tx) error) { preflightProbe = fn }

// Gates exposes the pure commit checks to the unit tests.
var (
	BatchGate        = batchGate
	SameSource       = sameSource
	RowGate          = rowGate
	ValidateApproval = validateApproval
	SameApproval     = sameApproval
	Classify         = classify
	Evaluate         = evaluate
	Blockers         = blockers
)
