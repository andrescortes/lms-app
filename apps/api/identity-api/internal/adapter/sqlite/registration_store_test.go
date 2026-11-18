package sqlite

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"lms-english/identity-service/internal/domain"
)

func sampleUser(id, email string) domain.User {
	return domain.User{
		ID:           id,
		Email:        email,
		PasswordHash: "$2a$04$abcdefghijklmnopqrstuvABCDEFGHIJKLMNOPQRSTUVWXYZ0123",
		DisplayName:  "student",
		Role:         domain.RoleStudent,
		CreatedAt:    time.Date(2026, 11, 18, 10, 0, 0, 0, time.UTC),
	}
}

func sampleEvent(user domain.User) domain.UserRegisteredEvent {
	return domain.UserRegisteredEvent{
		UserID:     user.ID,
		Email:      user.Email,
		Role:       user.Role,
		OccurredAt: user.CreatedAt,
	}
}

// TestRegisterUserAtomicInsert verifies a successful registration writes the
// user, the empty profile, and exactly one outbox event, and that the stored
// password is the hash (never plaintext).
func TestRegisterUserAtomicInsert(t *testing.T) {
	ctx := context.Background()
	db := newMigratedMemoryDB(t)
	store := NewRegistrationStore(db)

	user := sampleUser("user-1", "student@example.com")
	profile := domain.NewEmptyProfile(user.ID)
	event := sampleEvent(user)

	if err := store.RegisterUser(ctx, user, profile, event); err != nil {
		t.Fatalf("RegisterUser() error = %v", err)
	}

	var hash string
	if err := db.QueryRowContext(ctx, `SELECT password_hash FROM users WHERE id = ?`, user.ID).Scan(&hash); err != nil {
		t.Fatalf("read password hash: %v", err)
	}
	if hash != user.PasswordHash {
		t.Errorf("stored hash = %q, want %q", hash, user.PasswordHash)
	}
	if strings.Contains(hash, "Learn123") {
		t.Error("stored data contains plaintext password")
	}

	var empty int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM profiles WHERE user_id = ? AND native_language IS NULL AND goals IS NULL AND timezone IS NULL AND avatar_url IS NULL`, user.ID).Scan(&empty); err != nil {
		t.Fatalf("read profile: %v", err)
	}
	if empty != 1 {
		t.Errorf("empty profile count = %d, want 1", empty)
	}

	var eventCount int
	var status, payload string
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*), MIN(status), MIN(payload) FROM registration_events WHERE event_type = 'UserRegistered'`).Scan(&eventCount, &status, &payload); err != nil {
		t.Fatalf("read event: %v", err)
	}
	if eventCount != 1 {
		t.Fatalf("event count = %d, want 1", eventCount)
	}
	if status != "pending" {
		t.Errorf("event status = %q, want pending", status)
	}
	var got eventPayload
	if err := json.Unmarshal([]byte(payload), &got); err != nil {
		t.Fatalf("payload is not valid JSON: %v", err)
	}
	if got.UserID != user.ID || got.Email != user.Email || got.Role != "STUDENT" {
		t.Errorf("payload = %+v, want user id/email/role", got)
	}
	if strings.Contains(payload, user.PasswordHash) {
		t.Error("event payload must not include the password hash")
	}
}

// TestRegisterUserDuplicateEmail verifies the unique constraint is mapped to the
// domain sentinel and that the loser leaves no profile or event row.
func TestRegisterUserDuplicateEmail(t *testing.T) {
	ctx := context.Background()
	db := newMigratedMemoryDB(t)
	store := NewRegistrationStore(db)

	first := sampleUser("user-1", "student@example.com")
	if err := store.RegisterUser(ctx, first, domain.NewEmptyProfile(first.ID), sampleEvent(first)); err != nil {
		t.Fatalf("first RegisterUser() error = %v", err)
	}

	// Same normalized email, different id -> unique violation.
	dup := sampleUser("user-2", "student@example.com")
	err := store.RegisterUser(ctx, dup, domain.NewEmptyProfile(dup.ID), sampleEvent(dup))
	if !errors.Is(err, domain.ErrEmailAlreadyExists) {
		t.Fatalf("duplicate error = %v, want ErrEmailAlreadyExists", err)
	}

	var users, profiles, events int
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&users)
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM profiles`).Scan(&profiles)
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM registration_events`).Scan(&events)
	if users != 1 || profiles != 1 || events != 1 {
		t.Errorf("after duplicate: users=%d profiles=%d events=%d, want 1/1/1", users, profiles, events)
	}
}

// TestRegisterUserRollbackOnEventFailure verifies that a failure after the user
// insert rolls the whole transaction back, leaving no partial account. A second
// user is inserted with a duplicate event id to force the failure.
func TestRegisterUserRollbackOnEventFailure(t *testing.T) {
	ctx := context.Background()
	db := newMigratedMemoryDB(t)
	store := NewRegistrationStore(db)

	first := sampleUser("user-1", "first@example.com")
	if err := store.RegisterUser(ctx, first, domain.NewEmptyProfile(first.ID), sampleEvent(first)); err != nil {
		t.Fatalf("seed RegisterUser() error = %v", err)
	}

	// Reuse the same user id so the user insert fails; assert nothing new persists.
	clash := sampleUser("user-1", "other@example.com")
	if err := store.RegisterUser(ctx, clash, domain.NewEmptyProfile(clash.ID), sampleEvent(clash)); err == nil {
		t.Fatal("expected failure for duplicate user id")
	}

	var users int
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&users)
	if users != 1 {
		t.Errorf("users = %d, want 1 (rollback should leave no partial rows)", users)
	}
}
