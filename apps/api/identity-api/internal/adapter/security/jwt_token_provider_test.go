package security

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"lms-english/identity-service/internal/domain"
)

const testSecret = "test-secret-value"

// TestJwtTokenProviderIssue verifies the issued access token parses with HS256
// and carries the user id as subject, the role claim, and a 15-minute lifetime.
func TestJwtTokenProviderIssue(t *testing.T) {
	provider := NewJwtTokenProvider(testSecret, 15*time.Minute)
	user := domain.User{ID: "user-1", Role: domain.RoleStudent}

	token, err := provider.Issue(context.Background(), user)
	if err != nil {
		t.Fatalf("Issue() error = %v, want nil", err)
	}

	parsed, err := jwt.Parse(token, func(tok *jwt.Token) (any, error) {
		if tok.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			t.Fatalf("signing method = %q, want HS256", tok.Method.Alg())
		}
		return []byte(testSecret), nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil {
		t.Fatalf("Parse() error = %v, want nil", err)
	}
	if !parsed.Valid {
		t.Fatal("token is not valid")
	}

	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		t.Fatalf("claims type = %T, want jwt.MapClaims", parsed.Claims)
	}
	if sub, _ := claims["sub"].(string); sub != user.ID {
		t.Errorf("sub = %v, want %q", claims["sub"], user.ID)
	}
	if role, _ := claims["role"].(string); role != domain.RoleStudent.String() {
		t.Errorf("role = %v, want %q", claims["role"], domain.RoleStudent.String())
	}

	exp, err := claims.GetExpirationTime()
	if err != nil {
		t.Fatalf("read exp: %v", err)
	}
	iat, err := claims.GetIssuedAt()
	if err != nil {
		t.Fatalf("read iat: %v", err)
	}
	if got := exp.Sub(iat.Time); got != 15*time.Minute {
		t.Errorf("token lifetime = %v, want 15m", got)
	}
}

// TestJwtTokenProviderVerifyAcceptsItsOwnToken verifies a token the provider
// issued verifies and yields the user id and role.
func TestJwtTokenProviderVerifyAcceptsItsOwnToken(t *testing.T) {
	provider := NewJwtTokenProvider(testSecret, time.Minute)
	token, err := provider.Issue(context.Background(), domain.User{ID: "user-9", Role: domain.RoleTeacher})
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	claims, err := provider.Verify(context.Background(), token)
	if err != nil {
		t.Fatalf("Verify() error = %v, want nil", err)
	}
	if claims.UserID != "user-9" {
		t.Errorf("UserID = %q, want user-9", claims.UserID)
	}
	if claims.Role != domain.RoleTeacher.String() {
		t.Errorf("Role = %q, want %q", claims.Role, domain.RoleTeacher.String())
	}
}

// TestJwtTokenProviderVerifyRefusesBadTokens verifies expired, foreign-signed,
// malformed, and algorithm-confused tokens all report ErrInvalidToken.
func TestJwtTokenProviderVerifyRefusesBadTokens(t *testing.T) {
	provider := NewJwtTokenProvider(testSecret, time.Minute)

	expired := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "user-9", "role": "STUDENT",
		"iat": time.Now().Add(-2 * time.Hour).Unix(),
		"exp": time.Now().Add(-time.Hour).Unix(),
	})
	expiredToken, err := expired.SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("sign expired token: %v", err)
	}

	foreign, err := NewJwtTokenProvider("another-secret", time.Minute).
		Issue(context.Background(), domain.User{ID: "user-9", Role: domain.RoleStudent})
	if err != nil {
		t.Fatalf("sign foreign token: %v", err)
	}

	// alg=none is the classic algorithm-confusion attempt; WithValidMethods must
	// refuse it before any signature check.
	none := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{
		"sub": "user-9", "role": "STUDENT", "exp": time.Now().Add(time.Hour).Unix(),
	})
	noneToken, err := none.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("sign alg=none token: %v", err)
	}

	for name, token := range map[string]string{
		"expired":   expiredToken,
		"foreign":   foreign,
		"malformed": "not-a-token",
		"alg-none":  noneToken,
		"empty":     "",
	} {
		if _, err := provider.Verify(context.Background(), token); !errors.Is(err, domain.ErrInvalidToken) {
			t.Errorf("%s: Verify() error = %v, want ErrInvalidToken", name, err)
		}
	}
}
func TestJwtTokenProviderRejectsWrongSecret(t *testing.T) {
	token, err := NewJwtTokenProvider(testSecret, time.Minute).Issue(context.Background(), domain.User{ID: "u", Role: domain.RoleStudent})
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	if _, err := jwt.Parse(token, func(*jwt.Token) (any, error) {
		return []byte("different-secret"), nil
	}, jwt.WithValidMethods([]string{"HS256"})); err == nil {
		t.Fatal("Parse() with the wrong secret succeeded, want an error")
	}
}
