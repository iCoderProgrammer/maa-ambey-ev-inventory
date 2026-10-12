// Package server builds the Gin router and HTTP server.
package server

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/config"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/httpx"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/middleware"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/modules/access"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/modules/auth"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/modules/health"
)

// APIPrefix is the versioned base path for all business endpoints.
const APIPrefix = "/api/v1"

// Deps are the runtime dependencies shared by feature modules.
type Deps struct {
	Config  config.Config
	Log     *slog.Logger
	DB      health.Pinger
	Schema  health.SchemaChecker
	Auth    *auth.Handler
	Access  *access.Handler
	Version string
}

// NewRouter wires middleware and routes. Feature modules register themselves
// on the /api/v1 group here.
func NewRouter(d Deps) (*gin.Engine, error) {
	if d.DB == nil || d.Schema == nil || d.Auth == nil || d.Access == nil {
		return nil, errors.New("server: DB, Schema, Auth and Access dependencies are required")
	}
	if d.Config.AppEnv == config.EnvDevelopment {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.HandleMethodNotAllowed = true
	// Only trust X-Forwarded-For from configured proxies so rate limiting and
	// logs cannot be spoofed by clients.
	if err := r.SetTrustedProxies(d.Config.TrustedProxies); err != nil {
		return nil, err
	}

	// RequestLogger wraps Recovery so a recovered panic still produces an
	// access-log line with status 500.
	r.Use(
		middleware.RequestID(),
		middleware.RequestLogger(d.Log),
		middleware.Recovery(d.Log),
		middleware.CORS(d.Config.CORSAllowedOrigins),
		middleware.Timeout(d.Config.RequestTimeout),
	)

	r.NoRoute(func(c *gin.Context) {
		httpx.Error(c, http.StatusNotFound, httpx.CodeNotFound, "The requested resource was not found.")
	})
	r.NoMethod(func(c *gin.Context) {
		httpx.Error(c, http.StatusMethodNotAllowed, httpx.CodeMethodNotAllowed, "This method is not allowed for the requested resource.")
	})

	if d.Config.SwaggerEnabled {
		r.GET("/swagger", func(c *gin.Context) { c.Redirect(http.StatusFound, "/swagger/index.html") })
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	api := r.Group(APIPrefix)
	// Deny by default (B-06): every /api/v1 route needs a valid session, and
	// state-changing requests its CSRF token, unless listed as public here.
	// Feature routes then add access.Require(permission) (B-07): a session
	// alone grants nothing beyond /me and the auth endpoints.
	public := append([]string{
		"GET " + APIPrefix + "/health",
		"GET " + APIPrefix + "/ready",
	}, auth.PublicRoutes(APIPrefix)...)
	api.Use(
		middleware.NewRateLimiter(d.Config.RateLimitRPS, d.Config.RateLimitBurst).Middleware(),
		d.Auth.RequireSession(public...),
	)

	health.NewHandler(d.DB, d.Schema, d.Version, d.Log).Register(api)
	d.Auth.Register(api)
	d.Access.Register(api)

	return r, nil
}

// NewHTTPServer wraps the router with conservative transport timeouts.
func NewHTTPServer(cfg config.Config, h http.Handler) *http.Server {
	return &http.Server{
		Addr:              cfg.Addr(),
		Handler:           h,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       cfg.RequestTimeout + 5*time.Second,
		WriteTimeout:      cfg.RequestTimeout + 5*time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
}
