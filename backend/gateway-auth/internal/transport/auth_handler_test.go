package transport

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"lms-app/backend/gateway-auth/internal/model"
	"lms-app/backend/gateway-auth/internal/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// MockAuthRepository is a mock implementation of AuthRepository
type MockAuthRepository struct {
	mock.Mock
}

func (m *MockAuthRepository) CreateUser(ctx context.Context, user *model.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockAuthRepository) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockAuthRepository) GetUserById(ctx context.Context, id string) (*model.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockAuthRepository) SaveRefreshToken(ctx context.Context, rt *model.RefreshToken) error {
	args := m.Called(ctx, rt)
	return args.Error(0)
}

func (m *MockAuthRepository) GetRefreshTokenByHash(ctx context.Context, tokenHash string) (*model.RefreshToken, error) {
	args := m.Called(ctx, tokenHash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.RefreshToken), args.Error(1)
}

func (m *MockAuthRepository) RotateRefreshToken(ctx context.Context, oldTokenID string, newToken *model.RefreshToken) error {
	args := m.Called(ctx, oldTokenID, newToken)
	return args.Error(0)
}

func (m *MockAuthRepository) RevokeAllUserRefreshTokens(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockAuthRepository) RevokeRefreshTokenByHash(ctx context.Context, tokenHash string) error {
	args := m.Called(ctx, tokenHash)
	return args.Error(0)
}

func (m *MockAuthRepository) UpdatePasswordAndRevokeSessions(ctx context.Context, userID string, newPasswordHash string, updatedAt time.Time) error {
	args := m.Called(ctx, userID, newPasswordHash, updatedAt)
	return args.Error(0)
}

func (m *MockAuthRepository) CleanupExpiredRefreshTokens(ctx context.Context, olderThan time.Duration) (int64, error) {
	args := m.Called(ctx, olderThan)
	if args.Get(0) == nil {
		return 0, args.Error(1)
	}
	return args.Get(0).(int64), args.Error(1)
}

func createTestUser(id string) *model.User {
	now := time.Now().UTC()
	// Generate a proper bcrypt hash for "password123"
	hashedBytes, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	return &model.User{
		ID:           id,
		Email:        "test@example.com",
		PasswordHash: string(hashedBytes),
		FullName:     "Test User",
		Role:         "student",
		CEFRLevel:    "A1",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

func createAuthResponse(user *model.User, accessToken string) *model.AuthResponse {
	return &model.AuthResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   900,
		User:        user,
	}
}

func newTestAuthHandler(mockRepo *MockAuthRepository) *AuthHandler {
	jwtService := service.NewJWTService("test-secret-key", 15*time.Minute)
	authService := service.NewAuthService(mockRepo, jwtService, 15*time.Minute, 24*time.Hour)
	return NewAuthHandler(authService, 24*time.Hour, false, "")
}

func TestAuthHandler_RegisterUser(t *testing.T) {
	mockRepo := new(MockAuthRepository)
	handler := newTestAuthHandler(mockRepo)

	t.Run("success", func(t *testing.T) {
		mockRepo.On("CreateUser", mock.Anything, mock.MatchedBy(func(u *model.User) bool {
			return u.Email == "new@example.com" && u.FullName == "New User"
		})).Return(nil).Once()

		body := `{"email":"new@example.com","password":"password123","full_name":"New User","cefr_level":"B1"}`
		req := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.RegisterUser(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		var resp model.User
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "new@example.com", resp.Email)
		assert.Equal(t, "New User", resp.FullName)
		mockRepo.AssertExpectations(t)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		body := `{"email":"invalid json"`
		req := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.RegisterUser(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "invalid JSON payload")
	})

	t.Run("validation error - short password", func(t *testing.T) {
		// Validation happens before calling repository, so CreateUser should NOT be called
		body := `{"email":"valid@example.com","password":"short","full_name":"Valid Name"}`
		req := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.RegisterUser(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "password must have at least 8 characters")
		mockRepo.AssertNotCalled(t, "CreateUser")
	})

	t.Run("validation error - invalid email", func(t *testing.T) {
		body := `{"email":"invalid","password":"password123","full_name":"Valid Name"}`
		req := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.RegisterUser(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "invalid email format")
		mockRepo.AssertNotCalled(t, "CreateUser")
	})

	t.Run("email already exists", func(t *testing.T) {
		mockRepo.On("CreateUser", mock.Anything, mock.Anything).Return(model.ErrEmailAlreadyExists).Once()

		body := `{"email":"existing@example.com","password":"password123","full_name":"New User"}`
		req := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.RegisterUser(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)
		assert.Contains(t, w.Body.String(), "already exist")
		mockRepo.AssertExpectations(t)
	})
}

func TestAuthHandler_Login(t *testing.T) {
	mockRepo := new(MockAuthRepository)
	handler := newTestAuthHandler(mockRepo)

	user := createTestUser("user-1")

	t.Run("success", func(t *testing.T) {
		mockRepo.On("GetUserByEmail", mock.Anything, "test@example.com").Return(user, nil).Once()
		mockRepo.On("SaveRefreshToken", mock.Anything, mock.MatchedBy(func(rt *model.RefreshToken) bool {
			return rt.UserID == "user-1" && !rt.Revoked
		})).Return(nil).Once()

		body := `{"email":"test@example.com","password":"password123"}`
		req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.Login(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp model.AuthResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.NotEmpty(t, resp.AccessToken)
		assert.Equal(t, "Bearer", resp.TokenType)
		assert.Equal(t, user.ID, resp.User.ID)
		assert.Equal(t, user.Email, resp.User.Email)

		// Check refresh cookie
		cookies := w.Result().Cookies()
		assert.Len(t, cookies, 1)
		assert.Equal(t, "lms_refresh_token", cookies[0].Name)
		assert.NotEmpty(t, cookies[0].Value)
		assert.Equal(t, "/api/auth", cookies[0].Path)
		assert.True(t, cookies[0].HttpOnly)
		assert.Equal(t, http.SameSiteLaxMode, cookies[0].SameSite)
		mockRepo.AssertExpectations(t)
	})

	t.Run("invalid credentials", func(t *testing.T) {
		mockRepo.On("GetUserByEmail", mock.Anything, "nonexistent@example.com").Return(nil, model.ErrUserNotFound).Once()

		body := `{"email":"nonexistent@example.com","password":"wrong"}`
		req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.Login(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "invalid email or password")
		mockRepo.AssertExpectations(t)
	})

	t.Run("wrong password", func(t *testing.T) {
		mockRepo.On("GetUserByEmail", mock.Anything, "test@example.com").Return(user, nil).Once()

		body := `{"email":"test@example.com","password":"wrongpassword"}`
		req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.Login(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "invalid email or password")
		mockRepo.AssertExpectations(t)
	})

	t.Run("validation error", func(t *testing.T) {
		// Validation happens before calling repository, so GetUserByEmail should NOT be called
		body := `{"email":"","password":""}`
		req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.Login(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "email and password are required")
		mockRepo.AssertNotCalled(t, "GetUserByEmail")
	})
}

func TestAuthHandler_Refresh(t *testing.T) {
	mockRepo := new(MockAuthRepository)
	handler := newTestAuthHandler(mockRepo)

	user := createTestUser("user-1")

	// Create a valid refresh token hash
	rawRefresh := "valid-refresh-token"
	tokenHash := service.HashRefreshToken(rawRefresh)
	storedToken := &model.RefreshToken{
		ID:        "refresh-1",
		UserID:    "user-1",
		TokenHash: tokenHash,
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
		Revoked:   false,
		CreatedAt: time.Now().UTC(),
	}

	t.Run("success", func(t *testing.T) {
		mockRepo.On("GetRefreshTokenByHash", mock.Anything, tokenHash).Return(storedToken, nil).Once()
		mockRepo.On("GetUserById", mock.Anything, "user-1").Return(user, nil).Once()
		mockRepo.On("RotateRefreshToken", mock.Anything, "refresh-1", mock.MatchedBy(func(rt *model.RefreshToken) bool {
			return rt.UserID == "user-1" && rt.TokenHash != tokenHash
		})).Return(nil).Once()

		req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
		req.AddCookie(&http.Cookie{Name: "lms_refresh_token", Value: rawRefresh})
		w := httptest.NewRecorder()

		handler.Refresh(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp model.AuthResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.NotEmpty(t, resp.AccessToken)
		assert.Equal(t, "Bearer", resp.TokenType)
		assert.Equal(t, user.ID, resp.User.ID)
		assert.Equal(t, user.Email, resp.User.Email)

		// Check new refresh cookie
		cookies := w.Result().Cookies()
		assert.Len(t, cookies, 1)
		assert.NotEmpty(t, cookies[0].Value)
		assert.NotEqual(t, rawRefresh, cookies[0].Value) // Should be rotated
		mockRepo.AssertExpectations(t)
	})

	t.Run("missing cookie", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
		w := httptest.NewRecorder()

		handler.Refresh(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "missing refresh token cookie")
	})

	t.Run("empty cookie value", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
		req.AddCookie(&http.Cookie{Name: "lms_refresh_token", Value: ""})
		w := httptest.NewRecorder()

		handler.Refresh(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid refresh token", func(t *testing.T) {
		mockRepo.On("GetRefreshTokenByHash", mock.Anything, mock.Anything).Return(nil, model.ErrInvalidRefreshToken).Once()

		req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
		req.AddCookie(&http.Cookie{Name: "lms_refresh_token", Value: "invalid-token"})
		w := httptest.NewRecorder()

		handler.Refresh(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		// Cookie should be deleted (max-age=-1)
		cookies := w.Result().Cookies()
		assert.Len(t, cookies, 1)
		assert.Equal(t, -1, cookies[0].MaxAge)
		mockRepo.AssertExpectations(t)
	})
}

func TestAuthHandler_Me(t *testing.T) {
	mockRepo := new(MockAuthRepository)
	handler := newTestAuthHandler(mockRepo)

	user := createTestUser("user-1")

	t.Run("success", func(t *testing.T) {
		mockRepo.On("GetUserById", mock.Anything, "user-1").Return(user, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
		// Add user claims to context (simulating auth middleware)
		ctx := context.WithValue(req.Context(), userClaimsContextKey, &service.CustomClaims{
			Subject: "user-1",
			Email:   "test@example.com",
			Role:    "student",
		})
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.Me(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp model.User
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, user.ID, resp.ID)
		assert.Equal(t, user.Email, resp.Email)
		mockRepo.AssertExpectations(t)
	})

	t.Run("no claims in context", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
		w := httptest.NewRecorder()

		handler.Me(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("user not found", func(t *testing.T) {
		mockRepo.On("GetUserById", mock.Anything, "user-1").Return(nil, model.ErrUserNotFound).Once()

		req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
		ctx := context.WithValue(req.Context(), userClaimsContextKey, &service.CustomClaims{
			Subject: "user-1",
		})
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.Me(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		mockRepo.AssertExpectations(t)
	})
}

func TestAuthHandler_Logout(t *testing.T) {
	mockRepo := new(MockAuthRepository)
	handler := newTestAuthHandler(mockRepo)

	t.Run("success with refresh token", func(t *testing.T) {
		rawRefresh := "refresh-token"
		tokenHash := service.HashRefreshToken(rawRefresh)
		mockRepo.On("RevokeRefreshTokenByHash", mock.Anything, tokenHash).Return(nil).Once()

		req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
		req.AddCookie(&http.Cookie{Name: "lms_refresh_token", Value: rawRefresh})
		ctx := context.WithValue(req.Context(), userClaimsContextKey, &service.CustomClaims{
			Subject: "user-1",
		})
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.Logout(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "logged out successfully", resp["message"])

		// Cookie should be deleted
		cookies := w.Result().Cookies()
		assert.Len(t, cookies, 1)
		assert.Equal(t, -1, cookies[0].MaxAge)
		mockRepo.AssertExpectations(t)
	})

	t.Run("success without refresh token", func(t *testing.T) {
		// When no refresh token, AuthService.Logout returns early without calling repository
		mockRepo.AssertNotCalled(t, "RevokeRefreshTokenByHash")

		req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
		ctx := context.WithValue(req.Context(), userClaimsContextKey, &service.CustomClaims{
			Subject: "user-1",
		})
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.Logout(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("unauthorized - no claims", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
		w := httptest.NewRecorder()

		handler.Logout(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestAuthHandler_ChangePassword(t *testing.T) {
	user := createTestUser("user-1")

	t.Run("success", func(t *testing.T) {
		mockRepo := new(MockAuthRepository)
		handler := newTestAuthHandler(mockRepo)

		mockRepo.On("GetUserById", mock.Anything, "user-1").Return(user, nil).Once()
		mockRepo.On("UpdatePasswordAndRevokeSessions", mock.Anything, "user-1", mock.AnythingOfType("string"), mock.AnythingOfType("time.Time")).Return(nil).Once()

		body := `{"current_password":"password123","new_password":"newpass456"}`
		req := httptest.NewRequest(http.MethodPatch, "/api/auth/password", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		ctx := context.WithValue(req.Context(), userClaimsContextKey, &service.CustomClaims{
			Subject: "user-1",
		})
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.ChangePassword(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "password updated successfully", resp["message"])

		// Cookie should be deleted
		cookies := w.Result().Cookies()
		assert.Len(t, cookies, 1)
		assert.Equal(t, -1, cookies[0].MaxAge)
		mockRepo.AssertExpectations(t)
	})

	t.Run("validation error - short new password", func(t *testing.T) {
		mockRepo := new(MockAuthRepository)
		handler := newTestAuthHandler(mockRepo)

		// Validation fails before GetUserById is called
		mockRepo.AssertNotCalled(t, "GetUserById")
		mockRepo.AssertNotCalled(t, "UpdatePasswordAndRevokeSessions")

		body := `{"current_password":"password123","new_password":"short"}`
		req := httptest.NewRequest(http.MethodPatch, "/api/auth/password", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		ctx := context.WithValue(req.Context(), userClaimsContextKey, &service.CustomClaims{
			Subject: "user-1",
		})
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.ChangePassword(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "new_password must have at least 8 characters")
	})

	t.Run("wrong current password", func(t *testing.T) {
		mockRepo := new(MockAuthRepository)
		handler := newTestAuthHandler(mockRepo)

		mockRepo.On("GetUserById", mock.Anything, "user-1").Return(user, nil).Once()
		// bcrypt.CompareHashAndPassword will fail before calling UpdatePasswordAndRevokeSessions
		mockRepo.AssertNotCalled(t, "UpdatePasswordAndRevokeSessions")

		body := `{"current_password":"wrongpassword","new_password":"newpass456"}`
		req := httptest.NewRequest(http.MethodPatch, "/api/auth/password", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		ctx := context.WithValue(req.Context(), userClaimsContextKey, &service.CustomClaims{
			Subject: "user-1",
		})
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.ChangePassword(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "current password is incorrect")
	})

	t.Run("unauthorized - no claims", func(t *testing.T) {
		mockRepo := new(MockAuthRepository)
		handler := newTestAuthHandler(mockRepo)

		body := `{"current_password":"password123","new_password":"newpass456"}`
		req := httptest.NewRequest(http.MethodPatch, "/api/auth/password", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.ChangePassword(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}