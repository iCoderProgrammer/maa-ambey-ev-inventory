// Command seed applies the idempotent SQL seed files embedded from ./seeds in a
// single transaction. It refuses to run until the database schema matches this
// build (run migrations first). Safe to re-run.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/config"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/database"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/migrations"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/seeds"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "seed: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	pool, err := database.Connect(ctx, cfg)
	if err != nil {
		return err
	}
	defer pool.Close()

	// Seeds target the current schema: refuse to run against a database that
	// is unmigrated, partially migrated or left dirty by a failed migration.
	want, err := migrations.Latest()
	if err != nil {
		return err
	}
	if err := database.CheckSchema(ctx, pool, want); err != nil {
		return fmt.Errorf("database not ready for seeding: %w", err)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	applied, err := database.ApplySeeds(ctx, tx, seeds.FS)
	if err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	for _, f := range applied {
		fmt.Println("applied", f)
	}
	fmt.Printf("ok: %d seed file(s) applied\n", len(applied))
	return nil
}
