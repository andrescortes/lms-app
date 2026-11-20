package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"lms-english/identity-service/internal/usecase"
)

// TestRegisterRespondsWithinBudget checks SC-001: under normal conditions a
// successful registration responds well within three seconds. "Normal
// conditions" here means a local in-process handler with a stub store; this
// bounds handler overhead, not database or network latency.
func TestRegisterRespondsWithinBudget(t *testing.T) {
	uc := usecase.NewRegisterUserUseCase(stubStore{}, stubHasher{hash: "hashed"}, stubID{}, stubClock{}, stubPublisher{})
	loginUc := usecase.NewLoginUserUseCase(nil, nil, nil, nil, nil, nil)
	h := NewRouter(uc, loginUc, NewVisitorLimiter(), nil)

	req := httptest.NewRequest(http.MethodPost, RegisterRoute, nil)
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "10.9.9.9:1234"
	// Reuse a fresh body per attempt.
	start := time.Now()
	rec := post(h, "10.9.9.9:1234", `{"email":"a@b.com","password":"Learn123"}`)
	elapsed := time.Since(start)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", rec.Code)
	}
	if elapsed > 3*time.Second {
		t.Fatalf("registration took %v, want < 3s", elapsed)
	}
	_ = req
}
