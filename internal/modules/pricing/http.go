package pricing

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

// Handler is the HTTP adapter for price history.
type Handler struct {
	svc    *Service
	access *access.Handler
	log    *slog.Logger
}

// NewHandler builds the adapter.
func NewHandler(svc *Service, accessH *access.Handler, log *slog.Logger) *Handler {
	return &Handler{svc: svc, access: accessH, log: log}
}

// Register mounts the routes. There is no DELETE: drafts are cancelled and
// approved prices are superseded, never removed.
func (h *Handler) Register(rg *gin.RouterGroup) {
	view := h.access.RequireAny(access.PermPriceView, access.PermCostPriceView)
	manage := h.access.RequireAny(access.PermPriceManage, access.PermCostPriceEdit)
	approve := h.access.Require(access.PermPriceApprove)

	rg.GET("/price-lists", view, h.ListPriceLists)
	rg.POST("/price-lists", manage, h.CreatePriceList)
	rg.GET("/price-lists/:id", view, h.GetPriceList)
	rg.PATCH("/price-lists/:id", manage, h.RenamePriceList)
	rg.POST("/price-lists/:id/archive", manage, h.ArchivePriceList)

	rg.GET("/price-items", view, h.ListItems)
	rg.POST("/price-items", manage, h.CreateItem)
	rg.GET("/price-items/:id", view, h.GetItem)
	rg.PATCH("/price-items/:id", manage, h.UpdateItem)
	rg.POST("/price-items/:id/cancel", manage, h.CancelItem)
	rg.POST("/price-items/:id/approve", approve, h.ApproveItem)

	rg.GET("/prices/lookup", view, h.Lookup)
}

// PriceListPage is a page of price lists.
type PriceListPage struct {
	Items []PriceList `json:"items"`
	Page  int         `json:"page" example:"1"`
	Limit int         `json:"limit" example:"25"`
	Total int         `json:"total" example:"1"`
} //@name PriceListPage

// ItemPage is a page of price items (newest effective date first).
type ItemPage struct {
	Items []Item `json:"items"`
	Page  int    `json:"page" example:"1"`
	Limit int    `json:"limit" example:"25"`
	Total int    `json:"total" example:"1"`
} //@name PriceItemPage

func (h *Handler) actor(c *gin.Context) Actor {
	p, _ := auth.CurrentPrincipal(c)
	return Actor{UserID: p.UserID, Username: p.Username, RequestID: c.GetString(httpx.RequestIDKey)}
}

func listQuery(c *gin.Context) ListQuery {
	return ListQuery{BrandID: c.Query("brandId"), PriceListID: c.Query("priceListId"), VariantID: c.Query("variantId"),
		PriceType: c.Query("priceType"), Status: c.Query("status"), Query: c.Query("q"), Page: c.Query("page"), Limit: c.Query("limit")}
}

func (h *Handler) decode(c *gin.Context, dst any) bool {
	if c.ContentType() != "application/json" {
		httpx.Error(c, http.StatusUnsupportedMediaType, httpx.CodeUnsupportedMediaType, "Send the request body as application/json.")
		return false
	}
	dec := json.NewDecoder(io.LimitReader(c.Request.Body, 16<<10))
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
		overlap *OverlapError
		blocked *BlockedError
	)
	switch {
	case errors.Is(err, ErrForbidden):
		access.Forbidden(c)
	case errors.Is(err, ErrSelfApproval):
		httpx.Error(c, http.StatusForbidden, httpx.CodeSelfApproval,
			"You drafted this price, so another authorised person must approve it.")
	case errors.Is(err, ErrNotFound):
		httpx.Error(c, http.StatusNotFound, httpx.CodeNotFound, "The price record was not found.")
	case errors.Is(err, ErrNoPrice):
		httpx.Error(c, http.StatusNotFound, httpx.CodePriceNotFound, "No approved price is valid at that time.")
	case errors.Is(err, ErrVersionConflict):
		httpx.Error(c, http.StatusConflict, httpx.CodeVersionConflict, "This record was changed by someone else. Reload it and try again.")
	case errors.As(err, &verr):
		details := make([]httpx.FieldError, len(verr.Fields))
		for i, f := range verr.Fields {
			details[i] = httpx.FieldError{Field: f.Field, Message: f.Message}
		}
		httpx.Error(c, http.StatusUnprocessableEntity, httpx.CodeValidationFailed, "Check the highlighted fields.", details...)
	case errors.As(err, &dup):
		httpx.Error(c, http.StatusConflict, httpx.CodeDuplicate, "This "+dup.Field+" is already used.",
			httpx.FieldError{Field: dup.Field, Message: "is already used"})
	case errors.As(err, &overlap):
		var details []httpx.FieldError
		if overlap.ConflictingID != "" {
			details = append(details, httpx.FieldError{Field: "conflictingPriceId", Message: overlap.ConflictingID})
		}
		httpx.Error(c, http.StatusConflict, httpx.CodePriceOverlap,
			"An approved price already covers part of this period. Draft it as a superseding price or change the dates.", details...)
	case errors.As(err, &blocked):
		details := make([]httpx.FieldError, len(blocked.Unmet))
		for i, u := range blocked.Unmet {
			details[i] = httpx.FieldError{Field: u.Field, Message: u.Message}
		}
		httpx.Error(c, http.StatusConflict, httpx.CodeApprovalBlocked, "This price cannot be approved yet.", details...)
	case errors.As(err, &state):
		httpx.Error(c, http.StatusConflict, httpx.CodeInvalidState, state.Message)
	default:
		h.log.ErrorContext(c.Request.Context(), "price request failed",
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

// ListPriceLists godoc
//
//	@Summary		List price lists
//	@Description	Requires catalogue.price.view or catalogue.cost_price.view.
//	@Tags			pricing
//	@Produce		json
//	@Param			brandId	query	string	false	"Brand ID"
//	@Param			status	query	string	false	"Lifecycle"	Enums(ACTIVE, ARCHIVED)
//	@Param			q		query	string	false	"Search name"
//	@Param			page	query	int		false	"Page (from 1)"
//	@Param			limit	query	int		false	"Page size (1-100, default 25)"
//	@Success		200	{object}	httpx.DataResponse{data=pricing.PriceListPage}
//	@Failure		401	{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403	{object}	httpx.ErrorResponse	"FORBIDDEN"
//	@Failure		422	{object}	httpx.ErrorResponse	"VALIDATION_FAILED"
//	@Router			/price-lists [get]
func (h *Handler) ListPriceLists(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	p, err := h.svc.ListPriceLists(c.Request.Context(), a, listQuery(c))
	h.respond(c, http.StatusOK, PriceListPage{Items: p.Items, Page: p.Page, Limit: p.Limit, Total: p.Total}, err)
}

// GetPriceList godoc
//
//	@Summary		Get a price list
//	@Description	Requires catalogue.price.view or catalogue.cost_price.view.
//	@Tags			pricing
//	@Produce		json
//	@Param			id	path		string	true	"Price list ID"
//	@Success		200	{object}	httpx.DataResponse{data=pricing.PriceList}
//	@Failure		401	{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403	{object}	httpx.ErrorResponse	"FORBIDDEN"
//	@Failure		404	{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Router			/price-lists/{id} [get]
func (h *Handler) GetPriceList(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	l, err := h.svc.GetPriceList(c.Request.Context(), a, c.Param("id"))
	h.respond(c, http.StatusOK, l, err)
}

// CreatePriceList godoc
//
//	@Summary		Create a price list
//	@Description	A named list of prices for one brand (amounts in INR). Requires catalogue.price.manage or catalogue.cost_price.edit and X-CSRF-Token. Audited.
//	@Tags			pricing
//	@Accept			json
//	@Produce		json
//	@Param			body	body		pricing.PriceListCreateRequest	true	"Price list"
//	@Success		201		{object}	httpx.DataResponse{data=pricing.PriceList}
//	@Failure		401		{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403		{object}	httpx.ErrorResponse	"FORBIDDEN or CSRF_FAILED"
//	@Failure		409		{object}	httpx.ErrorResponse	"DUPLICATE (name in brand) or INVALID_STATE (archived brand)"
//	@Failure		422		{object}	httpx.ErrorResponse	"VALIDATION_FAILED"
//	@Router			/price-lists [post]
func (h *Handler) CreatePriceList(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	var in PriceListCreateRequest
	if !h.decode(c, &in) {
		return
	}
	l, err := h.svc.CreatePriceList(c.Request.Context(), h.actor(c), a, in)
	h.respond(c, http.StatusCreated, l, err)
}

// RenamePriceList godoc
//
//	@Summary		Rename a price list
//	@Description	Requires catalogue.price.manage or catalogue.cost_price.edit and X-CSRF-Token. Audited.
//	@Tags			pricing
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string							true	"Price list ID"
//	@Param			body	body		pricing.PriceListUpdateRequest	true	"Name and version"
//	@Success		200		{object}	httpx.DataResponse{data=pricing.PriceList}
//	@Failure		401		{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403		{object}	httpx.ErrorResponse	"FORBIDDEN or CSRF_FAILED"
//	@Failure		404		{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Failure		409		{object}	httpx.ErrorResponse	"DUPLICATE, VERSION_CONFLICT or INVALID_STATE"
//	@Failure		422		{object}	httpx.ErrorResponse	"VALIDATION_FAILED"
//	@Router			/price-lists/{id} [patch]
func (h *Handler) RenamePriceList(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	var in PriceListUpdateRequest
	if !h.decode(c, &in) {
		return
	}
	l, err := h.svc.RenamePriceList(c.Request.Context(), h.actor(c), a, c.Param("id"), in)
	h.respond(c, http.StatusOK, l, err)
}

// ArchivePriceList godoc
//
//	@Summary		Archive a price list
//	@Description	No new prices can be drafted or approved on it; its approved history stays readable and lookups still answer for past instants. Requires catalogue.price.manage or catalogue.cost_price.edit and X-CSRF-Token. Audited.
//	@Tags			pricing
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string					true	"Price list ID"
//	@Param			body	body		pricing.VersionRequest	true	"Version"
//	@Success		200		{object}	httpx.DataResponse{data=pricing.PriceList}
//	@Failure		401		{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403		{object}	httpx.ErrorResponse	"FORBIDDEN or CSRF_FAILED"
//	@Failure		404		{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Failure		409		{object}	httpx.ErrorResponse	"VERSION_CONFLICT or INVALID_STATE"
//	@Router			/price-lists/{id}/archive [post]
func (h *Handler) ArchivePriceList(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	var in VersionRequest
	if !h.decode(c, &in) {
		return
	}
	l, err := h.svc.ArchivePriceList(c.Request.Context(), h.actor(c), a, c.Param("id"), in)
	h.respond(c, http.StatusOK, l, err)
}

// ListItems godoc
//
//	@Summary		Price history
//	@Description	Drafts, approved and cancelled prices, newest effective date first. Only price types the caller may see are returned (COST needs catalogue.cost_price.view); asking for another type is 403. brandId, priceListId and variantId of different brands are a cross-brand mismatch: 422 (C-06).
//	@Tags			pricing
//	@Produce		json
//	@Param			priceListId	query	string	false	"Price list ID"
//	@Param			brandId		query	string	false	"Brand ID"
//	@Param			variantId	query	string	false	"Variant ID"
//	@Param			priceType	query	string	false	"Type"		Enums(SELLING, COST)
//	@Param			status		query	string	false	"Status"	Enums(DRAFT, APPROVED, CANCELLED)
//	@Param			page		query	int		false	"Page (from 1)"
//	@Param			limit		query	int		false	"Page size (1-100, default 25)"
//	@Success		200	{object}	httpx.DataResponse{data=pricing.ItemPage}
//	@Failure		401	{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403	{object}	httpx.ErrorResponse	"FORBIDDEN"
//	@Failure		422	{object}	httpx.ErrorResponse	"VALIDATION_FAILED"
//	@Router			/price-items [get]
func (h *Handler) ListItems(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	p, err := h.svc.ListItems(c.Request.Context(), a, listQuery(c))
	h.respond(c, http.StatusOK, ItemPage{Items: p.Items, Page: p.Page, Limit: p.Limit, Total: p.Total}, err)
}

// GetItem godoc
//
//	@Summary		Get a price
//	@Description	A price of a type the caller may not see is reported as 404.
//	@Tags			pricing
//	@Produce		json
//	@Param			id	path		string	true	"Price item ID"
//	@Success		200	{object}	httpx.DataResponse{data=pricing.Item}
//	@Failure		401	{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403	{object}	httpx.ErrorResponse	"FORBIDDEN"
//	@Failure		404	{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Router			/price-items/{id} [get]
func (h *Handler) GetItem(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	it, err := h.svc.GetItem(c.Request.Context(), a, c.Param("id"))
	h.respond(c, http.StatusOK, it, err)
}

// CreateItem godoc
//
//	@Summary		Draft a price
//	@Description	Drafts a price for a variant on a price list of the same brand (brand and model come from the variant). SELLING needs catalogue.price.manage, COST catalogue.cost_price.edit. taxConfigurationId must be empty until an approved tax configuration exists. supersedesId names the approved price this one replaces from its effectiveFrom. Requires X-CSRF-Token. Audited.
//	@Tags			pricing
//	@Accept			json
//	@Produce		json
//	@Param			body	body		pricing.ItemCreateRequest	true	"Draft price"
//	@Success		201		{object}	httpx.DataResponse{data=pricing.Item}
//	@Failure		401		{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403		{object}	httpx.ErrorResponse	"FORBIDDEN or CSRF_FAILED"
//	@Failure		409		{object}	httpx.ErrorResponse	"INVALID_STATE (archived list or variant)"
//	@Failure		422		{object}	httpx.ErrorResponse	"VALIDATION_FAILED (incl. variant of another brand)"
//	@Router			/price-items [post]
func (h *Handler) CreateItem(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	var in ItemCreateRequest
	if !h.decode(c, &in) {
		return
	}
	it, err := h.svc.CreateItem(c.Request.Context(), h.actor(c), a, in)
	h.respond(c, http.StatusCreated, it, err)
}

// UpdateItem godoc
//
//	@Summary		Edit a draft price
//	@Description	Only drafts change; approved prices are superseded instead. Requires the edit permission of its type and X-CSRF-Token. Audited.
//	@Tags			pricing
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string						true	"Price item ID"
//	@Param			body	body		pricing.ItemUpdateRequest	true	"Changes"
//	@Success		200		{object}	httpx.DataResponse{data=pricing.Item}
//	@Failure		401		{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403		{object}	httpx.ErrorResponse	"FORBIDDEN or CSRF_FAILED"
//	@Failure		404		{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Failure		409		{object}	httpx.ErrorResponse	"VERSION_CONFLICT or INVALID_STATE"
//	@Failure		422		{object}	httpx.ErrorResponse	"VALIDATION_FAILED"
//	@Router			/price-items/{id} [patch]
func (h *Handler) UpdateItem(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	var in ItemUpdateRequest
	if !h.decode(c, &in) {
		return
	}
	it, err := h.svc.UpdateItem(c.Request.Context(), h.actor(c), a, c.Param("id"), in)
	h.respond(c, http.StatusOK, it, err)
}

// CancelItem godoc
//
//	@Summary		Cancel a draft price
//	@Description	The draft is kept as cancelled history. Requires the edit permission of its type and X-CSRF-Token. Audited.
//	@Tags			pricing
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string					true	"Price item ID"
//	@Param			body	body		pricing.VersionRequest	true	"Version"
//	@Success		200		{object}	httpx.DataResponse{data=pricing.Item}
//	@Failure		401		{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403		{object}	httpx.ErrorResponse	"FORBIDDEN or CSRF_FAILED"
//	@Failure		404		{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Failure		409		{object}	httpx.ErrorResponse	"VERSION_CONFLICT or INVALID_STATE"
//	@Router			/price-items/{id}/cancel [post]
func (h *Handler) CancelItem(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	var in VersionRequest
	if !h.decode(c, &in) {
		return
	}
	it, err := h.svc.CancelItem(c.Request.Context(), h.actor(c), a, c.Param("id"), in)
	h.respond(c, http.StatusOK, it, err)
}

// ApproveItem godoc
//
//	@Summary		Approve a draft price
//	@Description	Requires catalogue.price.approve (plus view of its type); never the drafter (403 SELF_APPROVAL_NOT_ALLOWED). The price must not start in the past (no retroactive change), its list, brand, model and variant must not be archived and the variant must be verified (409 APPROVAL_BLOCKED). It may not overlap another approved price of the same list/variant/type (409 PRICE_OVERLAP) unless it supersedes it: then the superseded price ends at this price's start. Approved prices never change again. Refusals are audited as DENIED. Requires X-CSRF-Token.
//	@Tags			pricing
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string					true	"Price item ID"
//	@Param			body	body		pricing.ApproveRequest	true	"Version and approval reference"
//	@Success		200		{object}	httpx.DataResponse{data=pricing.Item}
//	@Failure		401		{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403		{object}	httpx.ErrorResponse	"FORBIDDEN, CSRF_FAILED or SELF_APPROVAL_NOT_ALLOWED"
//	@Failure		404		{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Failure		409		{object}	httpx.ErrorResponse	"PRICE_OVERLAP, APPROVAL_BLOCKED, VERSION_CONFLICT or INVALID_STATE"
//	@Failure		422		{object}	httpx.ErrorResponse	"VALIDATION_FAILED (retroactive start, bad supersession)"
//	@Router			/price-items/{id}/approve [post]
func (h *Handler) ApproveItem(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	var in ApproveRequest
	if !h.decode(c, &in) {
		return
	}
	it, err := h.svc.ApproveItem(c.Request.Context(), h.actor(c), a, c.Param("id"), in)
	h.respond(c, http.StatusOK, it, err)
}

// Lookup godoc
//
//	@Summary		Price valid at a time
//	@Description	The approved price of the variant and type on the list that is valid at `at` (default now). Quotations store this answer as a snapshot (SAL-003); later approvals never change the answer for a past instant. 404 PRICE_NOT_FOUND when none is valid; a variant of another brand than the list (or an unknown list/variant) is 422 (C-06). COST needs catalogue.cost_price.view.
//	@Tags			pricing
//	@Produce		json
//	@Param			priceListId	query		string	true	"Price list ID"
//	@Param			variantId	query		string	true	"Variant ID"
//	@Param			priceType	query		string	true	"Type"	Enums(SELLING, COST)
//	@Param			at			query		string	false	"RFC 3339 instant, e.g. 2026-11-01T10:00:00+05:30"
//	@Success		200			{object}	httpx.DataResponse{data=pricing.Item}
//	@Failure		401			{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403			{object}	httpx.ErrorResponse	"FORBIDDEN"
//	@Failure		404			{object}	httpx.ErrorResponse	"PRICE_NOT_FOUND"
//	@Failure		422			{object}	httpx.ErrorResponse	"VALIDATION_FAILED"
//	@Router			/prices/lookup [get]
func (h *Handler) Lookup(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	it, err := h.svc.Lookup(c.Request.Context(), a, LookupQuery{
		PriceListID: c.Query("priceListId"), VariantID: c.Query("variantId"), PriceType: c.Query("priceType"), At: c.Query("at"),
	})
	h.respond(c, http.StatusOK, it, err)
}
