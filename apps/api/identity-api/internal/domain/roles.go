package domain

import "fmt"

// ValidateAssignedRole checks a role an administrator is assigning. The accepted
// set is exactly the four stored roles, compared case-sensitively, and the error
// names the role.
func ValidateAssignedRole(role string) error {
	if _, err := ParseRole(role); err != nil {
		return newValidation(ErrInvalidRole, CodeRoleInvalid,
			fmt.Sprintf("role must be one of STUDENT, TEACHER, ADMIN, REVIEWER; got %q", role))
	}
	return nil
}

// ActorMayChangeRole reports whether the acting administrator may change the target
// account's role. An administrator changing their own role is refused, so nobody
// can strip or escalate their own privileges through this path.
func ActorMayChangeRole(actorID, targetID string) error {
	if actorID == "" || targetID == "" {
		return fmt.Errorf("role change needs an actor and a target: %w", ErrForbidden)
	}
	if actorID == targetID {
		return fmt.Errorf("an administrator cannot change their own role: %w", ErrForbidden)
	}
	return nil
}
