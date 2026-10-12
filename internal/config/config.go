// Package config loads and validates the API configuration from environment variables.
package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Supported application environments.
const (
	EnvDevelopment = "development"
	EnvTest        = "test"
	EnvStaging     = "staging"
	EnvProduction  = "production"
)

// Config holds all runtime settings for the API process.
type Config struct {
	AppEnv string
	Port   int

	DatabaseURL       string
	DBMaxConns        int32
	DBMinConns        int32
	DBMaxConnLifetime time.Duration
	DBMaxConnIdleTime time.Duration

	CORSAllowedOrigins []string
	TrustedProxies     []string

	LogLevel string

	RequestTimeout  time.Duration
	ShutdownTimeout time.Duration

	RateLimitRPS   float64
	RateLimitBurst int

	SwaggerEnabled bool
}

// IsProduction reports whether the API runs with production safeguards.
func (c Config) IsProduction() bool { return c.AppEnv == EnvProduction }

// Addr returns the HTTP listen address.
func (c Config) Addr() string { return fmt.Sprintf(":%d", c.Port) }

// Load reads configuration from the process environment and validates it.
func Load() (Config, error) {
	return load(os.Getenv)
}

func load(getenv func(string) string) (Config, error) {
	p := parser{getenv: getenv}

	cfg := Config{
		AppEnv:             p.str("APP_ENV", EnvDevelopment),
		Port:               p.int("APP_PORT", 8080),
		DatabaseURL:        strings.TrimSpace(getenv("DATABASE_URL")),
		DBMaxConns:         int32(p.int("DB_MAX_CONNS", 10)),
		DBMinConns:         int32(p.int("DB_MIN_CONNS", 1)),
		DBMaxConnLifetime:  p.duration("DB_MAX_CONN_LIFETIME", time.Hour),
		DBMaxConnIdleTime:  p.duration("DB_MAX_CONN_IDLE_TIME", 30*time.Minute),
		CORSAllowedOrigins: p.list("CORS_ALLOWED_ORIGINS"),
		TrustedProxies:     p.list("TRUSTED_PROXIES"),
		LogLevel:           strings.ToLower(p.str("LOG_LEVEL", "info")),
		RequestTimeout:     p.duration("REQUEST_TIMEOUT", 15*time.Second),
		ShutdownTimeout:    p.duration("SHUTDOWN_TIMEOUT", 20*time.Second),
		RateLimitRPS:       p.float("RATE_LIMIT_RPS", 20),
		RateLimitBurst:     p.int("RATE_LIMIT_BURST", 40),
	}
	// Swagger UI is on by default outside production; production must opt in.
	cfg.SwaggerEnabled = p.bool("SWAGGER_ENABLED", cfg.AppEnv != EnvProduction)

	errs := p.errs
	errs = append(errs, cfg.validate()...)
	if len(errs) > 0 {
		return Config{}, fmt.Errorf("invalid configuration:\n  - %w", errors.Join(errs...))
	}
	return cfg, nil
}

func (c Config) validate() []error {
	var errs []error

	switch c.AppEnv {
	case EnvDevelopment, EnvTest, EnvStaging, EnvProduction:
	default:
		errs = append(errs, fmt.Errorf("APP_ENV must be one of development, test, staging, production (got %q)", c.AppEnv))
	}
	if c.Port < 1 || c.Port > 65535 {
		errs = append(errs, fmt.Errorf("APP_PORT must be between 1 and 65535 (got %d)", c.Port))
	}

	if c.DatabaseURL == "" {
		errs = append(errs, errors.New("DATABASE_URL is required, e.g. postgres://user:password@localhost:5432/maa_ambey_inventory?sslmode=disable"))
	} else if u, err := url.Parse(c.DatabaseURL); err != nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") || u.Host == "" {
		// Do not echo the value: it contains credentials.
		errs = append(errs, errors.New("DATABASE_URL must be a postgres:// or postgresql:// URL with a host"))
	} else if c.IsProduction() && u.Query().Get("sslmode") == "disable" {
		errs = append(errs, errors.New("DATABASE_URL must not use sslmode=disable in production"))
	}

	if c.DBMaxConns < 1 {
		errs = append(errs, errors.New("DB_MAX_CONNS must be at least 1"))
	}
	if c.DBMinConns < 0 || c.DBMinConns > c.DBMaxConns {
		errs = append(errs, errors.New("DB_MIN_CONNS must be between 0 and DB_MAX_CONNS"))
	}

	for _, o := range c.CORSAllowedOrigins {
		if o == "*" {
			errs = append(errs, errors.New("CORS_ALLOWED_ORIGINS must list explicit origins; \"*\" is not allowed"))
			continue
		}
		if u, err := url.Parse(o); err != nil || u.Scheme == "" || u.Host == "" || u.Path != "" {
			errs = append(errs, fmt.Errorf("CORS_ALLOWED_ORIGINS entry %q must look like https://host[:port]", o))
		}
	}

	switch c.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		errs = append(errs, fmt.Errorf("LOG_LEVEL must be one of debug, info, warn, error (got %q)", c.LogLevel))
	}

	if c.RequestTimeout <= 0 {
		errs = append(errs, errors.New("REQUEST_TIMEOUT must be positive"))
	}
	if c.ShutdownTimeout <= 0 {
		errs = append(errs, errors.New("SHUTDOWN_TIMEOUT must be positive"))
	}
	if c.RateLimitRPS <= 0 || c.RateLimitBurst < 1 {
		errs = append(errs, errors.New("RATE_LIMIT_RPS must be positive and RATE_LIMIT_BURST at least 1"))
	}
	return errs
}

// parser collects parse errors so all problems are reported at once.
type parser struct {
	getenv func(string) string
	errs   []error
}

func (p *parser) raw(key string) (string, bool) {
	v := strings.TrimSpace(p.getenv(key))
	return v, v != ""
}

func (p *parser) str(key, def string) string {
	if v, ok := p.raw(key); ok {
		return v
	}
	return def
}

func (p *parser) int(key string, def int) int {
	v, ok := p.raw(key)
	if !ok {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		p.errs = append(p.errs, fmt.Errorf("%s must be an integer (got %q)", key, v))
		return def
	}
	return n
}

func (p *parser) float(key string, def float64) float64 {
	v, ok := p.raw(key)
	if !ok {
		return def
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		p.errs = append(p.errs, fmt.Errorf("%s must be a number (got %q)", key, v))
		return def
	}
	return f
}

func (p *parser) bool(key string, def bool) bool {
	v, ok := p.raw(key)
	if !ok {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		p.errs = append(p.errs, fmt.Errorf("%s must be true or false (got %q)", key, v))
		return def
	}
	return b
}

func (p *parser) duration(key string, def time.Duration) time.Duration {
	v, ok := p.raw(key)
	if !ok {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		p.errs = append(p.errs, fmt.Errorf("%s must be a duration such as 15s or 1m (got %q)", key, v))
		return def
	}
	return d
}

func (p *parser) list(key string) []string {
	v, ok := p.raw(key)
	if !ok {
		return nil
	}
	var out []string
	for _, item := range strings.Split(v, ",") {
		if item = strings.TrimSpace(item); item != "" {
			out = append(out, item)
		}
	}
	return out
}
