package usecase

import (
	"context"
	"errors"
	"testing"

	"lms-english/identity-service/internal/domain"
)

// profileStub serves a fixed view and records the change it was handed.
type profileStub struct {
	view   ProfileView
	getErr error
	change ProfileChange
	calls  int
	setErr error
}

func (f *profileStub) GetUserProfile(context.Context, string) (ProfileView, error) {
	return f.view, f.getErr
}

func (f *profileStub) UpdateUserProfile(_ context.Context, _ string, change ProfileChange) error {
	f.calls++
	f.change = change
	return f.setErr
}

// userByID serves one user for the password path.
type userByID struct {
	user domain.User
	err  error
}

func (f *userByID) FindByEmail(context.Context, string) (domain.User, error) { return f.user, nil }
func (f *userByID) FindByID(context.Context, string) (domain.User, error)    { return f.user, f.err }

// verifyStub answers with a fixed verification result.
type verifyStub struct{ err error }

func (f *verifyStub) Verify(string, string) error { return f.err }

// hashStub records the plaintext it was asked to hash.
type hashStub struct {
	seen string
	err  error
}

func (f *hashStub) Hash(password string) (string, error) {
	f.seen = password
	return "hashed:" + password, f.err
}

func newUpdateUC(profiles *profileStub, users *userByID, verify error) (*UpdateProfileUseCase, *hashStub) {
	hasher := &hashStub{}
	return NewUpdateProfileUseCase(profiles, users, &verifyStub{err: verify}, hasher), hasher
}

// TestUpdateProfilePartialChange verifies only the named fields reach the store.
func TestUpdateProfilePartialChange(t *testing.T) {
	profiles := &profileStub{}
	uc, _ := newUpdateUC(profiles, &userByID{}, nil)

	goal := "Pass B2"
	if _, err := uc.Execute(context.Background(), UpdateProfileCommand{UserID: "u1", Goals: &goal}); err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if profiles.change.Goals == nil || *profiles.change.Goals != goal {
		t.Errorf("Goals = %v, want %q", profiles.change.Goals, goal)
	}
	if profiles.change.Timezone != nil || profiles.change.DisplayName != nil || profiles.change.PasswordHash != nil {
		t.Errorf("unexpected fields set: %+v", profiles.change)
	}
}

// TestUpdateProfileTrimsDisplayNameAndRefusesBlank verifies the display-name rule.
func TestUpdateProfileTrimsDisplayNameAndRefusesBlank(t *testing.T) {
	profiles := &profileStub{}
	uc, _ := newUpdateUC(profiles, &userByID{}, nil)

	name := "  Learner One  "
	if _, err := uc.Execute(context.Background(), UpdateProfileCommand{UserID: "u1", DisplayName: &name}); err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if profiles.change.DisplayName == nil || *profiles.change.DisplayName != "Learner One" {
		t.Errorf("DisplayName = %v, want trimmed", profiles.change.DisplayName)
	}

	blank := "   "
	if _, err := uc.Execute(context.Background(), UpdateProfileCommand{UserID: "u1", DisplayName: &blank}); !errors.Is(err, domain.ErrInvalidDisplayName) {
		t.Errorf("error = %v, want ErrInvalidDisplayName", err)
	}
}

// TestUpdateProfilePasswordChange verifies the guarded password path.
func TestUpdateProfilePasswordChange(t *testing.T) {
	profiles := &profileStub{}
	users := &userByID{user: domain.User{ID: "u1", PasswordHash: "stored-hash"}}

	t.Run("valid", func(t *testing.T) {
		uc, hasher := newUpdateUC(profiles, users, nil)
		if _, err := uc.Execute(context.Background(), UpdateProfileCommand{
			UserID: "u1", CurrentPassword: "OldPass1", NewPassword: "NewPass1",
		}); err != nil {
			t.Fatalf("Execute() error = %v, want nil", err)
		}
		if profiles.change.PasswordHash == nil || *profiles.change.PasswordHash != "hashed:NewPass1" {
			t.Errorf("PasswordHash = %v, want the hashed new password", profiles.change.PasswordHash)
		}
		if hasher.seen != "NewPass1" {
			t.Errorf("hashed %q, want the new password", hasher.seen)
		}
	})

	t.Run("missing current", func(t *testing.T) {
		before := profiles.calls
		uc, hasher := newUpdateUC(profiles, users, nil)
		_, err := uc.Execute(context.Background(), UpdateProfileCommand{UserID: "u1", NewPassword: "NewPass1"})
		if !errors.Is(err, domain.ErrInvalidCredentials) {
			t.Fatalf("error = %v, want ErrInvalidCredentials", err)
		}
		if profiles.calls != before || hasher.seen != "" {
			t.Error("store or hasher touched after a refusal")
		}
	})

	t.Run("wrong current", func(t *testing.T) {
		before := profiles.calls
		uc, hasher := newUpdateUC(profiles, users, errors.New("mismatch"))
		_, err := uc.Execute(context.Background(), UpdateProfileCommand{
			UserID: "u1", CurrentPassword: "Wrong1aa", NewPassword: "NewPass1",
		})
		if !errors.Is(err, domain.ErrInvalidCredentials) {
			t.Fatalf("error = %v, want ErrInvalidCredentials", err)
		}
		if profiles.calls != before || hasher.seen != "" {
			t.Error("store or hasher touched after a refusal")
		}
	})

	t.Run("weak new password", func(t *testing.T) {
		before := profiles.calls
		uc, hasher := newUpdateUC(profiles, users, nil)
		_, err := uc.Execute(context.Background(), UpdateProfileCommand{
			UserID: "u1", CurrentPassword: "OldPass1", NewPassword: "weak",
		})
		if !errors.Is(err, domain.ErrWeakPassword) {
			t.Fatalf("error = %v, want ErrWeakPassword", err)
		}
		if profiles.calls != before || hasher.seen != "" {
			t.Error("store or hasher touched after a weak password")
		}
	})
}
