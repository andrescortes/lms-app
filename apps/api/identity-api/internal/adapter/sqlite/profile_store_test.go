package sqlite

import (
	"context"
	"errors"
	"testing"

	"lms-english/identity-service/internal/adapter/security"
	"lms-english/identity-service/internal/domain"
	"lms-english/identity-service/internal/usecase"
)

// TestProfileStoreReadsAccountAndProfile verifies a registered user reads back
// with empty profile fields and that the read model carries no hash.
func TestProfileStoreReadsAccountAndProfile(t *testing.T) {
	ctx := context.Background()
	db := newMigratedMemoryDB(t)
	user := sampleUser("user-profile-1", "profile1@example.com")
	seedRefreshUser(t, db, user)

	view, err := NewProfileStore(db).GetUserProfile(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetUserProfile() error = %v, want nil", err)
	}
	if view.Email != user.Email {
		t.Errorf("Email = %q, want %q", view.Email, user.Email)
	}
	if view.DisplayName != user.DisplayName {
		t.Errorf("DisplayName = %q, want %q", view.DisplayName, user.DisplayName)
	}
	if view.Role != domain.RoleStudent {
		t.Errorf("Role = %v, want STUDENT", view.Role)
	}
	if view.NativeLanguage != nil || view.Goals != nil || view.Timezone != nil || view.AvatarURL != nil {
		t.Errorf("expected every profile field unset, got %+v", view)
	}
}

// TestProfileStoreMissingAccount verifies a missing account reports ErrNotFound.
func TestProfileStoreMissingAccount(t *testing.T) {
	ctx := context.Background()
	db := newMigratedMemoryDB(t)

	if _, err := NewProfileStore(db).GetUserProfile(ctx, "nobody"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("GetUserProfile() error = %v, want ErrNotFound", err)
	}
}

// TestProfileStorePartialUpdate verifies only the named fields change.
func TestProfileStorePartialUpdate(t *testing.T) {
	ctx := context.Background()
	db := newMigratedMemoryDB(t)
	user := sampleUser("user-profile-2", "profile2@example.com")
	seedRefreshUser(t, db, user)

	store := NewProfileStore(db)
	goal := "Pass B2"
	if err := store.UpdateUserProfile(ctx, user.ID, usecase.ProfileChange{Goals: &goal}); err != nil {
		t.Fatalf("UpdateUserProfile() error = %v", err)
	}

	view, err := store.GetUserProfile(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetUserProfile() error = %v", err)
	}
	if view.Goals == nil || *view.Goals != goal {
		t.Errorf("Goals = %v, want %q", view.Goals, goal)
	}
	if view.DisplayName != user.DisplayName {
		t.Errorf("DisplayName = %q, want unchanged %q", view.DisplayName, user.DisplayName)
	}
	if view.Timezone != nil || view.NativeLanguage != nil || view.AvatarURL != nil {
		t.Errorf("untouched fields changed: %+v", view)
	}
}

// TestProfileStoreClearsWithEmptyString verifies an empty value clears a field.
func TestProfileStoreClearsWithEmptyString(t *testing.T) {
	ctx := context.Background()
	db := newMigratedMemoryDB(t)
	user := sampleUser("user-profile-3", "profile3@example.com")
	seedRefreshUser(t, db, user)

	store := NewProfileStore(db)
	tz := "Europe/Madrid"
	if err := store.UpdateUserProfile(ctx, user.ID, usecase.ProfileChange{Timezone: &tz}); err != nil {
		t.Fatalf("set timezone: %v", err)
	}

	empty := ""
	if err := store.UpdateUserProfile(ctx, user.ID, usecase.ProfileChange{Timezone: &empty}); err != nil {
		t.Fatalf("clear timezone: %v", err)
	}

	view, err := store.GetUserProfile(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetUserProfile() error = %v", err)
	}
	if view.Timezone != nil {
		t.Errorf("Timezone = %v, want cleared", *view.Timezone)
	}
}

// TestProfileStorePasswordChangeStoresHashOnly verifies a password update writes
// only the protected form.
func TestProfileStorePasswordChangeStoresHashOnly(t *testing.T) {
	ctx := context.Background()
	db := newMigratedMemoryDB(t)
	user := sampleUser("user-profile-4", "profile4@example.com")
	seedRefreshUser(t, db, user)

	hash := "$2a$04$newhashnewhashnewhashnewhashenewhashnewhashnewhashnewhasXXXXX"
	if err := NewProfileStore(db).UpdateUserProfile(ctx, user.ID, usecase.ProfileChange{PasswordHash: &hash}); err != nil {
		t.Fatalf("UpdateUserProfile() error = %v", err)
	}

	var stored string
	if err := db.QueryRowContext(ctx, `SELECT password_hash FROM users WHERE id = ?`, user.ID).Scan(&stored); err != nil {
		t.Fatalf("read hash: %v", err)
	}
	if stored != hash {
		t.Errorf("stored hash = %q, want the supplied hash", stored)
	}

	// The read model has no field that could carry it back out.
	view, err := NewProfileStore(db).GetUserProfile(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetUserProfile() error = %v", err)
	}
	if view.Email == "" {
		t.Fatal("view is empty")
	}
}

// TestProfileStoreEndToEndPasswordChange verifies the real hash chain: the old
// password stops verifying and the new one starts.
func TestProfileStoreEndToEndPasswordChange(t *testing.T) {
	ctx := context.Background()
	db := newMigratedMemoryDB(t)
	hasher := security.NewBcryptHasher(4)

	user := sampleUser("user-profile-5", "profile5@example.com")
	oldHash, err := hasher.Hash("OldPass1")
	if err != nil {
		t.Fatalf("hash old password: %v", err)
	}
	user.PasswordHash = oldHash
	seedRefreshUser(t, db, user)

	newHash := mustHash(t, hasher, "NewPass1")
	if err := NewProfileStore(db).UpdateUserProfile(ctx, user.ID, usecase.ProfileChange{PasswordHash: &newHash}); err != nil {
		t.Fatalf("UpdateUserProfile() error = %v", err)
	}

	reloaded, err := NewUserRepository(db).FindByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if err := hasher.Verify(reloaded.PasswordHash, "OldPass1"); err == nil {
		t.Error("the old password still verifies after the change")
	}
	if err := hasher.Verify(reloaded.PasswordHash, "NewPass1"); err != nil {
		t.Errorf("the new password does not verify: %v", err)
	}
}

func mustHash(t *testing.T, hasher *security.BcryptHasher, password string) string {
	t.Helper()
	hash, err := hasher.Hash(password)
	if err != nil {
		t.Fatalf("hash %q: %v", password, err)
	}
	return hash
}

// TestProfileStoreRefusedPasswordChangeKeepsOldPassword verifies SC-004 through the
// real chain: after a refused change the stored password is byte-identical, the old
// password still verifies, and the proposed new one does not.
func TestProfileStoreRefusedPasswordChangeKeepsOldPassword(t *testing.T) {
	ctx := context.Background()
	db := newMigratedMemoryDB(t)
	hasher := security.NewBcryptHasher(4)

	user := sampleUser("user-profile-6", "profile6@example.com")
	oldHash, err := hasher.Hash("OldPass1")
	if err != nil {
		t.Fatalf("hash old password: %v", err)
	}
	user.PasswordHash = oldHash
	seedRefreshUser(t, db, user)

	profiles := NewProfileStore(db)
	users := NewUserRepository(db)
	uc := usecase.NewUpdateProfileUseCase(profiles, users, hasher, hasher)

	refusals := map[string]usecase.UpdateProfileCommand{
		"wrong current":   {UserID: user.ID, CurrentPassword: "Wrong1aa", NewPassword: "NewPass1"},
		"missing current": {UserID: user.ID, NewPassword: "NewPass1"},
	}
	for name, cmd := range refusals {
		t.Run(name, func(t *testing.T) {
			if _, err := uc.Execute(ctx, cmd); !errors.Is(err, domain.ErrInvalidCredentials) {
				t.Fatalf("Execute() error = %v, want ErrInvalidCredentials", err)
			}
			reloaded, err := users.FindByID(ctx, user.ID)
			if err != nil {
				t.Fatalf("FindByID() error = %v", err)
			}
			if reloaded.PasswordHash != oldHash {
				t.Fatal("the stored password changed after a refused change")
			}
			if err := hasher.Verify(reloaded.PasswordHash, "OldPass1"); err != nil {
				t.Errorf("the old password no longer verifies: %v", err)
			}
			if err := hasher.Verify(reloaded.PasswordHash, "NewPass1"); err == nil {
				t.Error("the proposed new password verifies after a refused change")
			}
		})
	}
}
