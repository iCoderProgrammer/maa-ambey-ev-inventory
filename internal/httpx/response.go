// Package httpx defines the JSON response and error envelopes shared by all
// API handlers. See docs/API_CONVENTIONS.md.
package httpx

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RequestIDKey is the gin context key holding the request ID.
const RequestIDKey = "request_id"

// Error codes are stable, machine-readable identifiers for clients.
const (
	CodeBadRequest       = "BAD_REQUEST"
	CodeValidationFailed = "VALIDATION_FAILED"
	CodeUnauthenticated  = "UNAUTHENTICATED"
	CodeForbidden        = "FORBIDDEN"
	CodeNotFound         = "NOT_FOUND"
	CodeMethodNotAllowed = "METHOD_NOT_ALLOWED"
	CodeConflict         = "CONFLICT"
	CodeRateLimited      = "RATE_LIMITED"
	CodeTimeout          = "REQUEST_TIMEOUT"
	CodeInternal         = "INTERNAL_ERROR"
	CodeUnavailable      = "SERVICE_UNAVAILABLE"

	// Authentication (B-06).
	CodeInvalidCredentials   = "INVALID_CREDENTIALS"
	CodeAccountDisabled      = "ACCOUNT_DISABLED"
	CodeSessionExpired       = "SESSION_EXPIRED"
	CodeCSRFFailed           = "CSRF_FAILED"
	CodeUnsupportedMediaType = "UNSUPPORTED_MEDIA_TYPE"

	// Attachments (B-09).
	CodeFileTooLarge        = "FILE_TOO_LARGE"
	CodeDownloadLinkInvalid = "DOWNLOAD_LINK_INVALID"
	CodeDownloadLinkExpired = "DOWNLOAD_LINK_EXPIRED"
	// Catalogue (C-02); all 409.
	CodeDuplicate         = "DUPLICATE"
	CodeVersionConflict   = "VERSION_CONFLICT"
	CodeInvalidState      = "INVALID_STATE"
	CodeActivationBlocked = "ACTIVATION_BLOCKED"
)

// DataResponse is the success envelope. In swag annotations use
// httpx.DataResponse{data=pkg.Type} to document the payload.
type DataResponse struct {
	Data any `json:"data"`
} //@name DataResponse

// FieldError describes one invalid input field.
type FieldError struct {
	Field   string `json:"field" example:"slug"`
	Message string `json:"message" example:"must be lowercase letters, digits and hyphens"`
} //@name FieldError

// ErrorBody is the payload of every error response.
type ErrorBody struct {
	Code      string       `json:"code" example:"NOT_FOUND"`
	Message   string       `json:"message" example:"The requested resource was not found."`
	RequestID string       `json:"requestId,omitempty" example:"7f9d0c1e-5c1a-4c55-9e0f-0b7f1f0f2a11"`
	Details   []FieldError `json:"details,omitempty"`
} //@name ErrorBody

// ErrorResponse is the error envelope.
type ErrorResponse struct {
	Error ErrorBody `json:"error"`
} //@name ErrorResponse

// OK writes a 200 success envelope.
func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, DataResponse{Data: data})
}

// Error writes an error envelope and aborts the handler chain. Messages must be
// safe for clients: never pass raw database errors, SQL or stack traces.
func Error(c *gin.Context, status int, code, message string, details ...FieldError) {
	c.AbortWithStatusJSON(status, ErrorResponse{Error: ErrorBody{
		Code:      code,
		Message:   message,
		RequestID: c.GetString(RequestIDKey),
		Details:   details,
	}})
}

// InternalError writes a generic 500 response. Log the underlying error before calling it.
func InternalError(c *gin.Context) {
	Error(c, http.StatusInternalServerError, CodeInternal,
		"Something went wrong. Quote the request ID if you contact support.")
}
