package transport

import (
	"log/slog"
	"net/http"
)

func NewRouter(healthHandler *HealthHandler, authHandler *AuthHandler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", healthHandler.Check)

	// public routes
	mux.HandleFunc("POST /api/auth/register", authHandler.RegisterUser)

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
