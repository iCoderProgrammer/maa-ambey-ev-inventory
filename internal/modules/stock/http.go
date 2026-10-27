package stock

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/audit"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/httpx"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/modules/access"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/modules/auth"
)

// IdempotencyHeader is the required header of every movement request.
const IdempotencyHeader = "Idempotency-Key"

// maxBody is the largest request body (a version and a note).
const maxBody = 16 << 10

// Handler is the HTTP adapter for stock movements.
type Handler struct {
	svc    *Service
	access *access.Handler
	log    *slog.Logger
}

// NewHandler builds the adapter.
func NewHandler(svc *Service, accessH *access.Handler, log *slog.Logger) *Handler {
	return &Handler{svc: svc, access: accessH, log: log}
}

// Register mounts the routes: the two moves no other workflow owns. There is no
// route that writes an arbitrary movement or status, and none that edits or
// deletes one; listing movements is E-02.
func (h *Handler) Register(rg *gin.RouterGroup) {
	display := h.access.Require(access.PermVehicleUnitDisplayManage)
	rg.POST("/vehicle-units/:id/put-on-display", display, h.PutOnDisplay)
	rg.POST("/vehicle-units/:id/take-off-display", display, h.TakeOffDisplay)
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
	dec := json.NewDecoder(io.LimitReader(c.Request.Body, maxBody))
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
		httpx.Error(c, http.StatusBadRequest, httpx.CodeBadRequest, "The request body must be one JSON object (at most 16 KB).")
	}
	return false
}

func (h *Handler) respond(c *gin.Context, res MoveResult, replayed bool, err error) {
	var verr *ValidationError
	var inv *InvalidStateError
	switch {
	case err == nil:
		c.Header("Cache-Control", "no-store")
		if replayed {
			c.Header("Idempotent-Replayed", "true")
		}
		c.JSON(http.StatusOK, httpx.DataResponse{Data: res})
	case errors.Is(err, ErrForbidden):
		access.Forbidden(c)
	case errors.Is(err, ErrNotFound):
		httpx.Error(c, http.StatusNotFound, httpx.CodeNotFound, "The vehicle unit was not found.")
	case errors.As(err, &verr):
		out := make([]httpx.FieldError, len(verr.Fields))
		for i, f := range verr.Fields {
			out[i] = httpx.FieldError{Field: f.Field, Message: f.Message}
		}
		httpx.Error(c, http.StatusUnprocessableEntity, httpx.CodeValidationFailed, "Check the highlighted fields.", out...)
	case errors.Is(err, ErrVersionConflict):
		httpx.Error(c, http.StatusConflict, httpx.CodeVersionConflict,
			"This vehicle unit was changed by someone else. Reload it and try again. Nothing was changed.")
	case errors.As(err, &inv):
		httpx.Error(c, http.StatusConflict, httpx.CodeInvalidState,
			"This vehicle unit is "+inv.Current+"; this needs "+inv.Required+". Nothing was changed.")
	case errors.Is(err, ErrChassisRequired):
		httpx.Error(c, http.StatusConflict, httpx.CodeInvalidState,
			"This vehicle unit has no chassis/frame number yet. Nothing was changed.")
	case errors.Is(err, ErrIdempotencyKeyReused):
		httpx.Error(c, http.StatusConflict, httpx.CodeIdempotencyKeyReused,
			"This Idempotency-Key was already used for a different request. Send a new key. Nothing was changed.")
	default:
		h.log.ErrorContext(c.Request.Context(), "stock movement request failed",
			slog.String("request_id", c.GetString(httpx.RequestIDKey)), slog.Any("error", err))
		httpx.InternalError(c)
	}
}

func (h *Handler) display(c *gin.Context, on bool) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	var in DisplayInput
	if !h.decode(c, &in) {
		return
	}
	res, replayed, err := h.svc.Display(c.Request.Context(), a, h.actor(c), c.GetString(httpx.RequestIDKey), c.Param("id"), on,
		in, c.GetHeader(IdempotencyHeader))
	h.respond(c, res, replayed, err)
}

// PutOnDisplay godoc
//
//	@Summary		Put a vehicle unit on display
//	@Description	Moves an AVAILABLE unit to DISPLAY by recording one DISPLAY_ON stock movement (quantity 1 -> 1, location unchanged, source STAFF_ACTION). The unit's status, version and the movement are written in one transaction with an audit event (vehicle_unit.moved). version must be the unit's current version. Idempotency-Key is required: a repeat by the same user with the same key and body returns the first movement with 200 and Idempotent-Replayed: true (unit as it is now) and changes nothing; the same key with a different request is 409 IDEMPOTENCY_KEY_REUSED. A unit at a location outside the caller's scope is 404. Requires vehicle_units.display.manage and X-CSRF-Token.
//	@Tags			stock-movements
//	@Accept			json
//	@Produce		json
//	@Param			id				path		string					true	"Vehicle unit ID"
//	@Param			Idempotency-Key	header		string					true	"8-100 letters, digits, - or _"
//	@Param			body			body		stock.DisplayInput		true	"Current version and an optional note"
//	@Success		200				{object}	httpx.DataResponse{data=stock.MoveResult}	"Recorded, or a replay (Idempotent-Replayed: true)"
//	@Failure		400				{object}	httpx.ErrorResponse	"BAD_REQUEST"
//	@Failure		401				{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403				{object}	httpx.ErrorResponse	"FORBIDDEN or CSRF_FAILED"
//	@Failure		404				{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Failure		409				{object}	httpx.ErrorResponse	"VERSION_CONFLICT, INVALID_STATE (not AVAILABLE) or IDEMPOTENCY_KEY_REUSED"
//	@Failure		415				{object}	httpx.ErrorResponse	"UNSUPPORTED_MEDIA_TYPE"
//	@Failure		422				{object}	httpx.ErrorResponse	"VALIDATION_FAILED (version, note, Idempotency-Key)"
//	@Router			/vehicle-units/{id}/put-on-display [post]
func (h *Handler) PutOnDisplay(c *gin.Context) { h.display(c, true) }

// TakeOffDisplay godoc
//
//	@Summary		Take a vehicle unit off display
//	@Description	Moves a DISPLAY unit back to AVAILABLE by recording one DISPLAY_OFF stock movement. Same rules as put-on-display: version, Idempotency-Key, one transaction with the audit event, replays change nothing, location scope. Requires vehicle_units.display.manage and X-CSRF-Token.
//	@Tags			stock-movements
//	@Accept			json
//	@Produce		json
//	@Param			id				path		string					true	"Vehicle unit ID"
//	@Param			Idempotency-Key	header		string					true	"8-100 letters, digits, - or _"
//	@Param			body			body		stock.DisplayInput		true	"Current version and an optional note"
//	@Success		200				{object}	httpx.DataResponse{data=stock.MoveResult}	"Recorded, or a replay (Idempotent-Replayed: true)"
//	@Failure		400				{object}	httpx.ErrorResponse	"BAD_REQUEST"
//	@Failure		401				{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403				{object}	httpx.ErrorResponse	"FORBIDDEN or CSRF_FAILED"
//	@Failure		404				{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Failure		409				{object}	httpx.ErrorResponse	"VERSION_CONFLICT, INVALID_STATE (not DISPLAY) or IDEMPOTENCY_KEY_REUSED"
//	@Failure		415				{object}	httpx.ErrorResponse	"UNSUPPORTED_MEDIA_TYPE"
//	@Failure		422				{object}	httpx.ErrorResponse	"VALIDATION_FAILED (version, note, Idempotency-Key)"
//	@Router			/vehicle-units/{id}/take-off-display [post]
func (h *Handler) TakeOffDisplay(c *gin.Context) { h.display(c, false) }
