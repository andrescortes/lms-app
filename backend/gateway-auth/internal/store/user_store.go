package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"lms-app/backend/gateway-auth/internal/model"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type UserStore struct {
	db *sql.DB
}

func NewUserStore(db *sql.DB) *UserStore {
	return &UserStore{db: db}
}

func (s *UserStore) CreateUser(ctx context.Context, user *model.User) error {
	const query = `
	INSERT INTO users (id, email, password_hash, full_name, role, cefr_level, created_at, updated_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?);
	`
	_, err := s.db.ExecContext(
		ctx,
		query,
		user.ID,
		user.Email,
		user.PasswordHash,
		user.FullName,
		user.Role,
		user.CEFRLevel,
		user.CreatedAt,
		user.UpdatedAt,
	)

	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed: users.email") {
			return model.ErrEmailAlreadyExists
		}
		return fmt.Errorf("inserting user: %w", err)
	}
	return nil
}

func (s *UserStore) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	const query = `
	SELECT id, email, password_hash, full_name, role, cefr_level, created_at, updated_at
	FROM users
	WHERE email = ?;
	`
	var u model.User
	err := s.db.QueryRowContext(ctx, query, email).Scan(
		&u.ID,
		&u.Email,
		&u.PasswordHash,
		&u.FullName,
		&u.Role,
		&u.CEFRLevel,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrUserNotFound
		}
		return nil, fmt.Errorf("querying user by email: %w", err)
	}
	return &u, nil
}

func (s *UserStore) GetUserById(ctx context.Context, id string) (*model.User, error) {
	const query = `
	SELECT id, email, password_hash, full_name, role, cefr_level, created_at, updated_at
	FROM users
	WHERE id = ?
	LIMIT 1
	`
	var u model.User
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&u.ID,
		&u.Email,
		&u.PasswordHash,
		&u.FullName,
		&u.Role,
		&u.CEFRLevel,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrUserNotFound
		}
		return nil, fmt.Errorf("querying user by id: %w", err)
	}
	return &u, nil
}

func (s *UserStore) SaveRefreshToken(ctx context.Context, rt *model.RefreshToken) error {
	const query = `
		INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, revoked, created_at)
		VALUES (?, ?, ?, ?, 0, ?)
	`

	_, err := s.db.ExecContext(
		ctx,
		query,
		rt.ID,
		rt.UserID,
		rt.TokenHash,
		rt.ExpiresAt,
		rt.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("saving refresh token: %w", err)
	}

	return nil
}

func (s *UserStore) GetRefreshTokenByHash(ctx context.Context, tokenHash string) (*model.RefreshToken, error) {
	const query = `
		SELECT id, user_id, token_hash, expires_at, revoked, created_at
		FROM refresh_tokens
		WHERE token_hash = ?
		LIMIT 1
	`
	var rt model.RefreshToken
	var revokedInt int

	err := s.db.QueryRowContext(ctx, query, tokenHash).Scan(
		&rt.ID,
		&rt.UserID,
		&rt.TokenHash,
		&rt.ExpiresAt,
		&revokedInt,
		&rt.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrInvalidRefreshToken
		}
		return nil, fmt.Errorf("querying refresh token by hash: %w", err)
	}
	rt.Revoked = revokedInt == 1
	return &rt, nil
}

func (s *UserStore) RotateRefreshToken(ctx context.Context, oldTokenID string, newToken *model.RefreshToken) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning refresh rotation tx: %w", err)
	}

	// 1. Revoke if last token if still active, avoid race conditions
	res, err := tx.ExecContext(ctx, "UPDATE refresh_tokens SET revoked = 1 WHERE id = ? AND revoked = 0", oldTokenID)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("revoking old refresh token: %w", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("checking affected rows on token revocation: %w", err)
	}
	if affected == 0 {
		_ = tx.Rollback()
		return model.ErrInvalidRefreshToken
	}

	// 2. insert new token rotated
	const insertQuery = `
		INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, revoked, created_at)
		VALUES (?, ?, ?, ?, 0, ?)
	`
	if _, err := tx.ExecContext(
		ctx,
		insertQuery,
		newToken.ID,
		newToken.UserID,
		newToken.TokenHash,
		newToken.ExpiresAt,
		newToken.CreatedAt,
	); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("inserting rotated refresh token: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing refresh rotation tx: %w", err)
	}

	return nil
}

func (s *UserStore) RevokeAllUserRefreshTokens(ctx context.Context, userID string) error {
	const query = "UPDATE refresh_tokens SET revoked = 1 WHERE user_id = ? AND revoked = 0"
	_, err := s.db.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("revoking all user refresh tokens: %w", err)
	}
	return nil
}

func (s *UserStore) RevokeRefreshTokenByHash(ctx context.Context, tokenHash string) error {
	const query = `UPDATE refresh_tokens SET revoked = 1 WHERE token_hash = ? AND revoked = 0`
	if _, err := s.db.ExecContext(ctx, query, tokenHash); err != nil {
		return fmt.Errorf("revoking refresh token by hash: %w", err)
	}
	return nil
}

func (s *UserStore) UpdatePasswordAndRevokeSessions(
	ctx context.Context,
	userID string,
	newPasswordHash string,
	updatedAt time.Time,
) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning password update tx: %w", err)
	}
	const query = "UPDATE users SET password_hash = ?, updated_at = ? WHERE id = ?"
	res, err := tx.ExecContext(
		ctx,
		query,
		newPasswordHash,
		updatedAt,
		userID,
	)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("updating user password: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("checking affected rows on password update: %w", err)
	}
	if affected == 0 {
		_ = tx.Rollback()
		return model.ErrUserNotFound
	}
	if _, err := tx.ExecContext(
		ctx,
		"UPDATE refresh_tokens SET revoked = 1 WHERE user_id = ? AND revoked = 0",
		userID,
	); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("revoking user sessions after password change: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit password update tx: %w", err)
	}
	return nil
}
