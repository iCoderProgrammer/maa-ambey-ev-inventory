package middleware

import (
	"log/slog"
	"runtime/debug"

	"github.com/gin-gonic/gin"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/httpx"
)

// Recovery converts panics into a generic 500 error envelope. The stack trace
// is logged server-side only and never returned to the client.
func Recovery(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				log.ErrorContext(c.Request.Context(), "panic recovered",
					slog.String("request_id", c.GetString(httpx.RequestIDKey)),
					slog.Any("panic", rec),
					slog.String("stack", string(debug.Stack())),
				)
				if !c.Writer.Written() {
					httpx.InternalError(c)
				} else {
					c.Abort()
				}
			}
		}()
		c.Next()
	}
}
