package transport

import (
	"context"
	"lms-app/backend/gateway-auth/internal/service"
	"net/http"
	"strings"
)

type contextKey string

const userClaimsContextKey contextKey = "user_claims"

func AuthMiddleware(jwtService *service.JWTService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				writeError(w, http.StatusUnauthorized, "missing authorization header")
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
				writeError(w, http.StatusUnauthorized, "invalid authorization header format, expected Bearer <token>")
				return
			}

			claims, err := jwtService.ValidateAccessToken(strings.TrimSpace(parts[1]))
			if err != nil {
				writeError(w, http.StatusUnauthorized, "invalid or expired access token")
				return
			}
			ctx := context.WithValue(r.Context(), userClaimsContextKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetUserClaims(ctx context.Context) (*service.CustomClaims, bool) {
	claims, ok := ctx.Value(userClaimsContextKey).(*service.CustomClaims)
	return claims, ok && claims != nil
}
