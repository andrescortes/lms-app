package http

import (
	"net/http"
	"testing"

	"lms-english/identity-service/internal/domain"
	"lms-english/identity-service/internal/usecase"
)

// TestLoginRateLimitRejectsEleventhAttempt verifies the login route admits ten
// attempts per minute from one client and answers the eleventh with 429.
func TestLoginRateLimitRejectsEleventhAttempt(t *testing.T) {
	uc := usecase.NewLoginUserUseCase(
		&loginStubUserRepo{},
		&loginStubVerify{err: domain.ErrInvalidCredentials},
		&loginStubToken{},
		&loginStubRefresh{},
		stubID{},
		stubClock{},
	)
	h := newLoginRouter(t, uc)
	const addr = "192.0.2.30:1234"

	for i := 1; i <= loginAttemptsPerMinute; i++ {
		rec := postLogin(h, addr, `{"email":"user@example.com","password":"WrongPass1"}`)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d status = %d, want 401", i, rec.Code)
		}
	}
	rec := postLogin(h, addr, `{"email":"user@example.com","password":"WrongPass1"}`)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("eleventh attempt status = %d, want 429", rec.Code)
	}
}

// TestLoginRateLimitIsPerClient verifies one client hitting the limit does not
// block another.
func TestLoginRateLimitIsPerClient(t *testing.T) {
	uc := usecase.NewLoginUserUseCase(
		&loginStubUserRepo{},
		&loginStubVerify{err: domain.ErrInvalidCredentials},
		&loginStubToken{},
		&loginStubRefresh{},
		stubID{},
		stubClock{},
	)
	h := newLoginRouter(t, uc)

	for i := 0; i <= loginAttemptsPerMinute; i++ {
		postLogin(h, "192.0.2.40:1111", `{"email":"user@example.com","password":"WrongPass1"}`)
	}
	rec := postLogin(h, "192.0.2.41:2222", `{"email":"user@example.com","password":"WrongPass1"}`)
	if rec.Code == http.StatusTooManyRequests {
		t.Fatal("second client was rate limited by the first client's attempts")
	}
}
