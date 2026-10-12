// Package health exposes liveness and readiness endpoints.
package health

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/httpx"
)

// ServiceName identifies this API in health responses.
const ServiceName = "maa-ambey-inventory-api"

// Pinger is satisfied by *pgxpool.Pool.
type Pinger interface {
	Ping(ctx context.Context) error
}

// Status is the payload of GET /health.
type Status struct {
	Status  string    `json:"status" example:"ok"`
	Service string    `json:"service" example:"maa-ambey-inventory-api"`
	Version string    `json:"version" example:"dev"`
	Time    time.Time `json:"time" example:"2026-10-12T10:00:00Z"`
} //@name HealthStatus

// Readiness is the payload of GET /ready. Check values are "up" or "down";
// failure reasons are logged server-side, never returned.
type Readiness struct {
	Status string            `json:"status" example:"ready" enums:"ready,not_ready"`
	Checks map[string]string `json:"checks"`
} //@name Readiness

// Handler serves the health endpoints.
type Handler struct {
	db      Pinger
	version string
	log     *slog.Logger
	now     func() time.Time
}

// NewHandler builds a health handler.
func NewHandler(db Pinger, version string, log *slog.Logger) *Handler {
	return &Handler{db: db, version: version, log: log, now: time.Now}
}

// Register mounts the routes on the given group (e.g. /api/v1).
func (h *Handler) Register(rg *gin.RouterGroup) {
	rg.GET("/health", h.Health)
	rg.GET("/ready", h.Ready)
}

// Health godoc
//
//	@Summary		Liveness check
//	@Description	Confirms the API process is running. Does not check dependencies.
//	@Tags			health
//	@Produce		json
//	@Success		200	{object}	httpx.DataResponse{data=health.Status}
//	@Failure		429	{object}	httpx.ErrorResponse
//	@Router			/health [get]
func (h *Handler) Health(c *gin.Context) {
	httpx.OK(c, Status{Status: "ok", Service: ServiceName, Version: h.version, Time: h.now().UTC()})
}

// Ready godoc
//
//	@Summary		Readiness check
//	@Description	Reports whether required dependencies (PostgreSQL) are reachable. Returns 503 when not ready.
//	@Tags			health
//	@Produce		json
//	@Success		200	{object}	httpx.DataResponse{data=health.Readiness}
//	@Failure		429	{object}	httpx.ErrorResponse
//	@Failure		503	{object}	httpx.DataResponse{data=health.Readiness}
//	@Router			/ready [get]
func (h *Handler) Ready(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	res := Readiness{Status: "ready", Checks: map[string]string{"database": "up"}}
	status := http.StatusOK
	if err := h.db.Ping(ctx); err != nil {
		h.log.WarnContext(ctx, "readiness check failed",
			slog.String("request_id", c.GetString(httpx.RequestIDKey)),
			slog.String("dependency", "database"),
			slog.Any("error", err))
		res.Status = "not_ready"
		res.Checks["database"] = "down"
		status = http.StatusServiceUnavailable
	}
	c.JSON(status, httpx.DataResponse{Data: res})
}
