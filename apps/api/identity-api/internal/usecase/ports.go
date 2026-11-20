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
		FindByID(ctx context.Context, id string) (domain.User, error)
	}
	// PasswordVerifier checks a plaintext password against a stored hash.
	PasswordVerifier interface {
		Verify(hash, password string) error
	}
	// TokenProvider issues access tokens for authenticated users.
	TokenProvider interface {
		Issue(ctx context.Context, user domain.User) (string, error)
	}
	// TokenVerifier validates an access token and reports the identity it
	// carries. The HTTP adapter depends on this port rather than on the security
	// adapter, so the two adapters stay independent. A refusal is reported as
	// domain.ErrInvalidToken.
	TokenVerifier interface {
		Verify(ctx context.Context, token string) (TokenClaims, error)
	}
	// RefreshTokenRepository persists and revokes refresh tokens by jti.
	RefreshTokenRepository interface {
		Store(ctx context.Context, userID, jti, token string, expiresAt time.Time) error
		Revoke(ctx context.Context, jti string) error
		// Rotate consumes the presented refresh token and, when it is active,
		// replaces it with the replacement token in one transaction. A token that
		// was already spent revokes the caller's whole chain. The caller decides
		// the replacement's jti so identifier policy stays in the application layer.
		Rotate(ctx context.Context, presented, newJTI, replacement string, expiresAt time.Time) (Rotation, error)
		// RevokeByJTI ends the session named by one refresh token, but only when
		// it belongs to the given user and is still active. A missing token, one
		// owned by another user, an expired one, and an already-revoked one are
		// all reported as domain.ErrInvalidToken.
		RevokeByJTI(ctx context.Context, userID, jti string) error
	}
)

// RotationOutcome classifies what a rotation attempt found.
type RotationOutcome int

const (
	// RotationRotated reports an active token that was revoked and replaced.
	RotationRotated RotationOutcome = iota + 1
	// RotationSpent reports a token that had already been revoked. The whole
	// chain for its user is revoked as a precaution.
	RotationSpent
	// RotationExpired reports a token whose lifetime had passed.
	RotationExpired
	// RotationUnknown reports a token the service never issued.
	RotationUnknown
)

// Rotation reports the outcome of a rotation attempt and, when it succeeded, the
// user whose tokens were rotated.
type Rotation struct {
	Outcome RotationOutcome
	UserID  string
}

// TokenClaims is the identity a verified access token carries.
type TokenClaims struct {
	UserID string
	Role   string
}

// ProfileView is the account and profile a caller reads about themselves. It
// carries no password material.
type ProfileView struct {
	ID             string
	Email          string
	DisplayName    string
	Role           domain.Role
	CEFRLevel      *domain.CEFRLevel
	CreatedAt      time.Time
	NativeLanguage *string
	Goals          *string
	Timezone       *string
	AvatarURL      *string
}

// ProfileChange describes a partial update. A nil pointer leaves the stored value
// untouched; a non-nil pointer sets it, and a nil pointer inside an account field
// that is present means the value was supplied but empty.
type ProfileChange struct {
	// DisplayName is nil when the request did not name it.
	DisplayName *string
	// PasswordHash is nil when no password change was requested.
	PasswordHash *string
	// Profile fields: nil leaves the value, non-nil sets it (an empty string clears).
	NativeLanguage *string
	Goals          *string
	Timezone       *string
	AvatarURL      *string
}

// ProfileStore reads and writes one learner's account and profile.
type ProfileStore interface {
	GetUserProfile(ctx context.Context, userID string) (ProfileView, error)
	UpdateUserProfile(ctx context.Context, userID string, change ProfileChange) error
}

// UserSummary is one account as it appears in the admin list. It carries no
// password material.
type UserSummary struct {
	ID          string
	Email       string
	DisplayName string
	Role        domain.Role
	CEFRLevel   *domain.CEFRLevel
	CreatedAt   time.Time
}

// ListFilter narrows the admin list. An empty field means no filter on that field.
type ListFilter struct {
	Role      string
	CEFRLevel string
	Search    string
	Limit     int
	Offset    int
}

// UserPage is one page of the admin list and the total number of accounts matching
// the filter.
type UserPage struct {
	Items []UserSummary
	Total int
}

// UserLister reads pages of accounts for the admin list.
type UserLister interface {
	ListUsers(ctx context.Context, filter ListFilter) (UserPage, error)
}
