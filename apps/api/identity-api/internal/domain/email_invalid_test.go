package domain

import (
	"errors"
	"testing"
)

// TestValidateEmailInvalidCases covers the US2 disallowed-address cases declared
// in the canonical rule.
func TestValidateEmailInvalidCases(t *testing.T) {
	cases := map[string]string{
		"missing at":       "student.example.com",
		"missing dot":      "student@example",
		"trailing dot":     "student@example.",
		"empty local part": "@example.com",
		"embedded space":   "stu dent@example.com",
		"disallowed char":  "stu*dent@example.com",
		"empty":            "",
	}
	for name, addr := range cases {
		t.Run(name, func(t *testing.T) {
			if err := ValidateEmail(NormalizeEmail(addr)); !errors.Is(err, ErrInvalidEmail) {
				t.Fatalf("ValidateEmail(%q) error = %v, want ErrInvalidEmail", addr, err)
			}
		})
	}
}

// TestValidateEmailWhitespaceTrimmedValid confirms a well-formed address with
// surrounding spaces is accepted after trimming.
func TestValidateEmailWhitespaceTrimmedValid(t *testing.T) {
	if err := ValidateEmail(NormalizeEmail("  student@example.com  ")); err != nil {
		t.Fatalf("trimmed address rejected: %v", err)
	}
}
