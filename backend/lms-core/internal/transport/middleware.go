package transport

import (
	"context"
	"net/http"
	"strings"
)

type contextKey string

const trustedUserContextKey contextKey = "trusted_user"

// TrustedUser represents user already validated by gateway-auth.
type TrustedUser struct {
	ID        string
	Role      string
	CEFRLevel string
	Email     string
}

// TrustedGatewayMiddleware extract headers X-User-* injected by gateway-auth
func TrustedGatewayMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := strings.TrimSpace(r.Header.Get("X-User-ID"))
		if userID == "" {
			writeError(w, http.StatusUnauthorized, "missing trusted gateway identity headers")
			return
		}

		role := strings.TrimSpace(r.Header.Get("X-User-Role"))
		if role == "" {
			role = "student"
		}

		level := strings.ToUpper(strings.TrimSpace(r.Header.Get("X-User-Level")))
		if level == "" {
			level = "A1"
		}

		user := &TrustedUser{
			ID:        userID,
			Role:      role,
			CEFRLevel: level,
			Email:     strings.TrimSpace(r.Header.Get("X-User-Email")),
		}

		ctx := context.WithValue(r.Context(), trustedUserContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetTrustedUser(ctx context.Context) (*TrustedUser, bool) {
	user, ok := ctx.Value(trustedUserContextKey).(*TrustedUser)
	return user, ok && user != nil
}
