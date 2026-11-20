package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"lms-english/identity-service/internal/domain"
	"lms-english/identity-service/internal/usecase"
)

// stubVerifier answers with a fixed identity or error.
type stubVerifier struct {
	claims usecase.TokenClaims
	err    error
	calls  int
}

func (f *stubVerifier) Verify(context.Context, string) (usecase.TokenClaims, error) {
	f.calls++
	return f.claims, f.err
}

// recordingHandler records whether the protected handler ran.
type recordingHandler struct{ ran bool }

func (h *recordingHandler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	h.ran = true
	w.WriteHeader(http.StatusNoContent)
}

func withBearer(r *http.Request, value string) *http.Request {
	r.Header.Set("Authorization", value)
	return r
}

// TestAuthenticateAcceptsValidToken verifies the identity reaches the handler and
// the handler runs exactly once.
func TestAuthenticateAcceptsValidToken(t *testing.T) {
	next := &recordingHandler{}
	h := Authenticate(&stubVerifier{claims: usecase.TokenClaims{UserID: "user-1", Role: "STUDENT"}})(next)

	req := withBearer(httptest.NewRequest(http.MethodPost, "/x", nil), "Bearer good-token")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
	if !next.ran {
		t.Fatal("protected handler did not run for a valid token")
	}
}

// TestAuthenticateRefusesBadCredentials verifies missing, malformed, expired, and
// wrongly signed tokens all get the same 401 and never reach the handler.
func TestAuthenticateRefusesBadCredentials(t *testing.T) {
	cases := map[string]struct {
		header string
		set    bool
		err    error
	}{
		"missing header":  {set: false},
		"empty value":     {header: "", set: true},
		"no scheme":       {header: "good-token", set: true},
		"wrong scheme":    {header: "Basic good-token", set: true},
		"empty token":     {header: "Bearer ", set: true},
		"two words":       {header: "Bearer a b", set: true},
		"expired":         {header: "Bearer expired", set: true, err: domain.ErrInvalidToken},
		"wrong algorithm": {header: "Bearer alg-none", set: true, err: domain.ErrInvalidToken},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			next := &recordingHandler{}
			verifier := &stubVerifier{err: tc.err}
			if tc.err == nil {
				verifier.err = domain.ErrInvalidToken
			}
			h := Authenticate(verifier)(next)

			req := httptest.NewRequest(http.MethodPost, "/x", nil)
			if tc.set {
				req = withBearer(req, tc.header)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want 401", rec.Code)
			}
			if next.ran {
				t.Fatal("protected handler ran despite a refused credential")
			}
		})
	}
}

// TestAuthenticateRejectsDuplicateHeader verifies a second Authorization header
// is not silently ignored.
func TestAuthenticateRejectsDuplicateHeader(t *testing.T) {
	next := &recordingHandler{}
	h := Authenticate(&stubVerifier{claims: usecase.TokenClaims{UserID: "user-1", Role: "STUDENT"}})(next)

	req := httptest.NewRequest(http.MethodPost, "/x", nil)
	req.Header.Add("Authorization", "Bearer one")
	req.Header.Add("Authorization", "Bearer two")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	if next.ran {
		t.Fatal("handler ran with duplicate Authorization headers")
	}
}

// TestRequireRole verifies an allowed role passes and a wrong role is refused.
func TestRequireRole(t *testing.T) {
	mw := RequireRole("TEACHER", "ADMIN")

	t.Run("allowed", func(t *testing.T) {
		next := &recordingHandler{}
		h := mw(next)
		req := httptest.NewRequest(http.MethodPost, "/x", nil)
		req = req.WithContext(WithClaims(req.Context(), Claims{UserID: "u", Role: "TEACHER"}))
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want 204", rec.Code)
		}
		if !next.ran {
			t.Fatal("handler did not run for an allowed role")
		}
	})

	t.Run("wrong role", func(t *testing.T) {
		next := &recordingHandler{}
		h := mw(next)
		req := httptest.NewRequest(http.MethodPost, "/x", nil)
		req = req.WithContext(WithClaims(req.Context(), Claims{UserID: "u", Role: "STUDENT"}))
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want 403", rec.Code)
		}
		assertEnvelope(t, rec.Body.Bytes(), codeForbidden)
		if next.ran {
			t.Fatal("handler ran for a disallowed role")
		}
	})

	t.Run("no identity", func(t *testing.T) {
		next := &recordingHandler{}
		h := mw(next)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/x", nil))
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want 500 (route mounted without Authenticate)", rec.Code)
		}
	})
}

// TestRequireRolePanicsOnEmptyList verifies a route that would allow nobody fails
// when it is built.
func TestRequireRolePanicsOnEmptyList(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("RequireRole() with no roles did not panic")
		}
	}()
	RequireRole()
}

// TestClaimsFromContextRejectsZeroIdentity verifies an empty identity reads as
// absent rather than as a valid caller.
func TestClaimsFromContextRejectsZeroIdentity(t *testing.T) {
	ctx := WithClaims(context.Background(), Claims{})
	if _, ok := ClaimsFromContext(ctx); ok {
		t.Fatal("zero identity reported as present")
	}
	if once, _ := ClaimsFromContext(WithClaims(context.Background(), Claims{UserID: "u", Role: "STUDENT"})); once.UserID != "u" {
		t.Fatalf("UserID = %q, want u", once.UserID)
	}
}
