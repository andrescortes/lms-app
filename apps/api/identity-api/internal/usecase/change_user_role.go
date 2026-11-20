package usecase

import (
	"context"
	"fmt"

	"lms-english/identity-service/internal/domain"
)

// ChangeUserRoleCommand carries the acting administrator, the target account, and
// the role to set.
type ChangeUserRoleCommand struct {
	// ActorUserID is the caller's id, taken from their verified access token.
	ActorUserID string
	// TargetUserID is the account being changed, taken from the request path.
	TargetUserID string
	// NewRole is the role to assign.
	NewRole string
}

// ChangeUserRoleUseCase assigns a role to another account on an administrator's
// behalf.
type ChangeUserRoleUseCase struct {
	users UserRoleUpdater
}

// NewChangeUserRoleUseCase wires the role change use case with its port.
func NewChangeUserRoleUseCase(users UserRoleUpdater) *ChangeUserRoleUseCase {
	return &ChangeUserRoleUseCase{users: users}
}

// Execute validates the role and the actor, then sets it. Every refusal happens
// before the write, so a refused request never changes the account.
func (uc *ChangeUserRoleUseCase) Execute(ctx context.Context, cmd ChangeUserRoleCommand) (domain.User, error) {
	if err := ctx.Err(); err != nil {
		return domain.User{}, err
	}
	if err := domain.ActorMayChangeRole(cmd.ActorUserID, cmd.TargetUserID); err != nil {
		return domain.User{}, fmt.Errorf("change role: %w", err)
	}
	if err := domain.ValidateAssignedRole(cmd.NewRole); err != nil {
		return domain.User{}, fmt.Errorf("change role: %w", err)
	}

	role, err := domain.ParseRole(cmd.NewRole)
	if err != nil {
		// ValidateAssignedRole already refused anything unparseable; this guards
		// the conversion rather than reporting an empty role.
		return domain.User{}, fmt.Errorf("change role: %w", domain.ErrInvalidRole)
	}

	user, err := uc.users.UpdateUserRole(ctx, cmd.TargetUserID, role)
	if err != nil {
		return domain.User{}, fmt.Errorf("change role: %w", err)
	}
	return user, nil
}
