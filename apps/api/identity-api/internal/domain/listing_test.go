package domain

import (
	"errors"
	"testing"
)

// TestValidatePagination verifies the bounds and that each refusal names its rule.
func TestValidatePagination(t *testing.T) {
	valid := []struct{ page, limit int }{
		{1, 1}, {1, 20}, {7, 100}, {2, 50},
	}
	for _, tc := range valid {
		if err := ValidatePagination(tc.page, tc.limit); err != nil {
			t.Errorf("page %d limit %d: error = %v, want nil", tc.page, tc.limit, err)
		}
	}

	invalid := []struct {
		page, limit int
		code        ValidationCode
	}{
		{0, 20, CodePageInvalid},
		{-1, 20, CodePageInvalid},
		{1, 0, CodeLimitInvalid},
		{1, -5, CodeLimitInvalid},
		{1, 101, CodeLimitInvalid},
	}
	for _, tc := range invalid {
		err := ValidatePagination(tc.page, tc.limit)
		if !errors.Is(err, ErrInvalidFilter) {
			t.Errorf("page %d limit %d: error = %v, want ErrInvalidFilter", tc.page, tc.limit, err)
			continue
		}
		var ve *ValidationError
		if !errors.As(err, &ve) || ve.Code != tc.code {
			t.Errorf("page %d limit %d: code = %v, want %v", tc.page, tc.limit, ve, tc.code)
		}
	}
}

// TestValidateRoleFilter verifies a known role passes and an unknown one is refused.
func TestValidateRoleFilter(t *testing.T) {
	for _, role := range []string{"", "STUDENT", "TEACHER", "ADMIN", "REVIEWER"} {
		if err := ValidateRoleFilter(role); err != nil {
			t.Errorf("role %q: error = %v, want nil", role, err)
		}
	}
	for _, role := range []string{"student", "WIZARD", "ADMIN "} {
		if err := ValidateRoleFilter(role); !errors.Is(err, ErrInvalidFilter) {
			t.Errorf("role %q: error = %v, want ErrInvalidFilter", role, err)
		}
	}
}

// TestNormalizeSearch verifies trimming, lowercasing, and that spaces mean no search.
func TestNormalizeSearch(t *testing.T) {
	cases := map[string]string{
		"":            "",
		"   ":         "",
		" Ana ":       "ana",
		"ANa@Example": "ana@example",
		"pass b2":     "pass b2",
	}
	for in, want := range cases {
		if got := NormalizeSearch(in); got != want {
			t.Errorf("NormalizeSearch(%q) = %q, want %q", in, got, want)
		}
	}
}
