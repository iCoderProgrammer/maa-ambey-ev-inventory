// Package database owns the PostgreSQL connection pool.
package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/config"
)

// Connect creates a pgx connection pool and verifies connectivity.
// It never creates, resets or migrates the database; migrations are an
// explicit developer action (see cmd/migrate).
func Connect(ctx context.Context, cfg config.Config) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		// pgx parse errors can include the raw DSN; keep the message generic.
		return nil, fmt.Errorf("parse DATABASE_URL: invalid connection string")
	}
	poolCfg.MaxConns = cfg.DBMaxConns
	poolCfg.MinConns = cfg.DBMinConns
	poolCfg.MaxConnLifetime = cfg.DBMaxConnLifetime
	poolCfg.MaxConnIdleTime = cfg.DBMaxConnIdleTime
	poolCfg.HealthCheckPeriod = time.Minute
	poolCfg.ConnConfig.RuntimeParams["application_name"] = "maa-ambey-api"

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("create database pool: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("connect to PostgreSQL at %s:%d (is `make db-up` running?): %w",
			poolCfg.ConnConfig.Host, poolCfg.ConnConfig.Port, err)
	}
	return pool, nil
}
