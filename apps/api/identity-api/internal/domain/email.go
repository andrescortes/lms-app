package domain

import (
	"regexp"
	"strings"
)

// emailPattern is the canonical, intentionally-strict address rule. It is
// compiled once at package level. It permits a limited local-part and domain
// character set and requires a dotted domain ending in an alphabetic TLD.
var emailPattern = regexp.MustCompile(`^[A-Za-z0-9._%+-]+@([A-Za-z0-9](?:[A-Za-z0-9-]*[A-Za-z0-9])?\.)+[A-Za-z]{2,}$`)

// NormalizeEmail trims surrounding whitespace and lowercases the address. It is
// applied before validation, uniqueness checks, and storage so the same address
// always compares equal.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// ValidateEmail checks an already-normalized address against emailPattern and
// returns a typed ErrInvalidEmail carrying the specific format rule that was
// violated, so the delivery layer can name it.
func ValidateEmail(email string) error {
	if emailPattern.MatchString(email) {
		return nil
	}
	return newValidation(ErrInvalidEmail, CodeEmailMalformed, emailFormatDetail(email))
}

// emailFormatDetail picks the most useful specific reason for a malformed
// address, without echoing the address itself.
func emailFormatDetail(email string) string {
	switch {
	case email == "":
		return "email must not be empty"
	case strings.Count(email, "@") == 0:
		return "email must contain an @"
	case strings.Count(email, "@") > 1:
		return "email must contain a single @"
	case strings.HasPrefix(email, "@"):
		return "email must have a local part before the @"
	case strings.ContainsAny(email, " \t"):
		return "email must not contain whitespace"
	case strings.HasSuffix(email, "@"):
		return "email must have a domain after the @"
	case !strings.Contains(email[strings.IndexByte(email, '@')+1:], "."):
		return "email domain must contain a dot"
	case strings.HasSuffix(email, "."):
		return "email domain must not end with a dot"
	default:
		return "email must be a valid address"
	}
}
