package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"lms-english/identity-service/internal/adapter/sqlite/sqlc"
	"lms-english/identity-service/internal/domain"
	"lms-english/identity-service/internal/usecase"
)

// UserRepository reads users through the sqlc queries. It satisfies the use
// case port the login and refresh flows depend on.
var _ usecase.UserRepository = (*UserRepository)(nil)

// UserRepository implements usecase.UserRepository over the sqlc user queries.
type UserRepository struct {
	db *sql.DB
}

// NewUserRepository wraps db for the user lookups used by login and refresh.
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// FindByEmail returns the user with the normalized email, or the domain not
// found error when no account matches.
func (s *UserRepository) FindByEmail(ctx context.Context, email string) (domain.User, error) {
	u, err := sqlc.New(s.db).GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.User{}, fmt.Errorf("find user by email: %w", domain.ErrNotFound)
		}
		return domain.User{}, fmt.Errorf("find user by email: %w", err)
	}
	return toDomainUser(u, "find user by email")
}

// FindByID returns the user with the given id, or the domain not found error
// when no account matches.
func (s *UserRepository) FindByID(ctx context.Context, id string) (domain.User, error) {
	u, err := sqlc.New(s.db).GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.User{}, fmt.Errorf("find user by id: %w", domain.ErrNotFound)
		}
		return domain.User{}, fmt.Errorf("find user by id: %w", err)
	}
	return toDomainUser(u, "find user by id")
}

// toDomainUser converts a stored user row into the domain entity, wrapping any
// parse failure with the caller's context.
func toDomainUser(u sqlc.User, context string) (domain.User, error) {
	role, err := domain.ParseRole(u.Role)
	if err != nil {
		return domain.User{}, fmt.Errorf("%s: %w", context, err)
	}
	var cefr *domain.CEFRLevel
	if u.CefrLevel != nil {
		if level, ok := u.CefrLevel.(string); ok {
			lvl := domain.CEFRLevel(level)
			cefr = &lvl
		}
	}
	createdAt, err := time.Parse(time.RFC3339Nano, u.CreatedAt)
	if err != nil {
		return domain.User{}, fmt.Errorf("%s: parse created at: %w", context, err)
	}
	return domain.User{
		ID:           u.ID,
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		DisplayName:  u.DisplayName,
		Role:         role,
		CEFRLevel:    cefr,
		CreatedAt:    createdAt,
	}, nil
}
