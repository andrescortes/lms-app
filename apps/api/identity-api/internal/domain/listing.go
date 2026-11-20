package domain

import (
	"fmt"
	"strings"
)

// Pagination bounds for the admin user list.
const (
	// DefaultUserPageSize is the page size used when a caller names none.
	DefaultUserPageSize = 20
	// MaxUserPageSize is the largest page a caller may request.
	MaxUserPageSize = 100
)

// ValidatePagination checks a page number and a page size. A page below one, or a
// size outside one through the maximum, is refused naming the parameter.
func ValidatePagination(page, limit int) error {
	switch {
	case page < 1:
		return newValidation(ErrInvalidFilter, CodePageInvalid, fmt.Sprintf("page must be at least 1, got %d", page))
	case limit < 1:
		return newValidation(ErrInvalidFilter, CodeLimitInvalid, fmt.Sprintf("limit must be at least 1, got %d", limit))
	case limit > MaxUserPageSize:
		return newValidation(ErrInvalidFilter, CodeLimitInvalid, fmt.Sprintf("limit must be at most %d, got %d", MaxUserPageSize, limit))
	}
	return nil
}

// ValidateRoleFilter checks a role filter. An empty value means no filter. A value
// that is not a known role is refused naming the parameter.
func ValidateRoleFilter(role string) error {
	if role == "" {
		return nil
	}
	if _, err := ParseRole(role); err != nil {
		return newValidation(ErrInvalidFilter, CodeRoleInvalid,
			fmt.Sprintf("role must be one of STUDENT, TEACHER, ADMIN, REVIEWER; got %q", role))
	}
	return nil
}

// NormalizeSearch trims a search fragment and lowercases it for comparison. A
// fragment of only spaces means no search, so it returns empty rather than a search
// for spaces.
func NormalizeSearch(search string) string {
	return strings.ToLower(strings.TrimSpace(search))
}
