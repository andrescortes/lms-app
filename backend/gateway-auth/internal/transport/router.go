package transport

import (
	"lms-app/backend/gateway-auth/internal/service"
	"log/slog"
	"net/http"
)

func NewRouter(
	healthHandler *HealthHandler,
	authHandler *AuthHandler,
	jwtService *service.JWTService,
) http.Handler {
	mux := http.NewServeMux()
	requireAuth := AuthMiddleware(jwtService)

	mux.HandleFunc("GET /health", healthHandler.Check)

	// public routes auth
	mux.HandleFunc("POST /api/auth/register", authHandler.RegisterUser)
	mux.HandleFunc("POST /api/auth/login", authHandler.Login)
	mux.HandleFunc("POST /api/auth/refresh", authHandler.Refresh)

	// private routes with JWT
	mux.Handle("GET /api/auth/me", requireAuth(http.HandlerFunc(authHandler.Me)))
	mux.Handle("POST /api/auth/logout", requireAuth(http.HandlerFunc(authHandler.Logout)))
	mux.Handle("PATCH /api/auth/password", requireAuth(http.HandlerFunc(authHandler.ChangePassword)))

	return recoveryMiddleware(mux)
}

func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				slog.Error("panic retrieving in HTTP request",
					"error", err,
					"method", r.Method,
					"path", r.URL.Path,
				)
				writeError(w, http.StatusInternalServerError, "internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}
