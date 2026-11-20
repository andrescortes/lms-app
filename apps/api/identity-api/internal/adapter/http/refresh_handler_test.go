package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"lms-english/identity-service/internal/domain"
	"lms-english/identity-service/internal/usecase"
)

// refreshStubRepo answers Rotate with a fixed outcome.
type refreshStubRepo struct {
	outcome usecase.Rotation
	err     error
}

func (f *refreshStubRepo) Store(context.Context, string, string, string, time.Time) error {
	return nil
}

func (f *refreshStubRepo) Revoke(context.Context, string) error { return nil }

func (f *refreshStubRepo) RevokeByJTI(context.Context, string, string) error { return nil }

func (f *refreshStubRepo) Rotate(context.Context, string, string, string, time.Time) (usecase.Rotation, error) {
	return f.outcome, f.err
}

func newRefreshRouter(t *testing.T, repo *refreshStubRepo) http.Handler {
	t.Helper()
	uc := usecase.NewRefreshTokenUseCase(
		&loginStubUserRepo{user: domain.User{ID: "u1", Role: domain.RoleStudent}},
		&loginStubToken{token: "new-access"},
		repo,
		stubID{},
		stubClock{},
	)
	return NewRouter(nil, nil, uc, nil, nil, nil, nil, nil, NewVisitorLimiter(), nil)
}

func postRefresh(handler http.Handler, remoteAddr, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, RefreshRoute, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = remoteAddr
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

// TestRefreshSuccessReturnsNewPair verifies the 200 contract.
func TestRefreshSuccessReturnsNewPair(t *testing.T) {
	h := newRefreshRouter(t, &refreshStubRepo{outcome: usecase.Rotation{Outcome: usecase.RotationRotated, UserID: "u1"}})

	rec := postRefresh(h, "192.0.2.60:1", `{"refreshToken":"presented"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "new-access") {
		t.Fatalf("body = %q, want the new access token", rec.Body.String())
	}
}

// TestRefreshMalformedRequestReturns400 verifies the malformed-body contract.
func TestRefreshMalformedRequestReturns400(t *testing.T) {
	h := newRefreshRouter(t, &refreshStubRepo{outcome: usecase.Rotation{Outcome: usecase.RotationRotated}})

	rec := postRefresh(h, "192.0.2.61:1", `{"refreshToken":`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

// TestRefreshMissingTokenReturns400 verifies an empty token is a bad request,
// not an authentication failure.
func TestRefreshMissingTokenReturns400(t *testing.T) {
	h := newRefreshRouter(t, &refreshStubRepo{outcome: usecase.Rotation{Outcome: usecase.RotationRotated}})

	rec := postRefresh(h, "192.0.2.62:1", `{"refreshToken":""}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

// TestRefreshSpentTokenReturnsInvalidToken verifies every bad outcome maps to
// 401 INVALID_TOKEN.
func TestRefreshSpentTokenReturnsInvalidToken(t *testing.T) {
	for _, outcome := range []usecase.RotationOutcome{usecase.RotationSpent, usecase.RotationExpired, usecase.RotationUnknown} {
		h := newRefreshRouter(t, &refreshStubRepo{outcome: usecase.Rotation{Outcome: outcome}})
		rec := postRefresh(h, "192.0.2.63:1", `{"refreshToken":"presented"}`)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("outcome %v: status = %d, want 401", outcome, rec.Code)
		}
		assertEnvelope(t, rec.Body.Bytes(), codeInvalidToken)
	}
}

// TestRefreshRateLimitReturns429 verifies the refresh route caps attempts per
// client independently of sign-in.
func TestRefreshRateLimitReturns429(t *testing.T) {
	h := newRefreshRouter(t, &refreshStubRepo{outcome: usecase.Rotation{Outcome: usecase.RotationRotated}})
	const addr = "192.0.2.64:1"

	for i := 1; i <= loginAttemptsPerMinute; i++ {
		if rec := postRefresh(h, addr, `{"refreshToken":"presented"}`); rec.Code != http.StatusOK {
			t.Fatalf("attempt %d status = %d, want 200", i, rec.Code)
		}
	}
	if rec := postRefresh(h, addr, `{"refreshToken":"presented"}`); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("attempt %d status = %d, want 429", loginAttemptsPerMinute+1, rec.Code)
	}
}
