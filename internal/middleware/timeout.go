package middleware

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/httpx"
)

// Timeout bounds the request context. Handlers and pgx queries that honour
// ctx are cancelled once the deadline passes; the http.Server read/write
// timeouts provide the outer safety net. If the deadline passed and the handler
// wrote nothing, the client gets a 503 REQUEST_TIMEOUT envelope instead of an
// empty 200.
func Timeout(d time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), d)
		defer cancel()
		c.Request = c.Request.WithContext(ctx)
		c.Next()

		if !c.Writer.Written() && errors.Is(ctx.Err(), context.DeadlineExceeded) {
			httpx.Error(c, http.StatusServiceUnavailable, httpx.CodeTimeout,
				"The request took too long to complete. Check whether it took effect before trying again.")
		}
	}
}
