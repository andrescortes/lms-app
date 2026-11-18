package usecase

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"lms-english/identity-service/internal/domain"
)

// ---- fakes ----

type fakeStore struct {
	gotUser    domain.User
	gotProfile domain.Profile
	gotEvent   domain.UserRegisteredEvent
	calls      int
	err        error
}

func (f *fakeStore) RegisterUser(_ context.Context, u domain.User, p domain.Profile, e domain.UserRegisteredEvent) error {
	f.calls++
	f.gotUser, f.gotProfile, f.gotEvent = u, p, e
	return f.err
}

type fakeHasher struct {
	hash    string
	err     error
	gotArgs string
}

func (f *fakeHasher) Hash(pw string) (string, error) {
	f.gotArgs = pw
	return f.hash, f.err
}

type fakeID struct{ id string }

func (f fakeID) New() string { return f.id }

type fakeClock struct{ now time.Time }

func (f fakeClock) Now() time.Time { return f.now }

type fakePublisher struct {
	called bool
	err    error
}

func (f *fakePublisher) Publish(context.Context, string, []byte) error {
	f.called = true
	return f.err
}

func newUseCase(store *fakeStore, pub *fakePublisher) *RegisterUserUseCase {
	return NewRegisterUserUseCase(
		store,
		&fakeHasher{hash: "hashed"},
		fakeID{id: "user-1"},
		fakeClock{now: time.Date(2026, 11, 18, 10, 0, 0, 0, time.UTC)},
		pub,
	)
}

// ---- tests ----

func TestExecuteSuccess(t *testing.T) {
	store := &fakeStore{}
	pub := &fakePublisher{}
	uc := newUseCase(store, pub)

	res, err := uc.Execute(context.Background(), RegisterUserCommand{
		Email:    "Student@Example.com",
		Password: "Learn123",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if res.UserID != "user-1" {
		t.Fatalf("UserID = %q, want user-1", res.UserID)
	}
	if store.gotUser.Email != "student@example.com" {
		t.Errorf("stored email = %q, want normalized", store.gotUser.Email)
	}
	if store.gotUser.Role != domain.RoleStudent {
		t.Errorf("stored role = %v, want STUDENT", store.gotUser.Role)
	}
	if store.gotUser.DisplayName != "student" {
		t.Errorf("display name = %q, want default local part", store.gotUser.DisplayName)
	}
	if store.gotUser.PasswordHash == "Learn123" || store.gotUser.PasswordHash == "" {
		t.Errorf("password hash = %q, want hashed value", store.gotUser.PasswordHash)
	}
	if store.gotProfile.UserID != "user-1" {
		t.Errorf("profile user id = %q, want user-1", store.gotProfile.UserID)
	}
	if store.gotEvent.UserID != "user-1" || store.gotEvent.Email != "student@example.com" || store.gotEvent.Role != domain.RoleStudent {
		t.Errorf("event = %+v, want user id/email/role populated", store.gotEvent)
	}
	// Delivery is the worker's responsibility; Execute must not call the publisher.
	if pub.called {
		t.Error("Execute invoked the publisher; delivery must be deferred to the worker")
	}
}

func TestExecuteUsesProvidedDisplayName(t *testing.T) {
	store := &fakeStore{}
	uc := newUseCase(store, &fakePublisher{})
	_, err := uc.Execute(context.Background(), RegisterUserCommand{
		Email:       "student@example.com",
		Password:    "Learn123",
		DisplayName: "Student One",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if store.gotUser.DisplayName != "Student One" {
		t.Errorf("display name = %q, want provided value", store.gotUser.DisplayName)
	}
}

func TestExecuteInvalidEmail(t *testing.T) {
	store := &fakeStore{}
	uc := newUseCase(store, &fakePublisher{})
	_, err := uc.Execute(context.Background(), RegisterUserCommand{Email: "not-an-email", Password: "Learn123"})
	if !errors.Is(err, domain.ErrInvalidEmail) {
		t.Fatalf("error = %v, want ErrInvalidEmail", err)
	}
	if store.calls != 0 {
		t.Error("store was called for an invalid email")
	}
}

func TestExecuteWeakPassword(t *testing.T) {
	store := &fakeStore{}
	uc := newUseCase(store, &fakePublisher{})
	_, err := uc.Execute(context.Background(), RegisterUserCommand{Email: "student@example.com", Password: "weak"})
	if !errors.Is(err, domain.ErrWeakPassword) {
		t.Fatalf("error = %v, want ErrWeakPassword", err)
	}
	if store.calls != 0 {
		t.Error("store was called for a weak password")
	}
}

func TestExecuteHashFailure(t *testing.T) {
	store := &fakeStore{}
	uc := NewRegisterUserUseCase(
		store,
		&fakeHasher{err: errors.New("boom")},
		fakeID{id: "user-1"},
		fakeClock{now: time.Now()},
		&fakePublisher{},
	)
	_, err := uc.Execute(context.Background(), RegisterUserCommand{Email: "student@example.com", Password: "Learn123"})
	if err == nil {
		t.Fatal("expected error from hash failure")
	}
	if store.calls != 0 {
		t.Error("store was called despite hash failure")
	}
}

func TestExecuteStoreFailurePropagates(t *testing.T) {
	store := &fakeStore{err: domain.ErrEmailAlreadyExists}
	uc := newUseCase(store, &fakePublisher{})
	_, err := uc.Execute(context.Background(), RegisterUserCommand{Email: "student@example.com", Password: "Learn123"})
	if !errors.Is(err, domain.ErrEmailAlreadyExists) {
		t.Fatalf("error = %v, want ErrEmailAlreadyExists (wrapped)", err)
	}
}

func TestExecutePublisherFailureDoesNotAffectResult(t *testing.T) {
	store := &fakeStore{}
	pub := &fakePublisher{err: errors.New("hook down")}
	uc := newUseCase(store, pub)
	res, err := uc.Execute(context.Background(), RegisterUserCommand{Email: "student@example.com", Password: "Learn123"})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil despite publisher failure", err)
	}
	if res.UserID == "" {
		t.Fatal("expected a user id")
	}
}

func TestExecuteContextCancelled(t *testing.T) {
	store := &fakeStore{}
	uc := newUseCase(store, &fakePublisher{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := uc.Execute(ctx, RegisterUserCommand{Email: "student@example.com", Password: "Learn123"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
	if store.calls != 0 {
		t.Error("store was called with a cancelled context")
	}
}

// TestDefaultDisplayNameFromEmail checks the local-part derivation directly.
func TestDefaultDisplayNameFromEmail(t *testing.T) {
	if got := displayNameOrDefault("", "student@example.com"); got != "student" {
		t.Errorf("displayNameOrDefault = %q, want student", got)
	}
	if got := displayNameOrDefault("  ", "student@example.com"); got != "student" {
		t.Errorf("blank display name should default, got %q", got)
	}
	if got := displayNameOrDefault("Ali", "student@example.com"); got != "Ali" {
		t.Errorf("displayNameOrDefault = %q, want Ali", got)
	}
	if got := displayNameOrDefault("", "no-at-sign"); !strings.EqualFold(got, "no-at-sign") {
		t.Errorf("displayNameOrDefault fallback = %q", got)
	}
}
