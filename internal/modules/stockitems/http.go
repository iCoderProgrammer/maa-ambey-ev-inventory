package stockitems

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/httpx"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/access"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/auth"
)

// Handler is the HTTP adapter for the SKU master.
type Handler struct {
	svc    *Service
	access *access.Handler
	log    *slog.Logger
}

// NewHandler builds the adapter.
func NewHandler(svc *Service, accessH *access.Handler, log *slog.Logger) *Handler {
	return &Handler{svc: svc, access: accessH, log: log}
}

// Register mounts the routes. There is no DELETE: categories and SKUs are
// archived; no route records stock (E-04).
func (h *Handler) Register(rg *gin.RouterGroup) {
	view := h.access.Require(access.PermStockItemView)
	manage := h.access.Require(access.PermStockItemManage)
	archive := h.access.Require(access.PermStockItemArchive)

	rg.GET("/stock-item-categories", view, h.ListCategories)
	rg.POST("/stock-item-categories", manage, h.CreateCategory)
	rg.PATCH("/stock-item-categories/:id", manage, h.UpdateCategory)
	rg.POST("/stock-item-categories/:id/archive", archive, h.ArchiveCategory)
	rg.GET("/stock-items", view, h.ListItems)
	rg.POST("/stock-items", manage, h.CreateItem)
	rg.GET("/stock-items/:id", view, h.GetItem)
	rg.PATCH("/stock-items/:id", manage, h.UpdateItem)
	rg.POST("/stock-items/:id/archive", archive, h.ArchiveItem)
}

// CategoryPage is a page of categories.
type CategoryPage struct {
	Items []Category `json:"items"`
	Page  int        `json:"page" example:"1"`
	Limit int        `json:"limit" example:"25"`
	Total int        `json:"total" example:"0"`
} //@name StockItemCategoryPage

// ItemPage is a page of SKUs.
type ItemPage struct {
	Items []Item `json:"items"`
	Page  int    `json:"page" example:"1"`
	Limit int    `json:"limit" example:"25"`
	Total int    `json:"total" example:"0"`
} //@name StockItemPage

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
		httpx.Error(c, http.StatusBadRequest, httpx.CodeBadRequest, "The request body must be one JSON object (at most 64 KB).")
	}
	return false
}

func (h *Handler) fail(c *gin.Context, err error) {
	var (
		verr  *ValidationError
		dup   *DuplicateError
		state *StateError
	)
	switch {
	case errors.Is(err, ErrForbidden):
		access.Forbidden(c)
	case errors.Is(err, ErrNotFound):
		httpx.Error(c, http.StatusNotFound, httpx.CodeNotFound, "The record was not found.")
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
	case errors.As(err, &state):
		httpx.Error(c, http.StatusConflict, httpx.CodeInvalidState, state.Message)
	default:
		h.log.ErrorContext(c.Request.Context(), "stock item request failed",
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

// ListCategories godoc
//
//	@Summary		List item categories
//	@Description	Search by name; filter by status. Each category shows its number of ACTIVE SKUs. Requires stock_items.view.
//	@Tags			stock-items
//	@Produce		json
//	@Param			q		query	string	false	"Search name"
//	@Param			status	query	string	false	"Status"	Enums(ACTIVE, ARCHIVED)
//	@Param			page	query	int		false	"Page (from 1)"
//	@Param			limit	query	int		false	"Page size (1-100, default 25)"
//	@Success		200	{object}	httpx.DataResponse{data=stockitems.CategoryPage}
//	@Failure		401	{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403	{object}	httpx.ErrorResponse	"FORBIDDEN"
//	@Failure		422	{object}	httpx.ErrorResponse	"VALIDATION_FAILED (q, status, page, limit)"
//	@Router			/stock-item-categories [get]
func (h *Handler) ListCategories(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	p, err := h.svc.ListCategories(c.Request.Context(), a, CategoryQuery{Query: c.Query("q"), Status: c.Query("status"),
		Page: c.Query("page"), Limit: c.Query("limit")})
	h.respond(c, http.StatusOK, CategoryPage{Items: p.Items, Page: p.Page, Limit: p.Limit, Total: p.Total}, err)
}

// CreateCategory godoc
//
//	@Summary		Create an item category
//	@Description	Starts ACTIVE. Names are unique ignoring case (archived ones included). Requires stock_items.manage and X-CSRF-Token. Audited.
//	@Tags			stock-items
//	@Accept			json
//	@Produce		json
//	@Param			body	body		stockitems.CategoryInput	true	"Category"
//	@Success		201		{object}	httpx.DataResponse{data=stockitems.Category}
//	@Failure		401		{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403		{object}	httpx.ErrorResponse	"FORBIDDEN or CSRF_FAILED"
//	@Failure		409		{object}	httpx.ErrorResponse	"DUPLICATE (name)"
//	@Failure		422		{object}	httpx.ErrorResponse	"VALIDATION_FAILED"
//	@Router			/stock-item-categories [post]
func (h *Handler) CreateCategory(c *gin.Context) {
	with(h, c, http.StatusCreated, func(a access.Access, in CategoryInput) (any, error) {
		return h.svc.CreateCategory(c.Request.Context(), h.actor(c), a, in)
	})
}

// UpdateCategory godoc
//
//	@Summary		Edit an item category
//	@Description	Rename it or change its description. Archived categories cannot change. Requires stock_items.manage and X-CSRF-Token. Audited.
//	@Tags			stock-items
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string						true	"Category ID"
//	@Param			body	body		stockitems.CategoryPatch	true	"Changes"
//	@Success		200		{object}	httpx.DataResponse{data=stockitems.Category}
//	@Failure		401		{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403		{object}	httpx.ErrorResponse	"FORBIDDEN or CSRF_FAILED"
//	@Failure		404		{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Failure		409		{object}	httpx.ErrorResponse	"DUPLICATE (name), VERSION_CONFLICT or INVALID_STATE"
//	@Failure		422		{object}	httpx.ErrorResponse	"VALIDATION_FAILED"
//	@Router			/stock-item-categories/{id} [patch]
func (h *Handler) UpdateCategory(c *gin.Context) {
	with(h, c, http.StatusOK, func(a access.Access, in CategoryPatch) (any, error) {
		return h.svc.UpdateCategory(c.Request.Context(), h.actor(c), a, c.Param("id"), in)
	})
}

// ArchiveCategory godoc
//
//	@Summary		Archive an item category
//	@Description	Only a category without ACTIVE SKUs (move or archive them first). It stays readable and keeps its archived SKUs; it can no longer change or receive SKUs. No reactivation yet. Requires stock_items.archive and X-CSRF-Token. Audited.
//	@Tags			stock-items
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string					true	"Category ID"
//	@Param			body	body		stockitems.ArchiveInput	true	"Version and reason"
//	@Success		200		{object}	httpx.DataResponse{data=stockitems.Category}
//	@Failure		401		{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403		{object}	httpx.ErrorResponse	"FORBIDDEN or CSRF_FAILED"
//	@Failure		404		{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Failure		409		{object}	httpx.ErrorResponse	"VERSION_CONFLICT or INVALID_STATE (archived, or has active SKUs)"
//	@Failure		422		{object}	httpx.ErrorResponse	"VALIDATION_FAILED"
//	@Router			/stock-item-categories/{id}/archive [post]
func (h *Handler) ArchiveCategory(c *gin.Context) {
	with(h, c, http.StatusOK, func(a access.Access, in ArchiveInput) (any, error) {
		return h.svc.ArchiveCategory(c.Request.Context(), h.actor(c), a, c.Param("id"), in)
	})
}

// ListItems godoc
//
//	@Summary		List spare part and accessory SKUs
//	@Description	By SKU code. q matches the SKU code (case-insensitive) and the name. Filters combine; supplierId (items with a current link to that supplier) needs suppliers.view too; brandId lists items with compatibility for that brand. No stock quantities (E-04). Requires stock_items.view.
//	@Tags			stock-items
//	@Produce		json
//	@Param			q				query	string	false	"Search SKU code or name"
//	@Param			status			query	string	false	"Status"	Enums(ACTIVE, ARCHIVED)
//	@Param			categoryId		query	string	false	"Category ID"
//	@Param			trackingMode	query	string	false	"Tracking mode"	Enums(QUANTITY, SERIAL)
//	@Param			supplierId		query	string	false	"Supplier ID (needs suppliers.view)"
//	@Param			brandId			query	string	false	"Compatible brand ID"
//	@Param			page			query	int		false	"Page (from 1)"
//	@Param			limit			query	int		false	"Page size (1-100, default 25)"
//	@Success		200	{object}	httpx.DataResponse{data=stockitems.ItemPage}
//	@Failure		401	{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403	{object}	httpx.ErrorResponse	"FORBIDDEN"
//	@Failure		422	{object}	httpx.ErrorResponse	"VALIDATION_FAILED (q, status, categoryId, trackingMode, supplierId, brandId, page, limit)"
//	@Router			/stock-items [get]
func (h *Handler) ListItems(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	p, err := h.svc.ListItems(c.Request.Context(), a, ListQuery{Query: c.Query("q"), Status: c.Query("status"),
		CategoryID: c.Query("categoryId"), TrackingMode: c.Query("trackingMode"), SupplierID: c.Query("supplierId"),
		BrandID: c.Query("brandId"), Page: c.Query("page"), Limit: c.Query("limit")})
	h.respond(c, http.StatusOK, ItemPage{Items: p.Items, Page: p.Page, Limit: p.Limit, Total: p.Total}, err)
}

// GetItem godoc
//
//	@Summary		Get a SKU
//	@Description	The SKU with its supplier links (current first, then ended ones as history) and compatibility. Archived SKUs stay readable. Without suppliers.view, suppliers is empty and suppliersHidden true. Requires stock_items.view.
//	@Tags			stock-items
//	@Produce		json
//	@Param			id	path		string	true	"SKU ID"
//	@Success		200	{object}	httpx.DataResponse{data=stockitems.Detail}
//	@Failure		401	{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403	{object}	httpx.ErrorResponse	"FORBIDDEN"
//	@Failure		404	{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Router			/stock-items/{id} [get]
func (h *Handler) GetItem(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	d, err := h.svc.GetItem(c.Request.Context(), a, c.Param("id"))
	h.respond(c, http.StatusOK, d, err)
}

// CreateItem godoc
//
//	@Summary		Create a SKU
//	@Description	Starts ACTIVE in an ACTIVE category. sku: 1-40 letters, digits, . _ / - (no spaces), unique compared upper-cased across all SKUs (archived ones included); it and trackingMode (QUANTITY or SERIAL) cannot change later. uom: unit code, stored upper-case. reorderLevel: decimal string ≥ 0, up to 3 decimals, whole for SERIAL. suppliers (not archived; needs suppliers.view) and compatibility (brand, optionally one of its models; not archived; needs catalogue.view) are optional. Records no stock. Requires stock_items.manage and X-CSRF-Token. Audited.
//	@Tags			stock-items
//	@Accept			json
//	@Produce		json
//	@Param			body	body		stockitems.ItemInput	true	"SKU"
//	@Success		201		{object}	httpx.DataResponse{data=stockitems.Detail}
//	@Failure		401		{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403		{object}	httpx.ErrorResponse	"FORBIDDEN or CSRF_FAILED"
//	@Failure		409		{object}	httpx.ErrorResponse	"DUPLICATE (sku)"
//	@Failure		422		{object}	httpx.ErrorResponse	"VALIDATION_FAILED (sku, name, description, categoryId, uom, trackingMode, reorderLevel, suppliers[i].*, compatibility[i].*)"
//	@Router			/stock-items [post]
func (h *Handler) CreateItem(c *gin.Context) {
	with(h, c, http.StatusCreated, func(a access.Access, in ItemInput) (any, error) {
		return h.svc.CreateItem(c.Request.Context(), h.actor(c), a, in)
	})
}

// UpdateItem godoc
//
//	@Summary		Edit a SKU
//	@Description	Change name, description, category (ACTIVE), uom or reorderLevel; suppliers and compatibility, when sent, replace the current set (removed supplier links are ended and kept). sku and trackingMode cannot be changed. Archived SKUs cannot change. Requires stock_items.manage (plus suppliers.view / catalogue.view to send suppliers / compatibility) and X-CSRF-Token. Audited.
//	@Tags			stock-items
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string					true	"SKU ID"
//	@Param			body	body		stockitems.ItemPatch	true	"Changes"
//	@Success		200		{object}	httpx.DataResponse{data=stockitems.Detail}
//	@Failure		401		{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403		{object}	httpx.ErrorResponse	"FORBIDDEN or CSRF_FAILED"
//	@Failure		404		{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Failure		409		{object}	httpx.ErrorResponse	"VERSION_CONFLICT or INVALID_STATE (archived)"
//	@Failure		422		{object}	httpx.ErrorResponse	"VALIDATION_FAILED (sku and trackingMode: cannot be set here)"
//	@Router			/stock-items/{id} [patch]
func (h *Handler) UpdateItem(c *gin.Context) {
	with(h, c, http.StatusOK, func(a access.Access, in ItemPatch) (any, error) {
		return h.svc.UpdateItem(c.Request.Context(), h.actor(c), a, c.Param("id"), in)
	})
}

// ArchiveItem godoc
//
//	@Summary		Archive a SKU
//	@Description	The SKU stays readable with its code, supplier links, compatibility and references; it can no longer change, its code stays taken, and it is not offered for new transactions (status ARCHIVED). Never deleted; no reactivation yet. Requires stock_items.archive and X-CSRF-Token. Audited.
//	@Tags			stock-items
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string					true	"SKU ID"
//	@Param			body	body		stockitems.ArchiveInput	true	"Version and reason"
//	@Success		200		{object}	httpx.DataResponse{data=stockitems.Detail}
//	@Failure		401		{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403		{object}	httpx.ErrorResponse	"FORBIDDEN or CSRF_FAILED"
//	@Failure		404		{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Failure		409		{object}	httpx.ErrorResponse	"VERSION_CONFLICT or INVALID_STATE (already archived)"
//	@Failure		422		{object}	httpx.ErrorResponse	"VALIDATION_FAILED (version, reason)"
//	@Router			/stock-items/{id}/archive [post]
func (h *Handler) ArchiveItem(c *gin.Context) {
	with(h, c, http.StatusOK, func(a access.Access, in ArchiveInput) (any, error) {
		return h.svc.ArchiveItem(c.Request.Context(), h.actor(c), a, c.Param("id"), in)
	})
}
