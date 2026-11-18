package domain

import (
	"fmt"
	"time"
)

// Role is the capability tier assigned to a user. The zero value is invalid so a
// missing role cannot silently pass as a real one.
type Role int

const (
	// RoleUnknown is the invalid/unset zero value.
	RoleUnknown Role = iota
	// RoleStudent is the default role assigned at registration.
	RoleStudent
	// RoleTeacher can create and manage learning content.
	RoleTeacher
	// RoleAdmin administers the platform.
	RoleAdmin
	// RoleReviewer reviews learning content.
	RoleReviewer
)

// roleStrings maps each role to its canonical persistence and event string form.
// These values must match the CHECK constraint on users.role.
var roleStrings = map[Role]string{
	RoleStudent:  "STUDENT",
	RoleTeacher:  "TEACHER",
	RoleAdmin:    "ADMIN",
	RoleReviewer: "REVIEWER",
}

// roleByString is the reverse of roleStrings for parsing stored values.
var roleByString = map[string]Role{
	"STUDENT":  RoleStudent,
	"TEACHER":  RoleTeacher,
	"ADMIN":    RoleAdmin,
	"REVIEWER": RoleReviewer,
}

// String returns the canonical uppercase name of the role, or "UNKNOWN" when the
// value is not a known role.
func (r Role) String() string {
	if s, ok := roleStrings[r]; ok {
		return s
	}
	return "UNKNOWN"
}

// Valid reports whether the role is a known, non-zero value.
func (r Role) Valid() bool {
	_, ok := roleStrings[r]
	return ok
}

// ParseRole converts a stored or inbound role string into a Role. An unknown or
// empty value returns ErrUnknownRole rather than a zero-value role.
func ParseRole(s string) (Role, error) {
	if r, ok := roleByString[s]; ok {
		return r, nil
	}
	return RoleUnknown, fmt.Errorf("role %q: %w", s, ErrUnknownRole)
}

// CEFRLevel is a language-proficiency level. The valid set is finalized by the
// learning feature; at registration it is unset.
type CEFRLevel string

// User is a registered person. One user has exactly one Profile.
type User struct {
	ID           string
	Email        string
	PasswordHash string
	DisplayName  string
	Role         Role
	CEFRLevel    *CEFRLevel
	CreatedAt    time.Time
}

// Validate enforces the stored-user invariants: every required field is present
// and normalized, and the role is a known non-zero value.
func (u User) Validate() error {
	switch {
	case u.ID == "":
		return fmt.Errorf("user id is empty: %w", ErrInvalidUser)
	case u.Email == "":
		return fmt.Errorf("email is empty: %w", ErrInvalidUser)
	case u.Email != NormalizeEmail(u.Email):
		return fmt.Errorf("email is not normalized: %w", ErrInvalidUser)
	case u.PasswordHash == "":
		return fmt.Errorf("password hash is empty: %w", ErrInvalidUser)
	case u.DisplayName == "":
		return fmt.Errorf("display name is empty: %w", ErrInvalidUser)
	case !u.Role.Valid():
		return fmt.Errorf("role %d is not known: %w", u.Role, ErrInvalidUser)
	case u.CreatedAt.IsZero():
		return fmt.Errorf("created at is zero: %w", ErrInvalidUser)
	}
	return nil
}
