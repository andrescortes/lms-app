package http

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"lms-english/identity-service/internal/usecase"
)

// RegisterRoute is the single registration endpoint path.
const RegisterRoute = "/api/identity/auth/register"

// LoginRoute is the credential verification and token issuance endpoint path.
const LoginRoute = "/api/identity/auth/login"

// RefreshRoute is the refresh token rotation endpoint path.
const RefreshRoute = "/api/identity/auth/refresh"

// NewRouter builds the identity HTTP handler: the registration and login
// routes guarded by the per-visitor limiter, with request IDs, panic recovery,
// structured logging, and CORS.
func NewRouter(uc *usecase.RegisterUserUseCase, loginUc *usecase.LoginUserUseCase, refreshUc *usecase.RefreshTokenUseCase, limiter *VisitorLimiter, logger *slog.Logger) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(slogMiddleware(logger))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{http.MethodPost, http.MethodOptions},
		AllowedHeaders: []string{"Content-Type"},
		MaxAge:         300,
	}))

	registerHandler := &registerHandler{uc: uc}
	r.With(limiter.Middleware).Post(RegisterRoute, registerHandler.ServeHTTP)

	loginHandler := &loginHandler{uc: loginUc}
	r.With(loginRateLimiter()).Post(LoginRoute, loginHandler.ServeHTTP)

	refreshHandler := &refreshHandler{uc: refreshUc}
	r.With(loginRateLimiter()).Post(RefreshRoute, refreshHandler.ServeHTTP)

	return r
}

// slogMiddleware logs each request with method, path, status, and duration, and
// never logs request bodies or credentials.
func slogMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			start := time.Now()
			defer func() {
				logger.Info("http request",
					"method", r.Method,
					"path", r.URL.Path,
					"status", ww.Status(),
					"duration_ms", time.Since(start).Milliseconds(),
					"request_id", middleware.GetReqID(r.Context()),
				)
			}()
			next.ServeHTTP(ww, r)
		})
	}
}
