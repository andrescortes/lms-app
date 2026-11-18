package domain

import (
	"errors"
	"testing"
)

// TestValidatePasswordBoundaries covers the US3 rules explicitly, including the
// exact upper bound and the allowance for symbols, spaces, and accents.
func TestValidatePasswordBoundaries(t *testing.T) {
	cases := []struct {
		name    string
		pw      string
		wantErr error
	}{
		{"7 rejected", "Abcdef1", ErrWeakPassword},
		{"8 accepted", "Abcdefg1", nil},
		{"72 accepted", repeat("aB3", 24), nil},
		{"73 rejected", repeat("aB3", 24) + "x", ErrWeakPassword},
		{"no uppercase", "abcdefg1", ErrWeakPassword},
		{"no digit", "Abcdefgh", ErrWeakPassword},
		{"symbols allowed", "Ab!@#$%1", nil},
		{"spaces allowed", "Ab cd ef1", nil},
		{"accented allowed", "Ábcdefg1", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidatePasswordPolicy(tc.pw); !errors.Is(err, tc.wantErr) {
				t.Fatalf("error = %v, want %v", err, tc.wantErr)
			}
		})
	}
}
