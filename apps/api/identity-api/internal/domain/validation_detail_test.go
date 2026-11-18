package domain

import (
	"errors"
	"strings"
	"testing"
)

// TestPasswordViolationsCarrySpecificRule verifies each rejected password names
// the precise rule it violated (FR-003, SC-004).
func TestPasswordViolationsCarrySpecificRule(t *testing.T) {
	cases := []struct {
		name     string
		password string
		code     ValidationCode
		detail   string
	}{
		{"too short", "Ab1", CodePasswordTooShort, "at least 8 characters"},
		{"too long", repeat("aB3", 24) + "x", CodePasswordTooLong, "at most 72 characters"},
		{"no uppercase", "abcdefg1", CodePasswordNoUppercase, "uppercase letter"},
		{"no digit", "Abcdefgh", CodePasswordNoDigit, "digit"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidatePasswordPolicy(tc.password)
			if !errors.Is(err, ErrWeakPassword) {
				t.Fatalf("error = %v, want ErrWeakPassword", err)
			}
			var ve *ValidationError
			if !errors.As(err, &ve) {
				t.Fatalf("error = %T, want *ValidationError", err)
			}
			if ve.Code != tc.code {
				t.Fatalf("code = %q, want %q", ve.Code, tc.code)
			}
			if !strings.Contains(ve.Detail, tc.detail) {
				t.Fatalf("detail = %q, want it to name %q", ve.Detail, tc.detail)
			}
		})
	}
}

// TestEmailViolationsCarrySpecificRule verifies each rejected address names the
// precise format rule it violated (FR-002, SC-004).
func TestEmailViolationsCarrySpecificRule(t *testing.T) {
	cases := []struct {
		name   string
		email  string
		detail string
	}{
		{"empty", "", "must not be empty"},
		{"missing at", "student.example.com", "contain an @"},
		{"multiple at", "a@@b.com", "single @"},
		{"empty local part", "@example.com", "local part"},
		{"embedded space", "stu dent@example.com", "whitespace"},
		{"empty domain", "student@", "domain after the @"},
		{"missing dot", "student@example", "domain must contain a dot"},
		{"trailing dot", "student@example.", "must not end with a dot"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateEmail(NormalizeEmail(tc.email))
			if !errors.Is(err, ErrInvalidEmail) {
				t.Fatalf("error = %v, want ErrInvalidEmail", err)
			}
			var ve *ValidationError
			if !errors.As(err, &ve) {
				t.Fatalf("error = %T, want *ValidationError", err)
			}
			if ve.Code != CodeEmailMalformed {
				t.Fatalf("code = %q, want %q", ve.Code, CodeEmailMalformed)
			}
			if !strings.Contains(ve.Detail, tc.detail) {
				t.Fatalf("detail = %q, want it to name %q", ve.Detail, tc.detail)
			}
		})
	}
}
