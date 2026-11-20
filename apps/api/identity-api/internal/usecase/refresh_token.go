package usecase

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"lms-english/identity-service/internal/domain"
)

// RefreshCommand carries the refresh token the caller presented.
type RefreshCommand struct {
	RefreshToken string
}

// RefreshTokenUseCase rotates a refresh token and issues a new token pair. It
// depends only on the use case ports and the domain package.
type RefreshTokenUseCase struct {
	users   UserRepository
	tokens  TokenProvider
	refresh RefreshTokenRepository
	ids     IDGenerator
	clock   Clock
}

// NewRefreshTokenUseCase wires the refresh use case with its output ports.
func NewRefreshTokenUseCase(
	users UserRepository,
	tokens TokenProvider,
	refresh RefreshTokenRepository,
	ids IDGenerator,
	clock Clock,
) *RefreshTokenUseCase {
	return &RefreshTokenUseCase{users: users, tokens: tokens, refresh: refresh, ids: ids, clock: clock}
}

// Execute rotates the presented refresh token and returns a new access token and
// a new refresh token. Any token that cannot be rotated is reported as
// domain.ErrInvalidToken, so the delivery layer never reveals why it failed.
func (uc *RefreshTokenUseCase) Execute(ctx context.Context, cmd RefreshCommand) (LoginUserResult, error) {
	if err := ctx.Err(); err != nil {
		return LoginUserResult{}, err
	}
	if cmd.RefreshToken == "" {
		return LoginUserResult{}, domain.ErrInvalidToken
	}

	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return LoginUserResult{}, fmt.Errorf("generate refresh token: %w", err)
	}
	replacement := hex.EncodeToString(raw[:])
	expiresAt := uc.clock.Now().UTC().Add(refreshTokenTTL)

	rotation, err := uc.refresh.Rotate(ctx, cmd.RefreshToken, uc.ids.New(), replacement, expiresAt)
	if err != nil {
		return LoginUserResult{}, fmt.Errorf("rotate refresh token: %w", err)
	}
	// Every non-rotated outcome is the same refusal to the caller: spent,
	// expired, and unknown tokens must be indistinguishable.
	if rotation.Outcome != RotationRotated {
		return LoginUserResult{}, domain.ErrInvalidToken
	}

	user, err := uc.users.FindByID(ctx, rotation.UserID)
	if err != nil {
		return LoginUserResult{}, fmt.Errorf("load user: %w", err)
	}
	accessToken, err := uc.tokens.Issue(ctx, user)
	if err != nil {
		return LoginUserResult{}, fmt.Errorf("issue token: %w", err)
	}

	return LoginUserResult{AccessToken: accessToken, RefreshToken: replacement}, nil
}
