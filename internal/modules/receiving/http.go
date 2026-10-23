package receiving

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

// maxBody is the largest request body: a save carries every unit's identifiers.
const maxBody = 1 << 20

// Handler is the HTTP adapter for goods receipts.
type Handler struct {
	svc    *Service
	access *access.Handler
	log    *slog.Logger
}

// NewHandler builds the adapter.
func NewHandler(svc *Service, accessH *access.Handler, log *slog.Logger) *Handler {
	return &Handler{svc: svc, access: accessH, log: log}
}

// Register mounts the routes. There is no DELETE: a draft is cancelled and kept.
// Posting a receipt (stock changes) is a separate task (D-07).
func (h *Handler) Register(rg *gin.RouterGroup) {
	view := h.access.Require(access.PermGoodsReceiptView)
	manage := h.access.Require(access.PermGoodsReceiptManage)

	rg.GET("/goods-receipts", view, h.List)
	rg.POST("/goods-receipts", manage, h.Create)
	rg.GET("/goods-receipts/:id", view, h.Get)
	rg.PUT("/goods-receipts/:id", manage, h.Save)
	rg.POST("/goods-receipts/:id/cancel", manage, h.Cancel)
	rg.POST("/goods-receipts/:id/documents", manage, h.AddDocument)
	rg.GET("/receivable-purchase-orders", manage, h.ListReceivable)
}

// ReceiptPage is a page of goods receipts.
type ReceiptPage struct {
	Items []Receipt `json:"items"`
	Page  int       `json:"page" example:"1"`
	Limit int       `json:"limit" example:"25"`
	Total int       `json:"total" example:"0"`
} //@name GoodsReceiptPage

// ReceivablePage is a page of purchase orders open for receipts.
type ReceivablePage struct {
	Items []ReceivablePO `json:"items"`
	Page  int            `json:"page" example:"1"`
	Limit int            `json:"limit" example:"25"`
	Total int            `json:"total" example:"0"`
} //@name GoodsReceiptPurchaseOrderPage

func (h *Handler) actor(c *gin.Context) Actor {
	p, _ := auth.CurrentPrincipal(c)
	return Actor{UserID: p.UserID, Username: p.Username, RequestID: c.GetString(httpx.RequestIDKey)}
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
		httpx.Error(c, http.StatusBadRequest, httpx.CodeBadRequest, "The request body must be one JSON object (at most 1 MB).")
	}
	return false
}

func fieldErrors(fields []FieldProblem) []httpx.FieldError {
	out := make([]httpx.FieldError, len(fields))
	for i, f := range fields {
		out[i] = httpx.FieldError{Field: f.Field, Message: f.Message}
	}
	return out
}

func (h *Handler) fail(c *gin.Context, err error) {
	var (
		verr      *ValidationError
		dup       *DuplicateError
		state     *StateError
		over      *OverReceiptError
		forbidden *OverReceiptForbiddenError
	)
	switch {
	case errors.Is(err, ErrForbidden):
		access.Forbidden(c)
	case errors.Is(err, ErrNotFound):
		httpx.Error(c, http.StatusNotFound, httpx.CodeNotFound, "The goods receipt was not found.")
	case errors.Is(err, ErrVersionConflict):
		httpx.Error(c, http.StatusConflict, httpx.CodeVersionConflict, "This goods receipt was changed by someone else. Reload it and try again.")
	case errors.As(err, &verr):
		httpx.Error(c, http.StatusUnprocessableEntity, httpx.CodeValidationFailed, "Check the highlighted fields.", fieldErrors(verr.Fields)...)
	case errors.As(err, &dup):
		httpx.Error(c, http.StatusConflict, httpx.CodeDuplicate, "Some identifiers or values are already used. Nothing was saved.", fieldErrors(dup.Fields)...)
	case errors.As(err, &over):
		httpx.Error(c, http.StatusConflict, httpx.CodeOverReceiptNeedsOverride,
			"More is received than the purchase order has left. An authorised person must approve the extra units with a reason. Nothing was saved.",
			fieldErrors(over.Fields)...)
	case errors.As(err, &forbidden):
		httpx.Error(c, http.StatusForbidden, httpx.CodeOverReceiptForbidden,
			"You may not authorise over-receipt. Nothing was saved.", fieldErrors(forbidden.Fields)...)
	case errors.As(err, &state):
		httpx.Error(c, http.StatusConflict, httpx.CodeInvalidState, state.Message)
	default:
		h.log.ErrorContext(c.Request.Context(), "goods receipt request failed",
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
//	@Summary		List goods receipts
//	@Description	Newest first. Search the GRN or PO number, supplier name or supplier reference; filter by status or purchase order. Requires goods_receipts.view.
//	@Tags			goods-receipts
//	@Produce		json
//	@Param			q				query	string	false	"Search"
//	@Param			status			query	string	false	"Status"	Enums(DRAFT, POSTED, CANCELLED)
//	@Param			purchaseOrderId	query	string	false	"Purchase order ID"
//	@Param			page			query	int		false	"Page (from 1)"
//	@Param			limit			query	int		false	"Page size (1-100, default 25)"
//	@Success		200	{object}	httpx.DataResponse{data=receiving.ReceiptPage}
//	@Failure		401	{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403	{object}	httpx.ErrorResponse	"FORBIDDEN"
//	@Failure		422	{object}	httpx.ErrorResponse	"VALIDATION_FAILED"
//	@Router			/goods-receipts [get]
func (h *Handler) List(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	p, err := h.svc.List(c.Request.Context(), a, ListQuery{Query: c.Query("q"), Status: c.Query("status"),
		PurchaseOrderID: c.Query("purchaseOrderId"), Page: c.Query("page"), Limit: c.Query("limit")})
	h.respond(c, http.StatusOK, ReceiptPage{Items: p.Items, Page: p.Page, Limit: p.Limit, Total: p.Total}, err)
}

// ListReceivable godoc
//
//	@Summary		List purchase orders open for receipts
//	@Description	APPROVED and PARTIALLY_RECEIVED purchase orders with ordered, posted, draft-held and available quantities. Search the PO number or supplier name. Requires goods_receipts.manage.
//	@Tags			goods-receipts
//	@Produce		json
//	@Param			q		query	string	false	"Search"
//	@Param			page	query	int		false	"Page (from 1)"
//	@Param			limit	query	int		false	"Page size (1-100, default 25)"
//	@Success		200	{object}	httpx.DataResponse{data=receiving.ReceivablePage}
//	@Failure		401	{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403	{object}	httpx.ErrorResponse	"FORBIDDEN"
//	@Failure		422	{object}	httpx.ErrorResponse	"VALIDATION_FAILED"
//	@Router			/receivable-purchase-orders [get]
func (h *Handler) ListReceivable(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	p, err := h.svc.ListReceivable(c.Request.Context(), a, c.Query("q"), c.Query("page"), c.Query("limit"))
	h.respond(c, http.StatusOK, ReceivablePage{Items: p.Items, Page: p.Page, Limit: p.Limit, Total: p.Total}, err)
}

// Get godoc
//
//	@Summary		Get a goods receipt
//	@Description	The receipt with every line of its purchase order (ordered, posted, held by other drafts, available, received, accepted, rejected, over-receipt authorisation, units with identifiers, warnings), documents, history and the caller's actions. stockPosted is false for drafts: they change no stock. Requires goods_receipts.view.
//	@Tags			goods-receipts
//	@Produce		json
//	@Param			id	path		string	true	"Goods receipt ID"
//	@Success		200	{object}	httpx.DataResponse{data=receiving.Detail}
//	@Failure		401	{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403	{object}	httpx.ErrorResponse	"FORBIDDEN"
//	@Failure		404	{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Router			/goods-receipts/{id} [get]
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
//	@Summary		Start a draft goods receipt
//	@Description	Gets the next GRN number. The purchase order must be APPROVED or PARTIALLY_RECEIVED; the received date must not be in the future. Lines are entered with PUT. With Idempotency-Key, a repeat by the same user returns the first receipt with 200. Changes no stock. Requires goods_receipts.manage and X-CSRF-Token. Audited.
//	@Tags			goods-receipts
//	@Accept			json
//	@Produce		json
//	@Param			Idempotency-Key	header		string					false	"8-100 letters, digits, - or _"
//	@Param			body			body		receiving.CreateInput	true	"Receipt"
//	@Success		201				{object}	httpx.DataResponse{data=receiving.Detail}
//	@Success		200				{object}	httpx.DataResponse{data=receiving.Detail}	"Replay of an earlier create with the same Idempotency-Key"
//	@Failure		401				{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403				{object}	httpx.ErrorResponse	"FORBIDDEN or CSRF_FAILED"
//	@Failure		422				{object}	httpx.ErrorResponse	"VALIDATION_FAILED"
//	@Router			/goods-receipts [post]
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

// Save godoc
//
//	@Summary		Save a draft goods receipt
//	@Description	Replaces the draft's received date, supplier reference, notes, lines and units with what is sent (all or nothing). Per line: received = accepted + rejected; units at most the received quantity (accepted/rejected units at most those quantities); identifiers optional, a chassis/frame number once per receipt and not on another open or posted receipt (409 DUPLICATE, details per unit). A line above what the PO line has left (ordered - posted - other drafts) needs overReceiptReason from a holder of goods_receipts.over_receipt (else 409 OVER_RECEIPT_REQUIRES_OVERRIDE, or 403 OVER_RECEIPT_NOT_AUTHORISED when the reason comes from someone without it, audited DENIED). Changes no stock. Requires goods_receipts.manage and X-CSRF-Token. Audited.
//	@Tags			goods-receipts
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string				true	"Goods receipt ID"
//	@Param			body	body		receiving.SaveInput	true	"Draft"
//	@Success		200		{object}	httpx.DataResponse{data=receiving.Detail}
//	@Failure		401		{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403		{object}	httpx.ErrorResponse	"FORBIDDEN, OVER_RECEIPT_NOT_AUTHORISED or CSRF_FAILED"
//	@Failure		404		{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Failure		409		{object}	httpx.ErrorResponse	"OVER_RECEIPT_REQUIRES_OVERRIDE, DUPLICATE, VERSION_CONFLICT or INVALID_STATE"
//	@Failure		422		{object}	httpx.ErrorResponse	"VALIDATION_FAILED"
//	@Router			/goods-receipts/{id} [put]
func (h *Handler) Save(c *gin.Context) {
	with(h, c, http.StatusOK, func(a access.Access, in SaveInput) (any, error) {
		return h.svc.Save(c.Request.Context(), h.actor(c), a, c.Param("id"), in)
	})
}

// Cancel godoc
//
//	@Summary		Cancel a draft goods receipt
//	@Description	With a reason. The receipt, its lines and units stay as history and no longer hold purchase order quantity. Posted receipts are never cancelled. Requires goods_receipts.manage and X-CSRF-Token. Audited.
//	@Tags			goods-receipts
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string					true	"Goods receipt ID"
//	@Param			body	body		receiving.CancelInput	true	"Reason"
//	@Success		200		{object}	httpx.DataResponse{data=receiving.Detail}
//	@Failure		401		{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403		{object}	httpx.ErrorResponse	"FORBIDDEN or CSRF_FAILED"
//	@Failure		404		{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Failure		409		{object}	httpx.ErrorResponse	"VERSION_CONFLICT or INVALID_STATE"
//	@Failure		422		{object}	httpx.ErrorResponse	"VALIDATION_FAILED"
//	@Router			/goods-receipts/{id}/cancel [post]
func (h *Handler) Cancel(c *gin.Context) {
	with(h, c, http.StatusOK, func(a access.Access, in CancelInput) (any, error) {
		return h.svc.Cancel(c.Request.Context(), h.actor(c), a, c.Param("id"), in)
	})
}

// AddDocument godoc
//
//	@Summary		Attach a document to a draft goods receipt
//	@Description	Links one of the caller's own uploads (POST /attachments) that is not attached to anything yet; it then belongs to the receipt for good. Downloads need attachments.view and goods_receipts.view. Drafts only. Requires goods_receipts.manage, attachments.upload and X-CSRF-Token. Audited.
//	@Tags			goods-receipts
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string					true	"Goods receipt ID"
//	@Param			body	body		receiving.DocumentInput	true	"Document"
//	@Success		201		{object}	httpx.DataResponse{data=receiving.Detail}
//	@Failure		401		{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403		{object}	httpx.ErrorResponse	"FORBIDDEN or CSRF_FAILED"
//	@Failure		404		{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Failure		409		{object}	httpx.ErrorResponse	"DUPLICATE or INVALID_STATE"
//	@Failure		422		{object}	httpx.ErrorResponse	"VALIDATION_FAILED"
//	@Router			/goods-receipts/{id}/documents [post]
func (h *Handler) AddDocument(c *gin.Context) {
	with(h, c, http.StatusCreated, func(a access.Access, in DocumentInput) (any, error) {
		return h.svc.AddDocument(c.Request.Context(), h.actor(c), a, c.Param("id"), in)
	})
}
