package stock

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/audit"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/modules/access"
)

// PGStore is the PostgreSQL Repository.
type PGStore struct{ pool *pgxpool.Pool }

// NewPGStore returns a Repository backed by pool.
func NewPGStore(pool *pgxpool.Pool) *PGStore { return &PGStore{pool: pool} }

// Move implements Repository: MoveUnit in one transaction. When another
// request committed the same key first, the transaction is rolled back and the
// committed movement is replayed (or the key reuse refused).
func (s *PGStore) Move(ctx context.Context, actor audit.Actor, scope access.Scope, req MoveRequest) (MoveResult, bool, error) {
	var res MoveResult
	var replayed bool
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		var err error
		res, replayed, err = MoveUnit(ctx, tx, actor, scope, req)
		return err
	})
	if IsKeyRace(err) {
		res, err = Replay(ctx, s.pool, actor.UserID, req)
		if err != nil {
			return MoveResult{}, false, err
		}
		return res, true, nil
	}
	if err != nil {
		return MoveResult{}, false, err
	}
	return res, replayed, nil
}
