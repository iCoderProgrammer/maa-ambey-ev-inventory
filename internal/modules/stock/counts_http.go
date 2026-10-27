package stock

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/httpx"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/access"
)

// Stock count routes (E-05). No route deletes a count or line, edits a decided
// count, or sets an expected quantity, variance, balance or status directly.
func (h *Handler) registerCounts(rg *gin.RouterGroup) {
	view := h.access.Require(access.PermAdjustmentView)
	create := h.access.Require(access.PermAdjustmentCreate)
	approve := h.access.Require(access.PermAdjustmentApprove)
	rg.GET("/stock-counts", view, h.ListCounts)
	rg.POST("/stock-counts", create, h.CreateCount)
	rg.GET("/stock-counts/:id", view, h.GetCount)
	rg.POST("/stock-counts/:id/lines", create, h.RecordCountLine)
	rg.POST("/stock-counts/:id/submit", create, h.SubmitCount)
	rg.POST("/stock-counts/:id/approve", approve, h.ApproveCount)
	rg.POST("/stock-counts/:id/reject", approve, h.RejectCount)
}

// failCount maps count errors; everything else goes to fail.
func (h *Handler) failCount(c *gin.Context, err error) {
	var state *CountStateError
	var changed *StockChangedError
	switch {
	case errors.Is(err, ErrSelfDecision):
		httpx.Error(c, http.StatusForbidden, httpx.CodeSelfApproval,
			"You created, counted or submitted this stock count, so someone else must decide it. Nothing was changed.")
	case errors.Is(err, ErrCountVersion):
		httpx.Error(c, http.StatusConflict, httpx.CodeVersionConflict,
			"This stock count was changed by someone else. Reload it and try again. Nothing was changed.")
	case errors.As(err, &state):
		httpx.Error(c, http.StatusConflict, httpx.CodeInvalidState, state.Message+" Nothing was changed.")
	case errors.As(err, &changed):
		out := make([]httpx.FieldError, len(changed.LineIDs))
		for i, id := range changed.LineIDs {
			out[i] = httpx.FieldError{Field: "lines." + id, Message: "stock changed after this line was counted"}
		}
		httpx.Error(c, http.StatusConflict, httpx.CodeStockChanged,
			"Stock moved after some lines were counted, so the count no longer matches the record. Recount those lines (draft) or reject the count. Nothing was changed.",
			out...)
	default:
		h.fail(c, err)
	}
}

func (h *Handler) respondCount(c *gin.Context, status int, res any, replayed bool, err error) {
	if err != nil {
		h.failCount(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	if replayed {
		c.Header("Idempotent-Replayed", "true")
		status = http.StatusOK
	}
	c.JSON(status, httpx.DataResponse{Data: res})
}

// ListCounts godoc
//
//	@Summary		List stock counts
//	@Description	Counts (adjustment requests) at the locations in the caller's scope, newest first, without lines. status=SUBMITTED is the approval queue. Filters: status, locationId (must be in scope). Requires stock.adjustment.view.
//	@Tags			stock-counts
//	@Produce		json
//	@Param			status		query	string	false	"Status"	Enums(DRAFT, SUBMITTED, APPROVED, REJECTED)
//	@Param			locationId	query	string	false	"Location ID (in the caller's scope)"
//	@Param			page		query	int		false	"Page (from 1)"
//	@Param			limit		query	int		false	"Page size (1-100, default 25)"
//	@Success		200	{object}	httpx.DataResponse{data=stock.CountPage}
//	@Failure		401	{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403	{object}	httpx.ErrorResponse	"FORBIDDEN"
//	@Failure		422	{object}	httpx.ErrorResponse	"VALIDATION_FAILED (status, locationId, page, limit)"
//	@Router			/stock-counts [get]
func (h *Handler) ListCounts(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	p, err := h.svc.ListCounts(c.Request.Context(), a, CountListQuery{Status: c.Query("status"), LocationID: c.Query("locationId"),
		Page: c.Query("page"), Limit: c.Query("limit")})
	h.respondCount(c, http.StatusOK, p, false, err)
}

// GetCount godoc
//
//	@Summary		Get a stock count
//	@Description	The count with its lines: expected quantity/status (read from the ledger when the line was recorded), observed, server-computed variance, reason, who recorded it, stockChanged (the balance or unit moved since; undecided counts only), outcome and the posted movement. A count at a location outside the caller's scope is 404. Requires stock.adjustment.view.
//	@Tags			stock-counts
//	@Produce		json
//	@Param			id	path		string	true	"Stock count ID"
//	@Success		200	{object}	httpx.DataResponse{data=stock.StockCount}
//	@Failure		401	{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403	{object}	httpx.ErrorResponse	"FORBIDDEN"
//	@Failure		404	{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Router			/stock-counts/{id} [get]
func (h *Handler) GetCount(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	res, err := h.svc.GetCount(c.Request.Context(), a, c.Param("id"))
	h.respondCount(c, http.StatusOK, res, false, err)
}

// CreateCount godoc
//
//	@Summary		Start a stock count
//	@Description	Creates a DRAFT count (number SC-000001…) at an active location in the caller's scope. Changes no stock. Idempotency-Key is required: the same key and body by the same user returns the first count with 200 and Idempotent-Replayed: true; the same key with another body is 409 IDEMPOTENCY_KEY_REUSED. Requires stock.adjustment.create and X-CSRF-Token. Audited (stock_count.created).
//	@Tags			stock-counts
//	@Accept			json
//	@Produce		json
//	@Param			Idempotency-Key	header		string						true	"8-100 letters, digits, - or _"
//	@Param			body			body		stock.CountCreateInput		true	"Location and optional note"
//	@Success		201				{object}	httpx.DataResponse{data=stock.StockCount}	"Created"
//	@Success		200				{object}	httpx.DataResponse{data=stock.StockCount}	"Replay (Idempotent-Replayed: true)"
//	@Failure		400				{object}	httpx.ErrorResponse	"BAD_REQUEST"
//	@Failure		401				{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403				{object}	httpx.ErrorResponse	"FORBIDDEN or CSRF_FAILED"
//	@Failure		409				{object}	httpx.ErrorResponse	"IDEMPOTENCY_KEY_REUSED"
//	@Failure		415				{object}	httpx.ErrorResponse	"UNSUPPORTED_MEDIA_TYPE"
//	@Failure		422				{object}	httpx.ErrorResponse	"VALIDATION_FAILED (locationId, note, Idempotency-Key)"
//	@Router			/stock-counts [post]
func (h *Handler) CreateCount(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	var in CountCreateInput
	if !h.decode(c, &in) {
		return
	}
	res, replayed, err := h.svc.CreateCount(c.Request.Context(), a, h.actor(c), c.GetString(httpx.RequestIDKey), in,
		c.GetHeader(IdempotencyHeader))
	h.respondCount(c, http.StatusCreated, res, replayed, err)
}

// RecordCountLine godoc
//
//	@Summary		Record a counted item
//	@Description	Records (or re-records) one line on a DRAFT count: stockItemId + observedQuantity (>= 0, up to 3 decimals) + uom (the SKU's unit; ACTIVE, QUANTITY-tracked SKUs only — SERIAL is 409 TRACKING_NOT_SUPPORTED), or vehicleUnitId (a unit recorded at the count's location) + found. The server reads the expected quantity/status from the ledger now and computes the variance; reason is required when it is not zero. Changes no stock. version must be the count's current version. Requires stock.adjustment.create and X-CSRF-Token. Audited (stock_count.line_recorded).
//	@Tags			stock-counts
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string					true	"Stock count ID"
//	@Param			body	body		stock.CountLineInput	true	"Counted item"
//	@Success		200		{object}	httpx.DataResponse{data=stock.StockCount}
//	@Failure		400		{object}	httpx.ErrorResponse	"BAD_REQUEST"
//	@Failure		401		{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403		{object}	httpx.ErrorResponse	"FORBIDDEN or CSRF_FAILED"
//	@Failure		404		{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Failure		409		{object}	httpx.ErrorResponse	"VERSION_CONFLICT, INVALID_STATE (not a draft; archived SKU) or TRACKING_NOT_SUPPORTED"
//	@Failure		415		{object}	httpx.ErrorResponse	"UNSUPPORTED_MEDIA_TYPE"
//	@Failure		422		{object}	httpx.ErrorResponse	"VALIDATION_FAILED (version, stockItemId, vehicleUnitId, observedQuantity, uom, found, reason)"
//	@Router			/stock-counts/{id}/lines [post]
func (h *Handler) RecordCountLine(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	var in CountLineInput
	if !h.decode(c, &in) {
		return
	}
	res, err := h.svc.RecordLine(c.Request.Context(), a, h.actor(c), c.GetString(httpx.RequestIDKey), c.Param("id"), in)
	h.respondCount(c, http.StatusOK, res, false, err)
}

// SubmitCount godoc
//
//	@Summary		Submit a stock count for approval
//	@Description	DRAFT -> SUBMITTED. Needs at least one line, and every line must still match the ledger (otherwise 409 STOCK_CHANGED_SINCE_COUNT naming the lines: re-record them). Changes no stock. Every submitted count needs an approver who took no part in it (no approval threshold is configured). Requires stock.adjustment.create and X-CSRF-Token. Audited (stock_count.submitted).
//	@Tags			stock-counts
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string						true	"Stock count ID"
//	@Param			body	body		stock.CountVersionInput		true	"Current version"
//	@Success		200		{object}	httpx.DataResponse{data=stock.StockCount}
//	@Failure		400		{object}	httpx.ErrorResponse	"BAD_REQUEST"
//	@Failure		401		{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403		{object}	httpx.ErrorResponse	"FORBIDDEN or CSRF_FAILED"
//	@Failure		404		{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Failure		409		{object}	httpx.ErrorResponse	"VERSION_CONFLICT, INVALID_STATE (not a draft, no lines) or STOCK_CHANGED_SINCE_COUNT"
//	@Failure		415		{object}	httpx.ErrorResponse	"UNSUPPORTED_MEDIA_TYPE"
//	@Failure		422		{object}	httpx.ErrorResponse	"VALIDATION_FAILED (version)"
//	@Router			/stock-counts/{id}/submit [post]
func (h *Handler) SubmitCount(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	var in CountVersionInput
	if !h.decode(c, &in) {
		return
	}
	res, err := h.svc.SubmitCount(c.Request.Context(), a, h.actor(c), c.GetString(httpx.RequestIDKey), c.Param("id"), in)
	h.respondCount(c, http.StatusOK, res, false, err)
}

// ApproveCount godoc
//
//	@Summary		Approve a stock count and post its adjustments
//	@Description	SUBMITTED -> APPROVED in one transaction: every line is locked and re-checked against the ledger (409 STOCK_CHANGED_SINCE_COUNT if stock moved: nothing is posted; reject it and recount), each parts line with a variance posts one COUNT_ADJUSTMENT movement from the expected to the observed quantity (the balance follows; audit stock_item.adjusted), vehicle discrepancies are recorded only (no movement: no approved correction exists), and the decision is audited (stock_count.approved). Never by anyone who created, counted or submitted it (403 SELF_APPROVAL_NOT_ALLOWED, audited DENIED). Idempotency-Key is required: the same approver, key and body returns the approved count with 200 and Idempotent-Replayed: true and posts nothing again; any other decided count is 409 INVALID_STATE. Requires stock.adjustment.approve and X-CSRF-Token.
//	@Tags			stock-counts
//	@Accept			json
//	@Produce		json
//	@Param			id				path		string						true	"Stock count ID"
//	@Param			Idempotency-Key	header		string						true	"8-100 letters, digits, - or _"
//	@Param			body			body		stock.CountApproveInput		true	"Current version and an optional note"
//	@Success		200				{object}	httpx.DataResponse{data=stock.StockCount}	"Approved, or a replay (Idempotent-Replayed: true)"
//	@Failure		400				{object}	httpx.ErrorResponse	"BAD_REQUEST"
//	@Failure		401				{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403				{object}	httpx.ErrorResponse	"FORBIDDEN, SELF_APPROVAL_NOT_ALLOWED or CSRF_FAILED"
//	@Failure		404				{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Failure		409				{object}	httpx.ErrorResponse	"VERSION_CONFLICT, INVALID_STATE (not submitted; archived SKU), STOCK_CHANGED_SINCE_COUNT or IDEMPOTENCY_KEY_REUSED"
//	@Failure		415				{object}	httpx.ErrorResponse	"UNSUPPORTED_MEDIA_TYPE"
//	@Failure		422				{object}	httpx.ErrorResponse	"VALIDATION_FAILED (version, note, Idempotency-Key)"
//	@Router			/stock-counts/{id}/approve [post]
func (h *Handler) ApproveCount(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	var in CountApproveInput
	if !h.decode(c, &in) {
		return
	}
	res, replayed, err := h.svc.ApproveCount(c.Request.Context(), a, h.actor(c), c.GetString(httpx.RequestIDKey), c.Param("id"), in,
		c.GetHeader(IdempotencyHeader))
	h.respondCount(c, http.StatusOK, res, replayed, err)
}

// RejectCount godoc
//
//	@Summary		Reject a stock count
//	@Description	SUBMITTED -> REJECTED with a required reason. Posts nothing; stock is unchanged. Same separation of duties, idempotency and replay rules as approval. Requires stock.adjustment.approve and X-CSRF-Token. Audited (stock_count.rejected).
//	@Tags			stock-counts
//	@Accept			json
//	@Produce		json
//	@Param			id				path		string					true	"Stock count ID"
//	@Param			Idempotency-Key	header		string					true	"8-100 letters, digits, - or _"
//	@Param			body			body		stock.CountRejectInput	true	"Current version and the reason"
//	@Success		200				{object}	httpx.DataResponse{data=stock.StockCount}	"Rejected, or a replay (Idempotent-Replayed: true)"
//	@Failure		400				{object}	httpx.ErrorResponse	"BAD_REQUEST"
//	@Failure		401				{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403				{object}	httpx.ErrorResponse	"FORBIDDEN, SELF_APPROVAL_NOT_ALLOWED or CSRF_FAILED"
//	@Failure		404				{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Failure		409				{object}	httpx.ErrorResponse	"VERSION_CONFLICT, INVALID_STATE (not submitted) or IDEMPOTENCY_KEY_REUSED"
//	@Failure		415				{object}	httpx.ErrorResponse	"UNSUPPORTED_MEDIA_TYPE"
//	@Failure		422				{object}	httpx.ErrorResponse	"VALIDATION_FAILED (version, reason, Idempotency-Key)"
//	@Router			/stock-counts/{id}/reject [post]
func (h *Handler) RejectCount(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	var in CountRejectInput
	if !h.decode(c, &in) {
		return
	}
	res, replayed, err := h.svc.RejectCount(c.Request.Context(), a, h.actor(c), c.GetString(httpx.RequestIDKey), c.Param("id"), in,
		c.GetHeader(IdempotencyHeader))
	h.respondCount(c, http.StatusOK, res, replayed, err)
}
