package stock

import "github.com/gin-gonic/gin"

// ListTimeline godoc
//
//	@Summary		Stock movement timeline
//	@Description	Read-only list of stock ledger entries across items and locations, newest first by default (order = ledger sequence; asc for oldest first). Rows are what the caller may see: VEHICLE_UNIT movements need vehicle_units.view and a unit whose current location is in the caller's scope (or that has none yet); STOCK_ITEM movements need stock.parts.view at a location in scope. Filters combine with AND: itemType, vehicleUnitId, stockItemId, q (substring of the unit number, chassis/frame number or SKU code, case-insensitive), movementType, sourceType, sourceReference (case-insensitive exact; only sources the caller may open match), actorId, locationId (the movement's location; must be in scope), from (inclusive) and to (exclusive) as RFC 3339 instants on the server time of the movement. A source the caller may not open (goods receipt without goods_receipts.view, stock count without stock.adjustment.view) is returned as sourceHidden: true with sourceId, sourceReference, reason and note null; supplier fields need suppliers.view; a location outside the scope is named by ID only. No cost, document or storage reference is returned. Nothing can be created, edited or deleted here. Requires vehicle_units.view or stock.parts.view.
//	@Tags			stock-movements
//	@Produce		json
//	@Param			itemType		query	string	false	"Item type"	Enums(VEHICLE_UNIT, STOCK_ITEM)
//	@Param			vehicleUnitId	query	string	false	"Vehicle unit ID"
//	@Param			stockItemId		query	string	false	"SKU ID (not with vehicleUnitId)"
//	@Param			q				query	string	false	"Unit number, chassis/frame number or SKU code (substring, max 100)"
//	@Param			movementType	query	string	false	"Movement type"	Enums(RECEIPT, PDI_PASS, QUALITY_HOLD, REJECT, DISPLAY_ON, DISPLAY_OFF, PARTS_RECEIPT, PARTS_ISSUE, COUNT_ADJUSTMENT)
//	@Param			sourceType		query	string	false	"Source type"	Enums(GOODS_RECEIPT, PDI_INSPECTION, STAFF_ACTION, SUPPLIER_DELIVERY, STOCK_COUNT, OPENING_STOCK)
//	@Param			sourceReference	query	string	false	"Source reference, e.g. GRN-000001 (exact, case-insensitive, max 100)"
//	@Param			actorId			query	string	false	"User ID of the actor"
//	@Param			locationId		query	string	false	"Location ID (in the caller's scope)"
//	@Param			from			query	string	false	"From (inclusive), RFC 3339, e.g. 2026-01-15T00:00:00+05:30"
//	@Param			to				query	string	false	"To (exclusive), RFC 3339; after from"
//	@Param			order			query	string	false	"Ledger order (default desc)"	Enums(desc, asc)
//	@Param			page			query	int		false	"Page (from 1)"
//	@Param			limit			query	int		false	"Page size (1-100, default 25)"
//	@Success		200	{object}	httpx.DataResponse{data=stock.TimelinePage}
//	@Failure		401	{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403	{object}	httpx.ErrorResponse	"FORBIDDEN"
//	@Failure		422	{object}	httpx.ErrorResponse	"VALIDATION_FAILED (itemType, vehicleUnitId, stockItemId, q, movementType, sourceType, sourceReference, actorId, locationId, from, to, order, page, limit)"
//	@Router			/stock-movements [get]
func (h *Handler) ListTimeline(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	p, err := h.svc.Timeline(c.Request.Context(), a, TimelineQuery{ItemType: c.Query("itemType"),
		VehicleUnitID: c.Query("vehicleUnitId"), StockItemID: c.Query("stockItemId"), Query: c.Query("q"),
		MovementType: c.Query("movementType"), SourceType: c.Query("sourceType"), SourceReference: c.Query("sourceReference"),
		ActorID: c.Query("actorId"), LocationID: c.Query("locationId"), From: c.Query("from"), To: c.Query("to"),
		Order: c.Query("order"), Page: c.Query("page"), Limit: c.Query("limit")})
	h.respond(c, p, false, err)
}

// GetTimelineMovement godoc
//
//	@Summary		One stock movement
//	@Description	The same entry as in the timeline, with the same visibility and shaping. An unknown or malformed ID, a movement of an item type the caller may not view, or one outside the caller's location scope is 404 (its existence is not revealed). Read-only. Requires vehicle_units.view or stock.parts.view.
//	@Tags			stock-movements
//	@Produce		json
//	@Param			id	path		string	true	"Stock movement ID"
//	@Success		200	{object}	httpx.DataResponse{data=stock.TimelineMovement}
//	@Failure		401	{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403	{object}	httpx.ErrorResponse	"FORBIDDEN"
//	@Failure		404	{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Router			/stock-movements/{id} [get]
func (h *Handler) GetTimelineMovement(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	m, err := h.svc.TimelineMovement(c.Request.Context(), a, c.Param("id"))
	h.respond(c, m, false, err)
}
