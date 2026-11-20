package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"lms-english/identity-service/internal/domain"
)

// logoutRepo records the revoke it was asked to perform.
type logoutRepo struct {
	userID string
	jti    string
	calls  int
	err    error
}

func (f *logoutRepo) Store(context.Context, string, string, string, time.Time) error {
	return nil
}

func (f *logoutRepo) Revoke(context.Context, string) error { return nil }

func (f *logoutRepo) Rotate(context.Context, string, string, string, time.Time) (Rotation, error) {
	return Rotation{}, nil
}

func (f *logoutRepo) RevokeByJTI(_ context.Context, userID, jti string) error {
	f.calls++
	f.userID = userID
	f.jti = jti
	return f.err
}

// TestLogoutRevokesOwnedSession verifies the use case revokes exactly the token
// it was handed, under the caller's own identity.
func TestLogoutRevokesOwnedSession(t *testing.T) {
	repo := &logoutRepo{}
	uc := NewLogoutUseCase(repo)

	if err := uc.Execute(context.Background(), LogoutCommand{UserID: "user-1", RefreshToken: "jti-1"}); err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if repo.calls != 1 {
		t.Fatalf("RevokeByJTI calls = %d, want 1", repo.calls)
	}
	if repo.userID != "user-1" || repo.jti != "jti-1" {
		t.Errorf("revoked (%q, %q), want (user-1, jti-1)", repo.userID, repo.jti)
	}
}

// TestLogoutReportsRepositoryRefusalAsInvalidToken verifies a foreign, expired,
// revoked, or missing token surfaces as the same sentinel.
func TestLogoutReportsRepositoryRefusalAsInvalidToken(t *testing.T) {
	repo := &logoutRepo{err: domain.ErrInvalidToken}
	uc := NewLogoutUseCase(repo)

	err := uc.Execute(context.Background(), LogoutCommand{UserID: "user-1", RefreshToken: "foreign"})
	if !errors.Is(err, domain.ErrInvalidToken) {
		t.Fatalf("Execute() error = %v, want ErrInvalidToken", err)
	}
}

// TestLogoutRejectsEmptyInputs verifies an empty identity or token never reaches
// the store.
func TestLogoutRejectsEmptyInputs(t *testing.T) {
	for name, cmd := range map[string]LogoutCommand{
		"no user":  {RefreshToken: "jti-1"},
		"no token": {UserID: "user-1"},
	} {
		repo := &logoutRepo{}
		err := NewLogoutUseCase(repo).Execute(context.Background(), cmd)
		if !errors.Is(err, domain.ErrInvalidToken) {
			t.Errorf("%s: error = %v, want ErrInvalidToken", name, err)
		}
		if repo.calls != 0 {
			t.Errorf("%s: store called %d times, want 0", name, repo.calls)
		}
	}
}

// TestLogoutHonoursContextCancellation verifies a cancelled context is not
// reported as a bad token.
func TestLogoutHonoursContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := NewLogoutUseCase(&logoutRepo{}).Execute(ctx, LogoutCommand{UserID: "user-1", RefreshToken: "jti-1"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Execute() error = %v, want context.Canceled", err)
	}
}
