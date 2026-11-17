package openingstock

import (
	"errors"
	"log/slog"
	"mime"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/audit"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/httpx"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/access"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/auth"
)

// IdempotencyHeader is required on uploads.
const IdempotencyHeader = "Idempotency-Key"

// Handler is the HTTP adapter.
type Handler struct {
	svc    *Service
	access *access.Handler
	log    *slog.Logger
}

// NewHandler builds the adapter.
func NewHandler(svc *Service, accessH *access.Handler, log *slog.Logger) *Handler {
	return &Handler{svc: svc, access: accessH, log: log}
}

// Register mounts the routes. Batches are never edited or deleted; approval and
// commit (G-07) each happen once per batch.
func (h *Handler) Register(rg *gin.RouterGroup) {
	perm := h.access.Require(access.PermImportPreview)
	view := h.access.RequireAny(access.PermImportPreview, access.PermImportApprove, access.PermImportCommit)
	rg.GET("/opening-stock-imports/specification", perm, h.Spec)
	rg.GET("/opening-stock-imports/template", perm, h.Template)
	rg.POST("/opening-stock-imports", perm, h.Upload)
	rg.GET("/opening-stock-imports", view, h.List)
	rg.GET("/opening-stock-imports/:id", view, h.Get)
	rg.GET("/opening-stock-imports/:id/rows", view, h.Rows)
	rg.POST("/opening-stock-imports/:id/approval", h.access.Require(access.PermImportApprove), h.Approve)
	rg.POST("/opening-stock-imports/:id/commit", h.access.Require(access.PermImportCommit), h.Commit)
	rg.GET("/opening-stock-imports/:id/commit", view, h.GetCommit)
}

func (h *Handler) respond(c *gin.Context, status int, v any, err error) {
	var verr *ValidationError
	switch {
	case err == nil:
		c.Header("Cache-Control", "no-store")
		c.JSON(status, httpx.DataResponse{Data: v})
	case errors.Is(err, ErrForbidden):
		access.Forbidden(c)
	case errors.Is(err, ErrNotFound):
		httpx.Error(c, http.StatusNotFound, httpx.CodeNotFound, "The import was not found.")
	case errors.As(err, &verr):
		out := make([]httpx.FieldError, len(verr.Fields))
		for i, f := range verr.Fields {
			out[i] = httpx.FieldError{Field: f.Field, Message: f.Message}
		}
		httpx.Error(c, http.StatusUnprocessableEntity, httpx.CodeValidationFailed, "Check the highlighted fields.", out...)
	case errors.Is(err, ErrIdempotencyKey):
		httpx.Error(c, http.StatusUnprocessableEntity, httpx.CodeValidationFailed, "Check the highlighted fields.",
			httpx.FieldError{Field: IdempotencyHeader, Message: "must be 8-100 letters, digits, '-' or '_'"})
	case errors.Is(err, ErrFileFieldMissing):
		httpx.Error(c, http.StatusUnprocessableEntity, httpx.CodeValidationFailed, "Check the highlighted fields.",
			httpx.FieldError{Field: "file", Message: "is required (multipart/form-data field \"file\")"})
	case errors.Is(err, ErrEmptyFile):
		httpx.Error(c, http.StatusUnprocessableEntity, httpx.CodeValidationFailed, "The file is empty. Nothing was stored.",
			httpx.FieldError{Field: "file", Message: "is empty"})
	case errors.Is(err, ErrTooLarge):
		httpx.Error(c, http.StatusRequestEntityTooLarge, httpx.CodeFileTooLarge,
			"The file is larger than "+strconv.FormatInt(h.svc.limits.MaxBytes>>20, 10)+" MB. Split it into smaller files. Nothing was stored.")
	case errors.Is(err, ErrUnsupportedType):
		httpx.Error(c, http.StatusUnsupportedMediaType, httpx.CodeUnsupportedMediaType,
			"Only the CSV template (.csv, UTF-8 text) is accepted. In Excel use Save As → \"CSV UTF-8 (comma delimited)\". Nothing was stored.")
	case errors.Is(err, ErrKeyReused):
		httpx.Error(c, http.StatusConflict, httpx.CodeIdempotencyKeyReused,
			"This Idempotency-Key was already used for a different request. Send a new key. Nothing was stored.")
	default:
		h.respondCommit(c, err)
	}
}

// respondCommit maps the G-07 errors; anything else is a logged 500.
func (h *Handler) respondCommit(c *gin.Context, err error) {
	var already *AlreadyCommittedError
	var blocked *CommitBlockedError
	var dup *DuplicateAtCommitError
	switch {
	case errors.As(err, &already):
		msg := "This batch was already committed. Nothing new was created."
		if already.Commit.Reference != "" {
			msg = "This batch was already committed as " + already.Commit.Reference + " (" +
				strconv.Itoa(already.Commit.UnitsCreated) + " units). Nothing new was created."
		}
		httpx.Error(c, http.StatusConflict, httpx.CodeImportAlreadyCommitted, msg)
	case errors.As(err, &blocked):
		var details []httpx.FieldError
		for _, r := range blocked.Rows {
			for _, is := range r.Issues {
				field := "rows[" + strconv.Itoa(r.RowNumber) + "]"
				if is.Field != "" {
					field += "." + is.Field
				}
				details = append(details, httpx.FieldError{Field: field, Message: is.Code + " (" + is.Severity + "): " + is.Message})
			}
		}
		httpx.Error(c, http.StatusUnprocessableEntity, httpx.CodeImportRowsBlocked,
			strconv.Itoa(len(blocked.Rows))+" line(s) cannot be committed. Nothing was created; the batch is unchanged. "+
				"Correct the file, upload it again and have the new preview approved.", details...)
	case errors.As(err, &dup):
		msg := "A chassis/frame number in this batch was registered by another transaction meanwhile. Nothing was created."
		if dup.RowNumber > 0 {
			msg = "The chassis/frame number on line " + strconv.Itoa(dup.RowNumber) +
				" was registered by another transaction meanwhile. Nothing was created."
		}
		httpx.Error(c, http.StatusConflict, httpx.CodeDuplicateIdentifier, msg)
	case errors.Is(err, ErrApprovalRequired):
		httpx.Error(c, http.StatusConflict, httpx.CodeImportApprovalRequired,
			"This batch has no recorded approval: it must be reconciled to a signed physical count and approved before commit. Nothing was created.")
	case errors.Is(err, ErrAlreadyApproved):
		httpx.Error(c, http.StatusConflict, httpx.CodeImportAlreadyApproved,
			"This batch already has a different approval. An approval is recorded once and never changed.")
	case errors.Is(err, ErrApprovalBlocked):
		httpx.Error(c, http.StatusConflict, httpx.CodeInvalidState,
			"Only a preview whose every line is VALID can be approved. Correct the file and upload it again.")
	case errors.Is(err, ErrNotCommittable):
		httpx.Error(c, http.StatusConflict, httpx.CodeInvalidState,
			"Only a previewed batch in the current template can be approved or committed (a rejected file cannot).")
	case errors.Is(err, ErrPreviewExpired):
		httpx.Error(c, http.StatusConflict, httpx.CodeImportPreviewExpired,
			"This preview has expired and is history only. Upload the file again for a new preview. Nothing was created.")
	case errors.Is(err, ErrStatusesNotApproved):
		httpx.Error(c, http.StatusConflict, httpx.CodeImportStatusesNotApproved,
			"No stock status is approved for opening stock yet (OQ-47), so nothing can be committed. Nothing was created.")
	case errors.Is(err, ErrSourceInvalid):
		httpx.Error(c, http.StatusConflict, httpx.CodeImportSourceInvalid,
			"The stored file of this batch is missing or no longer matches its fingerprint and preview. Nothing was created.")
	default:
		h.log.ErrorContext(c.Request.Context(), "opening stock import request failed",
			slog.String("request_id", c.GetString(httpx.RequestIDKey)), slog.Any("error", err))
		httpx.InternalError(c)
	}
}

// Spec godoc
//
//	@Summary		Opening stock template specification
//	@Description	The template version, accepted format (CSV UTF-8 only; XLSX is refused), size/row limits, preview lifetime, every column (name, required, max length, format, meaning) and the accepted stock statuses, plus the business date dates are checked against. Requires stock.import.preview.
//	@Tags			opening-stock-imports
//	@Produce		json
//	@Success		200	{object}	httpx.DataResponse{data=openingstock.Specification}
//	@Failure		401	{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403	{object}	httpx.ErrorResponse	"FORBIDDEN"
//	@Router			/opening-stock-imports/specification [get]
func (h *Handler) Spec(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	spec, err := h.svc.Spec(a)
	h.respond(c, http.StatusOK, spec, err)
}

// Template godoc
//
//	@Summary		Download the opening stock template
//	@Description	A CSV file with the template marker line (#template:OS-IMPORT-1) and the column header line, and no data rows. Requires stock.import.preview.
//	@Tags			opening-stock-imports
//	@Produce		text/csv
//	@Success		200	{file}		file
//	@Failure		401	{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403	{object}	httpx.ErrorResponse	"FORBIDDEN"
//	@Router			/opening-stock-imports/template [get]
func (h *Handler) Template(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	data, name, err := h.svc.Template(a)
	if err != nil {
		h.respond(c, 0, nil, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": name}))
	c.Data(http.StatusOK, "text/csv; charset=utf-8", data)
}

// Upload godoc
//
//	@Summary		Upload an opening stock file for preview
//	@Description	Stores the CSV privately and validates it: the file (UTF-8 text, .csv, template marker and version, header with every required column, no duplicate or unknown column, at least one and at most maxRows data rows) and every row (required values, lengths, catalogue brand → model → variant → colour resolved exactly — never guessed —, brand/model/variant/colour verification and activation, location known, active and in the caller's scope, stock status, received date not after the business date, cost format, supplier, chassis/frame duplicates within the file, against vehicle units and against open receipt drafts, and motor/battery/charger possible duplicates as warnings while their uniqueness is unapproved). Row status: ERROR (cannot be imported as is), BLOCKED_UNVERIFIED_MAPPING (maps to a catalogue entry that is not verified/active), WARNING (importable, review), VALID. A file that cannot be read as the template is stored as a REJECTED batch with fileIssues and no rows. THE PREVIEW CREATES NO STOCK: no vehicle unit, receipt, movement, balance or catalogue change; only the batch, its rows, the file's private attachment and audit events are written. Idempotency-Key is required: the same key with the same file returns the first batch (200, Idempotent-Replayed: true); with another file 409. Requires stock.import.preview and X-CSRF-Token.
//	@Tags			opening-stock-imports
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			X-CSRF-Token	header		string	true	"CSRF token from the session"
//	@Param			Idempotency-Key	header		string	true	"8-100 letters, digits, - or _"
//	@Param			file			formData	file	true	"The filled template (.csv)"
//	@Success		201				{object}	httpx.DataResponse{data=openingstock.Batch}
//	@Success		200				{object}	httpx.DataResponse{data=openingstock.Batch}	"Replay of the same request"
//	@Failure		401				{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403				{object}	httpx.ErrorResponse	"FORBIDDEN or CSRF_FAILED"
//	@Failure		409				{object}	httpx.ErrorResponse	"IDEMPOTENCY_KEY_REUSED"
//	@Failure		413				{object}	httpx.ErrorResponse	"FILE_TOO_LARGE"
//	@Failure		415				{object}	httpx.ErrorResponse	"UNSUPPORTED_MEDIA_TYPE"
//	@Failure		422				{object}	httpx.ErrorResponse	"VALIDATION_FAILED (file, Idempotency-Key)"
//	@Router			/opening-stock-imports [post]
func (h *Handler) Upload(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, h.svc.limits.MaxBytes+1<<20)
	fh, err := c.FormFile("file")
	if err != nil {
		var mbe *http.MaxBytesError
		if errors.As(err, &mbe) {
			h.respond(c, 0, nil, ErrTooLarge)
			return
		}
		h.respond(c, 0, nil, ErrFileFieldMissing)
		return
	}
	f, err := fh.Open()
	if err != nil {
		h.respond(c, 0, nil, err)
		return
	}
	defer func() { _ = f.Close() }()
	p, _ := auth.CurrentPrincipal(c)
	b, replayed, err := h.svc.Upload(c.Request.Context(), a, audit.UserActor(p.UserID, p.Username), UploadInput{FileName: fh.Filename,
		DeclaredType: fh.Header.Get("Content-Type"), IdempotencyKey: c.GetHeader(IdempotencyHeader),
		RequestID: c.GetString(httpx.RequestIDKey), Body: f})
	status := http.StatusCreated
	if replayed {
		c.Header("Idempotent-Replayed", "true")
		status = http.StatusOK
	}
	h.respond(c, status, b, err)
}

// List godoc
//
//	@Summary		List opening stock import previews
//	@Description	The caller's batches, newest first (every batch for a caller with every location). Requires stock.import.preview.
//	@Tags			opening-stock-imports
//	@Produce		json
//	@Param			page	query		int	false	"Page (from 1)"
//	@Param			limit	query		int	false	"Page size (1-100, default 25)"
//	@Success		200		{object}	httpx.DataResponse{data=openingstock.BatchPage}
//	@Failure		401		{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403		{object}	httpx.ErrorResponse	"FORBIDDEN"
//	@Failure		422		{object}	httpx.ErrorResponse	"VALIDATION_FAILED (page, limit)"
//	@Router			/opening-stock-imports [get]
func (h *Handler) List(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	p, err := h.svc.List(c.Request.Context(), a, c.Query("page"), c.Query("limit"))
	h.respond(c, http.StatusOK, p, err)
}

// Get godoc
//
//	@Summary		Get an opening stock import preview
//	@Description	The batch: file reference (attachment ID, name, SHA-256, size), uploader, upload time, template version, business date, expiry (expired: history only, G-07 must refuse it), status, summary counts and file issues. commitAvailable is always false: committing is the separate G-07 workflow. Another user's batch is 404 unless the caller has every location. Requires stock.import.preview.
//	@Tags			opening-stock-imports
//	@Produce		json
//	@Param			id	path		string	true	"Batch ID"
//	@Success		200	{object}	httpx.DataResponse{data=openingstock.Batch}
//	@Failure		401	{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403	{object}	httpx.ErrorResponse	"FORBIDDEN"
//	@Failure		404	{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Router			/opening-stock-imports/{id} [get]
func (h *Handler) Get(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	b, err := h.svc.Get(c.Request.Context(), a, c.Param("id"))
	h.respond(c, http.StatusOK, b, err)
}

// Rows godoc
//
//	@Summary		Rows of an opening stock import preview
//	@Description	Staged rows in file order (rowNumber = line in the uploaded file): the values exactly as uploaded (source), the normalised values and resolved references G-07 will use (normalised, null when invalid), a display snapshot of the resolution (resolved) and every issue (field, code, severity, message, suggestion). Filters: status, duplicate=true (any duplicate identifier issue), unresolved=true (any catalogue/location mapping issue). Requires stock.import.preview.
//	@Tags			opening-stock-imports
//	@Produce		json
//	@Param			id			path		string	true	"Batch ID"
//	@Param			status		query		string	false	"Row status"	Enums(VALID, WARNING, ERROR, BLOCKED_UNVERIFIED_MAPPING)
//	@Param			duplicate	query		bool	false	"Only rows with a duplicate identifier issue"
//	@Param			unresolved	query		bool	false	"Only rows with an unresolved catalogue or location mapping"
//	@Param			page		query		int		false	"Page (from 1)"
//	@Param			limit		query		int		false	"Page size (1-200, default 50)"
//	@Success		200			{object}	httpx.DataResponse{data=openingstock.RowPage}
//	@Failure		401			{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403			{object}	httpx.ErrorResponse	"FORBIDDEN"
//	@Failure		404			{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Failure		422			{object}	httpx.ErrorResponse	"VALIDATION_FAILED (status, duplicate, unresolved, page, limit)"
//	@Router			/opening-stock-imports/{id}/rows [get]
func (h *Handler) Rows(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	p, err := h.svc.Rows(c.Request.Context(), a, c.Param("id"), RowQuery{Status: c.Query("status"), Duplicate: c.Query("duplicate"),
		Unresolved: c.Query("unresolved"), Page: c.Query("page"), Limit: c.Query("limit")})
	h.respond(c, http.StatusOK, p, err)
}

// Approve godoc
//
//	@Summary		Approve an opening stock import batch (G-07)
//	@Description	Records, once, that the batch was reconciled to a signed physical count: physicalCountReference (1-100 characters) and physicalCountDate (not after today), mappingsReviewed and duplicatesReviewed (both must be true) and receivedDatesApproved (whether the file's received dates are the agreed opening-stock dates; false leaves the units' stock age MISSING_ORIGIN in E-07). Only a non-expired PREVIEWED batch in OS-IMPORT-1 whose every line is VALID can be approved; every line's location must be in the caller's scope. The same approval repeated by its approver returns 200 (Idempotent-Replayed: true); any other approval of an approved batch is 409. Approvals are never changed or deleted. Requires stock.import.approve (granted to no role until the owner signs off) and X-CSRF-Token. Creates no stock.
//	@Tags			opening-stock-imports
//	@Accept			json
//	@Produce		json
//	@Param			X-CSRF-Token	header		string							true	"CSRF token from the session"
//	@Param			id				path		string							true	"Batch ID"
//	@Param			body			body		openingstock.ApprovalInput		true	"Approval"
//	@Success		201				{object}	httpx.DataResponse{data=openingstock.Approval}
//	@Success		200				{object}	httpx.DataResponse{data=openingstock.Approval}	"Replay of the same approval"
//	@Failure		400				{object}	httpx.ErrorResponse	"BAD_REQUEST (malformed JSON)"
//	@Failure		401				{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403				{object}	httpx.ErrorResponse	"FORBIDDEN (permission, or a line's location outside the caller's scope) or CSRF_FAILED"
//	@Failure		404				{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Failure		409				{object}	httpx.ErrorResponse	"INVALID_STATE (rejected/other template, or not every line VALID), IMPORT_PREVIEW_EXPIRED, IMPORT_ALREADY_APPROVED or IMPORT_ALREADY_COMMITTED"
//	@Failure		422				{object}	httpx.ErrorResponse	"VALIDATION_FAILED (physicalCountReference, physicalCountDate, mappingsReviewed, duplicatesReviewed, receivedDatesApproved, note)"
//	@Router			/opening-stock-imports/{id}/approval [post]
func (h *Handler) Approve(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	var in ApprovalInput
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.Error(c, http.StatusBadRequest, httpx.CodeBadRequest, "The request body must be a JSON approval.")
		return
	}
	p, _ := auth.CurrentPrincipal(c)
	ap, replayed, err := h.svc.Approve(c.Request.Context(), a, audit.UserActor(p.UserID, p.Username), c.Param("id"), in,
		c.GetString(httpx.RequestIDKey))
	status := http.StatusCreated
	if replayed {
		c.Header("Idempotent-Replayed", "true")
		status = http.StatusOK
	}
	h.respond(c, status, ap, err)
}

// Commit godoc
//
//	@Summary		Commit an approved opening stock import batch (G-07)
//	@Description	Turns the batch into stock exactly once, all lines or none. Never trusts the preview: the stored file is read again (SHA-256, size, template version, same lines and values as staged) and, in one transaction with the batch locked and the catalogue/locations/suppliers share-locked, every line is re-validated against the current database (catalogue verification and activation, location active and in the caller's scope, chassis/frame duplicates against vehicle units, receipt drafts and the file, stock status approved for opening stock, positive cost, same resolution as the approved preview). Warnings are not accepted (no approved rule, OQ-49). Any refused line → 422 IMPORT_ROWS_BLOCKED (details: rows[line].field — CODE (SEVERITY): message) and nothing is written. On success each line becomes a vehicle unit (source OPENING_STOCK, linked to its batch line; no supplier/PO/GRN history is invented), with a REGISTERED history entry, an OPENING_BALANCE stock movement (none → its status, 0 → 1, at its location, source = the batch) and audit events. Idempotency-Key is required: the same key from the same user returns the commit again (200, Idempotent-Replayed: true, replayed: true); the same key for another batch is 409. A batch committed by anyone else is 409 IMPORT_ALREADY_COMMITTED. Requires stock.import.commit (separate from preview and approval; granted to no role until the owner signs off) and X-CSRF-Token.
//	@Tags			opening-stock-imports
//	@Produce		json
//	@Param			X-CSRF-Token	header		string	true	"CSRF token from the session"
//	@Param			Idempotency-Key	header		string	true	"8-100 letters, digits, - or _"
//	@Param			id				path		string	true	"Batch ID"
//	@Success		201				{object}	httpx.DataResponse{data=openingstock.CommitResult}
//	@Success		200				{object}	httpx.DataResponse{data=openingstock.CommitResult}	"Replay of the same request"
//	@Failure		401				{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403				{object}	httpx.ErrorResponse	"FORBIDDEN (permission, or a line's location outside the caller's scope) or CSRF_FAILED"
//	@Failure		404				{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Failure		409				{object}	httpx.ErrorResponse	"IMPORT_APPROVAL_REQUIRED, IMPORT_ALREADY_COMMITTED, IMPORT_PREVIEW_EXPIRED, OPENING_STATUSES_NOT_APPROVED, IMPORT_SOURCE_INVALID, INVALID_STATE, DUPLICATE_IDENTIFIER (registered concurrently) or IDEMPOTENCY_KEY_REUSED"
//	@Failure		422				{object}	httpx.ErrorResponse	"IMPORT_ROWS_BLOCKED (details per line and field) or VALIDATION_FAILED (Idempotency-Key)"
//	@Router			/opening-stock-imports/{id}/commit [post]
func (h *Handler) Commit(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	p, _ := auth.CurrentPrincipal(c)
	res, err := h.svc.Commit(c.Request.Context(), a, audit.UserActor(p.UserID, p.Username), c.Param("id"),
		c.GetHeader(IdempotencyHeader), c.GetString(httpx.RequestIDKey))
	status := http.StatusCreated
	if res.Replayed {
		c.Header("Idempotent-Replayed", "true")
		status = http.StatusOK
	}
	h.respond(c, status, res, err)
}

// GetCommit godoc
//
//	@Summary		Get an opening stock import batch's commit (G-07)
//	@Description	The commit of a committed batch: commit ID and OSC reference, batch, COMMITTED status, units created (line, vehicle unit, unit number, status, OPENING_BALANCE movement), committer, time, approval and file fingerprint. 404 until the batch is committed. Requires stock.import.preview, stock.import.approve or stock.import.commit.
//	@Tags			opening-stock-imports
//	@Produce		json
//	@Param			id	path		string	true	"Batch ID"
//	@Success		200	{object}	httpx.DataResponse{data=openingstock.CommitResult}
//	@Failure		401	{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403	{object}	httpx.ErrorResponse	"FORBIDDEN"
//	@Failure		404	{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Router			/opening-stock-imports/{id}/commit [get]
func (h *Handler) GetCommit(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	res, err := h.svc.GetCommit(c.Request.Context(), a, c.Param("id"))
	h.respond(c, http.StatusOK, res, err)
}
