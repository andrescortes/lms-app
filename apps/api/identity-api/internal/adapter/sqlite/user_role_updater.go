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

// UserRoleUpdater changes an account's role. It satisfies the use case port.
var _ usecase.UserRoleUpdater = (*UserRoleUpdater)(nil)

// UserRoleUpdater implements usecase.UserRoleUpdater over the returning update.
type UserRoleUpdater struct {
	db *sql.DB
}

// NewUserRoleUpdater wraps db for the role change path.
func NewUserRoleUpdater(db *sql.DB) *UserRoleUpdater {
	return &UserRoleUpdater{db: db}
}

// UpdateUserRole sets the account's role and returns the account as stored after the
// change. A missing account is reported as domain.ErrNotFound.
func (s *UserRoleUpdater) UpdateUserRole(ctx context.Context, userID string, role domain.Role) (domain.User, error) {
	row, err := sqlc.New(s.db).UpdateUserRole(ctx, sqlc.UpdateUserRoleParams{
		Role: role.String(),
		ID:   userID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.User{}, fmt.Errorf("update user role: %w", domain.ErrNotFound)
		}
		return domain.User{}, fmt.Errorf("update user role: %w", err)
	}

	updatedRole, err := domain.ParseRole(row.Role)
	if err != nil {
		return domain.User{}, fmt.Errorf("update user role: %w", err)
	}
	createdAt, err := time.Parse(time.RFC3339Nano, row.CreatedAt)
	if err != nil {
		return domain.User{}, fmt.Errorf("update user role: parse created at: %w", err)
	}

	return domain.User{
		ID:          row.ID,
		Email:       row.Email,
		DisplayName: row.DisplayName,
		Role:        updatedRole,
		CEFRLevel:   cefrLevelOf(row.CefrLevel),
		CreatedAt:   createdAt,
	}, nil
}
