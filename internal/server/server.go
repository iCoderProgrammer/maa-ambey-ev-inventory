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
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/modules/attachments"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/modules/auth"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/modules/catalogue"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/modules/health"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/modules/pricing"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/modules/purchasing"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/modules/receiving"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/modules/stock"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/modules/suppliers"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/modules/vehicles"
)

// APIPrefix is the versioned base path for all business endpoints.
const APIPrefix = "/api/v1"

// Deps are the runtime dependencies shared by feature modules.
type Deps struct {
	Config config.Config
	Log    *slog.Logger
	DB     health.Pinger
	Schema health.SchemaChecker
	Auth   *auth.Handler
	Access *access.Handler
	// Attachments serves private files (B-09).
	Attachments *attachments.Handler
	// Catalogue serves brands, models, variants and colours (C-02).
	Catalogue *catalogue.Handler
	// Pricing serves price lists and price history (C-04).
	Pricing *pricing.Handler
	// Suppliers serves the supplier master (D-01).
	Suppliers *suppliers.Handler
	// PurchaseOrders serves the purchase order lifecycle (D-02).
	PurchaseOrders *purchasing.Handler
	// GoodsReceipts serves the goods receipt draft workflow (D-03).
	GoodsReceipts *receiving.Handler
	// VehicleUnits serves the vehicle unit registry (D-04).
	VehicleUnits *vehicles.Handler
	// StockMovements serves the stock movement operations (E-01).
	StockMovements *stock.Handler
	Version        string
}

// NewRouter wires middleware and routes. Feature modules register themselves
// on the /api/v1 group here.
func NewRouter(d Deps) (*gin.Engine, error) {
	if d.DB == nil || d.Schema == nil || d.Auth == nil || d.Access == nil || d.Attachments == nil || d.Catalogue == nil || d.Pricing == nil || d.Suppliers == nil || d.PurchaseOrders == nil || d.GoodsReceipts == nil || d.VehicleUnits == nil || d.StockMovements == nil {
		return nil, errors.New("server: DB, Schema, Auth, Access, Attachments, Catalogue, Pricing, Suppliers, PurchaseOrders, GoodsReceipts, VehicleUnits and StockMovements dependencies are required")
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
	d.Attachments.Register(api)
	d.Catalogue.Register(api)
	d.Pricing.Register(api)
	d.Suppliers.Register(api)
	d.PurchaseOrders.Register(api)
	d.GoodsReceipts.Register(api)
	d.VehicleUnits.Register(api)
	d.StockMovements.Register(api)

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
