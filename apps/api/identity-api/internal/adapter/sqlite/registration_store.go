package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"lms-english/identity-service/internal/adapter/sqlite/sqlc"
	"lms-english/identity-service/internal/domain"
	"lms-english/identity-service/internal/usecase"
)

// RegistrationStore persists a new user, their empty profile, and the outbox
// event in one transaction. It satisfies usecase.UserRegistrationStore.
type RegistrationStore struct {
	db *sql.DB
}

// Compile-time proof that the adapter fulfils the use case port.
var _ usecase.UserRegistrationStore = (*RegistrationStore)(nil)

// NewRegistrationStore wraps db for the registration write path.
func NewRegistrationStore(db *sql.DB) *RegistrationStore {
	return &RegistrationStore{db: db}
}

// eventPayload is the JSON shape stored in the outbox payload column. It holds
// only non-sensitive fields from the UserRegisteredEvent.
type eventPayload struct {
	UserID     string `json:"userId"`
	Email      string `json:"email"`
	Role       string `json:"role"`
	OccurredAt string `json:"occurredAt"`
}

// RegisterUser writes the user, the empty profile, and the outbox event
// atomically. If any insert fails the whole transaction rolls back, so a rejected
// or partial registration leaves no persisted state. A unique-email violation is
// mapped to domain.ErrEmailAlreadyExists.
func (s *RegistrationStore) RegisterUser(ctx context.Context, user domain.User, profile domain.Profile, event domain.UserRegisteredEvent) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin registration transaction: %w", err)
	}
	// Rollback is a no-op once the transaction has committed.
	defer func() { _ = tx.Rollback() }()

	q := sqlc.New(tx)

	if _, err := q.CreateUser(ctx, sqlc.CreateUserParams{
		ID:           user.ID,
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
		DisplayName:  user.DisplayName,
		Role:         user.Role.String(),
		CefrLevel:    nullableCEFRLevel(user.CEFRLevel),
		CreatedAt:    formatUTC(user.CreatedAt),
	}); err != nil {
		if isUniqueEmailViolation(err) {
			return fmt.Errorf("create user: %w", domain.ErrEmailAlreadyExists)
		}
		return fmt.Errorf("create user: %w", err)
	}

	if err := q.CreateProfile(ctx, sqlc.CreateProfileParams{
		UserID:         profile.UserID,
		NativeLanguage: profile.NativeLanguage,
		Goals:          profile.Goals,
		Timezone:       profile.Timezone,
		AvatarUrl:      profile.AvatarURL,
	}); err != nil {
		return fmt.Errorf("create profile: %w", err)
	}

	payload, err := json.Marshal(eventPayload{
		UserID:     event.UserID,
		Email:      event.Email,
		Role:       event.Role.String(),
		OccurredAt: formatUTC(event.OccurredAt),
	})
	if err != nil {
		return fmt.Errorf("marshal event payload: %w", err)
	}

	if err := q.CreateEvent(ctx, sqlc.CreateEventParams{
		ID:        event.UserID,
		EventType: "UserRegistered",
		Payload:   string(payload),
		CreatedAt: formatUTC(event.OccurredAt),
	}); err != nil {
		return fmt.Errorf("create event: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit registration transaction: %w", err)
	}
	return nil
}

// isUniqueEmailViolation reports whether err is the SQLite uniqueness violation
// raised for users.email specifically. modernc.org/sqlite reports constraint
// failures with the column/index name in the message, so the check is narrowed
// to the email constraint rather than treating every constraint failure as a
// duplicate.
func isUniqueEmailViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "UNIQUE constraint failed") && strings.Contains(msg, "users.email")
}

// nullableCEFRLevel maps an unset level to SQL NULL.
func nullableCEFRLevel(level *domain.CEFRLevel) any {
	if level == nil {
		return nil
	}
	return string(*level)
}

// formatUTC renders a timestamp in the canonical ISO-8601 UTC text form used by
// the schema.
func formatUTC(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05.000Z")
}
