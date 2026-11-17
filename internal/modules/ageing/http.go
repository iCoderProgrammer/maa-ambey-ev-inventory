package ageing

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/audit"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/httpx"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/access"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/auth"
)

// Handler is the HTTP adapter for the band configuration.
type Handler struct {
	svc    *Service
	access *access.Handler
	log    *slog.Logger
}

// NewHandler builds the adapter.
func NewHandler(svc *Service, accessH *access.Handler, log *slog.Logger) *Handler {
	return &Handler{svc: svc, access: accessH, log: log}
}

// Register mounts the routes. Ages themselves are part of the vehicle unit
// routes (D-08 list/detail, ageing summary); nothing here changes stock.
func (h *Handler) Register(rg *gin.RouterGroup) {
	rg.GET("/stock-ageing/configuration", h.access.RequireAny(access.PermVehicleUnitView, access.PermAgeingConfigure), h.Get)
	configure := h.access.Require(access.PermAgeingConfigure)
	rg.POST("/stock-ageing/band-sets", configure, h.Propose)
	rg.POST("/stock-ageing/band-sets/approve", configure, h.Approve)
}

func (h *Handler) actor(c *gin.Context) audit.Actor {
	p, _ := auth.CurrentPrincipal(c)
	return audit.UserActor(p.UserID, p.Username)
}

func (h *Handler) decode(c *gin.Context, dst any) bool {
	if c.ContentType() != "application/json" {
		httpx.Error(c, http.StatusUnsupportedMediaType, httpx.CodeUnsupportedMediaType, "Send the request body as application/json.")
		return false
	}
	dec := json.NewDecoder(io.LimitReader(c.Request.Body, 32<<10))
	dec.DisallowUnknownFields()
	err := dec.Decode(dst)
	if err == nil && dec.More() {
		err = errors.New("trailing data")
	}
	if err == nil {
		return true
	}
	var typeErr *json.UnmarshalTypeError
	switch {
	case errors.As(err, &typeErr):
		httpx.Error(c, http.StatusUnprocessableEntity, httpx.CodeValidationFailed, "Check the highlighted fields.",
			httpx.FieldError{Field: typeErr.Field, Message: "has the wrong type"})
	case strings.HasPrefix(err.Error(), "json: unknown field "):
		field := strings.Trim(strings.TrimPrefix(err.Error(), "json: unknown field "), `"`)
		httpx.Error(c, http.StatusUnprocessableEntity, httpx.CodeValidationFailed, "Check the highlighted fields.",
			httpx.FieldError{Field: field, Message: "cannot be set here"})
	default:
		httpx.Error(c, http.StatusBadRequest, httpx.CodeBadRequest, "The request body must be one JSON object (at most 32 KB).")
	}
	return false
}

func (h *Handler) respond(c *gin.Context, status int, v any, err error) {
	var verr *ValidationError
	switch {
	case err == nil:
		c.Header("Cache-Control", "no-store")
		c.JSON(status, httpx.DataResponse{Data: v})
	case errors.Is(err, ErrForbidden):
		access.Forbidden(c)
	case errors.As(err, &verr):
		out := make([]httpx.FieldError, len(verr.Fields))
		for i, f := range verr.Fields {
			out[i] = httpx.FieldError{Field: f.Field, Message: f.Message}
		}
		httpx.Error(c, http.StatusUnprocessableEntity, httpx.CodeValidationFailed, "Check the highlighted fields.", out...)
	case errors.Is(err, ErrVersionConflict):
		httpx.Error(c, http.StatusConflict, httpx.CodeVersionConflict,
			"The band configuration was changed by someone else. Reload it and try again. Nothing was saved.")
	case errors.Is(err, ErrNotProposed), errors.Is(err, ErrNotConfigured):
		httpx.Error(c, http.StatusConflict, httpx.CodeInvalidState,
			"Only the current proposed band configuration can be approved. Reload it. Nothing was changed.")
	case errors.Is(err, ErrSelfApproval):
		httpx.Error(c, http.StatusForbidden, httpx.CodeSelfApproval,
			"You proposed this band configuration, so someone else must approve it. Nothing was changed.")
	default:
		h.log.ErrorContext(c.Request.Context(), "stock ageing request failed",
			slog.String("request_id", c.GetString(httpx.RequestIDKey)), slog.Any("error", err))
		httpx.InternalError(c)
	}
}

// Get godoc
//
//	@Summary		Stock ageing configuration
//	@Description	The current ageing band configuration (latest version, PROPOSED until approved), the business time zone and today's business date ages are counted to, and the boundary rule: a band covers minDays <= age <= maxDays in whole days (both inclusive), the last band is open-ended, age = business date - origin date. configured is false (bandSet null) when no configuration exists; ages are then still calculated, without bands. Read-only. Requires vehicle_units.view or stock.ageing.configure.
//	@Tags			stock-ageing
//	@Produce		json
//	@Success		200	{object}	httpx.DataResponse{data=ageing.Configuration}
//	@Failure		401	{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403	{object}	httpx.ErrorResponse	"FORBIDDEN"
//	@Router			/stock-ageing/configuration [get]
func (h *Handler) Get(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	cfg, err := h.svc.Configuration(c.Request.Context(), a)
	h.respond(c, http.StatusOK, cfg, err)
}

// Propose godoc
//
//	@Summary		Propose an ageing band configuration
//	@Description	Records a new configuration version (status PROPOSED) after expectedVersion (409 VERSION_CONFLICT otherwise). Bands: 1-20, keys unique (1-32 lower-case letters, digits, - or _), labels 1-60 characters, whole days 0-36500; the first starts at 0, each starts the day after the previous one ends (no gap or overlap), only the last has no maxDays (open-ended). source names the document or decision the thresholds come from. Earlier versions are kept unchanged and no stock record is rewritten (ages and bands are calculated on read). Audited (stock_ageing.bands_proposed). Requires stock.ageing.configure and X-CSRF-Token.
//	@Tags			stock-ageing
//	@Accept			json
//	@Produce		json
//	@Param			X-CSRF-Token	header		string					true	"CSRF token from the session"
//	@Param			body			body		ageing.ProposeInput		true	"Configuration"
//	@Success		201				{object}	httpx.DataResponse{data=ageing.BandSet}
//	@Failure		401				{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403				{object}	httpx.ErrorResponse	"FORBIDDEN or CSRF_FAILED"
//	@Failure		409				{object}	httpx.ErrorResponse	"VERSION_CONFLICT"
//	@Failure		415				{object}	httpx.ErrorResponse	"UNSUPPORTED_MEDIA_TYPE"
//	@Failure		422				{object}	httpx.ErrorResponse	"VALIDATION_FAILED (expectedVersion, source, bands, bands[i].key|label|minDays|maxDays)"
//	@Router			/stock-ageing/band-sets [post]
func (h *Handler) Propose(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	var in ProposeInput
	if !h.decode(c, &in) {
		return
	}
	set, err := h.svc.Propose(c.Request.Context(), a, h.actor(c), c.GetString(httpx.RequestIDKey), in)
	h.respond(c, http.StatusCreated, set, err)
}

// Approve godoc
//
//	@Summary		Approve the proposed ageing band configuration
//	@Description	Approves the current version while it is PROPOSED (409 INVALID_STATE otherwise). The person who proposed it cannot approve it (403 SELF_APPROVAL_NOT_ALLOWED, audited DENIED); the migration's LED-008 proposal can be approved by any holder of the permission. Only the approval fields change. Audited (stock_ageing.bands_approved). Requires stock.ageing.configure and X-CSRF-Token.
//	@Tags			stock-ageing
//	@Accept			json
//	@Produce		json
//	@Param			X-CSRF-Token	header		string					true	"CSRF token from the session"
//	@Param			body			body		ageing.ApproveInput		true	"Version to approve"
//	@Success		200				{object}	httpx.DataResponse{data=ageing.BandSet}
//	@Failure		401				{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403				{object}	httpx.ErrorResponse	"FORBIDDEN, SELF_APPROVAL_NOT_ALLOWED or CSRF_FAILED"
//	@Failure		409				{object}	httpx.ErrorResponse	"INVALID_STATE"
//	@Failure		415				{object}	httpx.ErrorResponse	"UNSUPPORTED_MEDIA_TYPE"
//	@Failure		422				{object}	httpx.ErrorResponse	"VALIDATION_FAILED (version, reference)"
//	@Router			/stock-ageing/band-sets/approve [post]
func (h *Handler) Approve(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	var in ApproveInput
	if !h.decode(c, &in) {
		return
	}
	set, err := h.svc.Approve(c.Request.Context(), a, h.actor(c), c.GetString(httpx.RequestIDKey), in)
	h.respond(c, http.StatusOK, set, err)
}
