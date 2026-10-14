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
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/modules/access"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/modules/attachments"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/modules/auth"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/modules/catalogue"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/modules/pricing"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/server"
	"github.com/jarvisconsulting/maa-ambey-enterprises/migrations"
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

	schemaVersion, err := migrations.Latest()
	if err != nil {
		return err
	}

	pool, err := database.Connect(ctx, cfg)
	if err != nil {
		return err
	}
	defer pool.Close()
	log.Info("database connected", slog.Int("max_conns", int(cfg.DBMaxConns)))

	// The API never migrates or resets the database itself. A schema mismatch
	// is reported by GET /ready (503) until someone runs the migrations.
	schema := database.SchemaChecker{DB: pool, Want: schemaVersion}
	if err := schema.CheckSchema(ctx); err != nil {
		log.Warn("database schema not ready; /ready will report not_ready", slog.Any("error", err))
	} else {
		log.Info("database schema current", slog.Uint64("version", uint64(schemaVersion)))
	}

	authSvc, err := auth.NewService(auth.NewPGStore(pool), auth.Config{
		IdleTimeout:     cfg.SessionIdleTimeout,
		AbsoluteTimeout: cfg.SessionAbsoluteTimeout,
	}, auth.NewLoginThrottle(cfg.LoginMaxFailures, cfg.LoginMaxFailuresPerIP, cfg.LoginFailureWindow), log)
	if err != nil {
		return fmt.Errorf("auth service: %w", err)
	}
	authHandler := auth.NewHandler(authSvc, auth.CookieConfig{Secure: cfg.SessionCookieSecure}, log)

	if cfg.RBACAllowProposedGrants {
		log.Warn("RBAC_ALLOW_PROPOSED_GRANTS is on: unapproved role grants count (development/test only)")
	}
	accessHandler := access.NewHandler(access.NewPGStore(pool), cfg.RBACAllowProposedGrants, log)

	objects, err := attachments.NewLocalStore(cfg.AttachmentStorageDir)
	if err != nil {
		return err
	}
	if cfg.AttachmentSigningKey == "" {
		log.Warn("ATTACHMENT_SIGNING_KEY is empty: using a random key; download links stop working on restart (development/test only)")
	}
	signer, err := attachments.NewSigner(cfg.AttachmentSigningKey, cfg.AttachmentLinkTTL)
	if err != nil {
		return fmt.Errorf("attachment signer: %w", err)
	}
	attachmentSvc := attachments.NewService(attachments.NewPGStore(pool), objects, signer, cfg.AttachmentMaxBytes, log)
	attachmentHandler := attachments.NewHandler(attachmentSvc, accessHandler, server.APIPrefix, log)
	catalogueHandler := catalogue.NewHandler(catalogue.NewService(catalogue.NewPGStore(pool), log), accessHandler, log)
	pricingHandler := pricing.NewHandler(pricing.NewService(pricing.NewPGStore(pool), log), accessHandler, log)
	log.Info("attachment storage ready", slog.String("driver", objects.Driver()), slog.Int64("max_bytes", cfg.AttachmentMaxBytes))

	router, err := server.NewRouter(server.Deps{Config: cfg, Log: log, DB: pool, Schema: schema, Auth: authHandler, Access: accessHandler, Attachments: attachmentHandler, Catalogue: catalogueHandler, Pricing: pricingHandler, Version: version})
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
