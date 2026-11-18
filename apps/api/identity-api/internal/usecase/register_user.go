// Package usecase holds the Identity service application logic. It defines the
// output ports it needs and depends only on the domain and the standard library.
package usecase

import (
	"context"
	"fmt"
	"strings"

	"lms-english/identity-service/internal/domain"
)

// RegisterUserCommand carries the validated-by-shape registration input.
type RegisterUserCommand struct {
	Email       string
	Password    string
	DisplayName string
}

// RegisterUserResult reports the created account.
type RegisterUserResult struct {
	UserID string
}

// RegisterUserUseCase orchestrates a student registration: it normalizes and
// validates input, hashes the password, and asks the store to write the user,
// the empty profile, and the outbox event in one atomic step. It does not
// deliver the event; the outbox worker does that after commit.
type RegisterUserUseCase struct {
	store  UserRegistrationStore
	hasher PasswordHasher
	ids    IDGenerator
	clock  Clock
	pub    DomainEventPublisher
}

// NewRegisterUserUseCase wires the use case with its output ports. Pass a nil
// publisher only in tests; the store still records the event.
func NewRegisterUserUseCase(
	store UserRegistrationStore,
	hasher PasswordHasher,
	ids IDGenerator,
	clock Clock,
	pub DomainEventPublisher,
) *RegisterUserUseCase {
	return &RegisterUserUseCase{store: store, hasher: hasher, ids: ids, clock: clock, pub: pub}
}

// Execute performs a single registration. It returns a domain sentinel error
// (wrapped for context) on validation, hashing, or store failure, and nil on
// success. It never invokes downstream hooks directly and never waits for
// delivery.
func (uc *RegisterUserUseCase) Execute(ctx context.Context, cmd RegisterUserCommand) (RegisterUserResult, error) {
	if err := ctx.Err(); err != nil {
		return RegisterUserResult{}, err
	}

	email := domain.NormalizeEmail(cmd.Email)
	if err := domain.ValidateEmail(email); err != nil {
		return RegisterUserResult{}, fmt.Errorf("validate email: %w", err)
	}
	if err := domain.ValidatePasswordPolicy(cmd.Password); err != nil {
		return RegisterUserResult{}, fmt.Errorf("validate password: %w", err)
	}

	hash, err := uc.hasher.Hash(cmd.Password)
	if err != nil {
		return RegisterUserResult{}, fmt.Errorf("hash password: %w", err)
	}

	user := domain.User{
		ID:           uc.ids.New(),
		Email:        email,
		PasswordHash: hash,
		DisplayName:  displayNameOrDefault(cmd.DisplayName, email),
		Role:         domain.RoleStudent,
		CreatedAt:    uc.clock.Now().UTC(),
	}
	if err := user.Validate(); err != nil {
		return RegisterUserResult{}, fmt.Errorf("validate user: %w", err)
	}

	profile := domain.NewEmptyProfile(user.ID)

	event := domain.UserRegisteredEvent{
		UserID:     user.ID,
		Email:      user.Email,
		Role:       user.Role,
		OccurredAt: user.CreatedAt,
	}
	if err := event.Validate(); err != nil {
		return RegisterUserResult{}, fmt.Errorf("validate event: %w", err)
	}

	if err := uc.store.RegisterUser(ctx, user, profile, event); err != nil {
		return RegisterUserResult{}, fmt.Errorf("register user: %w", err)
	}

	return RegisterUserResult{UserID: user.ID}, nil
}

// displayNameOrDefault returns the trimmed display name, or the email local part
// when the caller left it blank. The email is expected to be normalized.
func displayNameOrDefault(displayName, email string) string {
	if name := strings.TrimSpace(displayName); name != "" {
		return name
	}
	if at := strings.IndexByte(email, '@'); at > 0 {
		return email[:at]
	}
	return email
}
