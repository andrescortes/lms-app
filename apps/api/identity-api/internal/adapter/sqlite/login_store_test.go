package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"lms-english/identity-service/internal/domain"
)

// TestUserRepositoryFindByEmailFound verifies a stored user is returned with the
// fields the login flow needs, including the parsed role.
func TestUserRepositoryFindByEmailFound(t *testing.T) {
	ctx := context.Background()
	db := newMigratedMemoryDB(t)

	user := sampleUser("user-repo-1", "repo@example.com")
	if err := NewRegistrationStore(db).RegisterUser(ctx, user, domain.NewEmptyProfile(user.ID), sampleEvent(user)); err != nil {
		t.Fatalf("seed user: %v", err)
	}

	got, err := NewUserRepository(db).FindByEmail(ctx, "repo@example.com")
	if err != nil {
		t.Fatalf("FindByEmail() error = %v, want nil", err)
	}
	if got.ID != user.ID {
		t.Errorf("ID = %q, want %q", got.ID, user.ID)
	}
	if got.PasswordHash != user.PasswordHash {
		t.Errorf("PasswordHash = %q, want the stored hash", got.PasswordHash)
	}
	if got.Role != domain.RoleStudent {
		t.Errorf("Role = %v, want %v", got.Role, domain.RoleStudent)
	}
}

// TestUserRepositoryFindByEmailNotFound verifies a missing account reports the
// domain not-found sentinel rather than a raw SQL error.
func TestUserRepositoryFindByEmailNotFound(t *testing.T) {
	ctx := context.Background()
	db := newMigratedMemoryDB(t)

	_, err := NewUserRepository(db).FindByEmail(ctx, "nobody@example.com")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("FindByEmail() error = %v, want ErrNotFound", err)
	}
}

// TestRefreshTokenStorePersistsAndRevokes verifies Store writes one row with the
// supplied expiry and Revoke clears it, and that the raw token is never stored.
func TestRefreshTokenStorePersistsAndRevokes(t *testing.T) {
	ctx := context.Background()
	db := newMigratedMemoryDB(t)

	user := sampleUser("user-token-1", "token@example.com")
	if err := NewRegistrationStore(db).RegisterUser(ctx, user, domain.NewEmptyProfile(user.ID), sampleEvent(user)); err != nil {
		t.Fatalf("seed user: %v", err)
	}

	store := NewRefreshTokenRepository(db)
	expiresAt := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
	const rawToken = "opaque-refresh-token-value"

	if err := store.Store(ctx, user.ID, "jti-1", rawToken, expiresAt); err != nil {
		t.Fatalf("Store() error = %v, want nil", err)
	}

	var (
		tokenHash string
		storedExp string
		revoked   interface{}
	)
	row := db.QueryRowContext(ctx,
		`SELECT token_hash, expires_at, revoked_at FROM refresh_tokens WHERE jti = ?`, "jti-1")
	if err := row.Scan(&tokenHash, &storedExp, &revoked); err != nil {
		t.Fatalf("read stored row: %v", err)
	}
	if tokenHash == rawToken {
		t.Fatal("the raw refresh token was stored; expected only its hash")
	}
	if storedExp != formatUTC(expiresAt) {
		t.Errorf("expires_at = %q, want %q", storedExp, formatUTC(expiresAt))
	}
	if revoked != nil {
		t.Errorf("revoked_at = %v, want NULL before revocation", revoked)
	}

	if err := store.Revoke(ctx, "jti-1"); err != nil {
		t.Fatalf("Revoke() error = %v, want nil", err)
	}
	var revokedAt interface{}
	row = db.QueryRowContext(ctx, `SELECT revoked_at FROM refresh_tokens WHERE jti = ?`, "jti-1")
	if err := row.Scan(&revokedAt); err != nil {
		t.Fatalf("read revoked row: %v", err)
	}
	if revokedAt == nil {
		t.Fatal("revoked_at is NULL after Revoke, want a timestamp")
	}
}
