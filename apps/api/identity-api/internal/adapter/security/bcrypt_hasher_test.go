package security

import (
	"errors"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestBcryptHasherHashAndVerify(t *testing.T) {
	h := NewBcryptHasher(bcrypt.MinCost)

	hash, err := h.Hash("Learn123")
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}
	if hash == "Learn123" || hash == "" {
		t.Fatalf("hash %q must be a protected representation, not the plaintext", hash)
	}
	if !strings.HasPrefix(hash, "$2") {
		t.Fatalf("hash %q does not look like a bcrypt hash", hash)
	}
	if err := h.Verify(hash, "Learn123"); err != nil {
		t.Fatalf("Verify(correct) error = %v", err)
	}
	if err := h.Verify(hash, "wrong"); !errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		t.Fatalf("Verify(wrong) error = %v, want mismatch", err)
	}
}

// TestBcryptHasherAcceptsPolicyMaxMultibyte ensures a 72-character password made
// of multibyte text (over bcrypt's 72-byte limit) still hashes and verifies,
// because the hasher pre-hashes before bcrypt.
func TestBcryptHasherAcceptsPolicyMaxMultibyte(t *testing.T) {
	h := NewBcryptHasher(bcrypt.MinCost)
	pw := strings.Repeat("Á", 71) + "1" // 72 characters, 143 bytes
	if len([]rune(pw)) != 72 || len(pw) <= 72 {
		t.Fatalf("test password has %d runes and %d bytes, want 72 runes and >72 bytes", len([]rune(pw)), len(pw))
	}

	hash, err := h.Hash(pw)
	if err != nil {
		t.Fatalf("Hash() of policy-max multibyte password failed: %v", err)
	}
	if err := h.Verify(hash, pw); err != nil {
		t.Fatalf("Verify(correct) error = %v", err)
	}
	if err := h.Verify(hash, pw+"x"); !errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		t.Fatalf("Verify(wrong) error = %v, want mismatch", err)
	}
}
