package attachments

import (
	"encoding/hex"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/httpx"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/modules/access"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/modules/auth"
)

// multipartOverhead is allowed on top of the file size for boundaries and headers.
const multipartOverhead = 64 << 10

const listLimit = 50

// Handler is the HTTP adapter for attachments.
type Handler struct {
	svc    *Service
	access *access.Handler
	prefix string
	log    *slog.Logger
}

// NewHandler builds the adapter. prefix is the API base path (/api/v1), used
// to build download links.
func NewHandler(svc *Service, accessH *access.Handler, prefix string, log *slog.Logger) *Handler {
	return &Handler{svc: svc, access: accessH, prefix: prefix, log: log}
}

// Register mounts the routes behind the session middleware. There is no
// update or delete route: stored attachments are immutable.
func (h *Handler) Register(rg *gin.RouterGroup) {
	upload := h.access.Require(access.PermAttachmentsUpload)
	view := h.access.Require(access.PermAttachmentsView)
	rg.GET("/attachments/limits", upload, h.Limits)
	rg.POST("/attachments", upload, h.Upload)
	rg.GET("/attachments", view, h.List)
	rg.GET("/attachments/:id", view, h.Get)
	rg.GET("/attachments/:id/link", view, h.Link)
	rg.GET("/attachments/:id/content", view, h.Content)
}

// AttachmentResponse is attachment metadata. Storage keys and locations of
// the bytes are never exposed.
type AttachmentResponse struct {
	ID          string    `json:"id" example:"0b8f5e9a-6d3c-4f21-9a7e-2c4d6e8f0a1b"`
	FileName    string    `json:"fileName" example:"supplier-bill.pdf"`
	ContentType string    `json:"contentType" example:"application/pdf"`
	SizeBytes   int64     `json:"sizeBytes" example:"482113"`
	SHA256      string    `json:"sha256" example:"9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08"`
	LocationID  *string   `json:"locationId" example:"7c9e6679-7425-40de-944b-e07fc1f90ae7"`
	Linked      bool      `json:"linked" example:"false"`
	UploadedBy  string    `json:"uploadedBy" example:"5b0f8a52-3c1e-4d6f-9a8b-0c1d2e3f4a5b"`
	CreatedAt   time.Time `json:"createdAt" example:"2026-10-13T09:30:00Z"`
	Previewable bool      `json:"previewable" example:"false"`
} //@name Attachment

func toResponse(a Attachment) AttachmentResponse {
	r := AttachmentResponse{
		ID: a.ID, FileName: a.OriginalName, ContentType: a.ContentType, SizeBytes: a.SizeBytes,
		SHA256: hex.EncodeToString(a.SHA256), Linked: a.OwnerType != "", UploadedBy: a.UploadedBy, CreatedAt: a.CreatedAt,
	}
	if a.LocationID != "" {
		id := a.LocationID
		r.LocationID = &id
	}
	if ft, ok := fileType(a.ContentType); ok {
		r.Previewable = ft.Previewable
	}
	return r
}

// LimitsResponse tells the client what the server accepts. The server checks again.
type LimitsResponse struct {
	MaxBytes     int64      `json:"maxBytes" example:"10485760"`
	AllowedTypes []FileType `json:"allowedTypes"`
} //@name AttachmentLimits

// LinkResponse is a short-lived download link for the signed-in user only.
type LinkResponse struct {
	URL         string    `json:"url" example:"/api/v1/attachments/0b8f5e9a-6d3c-4f21-9a7e-2c4d6e8f0a1b/content?disposition=attachment&expires=1792834200&signature=…"`
	ExpiresAt   time.Time `json:"expiresAt" example:"2026-10-13T09:35:00Z"`
	Disposition string    `json:"disposition" example:"attachment"`
} //@name AttachmentLink

func (h *Handler) actor(c *gin.Context) Actor {
	p, _ := auth.CurrentPrincipal(c)
	return Actor{UserID: p.UserID, Username: p.Username, RequestID: c.GetString(httpx.RequestIDKey)}
}

// Limits godoc
//
//	@Summary		Upload limits
//	@Description	Maximum file size and accepted types (content is detected server-side; the declared type and the file-name extension must match it). Requires attachments.upload.
//	@Tags			attachments
//	@Produce		json
//	@Success		200	{object}	httpx.DataResponse{data=attachments.LimitsResponse}
//	@Failure		401	{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403	{object}	httpx.ErrorResponse	"FORBIDDEN"
//	@Router			/attachments/limits [get]
func (h *Handler) Limits(c *gin.Context) {
	httpx.OK(c, LimitsResponse{MaxBytes: h.svc.MaxBytes(), AllowedTypes: AllowedTypes})
}

// Upload godoc
//
//	@Summary		Upload a private attachment
//	@Description	multipart/form-data with one part named "file". The file is stored privately under a server-generated key; nothing about it is public. Optional locationId must be in the caller's location scope. The new attachment is not linked to a business record yet, so only the uploader can see it. Requires attachments.upload and X-CSRF-Token.
//	@Tags			attachments
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			file		formData	file	true	"JPEG, PNG, WebP or PDF up to maxBytes"
//	@Param			locationId	query		string	false	"Location the file belongs to"
//	@Success		201	{object}	httpx.DataResponse{data=attachments.AttachmentResponse}
//	@Failure		400	{object}	httpx.ErrorResponse	"VALIDATION_FAILED (no file part)"
//	@Failure		401	{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403	{object}	httpx.ErrorResponse	"FORBIDDEN or CSRF_FAILED"
//	@Failure		413	{object}	httpx.ErrorResponse	"FILE_TOO_LARGE"
//	@Failure		415	{object}	httpx.ErrorResponse	"UNSUPPORTED_MEDIA_TYPE (type not allowed, or content does not match type/extension)"
//	@Failure		422	{object}	httpx.ErrorResponse	"VALIDATION_FAILED (empty file, unknown location)"
//	@Router			/attachments [post]
func (h *Handler) Upload(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, h.svc.MaxBytes()+multipartOverhead)
	mr, err := c.Request.MultipartReader()
	if err != nil {
		httpx.Error(c, http.StatusUnsupportedMediaType, httpx.CodeUnsupportedMediaType,
			"Send the file as multipart/form-data in a part named \"file\".")
		return
	}
	for {
		part, err := mr.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			h.uploadError(c, err)
			return
		}
		if part.FormName() != "file" || part.FileName() == "" {
			_ = part.Close()
			continue
		}
		att, err := h.svc.Upload(c.Request.Context(), h.actor(c), a, UploadInput{
			FileName: part.FileName(), DeclaredType: part.Header.Get("Content-Type"),
			LocationID: c.Query("locationId"), Body: part,
		})
		_ = part.Close()
		if err != nil {
			h.uploadError(c, err)
			return
		}
		c.JSON(http.StatusCreated, httpx.DataResponse{Data: toResponse(att)})
		return
	}
	httpx.Error(c, http.StatusBadRequest, httpx.CodeValidationFailed, "Choose a file to upload.",
		httpx.FieldError{Field: "file", Message: "is required"})
}

func (h *Handler) uploadError(c *gin.Context, err error) {
	var maxErr *http.MaxBytesError
	switch {
	case errors.Is(err, ErrTooLarge) || errors.As(err, &maxErr):
		httpx.Error(c, http.StatusRequestEntityTooLarge, httpx.CodeFileTooLarge,
			"The file is larger than the "+formatMB(h.svc.MaxBytes())+" limit.",
			httpx.FieldError{Field: "file", Message: "is too large"})
	case errors.Is(err, ErrTypeNotAllowed):
		httpx.Error(c, http.StatusUnsupportedMediaType, httpx.CodeUnsupportedMediaType,
			"This file type is not accepted. Upload a JPEG, PNG, WebP or PDF file.",
			httpx.FieldError{Field: "file", Message: "type is not allowed"})
	case errors.Is(err, ErrTypeMismatch):
		httpx.Error(c, http.StatusUnsupportedMediaType, httpx.CodeUnsupportedMediaType,
			"The file's content does not match its type or file-name extension.",
			httpx.FieldError{Field: "file", Message: "content does not match its type or extension"})
	case errors.Is(err, ErrEmpty):
		httpx.Error(c, http.StatusUnprocessableEntity, httpx.CodeValidationFailed, "The file is empty.",
			httpx.FieldError{Field: "file", Message: "is empty"})
	case errors.Is(err, ErrInvalidLocation):
		httpx.Error(c, http.StatusUnprocessableEntity, httpx.CodeValidationFailed, "The location does not exist.",
			httpx.FieldError{Field: "locationId", Message: "is not a known location"})
	case errors.Is(err, ErrForbidden):
		access.Forbidden(c)
	default:
		h.log.ErrorContext(c.Request.Context(), "attachment upload failed",
			slog.String("request_id", c.GetString(httpx.RequestIDKey)), slog.Any("error", err))
		httpx.InternalError(c)
	}
}

func formatMB(n int64) string {
	return strconv.FormatFloat(float64(n)/(1<<20), 'f', -1, 64) + " MB"
}

// List godoc
//
//	@Summary		My unlinked uploads
//	@Description	The caller's newest uploads (up to 50) that are not linked to a business record yet and are still in the caller's location scope. Requires attachments.view.
//	@Tags			attachments
//	@Produce		json
//	@Success		200	{object}	httpx.DataResponse{data=[]attachments.AttachmentResponse}
//	@Failure		401	{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403	{object}	httpx.ErrorResponse	"FORBIDDEN"
//	@Router			/attachments [get]
func (h *Handler) List(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	list, err := h.svc.ListMine(c.Request.Context(), a, listLimit)
	if err != nil {
		h.internal(c, "attachment list failed", err)
		return
	}
	out := make([]AttachmentResponse, len(list))
	for i, att := range list {
		out[i] = toResponse(att)
	}
	c.Header("Cache-Control", "no-store")
	httpx.OK(c, out)
}

// Get godoc
//
//	@Summary		Attachment metadata
//	@Description	Requires attachments.view and access to this attachment (location scope; unlinked attachments: uploader only). Attachments the caller cannot see return 404, so their existence is not revealed.
//	@Tags			attachments
//	@Produce		json
//	@Param			id	path		string	true	"Attachment ID"
//	@Success		200	{object}	httpx.DataResponse{data=attachments.AttachmentResponse}
//	@Failure		401	{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403	{object}	httpx.ErrorResponse	"FORBIDDEN"
//	@Failure		404	{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Router			/attachments/{id} [get]
func (h *Handler) Get(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	att, err := h.svc.Get(c.Request.Context(), a, c.Param("id"))
	if err != nil {
		h.readError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	httpx.OK(c, toResponse(att))
}

// Link godoc
//
//	@Summary		Short-lived download link
//	@Description	After the same checks as metadata, returns a link that works only for the caller, only with their session, and only until expiresAt (ATTACHMENT_LINK_TTL, default 5 minutes). disposition=inline is allowed for images (preview); everything else downloads.
//	@Tags			attachments
//	@Produce		json
//	@Param			id			path	string	true	"Attachment ID"
//	@Param			disposition	query	string	false	"attachment (default) or inline"	Enums(attachment, inline)
//	@Success		200	{object}	httpx.DataResponse{data=attachments.LinkResponse}
//	@Failure		401	{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403	{object}	httpx.ErrorResponse	"FORBIDDEN"
//	@Failure		404	{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Failure		422	{object}	httpx.ErrorResponse	"VALIDATION_FAILED (disposition)"
//	@Router			/attachments/{id}/link [get]
func (h *Handler) Link(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	disposition := c.DefaultQuery("disposition", DispositionAttachment)
	att, link, err := h.svc.IssueLink(c.Request.Context(), a, c.Param("id"), disposition)
	if err != nil {
		h.readError(c, err)
		return
	}
	q := url.Values{}
	q.Set("disposition", disposition)
	q.Set("expires", strconv.FormatInt(link.Expires.Unix(), 10))
	q.Set("signature", link.Signature)
	c.Header("Cache-Control", "no-store")
	httpx.OK(c, LinkResponse{
		URL:       h.prefix + "/attachments/" + url.PathEscape(att.ID) + "/content?" + q.Encode(),
		ExpiresAt: link.Expires.UTC(), Disposition: disposition,
	})
}

// Content godoc
//
//	@Summary		Download an attachment
//	@Description	Streams the file for a link from /attachments/{id}/link. Needs the same user's session; access is checked again, so a revoked permission stops downloads at once. Every download is audited. Served with nosniff, a sandboxing CSP and no caching.
//	@Tags			attachments
//	@Produce		octet-stream
//	@Param			id			path	string	true	"Attachment ID"
//	@Param			disposition	query	string	true	"attachment or inline"
//	@Param			expires		query	string	true	"From the link"
//	@Param			signature	query	string	true	"From the link"
//	@Success		200	{file}		file
//	@Failure		401	{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403	{object}	httpx.ErrorResponse	"FORBIDDEN, DOWNLOAD_LINK_INVALID or DOWNLOAD_LINK_EXPIRED"
//	@Failure		404	{object}	httpx.ErrorResponse	"NOT_FOUND"
//	@Router			/attachments/{id}/content [get]
func (h *Handler) Content(c *gin.Context) {
	a, ok := h.access.Current(c)
	if !ok {
		return
	}
	disposition := c.Query("disposition")
	att, f, err := h.svc.Open(c.Request.Context(), h.actor(c), a, c.Param("id"), disposition, c.Query("expires"), c.Query("signature"))
	if err != nil {
		h.readError(c, err)
		return
	}
	defer f.Close()
	w := c.Writer.Header()
	w.Set("Content-Type", att.ContentType)
	w.Set("Content-Disposition", mime.FormatMediaType(disposition, map[string]string{"filename": att.OriginalName}))
	w.Set("X-Content-Type-Options", "nosniff")
	w.Set("Cache-Control", "private, no-store")
	w.Set("Content-Security-Policy", "default-src 'none'; img-src 'self'; style-src 'unsafe-inline'; sandbox")
	w.Set("Cross-Origin-Resource-Policy", "same-origin")
	w.Set("Referrer-Policy", "no-referrer")
	http.ServeContent(c.Writer, c.Request, "", att.CreatedAt, f)
}

func (h *Handler) readError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.Error(c, http.StatusNotFound, httpx.CodeNotFound, "The attachment was not found.")
	case errors.Is(err, ErrLinkExpired):
		httpx.Error(c, http.StatusForbidden, httpx.CodeDownloadLinkExpired, "This download link has expired. Open the file again.")
	case errors.Is(err, ErrLinkInvalid):
		httpx.Error(c, http.StatusForbidden, httpx.CodeDownloadLinkInvalid, "This download link is not valid.")
	case errors.Is(err, ErrBadDisposition):
		httpx.Error(c, http.StatusUnprocessableEntity, httpx.CodeValidationFailed, "Unknown disposition.",
			httpx.FieldError{Field: "disposition", Message: "must be attachment or inline"})
	case errors.Is(err, ErrInlineNotAllowed):
		httpx.Error(c, http.StatusUnprocessableEntity, httpx.CodeValidationFailed, "Only images can be previewed.",
			httpx.FieldError{Field: "disposition", Message: "inline is allowed for images only"})
	default:
		h.internal(c, "attachment read failed", err)
	}
}

func (h *Handler) internal(c *gin.Context, msg string, err error) {
	h.log.ErrorContext(c.Request.Context(), msg,
		slog.String("request_id", c.GetString(httpx.RequestIDKey)), slog.Any("error", err))
	httpx.InternalError(c)
}
