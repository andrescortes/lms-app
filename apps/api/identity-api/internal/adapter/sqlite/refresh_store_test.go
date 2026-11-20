package sqlite

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"lms-english/identity-service/internal/domain"
	"lms-english/identity-service/internal/usecase"
)

// seedRefreshUser registers a user through the registration store so refresh
// token rows satisfy the users foreign key.
func seedRefreshUser(t *testing.T, db *sql.DB, user domain.User) {
	t.Helper()
	ctx := context.Background()
	if err := NewRegistrationStore(db).RegisterUser(ctx, user, domain.NewEmptyProfile(user.ID), sampleEvent(user)); err != nil {
		t.Fatalf("seed user %s: %v", user.ID, err)
	}
}

// rotatedExpiry is the replacement lifetime the tests hand to Rotate.
func rotatedExpiry(now time.Time) time.Time { return now.Add(7 * 24 * time.Hour) }

// activeChain returns the refresh token rows for a user that are unrevoked.
func activeChain(t *testing.T, db *sql.DB, userID string) int {
	t.Helper()
	var count int
	if err := db.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM refresh_tokens WHERE user_id = ? AND revoked_at IS NULL AND expires_at > ?`,
		userID, formatUTC(time.Now().UTC())).Scan(&count); err != nil {
		t.Fatalf("count active chain: %v", err)
	}
	return count
}

// TestRotateReplacesActiveToken verifies a valid token is revoked and a
// replacement row is inserted in one step.
func TestRotateReplacesActiveToken(t *testing.T) {
	ctx := context.Background()
	db := newMigratedMemoryDB(t)
	user := sampleUser("user-rotate-1", "rotate1@example.com")
	seedRefreshUser(t, db, user)

	store := NewRefreshTokenRepository(db)
	if err := store.Store(ctx, user.ID, "jti-1", "presented", time.Now().UTC().Add(time.Hour)); err != nil {
		t.Fatalf("store presented token: %v", err)
	}

	got, err := store.Rotate(ctx, "presented", "new-jti-1", "replacement", rotatedExpiry(time.Now().UTC()))
	if err != nil {
		t.Fatalf("Rotate() error = %v, want nil", err)
	}
	if got.Outcome != usecase.RotationRotated {
		t.Fatalf("outcome = %v, want RotationRotated", got.Outcome)
	}
	if got.UserID != user.ID {
		t.Errorf("UserID = %q, want %q", got.UserID, user.ID)
	}

	var revoked *string
	if err := db.QueryRowContext(ctx,
		`SELECT revoked_at FROM refresh_tokens WHERE jti = ?`, "jti-1").Scan(&revoked); err != nil {
		t.Fatalf("read presented row: %v", err)
	}
	if revoked == nil {
		t.Fatal("presented token was not revoked")
	}
	if n := activeChain(t, db, user.ID); n != 1 {
		t.Fatalf("active chain = %d, want the single replacement", n)
	}
}

// TestRotateRefusesSpentTokenAndRevokesChain verifies replaying a spent token
// revokes every active token for that user.
func TestRotateRefusesSpentTokenAndRevokesChain(t *testing.T) {
	ctx := context.Background()
	db := newMigratedMemoryDB(t)
	user := sampleUser("user-rotate-2", "rotate2@example.com")
	seedRefreshUser(t, db, user)

	store := NewRefreshTokenRepository(db)
	future := time.Now().UTC().Add(time.Hour)
	for _, token := range []string{"spent", "other-session"} {
		if err := store.Store(ctx, user.ID, "jti-"+token, token, future); err != nil {
			t.Fatalf("store %s: %v", token, err)
		}
	}

	if _, err := store.Rotate(ctx, "spent", "new-jti-2", "replacement-1", rotatedExpiry(time.Now().UTC())); err != nil {
		t.Fatalf("first Rotate() error = %v", err)
	}
	if n := activeChain(t, db, user.ID); n != 2 {
		t.Fatalf("active chain after rotation = %d, want 2", n)
	}

	got, err := store.Rotate(ctx, "spent", "new-jti-3", "replacement-2", rotatedExpiry(time.Now().UTC()))
	if err != nil {
		t.Fatalf("replay Rotate() error = %v, want nil", err)
	}
	if got.Outcome != usecase.RotationSpent {
		t.Fatalf("outcome = %v, want RotationSpent", got.Outcome)
	}
	if n := activeChain(t, db, user.ID); n != 0 {
		t.Fatalf("active chain after replay = %d, want 0 (whole chain revoked)", n)
	}
}

// TestRotateRefusesExpiredToken verifies an expired token is refused and leaves
// other sessions alone.
func TestRotateRefusesExpiredToken(t *testing.T) {
	ctx := context.Background()
	db := newMigratedMemoryDB(t)
	user := sampleUser("user-rotate-3", "rotate3@example.com")
	seedRefreshUser(t, db, user)

	store := NewRefreshTokenRepository(db)
	if err := store.Store(ctx, user.ID, "jti-expired", "expired", time.Now().UTC().Add(-time.Minute)); err != nil {
		t.Fatalf("store expired token: %v", err)
	}
	if err := store.Store(ctx, user.ID, "jti-live", "live", time.Now().UTC().Add(time.Hour)); err != nil {
		t.Fatalf("store live token: %v", err)
	}

	got, err := store.Rotate(ctx, "expired", "new-jti-4", "replacement", rotatedExpiry(time.Now().UTC()))
	if err != nil {
		t.Fatalf("Rotate() error = %v, want nil", err)
	}
	if got.Outcome != usecase.RotationExpired {
		t.Fatalf("outcome = %v, want RotationExpired", got.Outcome)
	}
	if n := activeChain(t, db, user.ID); n != 1 {
		t.Fatalf("active chain = %d, want the untouched live token", n)
	}
}

// TestRotateRefusesUnknownToken verifies a token the service never issued is
// reported as unknown without touching any row.
func TestRotateRefusesUnknownToken(t *testing.T) {
	ctx := context.Background()
	db := newMigratedMemoryDB(t)

	got, err := NewRefreshTokenRepository(db).Rotate(ctx, "never-issued", "new-jti-5", "replacement", rotatedExpiry(time.Now().UTC()))
	if err != nil {
		t.Fatalf("Rotate() error = %v, want nil", err)
	}
	if got.Outcome != usecase.RotationUnknown {
		t.Fatalf("outcome = %v, want RotationUnknown", got.Outcome)
	}
	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM refresh_tokens`).Scan(&count); err != nil {
		t.Fatalf("count rows: %v", err)
	}
	if count != 0 {
		t.Fatalf("refresh_tokens rows = %d, want 0", count)
	}
}
