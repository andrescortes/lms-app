package usecase

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"lms-english/identity-service/internal/domain"
)

// LoginUserCommand carries the submitted credentials.
type LoginUserCommand struct {
	Email    string
	Password string
}

// LoginUserResult carries the issued tokens for a valid credential pair.
type LoginUserResult struct {
	AccessToken  string
	RefreshToken string
}

// refreshTokenTTL is the refresh token lifetime for a login-issued token.
const refreshTokenTTL = 7 * 24 * time.Hour

// LoginUserUseCase orchestrates credential verification and token issuance.
// It depends only on the use case ports and the domain package.
type LoginUserUseCase struct {
	repo    UserRepository
	verify  PasswordVerifier
	token   TokenProvider
	refresh RefreshTokenRepository
	ids     IDGenerator
	clock   Clock
}

// NewLoginUserUseCase wires the login use case with its output ports. Pass a
// nil refresh store only in tests that do not persist tokens.
func NewLoginUserUseCase(
	repo UserRepository,
	verify PasswordVerifier,
	token TokenProvider,
	refresh RefreshTokenRepository,
	ids IDGenerator,
	clock Clock,
) *LoginUserUseCase {
	return &LoginUserUseCase{repo: repo, verify: verify, token: token, refresh: refresh, ids: ids, clock: clock}
}

// Execute verifies the credentials and issues the access and refresh tokens.
// It returns the domain sentinels for invalid credentials and validation
// failures so the delivery layer maps them to their standard outcomes.
func (uc *LoginUserUseCase) Execute(ctx context.Context, cmd LoginUserCommand) (LoginUserResult, error) {
	if err := ctx.Err(); err != nil {
		return LoginUserResult{}, err
	}

	email := domain.NormalizeEmail(cmd.Email)
	if err := domain.ValidateEmail(email); err != nil {
		return LoginUserResult{}, fmt.Errorf("validate email: %w", err)
	}
	user, err := uc.repo.FindByEmail(ctx, email)
	if err != nil {
		// A missing account and a wrong password must be indistinguishable, so
		// the lookup failure is reported as failed authentication, never as a
		// not-found that would reveal whether the account exists.
		if errors.Is(err, domain.ErrNotFound) {
			return LoginUserResult{}, domain.ErrInvalidCredentials
		}
		return LoginUserResult{}, fmt.Errorf("find user: %w", err)
	}
	if err := uc.verify.Verify(user.PasswordHash, cmd.Password); err != nil {
		return LoginUserResult{}, domain.ErrInvalidCredentials
	}

	accessToken, err := uc.token.Issue(ctx, user)
	if err != nil {
		return LoginUserResult{}, fmt.Errorf("issue token: %w", err)
	}

	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return LoginUserResult{}, fmt.Errorf("generate refresh token: %w", err)
	}
	refreshToken := hex.EncodeToString(b[:])
	expiresAt := uc.clock.Now().UTC().Add(refreshTokenTTL)
	if err := uc.refresh.Store(ctx, user.ID, uc.ids.New(), refreshToken, expiresAt); err != nil {
		return LoginUserResult{}, fmt.Errorf("store refresh token: %w", err)
	}

	return LoginUserResult{AccessToken: accessToken, RefreshToken: refreshToken}, nil
}
