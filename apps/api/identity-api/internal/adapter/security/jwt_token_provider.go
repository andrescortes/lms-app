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

// Compile-time proof the provider fulfils the verification port the HTTP
// adapter consumes.
var _ usecase.TokenVerifier = (*JwtTokenProvider)(nil)

// NewJwtTokenProvider returns a token provider signing with HMAC-SHA256 using
// the given secret and access-token TTL.
func NewJwtTokenProvider(secret string, ttl time.Duration) *JwtTokenProvider {
	return &JwtTokenProvider{secret: secret, ttl: ttl}
}

// Signed returns a signed access token carrying the user id and role, valid for
// the provider's TTL.
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

// Verify checks an access token's signature and expiry and returns the claims it
// carries. The signing method is pinned to HS256, so a token that claims another
// algorithm is refused rather than trusted. Any failure is reported as
// domain.ErrInvalidToken, never as a parsing detail.
func (p *JwtTokenProvider) Verify(ctx context.Context, token string) (usecase.TokenClaims, error) {
	if err := ctx.Err(); err != nil {
		return usecase.TokenClaims{}, err
	}
	parsed, err := jwt.Parse(token, func(*jwt.Token) (any, error) {
		return []byte(p.secret), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil || !parsed.Valid {
		return usecase.TokenClaims{}, domain.ErrInvalidToken
	}

	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		return usecase.TokenClaims{}, domain.ErrInvalidToken
	}
	subject, _ := claims["sub"].(string)
	role, _ := claims["role"].(string)
	if subject == "" || role == "" {
		return usecase.TokenClaims{}, domain.ErrInvalidToken
	}
	return usecase.TokenClaims{UserID: subject, Role: role}, nil
}
