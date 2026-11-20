package usecase

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"lms-english/identity-service/internal/domain"
)

type loginUserRepo struct {
	email string
	user  domain.User
	err   error
}

func (f *loginUserRepo) FindByEmail(_ context.Context, email string) (domain.User, error) {
	f.email = email
	return f.user, f.err
}

func (f *loginUserRepo) FindByID(_ context.Context, _ string) (domain.User, error) {
	return f.user, f.err
}

type loginVerify struct{ err error }

func (f *loginVerify) Verify(string, string) error { return f.err }

type loginToken struct {
	token string
	err   error
}

func (f *loginToken) Issue(context.Context, domain.User) (string, error) { return f.token, f.err }

type loginRefresh struct {
	calls    int
	storeErr error
}

func (f *loginRefresh) Store(context.Context, string, string, string, time.Time) error {
	f.calls++
	return f.storeErr
}

func (f *loginRefresh) Revoke(context.Context, string) error { return nil }

func (f *loginRefresh) Rotate(context.Context, string, string, string, time.Time) (Rotation, error) {
	return Rotation{}, nil
}

func newLoginUseCase(repo UserRepository, verify PasswordVerifier, token TokenProvider, refresh RefreshTokenRepository) *LoginUserUseCase {
	return NewLoginUserUseCase(repo, verify, token, refresh, fakeID{}, fakeClock{})
}

func TestLoginSuccessIssuesAccessAndRefreshTokens(t *testing.T) {
	repo := &loginUserRepo{user: domain.User{
		ID: "u1", Email: "user@example.com", PasswordHash: "hash", DisplayName: "n",
		Role: domain.RoleStudent, CreatedAt: time.Now(),
	}}
	token := &loginToken{token: "access"}
	refresh := &loginRefresh{}
	uc := newLoginUseCase(repo, &loginVerify{}, token, refresh)

	result, err := uc.Execute(context.Background(), LoginUserCommand{Email: " User@Example.com ", Password: "Learn123"})
	if err != nil {
		t.Fatalf("Execute error = %v, want nil", err)
	}
	if result.AccessToken != "access" {
		t.Fatalf("AccessToken = %q, want access-token", result.AccessToken)
	}
	if result.RefreshToken == "" {
		t.Fatal("RefreshToken is empty, want a persisted token")
	}
	if refresh.calls != 1 {
		t.Fatalf("refresh store called %d times, want 1", refresh.calls)
	}
}

func TestLoginNormalizesEmailBeforeLookup(t *testing.T) {
	repo := &loginUserRepo{user: domain.User{
		ID: "u1", Email: "user@example.com", PasswordHash: "hash", DisplayName: "n",
		Role: domain.RoleStudent, CreatedAt: time.Now(),
	}}
	uc := newLoginUseCase(repo, &loginVerify{}, &loginToken{token: "access"}, &loginRefresh{})

	if _, err := uc.Execute(context.Background(), LoginUserCommand{Email: "  User@Example.com  ", Password: "Learn123"}); err != nil {
		t.Fatalf("Execute error = %v, want nil", err)
	}
	if repo.email != "user@example.com" {
		t.Fatalf("repo lookup email = %q, want normalized %q", repo.email, "user@example.com")
	}
}

func TestLoginInvalidCredentials(t *testing.T) {
	repo := &loginUserRepo{user: domain.User{
		ID: "u1", Email: "user@example.com", PasswordHash: "hash", DisplayName: "n",
		Role: domain.RoleStudent, CreatedAt: time.Now(),
	}}
	refresh := &loginRefresh{}
	uc := newLoginUseCase(repo, &loginVerify{err: errors.New("bad")}, &loginToken{token: "access"}, refresh)

	_, err := uc.Execute(context.Background(), LoginUserCommand{Email: "user@example.com", Password: "WrongPass1"})
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("Execute error = %v, want ErrInvalidCredentials", err)
	}
	if refresh.calls != 0 {
		t.Fatalf("refresh store called %d times after failed login, want 0", refresh.calls)
	}
}

func TestLoginMalformedEmailFailsBeforeVerification(t *testing.T) {
	repo := &loginUserRepo{}
	token := &loginToken{token: "access"}
	refresh := &loginRefresh{}
	uc := newLoginUseCase(repo, &loginVerify{}, token, refresh)

	_, err := uc.Execute(context.Background(), LoginUserCommand{Email: "no-at-sign", Password: "Learn123"})
	if !errors.Is(err, domain.ErrInvalidEmail) {
		t.Fatalf("Execute error = %v, want ErrInvalidEmail", err)
	}
	if token.token != "access" || refresh.calls != 0 {
		t.Fatal("token or refresh store touched after malformed email")
	}
}

func TestLoginUnknownEmailReportsInvalidCredentials(t *testing.T) {
	repo := &loginUserRepo{err: fmt.Errorf("find user by email: %w", domain.ErrNotFound)}
	refresh := &loginRefresh{}
	uc := newLoginUseCase(repo, &loginVerify{}, &loginToken{token: "access"}, refresh)

	_, err := uc.Execute(context.Background(), LoginUserCommand{Email: "nobody@example.com", Password: "Learn123"})
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("Execute error = %v, want ErrInvalidCredentials", err)
	}
	if refresh.calls != 0 {
		t.Fatal("refresh token stored for an unknown account")
	}
}
