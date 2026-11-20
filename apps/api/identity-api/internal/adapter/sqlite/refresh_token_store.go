package sqlite

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"lms-english/identity-service/internal/adapter/sqlite/sqlc"
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
