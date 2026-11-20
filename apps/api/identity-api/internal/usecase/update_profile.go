package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"lms-english/identity-service/internal/domain"
)

// UpdateProfileCommand is a partial update. A nil pointer means the request did
// not name that field. For profile fields a non-nil pointer sets the value and an
// empty string clears it.
type UpdateProfileCommand struct {
	UserID      string
	DisplayName *string

	NativeLanguage *string
	Goals          *string
	Timezone       *string
	AvatarURL      *string

	CurrentPassword string
	NewPassword     string
}

// UpdateProfileUseCase applies a partial update to one learner's account and
// profile, and changes their password when asked, after proving the current one.
type UpdateProfileUseCase struct {
	profiles ProfileStore
	users    UserRepository
	verifier PasswordVerifier
	hasher   PasswordHasher
}

// NewUpdateProfileUseCase wires the update use case with its ports.
func NewUpdateProfileUseCase(
	profiles ProfileStore,
	users UserRepository,
	verifier PasswordVerifier,
	hasher PasswordHasher,
) *UpdateProfileUseCase {
	return &UpdateProfileUseCase{profiles: profiles, users: users, verifier: verifier, hasher: hasher}
}

// Execute applies the change and returns the resulting account and profile. A
// password change requires the current password and a new password meeting the
// policy; every refusal leaves the stored password untouched.
func (uc *UpdateProfileUseCase) Execute(ctx context.Context, cmd UpdateProfileCommand) (ProfileView, error) {
	if err := ctx.Err(); err != nil {
		return ProfileView{}, err
	}
	if cmd.UserID == "" {
		return ProfileView{}, fmt.Errorf("update profile: %w", domain.ErrNotFound)
	}

	change := ProfileChange{
		NativeLanguage: cmd.NativeLanguage,
		Goals:          cmd.Goals,
		Timezone:       cmd.Timezone,
		AvatarURL:      cmd.AvatarURL,
	}

	if cmd.DisplayName != nil {
		name := strings.TrimSpace(*cmd.DisplayName)
		if name == "" {
			return ProfileView{}, fmt.Errorf("update profile: %w", domain.ErrInvalidDisplayName)
		}
		change.DisplayName = &name
	}

	if cmd.NewPassword != "" {
		hash, err := uc.passwordHash(ctx, cmd)
		if err != nil {
			return ProfileView{}, err
		}
		change.PasswordHash = &hash
	}

	if err := uc.profiles.UpdateUserProfile(ctx, cmd.UserID, change); err != nil {
		return ProfileView{}, fmt.Errorf("update profile: %w", err)
	}

	view, err := uc.profiles.GetUserProfile(ctx, cmd.UserID)
	if err != nil {
		return ProfileView{}, fmt.Errorf("read updated profile: %w", err)
	}
	return view, nil
}

// passwordHash proves the current password and returns the protected form of the
// new one. The current password is checked first, so a caller who cannot prove
// identity does not learn whether their chosen new password would have passed.
func (uc *UpdateProfileUseCase) passwordHash(ctx context.Context, cmd UpdateProfileCommand) (string, error) {
	if cmd.CurrentPassword == "" {
		return "", fmt.Errorf("change password: %w", domain.ErrInvalidCredentials)
	}
	user, err := uc.users.FindByID(ctx, cmd.UserID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return "", fmt.Errorf("change password: %w", domain.ErrInvalidCredentials)
		}
		return "", fmt.Errorf("change password: load user: %w", err)
	}
	if err := uc.verifier.Verify(user.PasswordHash, cmd.CurrentPassword); err != nil {
		return "", fmt.Errorf("change password: %w", domain.ErrInvalidCredentials)
	}
	if err := domain.ValidatePasswordPolicy(cmd.NewPassword); err != nil {
		return "", fmt.Errorf("change password: %w", err)
	}
	hash, err := uc.hasher.Hash(cmd.NewPassword)
	if err != nil {
		return "", fmt.Errorf("change password: hash: %w", err)
	}
	return hash, nil
}
