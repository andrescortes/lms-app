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

// logoutStub refreshes answers RevokeByJTI with a fixed outcome.
type logoutStub struct {
	userID string
	jti    string
	err    error
}

func (f *logoutStub) Store(context.Context, string, string, string, time.Time) error { return nil }
func (f *logoutStub) Revoke(context.Context, string) error                           { return nil }
func (f *logoutStub) Rotate(context.Context, string, string, string, time.Time) (usecase.Rotation, error) {
	return usecase.Rotation{}, nil
}

func (f *logoutStub) RevokeByJTI(_ context.Context, userID, jti string) error {
	f.userID = userID
	f.jti = jti
	return f.err
}

// newLogoutRouter mounts the logout route behind a verifier that always trusts
// the single test token, so HTTP-level refusals come from the handler.
func newLogoutRouter(t *testing.T, repo *logoutStub) http.Handler {
	t.Helper()
	return NewRouter(
		nil, nil, nil,
		usecase.NewLogoutUseCase(repo),
		nil, nil,
		&stubVerifier{claims: usecase.TokenClaims{UserID: "user-1", Role: "STUDENT"}},
		NewVisitorLimiter(),
		nil,
	)
}

func postLogout(h http.Handler, body, bearer string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, LogoutRoute, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	req.RemoteAddr = "192.0.2.90:1"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// TestLogoutSuccessReturns204 verifies the success contract: 204 and an empty
// body, with the session revoked for the caller's own identity.
func TestLogoutSuccessReturns204(t *testing.T) {
	repo := &logoutStub{}
	h := newLogoutRouter(t, repo)

	rec := postLogout(h, `{"refreshToken":"jti-1"}`, "good-token")
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204; body=%s", rec.Code, rec.Body.String())
	}
	if body := strings.TrimSpace(rec.Body.String()); body != "" {
		t.Fatalf("body = %q, want empty", body)
	}
	if repo.userID != "user-1" || repo.jti != "jti-1" {
		t.Errorf("revoked (%q, %q), want (user-1, jti-1)", repo.userID, repo.jti)
	}
}

// TestLogoutMalformedRequestReturns400 verifies malformed and empty bodies.
func TestLogoutMalformedRequestReturns400(t *testing.T) {
	cases := map[string]string{
		"truncated":     `{"refreshToken":`,
		"unknown field": `{"refreshToken":"x","other":1}`,
		"empty token":   `{"refreshToken":""}`,
		"missing field": `{}`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			h := newLogoutRouter(t, &logoutStub{})
			rec := postLogout(h, body, "good-token")
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400; body=%s", rec.Code, rec.Body.String())
			}
		})
	}
}

// TestLogoutWithoutBearerReturns401 verifies the route is genuinely protected.
func TestLogoutWithoutBearerReturns401(t *testing.T) {
	h := newLogoutRouter(t, &logoutStub{})
	rec := postLogout(h, `{"refreshToken":"jti-1"}`, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

// TestLogoutInvalidBearerReturns401 verifies a verifier refusal is a 401.
func TestLogoutInvalidBearerReturns401(t *testing.T) {
	h := NewRouter(
		nil, nil, nil,
		usecase.NewLogoutUseCase(&logoutStub{}),
		nil, nil,
		&stubVerifier{err: domain.ErrInvalidToken},
		NewVisitorLimiter(),
		nil,
	)
	rec := postLogout(h, `{"refreshToken":"jti-1"}`, "expired-token")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

// TestLogoutInvalidRefreshTokenReturnsInvalidToken verifies a foreign, expired,
// or revoked session returns the same 401 INVALID_TOKEN.
func TestLogoutInvalidRefreshTokenReturnsInvalidToken(t *testing.T) {
	h := newLogoutRouter(t, &logoutStub{err: domain.ErrInvalidToken})
	rec := postLogout(h, `{"refreshToken":"foreign"}`, "good-token")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	assertEnvelope(t, rec.Body.Bytes(), codeInvalidToken)
}

// TestLogoutBodyIsNotIdentity verifies a caller cannot name another user in the
// body to end their session.
func TestLogoutBodyIsNotIdentity(t *testing.T) {
	repo := &logoutStub{}
	h := newLogoutRouter(t, repo)

	missing := postLogout(h, `{"refreshToken":"jti-1"}`, "")
	if missing.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", missing.Code)
	}
	if repo.jti != "" {
		t.Fatalf("store called with %q, want no call", repo.jti)
	}

	ok := postLogout(h, `{"refreshToken":"jti-2","userId":"someone-else"}`, "good-token")
	if ok.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 for the unknown userId field", ok.Code)
	}
}
