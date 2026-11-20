package sqlite

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"lms-english/identity-service/internal/adapter/sqlite/sqlc"
	"lms-english/identity-service/internal/domain"
	"lms-english/identity-service/internal/usecase"
)

// RefreshTokenRepository persists and revokes refresh tokens by jti. It
// satisfies the use case port.
var _ usecase.RefreshTokenRepository = (*RefreshTokenRepository)(nil)

// RefreshTokenRepository implements usecase.RefreshTokenRepository over the
// sqlc refresh token queries.
type RefreshTokenRepository struct {
	db *sql.DB
}

// NewRefreshTokenRepository wraps db for the refresh token store.
func NewRefreshTokenRepository(db *sql.DB) *RefreshTokenRepository {
	return &RefreshTokenRepository{db: db}
}

// Store persists a refresh token with its hash, owner, and expiry; the token
// itself is reduced to its SHA-256 hash so the raw value is never stored.
func (s *RefreshTokenRepository) Store(ctx context.Context, userID, jti, token string, expiresAt time.Time) error {
	sum := sha256.Sum256([]byte(token))
	tokenHash := hex.EncodeToString(sum[:])
	if err := sqlc.New(s.db).InsertRefreshToken(ctx, sqlc.InsertRefreshTokenParams{
		Jti:       jti,
		TokenHash: tokenHash,
		UserID:    userID,
		ExpiresAt: formatUTC(expiresAt),
		CreatedAt: formatUTC(time.Now().UTC()),
	}); err != nil {
		return fmt.Errorf("store refresh token: %w", err)
	}
	return nil
}

// Revoke marks a refresh token revoked by jti.
func (s *RefreshTokenRepository) Revoke(ctx context.Context, jti string) error {
	if err := sqlc.New(s.db).RevokeRefreshToken(ctx, sqlc.RevokeRefreshTokenParams{
		RevokedAt: formatUTC(time.Now().UTC()),
		Jti:       jti,
	}); err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}
	return nil
}

// Rotate consumes the presented refresh token and, when it is active, replaces
// it with the replacement in one transaction. A spent token revokes every active
// token for its user. The caller supplies the replacement expiry so the
// lifetime is decided in one place.
func (s *RefreshTokenRepository) Rotate(ctx context.Context, presented, newJTI, replacement string, expiresAt time.Time) (usecase.Rotation, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return usecase.Rotation{}, fmt.Errorf("begin rotation transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	q := sqlc.New(tx)
	now := time.Now().UTC()

	row, err := q.GetRefreshTokenByTokenHash(ctx, hashToken(presented))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return usecase.Rotation{Outcome: usecase.RotationUnknown}, nil
		}
		return usecase.Rotation{}, fmt.Errorf("load refresh token: %w", err)
	}

	// A spent token is the only signal that a token leaked. Refusing it while
	// leaving the user's other sessions alive would let an attacker keep using
	// them, so the whole chain goes.
	if row.RevokedAt != nil {
		if err := q.RevokeAllRefreshTokensForUser(ctx, sqlc.RevokeAllRefreshTokensForUserParams{
			RevokedAt: formatUTC(now),
			UserID:    row.UserID,
		}); err != nil {
			return usecase.Rotation{}, fmt.Errorf("revoke token chain: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return usecase.Rotation{}, fmt.Errorf("commit chain revocation: %w", err)
		}
		return usecase.Rotation{Outcome: usecase.RotationSpent, UserID: row.UserID}, nil
	}

	expiresAtStored, err := time.Parse(time.RFC3339Nano, row.ExpiresAt)
	if err != nil {
		return usecase.Rotation{}, fmt.Errorf("parse token expiry: %w", err)
	}
	if !now.Before(expiresAtStored) {
		return usecase.Rotation{Outcome: usecase.RotationExpired, UserID: row.UserID}, nil
	}

	if err := q.RevokeRefreshToken(ctx, sqlc.RevokeRefreshTokenParams{
		RevokedAt: formatUTC(now),
		Jti:       row.Jti,
	}); err != nil {
		return usecase.Rotation{}, fmt.Errorf("revoke presented token: %w", err)
	}
	if err := q.InsertRefreshToken(ctx, sqlc.InsertRefreshTokenParams{
		Jti:       newJTI,
		TokenHash: hashToken(replacement),
		UserID:    row.UserID,
		ExpiresAt: formatUTC(expiresAt),
		CreatedAt: formatUTC(now),
	}); err != nil {
		return usecase.Rotation{}, fmt.Errorf("insert replacement token: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return usecase.Rotation{}, fmt.Errorf("commit rotation: %w", err)
	}
	return usecase.Rotation{Outcome: usecase.RotationRotated, UserID: row.UserID}, nil
}

// hashToken reduces a refresh token to the SHA-256 hex form stored in the
// token_hash column.
func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// RevokeByJTI ends one session, but only when the token names the given user's
// active session. The ownership, activity, and expiry checks all live in the
// single UPDATE, so no other writer can slip between the check and the revoke.
// A refusal never says which of those failed.
func (s *RefreshTokenRepository) RevokeByJTI(ctx context.Context, userID, jti string) error {
	now := time.Now().UTC()
	affected, err := sqlc.New(s.db).RevokeRefreshTokenByOwner(ctx, sqlc.RevokeRefreshTokenByOwnerParams{
		RevokedAt: formatUTC(now),
		Jti:       jti,
		UserID:    userID,
		ExpiresAt: formatUTC(now),
	})
	if err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}
	if affected == 0 {
		return domain.ErrInvalidToken
	}
	return nil
}
