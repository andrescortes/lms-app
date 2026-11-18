package domain

import (
	"errors"
	"testing"
	"time"
)

// TestRoleValues locks the enum contract: zero is invalid, and each known role
// round-trips between its integer form and its storage/event string form.
func TestRoleValues(t *testing.T) {
	if RoleUnknown != 0 {
		t.Fatalf("RoleUnknown = %d, want 0", RoleUnknown)
	}
	cases := []struct {
		role Role
		str  string
	}{
		{RoleStudent, "STUDENT"},
		{RoleTeacher, "TEACHER"},
		{RoleAdmin, "ADMIN"},
		{RoleReviewer, "REVIEWER"},
	}
	for _, tc := range cases {
		if got := tc.role.String(); got != tc.str {
			t.Errorf("Role(%d).String() = %q, want %q", tc.role, got, tc.str)
		}
		parsed, err := ParseRole(tc.str)
		if err != nil {
			t.Errorf("ParseRole(%q) error = %v", tc.str, err)
			continue
		}
		if parsed != tc.role {
			t.Errorf("ParseRole(%q) = %d, want %d", tc.str, parsed, tc.role)
		}
	}
}

// TestParseRoleRejectsUnknown ensures unknown and empty strings do not silently
// map to a zero-value role.
func TestParseRoleRejectsUnknown(t *testing.T) {
	for _, s := range []string{"", "student", "GUEST", "  STUDENT"} {
		if _, err := ParseRole(s); !errors.Is(err, ErrUnknownRole) {
			t.Errorf("ParseRole(%q) error = %v, want ErrUnknownRole", s, err)
		}
	}
}

// TestUserValidate covers the stored-user invariants from the data model.
func TestUserValidate(t *testing.T) {
	valid := User{
		ID:           "9f1c7d3e-6b2a-4c88-9f0e-2a5b1c3d4e6f",
		Email:        "student@example.com",
		PasswordHash: "$2a$10$abcdefghijklmnopqrstuvABCDEFGHIJKLMNOPQRSTUVWXYZ0123",
		DisplayName:  "student",
		Role:         RoleStudent,
		CreatedAt:    time.Date(2026, 11, 18, 10, 0, 0, 0, time.UTC),
	}

	cases := []struct {
		name    string
		mutate  func(*User)
		wantErr error
	}{
		{"valid", func(*User) {}, nil},
		{"empty id", func(u *User) { u.ID = "" }, ErrInvalidUser},
		{"empty email", func(u *User) { u.Email = "" }, ErrInvalidUser},
		{"non-normalized email", func(u *User) { u.Email = "Student@Example.com" }, ErrInvalidUser},
		{"empty password hash", func(u *User) { u.PasswordHash = "" }, ErrInvalidUser},
		{"empty display name", func(u *User) { u.DisplayName = "" }, ErrInvalidUser},
		{"unknown role", func(u *User) { u.Role = RoleUnknown }, ErrInvalidUser},
		{"zero created at", func(u *User) { u.CreatedAt = time.Time{} }, ErrInvalidUser},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			u := valid
			tc.mutate(&u)
			err := u.Validate()
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("Validate() error = %v, want %v", err, tc.wantErr)
			}
		})
	}
}

// TestProfileEmptyAtRegistration confirms a freshly created profile has every
// optional field unset, which is the FR-006 registration invariant.
func TestProfileEmptyAtRegistration(t *testing.T) {
	p := NewEmptyProfile("user-1")
	if p.UserID != "user-1" {
		t.Fatalf("UserID = %q, want user-1", p.UserID)
	}
	if p.NativeLanguage != nil || p.Goals != nil || p.Timezone != nil || p.AvatarURL != nil {
		t.Fatal("new profile must have all optional fields unset")
	}
	if err := p.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

// TestProfileValidate rejects a profile with no owning user.
func TestProfileValidate(t *testing.T) {
	if err := NewEmptyProfile("").Validate(); !errors.Is(err, ErrInvalidProfile) {
		t.Fatalf("Validate() error = %v, want ErrInvalidProfile", err)
	}
}

// TestUserRegisteredEventCarriesRequired fields guards the event payload fields.
func TestUserRegisteredEventCarriesRequired(t *testing.T) {
	e := UserRegisteredEvent{
		UserID:     "user-1",
		Email:      "student@example.com",
		Role:       RoleStudent,
		OccurredAt: time.Date(2026, 11, 18, 10, 0, 0, 0, time.UTC),
	}
	if err := e.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	e.Email = ""
	if err := e.Validate(); !errors.Is(err, ErrInvalidEvent) {
		t.Fatalf("Validate() with empty email error = %v, want ErrInvalidEvent", err)
	}
}

// TestNormalizeEmail covers trimming and case normalization.
func TestNormalizeEmail(t *testing.T) {
	cases := map[string]string{
		"Student@Example.com":   "student@example.com",
		"  student@example.com": "student@example.com",
		"STUDENT@EXAMPLE.COM ":  "student@example.com",
	}
	for in, want := range cases {
		if got := NormalizeEmail(in); got != want {
			t.Errorf("NormalizeEmail(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestValidateEmail covers the canonical format rule and its disallowed cases.
func TestValidateEmail(t *testing.T) {
	valid := []string{
		"student@example.com",
		"a.b+c@sub.example.co",
		"USER_1%x@example.io",
	}
	for _, e := range valid {
		if err := ValidateEmail(NormalizeEmail(e)); err != nil {
			t.Errorf("ValidateEmail(%q) error = %v, want nil", e, err)
		}
	}
	invalid := []string{
		"",
		"not-an-email",
		"missing@dot",
		"trailing@dot.",
		"@example.com",
		"has space@example.com",
		"bad*char@example.com",
	}
	for _, e := range invalid {
		if err := ValidateEmail(NormalizeEmail(e)); !errors.Is(err, ErrInvalidEmail) {
			t.Errorf("ValidateEmail(%q) error = %v, want ErrInvalidEmail", e, err)
		}
	}
}

// TestValidatePasswordPolicy covers the 8..72 length bound and character rules.
func TestValidatePasswordPolicy(t *testing.T) {
	cases := []struct {
		name    string
		pw      string
		wantErr error
	}{
		{"exactly 8 with upper and digit", "Abcdefg1", nil},
		{"exactly 72 valid", repeat("aB3", 24), nil},
		{"7 too short", "Abcdef1", ErrWeakPassword},
		{"73 too long", repeat("aB3", 24) + "a", ErrWeakPassword},
		{"no uppercase", "abcdefg1", ErrWeakPassword},
		{"no digit", "Abcdefgh", ErrWeakPassword},
		{"symbols and spaces allowed", "Ab cdef1!@#", nil},
		{"accented uppercase", "Ábcdefg1", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidatePasswordPolicy(tc.pw); !errors.Is(err, tc.wantErr) {
				t.Fatalf("ValidatePasswordPolicy(%q) error = %v, want %v", tc.pw, err, tc.wantErr)
			}
		})
	}
}

// TestValidatePasswordPolicyCountsRunes ensures length is measured in characters,
// not bytes, so multibyte passwords are not rejected prematurely.
func TestValidatePasswordPolicyCountsRunes(t *testing.T) {
	// 8 runes, more than 8 bytes.
	if err := ValidatePasswordPolicy("ÁbcdefG1"); err != nil {
		t.Fatalf("8-rune multibyte password rejected: %v", err)
	}
}

func repeat(s string, n int) string {
	out := ""
	for i := 0; i < n; i++ {
		out += s
	}
	return out
}
