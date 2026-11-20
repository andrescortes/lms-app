package http

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"lms-english/identity-service/internal/domain"
	"lms-english/identity-service/internal/usecase"
)

// RegisterRoute is the single registration endpoint path.
const RegisterRoute = "/api/identity/auth/register"

// LoginRoute is the credential verification and token issuance endpoint path.
const LoginRoute = "/api/identity/auth/login"

// RefreshRoute is the refresh token rotation endpoint path.
const RefreshRoute = "/api/identity/auth/refresh"

// LogoutRoute is the session-ending endpoint path. It requires a bearer token.
const LogoutRoute = "/api/identity/auth/logout"

// ProfileRoute is the caller's own account and profile. It requires a bearer token.
const ProfileRoute = "/api/identity/users/me"

// UsersRoute lists accounts. It requires an ADMIN bearer token.
const UsersRoute = "/api/identity/users"

// UserRoleRoute changes one account's role. It requires an ADMIN bearer token.
const UserRoleRoute = "/api/identity/users/{id}/role"

// AuthenticatedRouteGroup is the chi group prefix for routes that require a
// valid access token.
const AuthenticatedRouteGroup = "/api/identity/auth"

// NewRouter builds the identity HTTP handler: the public registration, login,
// and refresh routes, plus the bearer-protected sign-out route.
func NewRouter(
	uc *usecase.RegisterUserUseCase,
	loginUc *usecase.LoginUserUseCase,
	refreshUc *usecase.RefreshTokenUseCase,
	logoutUc *usecase.LogoutUseCase,
	getProfileUc *usecase.GetProfileUseCase,
	updateProfileUc *usecase.UpdateProfileUseCase,
	listUsersUc *usecase.ListUsersUseCase,
	changeUserRoleUc *usecase.ChangeUserRoleUseCase,
	verifier usecase.TokenVerifier,
	limiter *VisitorLimiter,
	logger *slog.Logger,
) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(slogMiddleware(logger))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{http.MethodPost, http.MethodGet, http.MethodPatch, http.MethodOptions},
		AllowedHeaders: []string{"Content-Type", "Authorization"},
		MaxAge:         300,
	}))

	registerHandler := &registerHandler{uc: uc}
	r.With(limiter.Middleware).Post(RegisterRoute, registerHandler.ServeHTTP)

	loginHandler := &loginHandler{uc: loginUc}
	r.With(loginRateLimiter()).Post(LoginRoute, loginHandler.ServeHTTP)

	refreshHandler := &refreshHandler{uc: refreshUc}
	r.With(loginRateLimiter()).Post(RefreshRoute, refreshHandler.ServeHTTP)

	logoutHandler := &logoutHandler{uc: logoutUc}
	getProfileHandler := &getProfileHandler{uc: getProfileUc}
	updateProfileHandler := &updateProfileHandler{uc: updateProfileUc}
	r.Group(func(protected chi.Router) {
		protected.Use(Authenticate(verifier))
		protected.Post(LogoutRoute, logoutHandler.ServeHTTP)
		protected.With(withProfileErrorCodes).Get(ProfileRoute, getProfileHandler.ServeHTTP)
		protected.With(withProfileErrorCodes).Patch(ProfileRoute, updateProfileHandler.ServeHTTP)

		userListHandler := &userListHandler{uc: listUsersUc}
		protected.With(RequireRole(domain.RoleAdmin.String())).Get(UsersRoute, userListHandler.ServeHTTP)

		userRoleHandler := &userRoleHandler{uc: changeUserRoleUc}
		protected.With(RequireRole(domain.RoleAdmin.String())).Patch(UserRoleRoute, userRoleHandler.ServeHTTP)
	})

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
