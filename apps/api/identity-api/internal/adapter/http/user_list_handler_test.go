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

// listStub answers with a fixed page and records the filter it was handed.
type listStub struct {
	page   usecase.UserPage
	err    error
	filter usecase.ListFilter
	calls  int
}

func (f *listStub) ListUsers(_ context.Context, filter usecase.ListFilter) (usecase.UserPage, error) {
	f.calls++
	f.filter = filter
	return f.page, f.err
}

// stubVerifierWithRole trusts any token and reports the given role.
type stubVerifierWithRole struct{ role string }

func (v stubVerifierWithRole) Verify(context.Context, string) (usecase.TokenClaims, error) {
	return usecase.TokenClaims{UserID: "u-admin", Role: v.role}, nil
}

// adminRouter mounts the list route behind the real middlewares with a fixed caller role.
func adminRouter(repo *listStub, role string) http.Handler {
	return NewRouter(
		nil, nil, nil, nil, nil, nil,
		usecase.NewListUsersUseCase(repo),
		stubVerifierWithRole{role: role},
		NewVisitorLimiter(),
		nil,
	)
}

// getList issues the list request with the given query string.
func getList(h http.Handler, query string) *httptest.ResponseRecorder {
	target := UsersRoute
	if query != "" {
		target += "?" + query
	}
	req := httptest.NewRequest(http.MethodGet, target, nil)
	req.Header.Set("Authorization", "Bearer good-token")
	req.RemoteAddr = "192.0.2.121:1"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func sampleSummaryPage() usecase.UserPage {
	level := domain.CEFRLevel("B2")
	return usecase.UserPage{
		Total: 1,
		Items: []usecase.UserSummary{{
			ID:          "u-a",
			Email:       "ana@example.com",
			DisplayName: "Ana",
			Role:        domain.RoleStudent,
			CEFRLevel:   &level,
			CreatedAt:   time.Date(2026, 11, 20, 9, 0, 0, 0, time.UTC),
		}},
	}
}

// TestListUsersRequestHelper verifies the roles reach the route as expected.
func TestListUsersReturnsPage(t *testing.T) {
	h := adminRouter(&listStub{page: sampleSummaryPage()}, "ADMIN")

	rec := getList(h, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}

	var got struct {
		Items []map[string]any `json:"items"`
		Total int              `json:"total"`
		Page  int              `json:"page"`
		Limit int              `json:"limit"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Total != 1 || got.Page != 1 || got.Limit != domain.DefaultUserPageSize {
		t.Errorf("total/page/limit = %d/%d/%d", got.Total, got.Page, got.Limit)
	}
	if len(got.Items) != 1 {
		t.Fatalf("items = %d, want 1", len(got.Items))
	}
	for _, field := range []string{"id", "email", "displayName", "role", "cefrLevel", "createdAt"} {
		if _, ok := got.Items[0][field]; !ok {
			t.Errorf("item is missing %q", field)
		}
	}
	body := strings.ToLower(rec.Body.String())
	for _, leak := range []string{"password", "hash"} {
		if strings.Contains(body, leak) {
			t.Errorf("response leaks %q", leak)
		}
	}
}

// TestListUsersPassesQueryParams verifies the query string reaches the use case.
func TestListUsersPassesQueryParams(t *testing.T) {
	repo := &listStub{page: sampleSummaryPage()}
	h := adminRouter(repo, "ADMIN")

	rec := getList(h, "role=STUDENT&cefrLevel=B2&q=%20Ana%20&page=2&limit=5")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if repo.filter.Role != "STUDENT" || repo.filter.CEFRLevel != "B2" || repo.filter.Search != "ana" {
		t.Errorf("filter = %+v, want role, level, and normalised search", repo.filter)
	}
	if repo.filter.Offset != 5 || repo.filter.Limit != 5 {
		t.Errorf("offset/limit = %d/%d, want 5/5", repo.filter.Offset, repo.filter.Limit)
	}
}

// TestListUsersRequiresBearer verifies an unauthenticated request is refused.
func TestListUsersRequiresBearer(t *testing.T) {
	repo := &listStub{page: sampleSummaryPage()}
	h := NewRouter(
		nil, nil, nil, nil, nil, nil,
		usecase.NewListUsersUseCase(repo),
		&stubVerifier{err: domain.ErrInvalidToken},
		NewVisitorLimiter(),
		nil,
	)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, UsersRoute, nil))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	if repo.calls != 0 {
		t.Error("list ran for an unauthenticated caller")
	}
}

// TestListUsersForbidsNonAdmin verifies a STUDENT caller gets 403 and no data.
func TestListUsersForbidsNonAdmin(t *testing.T) {
	repo := &listStub{page: sampleSummaryPage()}
	h := adminRouter(repo, "STUDENT")

	rec := getList(h, "")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403; body=%s", rec.Code, rec.Body.String())
	}
	assertEnvelope(t, rec.Body.Bytes(), codeForbidden)
	if repo.calls != 0 {
		t.Error("list ran for a non-administrator")
	}
	if strings.Contains(rec.Body.String(), "ana@example.com") {
		t.Error("refusal leaked account data")
	}
}

// TestListUsersNonNumericParamReturns400 verifies a bad number is a bad request.
func TestListUsersNonNumericParamReturns400(t *testing.T) {
	h := adminRouter(&listStub{page: sampleSummaryPage()}, "ADMIN")

	for _, query := range []string{"page=abc", "limit=x"} {
		rec := getList(h, query)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400", query, rec.Code)
		}
	}
}

// TestListUsersInvalidParametersReturn422 verifies bad bounds and unknown roles.
func TestListUsersInvalidParametersReturn422(t *testing.T) {
	cases := map[string]string{
		"page zero":     "page=0",
		"limit zero":    "limit=0",
		"limit too big": "limit=101",
		"unknown role":  "role=WIZARD",
	}
	for name, query := range cases {
		t.Run(name, func(t *testing.T) {
			repo := &listStub{page: sampleSummaryPage()}
			h := adminRouter(repo, "ADMIN")
			rec := getList(h, query)

			if rec.Code != http.StatusUnprocessableEntity {
				t.Fatalf("status = %d, want 422; body=%s", rec.Code, rec.Body.String())
			}
			if repo.calls != 0 {
				t.Error("list ran for an invalid request")
			}
		})
	}
}
