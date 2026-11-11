package pdi

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/audit"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/httpx"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/access"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/auth"
)

// IdempotencyHeader is required when recording an inspection.
const IdempotencyHeader = "Idempotency-Key"

// maxBody is the largest request body (100 outcomes with notes, notes, 20 IDs).
const maxBody = 128 << 10

// Handler is the HTTP adapter for PDI.
type Handler struct {
	svc    *Service
	access *access.Handler
	log    *slog.Logger
}

// NewHandler builds the adapter.
func NewHandler(svc *Service, accessH *access.Handler, log *slog.Logger) *Handler {
	return &Handler{svc: svc, access: accessH, log: log}
}

// Register mounts the routes. No route edits or deletes an inspection, an
// outcome or a checklist item, or sets a unit's status directly.
func (h *Handler) Register(rg *gin.RouterGroup) {
	view := h.access.Require(access.PermVehicleUnitView)
	manage := h.access.Require(access.PermPDIChecklistManage)
	rg.GET("/pdi-checklist-items", h.access.RequireAny(access.PermVehicleUnitView, access.PermPDIRecord,
		access.PermPDIChecklistManage), h.ListChecklist)
	rg.POST("/pdi-checklist-items", manage, h.CreateChecklistItem)
	rg.POST("/pdi-checklist-items/:id/retire", manage, h.RetireChecklistItem)
	rg.GET("/vehicle-units/:id/pdi", view, h.Overview)
	rg.POST("/vehicle-units/:id/pdi", h.access.Require(access.PermPDIRecord), h.Record)
	rg.GET("/pdi-inspections/:id", view, h.GetInspection)
}

func (h *Handler) actor(c *gin.Context) audit.Actor {
	p, _ := auth.CurrentPrincipal(c)
	return audit.UserActor(p.UserID, p.Username)
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
		httpx.Error(c, http.StatusBadRequest, httpx.CodeBadRequest, "The request body must be one JSON object (at most 128 KB).")
	}
	return false
}

func (h *Handler) respond(c *gin.Context, status int, v any, replayed bool, err error) {
	if err != nil {
		h.fail(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	if replayed {
		c.Header("Idempotent-Replayed", "true")
		status = http.StatusOK
	}
	c.JSON(status, httpx.DataResponse{Data: v})
}

func (h *Handler) fail(c *gin.Context, err error) {
	var verr *ValidationError
	var state *StateError
	switch {
	case errors.Is(err, ErrForbidden):
		access.Forbidden(c)
	case errors.Is(err, ErrNotFound):
		httpx.Error(c, http.StatusNotFound, httpx.CodeNotFound, "The record was not found.")
	case errors.As(err, &verr):
		out := make([]httpx.FieldError, len(verr.Fields))
		for i, f := range verr.Fields {
			out[i] = httpx.FieldError{Field: f.Field, Message: f.Message}
		}
		httpx.Error(c, http.StatusUnprocessableEntity, httpx.CodeValidationFailed, "Check the highlighted fields. Nothing was saved.", out...)
	case errors.Is(err, ErrVersionConflict):
		httpx.Error(c, http.StatusConflict, httpx.CodeVersionConflict,
			"This record was changed by someone else. Reload it and try again. Nothing was saved.")
	case errors.As(err, &state):
		httpx.Error(c, http.StatusConflict, httpx.CodeInvalidState, state.Message+" Nothing was saved.")
	case errors.Is(err, ErrChecklistNotConfigured):
		httpx.Error(c, http.StatusConflict, httpx.CodePDIChecklistNotConfigured,
			"No PDI checklist applies to this vehicle yet, so it cannot pass PDI. Configure the checklist first. Nothing was saved.")
	case errors.Is(err, ErrChassisRequired):
		httpx.Error(c, http.StatusConflict, httpx.CodeInvalidState,
			"This vehicle unit has no chassis/frame number, so it cannot leave PDI. Nothing was saved.")
	case errors.Is(err, ErrIdempotencyKeyReused):
		httpx.Error(c, http.StatusConflict, httpx.CodeIdempotencyKeyReused,
			"This Idempotency-Key was already used for a different request. Send a new key. Nothing was saved.")
	case errors.Is(err, ErrDuplicate):
		httpx.Error(c, http.StatusConflict, httpx.CodeDuplicate,
			"An active checklist item with this code already exists for this brand scope.",
			httpx.FieldError{Field: "code", Message: "is already used by an active item"})
	case errors.Is(err, ErrItemRetired):
		httpx.Error(c, http.StatusConflict, httpx.CodeInvalidState, "This checklist item is already retired.")
	default:
		h.log.ErrorContext(c.Request.Context(), "PDI request failed",
			slog.String("request_id", c.GetString(httpx.RequestIDKey)), slog.Any("error", err))
		httpx.InternalError(c)
	}
}

// ListChecklist godoc
//
//	@Summary		List PDI checklist items
//	@Description	The configurable PDI checklist (no item is seeded: an empty list means it is not configured). Active items first, by sortOrder then code. brandId lists the items that apply to that brand (every-brand items and the brand's own). Requires vehicle_units.view, vehicle_units.pdi.record or vehicle_units.pdi.checklist.manage.
//	@Tags			pdi
//	@Produce		json
//	@Param			status	query	string	false	"Status"	Enums(ACTIVE, RETIRED)
//	@Param			brandId	query	string	false	"Brand ID: items that apply to this brand"
//	@Param			page	query	int		false	"Page (from 1)"
//	@Param			limit	query	int		false	"Page size (1-100, default 50)"
//	@Success		200	{object}	httpx.DataResponse{data=pdi.ChecklistPage}
//	@Failure		401	{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403	{object}	httpx.ErrorResponse	"FORBIDDEN"
//	@Failure		422	{object}	httpx.ErrorResponse	"VALIDATION_FAILED (status, brandId, page, limit)"
//	@Router			/pdi-checklist-items [get]
func (h *Handler) ListChecklist(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	p, err := h.svc.ListChecklist(c.Request.Context(), a, ChecklistQuery{Status: c.Query("status"), BrandID: c.Query("brandId"),
		Page: c.Query("page"), Limit: c.Query("limit")})
	h.respond(c, http.StatusOK, p, false, err)
}

// CreateChecklistItem godoc
//
//	@Summary		Create a PDI checklist item
//	@Description	Adds an ACTIVE item for every brand (no brandId) or for one brand. code (2-40 of A-Z, 0-9, _; upper-cased) is unique among active items of the same brand scope (409 DUPLICATE). Items are never edited or deleted: retire one and create another, so recorded outcomes keep their meaning. Mandatory inspection criteria are the client's to define; nothing is pre-filled. Requires vehicle_units.pdi.checklist.manage and X-CSRF-Token. Audited (pdi_checklist_item.created).
//	@Tags			pdi
//	@Accept			json
//	@Produce		json
//	@Param			body	body		pdi.ChecklistCreateInput	true	"Checklist item"
//	@Success		201		{object}	httpx.DataResponse{data=pdi.ChecklistItem}
//	@Failure		400		{object}	httpx.ErrorResponse	"BAD_REQUEST"
//	@Failure		401		{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403		{object}	httpx.ErrorResponse	"FORBIDDEN or CSRF_FAILED"
//	@Failure		409		{object}	httpx.ErrorResponse	"DUPLICATE (code)"
//	@Failure		415		{object}	httpx.ErrorResponse	"UNSUPPORTED_MEDIA_TYPE"
//	@Failure		422		{object}	httpx.ErrorResponse	"VALIDATION_FAILED (code, label, description, brandId, sortOrder)"
//	@Router			/pdi-checklist-items [post]
func (h *Handler) CreateChecklistItem(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	var in ChecklistCreateInput
	if !h.decode(c, &in) {
		return
	}
	res, err := h.svc.CreateChecklistItem(c.Request.Context(), a, h.actor(c), c.GetString(httpx.RequestIDKey), in)
	h.respond(c, http.StatusCreated, res, false, err)
}

// RetireChecklistItem godoc
//
//	@Summary		Retire a PDI checklist item
//	@Description	ACTIVE -> RETIRED with a reason (1-500). A retired item no longer applies to new inspections; recorded outcomes keep it. version must be the item's current version. Requires vehicle_units.pdi.checklist.manage and X-CSRF-Token. Audited (pdi_checklist_item.retired).
//	@Tags			pdi
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string						true	"Checklist item ID"
//	@Param			body	body		pdi.ChecklistRetireInput	true	"Version and reason"
//	@Success		200		{object}	httpx.DataResponse{data=pdi.ChecklistItem}
//	@Failure		400		{object}	httpx.ErrorResponse	"BAD_REQUEST"
//	@Failure		401		{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403		{object}	httpx.ErrorResponse	"FORBIDDEN or CSRF_FAILED"
//	@Failure		404		{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Failure		409		{object}	httpx.ErrorResponse	"VERSION_CONFLICT or INVALID_STATE (already retired)"
//	@Failure		415		{object}	httpx.ErrorResponse	"UNSUPPORTED_MEDIA_TYPE"
//	@Failure		422		{object}	httpx.ErrorResponse	"VALIDATION_FAILED (version, reason)"
//	@Router			/pdi-checklist-items/{id}/retire [post]
func (h *Handler) RetireChecklistItem(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	var in ChecklistRetireInput
	if !h.decode(c, &in) {
		return
	}
	res, err := h.svc.RetireChecklistItem(c.Request.Context(), a, h.actor(c), c.GetString(httpx.RequestIDKey), c.Param("id"), in)
	h.respond(c, http.StatusOK, res, false, err)
}

// Overview godoc
//
//	@Summary		Get a vehicle unit's PDI
//	@Description	The unit as it is now (status, version, saleable), the dispositions its status admits (PASS and HOLD while RECEIVED_PENDING_PDI; REJECT while QUALITY_HOLD; none otherwise — OQ-31), canRecord (the caller holds vehicle_units.pdi.record), the active checklist that applies to the unit (empty = not configured: PASS is refused) and its inspections, newest first. A unit at a location outside the caller's scope is 404. Requires vehicle_units.view.
//	@Tags			pdi
//	@Produce		json
//	@Param			id	path		string	true	"Vehicle unit ID"
//	@Success		200	{object}	httpx.DataResponse{data=pdi.Overview}
//	@Failure		401	{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403	{object}	httpx.ErrorResponse	"FORBIDDEN"
//	@Failure		404	{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Router			/vehicle-units/{id}/pdi [get]
func (h *Handler) Overview(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	res, err := h.svc.Overview(c.Request.Context(), a, c.Param("id"))
	h.respond(c, http.StatusOK, res, false, err)
}

// Record godoc
//
//	@Summary		Record a PDI inspection
//	@Description	Records one complete inspection and applies its disposition in one transaction: the stock movement (PDI_PASS, QUALITY_HOLD or REJECT; source PDI_INSPECTION with the inspection number), the unit's new status and version, the inspection with its outcomes, the evidence links and the audit events (pdi_inspection.recorded, vehicle_unit.moved) commit together or not at all. The actor is the session user and the time the database clock. Rules: version must be the unit's current version (409 VERSION_CONFLICT). PASS: RECEIVED_PENDING_PDI -> AVAILABLE; needs at least one applicable checklist item (409 PDI_CHECKLIST_NOT_CONFIGURED) and every outcome PASS. HOLD: RECEIVED_PENDING_PDI -> QUALITY_HOLD. REJECT: QUALITY_HOLD -> REJECTED only. A direct reject of a unit awaiting PDI and any release from hold are refused (409 INVALID_STATE, OQ-31). items must hold exactly one outcome for every active checklist item that applies to the unit (none missing, none unknown/retired/other brand); a FAIL needs a note. HOLD and REJECT need notes and at least one evidence attachment (INV-008). attachmentIds must be the caller's own unlinked uploads (POST /attachments) at the unit's location; they become private PDI evidence (attachments.view + vehicle_units.view to open). A unit without a chassis/frame number cannot leave RECEIVED_PENDING_PDI (409 INVALID_STATE, INV-002). Idempotency-Key is required: the same key and request by the same user returns the first inspection with 200 and Idempotent-Replayed: true (nothing written again; concurrent repeats wait and replay); the same key with another request is 409 IDEMPOTENCY_KEY_REUSED. Refusals for the unit's status, a missing checklist or a missing chassis number are audited (DENIED). Unknown fields (status, actorId, inspectedAt, result, ...) are 422. A unit outside the caller's scope is 404. Requires vehicle_units.pdi.record and X-CSRF-Token.
//	@Tags			pdi
//	@Accept			json
//	@Produce		json
//	@Param			id				path		string					true	"Vehicle unit ID"
//	@Param			Idempotency-Key	header		string					true	"8-100 letters, digits, - or _"
//	@Param			body			body		pdi.RecordInput			true	"Inspection"
//	@Success		201				{object}	httpx.DataResponse{data=pdi.RecordResult}	"Recorded"
//	@Success		200				{object}	httpx.DataResponse{data=pdi.RecordResult}	"Replay (Idempotent-Replayed: true)"
//	@Failure		400				{object}	httpx.ErrorResponse	"BAD_REQUEST"
//	@Failure		401				{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403				{object}	httpx.ErrorResponse	"FORBIDDEN or CSRF_FAILED"
//	@Failure		404				{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Failure		409				{object}	httpx.ErrorResponse	"VERSION_CONFLICT, INVALID_STATE, PDI_CHECKLIST_NOT_CONFIGURED or IDEMPOTENCY_KEY_REUSED"
//	@Failure		415				{object}	httpx.ErrorResponse	"UNSUPPORTED_MEDIA_TYPE"
//	@Failure		422				{object}	httpx.ErrorResponse	"VALIDATION_FAILED (version, disposition, items, items[i].checklistItemId, items[i].outcome, items[i].note, notes, attachmentIds, attachmentIds[i], Idempotency-Key)"
//	@Router			/vehicle-units/{id}/pdi [post]
func (h *Handler) Record(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	var in RecordInput
	if !h.decode(c, &in) {
		return
	}
	res, replayed, err := h.svc.Record(c.Request.Context(), a, h.actor(c), c.GetString(httpx.RequestIDKey), c.Param("id"), in,
		c.GetHeader(IdempotencyHeader))
	h.respond(c, http.StatusCreated, res, replayed, err)
}

// GetInspection godoc
//
//	@Summary		Get a PDI inspection
//	@Description	One inspection with its outcomes (checklist code and label, PASS/FAIL, note) and evidence (attachment ID, name, type, size; download through the attachments API). evidence is empty and evidenceHidden true without attachments.view. An inspection of a unit outside the caller's scope is 404. Requires vehicle_units.view.
//	@Tags			pdi
//	@Produce		json
//	@Param			id	path		string	true	"Inspection ID"
//	@Success		200	{object}	httpx.DataResponse{data=pdi.InspectionDetail}
//	@Failure		401	{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403	{object}	httpx.ErrorResponse	"FORBIDDEN"
//	@Failure		404	{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Router			/pdi-inspections/{id} [get]
func (h *Handler) GetInspection(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	res, err := h.svc.Inspection(c.Request.Context(), a, c.Param("id"))
	h.respond(c, http.StatusOK, res, false, err)
}
