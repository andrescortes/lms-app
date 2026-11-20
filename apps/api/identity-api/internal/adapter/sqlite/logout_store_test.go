package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"lms-english/identity-service/internal/domain"
)

// TestRevokeByJTIEndsOwnedSession verifies the owner can end their own token and
// the row records the revocation.
func TestRevokeByJTIEndsOwnedSession(t *testing.T) {
	ctx := context.Background()
	db := newMigratedMemoryDB(t)
	user := sampleUser("user-logout-1", "logout1@example.com")
	seedRefreshUser(t, db, user)

	store := NewRefreshTokenRepository(db)
	if err := store.Store(ctx, user.ID, "jti-logout-1", "token-1", time.Now().UTC().Add(time.Hour)); err != nil {
		t.Fatalf("store token: %v", err)
	}

	if err := store.RevokeByJTI(ctx, user.ID, "jti-logout-1"); err != nil {
		t.Fatalf("RevokeByJTI() error = %v, want nil", err)
	}

	var revoked *string
	if err := db.QueryRowContext(ctx, `SELECT revoked_at FROM refresh_tokens WHERE jti = ?`, "jti-logout-1").Scan(&revoked); err != nil {
		t.Fatalf("read row: %v", err)
	}
	if revoked == nil {
		t.Fatal("revoked_at is NULL after sign-out")
	}
}

// TestRevokeByJTIRejectsForeignToken verifies a token belonging to another user
// is refused and left untouched.
func TestRevokeByJTIRejectsForeignToken(t *testing.T) {
	ctx := context.Background()
	db := newMigratedMemoryDB(t)
	owner := sampleUser("user-logout-2", "logout2@example.com")
	other := sampleUser("user-logout-3", "logout3@example.com")
	seedRefreshUser(t, db, owner)
	seedRefreshUser(t, db, other)

	store := NewRefreshTokenRepository(db)
	if err := store.Store(ctx, owner.ID, "jti-owned", "token-owned", time.Now().UTC().Add(time.Hour)); err != nil {
		t.Fatalf("store token: %v", err)
	}

	err := store.RevokeByJTI(ctx, other.ID, "jti-owned")
	if !errors.Is(err, domain.ErrInvalidToken) {
		t.Fatalf("RevokeByJTI() error = %v, want ErrInvalidToken", err)
	}

	var revoked *string
	if err := db.QueryRowContext(ctx, `SELECT revoked_at FROM refresh_tokens WHERE jti = ?`, "jti-owned").Scan(&revoked); err != nil {
		t.Fatalf("read row: %v", err)
	}
	if revoked != nil {
		t.Fatal("another user's session was revoked")
	}
}

// TestRevokeByJTIRejectsExpiredAndRevoked verifies a token that is no longer
// active is refused rather than re-revoked.
func TestRevokeByJTIRejectsExpiredAndRevoked(t *testing.T) {
	ctx := context.Background()
	db := newMigratedMemoryDB(t)
	user := sampleUser("user-logout-4", "logout4@example.com")
	seedRefreshUser(t, db, user)

	store := NewRefreshTokenRepository(db)
	if err := store.Store(ctx, user.ID, "jti-expired", "token-expired", time.Now().UTC().Add(-time.Minute)); err != nil {
		t.Fatalf("store expired token: %v", err)
	}
	if err := store.Store(ctx, user.ID, "jti-revoked", "token-revoked", time.Now().UTC().Add(time.Hour)); err != nil {
		t.Fatalf("store revoked token: %v", err)
	}
	if err := store.Revoke(ctx, "jti-revoked"); err != nil {
		t.Fatalf("revoke token: %v", err)
	}

	for _, jti := range []string{"jti-expired", "jti-revoked", "jti-missing"} {
		if err := store.RevokeByJTI(ctx, user.ID, jti); !errors.Is(err, domain.ErrInvalidToken) {
			t.Errorf("%s: error = %v, want ErrInvalidToken", jti, err)
		}
	}
}
