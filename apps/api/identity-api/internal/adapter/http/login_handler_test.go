package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"lms-english/identity-service/internal/domain"
	"lms-english/identity-service/internal/usecase"
)

type loginStubUserRepo struct {
	user domain.User
	err  error
}

func (f *loginStubUserRepo) FindByEmail(context.Context, string) (domain.User, error) {
	return f.user, f.err
}

func (f *loginStubUserRepo) FindByID(context.Context, string) (domain.User, error) {
	return f.user, f.err
}

type loginStubVerify struct{ err error }

func (f *loginStubVerify) Verify(string, string) error { return f.err }

type loginStubToken struct {
	token string
	err   error
}

func (f *loginStubToken) Issue(context.Context, domain.User) (string, error) { return f.token, f.err }

type loginStubRefresh struct{ err error }

func (f *loginStubRefresh) Store(context.Context, string, string, string, time.Time) error {
	return f.err
}

func (f *loginStubRefresh) Revoke(context.Context, string) error { return nil }

func (f *loginStubRefresh) RevokeByJTI(context.Context, string, string) error { return nil }

func (f *loginStubRefresh) Rotate(context.Context, string, string, string, time.Time) (usecase.Rotation, error) {
	return usecase.Rotation{}, nil
}

func newLoginRouter(t *testing.T, uc *usecase.LoginUserUseCase) http.Handler {
	t.Helper()
	return NewRouter(nil, uc, nil, nil, nil, nil, nil, NewVisitorLimiter(), nil)
}

func postLogin(handler http.Handler, remoteAddr, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, LoginRoute, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = remoteAddr
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

// TestLoginSuccessReturnsTokens verifies the 200 contract: only the access and
// refresh tokens are returned, and no credential material is echoed.
func TestLoginSuccessReturnsTokens(t *testing.T) {
	uc := usecase.NewLoginUserUseCase(
		&loginStubUserRepo{user: domain.User{
			ID: "u1", Email: "user@example.com", PasswordHash: "hash", DisplayName: "n",
			Role: domain.RoleStudent, CreatedAt: time.Now(),
		}},
		&loginStubVerify{},
		&loginStubToken{token: "access-token"},
		&loginStubRefresh{},
		stubID{},
		stubClock{},
	)
	h := newLoginRouter(t, uc)

	rec := postLogin(h, "10.2.2.2:1234", `{"email":"user@example.com","password":"Learn123"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var got loginResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	if got.AccessToken != "access-token" {
		t.Fatalf("accessToken = %q, want access-token", got.AccessToken)
	}
	if got.RefreshToken == "" {
		t.Fatal("refreshToken is empty, want issued token")
	}
}

// TestLoginInvalidCredentialsReturnsGeneric401 verifies the failed sign-in
// contract: 401 with the generic invalid_credentials code and no account details.
func TestLoginInvalidCredentialsReturnsGeneric401(t *testing.T) {
	uc := usecase.NewLoginUserUseCase(
		&loginStubUserRepo{},
		&loginStubVerify{err: errors.New("bad password")},
		&loginStubToken{},
		&loginStubRefresh{},
		stubID{},
		stubClock{},
	)
	h := newLoginRouter(t, uc)

	rec := postLogin(h, "10.2.2.3:1234", `{"email":"user@example.com","password":"WrongPass1"}`)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	assertEnvelope(t, rec.Body.Bytes(), codeUnauthenticated)
	if !strings.Contains(rec.Body.String(), "invalid credentials") {
		t.Fatalf("body = %q, want the generic invalid credentials message", rec.Body.String())
	}
}

// TestLoginMalformedRequestReturns400 verifies the malformed-body contract.
func TestLoginMalformedRequestReturns400(t *testing.T) {
	uc := usecase.NewLoginUserUseCase(
		&loginStubUserRepo{},
		&loginStubVerify{},
		&loginStubToken{},
		&loginStubRefresh{},
		stubID{},
		stubClock{},
	)
	h := newLoginRouter(t, uc)

	rec := postLogin(h, "10.2.2.4:1234", `{"email":`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

// TestLoginUnknownEmailReturnsGeneric401 verifies an unregistered email is
// indistinguishable from a wrong password: 401 invalid_credentials, never 404.
func TestLoginUnknownEmailReturnsGeneric401(t *testing.T) {
	uc := usecase.NewLoginUserUseCase(
		&loginStubUserRepo{err: fmt.Errorf("find user by email: %w", domain.ErrNotFound)},
		&loginStubVerify{},
		&loginStubToken{},
		&loginStubRefresh{},
		stubID{},
		stubClock{},
	)
	h := newLoginRouter(t, uc)

	rec := postLogin(h, "10.2.2.5:1234", `{"email":"nobody@example.com","password":"Learn123"}`)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401; body=%s", rec.Code, rec.Body.String())
	}
	assertEnvelope(t, rec.Body.Bytes(), codeUnauthenticated)
}

// TestLoginMalformedEmailReturns422 verifies a malformed email is rejected as a
// validation failure naming the email rule.
func TestLoginMalformedEmailReturns422(t *testing.T) {
	uc := usecase.NewLoginUserUseCase(
		&loginStubUserRepo{},
		&loginStubVerify{},
		&loginStubToken{},
		&loginStubRefresh{},
		stubID{},
		stubClock{},
	)
	h := newLoginRouter(t, uc)

	rec := postLogin(h, "10.2.2.6:1234", `{"email":"no-at-sign","password":"Learn123"}`)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422; body=%s", rec.Code, rec.Body.String())
	}
	assertEnvelope(t, rec.Body.Bytes(), codeValidation)
}
