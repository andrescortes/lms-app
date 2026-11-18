// Package security provides password hashing for the Identity service.
package security

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// BcryptHasher hashes and verifies passwords with bcrypt. The password policy
// accepts up to 72 characters, which may exceed bcrypt's 72-byte input limit for
// multibyte text, so every password is SHA-256 pre-hashed into a fixed 64-byte
// hex digest before bcrypt. This accepts every policy-valid password without
// truncation while keeping bcrypt as the salted, adaptive hash.
type BcryptHasher struct {
	cost int
}

// NewBcryptHasher returns a hasher using the given bcrypt cost. A cost below
// bcrypt.MinCost is replaced by bcrypt.DefaultCost by the library, so this
// constructor accepts the configured value as-is.
func NewBcryptHasher(cost int) *BcryptHasher {
	return &BcryptHasher{cost: cost}
}

// Hash returns the bcrypt hash of password. It never logs or stores the
// plaintext. The SHA-256 pre-hash keeps arbitrary-length UTF-8 input within
// bcrypt's 72-byte limit instead of rejecting or truncating it.
func (h *BcryptHasher) Hash(password string) (string, error) {
	sum, err := bcrypt.GenerateFromPassword(prehash(password), h.cost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(sum), nil
}

// Verify reports whether password matches the stored hash. It returns nil on a
// match and a wrapped bcrypt error otherwise, so callers can compare with
// errors.Is against bcrypt.ErrMismatchedHashAndPassword. It applies the same
// pre-hash as Hash so stored hashes verify for any policy-valid password.
func (h *BcryptHasher) Verify(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), prehash(password))
}

// prehash maps a password to a fixed-length, NUL-free hex digest. Hex avoids the
// NUL-byte pitfalls of feeding raw digests to bcrypt and stays below its input
// limit at 64 bytes.
func prehash(password string) []byte {
	sum := sha256.Sum256([]byte(password))
	out := make([]byte, hex.EncodedLen(len(sum)))
	hex.Encode(out, sum[:])
	return out
}
