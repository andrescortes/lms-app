package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"lms-english/identity-service/internal/domain"
	"lms-english/identity-service/internal/usecase"
)

// roleUpdatedStub records the update and answers with a fixed user or error.
type roleUpdatedStub struct {
	user   domain.User
	err    error
	calls  int
	role   domain.Role
	userID string
}

func (f *roleUpdatedStub) UpdateUserRole(_ context.Context, userID string, role domain.Role) (domain.User, error) {
	f.calls++
	f.userID = userID
	f.role = role
	return f.user, f.err
}

func roleTargetUser() domain.User {
	return domain.User{
		ID: "target-1", Email: "target@example.com", DisplayName: "Target",
		Role: domain.RoleTeacher, CreatedAt: time.Date(2026, 11, 20, 9, 0, 0, 0, time.UTC),
	}
}

// roleRouter mounts the role route with the real middlewares and a fixed caller id.
func roleRouter(repo *roleUpdatedStub, callerID, callerRole string) http.Handler {
	return NewRouter(
		nil, nil, nil, nil, nil, nil, nil,
		usecase.NewChangeUserRoleUseCase(repo),
		stubVerifierWithRole{userID: callerID, role: callerRole},
		NewVisitorLimiter(),
		nil,
	)
}

// roleRequest issues a PATCH to the role route for one target id.
func roleRequest(h http.Handler, target, body string) *httptest.ResponseRecorder {
	path := strings.Replace(UserRoleRoute, "{id}", target, 1)
	req := httptest.NewRequest(http.MethodPatch, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer good-token")
	req.RemoteAddr = "192.0.2.130:1"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// TestChangeRoleSuccessReturnsAccount verifies the 200 shape and that no password
// material appears.
func TestChangeRoleSuccessReturnsAccount(t *testing.T) {
	repo := &roleUpdatedStub{user: roleTargetUser()}
	h := roleRouter(repo, "admin-1", "ADMIN")

	rec := roleRequest(h, "target-1", `{"role":"TEACHER"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	for _, field := range []string{"id", "email", "displayName", "role", "createdAt"} {
		if _, ok := got[field]; !ok {
			t.Errorf("response is missing %q", field)
		}
	}
	if repo.role != domain.RoleTeacher || repo.userID != "target-1" {
		t.Errorf("update = (%q, %v), want target-1 as TEACHER", repo.userID, repo.role)
	}
	if strings.Contains(strings.ToLower(rec.Body.String()), "password") {
		t.Error("response leaks password material")
	}
}

// TestChangeRoleUsesTokenIdentity verifies the actor comes from the token, so an
// administrator cannot bypass the self-change guard by naming another id.
func TestChangeRoleUsesTokenIdentity(t *testing.T) {
	repo := &roleUpdatedStub{user: roleTargetUser()}
	h := roleRouter(repo, "admin-1", "ADMIN")

	rec := roleRequest(h, "admin-1", `{"role":"STUDENT","actorUserId":"someone-else"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 for the unknown actorUserId field", rec.Code)
	}

	rec = roleRequest(h, "admin-1", `{"role":"STUDENT"}`)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 for the actor's own account", rec.Code)
	}
	assertEnvelope(t, rec.Body.Bytes(), codeForbidden)
	if repo.calls != 0 {
		t.Error("repository was called for a self-change")
	}
}

// TestChangeRoleRefusesUnknownRole verifies a bad role is a 422 with the repo untouched.
func TestChangeRoleRefusesUnknownRole(t *testing.T) {
	for _, role := range []string{"WIZARD", "admin"} {
		repo := &roleUpdatedStub{user: roleTargetUser()}
		h := roleRouter(repo, "admin-1", "ADMIN")
		body := fmt.Sprintf(`{"role":%q}`, role)
		rec := roleRequest(h, "target-1", body)
		if rec.Code != http.StatusUnprocessableEntity {
			t.Errorf("role %q: status = %d, want 422", role, rec.Code)
		}
		if repo.calls != 0 {
			t.Errorf("role %q: repository was called", role)
		}
	}
}

// TestChangeRoleRefusesMalformedBody verifies malformed and over-specified bodies.
func TestChangeRoleRefusesMalformedBody(t *testing.T) {
	for name, body := range map[string]string{
		"truncated":     `{"role":`,
		"unknown field": `{"role":"STUDENT","notes":"x"}`,
	} {
		h := roleRouter(&roleUpdatedStub{user: roleTargetUser()}, "admin-1", "ADMIN")
		rec := roleRequest(h, "target-1", body)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400", name, rec.Code)
		}
	}
}

// TestChangeRoleRequiresBearer verifies an unauthenticated request is refused.
func TestChangeRoleRequiresBearer(t *testing.T) {
	repo := &roleUpdatedStub{user: roleTargetUser()}
	h := NewRouter(
		nil, nil, nil, nil, nil, nil, nil,
		usecase.NewChangeUserRoleUseCase(repo),
		&stubVerifier{err: domain.ErrInvalidToken},
		NewVisitorLimiter(),
		nil,
	)
	path := strings.Replace(UserRoleRoute, "{id}", "target-1", 1)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPatch, path, strings.NewReader(`{"role":"TEACHER"}`)))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	if repo.calls != 0 {
		t.Error("repository was called for an unauthenticated caller")
	}
}

// TestChangeRoleForbidsNonAdmin verifies a learner cannot assign roles.
func TestChangeRoleForbidsNonAdmin(t *testing.T) {
	repo := &roleUpdatedStub{user: roleTargetUser()}
	h := roleRouter(repo, "learner-1", "STUDENT")

	rec := roleRequest(h, "target-1", `{"role":"ADMIN"}`)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
	assertEnvelope(t, rec.Body.Bytes(), codeForbidden)
	if repo.calls != 0 {
		t.Error("repository was called for a non-administrator")
	}
}

// TestChangeRoleUnknownAccountReturns404 verifies the not-found mapping.
func TestChangeRoleUnknownAccountReturns404(t *testing.T) {
	repo := &roleUpdatedStub{err: domain.ErrNotFound}
	h := roleRouter(repo, "admin-1", "ADMIN")

	rec := roleRequest(h, "gone", `{"role":"TEACHER"}`)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", rec.Code, rec.Body.String())
	}
	assertEnvelope(t, rec.Body.Bytes(), codeNotFound)
}
