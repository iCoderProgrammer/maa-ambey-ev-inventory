// Command seed applies the idempotent SQL seed files in ./seeds (or SEEDS_DIR)
// in a single transaction. Run migrations first. Safe to re-run.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/config"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/database"
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
	dir := os.Getenv("SEEDS_DIR")
	if dir == "" {
		dir = "seeds"
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	pool, err := database.Connect(ctx, cfg)
	if err != nil {
		return err
	}
	defer pool.Close()

	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	applied, err := database.ApplySeeds(ctx, tx, dir)
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
