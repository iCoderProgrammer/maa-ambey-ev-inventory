package auth

import (
	"sync"
	"time"
)

// LoginThrottle limits failed sign-in attempts per (username, client IP) and
// per client IP in a fixed window. It is in-memory, like the API rate limiter:
// enough for one API instance; a multi-instance deployment needs a shared store.
type LoginThrottle struct {
	maxPerAccount int
	maxPerIP      int
	window        time.Duration
	now           func() time.Time

	mu      sync.Mutex
	entries map[string]*failureWindow
	lastGC  time.Time
}

type failureWindow struct {
	start    time.Time
	failures int
}

// NewLoginThrottle builds a throttle; see config LOGIN_* settings.
func NewLoginThrottle(maxPerAccount, maxPerIP int, window time.Duration) *LoginThrottle {
	return &LoginThrottle{
		maxPerAccount: maxPerAccount,
		maxPerIP:      maxPerIP,
		window:        window,
		now:           time.Now,
		entries:       make(map[string]*failureWindow),
	}
}

func accountKey(username, ip string) string { return "a\x00" + username + "\x00" + ip }
func ipKey(ip string) string                { return "i\x00" + ip }

// Blocked reports whether another attempt is refused and for how long.
func (t *LoginThrottle) Blocked(username, ip string) (time.Duration, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	now := t.now()
	var wait time.Duration
	for key, limit := range map[string]int{accountKey(username, ip): t.maxPerAccount, ipKey(ip): t.maxPerIP} {
		if e, ok := t.entries[key]; ok && now.Before(e.start.Add(t.window)) && e.failures >= limit {
			wait = max(wait, e.start.Add(t.window).Sub(now))
		}
	}
	return wait, wait > 0
}

// Failure records a failed attempt.
func (t *LoginThrottle) Failure(username, ip string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	now := t.now()
	t.gc(now)
	for _, key := range []string{accountKey(username, ip), ipKey(ip)} {
		e, ok := t.entries[key]
		if !ok || !now.Before(e.start.Add(t.window)) {
			e = &failureWindow{start: now}
			t.entries[key] = e
		}
		e.failures++
	}
}

// Success clears the account counter after a successful sign-in. The per-IP
// counter is kept so one valid account cannot be used to reset guessing on others.
func (t *LoginThrottle) Success(username, ip string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.entries, accountKey(username, ip))
}

func (t *LoginThrottle) gc(now time.Time) {
	if now.Sub(t.lastGC) < t.window {
		return
	}
	for k, e := range t.entries {
		if !now.Before(e.start.Add(t.window)) {
			delete(t.entries, k)
		}
	}
	t.lastGC = now
}
