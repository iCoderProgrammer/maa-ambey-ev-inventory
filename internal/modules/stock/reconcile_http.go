package stock

import "github.com/gin-gonic/gin"

// Reconcile godoc
//
//	@Summary		Stock ledger reconciliation
//	@Description	Runs the read-only ledger reconciliation now and returns its summary and one page of discrepancies. Vehicle units are compared with their stock movements: no movement (VEHICLE_NO_MOVEMENTS), history not starting with the RECEIPT (VEHICLE_MISSING_RECEIPT), a later RECEIPT (VEHICLE_REPEATED_RECEIPT), status or location different from the latest movement (VEHICLE_STATUS_MISMATCH, VEHICLE_LOCATION_MISMATCH), latest movement not leaving the unit held (VEHICLE_NOT_HELD), a movement not starting where the previous one ended (VEHICLE_BROKEN_CHAIN). Parts balances per SKU and location: balance different from the quantity after the latest movement, 0 without one (PARTS_BALANCE_MISMATCH), movements without a balance record (PARTS_BALANCE_MISSING), a balance below zero (PARTS_NEGATIVE_BALANCE), a movement not starting from the quantity the previous one left, the first from 0 (PARTS_BROKEN_CHAIN). HIGH: the record disagrees with the ledger or its origin is missing; MEDIUM: the history is inconsistent while the record matches its latest movement. Every query runs in one read-only, repeatable-read transaction: the result is one consistent snapshot and nothing is written, corrected or stored (runs are logged on the server under runId only). The summary counts the whole run; itemType, type, severity and locationId (in scope) filter the returned items, ordered by severity, type and reference; reference is stable across runs. Rows are what the caller may see: vehicle units need vehicle_units.view and a current location in scope (or none yet); parts balances need stock.parts.view at a location in scope; a location outside the scope is named by ID only. No cost is returned. Requires stock.reconciliation.view and vehicle_units.view or stock.parts.view.
//	@Tags			stock-reconciliation
//	@Produce		json
//	@Param			itemType	query	string	false	"Item type"	Enums(VEHICLE_UNIT, STOCK_ITEM)
//	@Param			type		query	string	false	"Discrepancy type"	Enums(VEHICLE_NO_MOVEMENTS, VEHICLE_MISSING_RECEIPT, VEHICLE_REPEATED_RECEIPT, VEHICLE_STATUS_MISMATCH, VEHICLE_LOCATION_MISMATCH, VEHICLE_NOT_HELD, VEHICLE_BROKEN_CHAIN, PARTS_BALANCE_MISMATCH, PARTS_BALANCE_MISSING, PARTS_NEGATIVE_BALANCE, PARTS_BROKEN_CHAIN)
//	@Param			severity	query	string	false	"Severity"	Enums(HIGH, MEDIUM)
//	@Param			locationId	query	string	false	"Location ID of the record (in the caller's scope)"
//	@Param			page		query	int		false	"Page (from 1)"
//	@Param			limit		query	int		false	"Page size (1-100, default 50)"
//	@Success		200	{object}	httpx.DataResponse{data=stock.ReconcileReport}
//	@Failure		401	{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403	{object}	httpx.ErrorResponse	"FORBIDDEN"
//	@Failure		422	{object}	httpx.ErrorResponse	"VALIDATION_FAILED (itemType, type, severity, locationId, page, limit)"
//	@Failure		500	{object}	httpx.ErrorResponse	"INTERNAL_ERROR (the run failed; nothing was changed)"
//	@Router			/stock-reconciliation [get]
func (h *Handler) Reconcile(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	r, err := h.svc.Reconcile(c.Request.Context(), a, ReconcileQuery{ItemType: c.Query("itemType"), Type: c.Query("type"),
		Severity: c.Query("severity"), LocationID: c.Query("locationId"), Page: c.Query("page"), Limit: c.Query("limit")})
	h.respond(c, r, false, err)
}
