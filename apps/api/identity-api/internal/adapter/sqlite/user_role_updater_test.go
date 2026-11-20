package sqlite

import (
	"context"
	"errors"
	"testing"

	"lms-english/identity-service/internal/domain"
)

// TestUpdateUserRolePersists verifies the returned account reflects the new role and
// that the change is stored.
func TestUpdateUserRolePersists(t *testing.T) {
	ctx := context.Background()
	db := newMigratedMemoryDB(t)
	user := sampleUser("user-role-1", "role1@example.com")
	seedRefreshUser(t, db, user)

	updater := NewUserRoleUpdater(db)
	updated, err := updater.UpdateUserRole(ctx, user.ID, domain.RoleTeacher)
	if err != nil {
		t.Fatalf("UpdateUserRole() error = %v, want nil", err)
	}
	if updated.Role != domain.RoleTeacher {
		t.Errorf("returned role = %v, want TEACHER", updated.Role)
	}
	if updated.ID != user.ID || updated.Email != user.Email {
		t.Errorf("returned account = %+v, want the target account", updated)
	}

	reloaded, err := NewUserRepository(db).FindByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if reloaded.Role != domain.RoleTeacher {
		t.Errorf("stored role = %v, want TEACHER", reloaded.Role)
	}
}

// TestUpdateUserRoleMissingAccount verifies an unknown id reports ErrNotFound.
func TestUpdateUserRoleMissingAccount(t *testing.T) {
	ctx := context.Background()
	db := newMigratedMemoryDB(t)

	_, err := NewUserRoleUpdater(db).UpdateUserRole(ctx, "no-such-user", domain.RoleAdmin)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("UpdateUserRole() error = %v, want ErrNotFound", err)
	}
}

// TestUpdateUserRoleSameRoleSucceeds verifies setting the current role is a success
// with no visible change.
func TestUpdateUserRoleSameRoleSucceeds(t *testing.T) {
	ctx := context.Background()
	db := newMigratedMemoryDB(t)
	user := sampleUser("user-role-2", "role2@example.com")
	seedRefreshUser(t, db, user)

	updated, err := NewUserRoleUpdater(db).UpdateUserRole(ctx, user.ID, domain.RoleStudent)
	if err != nil {
		t.Fatalf("UpdateUserRole() error = %v, want nil", err)
	}
	if updated.Role != domain.RoleStudent {
		t.Errorf("role = %v, want the unchanged STUDENT", updated.Role)
	}
}
