package suppliers

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

// Handler is the HTTP adapter for the supplier master.
type Handler struct {
	svc    *Service
	access *access.Handler
	log    *slog.Logger
}

// NewHandler builds the adapter.
func NewHandler(svc *Service, accessH *access.Handler, log *slog.Logger) *Handler {
	return &Handler{svc: svc, access: accessH, log: log}
}

// Register mounts the routes. There is no DELETE: suppliers are archived,
// contacts removed (deactivated) and relationships ended.
func (h *Handler) Register(rg *gin.RouterGroup) {
	view := h.access.Require(access.PermSupplierView)
	manage := h.access.Require(access.PermSupplierManage)
	verify := h.access.Require(access.PermSupplierVerify)
	archive := h.access.Require(access.PermSupplierArchive)

	rg.GET("/suppliers", view, h.List)
	rg.POST("/suppliers", manage, h.Create)
	rg.GET("/suppliers/:id", view, h.Get)
	rg.PATCH("/suppliers/:id", manage, h.Update)
	rg.POST("/suppliers/:id/verify", verify, h.Verify)
	rg.POST("/suppliers/:id/archive", archive, h.Archive)
	rg.POST("/suppliers/:id/contacts", manage, h.AddContact)
	rg.PATCH("/suppliers/:id/contacts/:contactId", manage, h.UpdateContact)
	rg.POST("/suppliers/:id/contacts/:contactId/remove", manage, h.RemoveContact)
	rg.POST("/suppliers/:id/documents", manage, h.AddDocument)
	rg.POST("/suppliers/:id/brand-links", manage, h.CreateLink)
	rg.GET("/supplier-brand-links", view, h.ListLinks)
	rg.POST("/supplier-brand-links/:id/verify", verify, h.VerifyLink)
	rg.POST("/supplier-brand-links/:id/end", archive, h.EndLink)
}

// SupplierPage is a page of suppliers.
type SupplierPage struct {
	Items []Supplier `json:"items"`
	Page  int        `json:"page" example:"1"`
	Limit int        `json:"limit" example:"25"`
	Total int        `json:"total" example:"0"`
} //@name SupplierPage

func (h *Handler) actor(c *gin.Context) Actor {
	p, _ := auth.CurrentPrincipal(c)
	return Actor{UserID: p.UserID, Username: p.Username, RequestID: c.GetString(httpx.RequestIDKey)}
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
		verr  *ValidationError
		dup   *DuplicateError
		state *StateError
	)
	switch {
	case errors.Is(err, ErrForbidden):
		access.Forbidden(c)
	case errors.Is(err, ErrNotFound):
		httpx.Error(c, http.StatusNotFound, httpx.CodeNotFound, "The supplier record was not found.")
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
		h.log.ErrorContext(c.Request.Context(), "supplier request failed",
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
//	@Summary		List suppliers
//	@Description	Search legal name, trade name or GSTIN; filter by status or by brand (suppliers with an open relationship to it). Requires suppliers.view.
//	@Tags			suppliers
//	@Produce		json
//	@Param			q		query	string	false	"Search"
//	@Param			status	query	string	false	"Status"	Enums(PENDING_VERIFICATION, VERIFIED, ARCHIVED)
//	@Param			brandId	query	string	false	"Brand ID"
//	@Param			page	query	int		false	"Page (from 1)"
//	@Param			limit	query	int		false	"Page size (1-100, default 25)"
//	@Success		200	{object}	httpx.DataResponse{data=suppliers.SupplierPage}
//	@Failure		401	{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403	{object}	httpx.ErrorResponse	"FORBIDDEN"
//	@Failure		422	{object}	httpx.ErrorResponse	"VALIDATION_FAILED"
//	@Router			/suppliers [get]
func (h *Handler) List(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	p, err := h.svc.ListSuppliers(c.Request.Context(), a, ListQuery{Query: c.Query("q"), Status: c.Query("status"),
		BrandID: c.Query("brandId"), Page: c.Query("page"), Limit: c.Query("limit")})
	h.respond(c, http.StatusOK, SupplierPage{Items: p.Items, Page: p.Page, Limit: p.Limit, Total: p.Total}, err)
}

// Get godoc
//
//	@Summary		Get a supplier
//	@Description	The supplier with contacts (including removed ones, marked inactive), documents and brand relationships. Requires suppliers.view.
//	@Tags			suppliers
//	@Produce		json
//	@Param			id	path		string	true	"Supplier ID"
//	@Success		200	{object}	httpx.DataResponse{data=suppliers.Detail}
//	@Failure		401	{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403	{object}	httpx.ErrorResponse	"FORBIDDEN"
//	@Failure		404	{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Router			/suppliers/{id} [get]
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
//	@Summary		Create a supplier
//	@Description	Starts PENDING_VERIFICATION. A supplier record is not proof of dealership authorisation. Legal name and GSTIN are unique. Requires suppliers.manage and X-CSRF-Token. Audited.
//	@Tags			suppliers
//	@Accept			json
//	@Produce		json
//	@Param			body	body		suppliers.SupplierInput	true	"Supplier"
//	@Success		201		{object}	httpx.DataResponse{data=suppliers.Supplier}
//	@Failure		401		{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403		{object}	httpx.ErrorResponse	"FORBIDDEN or CSRF_FAILED"
//	@Failure		409		{object}	httpx.ErrorResponse	"DUPLICATE (legalName, gstin)"
//	@Failure		422		{object}	httpx.ErrorResponse	"VALIDATION_FAILED"
//	@Router			/suppliers [post]
func (h *Handler) Create(c *gin.Context) {
	with(h, c, http.StatusCreated, func(a access.Access, in SupplierInput) (any, error) {
		return h.svc.CreateSupplier(c.Request.Context(), h.actor(c), a, in)
	})
}

// Update godoc
//
//	@Summary		Edit a supplier
//	@Description	Changing the legal name or GSTIN of a verified supplier returns it to PENDING_VERIFICATION. Archived suppliers cannot change. Requires suppliers.manage and X-CSRF-Token. Audited.
//	@Tags			suppliers
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string					true	"Supplier ID"
//	@Param			body	body		suppliers.SupplierPatch	true	"Changes"
//	@Success		200		{object}	httpx.DataResponse{data=suppliers.Supplier}
//	@Failure		401		{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403		{object}	httpx.ErrorResponse	"FORBIDDEN or CSRF_FAILED"
//	@Failure		404		{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Failure		409		{object}	httpx.ErrorResponse	"DUPLICATE, VERSION_CONFLICT or INVALID_STATE"
//	@Failure		422		{object}	httpx.ErrorResponse	"VALIDATION_FAILED"
//	@Router			/suppliers/{id} [patch]
func (h *Handler) Update(c *gin.Context) {
	with(h, c, http.StatusOK, func(a access.Access, in SupplierPatch) (any, error) {
		return h.svc.UpdateSupplier(c.Request.Context(), h.actor(c), a, c.Param("id"), in)
	})
}

// Verify godoc
//
//	@Summary		Verify a supplier
//	@Description	Records a reference and one of the supplier's own documents; the caller is the verifier. Requires suppliers.verify and X-CSRF-Token. Audited.
//	@Tags			suppliers
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string					true	"Supplier ID"
//	@Param			body	body		suppliers.VerifyInput	true	"Evidence"
//	@Success		200		{object}	httpx.DataResponse{data=suppliers.Supplier}
//	@Failure		401		{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403		{object}	httpx.ErrorResponse	"FORBIDDEN or CSRF_FAILED"
//	@Failure		404		{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Failure		409		{object}	httpx.ErrorResponse	"VERSION_CONFLICT or INVALID_STATE"
//	@Failure		422		{object}	httpx.ErrorResponse	"VALIDATION_FAILED (document not of this supplier)"
//	@Router			/suppliers/{id}/verify [post]
func (h *Handler) Verify(c *gin.Context) {
	with(h, c, http.StatusOK, func(a access.Access, in VerifyInput) (any, error) {
		return h.svc.VerifySupplier(c.Request.Context(), h.actor(c), a, c.Param("id"), in)
	})
}

// Archive godoc
//
//	@Summary		Archive a supplier
//	@Description	Nothing is deleted; contacts, documents and relationships stay as history. Requires suppliers.archive and X-CSRF-Token. Audited.
//	@Tags			suppliers
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string					true	"Supplier ID"
//	@Param			body	body		suppliers.ArchiveInput	true	"Version and reason"
//	@Success		200		{object}	httpx.DataResponse{data=suppliers.Supplier}
//	@Failure		401		{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403		{object}	httpx.ErrorResponse	"FORBIDDEN or CSRF_FAILED"
//	@Failure		404		{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Failure		409		{object}	httpx.ErrorResponse	"VERSION_CONFLICT or INVALID_STATE"
//	@Router			/suppliers/{id}/archive [post]
func (h *Handler) Archive(c *gin.Context) {
	with(h, c, http.StatusOK, func(a access.Access, in ArchiveInput) (any, error) {
		return h.svc.ArchiveSupplier(c.Request.Context(), h.actor(c), a, c.Param("id"), in)
	})
}

// AddContact godoc
//
//	@Summary		Add a supplier contact
//	@Description	Business contact; phone or email required. A new primary contact replaces the previous primary. Audited without the phone/email. Requires suppliers.manage and X-CSRF-Token.
//	@Tags			suppliers
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string					true	"Supplier ID"
//	@Param			body	body		suppliers.ContactInput	true	"Contact"
//	@Success		201		{object}	httpx.DataResponse{data=suppliers.Contact}
//	@Failure		401		{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403		{object}	httpx.ErrorResponse	"FORBIDDEN or CSRF_FAILED"
//	@Failure		404		{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Failure		409		{object}	httpx.ErrorResponse	"INVALID_STATE (archived supplier)"
//	@Failure		422		{object}	httpx.ErrorResponse	"VALIDATION_FAILED"
//	@Router			/suppliers/{id}/contacts [post]
func (h *Handler) AddContact(c *gin.Context) {
	with(h, c, http.StatusCreated, func(a access.Access, in ContactInput) (any, error) {
		return h.svc.AddContact(c.Request.Context(), h.actor(c), a, c.Param("id"), in)
	})
}

// UpdateContact godoc
//
//	@Summary		Edit a supplier contact
//	@Description	Requires suppliers.manage and X-CSRF-Token. Audited.
//	@Tags			suppliers
//	@Accept			json
//	@Produce		json
//	@Param			id			path		string					true	"Supplier ID"
//	@Param			contactId	path		string					true	"Contact ID"
//	@Param			body		body		suppliers.ContactPatch	true	"Changes"
//	@Success		200			{object}	httpx.DataResponse{data=suppliers.Contact}
//	@Failure		401			{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403			{object}	httpx.ErrorResponse	"FORBIDDEN or CSRF_FAILED"
//	@Failure		404			{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Failure		409			{object}	httpx.ErrorResponse	"VERSION_CONFLICT or INVALID_STATE"
//	@Failure		422			{object}	httpx.ErrorResponse	"VALIDATION_FAILED"
//	@Router			/suppliers/{id}/contacts/{contactId} [patch]
func (h *Handler) UpdateContact(c *gin.Context) {
	with(h, c, http.StatusOK, func(a access.Access, in ContactPatch) (any, error) {
		return h.svc.UpdateContact(c.Request.Context(), h.actor(c), a, c.Param("id"), c.Param("contactId"), in)
	})
}

// RemoveContact godoc
//
//	@Summary		Remove a supplier contact
//	@Description	Deactivates the contact; it stays as history. Requires suppliers.manage and X-CSRF-Token. Audited.
//	@Tags			suppliers
//	@Accept			json
//	@Produce		json
//	@Param			id			path		string					true	"Supplier ID"
//	@Param			contactId	path		string					true	"Contact ID"
//	@Param			body		body		suppliers.VersionInput	true	"Version"
//	@Success		200			{object}	httpx.DataResponse{data=suppliers.Contact}
//	@Failure		401			{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403			{object}	httpx.ErrorResponse	"FORBIDDEN or CSRF_FAILED"
//	@Failure		404			{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Failure		409			{object}	httpx.ErrorResponse	"VERSION_CONFLICT or INVALID_STATE"
//	@Router			/suppliers/{id}/contacts/{contactId}/remove [post]
func (h *Handler) RemoveContact(c *gin.Context) {
	with(h, c, http.StatusOK, func(a access.Access, in VersionInput) (any, error) {
		return h.svc.RemoveContact(c.Request.Context(), h.actor(c), a, c.Param("id"), c.Param("contactId"), in)
	})
}

// AddDocument godoc
//
//	@Summary		Attach a document to a supplier
//	@Description	Links one of the caller's own unlinked private uploads (POST /attachments) to the supplier. From then on anyone holding suppliers.view and attachments.view may open it through the attachments API; nobody else can. Requires suppliers.manage, attachments.upload and X-CSRF-Token. Audited.
//	@Tags			suppliers
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string					true	"Supplier ID"
//	@Param			body	body		suppliers.DocumentInput	true	"Document"
//	@Success		201		{object}	httpx.DataResponse{data=suppliers.Document}
//	@Failure		401		{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403		{object}	httpx.ErrorResponse	"FORBIDDEN or CSRF_FAILED"
//	@Failure		404		{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Failure		409		{object}	httpx.ErrorResponse	"INVALID_STATE or DUPLICATE"
//	@Failure		422		{object}	httpx.ErrorResponse	"VALIDATION_FAILED (not your unlinked upload)"
//	@Router			/suppliers/{id}/documents [post]
func (h *Handler) AddDocument(c *gin.Context) {
	with(h, c, http.StatusCreated, func(a access.Access, in DocumentInput) (any, error) {
		return h.svc.AddDocument(c.Request.Context(), h.actor(c), a, c.Param("id"), in)
	})
}

// CreateLink godoc
//
//	@Summary		Propose a supplier-brand relationship
//	@Description	Starts UNVERIFIED and proves nothing until verified with evidence. One open relationship per supplier and brand; a supplier may serve several brands. Requires suppliers.manage and X-CSRF-Token. Audited.
//	@Tags			suppliers
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string				true	"Supplier ID"
//	@Param			body	body		suppliers.LinkInput	true	"Brand"
//	@Success		201		{object}	httpx.DataResponse{data=suppliers.BrandLink}
//	@Failure		401		{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403		{object}	httpx.ErrorResponse	"FORBIDDEN or CSRF_FAILED"
//	@Failure		404		{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Failure		409		{object}	httpx.ErrorResponse	"DUPLICATE (brandId) or INVALID_STATE"
//	@Failure		422		{object}	httpx.ErrorResponse	"VALIDATION_FAILED"
//	@Router			/suppliers/{id}/brand-links [post]
func (h *Handler) CreateLink(c *gin.Context) {
	with(h, c, http.StatusCreated, func(a access.Access, in LinkInput) (any, error) {
		return h.svc.CreateLink(c.Request.Context(), h.actor(c), a, c.Param("id"), in)
	})
}

// ListLinks godoc
//
//	@Summary		Supplier-brand relationships
//	@Description	By brand or by supplier (one is required); up to 200, open ones first. current=true means verified, supplier verified and today within the effective dates. Requires suppliers.view.
//	@Tags			suppliers
//	@Produce		json
//	@Param			brandId		query	string	false	"Brand ID"
//	@Param			supplierId	query	string	false	"Supplier ID"
//	@Param			status		query	string	false	"Status"	Enums(UNVERIFIED, VERIFIED, ENDED)
//	@Success		200	{object}	httpx.DataResponse{data=[]suppliers.BrandLink}
//	@Failure		401	{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403	{object}	httpx.ErrorResponse	"FORBIDDEN"
//	@Failure		422	{object}	httpx.ErrorResponse	"VALIDATION_FAILED"
//	@Router			/supplier-brand-links [get]
func (h *Handler) ListLinks(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	links, err := h.svc.ListLinks(c.Request.Context(), a, LinkQuery{BrandID: c.Query("brandId"), SupplierID: c.Query("supplierId"), Status: c.Query("status")})
	h.respond(c, http.StatusOK, links, err)
}

// VerifyLink godoc
//
//	@Summary		Verify a supplier-brand relationship
//	@Description	Needs the supplier verified first, an authorisation reference, an evidence document of the same supplier and the start date; the caller is the verifier. Requires suppliers.verify and X-CSRF-Token. Audited.
//	@Tags			suppliers
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string						true	"Relationship ID"
//	@Param			body	body		suppliers.LinkVerifyInput	true	"Evidence"
//	@Success		200		{object}	httpx.DataResponse{data=suppliers.BrandLink}
//	@Failure		401		{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403		{object}	httpx.ErrorResponse	"FORBIDDEN or CSRF_FAILED"
//	@Failure		404		{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Failure		409		{object}	httpx.ErrorResponse	"VERSION_CONFLICT or INVALID_STATE (supplier not verified, brand archived)"
//	@Failure		422		{object}	httpx.ErrorResponse	"VALIDATION_FAILED (evidence not of this supplier)"
//	@Router			/supplier-brand-links/{id}/verify [post]
func (h *Handler) VerifyLink(c *gin.Context) {
	with(h, c, http.StatusOK, func(a access.Access, in LinkVerifyInput) (any, error) {
		return h.svc.VerifyLink(c.Request.Context(), h.actor(c), a, c.Param("id"), in)
	})
}

// EndLink godoc
//
//	@Summary		End a supplier-brand relationship
//	@Description	Sets ENDED with an end date (default today) and reason; kept as history. Requires suppliers.archive and X-CSRF-Token. Audited.
//	@Tags			suppliers
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string					true	"Relationship ID"
//	@Param			body	body		suppliers.LinkEndInput	true	"Version, reason, end date"
//	@Success		200		{object}	httpx.DataResponse{data=suppliers.BrandLink}
//	@Failure		401		{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403		{object}	httpx.ErrorResponse	"FORBIDDEN or CSRF_FAILED"
//	@Failure		404		{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Failure		409		{object}	httpx.ErrorResponse	"VERSION_CONFLICT or INVALID_STATE"
//	@Failure		422		{object}	httpx.ErrorResponse	"VALIDATION_FAILED"
//	@Router			/supplier-brand-links/{id}/end [post]
func (h *Handler) EndLink(c *gin.Context) {
	with(h, c, http.StatusOK, func(a access.Access, in LinkEndInput) (any, error) {
		return h.svc.EndLink(c.Request.Context(), h.actor(c), a, c.Param("id"), in)
	})
}
