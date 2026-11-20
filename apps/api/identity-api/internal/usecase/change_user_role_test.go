package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"lms-english/identity-service/internal/domain"
)

// roleUpdaterStub records the update and returns a fixed user.
type roleUpdaterStub struct {
	user   domain.User
	err    error
	calls  int
	userID string
	role   domain.Role
}

func (f *roleUpdaterStub) UpdateUserRole(_ context.Context, userID string, role domain.Role) (domain.User, error) {
	f.calls++
	f.userID = userID
	f.role = role
	return f.user, f.err
}

func roleUser() domain.User {
	return domain.User{
		ID: "target-1", Email: "target@example.com", DisplayName: "Target",
		Role: domain.RoleTeacher, CreatedAt: time.Date(2026, 11, 20, 9, 0, 0, 0, time.UTC),
	}
}

// TestChangeUserRoleAppliesValidRole verifies a valid change reaches the repository.
func TestChangeUserRoleAppliesValidRole(t *testing.T) {
	repo := &roleUpdaterStub{user: roleUser()}
	uc := NewChangeUserRoleUseCase(repo)

	user, err := uc.Execute(context.Background(), ChangeUserRoleCommand{
		ActorUserID: "admin-1", TargetUserID: "target-1", NewRole: "TEACHER",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if repo.calls != 1 || repo.userID != "target-1" || repo.role != domain.RoleTeacher {
		t.Errorf("update = (%d, %q, %v), want one call for target-1 as TEACHER", repo.calls, repo.userID, repo.role)
	}
	if user.Role != domain.RoleTeacher {
		t.Errorf("returned role = %v, want TEACHER", user.Role)
	}
}

// TestChangeUserRoleRefusesSelfChange verifies the guard runs before the repository.
func TestChangeUserRoleRefusesSelfChange(t *testing.T) {
	repo := &roleUpdaterStub{user: roleUser()}
	uc := NewChangeUserRoleUseCase(repo)

	_, err := uc.Execute(context.Background(), ChangeUserRoleCommand{
		ActorUserID: "admin-1", TargetUserID: "admin-1", NewRole: "STUDENT",
	})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("Execute() error = %v, want ErrForbidden", err)
	}
	if repo.calls != 0 {
		t.Error("repository was called for a self-change")
	}
}

// TestChangeUserRoleRefusesInvalidRole verifies an unknown or wrongly cased role is
// refused before the repository.
func TestChangeUserRoleRefusesInvalidRole(t *testing.T) {
	for _, role := range []string{"WIZARD", "admin", ""} {
		repo := &roleUpdaterStub{user: roleUser()}
		_, err := NewChangeUserRoleUseCase(repo).Execute(context.Background(), ChangeUserRoleCommand{
			ActorUserID: "admin-1", TargetUserID: "target-1", NewRole: role,
		})
		if !errors.Is(err, domain.ErrInvalidRole) {
			t.Errorf("role %q: error = %v, want ErrInvalidRole", role, err)
		}
		if repo.calls != 0 {
			t.Errorf("role %q: repository was called", role)
		}
	}
}

// TestChangeUserRoleReportsMissingAccount verifies not-found propagates.
func TestChangeUserRoleReportsMissingAccount(t *testing.T) {
	repo := &roleUpdaterStub{err: domain.ErrNotFound}
	_, err := NewChangeUserRoleUseCase(repo).Execute(context.Background(), ChangeUserRoleCommand{
		ActorUserID: "admin-1", TargetUserID: "gone", NewRole: "STUDENT",
	})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("Execute() error = %v, want ErrNotFound", err)
	}
}

// TestChangeUserRoleHonoursContextCancellation verifies cancellation is not a refusal.
func TestChangeUserRoleHonoursContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := NewChangeUserRoleUseCase(&roleUpdaterStub{}).Execute(ctx, ChangeUserRoleCommand{
		ActorUserID: "admin-1", TargetUserID: "target-1", NewRole: "STUDENT",
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Execute() error = %v, want context.Canceled", err)
	}
}
