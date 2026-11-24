package transport

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"lms-app/backend/gateway-auth/internal/model"
	"lms-app/backend/gateway-auth/internal/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthMiddleware(t *testing.T) {
	jwtService := service.NewJWTService("test-secret-key", 15*time.Minute)
	user := &service.CustomClaims{
		Subject:   "user-1",
		Email:     "test@example.com",
		Role:      "student",
		CEFRLevel: "B1",
	}

	// Generate a valid token
	token, err := jwtService.GenerateAccessToken(&model.User{
		ID:        user.Subject,
		Email:     user.Email,
		Role:      user.Role,
		CEFRLevel: user.CEFRLevel,
	})
	require.NoError(t, err)

	middleware := AuthMiddleware(jwtService)
	nextCalled := false

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		claims, ok := GetUserClaims(r.Context())
		if !ok {
			http.Error(w, "claims not in context", http.StatusInternalServerError)
			return
		}
		assert.Equal(t, user.Subject, claims.Subject)
		assert.Equal(t, user.Email, claims.Email)
		assert.Equal(t, user.Role, claims.Role)
		assert.Equal(t, user.CEFRLevel, claims.CEFRLevel)
		w.WriteHeader(http.StatusOK)
	})

	t.Run("valid token", func(t *testing.T) {
		nextCalled = false
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		middleware(next).ServeHTTP(w, req)

		assert.True(t, nextCalled)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("missing authorization header", func(t *testing.T) {
		nextCalled = false
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		w := httptest.NewRecorder()

		middleware(next).ServeHTTP(w, req)

		assert.False(t, nextCalled)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "missing authorization header")
	})

	t.Run("empty authorization header", func(t *testing.T) {
		nextCalled = false
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "")
		w := httptest.NewRecorder()

		middleware(next).ServeHTTP(w, req)

		assert.False(t, nextCalled)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid format - not Bearer", func(t *testing.T) {
		nextCalled = false
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Basic "+token)
		w := httptest.NewRecorder()

		middleware(next).ServeHTTP(w, req)

		assert.False(t, nextCalled)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "invalid authorization header format")
	})

	t.Run("invalid format - missing token", func(t *testing.T) {
		nextCalled = false
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer ")
		w := httptest.NewRecorder()

		middleware(next).ServeHTTP(w, req)

		assert.False(t, nextCalled)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "invalid authorization header format")
	})

	t.Run("malformed token", func(t *testing.T) {
		nextCalled = false
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer invalid.token.here")
		w := httptest.NewRecorder()

		middleware(next).ServeHTTP(w, req)

		assert.False(t, nextCalled)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "invalid or expired access token")
	})

	t.Run("expired token", func(t *testing.T) {
		nextCalled = false
		// Create service with negative TTL (already expired)
		expiredJWT := service.NewJWTService("test-secret-key", -1*time.Hour)
		expiredToken, err := expiredJWT.GenerateAccessToken(&model.User{
			ID:        "user-1",
			Email:     "test@example.com",
			Role:      "student",
			CEFRLevel: "A1",
		})
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+expiredToken)
		w := httptest.NewRecorder()

		middleware(next).ServeHTTP(w, req)

		assert.False(t, nextCalled)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("wrong secret", func(t *testing.T) {
		nextCalled = false
		wrongJWT := service.NewJWTService("wrong-secret", 15*time.Minute)
		wrongToken, err := wrongJWT.GenerateAccessToken(&model.User{
			ID:        "user-1",
			Email:     "test@example.com",
			Role:      "student",
			CEFRLevel: "A1",
		})
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+wrongToken)
		w := httptest.NewRecorder()

		middleware(next).ServeHTTP(w, req)

		assert.False(t, nextCalled)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "invalid or expired access token")
	})

	t.Run("case insensitive Bearer", func(t *testing.T) {
		nextCalled = false
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "bearer "+token) // lowercase
		w := httptest.NewRecorder()

		middleware(next).ServeHTTP(w, req)

		assert.True(t, nextCalled)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestGetUserClaims(t *testing.T) {
	claims := &service.CustomClaims{
		Subject:   "user-1",
		Email:     "test@example.com",
		Role:      "student",
		CEFRLevel: "B1",
	}

	t.Run("claims present", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), userClaimsContextKey, claims)
		result, ok := GetUserClaims(ctx)

		assert.True(t, ok)
		assert.Equal(t, claims, result)
	})

	t.Run("claims not present", func(t *testing.T) {
		result, ok := GetUserClaims(context.Background())

		assert.False(t, ok)
		assert.Nil(t, result)
	})

	t.Run("wrong type in context", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), userClaimsContextKey, "not-claims")
		result, ok := GetUserClaims(ctx)

		assert.False(t, ok)
		assert.Nil(t, result)
	})

	t.Run("nil claims in context", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), userClaimsContextKey, (*service.CustomClaims)(nil))
		result, ok := GetUserClaims(ctx)

		assert.False(t, ok)
		assert.Nil(t, result)
	})
}