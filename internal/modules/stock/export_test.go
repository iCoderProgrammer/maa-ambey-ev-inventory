package stock

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// SetReconcileHook makes s run fn inside each reconciliation's transaction,
// between the vehicle and the parts queries (tests only).
func SetReconcileHook(s *PGStore, fn func(context.Context, pgx.Tx) error) { s.reconcileHook = fn }
