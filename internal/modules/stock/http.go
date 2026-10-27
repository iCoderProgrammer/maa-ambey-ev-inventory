package stock

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
	// Parts stock (E-04). Balances change only through these two posts.
	partsView := h.access.Require(access.PermPartsView)
	rg.GET("/stock-items/:id/balances", partsView, h.PartsBalances)
	rg.GET("/stock-items/:id/stock-movements", partsView, h.PartsHistory)
	rg.POST("/stock-items/:id/receipts", h.access.Require(access.PermPartsReceive), h.ReceiveParts)
	rg.POST("/stock-items/:id/issues", h.access.Require(access.PermPartsIssue), h.IssueParts)
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

func (h *Handler) respond(c *gin.Context, res any, replayed bool, err error) {
	if err != nil {
		h.fail(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	if replayed {
		c.Header("Idempotent-Replayed", "true")
	}
	c.JSON(http.StatusOK, httpx.DataResponse{Data: res})
}

func (h *Handler) fail(c *gin.Context, err error) {
	var verr *ValidationError
	var inv *InvalidStateError
	switch {
	case errors.Is(err, ErrInsufficientStock):
		httpx.Error(c, http.StatusConflict, httpx.CodeInsufficientStock,
			"There is not enough stock of this item at this location. Nothing was changed.",
			httpx.FieldError{Field: "quantity", Message: "is more than the available balance"})
	case errors.Is(err, ErrSerialNotSupported):
		httpx.Error(c, http.StatusConflict, httpx.CodeTrackingNotSupported,
			"Serial-tracked items cannot be received or issued yet: there is no serial register. Nothing was changed.")
	case errors.Is(err, ErrItemArchived):
		httpx.Error(c, http.StatusConflict, httpx.CodeInvalidState, "This SKU is archived and takes no new stock movements. Nothing was changed.")
	case errors.Is(err, ErrForbidden):
		access.Forbidden(c)
	case errors.Is(err, ErrNotFound):
		httpx.Error(c, http.StatusNotFound, httpx.CodeNotFound, "The record was not found.")
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

// PartsBalances godoc
//
//	@Summary		Stock balances of a part
//	@Description	The SKU's quantity at each location in the caller's scope that has (or had) stock; a location never stocked has no row (its balance is 0). Balances are computed by the server from receipts and issues and cannot be set. Requires stock.parts.view.
//	@Tags			parts-stock
//	@Produce		json
//	@Param			id	path		string	true	"SKU ID"
//	@Success		200	{object}	httpx.DataResponse{data=[]stock.PartsBalance}
//	@Failure		401	{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403	{object}	httpx.ErrorResponse	"FORBIDDEN"
//	@Failure		404	{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Router			/stock-items/{id}/balances [get]
func (h *Handler) PartsBalances(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	b, err := h.svc.PartsBalances(c.Request.Context(), a, c.Param("id"))
	h.respond(c, b, false, err)
}

// PartsHistory godoc
//
//	@Summary		Receipts and issues of a part
//	@Description	Newest first, at the locations in the caller's scope; filters locationId (must be in scope) and type. Each entry has the quantity, the balance before and after, unit, source, reference, reason, actor and server time. supplierId/supplierName are null without suppliers.view. Requires stock.parts.view.
//	@Tags			parts-stock
//	@Produce		json
//	@Param			id			path	string	true	"SKU ID"
//	@Param			locationId	query	string	false	"Location ID (in the caller's scope)"
//	@Param			type		query	string	false	"Movement type"	Enums(PARTS_RECEIPT, PARTS_ISSUE)
//	@Param			page		query	int		false	"Page (from 1)"
//	@Param			limit		query	int		false	"Page size (1-100, default 25)"
//	@Success		200	{object}	httpx.DataResponse{data=stock.PartsPage}
//	@Failure		401	{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403	{object}	httpx.ErrorResponse	"FORBIDDEN"
//	@Failure		404	{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Failure		422	{object}	httpx.ErrorResponse	"VALIDATION_FAILED (locationId, type, page, limit)"
//	@Router			/stock-items/{id}/stock-movements [get]
func (h *Handler) PartsHistory(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	p, err := h.svc.PartsHistory(c.Request.Context(), a, c.Param("id"), PartsHistoryQuery{LocationID: c.Query("locationId"),
		Type: c.Query("type"), Page: c.Query("page"), Limit: c.Query("limit")})
	h.respond(c, p, false, err)
}

// ReceiveParts godoc
//
//	@Summary		Receive parts from a supplier
//	@Description	Adds the quantity to the SKU's balance at the location by recording one PARTS_RECEIPT movement (balance, movement and audit stock_item.received in one transaction). The SKU must be ACTIVE and QUANTITY-tracked (SERIAL: 409 TRACKING_NOT_SUPPORTED) and uom must be its unit (no conversion); quantity > 0 with up to 3 decimals; supplierId (not archived) and sourceReference (challan/invoice number) are required. Idempotency-Key is required: the same key and body by the same user returns the first result with Idempotent-Replayed: true and changes nothing; the same key with another request is 409 IDEMPOTENCY_KEY_REUSED. Requires stock.parts.receive, suppliers.view and X-CSRF-Token; the location must be in the caller's scope.
//	@Tags			parts-stock
//	@Accept			json
//	@Produce		json
//	@Param			id				path		string					true	"SKU ID"
//	@Param			Idempotency-Key	header		string					true	"8-100 letters, digits, - or _"
//	@Param			body			body		stock.PartsReceiptInput	true	"Receipt"
//	@Success		200				{object}	httpx.DataResponse{data=stock.PartsResult}	"Recorded, or a replay (Idempotent-Replayed: true)"
//	@Failure		400				{object}	httpx.ErrorResponse	"BAD_REQUEST"
//	@Failure		401				{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403				{object}	httpx.ErrorResponse	"FORBIDDEN or CSRF_FAILED"
//	@Failure		404				{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Failure		409				{object}	httpx.ErrorResponse	"INVALID_STATE (archived SKU), TRACKING_NOT_SUPPORTED or IDEMPOTENCY_KEY_REUSED"
//	@Failure		415				{object}	httpx.ErrorResponse	"UNSUPPORTED_MEDIA_TYPE"
//	@Failure		422				{object}	httpx.ErrorResponse	"VALIDATION_FAILED (locationId, quantity, uom, supplierId, sourceReference, note, Idempotency-Key)"
//	@Router			/stock-items/{id}/receipts [post]
func (h *Handler) ReceiveParts(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	var in PartsReceiptInput
	if !h.decode(c, &in) {
		return
	}
	res, replayed, err := h.svc.ReceiveParts(c.Request.Context(), a, h.actor(c), c.GetString(httpx.RequestIDKey), c.Param("id"),
		in, c.GetHeader(IdempotencyHeader))
	h.respond(c, res, replayed, err)
}

// IssueParts godoc
//
//	@Summary		Issue (consume) parts
//	@Description	Takes the quantity from the SKU's balance at the location by recording one PARTS_ISSUE movement, in one transaction with the audit event stock_item.issued. The balance is locked and checked: more than is there is 409 INSUFFICIENT_STOCK and nothing changes (the database also refuses a negative balance). Same SKU, unit, quantity, idempotency and scope rules as receipts; reason is required, sourceReference (e.g. job card) optional. Requires stock.parts.issue and X-CSRF-Token.
//	@Tags			parts-stock
//	@Accept			json
//	@Produce		json
//	@Param			id				path		string					true	"SKU ID"
//	@Param			Idempotency-Key	header		string					true	"8-100 letters, digits, - or _"
//	@Param			body			body		stock.PartsIssueInput	true	"Issue"
//	@Success		200				{object}	httpx.DataResponse{data=stock.PartsResult}	"Recorded, or a replay (Idempotent-Replayed: true)"
//	@Failure		400				{object}	httpx.ErrorResponse	"BAD_REQUEST"
//	@Failure		401				{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403				{object}	httpx.ErrorResponse	"FORBIDDEN or CSRF_FAILED"
//	@Failure		404				{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Failure		409				{object}	httpx.ErrorResponse	"INSUFFICIENT_STOCK, INVALID_STATE (archived SKU), TRACKING_NOT_SUPPORTED or IDEMPOTENCY_KEY_REUSED"
//	@Failure		415				{object}	httpx.ErrorResponse	"UNSUPPORTED_MEDIA_TYPE"
//	@Failure		422				{object}	httpx.ErrorResponse	"VALIDATION_FAILED (locationId, quantity, uom, reason, sourceReference, note, Idempotency-Key)"
//	@Router			/stock-items/{id}/issues [post]
func (h *Handler) IssueParts(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	var in PartsIssueInput
	if !h.decode(c, &in) {
		return
	}
	res, replayed, err := h.svc.IssueParts(c.Request.Context(), a, h.actor(c), c.GetString(httpx.RequestIDKey), c.Param("id"),
		in, c.GetHeader(IdempotencyHeader))
	h.respond(c, res, replayed, err)
}
