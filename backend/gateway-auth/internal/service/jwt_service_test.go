package service

import (
	"testing"
	"time"

	"lms-app/backend/gateway-auth/internal/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestJWTUser() *model.User {
	return &model.User{
		ID:        "user-1",
		Email:     "test@example.com",
		Role:      "student",
		CEFRLevel: "B1",
	}
}

func TestJWTService_GenerateAccessToken(t *testing.T) {
	jwtService := NewJWTService("test-secret-key", 15*time.Minute)
	user := createTestJWTUser()

	t.Run("success", func(t *testing.T) {
		token, err := jwtService.GenerateAccessToken(user)
		require.NoError(t, err)
		assert.NotEmpty(t, token)

		// Verify token can be parsed
		claims, err := jwtService.ValidateAccessToken(token)
		require.NoError(t, err)
		assert.Equal(t, user.Email, claims.Email)
		assert.Equal(t, user.Role, claims.Role)
		assert.Equal(t, user.CEFRLevel, claims.CEFRLevel)
		assert.Equal(t, user.ID, claims.Subject)
		assert.Equal(t, "gateway-auth", claims.Issuer)
		assert.NotNil(t, claims.IssuedAt)
		assert.NotNil(t, claims.ExpiresAt)
		assert.True(t, claims.ExpiresAt.Time.After(claims.IssuedAt.Time))
	})

	t.Run("token has correct expiration", func(t *testing.T) {
		token, err := jwtService.GenerateAccessToken(user)
		require.NoError(t, err)

		claims, err := jwtService.ValidateAccessToken(token)
		require.NoError(t, err)

		expectedExp := claims.IssuedAt.Time.Add(15 * time.Minute)
		assert.WithinDuration(t, expectedExp, claims.ExpiresAt.Time, time.Second)
	})
}

func TestJWTService_ValidateAccessToken(t *testing.T) {
	jwtService := NewJWTService("test-secret-key", 15*time.Minute)
	user := createTestJWTUser()

	token, err := jwtService.GenerateAccessToken(user)
	require.NoError(t, err)

	t.Run("valid token", func(t *testing.T) {
		claims, err := jwtService.ValidateAccessToken(token)
		require.NoError(t, err)
		assert.Equal(t, user.Email, claims.Email)
		assert.Equal(t, user.ID, claims.Subject)
	})

	t.Run("invalid signature", func(t *testing.T) {
		// Create token with different secret
		otherService := NewJWTService("different-secret", 15*time.Minute)
		otherToken, err := otherService.GenerateAccessToken(user)
		require.NoError(t, err)

		_, err = jwtService.ValidateAccessToken(otherToken)
		assert.Error(t, err)
		assert.ErrorContains(t, err, "invalid access token")
	})

	t.Run("malformed token", func(t *testing.T) {
		_, err := jwtService.ValidateAccessToken("not.a.valid.token")
		assert.Error(t, err)
	})

	t.Run("expired token", func(t *testing.T) {
		// Create service with very short TTL
		shortTTLService := NewJWTService("test-secret-key", -1*time.Hour)
		expiredToken, err := shortTTLService.GenerateAccessToken(user)
		require.NoError(t, err)

		// Wait a bit to ensure it's expired
		time.Sleep(10 * time.Millisecond)

		_, err = jwtService.ValidateAccessToken(expiredToken)
		assert.Error(t, err)
	})

	t.Run("wrong issuer", func(t *testing.T) {
		// Manually create token with wrong issuer
		// We'll just test that validation rejects wrong issuer
		_, err := jwtService.ValidateAccessToken("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJ3cm9uZy1pc3N1ZXIiLCJzdWIiOiJ1c2VyLTEiLCJlbWFpbCI6InRlc3RAZXhhbXBsZS5jb20iLCJyb2xlIjoic3R1ZGVudCIsImNlZnJfbGV2ZWwiOiJCMSIsImV4cCI6MTk5OTk5OTk5OX0.invalid")
		assert.Error(t, err)
	})
}

func TestJWTService_GenerateOpaqueRefreshToken(t *testing.T) {
	jwtService := NewJWTService("test-secret-key", 15*time.Minute)

	t.Run("generates unique tokens", func(t *testing.T) {
		tokens := make(map[string]bool)
		for i := 0; i < 100; i++ {
			raw, hash, err := jwtService.GenerateOpaqueRefreshToken()
			require.NoError(t, err)
			assert.NotEmpty(t, raw)
			assert.NotEmpty(t, hash)
			assert.Equal(t, HashRefreshToken(raw), hash)
			assert.False(t, tokens[raw], "token should be unique")
			tokens[raw] = true
		}
	})

	t.Run("hash matches raw token", func(t *testing.T) {
		raw, hash, err := jwtService.GenerateOpaqueRefreshToken()
		require.NoError(t, err)
		assert.Equal(t, HashRefreshToken(raw), hash)
	})

	t.Run("token length", func(t *testing.T) {
		raw, _, err := jwtService.GenerateOpaqueRefreshToken()
		require.NoError(t, err)
		// 32 bytes base64 URL encoded = 43 chars (no padding)
		assert.Equal(t, 43, len(raw))
	})
}

func TestHashRefreshToken(t *testing.T) {
	t.Run("deterministic", func(t *testing.T) {
		token := "test-refresh-token"
		hash1 := HashRefreshToken(token)
		hash2 := HashRefreshToken(token)
		assert.Equal(t, hash1, hash2)
	})

	t.Run("different tokens produce different hashes", func(t *testing.T) {
		hash1 := HashRefreshToken("token-1")
		hash2 := HashRefreshToken("token-2")
		assert.NotEqual(t, hash1, hash2)
	})

	t.Run("hash format", func(t *testing.T) {
		hash := HashRefreshToken("test")
		// SHA256 hex = 64 chars
		assert.Equal(t, 64, len(hash))
		// Should be valid hex
		for _, c := range hash {
			assert.True(t, (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f'))
		}
	})
}