package catalogue

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

const maxBodyBytes = 16 << 10

// Handler is the HTTP adapter for the catalogue.
type Handler struct {
	svc    *Service
	access *access.Handler
	log    *slog.Logger
}

// NewHandler builds the adapter.
func NewHandler(svc *Service, accessH *access.Handler, log *slog.Logger) *Handler {
	return &Handler{svc: svc, access: accessH, log: log}
}

// Register mounts the routes behind the session middleware. There is no
// DELETE route: catalogue items are archived, never deleted (BR-006).
func (h *Handler) Register(rg *gin.RouterGroup) {
	view := h.access.Require(access.PermCatalogueView)
	manage := h.access.Require(access.PermCatalogueManage)
	verify := h.access.Require(access.PermCatalogueVerify)
	activate := h.access.Require(access.PermCatalogueActivate)
	archive := h.access.Require(access.PermCatalogueArchive)

	rg.GET("/brands", view, h.ListBrands)
	rg.POST("/brands", manage, h.CreateBrand)
	rg.GET("/brands/:id", view, h.GetBrand)
	rg.PATCH("/brands/:id", manage, h.UpdateBrand)
	rg.PUT("/brands/:id/authorisation", h.access.Require(access.PermBrandAuthorise), h.RecordBrandAuthorisation)
	rg.GET("/brands/:id/activation-readiness", view, h.BrandReadiness)
	rg.POST("/brands/:id/activate", h.access.Require(access.PermBrandActivate), h.ActivateBrand)
	rg.POST("/brands/:id/archive", archive, h.ArchiveBrand)

	rg.GET("/models", view, h.ListModels)
	rg.POST("/models", manage, h.CreateModel)
	rg.GET("/models/:id", view, h.GetModel)
	rg.PATCH("/models/:id", manage, h.UpdateModel)
	rg.POST("/models/:id/verify", verify, h.VerifyModel)
	rg.POST("/models/:id/activate", activate, h.ActivateModel)
	rg.POST("/models/:id/archive", archive, h.ArchiveModel)

	rg.GET("/variants", view, h.ListVariants)
	rg.POST("/variants", manage, h.CreateVariant)
	rg.GET("/variants/:id", view, h.GetVariant)
	rg.PATCH("/variants/:id", manage, h.UpdateVariant)
	rg.POST("/variants/:id/verify", verify, h.VerifyVariant)
	rg.POST("/variants/:id/activate", activate, h.ActivateVariant)
	rg.POST("/variants/:id/archive", archive, h.ArchiveVariant)

	rg.GET("/colours", view, h.ListColours)
	rg.POST("/colours", manage, h.CreateColour)
	rg.GET("/colours/:id", view, h.GetColour)
	rg.PATCH("/colours/:id", manage, h.UpdateColour)
	rg.POST("/colours/:id/verify", verify, h.VerifyColour)
	rg.POST("/colours/:id/activate", activate, h.ActivateColour)
	rg.POST("/colours/:id/archive", archive, h.ArchiveColour)
}

// List envelopes (one per type so the OpenAPI document is concrete).

// BrandList is a page of brands.
type BrandList struct {
	Items []Brand `json:"items"`
	Page  int     `json:"page" example:"1"`
	Limit int     `json:"limit" example:"25"`
	Total int     `json:"total" example:"3"`
} //@name CatalogueBrandList

// ModelList is a page of models.
type ModelList struct {
	Items []Model `json:"items"`
	Page  int     `json:"page" example:"1"`
	Limit int     `json:"limit" example:"25"`
	Total int     `json:"total" example:"0"`
} //@name CatalogueModelList

// VariantList is a page of variants.
type VariantList struct {
	Items []Variant `json:"items"`
	Page  int       `json:"page" example:"1"`
	Limit int       `json:"limit" example:"25"`
	Total int       `json:"total" example:"0"`
} //@name CatalogueVariantList

// ColourList is a page of colours.
type ColourList struct {
	Items []Colour `json:"items"`
	Page  int      `json:"page" example:"1"`
	Limit int      `json:"limit" example:"25"`
	Total int      `json:"total" example:"0"`
} //@name CatalogueColourList

func (h *Handler) actor(c *gin.Context) Actor {
	p, _ := auth.CurrentPrincipal(c)
	return Actor{UserID: p.UserID, Username: p.Username, RequestID: c.GetString(httpx.RequestIDKey)}
}

func listQuery(c *gin.Context) ListQuery {
	return ListQuery{
		Query: c.Query("q"), Status: c.Query("status"), Confidence: c.Query("confidence"),
		AuthStatus: c.Query("authorisationStatus"), BrandID: c.Query("brandId"), ModelID: c.Query("modelId"),
		VariantID: c.Query("variantId"), Page: c.Query("page"), Limit: c.Query("limit"),
	}
}

// decode reads one JSON object. Unknown fields are refused, so a client
// cannot set server-owned fields (status, verifier, a child's brandId).
func (h *Handler) decode(c *gin.Context, dst any) bool {
	if c.ContentType() != "application/json" {
		httpx.Error(c, http.StatusUnsupportedMediaType, httpx.CodeUnsupportedMediaType, "Send the request body as application/json.")
		return false
	}
	dec := json.NewDecoder(io.LimitReader(c.Request.Body, maxBodyBytes))
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

// fail maps service errors to responses.
func (h *Handler) fail(c *gin.Context, err error) {
	var (
		verr    *ValidationError
		dup     *DuplicateError
		state   *StateError
		blocked *ActivationBlockedError
	)
	switch {
	case errors.Is(err, ErrForbidden):
		access.Forbidden(c)
	case errors.Is(err, ErrNotFound):
		httpx.Error(c, http.StatusNotFound, httpx.CodeNotFound, "The catalogue item was not found.")
	case errors.As(err, &verr):
		details := make([]httpx.FieldError, len(verr.Fields))
		for i, f := range verr.Fields {
			details[i] = httpx.FieldError{Field: f.Field, Message: f.Message}
		}
		httpx.Error(c, http.StatusUnprocessableEntity, httpx.CodeValidationFailed, "Check the highlighted fields.", details...)
	case errors.As(err, &dup):
		httpx.Error(c, http.StatusConflict, httpx.CodeDuplicate, "This "+dup.Field+" is already used.",
			httpx.FieldError{Field: dup.Field, Message: "is already used"})
	case errors.Is(err, ErrVersionConflict):
		httpx.Error(c, http.StatusConflict, httpx.CodeVersionConflict, "This item was changed by someone else. Reload it and try again.")
	case errors.As(err, &state):
		httpx.Error(c, http.StatusConflict, httpx.CodeInvalidState, state.Message)
	case errors.As(err, &blocked):
		details := make([]httpx.FieldError, len(blocked.Unmet))
		for i, u := range blocked.Unmet {
			details[i] = httpx.FieldError{Field: u.Key, Message: u.Message}
		}
		httpx.Error(c, http.StatusConflict, httpx.CodeActivationBlocked, "Activation is not allowed yet.", details...)
	default:
		h.log.ErrorContext(c.Request.Context(), "catalogue request failed",
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

// ── Brands ───────────────────────────────────────────────────────────────────

// ListBrands godoc
//
//	@Summary		List brands
//	@Description	Brands are data. Includes pending and archived brands (BR-008); operational=true only for ACTIVE + AUTHORISED. Sorted by name. Requires catalogue.view.
//	@Tags			catalogue
//	@Produce		json
//	@Param			q					query	string	false	"Search name or slug"
//	@Param			status				query	string	false	"Lifecycle"	Enums(PENDING_VERIFICATION, ACTIVE, ARCHIVED)
//	@Param			authorisationStatus	query	string	false	"Authorisation"	Enums(UNVERIFIED, AUTHORISED, NOT_AUTHORISED, REVOKED)
//	@Param			page				query	int		false	"Page (from 1)"
//	@Param			limit				query	int		false	"Page size (1-100, default 25)"
//	@Success		200	{object}	httpx.DataResponse{data=catalogue.BrandList}
//	@Failure		401	{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403	{object}	httpx.ErrorResponse	"FORBIDDEN"
//	@Failure		422	{object}	httpx.ErrorResponse	"VALIDATION_FAILED"
//	@Router			/brands [get]
func (h *Handler) ListBrands(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	p, err := h.svc.ListBrands(c.Request.Context(), a, listQuery(c))
	h.respond(c, http.StatusOK, BrandList{Items: p.Items, Page: p.Page, Limit: p.Limit, Total: p.Total}, err)
}

// GetBrand godoc
//
//	@Summary		Get a brand
//	@Description	Requires catalogue.view.
//	@Tags			catalogue
//	@Produce		json
//	@Param			id	path		string	true	"Brand ID"
//	@Success		200	{object}	httpx.DataResponse{data=catalogue.Brand}
//	@Failure		401	{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403	{object}	httpx.ErrorResponse	"FORBIDDEN"
//	@Failure		404	{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Router			/brands/{id} [get]
func (h *Handler) GetBrand(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	b, err := h.svc.GetBrand(c.Request.Context(), a, c.Param("id"))
	h.respond(c, http.StatusOK, b, err)
}

// CreateBrand godoc
//
//	@Summary		Create a brand
//	@Description	Adds a brand as data. It starts PENDING_VERIFICATION / UNVERIFIED and is not usable for purchase/sales. Slug is derived from the name when omitted. Audited. Requires catalogue.manage and X-CSRF-Token.
//	@Tags			catalogue
//	@Accept			json
//	@Produce		json
//	@Param			body	body		catalogue.BrandCreateRequest	true	"Brand"
//	@Success		201		{object}	httpx.DataResponse{data=catalogue.Brand}
//	@Failure		400		{object}	httpx.ErrorResponse	"BAD_REQUEST"
//	@Failure		401		{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403		{object}	httpx.ErrorResponse	"FORBIDDEN or CSRF_FAILED"
//	@Failure		409		{object}	httpx.ErrorResponse	"DUPLICATE (name or slug)"
//	@Failure		415		{object}	httpx.ErrorResponse	"UNSUPPORTED_MEDIA_TYPE"
//	@Failure		422		{object}	httpx.ErrorResponse	"VALIDATION_FAILED"
//	@Router			/brands [post]
func (h *Handler) CreateBrand(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	var in BrandCreateRequest
	if !h.decode(c, &in) {
		return
	}
	b, err := h.svc.CreateBrand(c.Request.Context(), h.actor(c), a, in)
	h.respond(c, http.StatusCreated, b, err)
}

// UpdateBrand godoc
//
//	@Summary		Edit a brand
//	@Description	Changes name and/or slug. Send the version you read; archived brands cannot be changed. Audited. Requires catalogue.manage and X-CSRF-Token.
//	@Tags			catalogue
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string							true	"Brand ID"
//	@Param			body	body		catalogue.BrandUpdateRequest	true	"Changes"
//	@Success		200		{object}	httpx.DataResponse{data=catalogue.Brand}
//	@Failure		401		{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403		{object}	httpx.ErrorResponse	"FORBIDDEN or CSRF_FAILED"
//	@Failure		404		{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Failure		409		{object}	httpx.ErrorResponse	"DUPLICATE, VERSION_CONFLICT or INVALID_STATE"
//	@Failure		422		{object}	httpx.ErrorResponse	"VALIDATION_FAILED"
//	@Router			/brands/{id} [patch]
func (h *Handler) UpdateBrand(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	var in BrandUpdateRequest
	if !h.decode(c, &in) {
		return
	}
	b, err := h.svc.UpdateBrand(c.Request.Context(), h.actor(c), a, c.Param("id"), in)
	h.respond(c, http.StatusOK, b, err)
}

// RecordBrandAuthorisation godoc
//
//	@Summary		Record brand authorisation
//	@Description	Records the supplier/manufacturer legal identity, authorisation status, reference and evidence document (BR-002). AUTHORISED needs all three; the caller is recorded as verifier with the current time (UNVERIFIED clears both). Changing it on an ACTIVE brand takes it out of operation at once (operational=false). Archived brands cannot be changed. Audited. Requires catalogue.brand.authorise and X-CSRF-Token.
//	@Tags			catalogue
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string								true	"Brand ID"
//	@Param			body	body		catalogue.BrandAuthorisationRequest	true	"Authorisation"
//	@Success		200		{object}	httpx.DataResponse{data=catalogue.Brand}
//	@Failure		401		{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403		{object}	httpx.ErrorResponse	"FORBIDDEN or CSRF_FAILED"
//	@Failure		404		{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Failure		409		{object}	httpx.ErrorResponse	"VERSION_CONFLICT or INVALID_STATE"
//	@Failure		422		{object}	httpx.ErrorResponse	"VALIDATION_FAILED (missing evidence, unknown document)"
//	@Router			/brands/{id}/authorisation [put]
func (h *Handler) RecordBrandAuthorisation(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	var in BrandAuthorisationRequest
	if !h.decode(c, &in) {
		return
	}
	b, err := h.svc.RecordBrandAuthorisation(c.Request.Context(), h.actor(c), a, c.Param("id"), in)
	h.respond(c, http.StatusOK, b, err)
}

// BrandReadiness godoc
//
//	@Summary		Brand activation readiness
//	@Description	Every prerequisite for enabling the brand for purchase/sales (BR-002) and whether it is met. approved_prices needs an approved SELLING price valid now for an active variant (C-04). Checks marked pending belong to features not built yet (supplier links, warranty terms, identifier rules) and are never met, so no brand can be activated until those exist. Requires catalogue.view.
//	@Tags			catalogue
//	@Produce		json
//	@Param			id	path		string	true	"Brand ID"
//	@Success		200	{object}	httpx.DataResponse{data=catalogue.Readiness}
//	@Failure		401	{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403	{object}	httpx.ErrorResponse	"FORBIDDEN"
//	@Failure		404	{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Router			/brands/{id}/activation-readiness [get]
func (h *Handler) BrandReadiness(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	r, err := h.svc.BrandReadiness(c.Request.Context(), a, c.Param("id"))
	h.respond(c, http.StatusOK, r, err)
}

// ActivateBrand godoc
//
//	@Summary		Enable a brand for purchase and sales
//	@Description	Allowed only when every readiness check is met; otherwise 409 ACTIVATION_BLOCKED lists the unmet checks in details (field = check key) and the refused attempt is audited (outcome DENIED). Requires catalogue.brand.activate and X-CSRF-Token.
//	@Tags			catalogue
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string					true	"Brand ID"
//	@Param			body	body		catalogue.ActionRequest	true	"Version"
//	@Success		200		{object}	httpx.DataResponse{data=catalogue.Brand}
//	@Failure		401		{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403		{object}	httpx.ErrorResponse	"FORBIDDEN or CSRF_FAILED"
//	@Failure		404		{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Failure		409		{object}	httpx.ErrorResponse	"ACTIVATION_BLOCKED, VERSION_CONFLICT or INVALID_STATE"
//	@Failure		422		{object}	httpx.ErrorResponse	"VALIDATION_FAILED"
//	@Router			/brands/{id}/activate [post]
func (h *Handler) ActivateBrand(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	var in ActionRequest
	if !h.decode(c, &in) {
		return
	}
	b, err := h.svc.ActivateBrand(c.Request.Context(), h.actor(c), a, c.Param("id"), in)
	h.respond(c, http.StatusOK, b, err)
}

// ArchiveBrand godoc
//
//	@Summary		Archive a brand
//	@Description	Takes the brand out of use without deleting anything; its models, variants and history keep their references (BR-006, BR-008). Audited with the optional reason. Requires catalogue.archive and X-CSRF-Token.
//	@Tags			catalogue
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string						true	"Brand ID"
//	@Param			body	body		catalogue.ArchiveRequest	true	"Version and reason"
//	@Success		200		{object}	httpx.DataResponse{data=catalogue.Brand}
//	@Failure		401		{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403		{object}	httpx.ErrorResponse	"FORBIDDEN or CSRF_FAILED"
//	@Failure		404		{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Failure		409		{object}	httpx.ErrorResponse	"VERSION_CONFLICT or INVALID_STATE"
//	@Failure		422		{object}	httpx.ErrorResponse	"VALIDATION_FAILED"
//	@Router			/brands/{id}/archive [post]
func (h *Handler) ArchiveBrand(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	var in ArchiveRequest
	if !h.decode(c, &in) {
		return
	}
	b, err := h.svc.ArchiveBrand(c.Request.Context(), h.actor(c), a, c.Param("id"), in)
	h.respond(c, http.StatusOK, b, err)
}

// ── Models ───────────────────────────────────────────────────────────────────

// ListModels godoc
//
//	@Summary		List models
//	@Description	Sorted by name. Requires catalogue.view.
//	@Tags			catalogue
//	@Produce		json
//	@Param			brandId		query	string	false	"Brand ID"
//	@Param			q			query	string	false	"Search name, internal or manufacturer code"
//	@Param			status		query	string	false	"Lifecycle"	Enums(PENDING_VERIFICATION, ACTIVE, ARCHIVED)
//	@Param			confidence	query	string	false	"Confidence"	Enums(UNVERIFIED, SUPPLIER_CONFIRMED, CLIENT_APPROVED)
//	@Param			page		query	int		false	"Page (from 1)"
//	@Param			limit		query	int		false	"Page size (1-100, default 25)"
//	@Success		200	{object}	httpx.DataResponse{data=catalogue.ModelList}
//	@Failure		401	{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403	{object}	httpx.ErrorResponse	"FORBIDDEN"
//	@Failure		422	{object}	httpx.ErrorResponse	"VALIDATION_FAILED"
//	@Router			/models [get]
func (h *Handler) ListModels(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	p, err := h.svc.ListModels(c.Request.Context(), a, listQuery(c))
	h.respond(c, http.StatusOK, ModelList{Items: p.Items, Page: p.Page, Limit: p.Limit, Total: p.Total}, err)
}

// GetModel godoc
//
//	@Summary		Get a model
//	@Description	Requires catalogue.view.
//	@Tags			catalogue
//	@Produce		json
//	@Param			id	path		string	true	"Model ID"
//	@Success		200	{object}	httpx.DataResponse{data=catalogue.Model}
//	@Failure		401	{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403	{object}	httpx.ErrorResponse	"FORBIDDEN"
//	@Failure		404	{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Router			/models/{id} [get]
func (h *Handler) GetModel(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	m, err := h.svc.GetModel(c.Request.Context(), a, c.Param("id"))
	h.respond(c, http.StatusOK, m, err)
}

// CreateModel godoc
//
//	@Summary		Create a model
//	@Description	Adds a PENDING_VERIFICATION / UNVERIFIED model to a brand that is not archived. internalCode is stored upper-case and is unique within the brand. Audited. Requires catalogue.manage and X-CSRF-Token.
//	@Tags			catalogue
//	@Accept			json
//	@Produce		json
//	@Param			body	body		catalogue.ModelCreateRequest	true	"Model"
//	@Success		201		{object}	httpx.DataResponse{data=catalogue.Model}
//	@Failure		401		{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403		{object}	httpx.ErrorResponse	"FORBIDDEN or CSRF_FAILED"
//	@Failure		409		{object}	httpx.ErrorResponse	"DUPLICATE or INVALID_STATE (archived brand)"
//	@Failure		422		{object}	httpx.ErrorResponse	"VALIDATION_FAILED (incl. unknown brandId)"
//	@Router			/models [post]
func (h *Handler) CreateModel(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	var in ModelCreateRequest
	if !h.decode(c, &in) {
		return
	}
	m, err := h.svc.CreateModel(c.Request.Context(), h.actor(c), a, in)
	h.respond(c, http.StatusCreated, m, err)
}

// UpdateModel godoc
//
//	@Summary		Edit a pending model
//	@Description	Only PENDING_VERIFICATION models can be edited; the brand never changes (brandId is refused). Editing a verified model resets it to UNVERIFIED. Send null to clear manufacturerCode or vehicleType. Audited. Requires catalogue.manage and X-CSRF-Token.
//	@Tags			catalogue
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string							true	"Model ID"
//	@Param			body	body		catalogue.ModelUpdateRequest	true	"Changes"
//	@Success		200		{object}	httpx.DataResponse{data=catalogue.Model}
//	@Failure		401		{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403		{object}	httpx.ErrorResponse	"FORBIDDEN or CSRF_FAILED"
//	@Failure		404		{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Failure		409		{object}	httpx.ErrorResponse	"DUPLICATE, VERSION_CONFLICT or INVALID_STATE"
//	@Failure		422		{object}	httpx.ErrorResponse	"VALIDATION_FAILED"
//	@Router			/models/{id} [patch]
func (h *Handler) UpdateModel(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	var in ModelUpdateRequest
	if !h.decode(c, &in) {
		return
	}
	m, err := h.svc.UpdateModel(c.Request.Context(), h.actor(c), a, c.Param("id"), in)
	h.respond(c, http.StatusOK, m, err)
}

// VerifyModel godoc
//
//	@Summary		Verify a model
//	@Description	Records source (URL and/or private document), source date and confidence for a pending model; the caller is the verifier. Audited. Requires catalogue.verify and X-CSRF-Token.
//	@Tags			catalogue
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string					true	"Model ID"
//	@Param			body	body		catalogue.VerifyRequest	true	"Provenance"
//	@Success		200		{object}	httpx.DataResponse{data=catalogue.Model}
//	@Failure		401		{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403		{object}	httpx.ErrorResponse	"FORBIDDEN or CSRF_FAILED"
//	@Failure		404		{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Failure		409		{object}	httpx.ErrorResponse	"VERSION_CONFLICT or INVALID_STATE"
//	@Failure		422		{object}	httpx.ErrorResponse	"VALIDATION_FAILED"
//	@Router			/models/{id}/verify [post]
func (h *Handler) VerifyModel(c *gin.Context) { h.verify(c, kindModel) }

// ActivateModel godoc
//
//	@Summary		Activate a model
//	@Description	A pending, verified model of a brand that is not archived becomes ACTIVE. Otherwise 409 ACTIVATION_BLOCKED (details: verification, parent), audited as DENIED. Requires catalogue.activate and X-CSRF-Token.
//	@Tags			catalogue
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string					true	"Model ID"
//	@Param			body	body		catalogue.ActionRequest	true	"Version"
//	@Success		200		{object}	httpx.DataResponse{data=catalogue.Model}
//	@Failure		401		{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403		{object}	httpx.ErrorResponse	"FORBIDDEN or CSRF_FAILED"
//	@Failure		404		{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Failure		409		{object}	httpx.ErrorResponse	"ACTIVATION_BLOCKED, VERSION_CONFLICT or INVALID_STATE"
//	@Router			/models/{id}/activate [post]
func (h *Handler) ActivateModel(c *gin.Context) { h.activate(c, kindModel) }

// ArchiveModel godoc
//
//	@Summary		Archive a model
//	@Description	Never deletes; variants and history keep their references. Audited. Requires catalogue.archive and X-CSRF-Token.
//	@Tags			catalogue
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string						true	"Model ID"
//	@Param			body	body		catalogue.ArchiveRequest	true	"Version and reason"
//	@Success		200		{object}	httpx.DataResponse{data=catalogue.Model}
//	@Failure		401		{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403		{object}	httpx.ErrorResponse	"FORBIDDEN or CSRF_FAILED"
//	@Failure		404		{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Failure		409		{object}	httpx.ErrorResponse	"VERSION_CONFLICT or INVALID_STATE"
//	@Router			/models/{id}/archive [post]
func (h *Handler) ArchiveModel(c *gin.Context) { h.archive(c, kindModel) }

// ── Variants ─────────────────────────────────────────────────────────────────

// ListVariants godoc
//
//	@Summary		List variants
//	@Description	Sorted by name. brandId together with a modelId of another brand is a cross-brand mismatch: 422 (C-06). Requires catalogue.view.
//	@Tags			catalogue
//	@Produce		json
//	@Param			brandId		query	string	false	"Brand ID"
//	@Param			modelId		query	string	false	"Model ID"
//	@Param			q			query	string	false	"Search name, variant or manufacturer code"
//	@Param			status		query	string	false	"Lifecycle"	Enums(PENDING_VERIFICATION, ACTIVE, ARCHIVED)
//	@Param			confidence	query	string	false	"Confidence"	Enums(UNVERIFIED, SUPPLIER_CONFIRMED, CLIENT_APPROVED)
//	@Param			page		query	int		false	"Page (from 1)"
//	@Param			limit		query	int		false	"Page size (1-100, default 25)"
//	@Success		200	{object}	httpx.DataResponse{data=catalogue.VariantList}
//	@Failure		401	{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403	{object}	httpx.ErrorResponse	"FORBIDDEN"
//	@Failure		422	{object}	httpx.ErrorResponse	"VALIDATION_FAILED"
//	@Router			/variants [get]
func (h *Handler) ListVariants(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	p, err := h.svc.ListVariants(c.Request.Context(), a, listQuery(c))
	h.respond(c, http.StatusOK, VariantList{Items: p.Items, Page: p.Page, Limit: p.Limit, Total: p.Total}, err)
}

// GetVariant godoc
//
//	@Summary		Get a variant
//	@Description	Requires catalogue.view.
//	@Tags			catalogue
//	@Produce		json
//	@Param			id	path		string	true	"Variant ID"
//	@Success		200	{object}	httpx.DataResponse{data=catalogue.Variant}
//	@Failure		401	{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403	{object}	httpx.ErrorResponse	"FORBIDDEN"
//	@Failure		404	{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Router			/variants/{id} [get]
func (h *Handler) GetVariant(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	v, err := h.svc.GetVariant(c.Request.Context(), a, c.Param("id"))
	h.respond(c, http.StatusOK, v, err)
}

// CreateVariant godoc
//
//	@Summary		Create a variant
//	@Description	Adds a pending variant to a model that is not archived. The brand is taken from the model; a brandId that differs is refused (422). variantCode is unique within the model. Audited. Requires catalogue.manage and X-CSRF-Token.
//	@Tags			catalogue
//	@Accept			json
//	@Produce		json
//	@Param			body	body		catalogue.VariantCreateRequest	true	"Variant"
//	@Success		201		{object}	httpx.DataResponse{data=catalogue.Variant}
//	@Failure		401		{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403		{object}	httpx.ErrorResponse	"FORBIDDEN or CSRF_FAILED"
//	@Failure		409		{object}	httpx.ErrorResponse	"DUPLICATE or INVALID_STATE (archived model)"
//	@Failure		422		{object}	httpx.ErrorResponse	"VALIDATION_FAILED (incl. unknown modelId, brandId mismatch)"
//	@Router			/variants [post]
func (h *Handler) CreateVariant(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	var in VariantCreateRequest
	if !h.decode(c, &in) {
		return
	}
	v, err := h.svc.CreateVariant(c.Request.Context(), h.actor(c), a, in)
	h.respond(c, http.StatusCreated, v, err)
}

// UpdateVariant godoc
//
//	@Summary		Edit a pending variant
//	@Description	Only PENDING_VERIFICATION variants can be edited; model and brand never change. Editing a verified variant resets it to UNVERIFIED. Audited. Requires catalogue.manage and X-CSRF-Token.
//	@Tags			catalogue
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string							true	"Variant ID"
//	@Param			body	body		catalogue.VariantUpdateRequest	true	"Changes"
//	@Success		200		{object}	httpx.DataResponse{data=catalogue.Variant}
//	@Failure		401		{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403		{object}	httpx.ErrorResponse	"FORBIDDEN or CSRF_FAILED"
//	@Failure		404		{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Failure		409		{object}	httpx.ErrorResponse	"DUPLICATE, VERSION_CONFLICT or INVALID_STATE"
//	@Failure		422		{object}	httpx.ErrorResponse	"VALIDATION_FAILED"
//	@Router			/variants/{id} [patch]
func (h *Handler) UpdateVariant(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	var in VariantUpdateRequest
	if !h.decode(c, &in) {
		return
	}
	v, err := h.svc.UpdateVariant(c.Request.Context(), h.actor(c), a, c.Param("id"), in)
	h.respond(c, http.StatusOK, v, err)
}

// VerifyVariant godoc
//
//	@Summary		Verify a variant
//	@Description	Same as model verification. Requires catalogue.verify and X-CSRF-Token.
//	@Tags			catalogue
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string					true	"Variant ID"
//	@Param			body	body		catalogue.VerifyRequest	true	"Provenance"
//	@Success		200		{object}	httpx.DataResponse{data=catalogue.Variant}
//	@Failure		401		{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403		{object}	httpx.ErrorResponse	"FORBIDDEN or CSRF_FAILED"
//	@Failure		404		{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Failure		409		{object}	httpx.ErrorResponse	"VERSION_CONFLICT or INVALID_STATE"
//	@Failure		422		{object}	httpx.ErrorResponse	"VALIDATION_FAILED"
//	@Router			/variants/{id}/verify [post]
func (h *Handler) VerifyVariant(c *gin.Context) { h.verify(c, kindVariant) }

// ActivateVariant godoc
//
//	@Summary		Activate a variant
//	@Description	A pending, verified variant of an ACTIVE model becomes ACTIVE. Otherwise 409 ACTIVATION_BLOCKED, audited as DENIED. Requires catalogue.activate and X-CSRF-Token.
//	@Tags			catalogue
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string					true	"Variant ID"
//	@Param			body	body		catalogue.ActionRequest	true	"Version"
//	@Success		200		{object}	httpx.DataResponse{data=catalogue.Variant}
//	@Failure		401		{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403		{object}	httpx.ErrorResponse	"FORBIDDEN or CSRF_FAILED"
//	@Failure		404		{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Failure		409		{object}	httpx.ErrorResponse	"ACTIVATION_BLOCKED, VERSION_CONFLICT or INVALID_STATE"
//	@Router			/variants/{id}/activate [post]
func (h *Handler) ActivateVariant(c *gin.Context) { h.activate(c, kindVariant) }

// ArchiveVariant godoc
//
//	@Summary		Archive a variant
//	@Description	Never deletes; colours and history keep their references. Audited. Requires catalogue.archive and X-CSRF-Token.
//	@Tags			catalogue
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string						true	"Variant ID"
//	@Param			body	body		catalogue.ArchiveRequest	true	"Version and reason"
//	@Success		200		{object}	httpx.DataResponse{data=catalogue.Variant}
//	@Failure		401		{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403		{object}	httpx.ErrorResponse	"FORBIDDEN or CSRF_FAILED"
//	@Failure		404		{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Failure		409		{object}	httpx.ErrorResponse	"VERSION_CONFLICT or INVALID_STATE"
//	@Router			/variants/{id}/archive [post]
func (h *Handler) ArchiveVariant(c *gin.Context) { h.archive(c, kindVariant) }

// ── Colours ──────────────────────────────────────────────────────────────────

// ListColours godoc
//
//	@Summary		List colours
//	@Description	Sorted by name. brandId together with a variantId of another brand is a cross-brand mismatch: 422 (C-06). Requires catalogue.view.
//	@Tags			catalogue
//	@Produce		json
//	@Param			brandId		query	string	false	"Brand ID"
//	@Param			variantId	query	string	false	"Variant ID"
//	@Param			q			query	string	false	"Search name or colour code"
//	@Param			status		query	string	false	"Lifecycle"	Enums(PENDING_VERIFICATION, ACTIVE, ARCHIVED)
//	@Param			confidence	query	string	false	"Confidence"	Enums(UNVERIFIED, SUPPLIER_CONFIRMED, CLIENT_APPROVED)
//	@Param			page		query	int		false	"Page (from 1)"
//	@Param			limit		query	int		false	"Page size (1-100, default 25)"
//	@Success		200	{object}	httpx.DataResponse{data=catalogue.ColourList}
//	@Failure		401	{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403	{object}	httpx.ErrorResponse	"FORBIDDEN"
//	@Failure		422	{object}	httpx.ErrorResponse	"VALIDATION_FAILED"
//	@Router			/colours [get]
func (h *Handler) ListColours(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	p, err := h.svc.ListColours(c.Request.Context(), a, listQuery(c))
	h.respond(c, http.StatusOK, ColourList{Items: p.Items, Page: p.Page, Limit: p.Limit, Total: p.Total}, err)
}

// GetColour godoc
//
//	@Summary		Get a colour
//	@Description	Requires catalogue.view.
//	@Tags			catalogue
//	@Produce		json
//	@Param			id	path		string	true	"Colour ID"
//	@Success		200	{object}	httpx.DataResponse{data=catalogue.Colour}
//	@Failure		401	{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403	{object}	httpx.ErrorResponse	"FORBIDDEN"
//	@Failure		404	{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Router			/colours/{id} [get]
func (h *Handler) GetColour(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	col, err := h.svc.GetColour(c.Request.Context(), a, c.Param("id"))
	h.respond(c, http.StatusOK, col, err)
}

// CreateColour godoc
//
//	@Summary		Create a colour
//	@Description	Adds a pending colour to a variant that is not archived. The brand is taken from the variant; a brandId that differs is refused (422). Audited. Requires catalogue.manage and X-CSRF-Token.
//	@Tags			catalogue
//	@Accept			json
//	@Produce		json
//	@Param			body	body		catalogue.ColourCreateRequest	true	"Colour"
//	@Success		201		{object}	httpx.DataResponse{data=catalogue.Colour}
//	@Failure		401		{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403		{object}	httpx.ErrorResponse	"FORBIDDEN or CSRF_FAILED"
//	@Failure		409		{object}	httpx.ErrorResponse	"DUPLICATE or INVALID_STATE (archived variant)"
//	@Failure		422		{object}	httpx.ErrorResponse	"VALIDATION_FAILED (incl. unknown variantId, brandId mismatch)"
//	@Router			/colours [post]
func (h *Handler) CreateColour(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	var in ColourCreateRequest
	if !h.decode(c, &in) {
		return
	}
	col, err := h.svc.CreateColour(c.Request.Context(), h.actor(c), a, in)
	h.respond(c, http.StatusCreated, col, err)
}

// UpdateColour godoc
//
//	@Summary		Edit a pending colour
//	@Description	Only PENDING_VERIFICATION colours can be edited. Editing a verified colour resets it to UNVERIFIED. Audited. Requires catalogue.manage and X-CSRF-Token.
//	@Tags			catalogue
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string							true	"Colour ID"
//	@Param			body	body		catalogue.ColourUpdateRequest	true	"Changes"
//	@Success		200		{object}	httpx.DataResponse{data=catalogue.Colour}
//	@Failure		401		{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403		{object}	httpx.ErrorResponse	"FORBIDDEN or CSRF_FAILED"
//	@Failure		404		{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Failure		409		{object}	httpx.ErrorResponse	"DUPLICATE, VERSION_CONFLICT or INVALID_STATE"
//	@Failure		422		{object}	httpx.ErrorResponse	"VALIDATION_FAILED"
//	@Router			/colours/{id} [patch]
func (h *Handler) UpdateColour(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	var in ColourUpdateRequest
	if !h.decode(c, &in) {
		return
	}
	col, err := h.svc.UpdateColour(c.Request.Context(), h.actor(c), a, c.Param("id"), in)
	h.respond(c, http.StatusOK, col, err)
}

// VerifyColour godoc
//
//	@Summary		Verify a colour
//	@Description	Same as model verification. Requires catalogue.verify and X-CSRF-Token.
//	@Tags			catalogue
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string					true	"Colour ID"
//	@Param			body	body		catalogue.VerifyRequest	true	"Provenance"
//	@Success		200		{object}	httpx.DataResponse{data=catalogue.Colour}
//	@Failure		401		{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403		{object}	httpx.ErrorResponse	"FORBIDDEN or CSRF_FAILED"
//	@Failure		404		{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Failure		409		{object}	httpx.ErrorResponse	"VERSION_CONFLICT or INVALID_STATE"
//	@Failure		422		{object}	httpx.ErrorResponse	"VALIDATION_FAILED"
//	@Router			/colours/{id}/verify [post]
func (h *Handler) VerifyColour(c *gin.Context) { h.verify(c, kindColour) }

// ActivateColour godoc
//
//	@Summary		Activate a colour
//	@Description	A pending, verified colour of an ACTIVE variant becomes ACTIVE. Otherwise 409 ACTIVATION_BLOCKED, audited as DENIED. Requires catalogue.activate and X-CSRF-Token.
//	@Tags			catalogue
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string					true	"Colour ID"
//	@Param			body	body		catalogue.ActionRequest	true	"Version"
//	@Success		200		{object}	httpx.DataResponse{data=catalogue.Colour}
//	@Failure		401		{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403		{object}	httpx.ErrorResponse	"FORBIDDEN or CSRF_FAILED"
//	@Failure		404		{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Failure		409		{object}	httpx.ErrorResponse	"ACTIVATION_BLOCKED, VERSION_CONFLICT or INVALID_STATE"
//	@Router			/colours/{id}/activate [post]
func (h *Handler) ActivateColour(c *gin.Context) { h.activate(c, kindColour) }

// ArchiveColour godoc
//
//	@Summary		Archive a colour
//	@Description	Never deletes. Audited. Requires catalogue.archive and X-CSRF-Token.
//	@Tags			catalogue
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string						true	"Colour ID"
//	@Param			body	body		catalogue.ArchiveRequest	true	"Version and reason"
//	@Success		200		{object}	httpx.DataResponse{data=catalogue.Colour}
//	@Failure		401		{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403		{object}	httpx.ErrorResponse	"FORBIDDEN or CSRF_FAILED"
//	@Failure		404		{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Failure		409		{object}	httpx.ErrorResponse	"VERSION_CONFLICT or INVALID_STATE"
//	@Router			/colours/{id}/archive [post]
func (h *Handler) ArchiveColour(c *gin.Context) { h.archive(c, kindColour) }

// ── Shared lifecycle handlers ────────────────────────────────────────────────

func (h *Handler) verify(c *gin.Context, k kind) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	var in VerifyRequest
	if !h.decode(c, &in) {
		return
	}
	v, err := h.svc.Verify(c.Request.Context(), h.actor(c), a, k, c.Param("id"), in)
	h.respond(c, http.StatusOK, v, err)
}

func (h *Handler) activate(c *gin.Context, k kind) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	var in ActionRequest
	if !h.decode(c, &in) {
		return
	}
	v, err := h.svc.Activate(c.Request.Context(), h.actor(c), a, k, c.Param("id"), in)
	h.respond(c, http.StatusOK, v, err)
}

func (h *Handler) archive(c *gin.Context, k kind) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	var in ArchiveRequest
	if !h.decode(c, &in) {
		return
	}
	v, err := h.svc.Archive(c.Request.Context(), h.actor(c), a, k, c.Param("id"), in)
	h.respond(c, http.StatusOK, v, err)
}
