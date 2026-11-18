package sqlite

import (
	"context"
	"errors"
	"testing"

	"lms-english/identity-service/internal/domain"
)

// TestDuplicateEmailCaseInsensitive verifies that a differently-cased duplicate
// (normalized to the same stored email) is mapped to the domain sentinel and the
// loser persists nothing.
func TestDuplicateEmailCaseInsensitive(t *testing.T) {
	ctx := context.Background()
	db := newMigratedMemoryDB(t)
	store := NewRegistrationStore(db)

	first := sampleUser("user-1", "student@example.com")
	if err := store.RegisterUser(ctx, first, domain.NewEmptyProfile(first.ID), sampleEvent(first)); err != nil {
		t.Fatalf("seed RegisterUser() error = %v", err)
	}

	// The use case normalizes before calling the store; simulate that here.
	dup := sampleUser("user-2", domain.NormalizeEmail("Student@Example.COM"))
	err := store.RegisterUser(ctx, dup, domain.NewEmptyProfile(dup.ID), sampleEvent(dup))
	if !errors.Is(err, domain.ErrEmailAlreadyExists) {
		t.Fatalf("error = %v, want ErrEmailAlreadyExists", err)
	}

	var users, profiles, events int
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&users)
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM profiles`).Scan(&profiles)
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM registration_events`).Scan(&events)
	if users != 1 || profiles != 1 || events != 1 {
		t.Errorf("after case-insensitive duplicate: users=%d profiles=%d events=%d, want 1/1/1", users, profiles, events)
	}
}

// TestConcurrentDuplicateSingleWinner verifies that two concurrent inserts of the
// same email yield exactly one success and one conflict, with one persisted
// account and one outbox event (the winner's only).
func TestConcurrentDuplicateSingleWinner(t *testing.T) {
	ctx := context.Background()
	db := newMigratedMemoryDB(t)
	store := NewRegistrationStore(db)

	type outcome struct{ err error }
	results := make(chan outcome, 2)

	go func() {
		u := sampleUser("user-1", "race@example.com")
		results <- outcome{store.RegisterUser(ctx, u, domain.NewEmptyProfile(u.ID), sampleEvent(u))}
	}()
	go func() {
		u := sampleUser("user-2", "race@example.com")
		results <- outcome{store.RegisterUser(ctx, u, domain.NewEmptyProfile(u.ID), sampleEvent(u))}
	}()

	var ok, conflict int
	for i := 0; i < 2; i++ {
		res := <-results
		switch {
		case res.err == nil:
			ok++
		case errors.Is(res.err, domain.ErrEmailAlreadyExists):
			conflict++
		default:
			t.Fatalf("unexpected error: %v", res.err)
		}
	}
	if ok != 1 || conflict != 1 {
		t.Fatalf("ok=%d conflict=%d, want 1/1", ok, conflict)
	}

	var users, events int
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&users)
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM registration_events`).Scan(&events)
	if users != 1 || events != 1 {
		t.Errorf("users=%d events=%d, want 1/1", users, events)
	}
}
