package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"lms-english/identity-service/internal/domain"
)

// listerStub records the filter it was handed and returns a fixed page.
type listerStub struct {
	page   UserPage
	err    error
	filter ListFilter
	calls  int
}

func (f *listerStub) ListUsers(_ context.Context, filter ListFilter) (UserPage, error) {
	f.calls++
	f.filter = filter
	return f.page, f.err
}

// TestListUsersAppliesDefaults verifies page 1 and limit 20 when nothing is named.
func TestListUsersAppliesDefaults(t *testing.T) {
	lister := &listerStub{page: UserPage{Total: 3}}
	result, err := NewListUsersUseCase(lister).Execute(context.Background(), ListUsersQuery{})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if result.Page != 1 || result.Limit != domain.DefaultUserPageSize {
		t.Errorf("page/limit = %d/%d, want 1/%d", result.Page, result.Limit, domain.DefaultUserPageSize)
	}
	if lister.filter.Offset != 0 || lister.filter.Limit != domain.DefaultUserPageSize {
		t.Errorf("filter = %+v, want offset 0 limit 20", lister.filter)
	}
	if result.Total != 3 {
		t.Errorf("Total = %d, want 3", result.Total)
	}
}

// TestListUsersComputesOffset verifies page two starts where page one ended.
func TestListUsersComputesOffset(t *testing.T) {
	lister := &listerStub{}
	if _, err := NewListUsersUseCase(lister).Execute(context.Background(), ListUsersQuery{Page: 3, Limit: 25, HasPage: true, HasLimit: true}); err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if lister.filter.Offset != 50 {
		t.Errorf("offset = %d, want 50", lister.filter.Offset)
	}
}

// TestListUsersNormalizesFilters verifies the search is lowercased and trimmed.
func TestListUsersNormalizesFilters(t *testing.T) {
	lister := &listerStub{}
	if _, err := NewListUsersUseCase(lister).Execute(context.Background(), ListUsersQuery{
		Role: "ADMIN", CEFRLevel: "B2", Search: "  Ana@Example ",
	}); err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if lister.filter.Search != "ana@example" {
		t.Errorf("search = %q, want lowercased and trimmed", lister.filter.Search)
	}
	if lister.filter.Role != "ADMIN" || lister.filter.CEFRLevel != "B2" {
		t.Errorf("filter = %+v, want role and level preserved", lister.filter)
	}
}

// TestListUsersRefusesBadParameters verifies each refusal never reaches the lister.
func TestListUsersRefusesBadParameters(t *testing.T) {
	cases := map[string]ListUsersQuery{
		"page zero":     {Page: -1, HasPage: true},
		"limit too big": {Limit: 101, HasLimit: true},
		"unknown role":  {Role: "WIZARD"},
	}
	for name, query := range cases {
		t.Run(name, func(t *testing.T) {
			lister := &listerStub{}
			_, err := NewListUsersUseCase(lister).Execute(context.Background(), query)
			if !errors.Is(err, domain.ErrInvalidFilter) {
				t.Fatalf("error = %v, want ErrInvalidFilter", err)
			}
			if lister.calls != 0 {
				t.Error("lister was called for an invalid request")
			}
		})
	}
}

// TestListUsersPassesItemsThrough verifies the result carries the page items.
func TestListUsersPassesItemsThrough(t *testing.T) {
	now := time.Date(2026, 11, 20, 9, 0, 0, 0, time.UTC)
	lister := &listerStub{page: UserPage{
		Items: []UserSummary{{ID: "u1", Email: "a@b.com", Role: domain.RoleStudent, CreatedAt: now}},
		Total: 1,
	}}
	result, err := NewListUsersUseCase(lister).Execute(context.Background(), ListUsersQuery{})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(result.Items) != 1 || result.Items[0].ID != "u1" {
		t.Fatalf("items = %+v, want the seeded summary", result.Items)
	}
}
