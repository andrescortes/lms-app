package domain

import (
	"errors"
	"testing"
)

// TestValidateAssignedRole verifies the four roles pass and everything else is refused.
func TestValidateAssignedRole(t *testing.T) {
	for _, role := range []string{"STUDENT", "TEACHER", "ADMIN", "REVIEWER"} {
		if err := ValidateAssignedRole(role); err != nil {
			t.Errorf("role %q: error = %v, want nil", role, err)
		}
	}

	refused := map[string]ValidationCode{
		"admin":  CodeRoleInvalid,
		"WIZARD": CodeRoleInvalid,
		"":       CodeRoleInvalid,
	}
	for role, code := range refused {
		err := ValidateAssignedRole(role)
		if !errors.Is(err, ErrInvalidRole) {
			t.Errorf("role %q: error = %v, want ErrInvalidRole", role, err)
			continue
		}
		var ve *ValidationError
		if !errors.As(err, &ve) || ve.Code != code {
			t.Errorf("role %q: code = %v, want %v", role, ve, code)
		}
	}
}

// TestActorMayChangeRole verifies the self-change guard and that a missing identity
// is refused.
func TestActorMayChangeRole(t *testing.T) {
	if err := ActorMayChangeRole("admin-1", "learner-1"); err != nil {
		t.Errorf("different accounts: error = %v, want nil", err)
	}
	for _, ids := range [][2]string{{"admin-1", "admin-1"}, {"", "learner-1"}, {"admin-1", ""}} {
		if err := ActorMayChangeRole(ids[0], ids[1]); !errors.Is(err, ErrForbidden) {
			t.Errorf("actor %q target %q: error = %v, want ErrForbidden", ids[0], ids[1], err)
		}
	}
}
