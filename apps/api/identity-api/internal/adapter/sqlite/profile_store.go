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

// ProfileStore reads and writes a learner's account and profile. It satisfies
// the use case port.
var _ usecase.ProfileStore = (*ProfileStore)(nil)

// ProfileStore implements usecase.ProfileStore over the sqlc queries.
type ProfileStore struct {
	db *sql.DB
}

// NewProfileStore wraps db for the profile read and update paths.
func NewProfileStore(db *sql.DB) *ProfileStore {
	return &ProfileStore{db: db}
}

// GetUserProfile returns the account and profile for one user, or
// domain.ErrNotFound when the account does not exist.
func (s *ProfileStore) GetUserProfile(ctx context.Context, userID string) (usecase.ProfileView, error) {
	row, err := sqlc.New(s.db).GetUserWithProfileByID(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return usecase.ProfileView{}, fmt.Errorf("get profile: %w", domain.ErrNotFound)
		}
		return usecase.ProfileView{}, fmt.Errorf("get profile: %w", err)
	}

	role, err := domain.ParseRole(row.Role)
	if err != nil {
		return usecase.ProfileView{}, fmt.Errorf("get profile: %w", err)
	}
	createdAt, err := time.Parse(time.RFC3339Nano, row.CreatedAt)
	if err != nil {
		return usecase.ProfileView{}, fmt.Errorf("get profile: parse created at: %w", err)
	}

	return usecase.ProfileView{
		ID:             row.ID,
		Email:          row.Email,
		DisplayName:    row.DisplayName,
		Role:           role,
		CEFRLevel:      cefrLevelOf(row.CefrLevel),
		CreatedAt:      createdAt,
		NativeLanguage: nullableString(row.NativeLanguage),
		Goals:          nullableString(row.Goals),
		Timezone:       nullableString(row.Timezone),
		AvatarURL:      nullableString(row.AvatarUrl),
	}, nil
}

// UpdateUserProfile writes the fields the change names, in one transaction. Fields
// left nil are not touched. The account and profile always move together: if
// either write fails, neither is kept.
func (s *ProfileStore) UpdateUserProfile(ctx context.Context, userID string, change usecase.ProfileChange) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin profile update: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	q := sqlc.New(tx)

	// The profile row is always updated with the current values merged with the
	// change, so a single statement expresses "keep, set, or clear" per field.
	current, err := q.GetUserWithProfileByID(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("update profile: %w", domain.ErrNotFound)
		}
		return fmt.Errorf("update profile: %w", err)
	}

	if change.DisplayName != nil {
		if err := q.UpdateUserDisplayName(ctx, sqlc.UpdateUserDisplayNameParams{
			DisplayName: *change.DisplayName,
			ID:          userID,
		}); err != nil {
			return fmt.Errorf("update display name: %w", err)
		}
	}
	if change.PasswordHash != nil {
		if err := q.UpdateUserPasswordHash(ctx, sqlc.UpdateUserPasswordHashParams{
			PasswordHash: *change.PasswordHash,
			ID:           userID,
		}); err != nil {
			return fmt.Errorf("update password: %w", err)
		}
	}

	if err := q.UpdateProfileFields(ctx, sqlc.UpdateProfileFieldsParams{
		NativeLanguage: merge(current.NativeLanguage, change.NativeLanguage),
		Goals:          merge(current.Goals, change.Goals),
		Timezone:       merge(current.Timezone, change.Timezone),
		AvatarUrl:      merge(current.AvatarUrl, change.AvatarURL),
		UserID:         userID,
	}); err != nil {
		return fmt.Errorf("update profile fields: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit profile update: %w", err)
	}
	return nil
}

// merge returns the stored value when the change does not name the field, and the
// change otherwise (including an empty string, which clears the column).
func merge(stored interface{}, change *string) interface{} {
	if change == nil {
		return stored
	}
	return *change
}

// nullableString converts a stored nullable column into a pointer, treating an
// empty string as unset so a cleared field reads as null.
func nullableString(value interface{}) *string {
	if value == nil {
		return nil
	}
	text, ok := value.(string)
	if !ok || text == "" {
		return nil
	}
	return &text
}

// cefrLevelOf converts a stored learning level into its domain form.
func cefrLevelOf(value interface{}) *domain.CEFRLevel {
	text := nullableString(value)
	if text == nil {
		return nil
	}
	level := domain.CEFRLevel(*text)
	return &level
}
