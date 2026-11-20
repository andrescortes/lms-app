package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"lms-english/identity-service/internal/domain"
	"lms-english/identity-service/internal/usecase"
)

// profileRepoStub serves one view and records the change.
type profileRepoStub struct {
	view    usecase.ProfileView
	getErr  error
	setErr  error
	changes []usecase.ProfileChange
}

func (f *profileRepoStub) GetUserProfile(context.Context, string) (usecase.ProfileView, error) {
	return f.view, f.getErr
}

func (f *profileRepoStub) UpdateUserProfile(_ context.Context, _ string, change usecase.ProfileChange) error {
	f.changes = append(f.changes, change)
	return f.setErr
}

// profileUserStub serves one user for the password path.
type profileUserStub struct {
	user domain.User
	err  error
}

func (f *profileUserStub) FindByEmail(context.Context, string) (domain.User, error) {
	return f.user, nil
}

func (f *profileUserStub) FindByID(context.Context, string) (domain.User, error) {
	return f.user, f.err
}

// profileHasherStub records the password it hashed.
type profileHasherStub struct{ seen string }

func (f *profileHasherStub) Hash(password string) (string, error) {
	f.seen = password
	return "hashed:" + password, nil
}

func sampleView() usecase.ProfileView {
	return usecase.ProfileView{
		ID:          "u1",
		Email:       "learner@example.com",
		DisplayName: "Learner One",
		Role:        domain.RoleStudent,
		CreatedAt:   time.Date(2026, 11, 18, 10, 0, 0, 0, time.UTC),
	}
}

func newProfileRouter(repo *profileRepoStub, users *profileUserStub, verifyErr error) http.Handler {
	getUc := usecase.NewGetProfileUseCase(repo)
	updateUc := usecase.NewUpdateProfileUseCase(repo, users, &stubVerifier2{err: verifyErr}, &profileHasherStub{})
	return NewRouter(
		nil, nil, nil, nil,
		getUc, updateUc,
		&stubVerifier{claims: usecase.TokenClaims{UserID: "u1", Role: "STUDENT"}},
		NewVisitorLimiter(),
		nil,
	)
}

// stubVerifier2 answers password verification with a fixed result.
type stubVerifier2 struct{ err error }

func (f *stubVerifier2) Verify(string, string) error { return f.err }

func newProfileRouterUnauthenticated() http.Handler {
	repo := &profileRepoStub{view: sampleView()}
	getUc := usecase.NewGetProfileUseCase(repo)
	updateUc := usecase.NewUpdateProfileUseCase(repo, &profileUserStub{}, &stubVerifier2{}, &profileHasherStub{})
	return NewRouter(
		nil, nil, nil, nil,
		getUc, updateUc,
		&stubVerifier{err: domain.ErrInvalidToken},
		NewVisitorLimiter(),
		nil,
	)
}

// TestGetProfileReturnsAccountAndProfile verifies the 200 shape and that no
// password material reaches the body.
func TestGetProfileReturnsAccountAndProfile(t *testing.T) {
	r := newProfileRouter(&profileRepoStub{view: sampleView()}, &profileUserStub{}, nil)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, withBearer(httptest.NewRequest(http.MethodGet, ProfileRoute, nil), "Bearer good-token"))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	for _, field := range []string{"id", "email", "displayName", "role", "cefrLevel", "createdAt", "profile"} {
		if _, ok := got[field]; !ok {
			t.Errorf("response is missing %q", field)
		}
	}
	body := strings.ToLower(rec.Body.String())
	for _, leak := range []string{"password", "hash"} {
		if strings.Contains(body, leak) {
			t.Errorf("response leaks %q: %s", leak, body)
		}
	}
	profile, ok := got["profile"].(map[string]any)
	if !ok {
		t.Fatalf("profile is %T, want an object", got["profile"])
	}
	for _, field := range []string{"nativeLanguage", "goals", "timezone", "avatarUrl"} {
		if _, ok := profile[field]; !ok {
			t.Errorf("profile is missing %q", field)
		}
	}
}

// TestGetProfileMissingAccountReturns404 verifies the not-found mapping.
func TestGetProfileMissingAccountReturns404(t *testing.T) {
	r := newProfileRouter(&profileRepoStub{getErr: domain.ErrNotFound}, &profileUserStub{}, nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, withBearer(httptest.NewRequest(http.MethodGet, ProfileRoute, nil), "Bearer good-token"))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

// TestProfileRoutesRequireBearer verifies both routes refuse a missing or invalid
// credential with 401 and never reach the handler.
func TestProfileRoutesRequireBearer(t *testing.T) {
	unauthenticated := newProfileRouterUnauthenticated()
	for _, method := range []string{http.MethodGet, http.MethodPatch} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(method, ProfileRoute, strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "application/json")
		unauthenticated.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s without bearer: status = %d, want 401", method, rec.Code)
		}
	}
}

// TestPatchProfilePartialUpdate verifies only the named field is applied.
func TestPatchProfilePartialUpdate(t *testing.T) {
	repo := &profileRepoStub{view: sampleView()}
	r := newProfileRouter(repo, &profileUserStub{}, nil)

	req := httptest.NewRequest(http.MethodPatch, ProfileRoute, strings.NewReader(`{"goals":"Pass B2"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, withBearer(req, "Bearer good-token"))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	if len(repo.changes) != 1 {
		t.Fatalf("changes = %d, want 1", len(repo.changes))
	}
	change := repo.changes[0]
	if change.Goals == nil || *change.Goals != "Pass B2" {
		t.Errorf("Goals = %v, want Pass B2", change.Goals)
	}
	if change.Timezone != nil || change.DisplayName != nil {
		t.Errorf("unexpected fields set: %+v", change)
	}
}

// TestPatchProfileNullClearsField verifies an explicit null reaches the store as a
// clear, which a plain string field could not express.
func TestPatchProfileNullClearsField(t *testing.T) {
	repo := &profileRepoStub{view: sampleView()}
	r := newProfileRouter(repo, &profileUserStub{}, nil)

	req := httptest.NewRequest(http.MethodPatch, ProfileRoute, strings.NewReader(`{"timezone":null}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, withBearer(req, "Bearer good-token"))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	if len(repo.changes) != 1 || repo.changes[0].Timezone == nil || *repo.changes[0].Timezone != "" {
		t.Fatalf("changes = %+v, want timezone cleared", repo.changes)
	}
}

// TestPatchProfileMalformedRequestReturns400 verifies malformed and unknown fields.
func TestPatchProfileMalformedRequestReturns400(t *testing.T) {
	for name, body := range map[string]string{
		"truncated":     `{"goals":`,
		"unknown field": `{"nickname":"x"}`,
	} {
		t.Run(name, func(t *testing.T) {
			r := newProfileRouter(&profileRepoStub{view: sampleView()}, &profileUserStub{}, nil)
			req := httptest.NewRequest(http.MethodPatch, ProfileRoute, strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, withBearer(req, "Bearer good-token"))
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400", rec.Code)
			}
		})
	}
}

// TestPatchProfilePasswordRefusals verifies the 401 and 422 outcomes.
func TestPatchProfilePasswordRefusals(t *testing.T) {
	cases := []struct {
		name      string
		body      string
		verifyErr error
		want      int
		wantCode  string
	}{
		{"missing current", `{"newPassword":"NewPass1"}`, nil, http.StatusUnauthorized, codeUnauthenticated},
		{"wrong current", `{"currentPassword":"Wrong1aa","newPassword":"NewPass1"}`, errors.New("mismatch"), http.StatusUnauthorized, codeUnauthenticated},
		{"weak new", `{"currentPassword":"OldPass1","newPassword":"weak"}`, nil, http.StatusUnprocessableEntity, codeWeakPassword},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := newProfileRouter(&profileRepoStub{view: sampleView()}, &profileUserStub{}, tc.verifyErr)
			req := httptest.NewRequest(http.MethodPatch, ProfileRoute, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, withBearer(req, "Bearer good-token"))
			if rec.Code != tc.want {
				t.Fatalf("status = %d, want %d; body=%s", rec.Code, tc.want, rec.Body.String())
			}
			assertEnvelope(t, rec.Body.Bytes(), tc.wantCode)
		})
	}
}

// TestPatchProfileValidPasswordChange verifies the success path updates the hash.
func TestPatchProfileValidPasswordChange(t *testing.T) {
	repo := &profileRepoStub{view: sampleView()}
	r := newProfileRouter(repo, &profileUserStub{user: domain.User{ID: "u1", PasswordHash: "stored"}}, nil)

	req := httptest.NewRequest(http.MethodPatch, ProfileRoute, strings.NewReader(`{"currentPassword":"OldPass1","newPassword":"NewPass1"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, withBearer(req, "Bearer good-token"))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	if len(repo.changes) != 1 || repo.changes[0].PasswordHash == nil {
		t.Fatalf("changes = %+v, want a password hash", repo.changes)
	}
	if strings.Contains(rec.Body.String(), "NewPass1") {
		t.Fatal("response echoed the new password")
	}
}

// TestPatchProfileEmptyBodyChangesNothing verifies the edge case: an empty object
// leaves every field as it was and still answers with the current profile.
func TestPatchProfileEmptyBodyChangesNothing(t *testing.T) {
	repo := &profileRepoStub{view: sampleView()}
	r := newProfileRouter(repo, &profileUserStub{}, nil)

	req := httptest.NewRequest(http.MethodPatch, ProfileRoute, strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, withBearer(req, "Bearer good-token"))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	if len(repo.changes) != 1 {
		t.Fatalf("changes = %d, want 1", len(repo.changes))
	}
	change := repo.changes[0]
	if change.DisplayName != nil || change.Goals != nil || change.Timezone != nil ||
		change.NativeLanguage != nil || change.AvatarURL != nil || change.PasswordHash != nil {
		t.Fatalf("empty body produced a change: %+v", change)
	}
}
