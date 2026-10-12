package config

import (
	"strings"
	"testing"
	"time"
)

func envFrom(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func validEnv() map[string]string {
	return map[string]string{
		"DATABASE_URL":         "postgres://app_user:change_me@localhost:5432/maa_ambey_inventory?sslmode=disable",
		"CORS_ALLOWED_ORIGINS": "http://localhost:5173, http://127.0.0.1:5173",
	}
}

func TestLoadDefaults(t *testing.T) {
	cfg, err := load(envFrom(validEnv()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.AppEnv != EnvDevelopment || cfg.Port != 8080 || cfg.LogLevel != "info" {
		t.Errorf("unexpected defaults: %+v", cfg)
	}
	if cfg.RequestTimeout != 15*time.Second {
		t.Errorf("RequestTimeout = %v", cfg.RequestTimeout)
	}
	if !cfg.SwaggerEnabled {
		t.Error("swagger should be enabled by default in development")
	}
	if len(cfg.CORSAllowedOrigins) != 2 || cfg.CORSAllowedOrigins[1] != "http://127.0.0.1:5173" {
		t.Errorf("CORSAllowedOrigins = %v", cfg.CORSAllowedOrigins)
	}
}

func TestLoadRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		wantErr string
	}{
		{"missing database url", map[string]string{"DATABASE_URL": ""}, "DATABASE_URL is required"},
		{"non postgres url", map[string]string{"DATABASE_URL": "mysql://x@localhost/db"}, "postgres:// or postgresql://"},
		{"no database name", map[string]string{"DATABASE_URL": "postgres://u:p@localhost:5432/?sslmode=disable"}, "must name a database"},
		{"no host", map[string]string{"DATABASE_URL": "postgres:///maa_ambey_inventory"}, "URL with a host"},
		{"bad env", map[string]string{"APP_ENV": "prod"}, "APP_ENV must be one of"},
		{"bad port", map[string]string{"APP_PORT": "abc"}, "APP_PORT must be an integer"},
		{"port out of range", map[string]string{"APP_PORT": "70000"}, "APP_PORT must be between"},
		{"wildcard cors", map[string]string{"CORS_ALLOWED_ORIGINS": "*"}, `"*" is not allowed`},
		{"cors with path", map[string]string{"CORS_ALLOWED_ORIGINS": "http://localhost:5173/app"}, "must look like"},
		{"bad log level", map[string]string{"LOG_LEVEL": "verbose"}, "LOG_LEVEL must be one of"},
		{"bad duration", map[string]string{"REQUEST_TIMEOUT": "15"}, "REQUEST_TIMEOUT must be a duration"},
		{"min above max conns", map[string]string{"DB_MAX_CONNS": "2", "DB_MIN_CONNS": "5"}, "DB_MIN_CONNS must be between"},
		{"idle too short", map[string]string{"SESSION_IDLE_TIMEOUT": "10s"}, "SESSION_IDLE_TIMEOUT must be at least 1m"},
		{"absolute below idle", map[string]string{"SESSION_IDLE_TIMEOUT": "2h", "SESSION_ABSOLUTE_TIMEOUT": "1h"}, "SESSION_ABSOLUTE_TIMEOUT must be between"},
		{"insecure cookie in staging", map[string]string{"APP_ENV": "staging", "SESSION_COOKIE_SECURE": "false"}, "SESSION_COOKIE_SECURE must be true in staging"},
		{"zero login failures", map[string]string{"LOGIN_MAX_FAILURES": "0"}, "LOGIN_MAX_FAILURES must be at least 1"},
		{"proposed grants in test", map[string]string{"APP_ENV": "test", "RBAC_ALLOW_PROPOSED_GRANTS": "true"}, "RBAC_ALLOW_PROPOSED_GRANTS must be false in test"},
		{"proposed grants in staging", map[string]string{"APP_ENV": "staging", "SESSION_COOKIE_SECURE": "true", "RBAC_ALLOW_PROPOSED_GRANTS": "true"}, "RBAC_ALLOW_PROPOSED_GRANTS must be false in staging"},
		{"proposed grants in production", map[string]string{"APP_ENV": "production", "DATABASE_URL": "postgres://u:p@db:5432/app?sslmode=require", "SESSION_COOKIE_SECURE": "true", "CORS_ALLOWED_ORIGINS": "https://x.example", "RBAC_ALLOW_PROPOSED_GRANTS": "true"}, "RBAC_ALLOW_PROPOSED_GRANTS must be false in production"},
		{"insecure prod db", map[string]string{"APP_ENV": "production"}, "sslmode=disable in production"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := validEnv()
			for k, v := range tt.env {
				env[k] = v
			}
			_, err := load(envFrom(env))
			if err == nil {
				t.Fatalf("expected error containing %q", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error %q does not contain %q", err, tt.wantErr)
			}
		})
	}
}

func TestLoadNeverEchoesDatabaseCredentials(t *testing.T) {
	env := validEnv()
	env["DATABASE_URL"] = "mysql://secret_user:secret_pass@localhost/db"
	_, err := load(envFrom(env))
	if err == nil || strings.Contains(err.Error(), "secret_pass") {
		t.Fatalf("error must not contain credentials: %v", err)
	}
}

func TestProductionDisablesSwaggerByDefault(t *testing.T) {
	env := validEnv()
	env["APP_ENV"] = "production"
	env["DATABASE_URL"] = "postgres://u:p@db.internal:5432/app?sslmode=require"
	env["CORS_ALLOWED_ORIGINS"] = "https://inventory.example.com"
	cfg, err := load(envFrom(env))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.SwaggerEnabled {
		t.Error("swagger must be opt-in in production")
	}
	if !cfg.SessionCookieSecure {
		t.Error("session cookies must be Secure by default in production")
	}
}

func TestProposedGrantsAllowedInLocalDevelopment(t *testing.T) {
	env := validEnv()
	env["RBAC_ALLOW_PROPOSED_GRANTS"] = "true"
	cfg, err := load(envFrom(env))
	if err != nil || !cfg.RBACAllowProposedGrants {
		t.Fatalf("development must allow the switch: %v", err)
	}
}
