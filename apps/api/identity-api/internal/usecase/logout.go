package usecase

import (
	"context"
	"fmt"

	"lms-english/identity-service/internal/domain"
)

// LogoutCommand carries the caller's identity and the session to end.
type LogoutCommand struct {
	// UserID is the caller's id, taken from their verified access token. It is
	// never read from the request body.
	UserID string
	// RefreshToken names the session to end.
	RefreshToken string
}

// LogoutUseCase ends one session. It knows the owner's identity and the refresh
// token, and nothing about how tokens are stored.
type LogoutUseCase struct {
	refresh RefreshTokenRepository
}

// NewLogoutUseCase wires the logout use case with its output port.
func NewLogoutUseCase(refresh RefreshTokenRepository) *LogoutUseCase {
	return &LogoutUseCase{refresh: refresh}
}

// Execute ends the session named by the refresh token, but only when it belongs
// to the caller. Every refusal is reported as domain.ErrInvalidToken so a caller
// cannot tell a foreign token from a missing one.
func (uc *LogoutUseCase) Execute(ctx context.Context, cmd LogoutCommand) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if cmd.UserID == "" {
		return domain.ErrInvalidToken
	}
	if cmd.RefreshToken == "" {
		return domain.ErrInvalidToken
	}
	if err := uc.refresh.RevokeByJTI(ctx, cmd.UserID, cmd.RefreshToken); err != nil {
		return fmt.Errorf("end session: %w", err)
	}
	return nil
}
