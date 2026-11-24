package store

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"lms-app/backend/gateway-auth/internal/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestDB creates an in-memory SQLite database with migrations applied
func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", "file::memory:?cache=shared&_pragma=foreign_keys(ON)")
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Apply migrations
	if err := RunMigrations(ctx, db); err != nil {
		db.Close()
		require.NoError(t, err)
	}

	return db
}

// createTestUser creates a test user in the database and returns it
func createTestUser(t *testing.T, db *sql.DB) *model.User {
	t.Helper()
	user := &model.User{
		ID:           "test-user-1",
		Email:        "test@example.com",
		PasswordHash: "$2a$10$hashedpassword",
		FullName:     "Test User",
		Role:         "student",
		CEFRLevel:    "A1",
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	store := NewUserStore(db)
	err := store.CreateUser(context.Background(), user)
	require.NoError(t, err)
	return user
}

func TestUserStore_CreateUser(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewUserStore(db)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		user := &model.User{
			ID:           "user-1",
			Email:        "new@example.com",
			PasswordHash: "$2a$10$hashedpassword",
			FullName:     "New User",
			Role:         "student",
			CEFRLevel:    "B1",
			CreatedAt:    time.Now().UTC(),
			UpdatedAt:    time.Now().UTC(),
		}

		err := store.CreateUser(ctx, user)
		assert.NoError(t, err)

		// Verify user was created
		retrieved, err := store.GetUserByEmail(ctx, "new@example.com")
		assert.NoError(t, err)
		assert.Equal(t, user.ID, retrieved.ID)
		assert.Equal(t, user.Email, retrieved.Email)
		assert.Equal(t, user.FullName, retrieved.FullName)
		assert.Equal(t, user.Role, retrieved.Role)
		assert.Equal(t, user.CEFRLevel, retrieved.CEFRLevel)
	})

	t.Run("duplicate email", func(t *testing.T) {
		// First create a user
		existingUser := &model.User{
			ID:           "existing-1",
			Email:        "duplicate@example.com",
			PasswordHash: "$2a$10$hashedpassword",
			FullName:     "Existing User",
			Role:         "student",
			CEFRLevel:    "A1",
			CreatedAt:    time.Now().UTC(),
			UpdatedAt:    time.Now().UTC(),
		}
		require.NoError(t, store.CreateUser(ctx, existingUser))

		// Try to create another with same email
		user := &model.User{
			ID:           "user-2",
			Email:        "duplicate@example.com", // Same email
			PasswordHash: "$2a$10$hashedpassword",
			FullName:     "Another User",
			Role:         "student",
			CEFRLevel:    "A1",
			CreatedAt:    time.Now().UTC(),
			UpdatedAt:    time.Now().UTC(),
		}

		err := store.CreateUser(ctx, user)
		assert.Error(t, err)
		assert.ErrorIs(t, err, model.ErrEmailAlreadyExists)
	})
}

func TestUserStore_GetUserByEmail(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createTestUser(t, db)
	store := NewUserStore(db)
	ctx := context.Background()

	t.Run("found", func(t *testing.T) {
		user, err := store.GetUserByEmail(ctx, "test@example.com")
		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, "test@example.com", user.Email)
		assert.Equal(t, "Test User", user.FullName)
	})

	t.Run("not found", func(t *testing.T) {
		user, err := store.GetUserByEmail(ctx, "nonexistent@example.com")
		assert.Error(t, err)
		assert.ErrorIs(t, err, model.ErrUserNotFound)
		assert.Nil(t, user)
	})

	t.Run("case insensitive - exact match", func(t *testing.T) {
		// SQLite's default collation is case-sensitive for non-ASCII
		// But for ASCII, it depends on the collation. We test exact match.
		user, err := store.GetUserByEmail(ctx, "test@example.com")
		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, "test@example.com", user.Email)
	})
}

func TestUserStore_GetUserById(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createTestUser(t, db)
	store := NewUserStore(db)
	ctx := context.Background()

	t.Run("found", func(t *testing.T) {
		user, err := store.GetUserById(ctx, "test-user-1")
		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, "test-user-1", user.ID)
		assert.Equal(t, "test@example.com", user.Email)
	})

	t.Run("not found", func(t *testing.T) {
		user, err := store.GetUserById(ctx, "nonexistent-id")
		assert.Error(t, err)
		assert.ErrorIs(t, err, model.ErrUserNotFound)
		assert.Nil(t, user)
	})
}

func TestUserStore_SaveRefreshToken(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createTestUser(t, db)
	store := NewUserStore(db)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		rt := &model.RefreshToken{
			ID:        "refresh-1",
			UserID:    "test-user-1",
			TokenHash: "hash123",
			ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
			Revoked:   false,
			CreatedAt: time.Now().UTC(),
		}

		err := store.SaveRefreshToken(ctx, rt)
		assert.NoError(t, err)

		// Verify token was saved
		retrieved, err := store.GetRefreshTokenByHash(ctx, "hash123")
		assert.NoError(t, err)
		assert.Equal(t, rt.ID, retrieved.ID)
		assert.Equal(t, rt.UserID, retrieved.UserID)
		assert.Equal(t, rt.TokenHash, retrieved.TokenHash)
		assert.Equal(t, rt.Revoked, retrieved.Revoked)
	})
}

func TestUserStore_GetRefreshTokenByHash(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createTestUser(t, db)
	store := NewUserStore(db)
	ctx := context.Background()

	// First save a token
	rt := &model.RefreshToken{
		ID:        "refresh-1",
		UserID:    "test-user-1",
		TokenHash: "hash123",
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
		Revoked:   false,
		CreatedAt: time.Now().UTC(),
	}
	require.NoError(t, store.SaveRefreshToken(ctx, rt))

	t.Run("found", func(t *testing.T) {
		retrieved, err := store.GetRefreshTokenByHash(ctx, "hash123")
		assert.NoError(t, err)
		assert.NotNil(t, retrieved)
		assert.Equal(t, "refresh-1", retrieved.ID)
		assert.False(t, retrieved.Revoked)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := store.GetRefreshTokenByHash(ctx, "nonexistent-hash")
		assert.Error(t, err)
		assert.ErrorIs(t, err, model.ErrInvalidRefreshToken)
	})
}

func TestUserStore_RotateRefreshToken(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createTestUser(t, db)
	store := NewUserStore(db)
	ctx := context.Background()

	// Save initial token
	oldToken := &model.RefreshToken{
		ID:        "old-refresh",
		UserID:    "test-user-1",
		TokenHash: "old-hash",
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
		Revoked:   false,
		CreatedAt: time.Now().UTC(),
	}
	require.NoError(t, store.SaveRefreshToken(ctx, oldToken))

	t.Run("success", func(t *testing.T) {
		newToken := &model.RefreshToken{
			ID:        "new-refresh",
			UserID:    "test-user-1",
			TokenHash: "new-hash",
			ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
			Revoked:   false,
			CreatedAt: time.Now().UTC(),
		}

		err := store.RotateRefreshToken(ctx, "old-refresh", newToken)
		assert.NoError(t, err)

		// Old token should be revoked
		oldToken, err := store.GetRefreshTokenByHash(ctx, "old-hash")
		assert.NoError(t, err)
		assert.True(t, oldToken.Revoked, "old token should be revoked")

		// New token should exist and not be revoked
		newTokenRetrieved, err := store.GetRefreshTokenByHash(ctx, "new-hash")
		assert.NoError(t, err)
		assert.Equal(t, "new-refresh", newTokenRetrieved.ID)
		assert.False(t, newTokenRetrieved.Revoked)
	})

	t.Run("revoked old token fails", func(t *testing.T) {
		// Try to rotate an already revoked token
		newToken := &model.RefreshToken{
			ID:        "another-refresh",
			UserID:    "test-user-1",
			TokenHash: "another-hash",
			ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
			Revoked:   false,
			CreatedAt: time.Now().UTC(),
		}

		err := store.RotateRefreshToken(ctx, "old-refresh", newToken)
		assert.Error(t, err)
		assert.ErrorIs(t, err, model.ErrInvalidRefreshToken)
	})
}

func TestUserStore_RevokeAllUserRefreshTokens(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createTestUser(t, db)
	store := NewUserStore(db)
	ctx := context.Background()

	// Save multiple tokens
	tokens := []*model.RefreshToken{
		{ID: "rt-1", UserID: "test-user-1", TokenHash: "hash1", ExpiresAt: time.Now().UTC().Add(24 * time.Hour), Revoked: false, CreatedAt: time.Now().UTC()},
		{ID: "rt-2", UserID: "test-user-1", TokenHash: "hash2", ExpiresAt: time.Now().UTC().Add(24 * time.Hour), Revoked: false, CreatedAt: time.Now().UTC()},
		{ID: "rt-3", UserID: "test-user-1", TokenHash: "hash3", ExpiresAt: time.Now().UTC().Add(24 * time.Hour), Revoked: false, CreatedAt: time.Now().UTC()},
	}
	for _, rt := range tokens {
		require.NoError(t, store.SaveRefreshToken(ctx, rt))
	}

	t.Run("revoke all", func(t *testing.T) {
		err := store.RevokeAllUserRefreshTokens(ctx, "test-user-1")
		assert.NoError(t, err)

		// All tokens should be revoked
		for _, rt := range tokens {
			retrieved, err := store.GetRefreshTokenByHash(ctx, rt.TokenHash)
			assert.NoError(t, err)
			assert.True(t, retrieved.Revoked)
		}
	})
}

func TestUserStore_RevokeRefreshTokenByHash(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createTestUser(t, db)
	store := NewUserStore(db)
	ctx := context.Background()

	rt := &model.RefreshToken{
		ID:        "rt-1",
		UserID:    "test-user-1",
		TokenHash: "hash-to-revoke",
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
		Revoked:   false,
		CreatedAt: time.Now().UTC(),
	}
	require.NoError(t, store.SaveRefreshToken(ctx, rt))

	t.Run("success", func(t *testing.T) {
		err := store.RevokeRefreshTokenByHash(ctx, "hash-to-revoke")
		assert.NoError(t, err)

		retrieved, err := store.GetRefreshTokenByHash(ctx, "hash-to-revoke")
		assert.NoError(t, err)
		assert.True(t, retrieved.Revoked)
	})

	t.Run("idempotent", func(t *testing.T) {
		// Revoking again should not error
		err := store.RevokeRefreshTokenByHash(ctx, "hash-to-revoke")
		assert.NoError(t, err)
	})
}

func TestUserStore_UpdatePasswordAndRevokeSessions(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createTestUser(t, db)
	store := NewUserStore(db)
	ctx := context.Background()

	// Save some refresh tokens
	tokens := []*model.RefreshToken{
		{ID: "rt-1", UserID: "test-user-1", TokenHash: "hash1", ExpiresAt: time.Now().UTC().Add(24 * time.Hour), Revoked: false, CreatedAt: time.Now().UTC()},
		{ID: "rt-2", UserID: "test-user-1", TokenHash: "hash2", ExpiresAt: time.Now().UTC().Add(24 * time.Hour), Revoked: false, CreatedAt: time.Now().UTC()},
	}
	for _, rt := range tokens {
		require.NoError(t, store.SaveRefreshToken(ctx, rt))
	}

	t.Run("success", func(t *testing.T) {
		newHash := "$2a$10$newhashedpassword"
		err := store.UpdatePasswordAndRevokeSessions(ctx, "test-user-1", newHash, time.Now().UTC())
		assert.NoError(t, err)

		// Verify password was updated
		user, err := store.GetUserById(ctx, "test-user-1")
		assert.NoError(t, err)
		assert.Equal(t, newHash, user.PasswordHash)

		// Verify all tokens were revoked
		for _, rt := range tokens {
			retrieved, err := store.GetRefreshTokenByHash(ctx, rt.TokenHash)
			assert.NoError(t, err)
			assert.True(t, retrieved.Revoked)
		}
	})

	t.Run("user not found", func(t *testing.T) {
		err := store.UpdatePasswordAndRevokeSessions(ctx, "nonexistent", "hash", time.Now().UTC())
		assert.Error(t, err)
		assert.ErrorIs(t, err, model.ErrUserNotFound)
	})
}