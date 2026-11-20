package http

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"lms-english/identity-service/internal/domain"
	"lms-english/identity-service/internal/usecase"
)

// errMissingIdentity is the server-side mistake of mounting a protected route
// without the Authenticate middleware. It is not a caller error.
var errMissingIdentity = errors.New("route requires the Authenticate middleware")

// Claims is the identity derived from a validated access token. It is the only
// identity the service trusts.
type Claims = usecase.TokenClaims

type claimsContextKey struct{}

// WithClaims stores the caller's identity on a context. Tests use it to stand in
// for a completed Authenticate pass.
func WithClaims(ctx context.Context, claims Claims) context.Context {
	return context.WithValue(ctx, claimsContextKey{}, claims)
}

// ClaimsFromContext returns the caller's identity, reporting false when the
// request did not pass Authenticate. It never invents a zero identity.
func ClaimsFromContext(ctx context.Context) (Claims, bool) {
	claims, ok := ctx.Value(claimsContextKey{}).(Claims)
	if !ok || claims.UserID == "" {
		return Claims{}, false
	}
	return claims, true
}

// Authenticate returns middleware that requires a valid bearer access token,
// stores the caller's identity on the request context, and refuses anything else
// with a 401 before the protected handler runs. The verifier is an injected port,
// so this adapter does not depend on the security adapter.
func Authenticate(verifier usecase.TokenVerifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := bearerToken(r)
			if !ok {
				respondError(w, r, domain.ErrInvalidToken)
				return
			}
			verified, err := verifier.Verify(r.Context(), token)
			if err != nil {
				// A verification failure is an invalid-token refusal; any other
				// error is collapsed into the same refusal so the caller learns
				// nothing about why.
				respondError(w, r, domain.ErrInvalidToken)
				return
			}
			ctx := WithClaims(r.Context(), Claims(verified))
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole returns middleware that allows only the listed roles. It runs after
// Authenticate and refuses an authenticated caller whose role is not listed with
// a 403. An empty list is a programming error: it would allow nobody, so it
// panics when the middleware is built rather than on the first request.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	if len(roles) == 0 {
		panic("http: RequireRole needs at least one role")
	}
	allowed := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		allowed[role] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := ClaimsFromContext(r.Context())
			if !ok {
				respondError(w, r, errMissingIdentity)
				return
			}
			if _, permitted := allowed[claims.Role]; !permitted {
				respondError(w, r, domain.ErrForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// bearerToken extracts the token from a single Authorization header of the form
// "Bearer <token>". Any other shape, a second header, or an empty token is not a
// credential the service issued.
func bearerToken(r *http.Request) (string, bool) {
	values := r.Header.Values("Authorization")
	if len(values) != 1 {
		return "", false
	}
	scheme, token, found := strings.Cut(strings.TrimSpace(values[0]), " ")
	if !found || !strings.EqualFold(scheme, "Bearer") {
		return "", false
	}
	token = strings.TrimSpace(token)
	if token == "" || strings.ContainsAny(token, " \t") {
		return "", false
	}
	return token, true
}
