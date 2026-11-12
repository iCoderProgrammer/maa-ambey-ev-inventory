package vehicles

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/httpx"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/access"
)

// Handler is the HTTP adapter for vehicle units.
type Handler struct {
	svc    *Service
	access *access.Handler
	log    *slog.Logger
}

// NewHandler builds the adapter.
func NewHandler(svc *Service, accessH *access.Handler, log *slog.Logger) *Handler {
	return &Handler{svc: svc, access: accessH, log: log}
}

// Register mounts the routes. Read-only: units are created only by receipt
// posting (D-07), and change only through the stock movement ledger (package stock, E-01).
func (h *Handler) Register(rg *gin.RouterGroup) {
	view := h.access.Require(access.PermVehicleUnitView)
	rg.GET("/vehicle-units", view, h.List)
	rg.GET("/vehicle-units/:id", view, h.Get)
	rg.GET("/vehicle-units/:id/stock-movements", view, h.Movements)
}

// UnitPage is a page of vehicle units.
type UnitPage struct {
	Items []Unit `json:"items"`
	Page  int    `json:"page" example:"1"`
	Limit int    `json:"limit" example:"25"`
	Total int    `json:"total" example:"0"`
} //@name VehicleUnitPage

// MovementPage is a page of a unit's stock movements.
type MovementPage struct {
	Items []Movement `json:"items"`
	Page  int        `json:"page" example:"1"`
	Limit int        `json:"limit" example:"50"`
	Total int        `json:"total" example:"0"`
} //@name VehicleUnitMovementPage

func fieldErrors(fields []FieldProblem) []httpx.FieldError {
	out := make([]httpx.FieldError, len(fields))
	for i, f := range fields {
		out[i] = httpx.FieldError{Field: f.Field, Message: f.Message}
	}
	return out
}

func (h *Handler) respond(c *gin.Context, v any, err error) {
	var verr *ValidationError
	switch {
	case err == nil:
		c.Header("Cache-Control", "no-store")
		c.JSON(http.StatusOK, httpx.DataResponse{Data: v})
	case errors.Is(err, ErrForbidden):
		access.Forbidden(c)
	case errors.Is(err, ErrNotFound):
		httpx.Error(c, http.StatusNotFound, httpx.CodeNotFound, "The vehicle unit was not found.")
	case errors.As(err, &verr):
		httpx.Error(c, http.StatusUnprocessableEntity, httpx.CodeValidationFailed, "Check the highlighted fields.", fieldErrors(verr.Fields)...)
	default:
		h.log.ErrorContext(c.Request.Context(), "vehicle unit request failed",
			slog.String("request_id", c.GetString(httpx.RequestIDKey)), slog.Any("error", err))
		httpx.InternalError(c)
	}
}

// WriteDuplicateIdentifier answers 409 DUPLICATE_IDENTIFIER when err is a
// *DuplicateIdentifierError and reports whether it did. For the route that
// registers units (receipt posting, D-07): field is the request path of the
// conflicting identifier, e.g. "lines[0].units[1].chassisNumber". The message is
// generic and nothing about the already registered vehicle is returned.
func WriteDuplicateIdentifier(c *gin.Context, err error, field string) bool {
	var dup *DuplicateIdentifierError
	if !errors.As(err, &dup) {
		return false
	}
	httpx.Error(c, http.StatusConflict, httpx.CodeDuplicateIdentifier,
		"This identifier is already registered to another vehicle. Correct it and try again. Nothing was saved.",
		httpx.FieldError{Field: field, Message: "is already registered"})
	return true
}

// List godoc
//
//	@Summary		List vehicle units
//	@Description	Newest registration first by default (sort). q matches the unit number and the chassis/frame, motor, battery and charger identifiers after normalisation (trimmed, upper-case; part of a value is enough). Filters combine (AND). receivedFrom/receivedTo are inclusive dates; minAgeDays/maxAgeDays are inclusive whole days since the received date (ageDays, database date). Colour is not filterable: units carry no colour (not captured on receipt). Every sort ends with the unit ID, so pages are stable. saleable=true lists only AVAILABLE units: units pending PDI, on QUALITY_HOLD or SERVICE_HOLD, REJECTED, on DISPLAY, RESERVED, SOLD or DELIVERED are never saleable stock (INV-006/007, PO-006). Only units without a location or at a location in the caller's scope are listed; a locationId outside the scope is refused. unitCost and costCurrency are null, and costHidden true, without vehicle_units.cost.view. Requires vehicle_units.view.
//	@Tags			vehicle-units
//	@Produce		json
//	@Param			q				query	string	false	"Search unit number or identifier"
//	@Param			status			query	string	false	"Lifecycle status"	Enums(ORDERED, IN_TRANSIT, RECEIVED_PENDING_PDI, QUALITY_HOLD, AVAILABLE, DISPLAY, RESERVED, SERVICE_HOLD, SOLD, DELIVERED, REJECTED, RETURN_PENDING_INSPECTION, RETURNED_TO_SUPPLIER)
//	@Param			brandId			query	string	false	"Brand ID"
//	@Param			modelId			query	string	false	"Model ID"
//	@Param			variantId		query	string	false	"Variant ID"
//	@Param			supplierId		query	string	false	"Supplier ID"
//	@Param			purchaseOrderId	query	string	false	"Purchase order ID"
//	@Param			goodsReceiptId	query	string	false	"Goods receipt ID"
//	@Param			locationId		query	string	false	"Location ID (must be in the caller's scope)"
//	@Param			saleable		query	bool	false	"true: AVAILABLE units only (saleable, INV-006); false: every other status (pending PDI, held, rejected, display, ...)"
//	@Param			receivedFrom	query	string	false	"Received on or after (YYYY-MM-DD)"
//	@Param			receivedTo		query	string	false	"Received on or before (YYYY-MM-DD)"
//	@Param			minAgeDays		query	int		false	"At least this many days since received (0-36500)"
//	@Param			maxAgeDays		query	int		false	"At most this many days since received (0-36500)"
//	@Param			sort			query	string	false	"Order (default -createdAt)"	Enums(-createdAt, createdAt, -receivedDate, receivedDate, unitNumber, -unitNumber)
//	@Param			page			query	int		false	"Page (from 1)"
//	@Param			limit			query	int		false	"Page size (1-100, default 25)"
//	@Success		200	{object}	httpx.DataResponse{data=vehicles.UnitPage}
//	@Failure		401	{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403	{object}	httpx.ErrorResponse	"FORBIDDEN"
//	@Failure		422	{object}	httpx.ErrorResponse	"VALIDATION_FAILED (q, status, saleable, brandId, modelId, variantId, supplierId, purchaseOrderId, goodsReceiptId, locationId, receivedFrom, receivedTo, minAgeDays, maxAgeDays, sort, page, limit)"
//	@Router			/vehicle-units [get]
func (h *Handler) List(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	p, err := h.svc.List(c.Request.Context(), a, ListQuery{Query: c.Query("q"), Status: c.Query("status"),
		BrandID: c.Query("brandId"), ModelID: c.Query("modelId"), VariantID: c.Query("variantId"), SupplierID: c.Query("supplierId"),
		PurchaseOrderID: c.Query("purchaseOrderId"), GoodsReceiptID: c.Query("goodsReceiptId"),
		LocationID: c.Query("locationId"), Saleable: c.Query("saleable"), ReceivedFrom: c.Query("receivedFrom"),
		ReceivedTo: c.Query("receivedTo"), MinAgeDays: c.Query("minAgeDays"), MaxAgeDays: c.Query("maxAgeDays"),
		Sort: c.Query("sort"), Page: c.Query("page"), Limit: c.Query("limit")})
	h.respond(c, UnitPage{Items: p.Items, Page: p.Page, Limit: p.Limit, Total: p.Total}, err)
}

// Get godoc
//
//	@Summary		Get a vehicle unit
//	@Description	The unit with its source (supplier, purchase order and line, goods receipt, line and captured unit, supplier reference, received date) its registration history (each entry names its source document), and the metadata of the files about it that the caller may open: its goods receipt's documents (attachments.view + goods_receipts.view) and its PDI evidence (attachments.view), each at a location in the caller's scope. Files open through GET /attachments/{id}/link or /content; no storage key or URL is returned. Without attachments.view documents is empty and documentsHidden true. Later status changes are its stock movements (GET /vehicle-units/{id}/stock-movements). The referenced records open through their own APIs and permissions. A unit at a location outside the caller's scope is 404. unitCost and costCurrency are null, and costHidden true, without vehicle_units.cost.view. Requires vehicle_units.view.
//	@Tags			vehicle-units
//	@Produce		json
//	@Param			id	path		string	true	"Vehicle unit ID"
//	@Success		200	{object}	httpx.DataResponse{data=vehicles.Detail}
//	@Failure		401	{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403	{object}	httpx.ErrorResponse	"FORBIDDEN"
//	@Failure		404	{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Router			/vehicle-units/{id} [get]
func (h *Handler) Get(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	d, err := h.svc.GetDetail(c.Request.Context(), a, c.Param("id"))
	h.respond(c, d, err)
}

// Movements godoc
//
//	@Summary		List a vehicle unit's stock movements
//	@Description	The unit's append-only stock movement ledger entries (E-01), read-only: receipt, PDI disposition (PDI_PASS, QUALITY_HOLD, REJECT) and display moves, with status and location before and after, the source (goods receipt with its number, PDI inspection with its number, or a staff action), the actor and the server time. Oldest first (order=asc, default) or newest first (order=desc), in ledger sequence. A unit at a location outside the caller's scope is 404; a location outside the scope is named by ID only (code and name null). The cross-unit movement timeline is E-02. Requires vehicle_units.view.
//	@Tags			vehicle-units
//	@Produce		json
//	@Param			id		path		string	true	"Vehicle unit ID"
//	@Param			order	query		string	false	"asc (default) or desc"	Enums(asc, desc)
//	@Param			page	query		int		false	"Page (from 1)"
//	@Param			limit	query		int		false	"Page size (1-100, default 50)"
//	@Success		200		{object}	httpx.DataResponse{data=vehicles.MovementPage}
//	@Failure		401		{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403		{object}	httpx.ErrorResponse	"FORBIDDEN"
//	@Failure		404		{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Failure		422		{object}	httpx.ErrorResponse	"VALIDATION_FAILED (order, page, limit)"
//	@Router			/vehicle-units/{id}/stock-movements [get]
func (h *Handler) Movements(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	p, err := h.svc.Movements(c.Request.Context(), a, c.Param("id"),
		MovementQuery{Order: c.Query("order"), Page: c.Query("page"), Limit: c.Query("limit")})
	h.respond(c, MovementPage{Items: p.Items, Page: p.Page, Limit: p.Limit, Total: p.Total}, err)
}
