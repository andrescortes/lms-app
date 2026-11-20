package http

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"lms-english/identity-service/internal/domain"
)

// errorResponse is the single error envelope used for every non-2xx response.
type errorResponse struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
	Path      string `json:"path"`
}

// Error codes returned in the envelope.
const (
	codeValidation      = "validation_error"
	codeEmailExists     = "email_already_exists"
	codeInvalidRequest  = "invalid_request"
	codeRateLimited     = "rate_limited"
	codeInternal        = "internal_error"
	codeForbidden       = "forbidden"
	codeNotFound        = "not_found"
	codeUnauthenticated = "invalid_credentials"
	codeInvalidToken    = "INVALID_TOKEN"
	codeWeakPassword    = "WEAK_PASSWORD"
)

// weakPasswordOnProfileRequest marks a request context whose contract classifies a
// weak password as WEAK_PASSWORD rather than validation_error. Registration keeps
// its original code, and both contracts stay satisfied.
type weakPasswordOnProfileRequest struct{}

// withProfileErrorCodes marks r as a profile request, which uses its own error
// codes for outcomes registration already assigns differently.
func withProfileErrorCodes(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), weakPasswordOnProfileRequest{}, true)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// respondError maps a domain or transport error to the single error envelope.
// Validation and duplicate-conflict failures name the specific violated rule;
// other outcomes use their own standard code and a fixed message, and never leak
// internal details.
func respondError(w http.ResponseWriter, r *http.Request, err error) {
	status, code, message := classifyFor(r, err)
	if status >= http.StatusInternalServerError {
		slog.Error("request failed", "method", r.Method, "path", r.URL.Path, "error", err)
	}
	writeJSON(w, status, errorResponse{
		Code:      code,
		Message:   message,
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Path:      r.URL.Path,
	})
}

// classifyFor applies the request-specific code for a weak password before falling
// back to the shared classification.
func classifyFor(r *http.Request, err error) (int, string, string) {
	if errors.Is(err, domain.ErrWeakPassword) && r.Context().Value(weakPasswordOnProfileRequest{}) == true {
		return http.StatusUnprocessableEntity, codeWeakPassword, validationDetail(err, "password does not meet the policy")
	}
	return classify(err)
}

// classify maps an error to an HTTP status, stable code, and safe message. For
// validation failures it names the specific violated rule carried by the typed
// domain error; other outcomes use their own standard code and fixed message.
func classify(err error) (int, string, string) {
	switch {
	case errors.Is(err, domain.ErrInvalidEmail):
		return http.StatusUnprocessableEntity, codeValidation, validationDetail(err, "email must be a valid address")
	case errors.Is(err, domain.ErrWeakPassword):
		return http.StatusUnprocessableEntity, codeValidation, validationDetail(err, "password does not meet the policy")
	case errors.Is(err, domain.ErrInvalidDisplayName):
		return http.StatusUnprocessableEntity, codeValidation, "display name must not be blank"
	case errors.Is(err, domain.ErrInvalidFilter):
		return http.StatusUnprocessableEntity, codeValidation, validationDetail(err, "a list filter is not valid")
	case errors.Is(err, domain.ErrEmailAlreadyExists):
		return http.StatusConflict, codeEmailExists, "an account with this email already exists"
	case errors.Is(err, domain.ErrInvalidCredentials):
		return http.StatusUnauthorized, codeUnauthenticated, "invalid credentials"
	case errors.Is(err, domain.ErrInvalidToken):
		return http.StatusUnauthorized, codeInvalidToken, "invalid token"
	case errors.Is(err, domain.ErrForbidden):
		return http.StatusForbidden, codeForbidden, "forbidden"
	case errors.Is(err, domain.ErrNotFound):
		return http.StatusNotFound, codeNotFound, "not found"
	default:
		return http.StatusInternalServerError, codeInternal, "an unexpected error occurred"
	}
}

// validationDetail extracts the specific rule detail from a typed domain
// ValidationError, falling back to a generic message when no detail is present.
func validationDetail(err error, fallback string) string {
	var ve *domain.ValidationError
	if errors.As(err, &ve) && ve.Detail != "" {
		return ve.Detail
	}
	return fallback
}

// writeJSON encodes v with the status. It disables HTML escaping so payloads are
// not perturbed for non-HTML clients.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
}

// respondBadRequest writes the malformed-request envelope.
func respondBadRequest(w http.ResponseWriter, r *http.Request, message string) {
	writeJSON(w, http.StatusBadRequest, errorResponse{
		Code:      codeInvalidRequest,
		Message:   message,
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Path:      r.URL.Path,
	})
}
