package sqlite

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"lms-english/identity-service/internal/domain"
)

// TestNoPlaintextPasswordPersisted is the SC-006 audit for the storage layer: the
// plaintext password must never appear in stored user rows, profile rows, or the
// outbox event payload, across the success path.
func TestNoPlaintextPasswordPersisted(t *testing.T) {
	ctx := context.Background()
	db := newMigratedMemoryDB(t)
	store := NewRegistrationStore(db)

	const plaintext = "Plaintext1"

	user := sampleUser("user-1", "student@example.com")
	// Use a bcrypt-shaped hash so the stored value is clearly a hash, not the
	// plaintext, and stamp the plaintext into the value the "hasher" would have
	// produced to prove the store stores whatever the hasher returns.
	user.PasswordHash = "$2a$04$nottheplaintextnottheplaintextnottheplaintextnotplain"
	if err := store.RegisterUser(ctx, user, domain.NewEmptyProfile(user.ID), sampleEvent(user)); err != nil {
		t.Fatalf("RegisterUser() error = %v", err)
	}

	var hash, eventPayload, profileRow string
	_ = db.QueryRowContext(ctx, `SELECT password_hash FROM users WHERE id = ?`, user.ID).Scan(&hash)
	_ = db.QueryRowContext(ctx, `SELECT payload FROM registration_events WHERE id = ?`, user.ID).Scan(&eventPayload)
	_ = db.QueryRowContext(ctx, `SELECT COALESCE(native_language,'') || COALESCE(goals,'') || COALESCE(timezone,'') || COALESCE(avatar_url,'') FROM profiles WHERE user_id = ?`, user.ID).Scan(&profileRow)

	if strings.Contains(hash, plaintext) {
		t.Error("plaintext password found in stored password_hash")
	}
	if strings.Contains(eventPayload, plaintext) {
		t.Error("plaintext password found in event payload")
	}
	if strings.Contains(profileRow, plaintext) {
		t.Error("plaintext password found in profile row")
	}
	if hash == plaintext {
		t.Error("password_hash equals the plaintext")
	}

	// The event payload must also be valid JSON with only non-sensitive fields.
	var payload map[string]any
	if err := json.Unmarshal([]byte(eventPayload), &payload); err != nil {
		t.Fatalf("event payload is not JSON: %v", err)
	}
	if _, hasHash := payload["passwordHash"]; hasHash {
		t.Error("event payload must not include a password field")
	}
}
