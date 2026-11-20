package security

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"lms-english/identity-service/internal/domain"
	"lms-english/identity-service/internal/usecase"
)

// JwtTokenProvider issues access tokens for authenticated users. It satisfies
// the use case port; signing uses HS256 with the configured secret and a
// caller-supplied TTL.
var _ usecase.TokenProvider = (*JwtTokenProvider)(nil)

// JwtTokenProvider is the TokenProvider implementation backed by
// github.com/golang-jwt/jwt/v5.
type JwtTokenProvider struct {
	secret string
	ttl    time.Duration
}

// NewJwtTokenProvider returns a token provider signing with HMAC-SHA256 using
// the given secret and access-token TTL.
func NewJwtTokenProvider(secret string, ttl time.Duration) *JwtTokenProvider {
	return &JwtTokenProvider{secret: secret, ttl: ttl}
}

// Issue returns a signed access token for the user with the user id as the
// subject and the user role as the role claim, valid for the provider's TTL.
func (p *JwtTokenProvider) Issue(ctx context.Context, user domain.User) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":  user.ID,
		"role": user.Role.String(),
		"iat":  now.Unix(),
		"exp":  now.Add(p.ttl).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(p.secret))
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return signed, nil
}
