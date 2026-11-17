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
	_ "time/tzdata" // BUSINESS_TIME_ZONE must load in minimal containers
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

	// Sessions and sign-in (B-06). Defaults are proposals pending owner sign-off.
	SessionIdleTimeout     time.Duration
	SessionAbsoluteTimeout time.Duration
	SessionCookieSecure    bool
	LoginMaxFailures       int
	LoginMaxFailuresPerIP  int
	LoginFailureWindow     time.Duration

	// RBACAllowProposedGrants makes PROPOSED (not yet owner-approved) role grants
	// count. Local development only (APP_ENV=development); refused elsewhere (B-07).
	RBACAllowProposedGrants bool

	// Private attachments (B-09). Limits are proposals pending owner confirmation.
	AttachmentStorageDriver string
	AttachmentStorageDir    string
	AttachmentMaxBytes      int64
	AttachmentLinkTTL       time.Duration
	// AttachmentSigningKey signs short-lived download links. Required outside
	// development/test; there an empty key means a random per-process key.
	AttachmentSigningKey string
	// BusinessTimeZone is the IANA zone of the business calendar stock ages are
	// counted on (E-07, ADR-029). Default Asia/Kolkata (the showroom's zone);
	// confirmation pending (OQ-45).
	BusinessTimeZone string
	// Opening stock import preview (G-06): file size and row limits, and how long
	// a preview stays usable for G-07 (proposals; configurable).
	ImportMaxBytes   int64
	ImportMaxRows    int
	ImportPreviewTTL time.Duration
	// OpeningStockApprovedStatuses (G-07, OQ-47): comma-separated stock statuses the
	// owner approved for committed opening stock. Empty (the default) blocks every
	// commit; checked against the template's statuses at start-up (cmd/api).
	OpeningStockApprovedStatuses string
}

// Attachment limits.
const (
	AttachmentMaxBytesLimit = 25 << 20
	minSigningKeyLength     = 32
)

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

	cfg.SessionIdleTimeout = p.duration("SESSION_IDLE_TIMEOUT", 2*time.Hour)
	cfg.SessionAbsoluteTimeout = p.duration("SESSION_ABSOLUTE_TIMEOUT", 12*time.Hour)
	// Secure cookies need HTTPS; plain-HTTP local development and tests may turn them off.
	cfg.SessionCookieSecure = p.bool("SESSION_COOKIE_SECURE", cfg.AppEnv != EnvDevelopment && cfg.AppEnv != EnvTest)
	cfg.LoginMaxFailures = p.int("LOGIN_MAX_FAILURES", 5)
	cfg.LoginMaxFailuresPerIP = p.int("LOGIN_MAX_FAILURES_PER_IP", 20)
	cfg.LoginFailureWindow = p.duration("LOGIN_FAILURE_WINDOW", 15*time.Minute)
	cfg.RBACAllowProposedGrants = p.bool("RBAC_ALLOW_PROPOSED_GRANTS", false)

	cfg.AttachmentStorageDriver = p.str("ATTACHMENT_STORAGE_DRIVER", "local")
	defaultDir := ""
	if cfg.AppEnv == EnvDevelopment || cfg.AppEnv == EnvTest {
		defaultDir = "var/attachments"
	}
	cfg.AttachmentStorageDir = p.str("ATTACHMENT_STORAGE_DIR", defaultDir)
	cfg.AttachmentMaxBytes = int64(p.int("ATTACHMENT_MAX_BYTES", 10<<20))
	cfg.AttachmentLinkTTL = p.duration("ATTACHMENT_LINK_TTL", 5*time.Minute)
	cfg.AttachmentSigningKey = strings.TrimSpace(getenv("ATTACHMENT_SIGNING_KEY"))
	cfg.BusinessTimeZone = p.str("BUSINESS_TIME_ZONE", "Asia/Kolkata")
	cfg.ImportMaxBytes = int64(p.int("IMPORT_MAX_BYTES", 5<<20))
	cfg.ImportMaxRows = p.int("IMPORT_MAX_ROWS", 5000)
	cfg.ImportPreviewTTL = p.duration("IMPORT_PREVIEW_TTL", 7*24*time.Hour)
	cfg.OpeningStockApprovedStatuses = p.str("OPENING_STOCK_APPROVED_STATUSES", "")

	errs := p.errs
	errs = append(errs, cfg.validate()...)
	if len(errs) > 0 {
		return Config{}, fmt.Errorf("invalid configuration:\n  - %w", errors.Join(errs...))
	}
	return cfg, nil
}

func (c Config) validate() []error {
	var errs []error
	if c.ImportMaxBytes < 1024 || c.ImportMaxBytes > 50<<20 {
		errs = append(errs, fmt.Errorf("IMPORT_MAX_BYTES must be from 1024 to %d (got %d)", 50<<20, c.ImportMaxBytes))
	}
	if c.ImportMaxRows < 1 || c.ImportMaxRows > 50000 {
		errs = append(errs, fmt.Errorf("IMPORT_MAX_ROWS must be from 1 to 50000 (got %d)", c.ImportMaxRows))
	}
	if c.ImportPreviewTTL < time.Hour {
		errs = append(errs, fmt.Errorf("IMPORT_PREVIEW_TTL must be at least 1h (got %s)", c.ImportPreviewTTL))
	}
	if _, err := time.LoadLocation(c.BusinessTimeZone); err != nil || c.BusinessTimeZone == "" {
		errs = append(errs, fmt.Errorf("BUSINESS_TIME_ZONE must be an IANA time zone name such as Asia/Kolkata (got %q)", c.BusinessTimeZone))
	}

	switch c.AppEnv {
	case EnvDevelopment, EnvTest, EnvStaging, EnvProduction:
	default:
		errs = append(errs, fmt.Errorf("APP_ENV must be one of development, test, staging, production (got %q)", c.AppEnv))
	}
	if c.Port < 1 || c.Port > 65535 {
		errs = append(errs, fmt.Errorf("APP_PORT must be between 1 and 65535 (got %d)", c.Port))
	}

	if err := ValidateDatabaseURL(c.DatabaseURL, c.AppEnv); err != nil {
		errs = append(errs, err)
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

	if c.SessionIdleTimeout < time.Minute {
		errs = append(errs, errors.New("SESSION_IDLE_TIMEOUT must be at least 1m"))
	}
	if c.SessionAbsoluteTimeout < c.SessionIdleTimeout || c.SessionAbsoluteTimeout > 7*24*time.Hour {
		errs = append(errs, errors.New("SESSION_ABSOLUTE_TIMEOUT must be between SESSION_IDLE_TIMEOUT and 168h"))
	}
	if (c.AppEnv == EnvStaging || c.AppEnv == EnvProduction) && !c.SessionCookieSecure {
		errs = append(errs, fmt.Errorf("SESSION_COOKIE_SECURE must be true in %s (HTTPS only)", c.AppEnv))
	}
	if c.LoginMaxFailures < 1 || c.LoginMaxFailuresPerIP < c.LoginMaxFailures {
		errs = append(errs, errors.New("LOGIN_MAX_FAILURES must be at least 1 and LOGIN_MAX_FAILURES_PER_IP at least LOGIN_MAX_FAILURES"))
	}
	if c.RBACAllowProposedGrants && c.AppEnv != EnvDevelopment {
		errs = append(errs, fmt.Errorf("RBAC_ALLOW_PROPOSED_GRANTS must be false in %s: it is for local development only; elsewhere only owner-approved grants count", c.AppEnv))
	}
	if c.LoginFailureWindow < time.Minute {
		errs = append(errs, errors.New("LOGIN_FAILURE_WINDOW must be at least 1m"))
	}
	errs = append(errs, c.validateAttachments()...)
	return errs
}

func (c Config) validateAttachments() []error {
	var errs []error
	if c.AttachmentStorageDriver != "local" {
		errs = append(errs, fmt.Errorf("ATTACHMENT_STORAGE_DRIVER must be local (got %q); an object-storage driver is pending the hosting decision", c.AttachmentStorageDriver))
	}
	if c.AttachmentStorageDir == "" {
		errs = append(errs, fmt.Errorf("ATTACHMENT_STORAGE_DIR is required in %s: a private directory outside any web root", c.AppEnv))
	}
	if c.AttachmentMaxBytes < 1024 || c.AttachmentMaxBytes > AttachmentMaxBytesLimit {
		errs = append(errs, fmt.Errorf("ATTACHMENT_MAX_BYTES must be between 1024 and %d", AttachmentMaxBytesLimit))
	}
	if c.AttachmentLinkTTL < 30*time.Second || c.AttachmentLinkTTL > 15*time.Minute {
		errs = append(errs, errors.New("ATTACHMENT_LINK_TTL must be between 30s and 15m (links must be short-lived)"))
	}
	needKey := c.AppEnv == EnvStaging || c.AppEnv == EnvProduction
	if (needKey || c.AttachmentSigningKey != "") && len(c.AttachmentSigningKey) < minSigningKeyLength {
		errs = append(errs, fmt.Errorf("ATTACHMENT_SIGNING_KEY must be at least %d characters (required in staging/production)", minSigningKeyLength))
	}
	return errs
}

// ValidateDatabaseURL checks a PostgreSQL connection URL for the given APP_ENV.
// It is shared by the API, cmd/migrate and cmd/seed so every entry point
// rejects bad configuration the same way. Errors never echo the value, which
// contains credentials.
func ValidateDatabaseURL(raw, appEnv string) error {
	if raw == "" {
		return errors.New("DATABASE_URL is required, e.g. postgres://user:password@localhost:5432/maa_ambey_inventory?sslmode=disable")
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") || u.Host == "" {
		return errors.New("DATABASE_URL must be a postgres:// or postgresql:// URL with a host")
	}
	if strings.Trim(u.Path, "/") == "" {
		return errors.New("DATABASE_URL must name a database, e.g. .../maa_ambey_inventory")
	}
	if appEnv == EnvProduction && u.Query().Get("sslmode") == "disable" {
		return errors.New("DATABASE_URL must not use sslmode=disable in production")
	}
	return nil
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
