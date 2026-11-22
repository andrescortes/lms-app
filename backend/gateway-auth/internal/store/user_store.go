package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"lms-app/backend/gateway-auth/internal/model"
	"strings"

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
