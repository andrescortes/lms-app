package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"lms-english/identity-service/internal/domain"
	"lms-english/identity-service/internal/usecase"
)

// ---- fakes for the HTTP layer ----

type stubStore struct {
	err error
}

func (s stubStore) RegisterUser(context.Context, domain.User, domain.Profile, domain.UserRegisteredEvent) error {
	return s.err
}

type stubHasher struct{ hash string }

func (h stubHasher) Hash(string) (string, error) { return h.hash, nil }

type stubID struct{}

func (stubID) New() string { return "11111111-1111-4111-8111-111111111111" }

type stubClock struct{}

func (stubClock) Now() time.Time { return time.Date(2026, 11, 18, 10, 0, 0, 0, time.UTC) }

type stubPublisher struct{}

func (stubPublisher) Publish(context.Context, string, []byte) error { return nil }

func newTestHandler(t *testing.T, store stubStore) http.Handler {
	t.Helper()
	uc := usecase.NewRegisterUserUseCase(store, stubHasher{hash: "hashed"}, stubID{}, stubClock{}, stubPublisher{})
	loginUc := usecase.NewLoginUserUseCase(nil, nil, nil, nil, nil, nil)
	return NewRouter(uc, loginUc, NewVisitorLimiter(), nil)
}

func post(handler http.Handler, remoteAddr, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, RegisterRoute, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = remoteAddr
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

// TestRegisterSuccess verifies the 201 contract: only userId, with a Location.
func TestRegisterSuccess(t *testing.T) {
	h := newTestHandler(t, stubStore{})
	rec := post(h, "10.0.0.1:1234", `{"email":"Student@Example.com","password":"Learn123"}`)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("body is not JSON: %v", err)
	}
	if len(resp) != 1 {
		t.Fatalf("response keys = %v, want only userId", resp)
	}
	if resp["userId"] != "11111111-1111-4111-8111-111111111111" {
		t.Fatalf("userId = %v", resp["userId"])
	}
	if loc := rec.Header().Get("Location"); !strings.HasSuffix(loc, "/11111111-1111-4111-8111-111111111111") {
		t.Fatalf("Location = %q, want created user path", loc)
	}
	if strings.Contains(rec.Body.String(), "Learn123") {
		t.Fatal("response must not echo the password")
	}
}

// TestRegisterMalformedJSON verifies a 400 with the standard envelope.
func TestRegisterMalformedJSON(t *testing.T) {
	h := newTestHandler(t, stubStore{})
	rec := post(h, "10.0.0.2:1234", `{"email":"a@b.com",`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	assertEnvelope(t, rec.Body.Bytes(), codeInvalidRequest)
}

// TestRegisterUnknownField verifies DisallowUnknownFields yields a 400.
func TestRegisterUnknownField(t *testing.T) {
	h := newTestHandler(t, stubStore{})
	rec := post(h, "10.0.0.3:1234", `{"email":"a@b.com","password":"Learn123","admin":true}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

// TestRegisterInvalidEmail verifies the 422 email-rule response (US2).
func TestRegisterInvalidEmail(t *testing.T) {
	h := newTestHandler(t, stubStore{})
	rec := post(h, "10.0.0.4:1234", `{"email":"not-an-email","password":"Learn123"}`)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422; body=%s", rec.Code, rec.Body.String())
	}
	assertEnvelopeMessage(t, rec.Body.Bytes(), codeValidation, "email")
}

// TestRegisterWeakPassword verifies the 422 password-rule response (US3).
func TestRegisterWeakPassword(t *testing.T) {
	h := newTestHandler(t, stubStore{})
	rec := post(h, "10.0.0.5:1234", `{"email":"a@b.com","password":"weak"}`)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", rec.Code)
	}
	assertEnvelopeMessage(t, rec.Body.Bytes(), codeValidation, "password")
	if strings.Contains(rec.Body.String(), "weak") {
		t.Fatal("response must not echo the password")
	}
}

// TestRegisterDuplicate verifies the 409 conflict response (US4).
func TestRegisterDuplicate(t *testing.T) {
	h := newTestHandler(t, stubStore{err: domain.ErrEmailAlreadyExists})
	rec := post(h, "10.0.0.6:1234", `{"email":"a@b.com","password":"Learn123"}`)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", rec.Code)
	}
	assertEnvelope(t, rec.Body.Bytes(), codeEmailExists)
	if strings.Contains(strings.ToLower(rec.Body.String()), "hash") {
		t.Fatal("conflict response must not leak account details")
	}
}

// TestRateLimitSixthAttempt verifies the sixth attempt is rejected with 429 and
// no Retry-After (US1 rate limiting).
func TestRateLimitSixthAttempt(t *testing.T) {
	limiter := NewVisitorLimiter()
	uc := usecase.NewRegisterUserUseCase(stubStore{}, stubHasher{hash: "hashed"}, stubID{}, stubClock{}, stubPublisher{})
	loginUc := usecase.NewLoginUserUseCase(nil, nil, nil, nil, nil, nil)
	h := NewRouter(uc, loginUc, limiter, nil)

	for i := 1; i <= 5; i++ {
		rec := post(h, "10.0.0.9:1234", `{"email":"a@b.com","password":"Learn123"}`)
		if rec.Code != http.StatusCreated {
			t.Fatalf("attempt %d status = %d, want 201", i, rec.Code)
		}
	}
	rec := post(h, "10.0.0.9:1234", `{"email":"a@b.com","password":"Learn123"}`)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("sixth attempt status = %d, want 429", rec.Code)
	}
	if rec.Header().Get("Retry-After") != "" {
		t.Fatal("429 must not include Retry-After")
	}
	assertEnvelope(t, rec.Body.Bytes(), codeRateLimited)
}

// TestRateLimitIsolation verifies another visitor is not affected by one
// visitor's lockout (same-network caveat is documented).
func TestRateLimitIsolation(t *testing.T) {
	limiter := NewVisitorLimiter()
	uc := usecase.NewRegisterUserUseCase(stubStore{}, stubHasher{hash: "hashed"}, stubID{}, stubClock{}, stubPublisher{})
	loginUc := usecase.NewLoginUserUseCase(nil, nil, nil, nil, nil, nil)
	h := NewRouter(uc, loginUc, limiter, nil)

	for i := 0; i < 6; i++ {
		post(h, "10.1.1.1:1234", `{"email":"a@b.com","password":"Learn123"}`)
	}
	rec := post(h, "10.2.2.2:1234", `{"email":"a@b.com","password":"Learn123"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("other visitor status = %d, want 201", rec.Code)
	}
}

func assertEnvelope(t *testing.T, body []byte, wantCode string) {
	t.Helper()
	var env errorResponse
	if err := json.Unmarshal(body, &env); err != nil {
		t.Fatalf("error body is not JSON: %v (%s)", err, body)
	}
	if env.Code != wantCode {
		t.Fatalf("error code = %q, want %q", env.Code, wantCode)
	}
	if env.Message == "" || env.Timestamp == "" || env.Path == "" {
		t.Fatalf("error envelope missing required fields: %+v", env)
	}
}

func assertEnvelopeMessage(t *testing.T, body []byte, wantCode, contains string) {
	t.Helper()
	var env errorResponse
	if err := json.Unmarshal(body, &env); err != nil {
		t.Fatalf("error body is not JSON: %v", err)
	}
	if env.Code != wantCode {
		t.Fatalf("error code = %q, want %q", env.Code, wantCode)
	}
	if !strings.Contains(strings.ToLower(env.Message), strings.ToLower(contains)) {
		t.Fatalf("message %q does not name %q", env.Message, contains)
	}
}
