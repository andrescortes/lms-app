package transport

import (
	"log/slog"
	"net/http"
)

func NewRouter(healthHandler *HealthHandler, levelHandler *LevelHandler) http.Handler {
	mux := http.NewServeMux()

	// monitoring intern
	mux.HandleFunc("GET /health", healthHandler.Check)

	// Endpoints protect pedagogy by headers from gateway
	mux.Handle("GET /api/lms/levels", TrustedGatewayMiddleware(http.HandlerFunc(levelHandler.GetLevels)))

	return recoveryMiddleware(mux)
}

func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				slog.Error("panic recovered in HTTP request",
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
