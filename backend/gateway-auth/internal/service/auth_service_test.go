package service

import (
	"context"
	"testing"
	"time"

	"lms-app/backend/gateway-auth/internal/model"

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

// Test user fixture
func createTestUser(id string) *model.User {
	now := time.Now().UTC()
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

func TestAuthService_RegisterUser(t *testing.T) {
	mockRepo := new(MockAuthRepository)
	jwtService := NewJWTService("test-secret-key", 15*time.Minute)
	svc := NewAuthService(mockRepo, jwtService, 15*time.Minute, 24*time.Hour)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mockRepo.On("CreateUser", mock.Anything, mock.MatchedBy(func(u *model.User) bool {
			return u.Email == "new@example.com" && u.FullName == "New User"
		})).Return(nil).Once()

		req := model.RegisterRequest{
			Email:     "new@example.com",
			Password:  "password123",
			FullName:  "New User",
			CEFRLevel: "B1",
		}

		user, err := svc.RegisterUser(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, "new@example.com", user.Email)
		assert.Equal(t, "New User", user.FullName)
		assert.Equal(t, "student", user.Role)
		assert.Equal(t, "B1", user.CEFRLevel)
		assert.NotEmpty(t, user.ID)
		assert.NotEmpty(t, user.PasswordHash)
		mockRepo.AssertExpectations(t)
	})

	t.Run("empty email", func(t *testing.T) {
		req := model.RegisterRequest{
			Email:     "",
			Password:  "password123",
			FullName:  "New User",
			CEFRLevel: "A1",
		}

		_, err := svc.RegisterUser(ctx, req)
		assert.Error(t, err)
		assert.ErrorContains(t, err, "email is required")
	})

	t.Run("invalid email format", func(t *testing.T) {
		req := model.RegisterRequest{
			Email:     "invalid-email",
			Password:  "password123",
			FullName:  "New User",
			CEFRLevel: "A1",
		}

		_, err := svc.RegisterUser(ctx, req)
		assert.Error(t, err)
		assert.ErrorContains(t, err, "invalid email format")
	})

	t.Run("short full name", func(t *testing.T) {
		req := model.RegisterRequest{
			Email:     "new@example.com",
			Password:  "password123",
			FullName:  "A",
			CEFRLevel: "A1",
		}

		_, err := svc.RegisterUser(ctx, req)
		assert.Error(t, err)
		assert.ErrorContains(t, err, "full_name must have at least 2 characters")
	})

	t.Run("short password", func(t *testing.T) {
		req := model.RegisterRequest{
			Email:     "new@example.com",
			Password:  "short",
			FullName:  "New User",
			CEFRLevel: "A1",
		}

		_, err := svc.RegisterUser(ctx, req)
		assert.Error(t, err)
		assert.ErrorContains(t, err, "password must have at least 8 characters")
	})

	t.Run("long password", func(t *testing.T) {
		req := model.RegisterRequest{
			Email:     "new@example.com",
			Password:  string(make([]byte, 73)), // 73 chars
			FullName:  "New User",
			CEFRLevel: "A1",
		}

		_, err := svc.RegisterUser(ctx, req)
		assert.Error(t, err)
		assert.ErrorContains(t, err, "password cannot exceed 72 characters")
	})

	t.Run("invalid CEFR level", func(t *testing.T) {
		req := model.RegisterRequest{
			Email:     "new@example.com",
			Password:  "password123",
			FullName:  "New User",
			CEFRLevel: "Z9",
		}

		_, err := svc.RegisterUser(ctx, req)
		assert.Error(t, err)
		assert.ErrorContains(t, err, "cefr_level must be one of")
	})

	t.Run("default CEFR level", func(t *testing.T) {
		mockRepo.On("CreateUser", mock.Anything, mock.MatchedBy(func(u *model.User) bool {
			return u.CEFRLevel == "A1"
		})).Return(nil).Once()

		req := model.RegisterRequest{
			Email:     "new@example.com",
			Password:  "password123",
			FullName:  "New User",
			CEFRLevel: "", // Should default to A1
		}

		user, err := svc.RegisterUser(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, "A1", user.CEFRLevel)
		mockRepo.AssertExpectations(t)
	})

	t.Run("email already exists", func(t *testing.T) {
		mockRepo.On("CreateUser", mock.Anything, mock.Anything).Return(model.ErrEmailAlreadyExists).Once()

		req := model.RegisterRequest{
			Email:     "existing@example.com",
			Password:  "password123",
			FullName:  "New User",
			CEFRLevel: "A1",
		}

		_, err := svc.RegisterUser(ctx, req)
		assert.Error(t, err)
		assert.ErrorIs(t, err, model.ErrEmailAlreadyExists)
		mockRepo.AssertExpectations(t)
	})
}

func TestAuthService_Login(t *testing.T) {
	mockRepo := new(MockAuthRepository)
	jwtService := NewJWTService("test-secret-key", 15*time.Minute)
	svc := NewAuthService(mockRepo, jwtService, 15*time.Minute, 24*time.Hour)
	ctx := context.Background()

	user := createTestUser("user-1")

	t.Run("success", func(t *testing.T) {
		mockRepo.On("GetUserByEmail", ctx, "test@example.com").Return(user, nil).Once()
		mockRepo.On("SaveRefreshToken", mock.Anything, mock.MatchedBy(func(rt *model.RefreshToken) bool {
			return rt.UserID == "user-1" && !rt.Revoked
		})).Return(nil).Once()

		req := model.LoginRequest{
			Email:    "test@example.com",
			Password: "password123",
		}

		authResp, rawRefresh, err := svc.Login(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, authResp)
		assert.NotEmpty(t, authResp.AccessToken)
		assert.Equal(t, "Bearer", authResp.TokenType)
		assert.Equal(t, int(15*time.Minute.Seconds()), authResp.ExpiresIn)
		assert.Equal(t, user.ID, authResp.User.ID)
		assert.NotEmpty(t, rawRefresh)
		mockRepo.AssertExpectations(t)
	})

	t.Run("invalid email", func(t *testing.T) {
		req := model.LoginRequest{
			Email:    "nonexistent@example.com",
			Password: "password123",
		}

		mockRepo.On("GetUserByEmail", ctx, "nonexistent@example.com").Return(nil, model.ErrUserNotFound).Once()

		_, _, err := svc.Login(ctx, req)
		assert.Error(t, err)
		assert.ErrorIs(t, err, model.ErrInvalidCredentials)
		mockRepo.AssertExpectations(t)
	})

	t.Run("invalid password", func(t *testing.T) {
		req := model.LoginRequest{
			Email:    "test@example.com",
			Password: "wrongpassword",
		}

		mockRepo.On("GetUserByEmail", ctx, "test@example.com").Return(user, nil).Once()

		_, _, err := svc.Login(ctx, req)
		assert.Error(t, err)
		assert.ErrorIs(t, err, model.ErrInvalidCredentials)
		mockRepo.AssertExpectations(t)
	})

	t.Run("empty credentials", func(t *testing.T) {
		req := model.LoginRequest{
			Email:    "",
			Password: "",
		}

		_, _, err := svc.Login(ctx, req)
		assert.Error(t, err)
		assert.ErrorContains(t, err, "email and password are required")
	})
}

func TestAuthService_Refresh(t *testing.T) {
	mockRepo := new(MockAuthRepository)
	jwtService := NewJWTService("test-secret-key", 15*time.Minute)
	svc := NewAuthService(mockRepo, jwtService, 15*time.Minute, 24*time.Hour)
	ctx := context.Background()

	user := createTestUser("user-1")
	now := time.Now().UTC()

	// Create a valid refresh token hash
	rawRefresh := "valid-refresh-token"
	tokenHash := HashRefreshToken(rawRefresh)
	storedToken := &model.RefreshToken{
		ID:        "refresh-1",
		UserID:    "user-1",
		TokenHash: tokenHash,
		ExpiresAt: now.Add(24 * time.Hour),
		Revoked:   false,
		CreatedAt: now,
	}

	t.Run("success", func(t *testing.T) {
		mockRepo.On("GetRefreshTokenByHash", ctx, tokenHash).Return(storedToken, nil).Once()
		mockRepo.On("GetUserById", ctx, "user-1").Return(user, nil).Once()
		mockRepo.On("RotateRefreshToken", ctx, "refresh-1", mock.MatchedBy(func(rt *model.RefreshToken) bool {
			return rt.UserID == "user-1" && rt.TokenHash != tokenHash
		})).Return(nil).Once()

		authResp, newRawRefresh, err := svc.Refresh(ctx, rawRefresh)
		require.NoError(t, err)
		assert.NotNil(t, authResp)
		assert.NotEmpty(t, authResp.AccessToken)
		assert.NotEmpty(t, newRawRefresh)
		assert.NotEqual(t, rawRefresh, newRawRefresh) // Should be rotated
		mockRepo.AssertExpectations(t)
	})

	t.Run("empty token", func(t *testing.T) {
		_, _, err := svc.Refresh(ctx, "")
		assert.Error(t, err)
		assert.ErrorIs(t, err, model.ErrInvalidRefreshToken)
	})

	t.Run("invalid token hash", func(t *testing.T) {
		mockRepo.On("GetRefreshTokenByHash", ctx, mock.Anything).Return(nil, model.ErrInvalidRefreshToken).Once()

		_, _, err := svc.Refresh(ctx, "invalid-token")
		assert.Error(t, err)
		assert.ErrorIs(t, err, model.ErrInvalidRefreshToken)
		mockRepo.AssertExpectations(t)
	})

	t.Run("revoked token", func(t *testing.T) {
		revokedToken := *storedToken
		revokedToken.Revoked = true
		mockRepo.On("GetRefreshTokenByHash", ctx, tokenHash).Return(&revokedToken, nil).Once()
		mockRepo.On("RevokeAllUserRefreshTokens", ctx, "user-1").Return(nil).Once()

		_, _, err := svc.Refresh(ctx, rawRefresh)
		assert.Error(t, err)
		assert.ErrorIs(t, err, model.ErrInvalidRefreshToken)
		mockRepo.AssertExpectations(t)
	})

	t.Run("expired token", func(t *testing.T) {
		expiredToken := *storedToken
		expiredToken.ExpiresAt = now.Add(-1 * time.Hour)
		mockRepo.On("GetRefreshTokenByHash", ctx, tokenHash).Return(&expiredToken, nil).Once()

		_, _, err := svc.Refresh(ctx, rawRefresh)
		assert.Error(t, err)
		assert.ErrorIs(t, err, model.ErrInvalidRefreshToken)
		mockRepo.AssertExpectations(t)
	})
}

func TestAuthService_GetProfile(t *testing.T) {
	mockRepo := new(MockAuthRepository)
	jwtService := NewJWTService("test-secret-key", 15*time.Minute)
	svc := NewAuthService(mockRepo, jwtService, 15*time.Minute, 24*time.Hour)
	ctx := context.Background()

	user := createTestUser("user-1")

	t.Run("success", func(t *testing.T) {
		mockRepo.On("GetUserById", ctx, "user-1").Return(user, nil).Once()

		result, err := svc.GetProfile(ctx, "user-1")
		require.NoError(t, err)
		assert.Equal(t, user, result)
		mockRepo.AssertExpectations(t)
	})

	t.Run("empty user ID", func(t *testing.T) {
		_, err := svc.GetProfile(ctx, "")
		assert.Error(t, err)
		assert.ErrorIs(t, err, model.ErrUserNotFound)
	})

	t.Run("user not found", func(t *testing.T) {
		mockRepo.On("GetUserById", ctx, "nonexistent").Return(nil, model.ErrUserNotFound).Once()

		_, err := svc.GetProfile(ctx, "nonexistent")
		assert.Error(t, err)
		assert.ErrorIs(t, err, model.ErrUserNotFound)
		mockRepo.AssertExpectations(t)
	})
}

func TestAuthService_Logout(t *testing.T) {
	mockRepo := new(MockAuthRepository)
	jwtService := NewJWTService("test-secret-key", 15*time.Minute)
	svc := NewAuthService(mockRepo, jwtService, 15*time.Minute, 24*time.Hour)
	ctx := context.Background()

	t.Run("success with token", func(t *testing.T) {
		rawRefresh := "refresh-to-revoke"
		tokenHash := HashRefreshToken(rawRefresh)
		mockRepo.On("RevokeRefreshTokenByHash", ctx, tokenHash).Return(nil).Once()

		err := svc.Logout(ctx, rawRefresh)
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("empty token (no-op)", func(t *testing.T) {
		err := svc.Logout(ctx, "")
		assert.NoError(t, err)
		mockRepo.AssertNotCalled(t, "RevokeRefreshTokenByHash")
	})
}

func TestAuthService_ChangePassword(t *testing.T) {
	mockRepo := new(MockAuthRepository)
	jwtService := NewJWTService("test-secret-key", 15*time.Minute)
	svc := NewAuthService(mockRepo, jwtService, 15*time.Minute, 24*time.Hour)
	ctx := context.Background()

	user := createTestUser("user-1")

	t.Run("success", func(t *testing.T) {
		mockRepo.On("GetUserById", ctx, "user-1").Return(user, nil).Once()
		mockRepo.On("UpdatePasswordAndRevokeSessions", ctx, "user-1", mock.AnythingOfType("string"), mock.AnythingOfType("time.Time")).Return(nil).Once()

		req := model.ChangePasswordRequest{
			CurrentPassword: "password123",
			NewPassword:     "newpassword456",
		}

		err := svc.ChangePassword(ctx, "user-1", req)
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("empty passwords", func(t *testing.T) {
		req := model.ChangePasswordRequest{
			CurrentPassword: "",
			NewPassword:     "",
		}

		err := svc.ChangePassword(ctx, "user-1", req)
		assert.Error(t, err)
		assert.ErrorContains(t, err, "current_password and new_password are required")
	})

	t.Run("short new password", func(t *testing.T) {
		req := model.ChangePasswordRequest{
			CurrentPassword: "password123",
			NewPassword:     "short",
		}

		err := svc.ChangePassword(ctx, "user-1", req)
		assert.Error(t, err)
		assert.ErrorContains(t, err, "new_password must have at least 8 characters")
	})

	t.Run("same password", func(t *testing.T) {
		req := model.ChangePasswordRequest{
			CurrentPassword: "password123",
			NewPassword:     "password123",
		}

		err := svc.ChangePassword(ctx, "user-1", req)
		assert.Error(t, err)
		assert.ErrorContains(t, err, "new_password must be different from current_password")
	})

	t.Run("wrong current password", func(t *testing.T) {
		mockRepo.On("GetUserById", ctx, "user-1").Return(user, nil).Once()

		req := model.ChangePasswordRequest{
			CurrentPassword: "wrongpassword",
			NewPassword:     "newpassword456",
		}

		err := svc.ChangePassword(ctx, "user-1", req)
		assert.Error(t, err)
		assert.ErrorIs(t, err, model.ErrInvalidCredentials)
		mockRepo.AssertExpectations(t)
	})

	t.Run("user not found", func(t *testing.T) {
		mockRepo.On("GetUserById", ctx, "nonexistent").Return(nil, model.ErrUserNotFound).Once()

		req := model.ChangePasswordRequest{
			CurrentPassword: "password123",
			NewPassword:     "newpassword456",
		}

		err := svc.ChangePassword(ctx, "nonexistent", req)
		assert.Error(t, err)
		assert.ErrorIs(t, err, model.ErrUserNotFound)
		mockRepo.AssertExpectations(t)
	})
}