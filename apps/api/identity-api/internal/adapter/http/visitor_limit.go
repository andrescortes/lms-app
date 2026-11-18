package http

import (
	"net"
	"net/http"
	"sync"
	"time"
)

// Lockout policy: the first allowedAttempts attempts within window are accepted;
// the next attempt starts a lockout that rejects further attempts for
// lockoutDuration. The response never reveals the remaining time.
const (
	allowedAttempts = 5
	window          = 15 * time.Minute
	lockoutDuration = 30 * time.Minute
)

// visitorLimitCap bounds the number of tracked visitors so the in-memory map
// cannot grow without limit under address churn.
const visitorLimitCap = 100_000

// visitorState tracks attempt timestamps and an optional lockout expiry.
type visitorState struct {
	attempts   []time.Time
	lockedTill time.Time
}

// VisitorLimiter enforces the per-visitor registration attempt limit. It is
// safe for concurrent use and keyed on the server-observed peer address, so
// unconfigured forwarding headers cannot bypass it.
type VisitorLimiter struct {
	mu      sync.Mutex
	now     func() time.Time
	state   map[string]*visitorState
	allowed int
	window  time.Duration
	lockout time.Duration
	cap     int
}

// NewVisitorLimiter builds a limiter with the documented policy.
func NewVisitorLimiter() *VisitorLimiter {
	return &VisitorLimiter{
		now:     time.Now,
		state:   make(map[string]*visitorState),
		allowed: allowedAttempts,
		window:  window,
		lockout: lockoutDuration,
		cap:     visitorLimitCap,
	}
}

// Allow records an attempt for the request's visitor and reports whether it may
// proceed. A locked-out visitor is always rejected. The sixth attempt within the
// window starts the lockout and is itself rejected.
func (l *VisitorLimiter) Allow(r *http.Request) bool {
	key := visitorKey(r.RemoteAddr)
	if key == "" {
		// Without a usable peer address we fail open rather than block everyone.
		return true
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	st := l.state[key]
	if st == nil {
		if len(l.state) >= l.cap {
			l.evictExpiredLocked(now)
		}
		st = &visitorState{}
		l.state[key] = st
	}

	if now.Before(st.lockedTill) {
		return false
	}

	// Drop attempts outside the window.
	cutoff := now.Add(-l.window)
	kept := st.attempts[:0]
	for _, t := range st.attempts {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	st.attempts = kept

	if len(st.attempts) >= l.allowed {
		// This (sixth) attempt triggers the lockout and is rejected.
		st.lockedTill = now.Add(l.lockout)
		st.attempts = nil
		return false
	}

	st.attempts = append(st.attempts, now)
	return true
}

// evictExpiredLocked removes visitors with no active window or lockout, keeping
// the map bounded without dropping active lockouts early.
func (l *VisitorLimiter) evictExpiredLocked(now time.Time) {
	cutoff := now.Add(-l.window)
	for k, st := range l.state {
		if now.Before(st.lockedTill) {
			continue
		}
		active := false
		for _, t := range st.attempts {
			if t.After(cutoff) {
				active = true
				break
			}
		}
		if !active {
			delete(l.state, k)
		}
	}
}

// Middleware wraps next, rejecting locked-out visitors with the standard
// rate-limit envelope (no Retry-After, no remaining time).
func (l *VisitorLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !l.Allow(r) {
			writeJSON(w, http.StatusTooManyRequests, errorResponse{
				Code:      codeRateLimited,
				Message:   "too many registration attempts; please try again later",
				Timestamp: l.now().UTC().Format(time.RFC3339Nano),
				Path:      r.URL.Path,
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// visitorKey extracts the peer host from a RemoteAddr of the form host:port. The
// port is dropped so a single address shares one budget.
func visitorKey(remoteAddr string) string {
	if remoteAddr == "" {
		return ""
	}
	if host, _, err := net.SplitHostPort(remoteAddr); err == nil {
		return host
	}
	return remoteAddr
}
