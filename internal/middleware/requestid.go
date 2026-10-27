// Package middleware contains the cross-cutting Gin middleware for the API.
package middleware

import (
	"regexp"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/httpx"
)

// RequestIDHeader carries the correlation ID in requests and responses.
const RequestIDHeader = "X-Request-ID"

var validRequestID = regexp.MustCompile(`^[A-Za-z0-9._-]{8,64}$`)

// RequestID reuses a well-formed incoming X-Request-ID or generates a new one,
// stores it in the context and echoes it in the response.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(RequestIDHeader)
		if !validRequestID.MatchString(id) {
			id = uuid.NewString()
		}
		c.Set(httpx.RequestIDKey, id)
		c.Header(RequestIDHeader, id)
		c.Next()
	}
}
