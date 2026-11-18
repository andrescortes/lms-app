package domain

import (
	"fmt"
	"unicode"
)

// Password policy bounds, inclusive, counted in characters (runes).
const (
	MinPasswordLength = 8
	MaxPasswordLength = 72
)

// ValidatePasswordPolicy enforces the registration password rule: length within
// MinPasswordLength and MaxPasswordLength characters, at least one uppercase
// letter, and at least one digit. Length is counted in characters (runes), so
// multibyte passwords are accepted; the hasher pre-hashes before bcrypt to
// satisfy bcrypt's byte limit. A violation returns a typed ErrWeakPassword
// carrying the specific rule that failed.
func ValidatePasswordPolicy(password string) error {
	runes := []rune(password)
	switch {
	case len(runes) < MinPasswordLength:
		return newValidation(ErrWeakPassword, CodePasswordTooShort,
			fmt.Sprintf("password must be at least %d characters", MinPasswordLength))
	case len(runes) > MaxPasswordLength:
		return newValidation(ErrWeakPassword, CodePasswordTooLong,
			fmt.Sprintf("password must be at most %d characters", MaxPasswordLength))
	}

	var hasUpper, hasDigit bool
	for _, r := range runes {
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsDigit(r):
			hasDigit = true
		}
	}
	switch {
	case !hasUpper:
		return newValidation(ErrWeakPassword, CodePasswordNoUppercase,
			"password must contain at least one uppercase letter")
	case !hasDigit:
		return newValidation(ErrWeakPassword, CodePasswordNoDigit,
			"password must contain at least one digit")
	}
	return nil
}
