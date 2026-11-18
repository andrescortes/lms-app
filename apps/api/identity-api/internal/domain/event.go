package domain

import (
	"fmt"
	"time"
)

// UserRegisteredEvent is the internal signal emitted once per successful
// registration. It carries only non-sensitive fields.
type UserRegisteredEvent struct {
	UserID     string
	Email      string
	Role       Role
	OccurredAt time.Time
}

// Validate enforces the fields the downstream consumer relies on.
func (e UserRegisteredEvent) Validate() error {
	switch {
	case e.UserID == "":
		return fmt.Errorf("event user id is empty: %w", ErrInvalidEvent)
	case e.Email == "":
		return fmt.Errorf("event email is empty: %w", ErrInvalidEvent)
	case !e.Role.Valid():
		return fmt.Errorf("event role %d is not known: %w", e.Role, ErrInvalidEvent)
	case e.OccurredAt.IsZero():
		return fmt.Errorf("event occurred at is zero: %w", ErrInvalidEvent)
	}
	return nil
}
