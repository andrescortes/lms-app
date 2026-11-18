package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// fixedLimiter builds a limiter whose clock is controllable.
func fixedLimiter(now *time.Time) *VisitorLimiter {
	l := NewVisitorLimiter()
	l.now = func() time.Time { return *now }
	return l
}

func reqFrom(addr string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, RegisterRoute, nil)
	r.RemoteAddr = addr
	return r
}

// TestVisitorLimiterBoundary verifies attempts 1..5 pass and the sixth locks out.
func TestVisitorLimiterBoundary(t *testing.T) {
	now := time.Date(2026, 11, 18, 10, 0, 0, 0, time.UTC)
	l := fixedLimiter(&now)
	r := reqFrom("192.0.2.1:5555")

	for i := 1; i <= 5; i++ {
		if !l.Allow(r) {
			t.Fatalf("attempt %d blocked, want allowed", i)
		}
	}
	if l.Allow(r) {
		t.Fatal("sixth attempt allowed, want blocked")
	}
	// Still blocked within the lockout window.
	now = now.Add(29 * time.Minute)
	if l.Allow(r) {
		t.Fatal("attempt within lockout allowed, want blocked")
	}
	// Allowed after the lockout expires.
	now = now.Add(2 * time.Minute)
	if !l.Allow(r) {
		t.Fatal("attempt after lockout expiry blocked, want allowed")
	}
}

// TestVisitorLimiterWindowResets verifies attempts older than the window do not
// count toward the limit.
func TestVisitorLimiterWindowResets(t *testing.T) {
	now := time.Date(2026, 11, 18, 10, 0, 0, 0, time.UTC)
	l := fixedLimiter(&now)
	r := reqFrom("192.0.2.2:5555")

	for i := 0; i < 4; i++ {
		l.Allow(r)
	}
	now = now.Add(16 * time.Minute) // all prior attempts fall out of the window
	for i := 0; i < 5; i++ {
		if !l.Allow(r) {
			t.Fatalf("attempt %d after window reset blocked", i+1)
		}
	}
}

// TestVisitorLimiterIsolation verifies different peer addresses have separate
// budgets.
func TestVisitorLimiterIsolation(t *testing.T) {
	now := time.Date(2026, 11, 18, 10, 0, 0, 0, time.UTC)
	l := fixedLimiter(&now)
	a := reqFrom("192.0.2.10:1")
	b := reqFrom("192.0.2.11:2")

	for i := 0; i < 6; i++ {
		l.Allow(a)
	}
	if l.Allow(a) {
		t.Fatal("address A should be locked out")
	}
	if !l.Allow(b) {
		t.Fatal("address B should be unaffected")
	}
}

// TestVisitorKeyStripsPort verifies port is removed so one address shares a budget.
func TestVisitorKeyStripsPort(t *testing.T) {
	if got := visitorKey("192.0.2.5:1234"); got != "192.0.2.5" {
		t.Fatalf("visitorKey = %q, want host only", got)
	}
	if got := visitorKey("192.0.2.6"); got != "192.0.2.6" {
		t.Fatalf("visitorKey = %q, want unchanged host", got)
	}
}
