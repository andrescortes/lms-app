package sqlite

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"lms-english/identity-service/internal/domain"
	"lms-english/identity-service/internal/usecase"
)

// seedListedUser registers one account with a chosen role and learning level.
func seedListedUser(t *testing.T, db *sql.DB, id, email, name string, role domain.Role, level *domain.CEFRLevel, createdAt time.Time) {
	t.Helper()
	user := domain.User{
		ID:           id,
		Email:        email,
		PasswordHash: "$2a$04$seedhashseedhashseedhashseedhashseedhashseedhashseedXXXX",
		DisplayName:  name,
		Role:         role,
		CEFRLevel:    level,
		CreatedAt:    createdAt,
	}
	seedRefreshUser(t, db, user)
}

func level(value string) *domain.CEFRLevel {
	l := domain.CEFRLevel(value)
	return &l
}

// seedListing populates a stable set of accounts for the list tests.
func seedListing(t *testing.T) *sql.DB {
	t.Helper()
	db := newMigratedMemoryDB(t)
	base := time.Date(2026, 11, 20, 9, 0, 0, 0, time.UTC)
	seedListedUser(t, db, "u-a", "ana@example.com", "Ana Lopez", domain.RoleStudent, level("B2"), base.Add(3*time.Hour))
	seedListedUser(t, db, "u-b", "ben@example.com", "Ben Smith", domain.RoleTeacher, level("C1"), base.Add(2*time.Hour))
	seedListedUser(t, db, "u-c", "cara@other.com", "Cara Diaz", domain.RoleStudent, nil, base.Add(1*time.Hour))
	seedListedUser(t, db, "u-d", "admin@example.com", "Admin", domain.RoleAdmin, nil, base)
	return db
}

// TestListUsersNewestFirst verifies ordering and the total.
func TestListUsersNewestFirst(t *testing.T) {
	db := seedListing(t)

	page, err := NewUserLister(db).ListUsers(context.Background(), usecase.ListFilter{Limit: 20})
	if err != nil {
		t.Fatalf("ListUsers() error = %v", err)
	}
	if page.Total != 4 {
		t.Errorf("Total = %d, want 4", page.Total)
	}
	want := []string{"u-a", "u-b", "u-c", "u-d"}
	for i, id := range want {
		if page.Items[i].ID != id {
			t.Fatalf("position %d = %q, want %q (newest first)", i, page.Items[i].ID, id)
		}
	}
}

// TestListUsersRoleFilter verifies the filter narrows the page and the total.
func TestListUsersRoleFilter(t *testing.T) {
	db := seedListing(t)

	page, err := NewUserLister(db).ListUsers(context.Background(), usecase.ListFilter{Role: "STUDENT", Limit: 20})
	if err != nil {
		t.Fatalf("ListUsers() error = %v", err)
	}
	if page.Total != 2 || len(page.Items) != 2 {
		t.Fatalf("total/items = %d/%d, want 2/2", page.Total, len(page.Items))
	}
	for _, item := range page.Items {
		if item.Role != domain.RoleStudent {
			t.Errorf("item %q has role %v, want STUDENT", item.ID, item.Role)
		}
	}
}

// TestListUsersLevelFilter verifies level filtering, including that an unset level
// is excluded from a level-filtered result.
func TestListUsersLevelFilter(t *testing.T) {
	db := seedListing(t)

	page, err := NewUserLister(db).ListUsers(context.Background(), usecase.ListFilter{CEFRLevel: "C1", Limit: 20})
	if err != nil {
		t.Fatalf("ListUsers() error = %v", err)
	}
	if page.Total != 1 || len(page.Items) != 1 || page.Items[0].ID != "u-b" {
		t.Fatalf("page = %+v, want only u-b", page)
	}
}

// TestListUsersSearchIsCaseInsensitive verifies a fragment matches email and
// display name regardless of letter case.
func TestListUsersSearchIsCaseInsensitive(t *testing.T) {
	db := seedListing(t)
	lister := NewUserLister(db)

	byEmail, err := lister.ListUsers(context.Background(), usecase.ListFilter{Search: "ana@", Limit: 20})
	if err != nil {
		t.Fatalf("search by email: %v", err)
	}
	if byEmail.Total != 1 || byEmail.Items[0].ID != "u-a" {
		t.Fatalf("email search = %+v, want u-a", byEmail)
	}

	byName, err := lister.ListUsers(context.Background(), usecase.ListFilter{Search: "cara", Limit: 20})
	if err != nil {
		t.Fatalf("search by name: %v", err)
	}
	if byName.Total != 1 || byName.Items[0].ID != "u-c" {
		t.Fatalf("name search = %+v, want u-c", byName)
	}

	broad, err := lister.ListUsers(context.Background(), usecase.ListFilter{Search: "example.com", Limit: 20})
	if err != nil {
		t.Fatalf("broad search: %v", err)
	}
	if broad.Total != 3 {
		t.Errorf("broad search total = %d, want 3", broad.Total)
	}
}

// TestListUsersCombinedFilters verifies role and search apply together.
func TestListUsersCombinedFilters(t *testing.T) {
	db := seedListing(t)

	page, err := NewUserLister(db).ListUsers(context.Background(), usecase.ListFilter{
		Role: "STUDENT", Search: "example.com", Limit: 20,
	})
	if err != nil {
		t.Fatalf("ListUsers() error = %v", err)
	}
	if page.Total != 1 || page.Items[0].ID != "u-a" {
		t.Fatalf("page = %+v, want only u-a", page)
	}
}

// TestListUsersEmptyResult verifies a filter that matches nothing.
func TestListUsersEmptyResult(t *testing.T) {
	db := seedListing(t)

	page, err := NewUserLister(db).ListUsers(context.Background(), usecase.ListFilter{Search: "zzz", Limit: 20})
	if err != nil {
		t.Fatalf("ListUsers() error = %v", err)
	}
	if page.Total != 0 || len(page.Items) != 0 {
		t.Fatalf("page = %+v, want empty with total zero", page)
	}
}

// TestListUsersPagingIsStable verifies walking the list in pages returns every
// account once, newest first.
func TestListUsersPagingIsStable(t *testing.T) {
	db := seedListing(t)
	lister := NewUserLister(db)

	seen := map[string]int{}
	var order []string
	for page := 0; page < 3; page++ {
		result, err := lister.ListUsers(context.Background(), usecase.ListFilter{Limit: 2, Offset: page * 2})
		if err != nil {
			t.Fatalf("page %d: %v", page, err)
		}
		if result.Total != 4 {
			t.Errorf("page %d total = %d, want 4", page, result.Total)
		}
		for _, item := range result.Items {
			seen[item.ID]++
			order = append(order, item.ID)
		}
	}

	if len(order) != 4 {
		t.Fatalf("walked %d accounts, want 4", len(order))
	}
	for id, count := range seen {
		if count != 1 {
			t.Errorf("account %q appeared %d times", id, count)
		}
	}
	wantOrder := []string{"u-a", "u-b", "u-c", "u-d"}
	for i, id := range wantOrder {
		if order[i] != id {
			t.Errorf("walk position %d = %q, want %q", i, order[i], id)
		}
	}

	beyond, err := lister.ListUsers(context.Background(), usecase.ListFilter{Limit: 20, Offset: 100})
	if err != nil {
		t.Fatalf("page past the end: %v", err)
	}
	if len(beyond.Items) != 0 || beyond.Total != 4 {
		t.Errorf("page past the end = %+v, want empty items and total 4", beyond)
	}
}
