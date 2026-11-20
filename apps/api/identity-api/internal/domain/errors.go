// Package domain holds the Identity service business entities, value objects,
// and sentinel errors. It imports only the Go standard library so business rules
// stay independent of transport, persistence, and authentication libraries.
package domain

import (
	"errors"
	"fmt"
)

// Validation and conflict sentinels returned by domain rules and translated to
// HTTP outcomes by the delivery layer.
var (
	// ErrInvalidEmail reports an address that is not well-formed after normalization.
	ErrInvalidEmail = errors.New("invalid email")
	// ErrWeakPassword reports a password that fails the length, uppercase, or digit rule.
	ErrWeakPassword = errors.New("weak password")
	// ErrEmailAlreadyExists reports a normalized email that is already registered.
	ErrEmailAlreadyExists = errors.New("email already exists")
	// ErrInvalidDisplayName reports a display name that is present but blank.
	ErrInvalidDisplayName = errors.New("invalid display name")
	// ErrInvalidFilter reports a list filter or pagination value that is not allowed.
	ErrInvalidFilter = errors.New("invalid filter")
)

// ValidationCode identifies the specific rule a validation error violated so the
// delivery layer can name it without guessing.
type ValidationCode string

// Specific validation-rule codes named in a validation outcome (FR-002, FR-003).
const (
	// Email codes.
	CodeEmailMalformed ValidationCode = "email_malformed"
	// Password codes.
	CodePasswordTooShort    ValidationCode = "password_too_short"
	CodePasswordTooLong     ValidationCode = "password_too_long"
	CodePasswordNoUppercase ValidationCode = "password_no_uppercase"
	CodePasswordNoDigit     ValidationCode = "password_no_digit"
	// List filter and pagination codes.
	CodePageInvalid  ValidationCode = "page_invalid"
	CodeLimitInvalid ValidationCode = "limit_invalid"
	CodeRoleInvalid  ValidationCode = "role_invalid"
)

// ValidationError is a typed error that carries both a sentinel (for errors.Is)
// and the precise rule that failed (for a specific message). Its Unwrap exposes
// the sentinel so existing errors.Is checks keep working.
type ValidationError struct {
	Sentinel error
	Code     ValidationCode
	Detail   string
}

// Error implements error.
func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Sentinel.Error(), e.Detail)
}

// Unwrap returns the sentinel so errors.Is(err, ErrInvalidEmail|ErrWeakPassword)
// continues to match.
func (e *ValidationError) Unwrap() error { return e.Sentinel }

// newValidation builds a ValidationError with a rule code and a human detail.
func newValidation(sentinel error, code ValidationCode, detail string) error {
	return &ValidationError{Sentinel: sentinel, Code: code, Detail: detail}
}

// Entity invariant sentinels.
var (
	// ErrInvalidUser reports a User that violates a stored-field invariant.
	ErrInvalidUser = errors.New("invalid user")
	// ErrInvalidProfile reports a Profile that violates an invariant.
	ErrInvalidProfile = errors.New("invalid profile")
	// ErrInvalidEvent reports a UserRegisteredEvent missing a required field.
	ErrInvalidEvent = errors.New("invalid event")
	// ErrUnknownRole reports a role value outside the known set.
	ErrUnknownRole = errors.New("unknown role")
)

// Constitution-mandated sentinels reserved for the wider identity surface
// (authentication, authorization, lookups) delivered by later features.
var (
	// ErrInvalidCredentials reports failed authentication.
	ErrInvalidCredentials = errors.New("invalid credentials")
	// ErrInvalidToken reports a token that is malformed, expired, revoked, or
	// signed with an unexpected method.
	ErrInvalidToken = errors.New("invalid token")
	// ErrForbidden reports an authenticated caller lacking permission.
	ErrForbidden = errors.New("forbidden")
	// ErrNotFound reports a missing resource.
	ErrNotFound = errors.New("not found")
)
