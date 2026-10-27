package middleware

import (
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/httpx"
)

// CORS allows only the configured origins. With no origins configured, no
// cross-origin access is granted (same-origin and the Vite proxy still work).
// Rejected origins get the standard 403 error envelope; gin-contrib/cors on
// its own would answer with an empty body.
func CORS(allowedOrigins []string) gin.HandlerFunc {
	if len(allowedOrigins) == 0 {
		return func(c *gin.Context) { c.Next() }
	}
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[o] = struct{}{}
	}
	handler := cors.New(cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", RequestIDHeader, "Idempotency-Key", "If-Match", "X-CSRF-Token"},
		ExposeHeaders:    []string{RequestIDHeader, "Retry-After"},
		AllowCredentials: true,
		MaxAge:           10 * time.Minute,
	})
	return func(c *gin.Context) {
		// Same checks as gin-contrib/cors: no Origin or same host is not a CORS request.
		origin := c.GetHeader("Origin")
		host := c.Request.Host
		if origin != "" && origin != "http://"+host && origin != "https://"+host {
			if _, ok := allowed[origin]; !ok {
				httpx.Error(c, http.StatusForbidden, httpx.CodeForbidden, "This origin is not allowed to call the API.")
				return
			}
		}
		handler(c)
	}
}
