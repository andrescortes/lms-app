package usecase

import (
	"context"
	"time"

	"lms-english/identity-service/internal/domain"
)

// UserRegistrationStore atomically persists a new user, their empty profile, and
// the outbox event row in one transaction. It is the use case's output port; the
// SQLite adapter implements it. A unique-email violation is reported as
// domain.ErrEmailAlreadyExists.
type UserRegistrationStore interface {
	RegisterUser(ctx context.Context, user domain.User, profile domain.Profile, event domain.UserRegisteredEvent) error
}

// PasswordHasher turns a plaintext password into a protected, non-reversible
// representation used for storage.
type PasswordHasher interface {
	Hash(password string) (string, error)
}

// IDGenerator produces unique, opaque identifiers for new users.
type IDGenerator interface {
	New() string
}

// Clock supplies the current time so timestamps are injectable and testable.
type Clock interface {
	Now() time.Time
}

// DomainEventPublisher delivers a recorded event to downstream consumers. The
// use case writes the outbox record and returns; delivery is performed
// out-of-band by the publisher's worker so a slow or failing hook cannot delay
// or fail registration.
type DomainEventPublisher interface {
	Publish(ctx context.Context, eventID string, payload []byte) error
}

// The following ports are mandated by the project constitution for the wider
// identity surface. They are declared here so the application layer owns its
// dependencies; the authentication feature implements them.
type (
	// UserRepository reads and writes users outside the registration write path.
	UserRepository interface {
		FindByEmail(ctx context.Context, email string) (domain.User, error)
	}
	// PasswordVerifier checks a plaintext password against a stored hash.
	PasswordVerifier interface {
		Verify(hash, password string) error
	}
	// TokenProvider issues access tokens for authenticated users.
	TokenProvider interface {
		Issue(ctx context.Context, user domain.User) (string, error)
	}
	// RefreshTokenRepository persists and revokes refresh tokens by jti.
	RefreshTokenRepository interface {
		Store(ctx context.Context, userID, jti, token string, expiresAt time.Time) error
		Revoke(ctx context.Context, jti string) error
	}
)
