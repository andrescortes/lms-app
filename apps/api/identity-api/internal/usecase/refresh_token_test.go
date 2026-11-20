package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"lms-english/identity-service/internal/domain"
)

// refreshUserRepo serves one user for both lookups.
type refreshUserRepo struct {
	user   domain.User
	byID   string
	called bool
}

func (f *refreshUserRepo) FindByEmail(context.Context, string) (domain.User, error) {
	return f.user, nil
}

func (f *refreshUserRepo) FindByID(_ context.Context, id string) (domain.User, error) {
	f.called = true
	f.byID = id
	return f.user, nil
}

// rotatingRepo records the rotation and returns the outcome the test asked for.
type rotatingRepo struct {
	outcome     RotationOutcome
	presented   string
	replacement string
	expiresAt   time.Time
	newJTI      string
	err         error
}

func (f *rotatingRepo) Store(_ context.Context, _ string, _ string, token string, _ time.Time) error {
	return nil
}
func (f *rotatingRepo) Revoke(context.Context, string) error { return nil }

func (f *rotatingRepo) Rotate(_ context.Context, presented, newJTI, replacement string, expiresAt time.Time) (Rotation, error) {
	f.newJTI = newJTI
	f.presented = presented
	f.replacement = replacement
	f.expiresAt = expiresAt
	if f.err != nil {
		return Rotation{}, f.err
	}
	return Rotation{Outcome: f.outcome, UserID: "user-1"}, nil
}

func newRefreshUseCase(repo *rotatingRepo, users *refreshUserRepo) *RefreshTokenUseCase {
	return NewRefreshTokenUseCase(users, &loginToken{token: "access-token"}, repo, fakeID{}, fakeClock{})
}

// TestRefreshRotatesAndIssuesTokens verifies a valid token is rotated and a new
// pair is returned with the presented replacement.
func TestRefreshRotatesAndIssuesTokens(t *testing.T) {
	repo := &rotatingRepo{outcome: RotationRotated}
	users := &refreshUserRepo{user: domain.User{ID: "user-1", Role: domain.RoleStudent}}
	uc := newRefreshUseCase(repo, users)

	result, err := uc.Execute(context.Background(), RefreshCommand{RefreshToken: "presented-token"})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if result.AccessToken != "access-token" {
		t.Errorf("AccessToken = %q, want access-token", result.AccessToken)
	}
	if result.RefreshToken != repo.replacement {
		t.Errorf("RefreshToken = %q, want the rotated replacement", result.RefreshToken)
	}
	if repo.presented != "presented-token" {
		t.Errorf("presented = %q, want presented-token", repo.presented)
	}
	if !users.called || users.byID != "user-1" {
		t.Errorf("user lookup = (%v, %q), want the rotated user", users.called, users.byID)
	}
}

// TestRefreshRefusesEveryBadToken verifies expired, spent, and unknown tokens
// all report the same invalid-token sentinel.
func TestRefreshRefusesEveryBadToken(t *testing.T) {
	for _, tc := range []struct {
		name    string
		outcome RotationOutcome
	}{
		{"expired", RotationExpired},
		{"spent", RotationSpent},
		{"unknown", RotationUnknown},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &rotatingRepo{outcome: tc.outcome}
			uc := newRefreshUseCase(repo, &refreshUserRepo{})

			_, err := uc.Execute(context.Background(), RefreshCommand{RefreshToken: "token"})
			if !errors.Is(err, domain.ErrInvalidToken) {
				t.Fatalf("Execute() error = %v, want ErrInvalidToken", err)
			}
		})
	}
}

// TestRefreshRejectsEmptyToken verifies an empty token never reaches the store.
func TestRefreshRejectsEmptyToken(t *testing.T) {
	repo := &rotatingRepo{outcome: RotationRotated}
	uc := newRefreshUseCase(repo, &refreshUserRepo{})

	_, err := uc.Execute(context.Background(), RefreshCommand{})
	if !errors.Is(err, domain.ErrInvalidToken) {
		t.Fatalf("Execute() error = %v, want ErrInvalidToken", err)
	}
	if repo.presented != "" {
		t.Errorf("store called with %q, want no call", repo.presented)
	}
}

// TestRefreshHonoursContextCancellation verifies a cancelled context is not
// treated as a bad token.
func TestRefreshHonoursContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	uc := newRefreshUseCase(&rotatingRepo{}, &refreshUserRepo{})
	_, err := uc.Execute(ctx, RefreshCommand{RefreshToken: "token"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Execute() error = %v, want context.Canceled", err)
	}
}
