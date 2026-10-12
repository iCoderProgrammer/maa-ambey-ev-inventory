package middleware

import (
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"

	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/httpx"
)

// RateLimiter is an in-memory, per-client-IP token bucket. It is a baseline
// abuse guard for a single API instance; multi-instance deployments should
// add a limit at the load balancer / gateway as well.
type RateLimiter struct {
	rps   rate.Limit
	burst int
	ttl   time.Duration

	mu       sync.Mutex
	visitors map[string]*visitor
	lastGC   time.Time
	now      func() time.Time
}

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// NewRateLimiter creates a limiter allowing rps requests per second with the given burst per client IP.
func NewRateLimiter(rps float64, burst int) *RateLimiter {
	return &RateLimiter{
		rps:      rate.Limit(rps),
		burst:    burst,
		ttl:      10 * time.Minute,
		visitors: make(map[string]*visitor),
		now:      time.Now,
	}
}

func (rl *RateLimiter) limiterFor(key string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := rl.now()
	if now.Sub(rl.lastGC) > rl.ttl {
		for k, v := range rl.visitors {
			if now.Sub(v.lastSeen) > rl.ttl {
				delete(rl.visitors, k)
			}
		}
		rl.lastGC = now
	}

	v, ok := rl.visitors[key]
	if !ok {
		v = &visitor{limiter: rate.NewLimiter(rl.rps, rl.burst)}
		rl.visitors[key] = v
	}
	v.lastSeen = now
	return v.limiter
}

// Middleware rejects requests over the limit with 429 and a Retry-After header.
func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !rl.limiterFor(c.ClientIP()).AllowN(rl.now(), 1) {
			retry := int(math.Ceil(1 / float64(rl.rps)))
			c.Header("Retry-After", strconv.Itoa(max(retry, 1)))
			httpx.Error(c, http.StatusTooManyRequests, httpx.CodeRateLimited,
				"Too many requests. Please wait a moment and try again.")
			return
		}
		c.Next()
	}
}
