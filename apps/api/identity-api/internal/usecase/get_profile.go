package usecase

import (
	"context"
	"fmt"
)

// GetProfileUseCase reads one learner's account and profile.
type GetProfileUseCase struct {
	profiles ProfileStore
}

// NewGetProfileUseCase wires the read use case with its port.
func NewGetProfileUseCase(profiles ProfileStore) *GetProfileUseCase {
	return &GetProfileUseCase{profiles: profiles}
}

// Execute returns the account and profile for the caller. A missing account is
// reported as domain.ErrNotFound.
func (uc *GetProfileUseCase) Execute(ctx context.Context, userID string) (ProfileView, error) {
	if err := ctx.Err(); err != nil {
		return ProfileView{}, err
	}
	view, err := uc.profiles.GetUserProfile(ctx, userID)
	if err != nil {
		return ProfileView{}, fmt.Errorf("read profile: %w", err)
	}
	return view, nil
}
