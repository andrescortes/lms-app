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
// case port the login flow depends on.
var _ usecase.UserRepository = (*UserRepository)(nil)

// UserRepository implements usecase.UserRepository over the sqlc user queries.
type UserRepository struct {
	db *sql.DB
}

// NewUserRepository wraps db for the user lookup used by login.
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// FindByEmail returns the user with the normalized email, or the domain not
// found error when no account matches.
func (s *UserRepository) FindByEmail(ctx context.Context, email string) (domain.User, error) {
	q := sqlc.New(s.db)
	u, err := q.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.User{}, fmt.Errorf("find user by email: %w", domain.ErrNotFound)
		}
		return domain.User{}, fmt.Errorf("find user by email: %w", err)
	}
	role, err := domain.ParseRole(u.Role)
	if err != nil {
		return domain.User{}, fmt.Errorf("find user by email: %w", err)
	}
	var cefr *domain.CEFRLevel
	if u.CefrLevel != nil {
		lvl := domain.CEFRLevel(u.CefrLevel.(string))
		cefr = &lvl
	}
	createdAt, err := time.Parse(time.RFC3339Nano, u.CreatedAt)
	if err != nil {
		return domain.User{}, fmt.Errorf("find user by email: parse created at: %w", err)
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
