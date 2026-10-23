package purchasing

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/httpx"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/modules/access"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/modules/auth"
)

// IdempotencyHeader is the optional create header for double-submit protection.
const IdempotencyHeader = "Idempotency-Key"

// Handler is the HTTP adapter for purchase orders.
type Handler struct {
	svc    *Service
	access *access.Handler
	log    *slog.Logger
}

// NewHandler builds the adapter.
func NewHandler(svc *Service, accessH *access.Handler, log *slog.Logger) *Handler {
	return &Handler{svc: svc, access: accessH, log: log}
}

// Register mounts the routes. There is no DELETE of a PO: it is cancelled and
// kept; only lines of a draft are removed.
func (h *Handler) Register(rg *gin.RouterGroup) {
	view := h.access.Require(access.PermPurchaseOrderView)
	manage := h.access.Require(access.PermPurchaseOrderManage)
	approve := h.access.Require(access.PermPurchaseOrderApprove)
	cancel := h.access.Require(access.PermPurchaseOrderCancel)

	rg.GET("/purchase-orders", view, h.List)
	rg.POST("/purchase-orders", manage, h.Create)
	rg.GET("/purchase-orders/:id", view, h.Get)
	rg.PATCH("/purchase-orders/:id", manage, h.Update)
	rg.POST("/purchase-orders/:id/lines", manage, h.AddLine)
	rg.PATCH("/purchase-orders/:id/lines/:lineId", manage, h.UpdateLine)
	rg.POST("/purchase-orders/:id/lines/:lineId/remove", manage, h.RemoveLine)
	rg.POST("/purchase-orders/:id/submit", manage, h.Submit)
	rg.POST("/purchase-orders/:id/approve", approve, h.Approve)
	rg.POST("/purchase-orders/:id/return", approve, h.Return)
	rg.POST("/purchase-orders/:id/cancel", cancel, h.Cancel)
	rg.GET("/purchase-order-settings", h.access.RequireAny(access.PermPurchaseOrderView, access.PermPurchaseOrderSettings), h.GetSettings)
	rg.PUT("/purchase-order-settings", h.access.Require(access.PermPurchaseOrderSettings), h.UpdateSettings)
}

// POPage is a page of purchase orders.
type POPage struct {
	Items []PurchaseOrder `json:"items"`
	Page  int             `json:"page" example:"1"`
	Limit int             `json:"limit" example:"25"`
	Total int             `json:"total" example:"0"`
} //@name PurchaseOrderPage

func (h *Handler) actor(c *gin.Context) Actor {
	p, _ := auth.CurrentPrincipal(c)
	return Actor{UserID: p.UserID, Username: p.Username, RequestID: c.GetString(httpx.RequestIDKey)}
}

func (h *Handler) decode(c *gin.Context, dst any) bool {
	if c.ContentType() != "application/json" {
		httpx.Error(c, http.StatusUnsupportedMediaType, httpx.CodeUnsupportedMediaType, "Send the request body as application/json.")
		return false
	}
	dec := json.NewDecoder(io.LimitReader(c.Request.Body, 64<<10))
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
		httpx.Error(c, http.StatusBadRequest, httpx.CodeBadRequest, "The request body must be one JSON object.")
	}
	return false
}

func (h *Handler) fail(c *gin.Context, err error) {
	var (
		verr    *ValidationError
		dup     *DuplicateError
		state   *StateError
		blocked *BlockedError
	)
	switch {
	case errors.Is(err, ErrForbidden):
		access.Forbidden(c)
	case errors.Is(err, ErrSelfApproval):
		httpx.Error(c, http.StatusForbidden, httpx.CodeSelfApproval,
			"You created or submitted this purchase order, so another authorised person must approve it.")
	case errors.Is(err, ErrNotFound):
		httpx.Error(c, http.StatusNotFound, httpx.CodeNotFound, "The purchase order or line was not found.")
	case errors.Is(err, ErrVersionConflict):
		httpx.Error(c, http.StatusConflict, httpx.CodeVersionConflict, "This purchase order was changed by someone else. Reload it and try again.")
	case errors.As(err, &verr):
		details := make([]httpx.FieldError, len(verr.Fields))
		for i, f := range verr.Fields {
			details[i] = httpx.FieldError{Field: f.Field, Message: f.Message}
		}
		httpx.Error(c, http.StatusUnprocessableEntity, httpx.CodeValidationFailed, "Check the highlighted fields.", details...)
	case errors.As(err, &dup):
		msg := "This variant is already on another line of this purchase order. Change that line's quantity instead."
		if dup.Field != "variantId" {
			msg = "This " + dup.Field + " is already used."
		}
		httpx.Error(c, http.StatusConflict, httpx.CodeDuplicate, msg, httpx.FieldError{Field: dup.Field, Message: "is already used"})
	case errors.As(err, &blocked):
		details := make([]httpx.FieldError, len(blocked.Unmet))
		for i, u := range blocked.Unmet {
			details[i] = httpx.FieldError{Field: u.Key, Message: u.Message}
		}
		if blocked.Approval {
			httpx.Error(c, http.StatusConflict, httpx.CodeApprovalBlocked, "This purchase order cannot be approved yet.", details...)
		} else {
			httpx.Error(c, http.StatusConflict, httpx.CodeSubmissionBlocked, "This purchase order cannot be submitted yet.", details...)
		}
	case errors.As(err, &state):
		httpx.Error(c, http.StatusConflict, httpx.CodeInvalidState, state.Message)
	default:
		h.log.ErrorContext(c.Request.Context(), "purchase order request failed",
			slog.String("request_id", c.GetString(httpx.RequestIDKey)), slog.Any("error", err))
		httpx.InternalError(c)
	}
}

func (h *Handler) respond(c *gin.Context, status int, v any, err error) {
	if err != nil {
		h.fail(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(status, httpx.DataResponse{Data: v})
}

// with decodes the body into T and runs fn with the caller's access.
func with[T any](h *Handler, c *gin.Context, status int, fn func(a access.Access, in T) (any, error)) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	var in T
	if !h.decode(c, &in) {
		return
	}
	v, err := fn(a, in)
	h.respond(c, status, v, err)
}

// List godoc
//
//	@Summary		List purchase orders
//	@Description	Newest first. Search the PO number or supplier name; filter by status, supplier or brand (a line of that brand). Totals and remaining quantities are computed by the server. Requires purchase_orders.view.
//	@Tags			purchase-orders
//	@Produce		json
//	@Param			q			query	string	false	"Search"
//	@Param			status		query	string	false	"Status"	Enums(DRAFT, SUBMITTED, APPROVED, PARTIALLY_RECEIVED, RECEIVED, CLOSED, CANCELLED)
//	@Param			supplierId	query	string	false	"Supplier ID"
//	@Param			brandId		query	string	false	"Brand ID"
//	@Param			page		query	int		false	"Page (from 1)"
//	@Param			limit		query	int		false	"Page size (1-100, default 25)"
//	@Success		200	{object}	httpx.DataResponse{data=purchasing.POPage}
//	@Failure		401	{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403	{object}	httpx.ErrorResponse	"FORBIDDEN"
//	@Failure		422	{object}	httpx.ErrorResponse	"VALIDATION_FAILED"
//	@Router			/purchase-orders [get]
func (h *Handler) List(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	p, err := h.svc.List(c.Request.Context(), a, ListQuery{Query: c.Query("q"), Status: c.Query("status"),
		SupplierID: c.Query("supplierId"), BrandID: c.Query("brandId"), Page: c.Query("page"), Limit: c.Query("limit")})
	h.respond(c, http.StatusOK, POPage{Items: p.Items, Page: p.Page, Limit: p.Limit, Total: p.Total}, err)
}

// Get godoc
//
//	@Summary		Get a purchase order
//	@Description	The PO with lines (agreed costs, received and remaining quantities, warnings), status history, unmet prerequisites for the next step and the caller's possible actions. Catalogue cost snapshots need catalogue.cost_price.view. Requires purchase_orders.view.
//	@Tags			purchase-orders
//	@Produce		json
//	@Param			id	path		string	true	"Purchase order ID"
//	@Success		200	{object}	httpx.DataResponse{data=purchasing.Detail}
//	@Failure		401	{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403	{object}	httpx.ErrorResponse	"FORBIDDEN"
//	@Failure		404	{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Router			/purchase-orders/{id} [get]
func (h *Handler) Get(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	d, err := h.svc.GetDetail(c.Request.Context(), a, c.Param("id"))
	h.respond(c, http.StatusOK, d, err)
}

// Create godoc
//
//	@Summary		Create a draft purchase order
//	@Description	Gets the next PO number. The supplier must exist and not be archived; each line's brand, model and variant must belong together and not be archived. A line's priceListId (needs catalogue.cost_price.view) snapshots that list's approved COST price valid now. With Idempotency-Key, a repeat by the same user returns the first PO with 200. Requires purchase_orders.manage and X-CSRF-Token. Audited.
//	@Tags			purchase-orders
//	@Accept			json
//	@Produce		json
//	@Param			Idempotency-Key	header		string					false	"8-100 letters, digits, - or _"
//	@Param			body			body		purchasing.CreateInput	true	"Purchase order"
//	@Success		201				{object}	httpx.DataResponse{data=purchasing.Detail}
//	@Success		200				{object}	httpx.DataResponse{data=purchasing.Detail}	"Replay of an earlier create with the same Idempotency-Key"
//	@Failure		401				{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403				{object}	httpx.ErrorResponse	"FORBIDDEN or CSRF_FAILED"
//	@Failure		422				{object}	httpx.ErrorResponse	"VALIDATION_FAILED"
//	@Router			/purchase-orders [post]
func (h *Handler) Create(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	var in CreateInput
	if !h.decode(c, &in) {
		return
	}
	d, replayed, err := h.svc.Create(c.Request.Context(), h.actor(c), a, in, c.GetHeader(IdempotencyHeader))
	status := http.StatusCreated
	if replayed {
		status = http.StatusOK
		c.Header("Idempotent-Replayed", "true")
	}
	h.respond(c, status, d, err)
}

// Update godoc
//
//	@Summary		Edit a draft purchase order
//	@Description	Supplier, expected date (YYYY-MM-DD) or notes; null clears the date or notes. Drafts only. Requires purchase_orders.manage and X-CSRF-Token. Audited.
//	@Tags			purchase-orders
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string					true	"Purchase order ID"
//	@Param			body	body		purchasing.HeaderPatch	true	"Changes"
//	@Success		200		{object}	httpx.DataResponse{data=purchasing.Detail}
//	@Failure		401		{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403		{object}	httpx.ErrorResponse	"FORBIDDEN or CSRF_FAILED"
//	@Failure		404		{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Failure		409		{object}	httpx.ErrorResponse	"VERSION_CONFLICT or INVALID_STATE"
//	@Failure		422		{object}	httpx.ErrorResponse	"VALIDATION_FAILED"
//	@Router			/purchase-orders/{id} [patch]
func (h *Handler) Update(c *gin.Context) {
	with(h, c, http.StatusOK, func(a access.Access, in HeaderPatch) (any, error) {
		return h.svc.UpdateHeader(c.Request.Context(), h.actor(c), a, c.Param("id"), in)
	})
}

// AddLine godoc
//
//	@Summary		Add a line to a draft
//	@Description	One line per variant. Brand, model and variant must belong together and not be archived. priceListId (needs catalogue.cost_price.view) snapshots the approved COST price valid now. version is the PO's. Requires purchase_orders.manage and X-CSRF-Token. Audited.
//	@Tags			purchase-orders
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string					true	"Purchase order ID"
//	@Param			body	body		purchasing.LineAddInput	true	"Line"
//	@Success		201		{object}	httpx.DataResponse{data=purchasing.Detail}
//	@Failure		401		{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403		{object}	httpx.ErrorResponse	"FORBIDDEN or CSRF_FAILED"
//	@Failure		404		{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Failure		409		{object}	httpx.ErrorResponse	"DUPLICATE (variantId), VERSION_CONFLICT or INVALID_STATE"
//	@Failure		422		{object}	httpx.ErrorResponse	"VALIDATION_FAILED"
//	@Router			/purchase-orders/{id}/lines [post]
func (h *Handler) AddLine(c *gin.Context) {
	with(h, c, http.StatusCreated, func(a access.Access, in LineAddInput) (any, error) {
		return h.svc.AddLine(c.Request.Context(), h.actor(c), a, c.Param("id"), in)
	})
}

// UpdateLine godoc
//
//	@Summary		Edit a draft line
//	@Description	Quantity, unit cost or note. Sending priceListId takes a fresh cost snapshot; null removes it; leaving it out keeps the snapshot as it was. version is the PO's. Requires purchase_orders.manage and X-CSRF-Token. Audited.
//	@Tags			purchase-orders
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string					true	"Purchase order ID"
//	@Param			lineId	path		string					true	"Line ID"
//	@Param			body	body		purchasing.LinePatch	true	"Changes"
//	@Success		200		{object}	httpx.DataResponse{data=purchasing.Detail}
//	@Failure		401		{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403		{object}	httpx.ErrorResponse	"FORBIDDEN or CSRF_FAILED"
//	@Failure		404		{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Failure		409		{object}	httpx.ErrorResponse	"VERSION_CONFLICT or INVALID_STATE"
//	@Failure		422		{object}	httpx.ErrorResponse	"VALIDATION_FAILED"
//	@Router			/purchase-orders/{id}/lines/{lineId} [patch]
func (h *Handler) UpdateLine(c *gin.Context) {
	with(h, c, http.StatusOK, func(a access.Access, in LinePatch) (any, error) {
		return h.svc.UpdateLine(c.Request.Context(), h.actor(c), a, c.Param("id"), c.Param("lineId"), in)
	})
}

// RemoveLine godoc
//
//	@Summary		Remove a draft line
//	@Description	Only lines of a draft are removed; submitted POs keep their lines. version is the PO's. Requires purchase_orders.manage and X-CSRF-Token. Audited.
//	@Tags			purchase-orders
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string					true	"Purchase order ID"
//	@Param			lineId	path		string					true	"Line ID"
//	@Param			body	body		purchasing.VersionInput	true	"Version"
//	@Success		200		{object}	httpx.DataResponse{data=purchasing.Detail}
//	@Failure		401		{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403		{object}	httpx.ErrorResponse	"FORBIDDEN or CSRF_FAILED"
//	@Failure		404		{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Failure		409		{object}	httpx.ErrorResponse	"VERSION_CONFLICT or INVALID_STATE"
//	@Failure		422		{object}	httpx.ErrorResponse	"VALIDATION_FAILED"
//	@Router			/purchase-orders/{id}/lines/{lineId}/remove [post]
func (h *Handler) RemoveLine(c *gin.Context) {
	with(h, c, http.StatusOK, func(a access.Access, in VersionInput) (any, error) {
		return h.svc.RemoveLine(c.Request.Context(), h.actor(c), a, c.Param("id"), c.Param("lineId"), in)
	})
}

// Submit godoc
//
//	@Summary		Submit a draft
//	@Description	Needs at least one line, an expected date not in the past, a VERIFIED supplier with a current verified relationship to every line's brand, and each brand (active and authorised), model and variant active. Without a signed-off threshold every PO then waits for approval (SUBMITTED); within a signed-off threshold it is APPROVED with basis BELOW_THRESHOLD. Requires purchase_orders.manage and X-CSRF-Token. Audited.
//	@Tags			purchase-orders
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string					true	"Purchase order ID"
//	@Param			body	body		purchasing.VersionInput	true	"Version"
//	@Success		200		{object}	httpx.DataResponse{data=purchasing.Detail}
//	@Failure		401		{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403		{object}	httpx.ErrorResponse	"FORBIDDEN or CSRF_FAILED"
//	@Failure		404		{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Failure		409		{object}	httpx.ErrorResponse	"SUBMISSION_BLOCKED (details: unmet keys), VERSION_CONFLICT or INVALID_STATE"
//	@Failure		422		{object}	httpx.ErrorResponse	"VALIDATION_FAILED"
//	@Router			/purchase-orders/{id}/submit [post]
func (h *Handler) Submit(c *gin.Context) {
	with(h, c, http.StatusOK, func(a access.Access, in VersionInput) (any, error) {
		return h.svc.Submit(c.Request.Context(), h.actor(c), a, c.Param("id"), in)
	})
}

// Approve godoc
//
//	@Summary		Approve a submitted purchase order
//	@Description	Never by the PO's creator or submitter (403 SELF_APPROVAL_NOT_ALLOWED). The submission prerequisites are checked again. Refusals are audited as DENIED. Requires purchase_orders.approve and X-CSRF-Token. Audited.
//	@Tags			purchase-orders
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string					true	"Purchase order ID"
//	@Param			body	body		purchasing.ApproveInput	true	"Approval"
//	@Success		200		{object}	httpx.DataResponse{data=purchasing.Detail}
//	@Failure		401		{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403		{object}	httpx.ErrorResponse	"FORBIDDEN, SELF_APPROVAL_NOT_ALLOWED or CSRF_FAILED"
//	@Failure		404		{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Failure		409		{object}	httpx.ErrorResponse	"APPROVAL_BLOCKED, VERSION_CONFLICT or INVALID_STATE"
//	@Failure		422		{object}	httpx.ErrorResponse	"VALIDATION_FAILED"
//	@Router			/purchase-orders/{id}/approve [post]
func (h *Handler) Approve(c *gin.Context) {
	with(h, c, http.StatusOK, func(a access.Access, in ApproveInput) (any, error) {
		return h.svc.Approve(c.Request.Context(), h.actor(c), a, c.Param("id"), in)
	})
}

// Return godoc
//
//	@Summary		Return a submitted purchase order to draft
//	@Description	With a reason, recorded in the status history. Requires purchase_orders.approve and X-CSRF-Token. Audited.
//	@Tags			purchase-orders
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string					true	"Purchase order ID"
//	@Param			body	body		purchasing.ReasonInput	true	"Reason"
//	@Success		200		{object}	httpx.DataResponse{data=purchasing.Detail}
//	@Failure		401		{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403		{object}	httpx.ErrorResponse	"FORBIDDEN or CSRF_FAILED"
//	@Failure		404		{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Failure		409		{object}	httpx.ErrorResponse	"VERSION_CONFLICT or INVALID_STATE"
//	@Failure		422		{object}	httpx.ErrorResponse	"VALIDATION_FAILED"
//	@Router			/purchase-orders/{id}/return [post]
func (h *Handler) Return(c *gin.Context) {
	with(h, c, http.StatusOK, func(a access.Access, in ReasonInput) (any, error) {
		return h.svc.Return(c.Request.Context(), h.actor(c), a, c.Param("id"), in)
	})
}

// Cancel godoc
//
//	@Summary		Cancel a purchase order
//	@Description	Drafts, submitted POs and approved POs with nothing received. The PO stays as history. Requires purchase_orders.cancel and X-CSRF-Token. Audited.
//	@Tags			purchase-orders
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string					true	"Purchase order ID"
//	@Param			body	body		purchasing.ReasonInput	true	"Reason"
//	@Success		200		{object}	httpx.DataResponse{data=purchasing.Detail}
//	@Failure		401		{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403		{object}	httpx.ErrorResponse	"FORBIDDEN or CSRF_FAILED"
//	@Failure		404		{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Failure		409		{object}	httpx.ErrorResponse	"VERSION_CONFLICT or INVALID_STATE"
//	@Failure		422		{object}	httpx.ErrorResponse	"VALIDATION_FAILED"
//	@Router			/purchase-orders/{id}/cancel [post]
func (h *Handler) Cancel(c *gin.Context) {
	with(h, c, http.StatusOK, func(a access.Access, in ReasonInput) (any, error) {
		return h.svc.Cancel(c.Request.Context(), h.actor(c), a, c.Param("id"), in)
	})
}

// GetSettings godoc
//
//	@Summary		Get the purchase order approval settings
//	@Description	approvalThreshold null means none is configured and every PO needs an approver. Requires purchase_orders.view or purchase_orders.settings.manage.
//	@Tags			purchase-orders
//	@Produce		json
//	@Success		200	{object}	httpx.DataResponse{data=purchasing.Settings}
//	@Failure		401	{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403	{object}	httpx.ErrorResponse	"FORBIDDEN"
//	@Router			/purchase-order-settings [get]
func (h *Handler) GetSettings(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	st, err := h.svc.GetSettings(c.Request.Context(), a)
	h.respond(c, http.StatusOK, st, err)
}

// UpdateSettings godoc
//
//	@Summary		Change the purchase order approval settings
//	@Description	A threshold needs the owner's sign-off reference. POs already submitted keep the threshold they were submitted under. Requires purchase_orders.settings.manage and X-CSRF-Token. Audited.
//	@Tags			purchase-orders
//	@Accept			json
//	@Produce		json
//	@Param			body	body		purchasing.SettingsInput	true	"Settings"
//	@Success		200		{object}	httpx.DataResponse{data=purchasing.Settings}
//	@Failure		401		{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403		{object}	httpx.ErrorResponse	"FORBIDDEN or CSRF_FAILED"
//	@Failure		409		{object}	httpx.ErrorResponse	"VERSION_CONFLICT"
//	@Failure		422		{object}	httpx.ErrorResponse	"VALIDATION_FAILED"
//	@Router			/purchase-order-settings [put]
func (h *Handler) UpdateSettings(c *gin.Context) {
	with(h, c, http.StatusOK, func(a access.Access, in SettingsInput) (any, error) {
		return h.svc.UpdateSettings(c.Request.Context(), h.actor(c), a, in)
	})
}
