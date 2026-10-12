// Command api runs the Maa Ambey Enterprises inventory REST API.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/jarvisconsulting/maa-ambey-enterprises/docs/openapi"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/config"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/database"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/logger"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/server"
)

// version is overridden at build time: -ldflags "-X main.version=<git sha>".
var version = "dev"

//	@title			Maa Ambey Enterprises Inventory API
//	@version		0.1.0
//	@description	Multi-brand EV inventory management API. All business endpoints live under /api/v1.
//	@BasePath		/api/v1

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "api: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	log := logger.New(cfg.LogLevel)
	slog.SetDefault(log)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := database.Connect(ctx, cfg)
	if err != nil {
		return err
	}
	defer pool.Close()
	log.Info("database connected", slog.Int("max_conns", int(cfg.DBMaxConns)))

	router, err := server.NewRouter(server.Deps{Config: cfg, Log: log, DB: pool, Version: version})
	if err != nil {
		return fmt.Errorf("build router: %w", err)
	}
	srv := server.NewHTTPServer(cfg, router)

	serveErr := make(chan error, 1)
	go func() {
		log.Info("api listening", slog.String("addr", srv.Addr), slog.String("env", cfg.AppEnv), slog.String("version", version))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
		}
		close(serveErr)
	}()

	select {
	case err := <-serveErr:
		if err != nil {
			return fmt.Errorf("http server: %w", err)
		}
	case <-ctx.Done():
	}

	log.Info("shutting down", slog.Duration("timeout", cfg.ShutdownTimeout))
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}
	log.Info("shutdown complete")
	return nil
}
